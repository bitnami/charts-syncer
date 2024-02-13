// Package local implements a client for local repositories
package local

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/juju/errors"

	"github.com/bitnami/charts-syncer/internal/utils"
	"github.com/bitnami/charts-syncer/pkg/client/types"
	"helm.sh/helm/v3/pkg/chart"
)

var (
	versionRe = regexp.MustCompile(`(.*)-(\d+\.\d+\.\d+)(\.wrap)?\.tgz`)
)

// Repo allows to operate a chart repository.
type Repo struct {
	dir     string
	entries map[string][]string
}

// New creates a Repo object from an api.Repo object.
func New(dir string) (*Repo, error) {
	d, err := filepath.Abs(dir)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if err := os.MkdirAll(d, 0755); err != nil {
		return nil, errors.Trace(err)
	}

	// Populate entries from directory
	entries := make(map[string][]string)
	matches, err := filepath.Glob(filepath.Join(d, "*.wrap.tgz"))
	if err != nil {
		return nil, errors.Trace(err)
	}
	for _, m := range matches {
		filename := filepath.Base(m)
		s := versionRe.FindStringSubmatch(filename)
		entries[s[1]] = append(entries[s[1]], s[2])
		sort.Strings(entries[s[0]])
	}

	return &Repo{dir: d, entries: entries}, nil
}

// Dir returns the absolute path to the repository's directory
func (r *Repo) Dir() string {
	return r.dir
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
	return path.Join(r.dir, fmt.Sprintf("%s-%s.wrap.tgz", name, version)), nil
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

// GetUploadURL returns the URL to upload a chart
func (r *Repo) GetUploadURL() string {
	return r.dir
}

// Upload uploads a chart to the repo
func (r *Repo) Upload(filepath string, metadata *chart.Metadata) error {
	name := metadata.Name
	version := metadata.Version
	if _, ok := r.entries[name]; ok {
		for _, v := range r.entries[name] {
			if v == version {
				return errors.AlreadyExistsf("%s-%s", name, version)
			}
		}
	}

	input, err := os.ReadFile(filepath)
	if err != nil {
		return errors.Annotatef(err, "reading %q", filepath)
	}

	out := path.Join(r.dir, fmt.Sprintf("%s-%s.wrap.tgz", name, version))
	if err := os.WriteFile(out, input, 0644); err != nil {
		return errors.Annotatef(err, "creating %q", out)
	}

	r.entries[name] = append(r.entries[name], version)
	sort.Strings(r.entries[name])

	return nil
}

// GetChartDetails returns the details of a chart
func (r *Repo) GetChartDetails(_ string, _ string) (*types.ChartDetails, error) {
	return &types.ChartDetails{
		PublishedAt: utils.UnixEpoch,
		Digest:      "deadbuff",
	}, nil
}

// Reload reloads the index
func (r *Repo) Reload() error {
	return nil
}
