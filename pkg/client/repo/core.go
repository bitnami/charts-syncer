// Package repo creates a new client for a given repository
package repo

import (
	"net/url"
	"os"

	"github.com/juju/errors"

	apiv1 "github.com/bitnami/charts-syncer/gen/proto/v1"
	"github.com/bitnami/charts-syncer/internal/cache/cachedisk"
	"github.com/bitnami/charts-syncer/pkg/client"
	"github.com/bitnami/charts-syncer/pkg/client/repo/chartmuseum"
	"github.com/bitnami/charts-syncer/pkg/client/repo/harbor"
	"github.com/bitnami/charts-syncer/pkg/client/repo/helmclassic"
	"github.com/bitnami/charts-syncer/pkg/client/repo/local"

	"github.com/bitnami/charts-syncer/pkg/client/repo/oci"
	"github.com/bitnami/charts-syncer/pkg/client/types"
)

// NewClient returns a Client object
func NewClient(repo *apiv1.Repo, opts ...types.Option) (client.ChartsReaderWriter, error) {
	copts := &types.ClientOpts{}
	for _, o := range opts {
		o(copts)
	}

	insecure := copts.GetInsecure()
	usePlainHTTP := copts.GetUsePlainHTTP()
	// Define cache dir if it hasn't been provided
	cacheDir := copts.GetCache()
	if cacheDir == "" {
		dir, err := os.MkdirTemp("", "client")
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
	case apiv1.Kind_HELM:
		return helmclassic.New(repo, c, insecure)
	case apiv1.Kind_CHARTMUSEUM:
		return chartmuseum.New(repo, c, insecure)
	case apiv1.Kind_HARBOR:
		return harbor.New(repo, c, insecure)
	case apiv1.Kind_OCI:
		return oci.New(repo, c, insecure, usePlainHTTP)
	case apiv1.Kind_LOCAL:
		return local.New(repo.Path)
	default:
		return nil, errors.Errorf("unsupported repo kind %q", repo.Kind)
	}
}

// NewContainerClient returns a Client object
func NewContainerClient(containers *apiv1.Containers, opts ...types.Option) (client.ContainersReaderWriter, error) {
	copts := &types.ClientOpts{}
	for _, o := range opts {
		o(copts)
	}

	insecure := copts.GetInsecure()
	usePlainHTTP := copts.GetUsePlainHTTP()
	// Define cache dir if it hasn't been provided
	cacheDir := copts.GetCache()
	if cacheDir == "" {
		dir, err := os.MkdirTemp("", "client")
		if err != nil {
			return nil, errors.Annotatef(err, "creating temporary dir")
		}
		cacheDir = dir
	}
	c, err := cachedisk.New(cacheDir, containers.GetUrl())
	if err != nil {
		return nil, errors.Annotatef(err, "allocating cache")
	}

	entries := make(map[string][]string)
	u, err := url.Parse(containers.GetUrl())
	if err != nil {
		return nil, errors.Trace(err)
	}
	resolver := oci.NewDockerResolver(u, containers.GetAuth().GetUsername(), containers.GetAuth().GetPassword(), insecure)

	return oci.NewRaw(u, containers.GetAuth().GetUsername(), containers.GetAuth().GetPassword(), c, insecure, usePlainHTTP, entries, resolver)
}
