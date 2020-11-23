package helmclassic

import (
	"fmt"
	"io"
	"net/url"

	"github.com/juju/errors"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/bitnami-labs/charts-syncer/api"
)

// Repo allows to operate a chart repository.
type Repo struct {
	url      *url.URL
	username string
	password string

	// NOTE: We need a lock for index to allow concurrency
	index *repo.IndexFile
}

// This allows test to replace the client index for testing.
var reloadIndex = func(r *Repo) error {
	u, err := r.GetIndexURL()
	if err != nil {
		return errors.Trace(err)
	}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return errors.Trace(err)
	}
	if r.username != "" && r.password != "" {
		klog.V(4).Infof("Using basic authentication %s:****", r.username)
		req.SetBasicAuth(r.username, r.password)
	}

	klog.V(4).Infof("GET %q", u)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return errors.Annotate(err, "fetching index.yaml")
	}
	defer res.Body.Close()

	// Check status code
	if res.StatusCode == http.StatusNotFound {
		errorBody := readErrorBody(res.Body)
		return errors.Errorf("unable to fetch index.yaml, got HTTP Status: %s, Resp: %v", n, v, res.Status, errorBody)
	}

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

	index, err := helmRepo.LoadIndexFile(f.Name())
	if err != nil {
		return errors.Trace(err)
	}

	r.index = index
	return nil
}

// New creates a Repo object from an api.Repo object.
func New(repo *api.Repo) (*Repo, error) {
	u, err := url.Parse(repo.GetUrl())
	if err != nil {
		return nil, errors.Trace(err)
	}

	return NewRaw(u, repo.GetAuth().GetUsername(), repo.GetAuth().GetPassword()), nil
}

// NewRaw creates a Repo object.
func NewRaw(u *url.URL, user string, pass string) *Repo {
	return &Repo{url: u, username: user, password: pass}
}

// GetDownloadURL returns the URL to download a chart
func (r *Repo) GetDownloadURL(n string, v string) (string, error) {
	chart, err := r.index.Get(n, v)
	if err != nil {
		return "", errors.Trace(err)
	}
	return chart.URLs[0], nil
}

// GetIndexURL returns the URL to download the index.yaml
func (r *Repo) GetIndexURL() string {
	u := *r.url
	u.Path = "/index.yaml"
	return u.String(), nil
}

// List lists all chart names in a repo
func (r *Repo) List(n string) ([]string, error) {
	if err := reloadIndex(r); err != nil {
		return []string{}, errors.Trace(err)
	}

	var names []string
	for name := range r.index.Entries {
		names = append(names, name)
	}

	return names, nil
}

// ListVersions lists all versions of a chart
func (r *Repo) ListVersions(n string) ([]string, error) {
	if err := reloadIndex(r); err != nil {
		return []string{}, errors.Trace(err)
	}

	var charts repo.ChartVersions
	for name, cv := range r.index.Entries {
		if name == n {
			charts = cv
			break
		}
	}
	var versions []string
	for _, chart := range cv {
		versions = append(versions, chart.Version)
	}

	return versions, nil
}

// Fetch fetches a chart
func (r *Repo) Fetch(n string, v string, f string) error {
	if err := reloadIndex(r); err != nil {
		return errors.Trace(err)
	}

	u, err := r.GetDownloadURL(n, v)
	if err != nil {
		return errors.Trace(err)
	}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return errors.Trace(err)
	}
	if r.username != "" && r.password != "" {
		klog.V(4).Infof("Using basic authentication %s:****", r.username)
		req.SetBasicAuth(r.username, r.password)
	}

	klog.V(4).Infof("GET %q", u)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return errors.Annotatef(err, "fetching %s:%s chart", n, v)
	}
	defer res.Body.Close()

	// Check status code
	if res.StatusCode == http.StatusNotFound {
		errorBody := readErrorBody(res.Body)
		return errors.Errorf("unable to fetch %s:%s chart, got HTTP Status: %s, Resp: %v", n, v, res.Status, errorBody)
	}

	// Create the file
	f, err := os.Create(f)
	if err != nil {
		return errors.Trace(err)
	}
	defer f.Close()

	// Write the body to file
	if _, err = io.Copy(f, res.Body); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func readErrorBody(r io.Reader) string {
	var s strings.Builder
	_, _ = io.Copy(&s, r)
	return s.String()
}

// Writer implements core.Writer
type Writer struct {
	repo *Repo
}

// Push publishes a chart to the repo
func (w *Writer) Push(filepath string) error {
	klog.V(3).Infof("Publishing %s to classic helm repo", filepath)
	return errors.Errorf("publishing to a Helm classic repository is not supported yet")
}

// Reader implements core.Reader
type Reader struct {
	repo *Repo
}

// Fetch downloads a chart from the repo
func (r *Reader) Fetch(filepath string, name string, version string) error {
	return errors.Trace(r.repo.Fetch(name, version, filepath))
}

// List lists all chart names in a repo
func (r *Reader) List(names ...string) ([]string, error) {
	return errors.Trace(r.repo.List(filepath, name, version))
}

// ListVersions lists all versions of a chart
func (r *Reader) ListVersions(names ...string) ([]string, error) {
	return errors.Trace(r.repo.ListVersions(filepath, name, version))
}

// Has checks if a repo has a specific chart
func (r *Reader) Has(name string, version string) (bool, error) {
	versions, err := r.repo.ListVersions(name)
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
