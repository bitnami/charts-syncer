package core

import (
	"github.com/juju/errors"

	"github.com/bitnami-labs/charts-syncer/pkg/utils"
)

// Reader defines the methods that a ReadOnly chart client should implement.
type Reader interface {
	Fetch(filepath string, name string, version string) error
	List() ([]string, error)
	ListChartVersions(names ...string) ([]string, error)
	Has(name string, version string) (bool, error)
}

// Writer defines the methods that a WriteOnly chart client should implement.
type Writer interface {
	Push(filepath string) error
}

// ValidateChartTgz validates if a chart is a valid tgz file
func ValidateChartTgz(filepath string) error {
	contentType, err := utils.GetFileContentType(filepath)
	if err != nil {
		return errors.Trace(err)
	}
	if contentType != "application/x-gzip" {
		return errors.Errorf("%q is not a gzipped tarball", filepath)
	}
	return nil
}
