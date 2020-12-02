package core

import (
	"github.com/juju/errors"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/internal/utils"
	"github.com/bitnami-labs/charts-syncer/pkg/client/chartmuseum"
	"github.com/bitnami-labs/charts-syncer/pkg/client/harbor"
	"github.com/bitnami-labs/charts-syncer/pkg/client/helmclassic"
	"github.com/bitnami-labs/charts-syncer/pkg/client/types"
)

// Reader defines the methods that a ReadOnly chart client should implement.
type Reader interface {
	Fetch(filepath string, name string, version string) error
	List() ([]string, error)
	ListChartVersions(name string) ([]string, error)
	Has(name string, version string) (bool, error)
	GetChartDetails(name string, version string) (*types.ChartDetails, error)

	// Reload reloads or refresh the client-side data, in case it needs it
	Reload() error
}

// Writer defines the methods that a WriteOnly chart client should implement.
type Writer interface {
	Upload(filepath string, name string, version string) error
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

// ClientV2 defines the methods that a chart client should implement.
type ClientV2 interface {
	Reader
	Writer
}

// NewClientV2 returns a ClientV2 object
//
// The func is exposed as a var to allow tests to temporarily replace its
// implementation, e.g. to return a fake.
var NewClientV2 = func(repo *api.Repo) (ClientV2, error) {
	switch repo.Kind {
	case api.Kind_HELM:
		return helmclassic.New(repo)
	case api.Kind_CHARTMUSEUM:
		return chartmuseum.New(repo)
	case api.Kind_HARBOR:
		return harbor.New(repo)
	default:
		return nil, errors.Errorf("unsupported repo kind %q", repo.Kind)
	}
}
