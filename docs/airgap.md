# Air gap scenario

In some situations you may want to sync two Helm chart repositories without direct connectivity between them. 
charts-syncer support this scenario via intermediate Chart Bundles.

An intermediate Chart Bundle is just a tarball containing the original chart code plus the container images needed to deploy the chart.

This intermediate Chart Bundle enables a two-step process where its content will be used to relocate the container images 
and the rewritten Helm Chart without contacting the source Container images registry or Helm repository. 
Making the relocation process suitable for disconnected, air-gap target environments

Below you can see how the whole process looks like:

## Step 1: Save Chart Bundles

In this first step charts-syncer needs access to the source charts repository and container images registry

Create a config file like the following and save it to `config-save-bundles.yaml`

```yaml
source:
  repo:
    kind: HELM
    url: https://charts.trials.tac.bitnami.com/demo
target:
  intermediateBundlesPath: /tmp/chart-bundles-dir
```

Execute charts-syncer as follows:

```bash
charts-syncer sync --config ./config-save-bundles.yaml
```

Once charts-syncer finishes preparing the bundles you will see them in the directory set in the `intermediateBundlesPath`
configuration property.

The directory will contain an intermediate Chart Bundle for every imported Helm Chart.
> :warning: IMPORTANT: The content of the bundle is not meant to be used directly but instead as the input for the step 2 of the process.

## Step 2: Move Chart Bundles

Move all the Chart Bundles from the machine with access to the source repo to the machine with access to the target repo.

## Step 3: Load Chart Bundles

In this final step charts-syncer needs access to the target charts repository and container images registry.
The tool will iterate over each of the Chart Bundle re-tagging the container images and pushing them to the final container registry.
Additionally, it will re-write the chart code to pull the images from the target container registry.

Create a config file like the following and save it to `config-load-bundles.yaml`

```yaml
source:
  intermediateBundlesPath: /tmp/chart-bundles-dir
target:
  containerRegistry: my.harbor.io
  containerRepository: my-project/containers
  repo:
    kind: OCI
    url: https://my.harbor.io/my-project/subpath
```

Execute charts-syncer as usual:

```bash
charts-syncer sync --config ./config-load-bundles.yaml
```

Once charts-syncer finishes all your charts and images should be pushed to the configured chart repository and container registry.