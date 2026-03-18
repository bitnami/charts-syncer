// Package local provides a Unwrapper implementation for local chart sources
package local

import (
	"path/filepath"

	"github.com/bitnami/charts-syncer/internal/utils"
	"github.com/bitnami/charts-syncer/pkg/client/config"
	"github.com/bitnami/charts-syncer/pkg/client/repo/local"
	"github.com/juju/errors"
	"helm.sh/helm/v3/pkg/chart"
)

// Target allows to operate a local chart target
type Target struct {
	*local.Repo
}

// New creates a Repo object from an api.Repo object.
func New(dir string) (*Target, error) {
	r, err := local.New(dir)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &Target{Repo: r}, nil
}

// UnwrapChart unwraps a chart. In local mode, we do not actually unwrap, we just copy over the file as
// we do not have a registry to write into the images and relocate
func (t *Target) UnwrapChart(file string, metadata *chart.Metadata, _ ...config.Option) error {
	return t.Upload(file, metadata)
}

// UnwrapContainer unwraps a container. In local mode, we do not actually unwrap, we just copy over the file as
// we do not have a registry to write into the images and relocate
func (t *Target) UnwrapContainer(file string, _ ...config.Option) error {
	dest := filepath.Join(t.Dir(), "containers", filepath.Base(file))
	return errors.Trace(utils.CopyFile(dest, file))
}
