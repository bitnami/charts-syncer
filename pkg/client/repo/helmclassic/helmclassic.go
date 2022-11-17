package helmclassic

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/juju/errors"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/klog"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/internal/cache"
	"github.com/bitnami-labs/charts-syncer/internal/utils"
	"github.com/bitnami-labs/charts-syncer/pkg/client/types"
)

// Repo allows to operate a chart repository.
type Repo struct {
	url      *url.URL
	username string
	password string
	insecure bool

	// NOTE: We need a lock for index to allow concurrency
	Index *repo.IndexFile

	cache cache.Cacher
}

// This allows test to replace the client index for testing.
var reloadIndex = func(r *Repo) error {
	u := r.GetIndexURL()
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return errors.Trace(err)
	}
	if r.username != "" && r.password != "" {
		req.SetBasicAuth(r.username, r.password)
	}

	reqID := utils.EncodeSha1(u + "index.yaml")
	klog.V(4).Infof("[%s] GET %q", reqID, u)
	client := utils.DefaultClient
	if r.insecure {
		client = utils.InsecureClient
	}
	res, err := client.Do(req)
	if err != nil {
		return errors.Annotate(err, "fetching index.yaml")
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		r.Index = repo.NewIndexFile()
		return nil
	}

	if ok := res.StatusCode >= 200 && res.StatusCode <= 299; !ok {
		bodyStr := utils.HTTPResponseBody(res)
		return errors.Errorf("unable to fetch index.yaml, got HTTP Status: %s, Resp: %v", res.Status, bodyStr)
	}
	klog.V(4).Infof("[%s] HTTP Status: %s", reqID, res.Status)

	// Create the index.yaml file to use the helm Go library, which does not
	// expose a Loader from bytes.
	f, err := ioutil.TempFile("", "index.*.yaml")
	if err != nil {
		return errors.Trace(err)
	}
	defer os.Remove(f.Name())

	// Write the body to file
	if _, err = io.Copy(f, res.Body); err != nil {
		return errors.Trace(err)
	}
	if err := f.Close(); err != nil {
		return errors.Trace(err)
	}

	index, err := repo.LoadIndexFile(f.Name())
	if err != nil {
		return errors.Annotate(err, "loading index.yaml file")
	}

	r.Index = index
	return nil
}

// New creates a Repo object from an api.Repo object.
func New(repo *api.Repo, c cache.Cacher, insecure bool) (*Repo, error) {
	u, err := url.Parse(repo.GetUrl())
	if err != nil {
		return nil, errors.Trace(err)
	}

	return NewRaw(u, repo.GetAuth().GetUsername(), repo.GetAuth().GetPassword(), c, insecure)
}

// NewRaw creates a Repo object.
func NewRaw(u *url.URL, user string, pass string, c cache.Cacher, insecure bool) (*Repo, error) {
	r := &Repo{url: u, username: user, password: pass, cache: c, insecure: insecure}

	if err := r.Reload(); err != nil {
		return nil, errors.Trace(err)
	}

	return r, nil
}

// GetDownloadURL returns the URL to download a chart
func (r *Repo) GetDownloadURL(n string, v string) (string, error) {
	chart, err := r.Index.Get(n, v)
	if err != nil {
		return "", errors.Annotatef(err, "getting %s-%s from index file", n, v)
	}
	u, err := utils.NormalizeChartURL(r.url.String(), chart.URLs[0])
	if err != nil {
		return "", errors.Trace(err)
	}
	return u, nil
}

// GetIndexURL returns the URL to download the index.yaml
func (r *Repo) GetIndexURL() string {
	u := *r.url
	u.Path = u.Path + "/index.yaml"
	return u.String()
}

// List lists all chart names in a repo
func (r *Repo) List() ([]string, error) {
	var names []string
	for name := range r.Index.Entries {
		names = append(names, name)
	}

	return names, nil
}

// ListChartVersions lists all versions of a chart
func (r *Repo) ListChartVersions(name string) ([]string, error) {
	cv, ok := r.Index.Entries[name]
	if !ok {
		return []string{}, nil
	}

	var versions []string
	for _, chart := range cv {
		versions = append(versions, chart.Version)
	}

	return versions, nil
}

// Fetch fetches a chart
func (r *Repo) Fetch(name string, version string) (string, error) {
	fetchOpts := []utils.FetchOption{
		utils.WithFetchUsername(r.username),
		utils.WithFetchPassword(r.password),
		utils.WithFetchInsecure(r.insecure),
		utils.WithFetchURLBuilder(r.GetDownloadURL),
	}
	chartPath, err := utils.FetchAndCache(name, version, r.cache, fetchOpts...)
	if err != nil {
		return "", errors.Annotatef(err, "fetching %s:%s chart", name, version)
	}

	return chartPath, nil
}

// Has checks if a repo has a specific chart
func (r *Repo) Has(name string, version string) (bool, error) {
	versions, err := r.ListChartVersions(name)
	if err != nil {
		return false, errors.Trace(err)
	}

	for _, v := range versions {
		if v == version {
			return true, nil
		}
	}
	return false, nil
}

func (r *Repo) CreateRepository(repository string) error {
	return nil
}

// Upload uploads a chart to the repo
func (r *Repo) Upload(_ string, _ *chart.Metadata) error {
	return errors.Errorf("upload method is not supported yet")
}

// GetChartDetails returns the details of a chart
func (r *Repo) GetChartDetails(name string, version string) (*types.ChartDetails, error) {
	cv, err := r.Index.Get(name, version)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &types.ChartDetails{
		PublishedAt: cv.Created,
		Digest:      cv.Digest,
	}, nil
}

// Reload reloads the index
func (r *Repo) Reload() error {
	return errors.Annotatef(reloadIndex(r), "reloading %q chart repo", r.url)
}
