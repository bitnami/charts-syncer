package utils

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha1"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/juju/errors"
	"github.com/mkmik/multierror"
	helmRepo "helm.sh/helm/v3/pkg/repo"
	"k8s.io/klog"

	"github.com/bitnami/charts-syncer/api"
	"github.com/bitnami/charts-syncer/internal/cache"
)

const (
	timeLayoutISO = "2006-01-02"
)

var (
	// MaxDecompressionSize established a high enough maximum tar size to decompres
	// to prevent decompression bombs (8GB)
	MaxDecompressionSize int64 = 8 * 1024 * 1024 * 1024
	// UnixEpoch is the number of seconds that have elapsed since January 1, 1970
	UnixEpoch = time.Unix(0, 0)
	// DefaultClient is a default HTTP client
	DefaultClient = &http.Client{Transport: &http.Transport{Proxy: http.ProxyFromEnvironment}}
	// InsecureClient is a default insecure HTTPS client
	InsecureClient = &http.Client{Transport: &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, // #nosec G402
	}
)

// LoadIndexFromRepo get the index.yaml from a Helm repo and returns an index object
func LoadIndexFromRepo(repo *api.Repo) (*helmRepo.IndexFile, error) {
	indexFile, err := downloadIndex(repo)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer os.Remove(indexFile)
	index, err := helmRepo.LoadIndexFile(indexFile)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return index, errors.Trace(err)
}

// ChartExistInIndex checks if a specific chart version is present in the index file.
func ChartExistInIndex(name string, version string, index *helmRepo.IndexFile) bool {
	chartVersionFound := false
	if index.Entries[name] != nil {
		klog.V(3).Infof("Chart %q exists in index.yaml file. Searching %q version", name, version)
		for i := range index.Entries[name] {
			if index.Entries[name][i].Metadata.Version == version {
				klog.V(3).Infof("Version %q found for chart %q in index.yaml file", index.Entries[name][i].Metadata.Version, name)
				chartVersionFound = true
				break
			}
		}
		if !chartVersionFound {
			return false
		}
	} else {
		return false
	}

	return chartVersionFound
}

// downloadIndex will download the index.yaml file of a chart repository and return
// the path to the downloaded file.
func downloadIndex(repo *api.Repo) (string, error) {
	downloadURL := repo.GetUrl() + "/index.yaml"

	// Get the data
	client := DefaultClient
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return "", errors.Trace(err)
	}
	if repo.GetAuth() != nil && repo.GetAuth().GetUsername() != "" && repo.GetAuth().GetPassword() != "" {
		klog.V(4).Info("Repo configures basic authentication. Downloading index.yaml...")
		req.SetBasicAuth(repo.GetAuth().GetUsername(), repo.GetAuth().GetPassword())
	}
	res, err := client.Do(req)
	if err != nil {
		return "", errors.Trace(err)
	}
	defer res.Body.Close()
	// Check status code
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return "", errors.Errorf("error downloading index.yaml from %s. Status code is %d", repo.GetUrl(), res.StatusCode)
	}

	// Create the file
	out, err := os.CreateTemp("", "index.*.yaml")
	if err != nil {
		klog.Fatal(err)
	}

	// Write the body to file
	_, err = io.Copy(out, res.Body)
	return out.Name(), errors.Trace(err)
}

// Untar extracts compressed archives
func Untar(tarball, targetDir string) error {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return errors.Trace(err)
	}

	f, err := os.Open(tarball)
	if err != nil {
		return errors.Trace(err)
	}
	defer f.Close()
	gzipReader, err := gzip.NewReader(f)
	if err != nil {
		return errors.Trace(err)
	}
	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Trace(err)
		}
		path := filepath.Join(targetDir, header.Name)
		targetFolder := filepath.Join(targetDir, filepath.Dir(header.Name))
		// For some reason the for loop only iterates over files and not folders, so the switch below for folders is
		// never executed and so we are creating the target folder at this point.
		if _, err := os.Stat(targetFolder); err != nil {
			if err := os.MkdirAll(targetFolder, 0755); err != nil {
				return err
			}
		}
		switch header.Typeflag {
		// Related to previous comment. It seems this block of code is never executed.
		case tar.TypeDir:
			if _, err := os.Stat(path); err != nil {
				if err := os.Mkdir(path, 0755); err != nil {
					return errors.Trace(err)
				}
			}
		case tar.TypeReg:
			outFile, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return errors.Trace(err)
			}
			if _, err := io.CopyN(outFile, tarReader, MaxDecompressionSize); err != nil && err != io.EOF {
				_ = outFile.Close()
				return errors.Trace(err)
			}
			if err := outFile.Close(); err != nil {
				return errors.Trace(err)
			}
		// We don't want to process these extension header files.
		case tar.TypeXGlobalHeader, tar.TypeXHeader:
			continue
		default:
			return errors.Errorf("unknown type: %b in %s", header.Typeflag, header.Name)
		}
	}
	return nil
}

// GetFileContentType returns the content type of a file.
func GetFileContentType(filepath string) (string, error) {
	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)
	file, err := os.Open(filepath)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	n, err := file.Read(buffer)
	if err != nil {
		return "", err
	}
	// Use the net/http package's handy DectectContentType function. Always returns a valid
	// content-type by returning "application/octet-stream" if no others seemed to match.
	contentType := http.DetectContentType(buffer[:n])
	return contentType, err
}

// GetDateThreshold will parse a string date against a fixed layout and return a time.Date value
func GetDateThreshold(date string) (time.Time, error) {
	if date == "" {
		return UnixEpoch, nil
	}
	dateThreshold, err := time.Parse(timeLayoutISO, date)
	if err != nil {
		return dateThreshold, errors.Trace(err)
	}
	return dateThreshold, nil
}

// FindChartURL will return the chart url
func FindChartURL(name string, version string, index *helmRepo.IndexFile, sourceURL string) (string, error) {
	chart := findChartByVersion(index.Entries[name], version)
	if chart != nil && len(chart.URLs) > 0 {
		if isValidURL(chart.URLs[0]) {
			return chart.URLs[0], nil
		}
		return fmt.Sprintf("%s/%s", sourceURL, chart.URLs[0]), nil
	}
	return "", fmt.Errorf("unable to find chart url in index")
}

// findChartByVersion returns the chart that matches a provided version from a list of charts.
func findChartByVersion(chartVersions []*helmRepo.ChartVersion, version string) *helmRepo.ChartVersion {
	for _, chart := range chartVersions {
		if chart.Version == version {
			return chart
		}
	}
	return nil
}

// isValidUrl tests a string to determine if it is a well-structured url or not.
func isValidURL(text string) bool {
	_, err := url.ParseRequestURI(text)
	return err == nil
}

// FileExists will test if a file exists
func FileExists(f string) (bool, error) {
	if _, err := os.Stat(f); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, errors.Trace(err)
	}
	return true, nil
}

// CopyFile copies a file from srcPath to destPath, ensuring the destPath directory exists.
func CopyFile(destPath string, srcPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return errors.Trace(err)
	}
	src, err := os.Open(srcPath)
	if err != nil {
		return errors.Trace(err)
	}
	defer src.Close()

	dest, err := os.Create(destPath)
	if err != nil {
		return errors.Trace(err)
	}
	defer dest.Close()

	if _, err := io.Copy(dest, src); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// HTTPResponseBody returns the body of an HTTP response
func HTTPResponseBody(res *http.Response) string {
	var s strings.Builder
	_, _ = io.Copy(&s, res.Body)
	return s.String()
}

// EncodeSha1 returns a SHA1 representation of the provided string
func EncodeSha1(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// NormalizeChartURL forms the full download URL in case we pass a relative URL
func NormalizeChartURL(repoURL, chartURL string) (string, error) {
	if chartURL == "" {
		return "", errors.New("chart URL cannot be empty")
	}

	// Return chart URL if it refers to a host (it is absolute)
	if cu, err := url.Parse(chartURL); err != nil {
		return "", errors.Trace(err)
	} else if cu.Host != "" {
		return chartURL, nil
	}

	// Build chart URL using the repository URL if, and only if, the chart
	// URL is relative
	if repoURL == "" {
		return "", errors.New("repository URL cannot be empty")
	}
	if _, err := url.Parse(repoURL); err != nil {
		return "", errors.Trace(err)
	}
	return fmt.Sprintf("%s/%s", repoURL, chartURL), nil
}

// GetListenAddress returns a free local direction
func GetListenAddress() (string, error) {
	lst, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", errors.Trace(err)
	}
	defer lst.Close()

	return lst.Addr().String(), nil
}

type statusHandler func(res *http.Response) error
type urlBuilder func(name, version string) (string, error)

type fetchOptions struct {
	user            string
	pass            string
	insecure        bool
	statusHandlerFn statusHandler
	urlBuilderFn    urlBuilder
}

// FetchOption defines a fetchOptions setting
type FetchOption func(opts *fetchOptions)

// WithFetchUsername configures a username for fetch operations
func WithFetchUsername(user string) FetchOption {
	return func(opts *fetchOptions) {
		opts.user = user
	}
}

// WithFetchPassword configures a password for fetch operations
func WithFetchPassword(pass string) FetchOption {
	return func(opts *fetchOptions) {
		opts.pass = pass
	}
}

// WithFetchInsecure enables insecure connection for fetch operations
func WithFetchInsecure(insecure bool) FetchOption {
	return func(opts *fetchOptions) {
		opts.insecure = insecure
	}
}

// WithFetchStatusHandler configures a status handler for fetch operations
func WithFetchStatusHandler(h statusHandler) FetchOption {
	return func(opts *fetchOptions) {
		opts.statusHandlerFn = h
	}
}

// WithFetchURLBuilder configures a URL builder for fetch operations
func WithFetchURLBuilder(h urlBuilder) FetchOption {
	return func(opts *fetchOptions) {
		opts.urlBuilderFn = h
	}
}

var defaultStatusHandler = func(res *http.Response) error {
	if ok := res.StatusCode >= 200 && res.StatusCode <= 299; !ok {
		bodyStr := HTTPResponseBody(res)
		return errors.Errorf("got HTTP Status: %s, Resp: %v", res.Status, bodyStr)
	}
	return nil
}

// FetchAndCache fetches a chart and stores it in provided cache
func FetchAndCache(name, version string, cache cache.Cacher, fopts ...FetchOption) (string, error) {
	id := fmt.Sprintf("%s-%s.tgz", name, version)
	if cache.Has(id) {
		return cache.Path(id), nil
	}

	opts := fetchOptions{statusHandlerFn: defaultStatusHandler}
	for _, opt := range fopts {
		opt(&opts)
	}

	if opts.urlBuilderFn == nil {
		return "", fmt.Errorf("requires a download URL builder")
	}

	u, err := opts.urlBuilderFn(name, version)
	if err != nil {
		return "", errors.Trace(err)
	}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return "", errors.Trace(err)
	}

	if opts.user != "" && opts.pass != "" {
		req.SetBasicAuth(opts.user, opts.pass)
	}

	reqID := EncodeSha1(u + id)
	klog.V(4).Infof("[%s] GET %q", reqID, u)

	client := DefaultClient
	if opts.insecure {
		client = InsecureClient
	}

	res, err := client.Do(req)
	if err != nil {
		return "", errors.Trace(err)
	}

	klog.V(4).Infof("[%s] HTTP Status: %s", reqID, res.Status)
	if opts.statusHandlerFn != nil {
		if err := opts.statusHandlerFn(res); err != nil {
			return "", errors.Trace(err)
		}
	}

	w, err := cache.Writer(id)
	if err != nil {
		return "", errors.Trace(err)
	}

	if _, err := io.Copy(w, res.Body); err != nil {
		// Invalidate the cache
		return "", errors.Trace(multierror.Append(err, cache.Invalidate(id)))
	}

	if err := w.Close(); err != nil {
		// Invalidate the cache
		return "", errors.Trace(multierror.Append(err, cache.Invalidate(id)))
	}
	if err := res.Body.Close(); err != nil {
		return "", errors.Trace(err)
	}

	return cache.Path(id), nil
}
