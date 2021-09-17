module github.com/bitnami-labs/charts-syncer

go 1.15

// Pin to specific version to not hit the next issue:
// Error:    vendor/github.com/docker/distribution/registry/registry.go:157:10: undefined: letsencrypt.Manager
replace github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d

require (
	github.com/bitnami-labs/pbjson v1.1.0
	github.com/containerd/containerd v1.4.4
	github.com/deislabs/oras v0.11.1
	github.com/docker/distribution v2.7.1+incompatible
	github.com/golang/protobuf v1.4.3
	github.com/google/go-cmp v0.5.2
	github.com/juju/errors v0.0.0-20200330140219-3fe23663418f
	github.com/juju/testing v0.0.0-20200923013621-75df6121fbb0 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mkmik/multierror v0.3.0
	github.com/opencontainers/image-spec v1.0.1
	github.com/philopon/go-toposort v0.0.0-20170620085441-9be86dbd762f
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.0
	google.golang.org/protobuf v1.25.0
	gopkg.in/yaml.v2 v2.4.0
	helm.sh/helm/v3 v3.6.1
	k8s.io/klog v1.0.0
	sigs.k8s.io/yaml v1.2.0
)
