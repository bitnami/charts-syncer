package chartmuseum

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/juju/errors"
	"k8s.io/klog"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/internal/cache"
	"github.com/bitnami-labs/charts-syncer/internal/utils"
	"github.com/bitnami-labs/charts-syncer/pkg/client/helmclassic"
	"github.com/bitnami-labs/charts-syncer/pkg/client/types"
)

// Repo allows to operate a chart repository.
type Repo struct {
	*types.ClientOpts

	url      *url.URL
	username string
	password string

	helm *helmclassic.Repo

	cache cache.Cacher
}

// New creates a Repo object from an api.Repo object.
func New(repo *api.Repo, c cache.Cacher) (*Repo, error) {
	u, err := url.Parse(repo.GetUrl())
	if err != nil {
		return nil, errors.Trace(err)
	}

	return NewRaw(u, repo.GetAuth().GetUsername(), repo.GetAuth().GetPassword(), c)
}

// NewRaw creates a Repo object.
func NewRaw(u *url.URL, user string, pass string, c cache.Cacher) (*Repo, error) {
	helm, err := helmclassic.NewRaw(u, user, pass, c)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &Repo{url: u, username: user, password: pass, helm: helm, cache: c}, nil
}

// GetUploadURL returns the URL to upload a chart
func (r *Repo) GetUploadURL() string {
	u := *r.url
	u.Path = "/api/charts"
	return u.String()
}

// Upload uploads a chart to the repo.
func (r *Repo) Upload(file, _, _ string) error {
	f, err := os.Open(file)
	if err != nil {
		return errors.Trace(err)
	}
	defer f.Close()

	body := &bytes.Buffer{}
	mpw := multipart.NewWriter(body)
	cw, err := mpw.CreateFormFile("chart", file)
	if err != nil {
		return errors.Trace(err)
	}

	var w io.Writer = cw
	if cf := filepath.Base(file); !r.cache.Has(cf) {
		// Write file to the multipart and cache writers at the same time.
		cachew := r.cache.Writer(cf)
		w = io.MultiWriter(cw, cachew)
	}

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
		req.SetBasicAuth(r.username, r.password)
	}

	reqID := utils.EncodeSha1(u + file)
	klog.V(4).Infof("[%s] POST %q", reqID, u)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return errors.Annotatef(err, "uploading %q chart", file)
	}
	defer res.Body.Close()

	bodyStr := utils.HTTPResponseBody(res)
	if ok := res.StatusCode >= 200 && res.StatusCode <= 299; !ok {
		return errors.Errorf("unable to upload %q chart, got HTTP Status: %s, Resp: %v", file, res.Status, bodyStr)
	}
	klog.V(4).Infof("[%s] HTTP Status: %s, Resp: %v", reqID, res.Status, bodyStr)

	return nil
}

// Fetch downloads a chart from the repo
func (r *Repo) Fetch(name string, version string) (string, error) {
	return r.helm.Fetch(name, version)
}

// List lists all chart names in the repo
func (r *Repo) List() ([]string, error) {
	return r.helm.List()
}

// ListChartVersions lists all versions of a chart
func (r *Repo) ListChartVersions(name string) ([]string, error) {
	return r.helm.ListChartVersions(name)
}

// Has checks if a repo has a specific chart
func (r *Repo) Has(name string, version string) (bool, error) {
	return r.helm.Has(name, version)
}

// GetChartDetails returns the details of a chart
func (r *Repo) GetChartDetails(name string, version string) (*types.ChartDetails, error) {
	return r.helm.GetChartDetails(name, version)
}

// Reload reloads the index
func (r *Repo) Reload() error {
	return r.helm.Reload()
}
