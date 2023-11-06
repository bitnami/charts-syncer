package intermediate

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/bitnami/charts-syncer/internal/utils"
	"github.com/bitnami/charts-syncer/pkg/client"
	"github.com/bitnami/charts-syncer/pkg/client/types"
	"github.com/juju/errors"
	"helm.sh/helm/v3/pkg/chart"
)

var (
	versionRe = regexp.MustCompile("(.*)-(\\d+\\.\\d+\\.\\d+)\\.bundle.tar")
)

type chartVersions []string

// BundlesDir allows to operate a chart bundles directory
// It should implement pkg/client ChartsReaderWriter interface
type BundlesDir struct {
	dir     string
	entries map[string]chartVersions
}

// NewIntermediateClient returns a ChartsReaderWriter object
func NewIntermediateClient(intermediateBundlesPath string) (client.ChartsReaderWriter, error) {
	return New(intermediateBundlesPath)
}

// New creates a Repo object from an api.Repo object.
func New(dir string) (*BundlesDir, error) {
	d, err := filepath.Abs(dir)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if err := os.MkdirAll(d, 0755); err != nil {
		return nil, errors.Trace(err)
	}

	// Populate entries from directory
	entries := make(map[string]chartVersions)
	matches, err := filepath.Glob(filepath.Join(d, "*.tar"))
	if err != nil {
		return nil, errors.Trace(err)
	}
	for _, m := range matches {
		filename := filepath.Base(m)
		s := versionRe.FindStringSubmatch(filename)
		entries[s[1]] = append(entries[s[1]], s[2])
		sort.Strings(entries[s[0]])
	}

	return &BundlesDir{dir: d, entries: entries}, nil
}

// List lists all chart names in a repo
func (bd *BundlesDir) List() ([]string, error) {
	var names []string
	for name := range bd.entries {
		names = append(names, name)
	}
	return names, nil
}

// ListChartVersions lists all versions of a chart
func (bd *BundlesDir) ListChartVersions(name string) ([]string, error) {
	versions, ok := bd.entries[name]
	if !ok {
		return []string{}, nil
	}
	return versions, nil
}

// Fetch fetches a chart
func (bd *BundlesDir) Fetch(name string, version string) (string, error) {
	return path.Join(bd.dir, fmt.Sprintf("%s-%s.bundle.tar", name, version)), nil
}

// Has checks if a repo has a specific chart
func (bd *BundlesDir) Has(name string, version string) (bool, error) {
	versions, err := bd.ListChartVersions(name)
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
func (bd *BundlesDir) Upload(filepath string, metadata *chart.Metadata) error {
	name := metadata.Name
	version := metadata.Version
	exists, err := bd.Has(name, version)
	if err != nil {
		return errors.Trace(err)
	}
	if exists {
		return errors.AlreadyExistsf("%s-%s", name, version)
	}

	src, err := os.Open(filepath)
	if err != nil {
		return errors.Annotatef(err, "reading %q", filepath)
	}

	out := path.Join(bd.dir, fmt.Sprintf("%s-%s.bundle.tar", name, version))
	dst, err := os.Create(out)
	if err != nil {
		return errors.Annotatef(err, "creating %q", out)
	}
	_, err = io.Copy(dst, src)
	if err != nil {
		return errors.Annotatef(err, "copying %q to %q", filepath, out)
	}

	bd.entries[name] = append(bd.entries[name], version)
	sort.Strings(bd.entries[name])

	return nil
}

// GetChartDetails returns the details of a chart
func (bd *BundlesDir) GetChartDetails(name string, version string) (*types.ChartDetails, error) {
	return &types.ChartDetails{
		PublishedAt: utils.UnixEpoch,
		Digest:      "deadbeef",
	}, nil
}

// Reload reloads the index
func (bd *BundlesDir) Reload() error {
	return nil
}
