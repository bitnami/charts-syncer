package harbor

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/juju/errors"
	"k8s.io/klog"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/client/helmclassic"
	"github.com/bitnami-labs/charts-syncer/pkg/client/types"
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

	return NewRaw(u, repo.GetAuth().GetUsername(), repo.GetAuth().GetPassword())
}

// NewRaw creates a Repo object.
func NewRaw(u *url.URL, user string, pass string) (*Repo, error) {
	helm, err := helmclassic.NewRaw(u, user, pass)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &Repo{url: u, username: user, password: pass, helm: helm}, nil
}

// GetUploadURL returns the URL to upload a chart
func (r *Repo) GetUploadURL() string {
	u := *r.url
	u.Path = strings.Replace(u.Path, "/chartrepo/", "/api/chartrepo/", 1) + "/charts"
	return u.String()
}

// Upload uploads a chart to the repo
func (r *Repo) Upload(filepath string) error {
	body := &bytes.Buffer{}
	mpw := multipart.NewWriter(body)

	w, err := mpw.CreateFormFile("chart", filepath)
	if err != nil {
		return errors.Trace(err)
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
		req.SetBasicAuth(r.username, r.password)
	}

	reqID := utils.EncodeSha1(u + filepath)
	klog.V(4).Infof("[%s] POST %q", reqID, u)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return errors.Annotatef(err, "uploading %q chart", filepath)
	}
	defer res.Body.Close()

	if ok := res.StatusCode >= 200 && res.StatusCode <= 299; !ok {
		bodyStr := utils.HTTPResponseBody(res)
		return errors.Errorf("unable to fetch index.yaml, got HTTP Status: %s, Resp: %v", res.Status, bodyStr)
	}
	klog.V(4).Infof("[%s] Got HTTP Status: %s", reqID, res.Status)

	return nil
}

// Fetch downloads a chart from the repo
func (r *Repo) Fetch(filepath string, name string, version string) error {
	return r.helm.Fetch(filepath, name, version)
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
