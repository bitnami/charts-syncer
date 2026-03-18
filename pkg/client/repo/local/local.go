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
	"helm.sh/helm/v3/pkg/chart"

	"github.com/bitnami/charts-syncer/internal/utils"
	"github.com/bitnami/charts-syncer/pkg/client/types"
)

var (
	chartVersionRe     = regexp.MustCompile(`(.*)-(\d+\.\d+\.\d+(-rc|-alpha|-preview)?)(\.wrap)?\.tgz`)
	containerVersionRe = regexp.MustCompile(`(.*)-(\d+\.\d+\.\d+-(.*)-r\d+)(\.container\.wrap)?\.tgz`)
)

// Repo allows to operate a chart repository.
type Repo struct {
	dir              string
	chartEntries     map[string][]string
	containerEntries map[string][]string
}

// New creates a Repo object from an api.Repo object.
func New(dir string) (*Repo, error) {
	d, err := filepath.Abs(dir)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if err = os.MkdirAll(d, 0755); err != nil {
		return nil, errors.Trace(err)
	}

	// Populate chart entries from directory
	chartEntries := make(map[string][]string)
	matches, err := filepath.Glob(filepath.Join(d, "charts", "*.wrap.tgz"))
	if err != nil {
		return nil, errors.Trace(err)
	}
	for _, m := range matches {
		filename := filepath.Base(m)
		s := chartVersionRe.FindStringSubmatch(filename)
		if len(s) < 3 {
			continue
		}
		chartEntries[s[1]] = append(chartEntries[s[1]], s[2])
		sort.Strings(chartEntries[s[1]])
	}

	// Populate container entries from directory
	containerEntries := make(map[string][]string)
	matches, err = filepath.Glob(filepath.Join(d, "containers", "*.container.wrap.tgz"))
	if err != nil {
		return nil, errors.Trace(err)
	}
	for _, m := range matches {
		filename := filepath.Base(m)
		s := containerVersionRe.FindStringSubmatch(filename)
		if len(s) < 3 {
			continue
		}
		containerEntries[s[1]] = append(containerEntries[s[1]], s[2])
		sort.Strings(containerEntries[s[1]])
	}

	return &Repo{dir: d, chartEntries: chartEntries, containerEntries: containerEntries}, nil
}

// Dir returns the absolute path to the repository's directory
func (r *Repo) Dir() string {
	return r.dir
}

// List lists all chart names in a repo
func (r *Repo) List() ([]string, error) {
	var names []string
	for name := range r.chartEntries {
		names = append(names, name)
	}
	return names, nil
}

// ListChartVersions lists all versions of a chart
func (r *Repo) ListChartVersions(name string) ([]string, error) {
	versions, ok := r.chartEntries[name]
	if !ok {
		return []string{}, nil
	}
	return versions, nil
}

// ListContainerTags lists all versions of a container
func (r *Repo) ListContainerTags(name string) ([]string, error) {
	versions, ok := r.containerEntries[name]
	if !ok {
		return []string{}, nil
	}
	return versions, nil
}

// Fetch fetches a chart
func (r *Repo) Fetch(name string, version string) (string, error) {
	return path.Join(r.dir, "charts", fmt.Sprintf("%s-%s.wrap.tgz", name, version)), nil
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

// HasContainer checks if a repo has a specific container
func (r *Repo) HasContainer(name string, tag string) (bool, error) {
	tags, err := r.ListContainerTags(name)
	if err != nil {
		return false, errors.Trace(err)
	}

	for _, t := range tags {
		if t == tag {
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
	if _, ok := r.chartEntries[name]; ok {
		for _, v := range r.chartEntries[name] {
			if v == version {
				return errors.AlreadyExistsf("%s-%s", name, version)
			}
		}
	}

	input, err := os.ReadFile(filepath)
	if err != nil {
		return errors.Annotatef(err, "reading %q", filepath)
	}

	out := path.Join(r.dir, "charts", fmt.Sprintf("%s-%s.wrap.tgz", name, version))
	if err := os.MkdirAll(path.Dir(out), 0755); err != nil {
		return errors.Annotatef(err, "creating directory for %q", out)
	}
	if err := os.WriteFile(out, input, 0644); err != nil {
		return errors.Annotatef(err, "creating %q", out)
	}

	r.chartEntries[name] = append(r.chartEntries[name], version)
	sort.Strings(r.chartEntries[name])

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
