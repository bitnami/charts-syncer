// Package target provides a client for chart and containers targets
package target

import (
	"github.com/juju/errors"

	apiv1 "github.com/bitnami/charts-syncer/gen/proto/v1"
	"github.com/bitnami/charts-syncer/pkg/client"

	"github.com/bitnami/charts-syncer/pkg/client/repo"
	"github.com/bitnami/charts-syncer/pkg/client/target/common"
	"github.com/bitnami/charts-syncer/pkg/client/target/local"

	"github.com/bitnami/charts-syncer/pkg/client/types"
)

// NewClient returns a Client object
func NewClient(target *apiv1.Target, opts ...types.Option) (client.ChartsUnwrapper, error) {
	copts := &types.ClientOpts{}
	for _, o := range opts {
		o(copts)
	}
	r := target.GetRepo()
	insecure := copts.GetInsecure()
	usePlainHTTP := copts.GetUsePlainHTTP()

	if r.Kind == apiv1.Kind_LOCAL {
		return local.New(r.Path)
	}

	c, err := repo.NewClient(r, opts...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return common.New(target, c, insecure, usePlainHTTP)
}

// NewContainerClient returns a Client object
func NewContainerClient(target *apiv1.Target, opts ...types.Option) (client.ContainersUnwrapper, error) {
	copts := &types.ClientOpts{}
	for _, o := range opts {
		o(copts)
	}
	insecure := copts.GetInsecure()
	usePlainHTTP := copts.GetUsePlainHTTP()

	if r := target.GetRepo(); r != nil && r.Kind == apiv1.Kind_LOCAL {
		return local.New(r.Path)
	}

	c, err := repo.NewContainerClient(target.GetContainers(), opts...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return common.NewContainer(target, c, insecure, usePlainHTTP)
}
