// Package source provides a client for chart and containers sources
package source

import (
	"github.com/juju/errors"

	"github.com/bitnami/charts-syncer/api"
	"github.com/bitnami/charts-syncer/pkg/client"

	"github.com/bitnami/charts-syncer/pkg/client/repo"
	"github.com/bitnami/charts-syncer/pkg/client/source/common"
	"github.com/bitnami/charts-syncer/pkg/client/source/local"
	"github.com/bitnami/charts-syncer/pkg/client/types"
)

// NewClient returns a Client object
func NewClient(source *api.Source, opts ...types.Option) (client.ChartsWrapper, error) {
	copts := &types.ClientOpts{}
	for _, o := range opts {
		o(copts)
	}
	r := source.GetRepo()
	insecure := copts.GetInsecure()
	usePlainHTTP := copts.GetUsePlainHTTP()

	if r.Kind == api.Kind_LOCAL {
		return local.New(r.Path)
	}

	c, err := repo.NewClient(r, opts...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return common.New(source, c, insecure, usePlainHTTP)
}
