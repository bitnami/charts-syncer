package indexer

import (
	"net/url"

	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"

	"github.com/bitnami-labs/charts-syncer/internal/utils"
)

func newDockerResolver(u *url.URL, username, password string, insecure bool) remotes.Resolver {
	client := utils.DefaultClient
	if insecure {
		client = utils.InsecureClient
	}
	opts := docker.ResolverOptions{
		Hosts: func(s string) ([]docker.RegistryHost, error) {
			return []docker.RegistryHost{
				{
					Authorizer: docker.NewDockerAuthorizer(
						docker.WithAuthCreds(func(s string) (string, string, error) {
							return username, password, nil
						})),
					Host:         u.Host,
					Scheme:       u.Scheme,
					Path:         "/v2",
					Capabilities: docker.HostCapabilityPull | docker.HostCapabilityResolve | docker.HostCapabilityPush,
					Client:       client,
				},
			}, nil
		},
	}

	return docker.NewResolver(opts)
}
