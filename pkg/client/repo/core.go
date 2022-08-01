package repo

import (
	"io/ioutil"

	"github.com/juju/errors"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/internal/cache/cachedisk"
	"github.com/bitnami-labs/charts-syncer/pkg/client"
	"github.com/bitnami-labs/charts-syncer/pkg/client/repo/chartmuseum"
	"github.com/bitnami-labs/charts-syncer/pkg/client/repo/harbor"
	"github.com/bitnami-labs/charts-syncer/pkg/client/repo/helmclassic"
	"github.com/bitnami-labs/charts-syncer/pkg/client/repo/local"
	"github.com/bitnami-labs/charts-syncer/pkg/client/repo/oci"
	"github.com/bitnami-labs/charts-syncer/pkg/client/types"
)

// NewClient returns a Client object
func NewClient(repo *api.Repo, opts ...types.Option) (client.ChartsReaderWriter, error) {
	copts := &types.ClientOpts{}
	for _, o := range opts {
		o(copts)
	}

	insecure := copts.GetInsecure()
	// Define cache dir if it hasn't been provided
	cacheDir := copts.GetCache()
	if cacheDir == "" {
		dir, err := ioutil.TempDir("", "client")
		if err != nil {
			return nil, errors.Annotatef(err, "creating temporary dir")
		}
		cacheDir = dir
	}
	c, err := cachedisk.New(cacheDir, repo.GetUrl())
	if err != nil {
		return nil, errors.Annotatef(err, "allocating cache")
	}

	switch repo.Kind {
	case api.Kind_HELM:
		return helmclassic.New(repo, c, insecure)
	case api.Kind_CHARTMUSEUM:
		return chartmuseum.New(repo, c, insecure)
	case api.Kind_HARBOR:
		return harbor.New(repo, c, insecure)
	case api.Kind_OCI:
		return oci.New(repo, c, insecure)
	case api.Kind_LOCAL:
		return local.New(repo.Path)
	default:
		return nil, errors.Errorf("unsupported repo kind %q", repo.Kind)
	}
}
