// Package local provides a Wrapper implementation for local chart sources
package local

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/bitnami/charts-syncer/internal/utils"
	"github.com/bitnami/charts-syncer/pkg/client/config"
	"github.com/bitnami/charts-syncer/pkg/client/repo/local"
	"github.com/google/go-containerregistry/pkg/name"
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

// WrapChart wraps a chart. In local mode, we do not actually wrap, we just copy over the file as
// we already operate over wrapped charts
func (t *Source) WrapChart(tgz, dest string, _ ...config.Option) (string, error) {
	if err := utils.CopyFile(dest, tgz); err != nil {
		return "", errors.Trace(err)
	}
	return tgz, nil
}

// WrapContainer wraps a container. In local mode, the container wrap file already exists on disk,
// so we just copy it to the destination path.
func (t *Source) WrapContainer(imageRef string, destination string, _ ...config.Option) (string, error) {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return "", errors.Annotatef(err, "parsing container image reference %q", imageRef)
	}

	imageName := path.Base(ref.Context().RepositoryStr())
	tag := ref.Identifier()
	src := filepath.Join(t.Dir(), "containers", fmt.Sprintf("%s-%s.container.wrap.tgz", imageName, tag))

	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return "", errors.Annotatef(err, "creating destination directory for %q", destination)
	}

	if err := utils.CopyFile(destination, src); err != nil {
		return "", errors.Annotatef(err, "copying container wrap file %q", src)
	}

	return destination, nil
}
