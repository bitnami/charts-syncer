package core

import (
	"fmt"
	"net/url"

	"github.com/juju/errors"
	helmRepo "helm.sh/helm/v3/pkg/repo"
	"k8s.io/klog"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/client/helmclassic"
	"github.com/bitnami-labs/charts-syncer/pkg/utils"
)

// Repo allows to operate a chart repository.
type Repo struct {
	url      *url.URL
	username string
	password string

	helm *helmclassic.Repo
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
	helm := helmclassic.NewRaw(u, user, pass)
	return &Repo{url: u, username: user, password: pass, helm: helm}
}

// GetUploadURL returns the URL to upload a chart
func (r *Repo) GetUploadURL() string {
	u := *r.url
	u.Path = "/api/charts"
	return u.String()
}

// Upload uploads a chart to the repo.
func (r *Repo) Upload(filepath string) error {
	klog.V(4).Infof("Publishing %q", filepath)

	body := &bytes.Buffer{}
	mpw := multipart.NewWriter(body)

	w, err := mpw.CreateFormFile("chart", filepath)
	if err != nil {
		return errors.Trace(Err)
	}

	f, err := os.Open(filepath)
	if err != nil {
		return errors.Trace(err)
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	if err != nil {
		return errors.Trace(err)
	}

	contentType := mpw.FormDataContentType()
	if err := mpw.Close(); err != nil {
		return errors.Trace(err)
	}

	u := r.GetUploadURL()
	req, err := http.NewRequest("POST", u, body)
	if err != nil {
		return errors.Trace(err)
	}
	req.Header.Add("content-type", contentType)
	if r.username != "" && r.password != "" {
		klog.V(4).Infof("Using basic authentication %s:****", r.username)
		req.SetBasicAuth(r.username, r.password)
	}

	klog.V(4).Infof("POST %q", u)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return errors.Annotatef(err, "uploading %q chart", filepath)
	}
	defer res.Body.Close()

	// Check status code
	if res.StatusCode == http.StatusNotFound {
		errorBody := readErrorBody(res.Body)
		return errors.Errorf("unable to upload chart, got HTTP Status: %s, Resp: %v", n, v, res.Status, errorBody)
	}

	return nil
}

// Writer implements core.Writer
type Writer struct {
	repo *Repo
}

// Push publishes a packaged chart to classic helm repository
func (w *Writer) Push(filepath string) error {
	return errors.Trace(w.repo.Upload(filepath))
}

// Reader implements core.Reader
type Reader struct {
	repo *Repo
}

// Fetch downloads a chart from the repo
func (r *Reader) Fetch(filepath string, name string, version string) error {
	return errors.Trace(r.repo.helm.Fetch(filepath, name, version))
}

// List lists all chart names in the repo
func (r *Reader) List(names ...string) ([]string, error) {
	return errors.Trace(r.repo.helm.List(filepath, name, version))
}

// ListVersions lists all versions of a chart
func (r *Reader) ListVersions(names ...string) ([]string, error) {
	return errors.Trace(r.repo.helm.ListVersions(filepath, name, version))
}

// Has checks if a repo has a specific chart
func (r *Reader) Has(name string, version string) (bool, error) {
	return errors.Trace(r.repo.helm.ChartExists(filepath, name, version))
}
