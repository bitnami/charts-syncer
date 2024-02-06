// Package local provides a Wrapper implementation for local chart sources
package local

import (
	"github.com/bitnami/charts-syncer/internal/utils"
	"github.com/bitnami/charts-syncer/pkg/client/config"
	"github.com/bitnami/charts-syncer/pkg/client/repo/local"
	"github.com/juju/errors"
)

// Source allows to operate a local chart source
type Source struct {
	*local.Repo
}

// New creates a Repo object from an api.Repo object.
func New(dir string) (*Source, error) {
	r, err := local.New(dir)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &Source{Repo: r}, nil
}

// Wrap wraps a chart. In local mode, we do not actually wrap, we just copy over the file as
// we already operate over wrapped charts
func (t *Source) Wrap(tgz, dest string, _ ...config.Option) (string, error) {
	if err := utils.CopyFile(dest, tgz); err != nil {
		return "", errors.Trace(err)
	}
	return tgz, nil
}
