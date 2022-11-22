# Asset Relocation Tool for Kubernetes

The Asset Relocation Tool for Kubernetes is a tool used for relocating Kubernetes assets from one place to another.
It's first focus is on relocating Helm Charts, which is done by:
1. Copying the container images referenced in the chart to a new image registry, and 
2. Modifying the chart with the updated image references.

The tool comes in the form of a CLI, named `relok8s`.

## Running relok8s

```bash
$ relok8s chart move mysql-8.5.8.tgz --image-patterns mysql.images.yaml --registry harbor-repo.vmware.com
Pulling docker.io/bitnami/mysql:8.0.25-debian-10-r0... Done

Images to be pushed:
  harbor-repo.vmware.com/bitnami/mysql:8.0.25-debian-10-r0 (sha256:ae8c4c719352a58abc99c866986ee11578bc43e90d794c6705f7b1eb12c7289e)

Changes written to mysql/values.yaml:
  .image.registry: harbor-repo.vmware.com
Would you like to proceed? (y/N)
y
Pushing harbor-repo.vmware.com/bitnami/mysql:8.0.25-debian-10-r0...Done

New chart: mysql-8.5.8.rewritten.tgz
```

## Inputs

The Asset Relocation Tool for Kubernetes requires a few inputs for the various commands.

### Helm Chart

Each command requires a Helm chart.
The chart can be in directory format, or TGZ bundle.
It can contain dependent charts.

### Image Hints File

The tool requires an image hints file which
will be used to determine the list of images encoded in the Helm chart.

This file can be either explicitly provided to the relok8s tool at runtime or **embedded inside the Helm Chart with the name `.relok8s-images.yaml`**

```yaml
---
- "{{ .image }}:{{ .tag }}",
- "{{ .proxy.image }}:{{ .proxy.tag }}",
# You can also reference subcharts by prepending the subchart name
- "{{ .mysubchart.image.repository }}@{{ .mysubchart.image.digest }}",
```

The content is a list of string patterns referencing each container image path encoded
in the chart/subcharts `values.yaml` files. Both :tags and @digest formats are allowed.
To reference images encoded inside a dependent chart, the first key should be the dependent chart's name.

For more information refer to [this example](examples/chart-with-subcharts).

### Rules

The Asset Relocation Tool for Kubernetes allows for two rules to be specified on the command line:

#### Registry
```bash
--registry <registry>
```
This overwrites the image registry

#### Repository Prefix
```bash
--repo-prefix <string>
```
This modifies the image repository name for all parts except for the final word.

Rule                | Example                   | Input                             | Output
------------------- | ------------------------- | --------------------------------- | -----------------------------------------------
Registry            | `harbor-repo.vmware.com`  | `docker.io/mycompany/myapp:1.2.3` | `harbor-repo.vmware.com/mycompany/myapp:1.2.3`
Repository Prefix   | `mytenant`                | `docker.io/mycompany/myapp:1.2.3` | `docker.io/mytenant/myapp:1.2.3`

## Installation

The latest version of the relok8s binary can be found in the [releases section](https://github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/releases). Additionally a containerized version can be also found [here](https://github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/pkgs/container/asset-relocation-tool-for-kubernetes)

## Running in CI

It may be useful to run `relok8s` inside a CI pipeline to automatically move a chart when there are updates.
An example [Concourse](https://concourse-ci.org/) pipeline can be found here: [docs/example-pipeline.yaml](docs/example-pipeline.yaml)

## Building

Building the tool from source is very simple with: 

```bash
$ make build
go build -o build/relok8s -ldflags "-X github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/cmd.Version=dev" ./main.go
pwall@pwall-a01:~/src/vmware-tanzu/asset-relocation-tool-for-kubernetes $ ls ./build/relok8s 
./build/relok8s
```

## Development

See [Development](DEVELOPMENT.md)
