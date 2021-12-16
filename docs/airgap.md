# Air gap scenario

In some situations you may want to sync two Helm chart repositories without direct connectivity between them. 
charts-syncer support this scenario via intermediate chart bundles.

An intermediate chart bundle is just a tarball containing the original chart code plus the container images needed to deploy the chart.

Below you can see how the whole process looks like:

## Step 1: Save chart bundles

In this first step charts-syncer needs access to the source charts repository. 

Create a config file like the following and save it to `config-save-bundles.yaml`

```yaml
source:
  repo:
    kind: HELM
    url: https://charts.trials.tac.bitnami.com/demo
target:
  intermediateBundlesPath: /path/to/local/output-dir
charts:
  - wordpress
```

Execute charts-syncer as follows:

```bash
charts-syncer sync --config ./config-save-bundles.yaml
```

Once charts-syncer finished preparing the bundles you will see them in the directory set in the `intermediateBundlesPath`
configuration property.

## Step 2: Move chart bundles

Move all the chart bundles from the machine with access to the source repo to the machine with access to the target repo. 

You can do it however you want.

## Step 3: Load chart bundles

In this first step charts-syncer needs access to the target charts repository.

Create a config file like the following and save it to `config-load-bundles.yaml`

```yaml
source:
  intermediateBundlesPath: /path/to/local/input-dir
target:
  containerRegistry: my.harbor.io
  containerRepository: my-project/containers
  repo:
    kind: OCI
    url: https://my.harbor.io/my-project/subpath
charts:
  - wordpress
```

Execute charts-syncer as follows:

```bash
charts-syncer sync --config ./config-load-bundles.yaml
```

Once charts-syncer finished preparing the bundles you will see them in the directory set in the `intermediateBundlesPath`
configuration property.