// Package source provides a client for chart and containers sources
package source

import (
	"github.com/juju/errors"

	apiv1 "github.com/bitnami/charts-syncer/gen/proto/v1"
	"github.com/bitnami/charts-syncer/pkg/client"

	"github.com/bitnami/charts-syncer/pkg/client/repo"
	"github.com/bitnami/charts-syncer/pkg/client/source/common"
	"github.com/bitnami/charts-syncer/pkg/client/source/local"
	"github.com/bitnami/charts-syncer/pkg/client/types"
)

// NewClient returns a Client object
func NewClient(source *apiv1.Source, opts ...types.Option) (client.ChartsWrapper, error) {
	copts := &types.ClientOpts{}
	for _, o := range opts {
		o(copts)
	}
	r := source.GetRepo()
	insecure := copts.GetInsecure()
	usePlainHTTP := copts.GetUsePlainHTTP()

	if r.Kind == apiv1.Kind_LOCAL {
		return local.New(r.Path)
	}

	c, err := repo.NewClient(r, opts...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return common.New(source, c, insecure, usePlainHTTP)
}
