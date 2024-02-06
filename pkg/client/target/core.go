// Package target provides a client for chart and containers targets
package target

import (
	"github.com/juju/errors"

	"github.com/bitnami/charts-syncer/api"
	"github.com/bitnami/charts-syncer/pkg/client"

	"github.com/bitnami/charts-syncer/pkg/client/repo"
	"github.com/bitnami/charts-syncer/pkg/client/target/common"
	"github.com/bitnami/charts-syncer/pkg/client/target/local"

	"github.com/bitnami/charts-syncer/pkg/client/types"
)

// NewClient returns a Client object
func NewClient(target *api.Target, opts ...types.Option) (client.ChartsUnwrapper, error) {
	copts := &types.ClientOpts{}
	for _, o := range opts {
		o(copts)
	}
	r := target.GetRepo()
	insecure := copts.GetInsecure()
	usePlainHTTP := copts.GetUsePlainHTTP()

	if r.Kind == api.Kind_LOCAL {
		return local.New(r.Path)
	}

	c, err := repo.NewClient(r, opts...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return common.New(target, c, insecure, usePlainHTTP)
}
