module github.com/bitnami-labs/charts-syncer

go 1.16

// Needed so we can require asset-relocation-tool-for-kubernetes packages
// https://github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/issues/89
replace gopkg.in/yaml.v3 => github.com/atomatt/yaml v0.0.0-20200403124456-7b932d16ab90

require (
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/bitnami-labs/pbjson v1.1.0
	github.com/containerd/containerd v1.5.9
	github.com/distribution/distribution/v3 v3.0.0-20210804104954-38ab4c606ee3
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.6
	github.com/google/go-containerregistry v0.7.0
	github.com/juju/errors v0.0.0-20200330140219-3fe23663418f
	github.com/juju/testing v0.0.0-20200923013621-75df6121fbb0 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mkmik/multierror v0.3.0
	github.com/opencontainers/image-spec v1.0.2
	github.com/philopon/go-toposort v0.0.0-20170620085441-9be86dbd762f
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/viper v1.8.1
	github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes v0.3.60
	google.golang.org/protobuf v1.27.1
	gopkg.in/yaml.v2 v2.4.0
	helm.sh/helm/v3 v3.7.0
	k8s.io/klog v1.0.0
	oras.land/oras-go v0.4.0
	sigs.k8s.io/yaml v1.2.0
)
