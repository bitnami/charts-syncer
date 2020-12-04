package fake

import (
	"fmt"
	"io/ioutil"
	"path"

	"github.com/juju/errors"

	"github.com/bitnami-labs/charts-syncer/internal/utils"
	"github.com/bitnami-labs/charts-syncer/pkg/client/types"
)

// Repo allows to operate a chart repository.
type Repo struct {
	dir     string
	entries map[string][]string
}

// New creates a Repo object from an api.Repo object.
func New(dir string, entries map[string][]string) *Repo {
	return &Repo{dir: dir, entries: entries}
}

// List lists all chart names in a repo
func (r *Repo) List() ([]string, error) {
	var names []string
	for name := range r.entries {
		names = append(names, name)
	}
	return names, nil
}

// ListChartVersions lists all versions of a chart
func (r *Repo) ListChartVersions(name string) ([]string, error) {
	versions, ok := r.entries[name]
	if !ok {
		return []string{}, nil
	}
	return versions, nil
}

// Fetch fetches a chart
func (r *Repo) Fetch(name string, version string) (string, error) {
	return path.Join(r.dir, fmt.Sprintf("%s-%s.tgz", name, version)), nil
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

// Upload uploads a chart to the repo
func (r *Repo) Upload(filepath string, name string, version string) error {
	input, err := ioutil.ReadFile(filepath)
	if err != nil {
		return errors.Annotatef(err, "reading %q", filepath)
	}

	out := path.Join(r.dir, fmt.Sprintf("%s-%s.tgz", name, version))
	if err := ioutil.WriteFile(out, input, 0644); err != nil {
		return errors.Annotatef(err, "creating %q", out)
	}

	return nil
}

// GetChartDetails returns the details of a chart
func (r *Repo) GetChartDetails(name string, version string) (*types.ChartDetails, error) {
	return &types.ChartDetails{
		PublishedAt: utils.UnixEpoch,
		Digest:      "deadbuff",
	}, nil
}

// Reload reloads the index
func (r *Repo) Reload() error {
	return nil
}
