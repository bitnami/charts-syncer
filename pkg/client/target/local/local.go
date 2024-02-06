// Package local provides a Unwrapper implementation for local chart sources
package local

import (
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

// Unwrap unwraps a chart. In local mode, we do not actually unwrap, we just copy over the file as
// we do not have a registry to write into the images and relocate
func (t *Target) Unwrap(file string, metadata *chart.Metadata, _ ...config.Option) error {
	return t.Repo.Upload(file, metadata)
}
