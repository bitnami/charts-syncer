// Package harbor implements a client for Harbor repositories
package harbor

import (
	"net/url"
	"strings"

	apiv1 "github.com/bitnami/charts-syncer/gen/proto/v1"
	"github.com/bitnami/charts-syncer/internal/cache"
	"github.com/bitnami/charts-syncer/pkg/client/repo/helmclassic"
	"github.com/bitnami/charts-syncer/pkg/client/types"
	"github.com/juju/errors"
)

// Repo allows to operate a chart repository.
type Repo struct {
	url      *url.URL
	username string
	password string
	insecure bool

	helm *helmclassic.Repo

	cache cache.Cacher
}

// New creates a Repo object from an apiv1.Repo object.
func New(repo *apiv1.Repo, c cache.Cacher, insecure bool) (*Repo, error) {
	u, err := url.Parse(repo.GetUrl())
	if err != nil {
		return nil, errors.Trace(err)
	}

	return NewRaw(u, repo.GetAuth().GetUsername(), repo.GetAuth().GetPassword(), c, insecure)
}

// NewRaw creates a Repo object.
func NewRaw(u *url.URL, user string, pass string, c cache.Cacher, insecure bool) (*Repo, error) {
	helm, err := helmclassic.NewRaw(u, user, pass, c, insecure)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &Repo{url: u, username: user, password: pass, helm: helm, cache: c, insecure: insecure}, nil
}

// GetUploadURL returns the URL to upload a chart
func (r *Repo) GetUploadURL() string {
	u := *r.url
	u.Path = strings.Replace(u.Path, "/chartrepo/", "/api/chartrepo/", 1) + "/charts"
	return u.String()
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
