module github.com/bitnami-labs/charts-syncer

go 1.15

replace github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d

require (
	github.com/bitnami-labs/pbjson v1.1.0
	github.com/containerd/containerd v1.3.2
	github.com/deislabs/oras v0.8.1
	github.com/golang/protobuf v1.4.2
	github.com/google/go-cmp v0.5.0
	github.com/juju/errors v0.0.0-20200330140219-3fe23663418f
	github.com/juju/testing v0.0.0-20200923013621-75df6121fbb0 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mkmik/multierror v0.3.0
	github.com/opencontainers/image-spec v1.0.1
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/philopon/go-toposort v0.0.0-20170620085441-9be86dbd762f
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.7.0
	google.golang.org/protobuf v1.25.0
	gopkg.in/yaml.v2 v2.3.0
	helm.sh/helm/v3 v3.2.1
	k8s.io/klog v1.0.0
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/yaml v1.2.0
)
