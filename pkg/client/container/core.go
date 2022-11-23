package container

import (
	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/client"
	"github.com/bitnami-labs/charts-syncer/pkg/client/container/harbor"
	"github.com/bitnami-labs/charts-syncer/pkg/client/types"
)

// NewClient returns a Client object
func NewClient(registry string, container *api.Containers, opts ...types.Option) (client.ContainersWriter, error) {
	copts := &types.ClientOpts{}
	for _, o := range opts {
		o(copts)
	}

	insecure := copts.GetInsecure()
	return harbor.New(registry, container, insecure)
}
