package core

import (
	"io/ioutil"

	"github.com/juju/errors"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/internal/cache"
	"github.com/bitnami-labs/charts-syncer/internal/utils"
	"github.com/bitnami-labs/charts-syncer/pkg/client/chartmuseum"
	"github.com/bitnami-labs/charts-syncer/pkg/client/harbor"
	"github.com/bitnami-labs/charts-syncer/pkg/client/helmclassic"
	"github.com/bitnami-labs/charts-syncer/pkg/client/local"
	"github.com/bitnami-labs/charts-syncer/pkg/client/oci"
	"github.com/bitnami-labs/charts-syncer/pkg/client/types"
	"helm.sh/helm/v3/pkg/chart"
)

// Reader defines the methods that a ReadOnly chart client should implement.
type Reader interface {
	Fetch(name string, version string) (string, error)
	List() ([]string, error)
	ListChartVersions(name string) ([]string, error)
	Has(name string, version string) (bool, error)
	GetChartDetails(name string, version string) (*types.ChartDetails, error)

	// Reload reloads or refresh the client-side data, in case it needs it
	Reload() error
}

// Writer defines the methods that a WriteOnly chart client should implement.
type Writer interface {
	Upload(filepath string, metadata *chart.Metadata) error
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
var NewClientV2 = func(repo *api.Repo, opts ...types.Option) (ClientV2, error) {
	copts := &types.ClientOpts{}
	for _, o := range opts {
		o(copts)
	}

	// Define cache dir if it hasn't been provided
	cacheDir := copts.GetCache()
	if cacheDir == "" {
		dir, err := ioutil.TempDir("", "client")
		if err != nil {
			return nil, errors.Annotatef(err, "creating temporary dir")
		}
		cacheDir = dir
	}
	c, err := cache.New(cacheDir, repo.GetUrl())
	if err != nil {
		return nil, errors.Annotatef(err, "allocating cache")
	}

	switch repo.Kind {
	case api.Kind_HELM:
		return helmclassic.New(repo, c)
	case api.Kind_CHARTMUSEUM:
		return chartmuseum.New(repo, c)
	case api.Kind_HARBOR:
		return harbor.New(repo, c)
	case api.Kind_OCI:
		return oci.New(repo, c)
	case api.Kind_LOCAL:
		return local.New(repo.Path)
	default:
		return nil, errors.Errorf("unsupported repo kind %q", repo.Kind)
	}
}
