# Air gap scenario

In some situations you may want to sync two Helm chart repositories without direct connectivity between them. 
charts-syncer support this scenario via using the LOCAL kind.

When using LOCAL target kind, the chart-syncer will save all the synced charts wrapped into the defined directory. These wraps contain the original chart as well as the contained images and Image.lock file. You can find additional information about wraps in the [dt tool repository](https://github.com/vmware-labs/distribution-tooling-for-helm)

This directory enables a two-setp process where it will be used as the source of the new executin, allowing publishing all charts into your air-gap target environments.

As you can see in the diagram below, the wrapped charts are the only bits that ever live in both the source and target environments.
The environment A (source) does not know about the final location (environment B) of the Charts/Container images and the other way around.

![two steps relocation](./assets/two-steps-relocation.jpg)

## Relocation process

An outline of the process looks like

1. Run charts-syncer to import the desired Helm Charts into local directory
2. Manually move the previously generated directory containing the `*.wrap.tgz` wraps to the target environment
3. Run charts-syncer (for a second time) but this time using the wraps directory as the source

### Step 1: Save Chart Bundles

In this first step charts-syncer needs access to the source charts repository and container images registry

Create a config file like the following and save it to `config-save-bundles.yaml`

```yaml
source:
  repo:
    kind: HELM # or as any other supported Helm Chart repository kinds
    url: https://charts.trials.tac.bitnami.com/demo
    ## Helm repository credentials. Alternatively you can use environmental variables
    # auth:
    #   username: [USERNAME]
    #   password: [PASSWORD]  
  ## Container registry authentication
  # containers:
  #   auth:
  #     registry: [URL] # i.e my.harbor.io
  #     username: [USERNAME]
  #     password: [PASSWORD]    
target:
  # This instructs charts-syncer to store the wrapped charts in the given directory
  repo:
    kind: LOCAL
    path: /tmp/chart-bundles-dir
```

Execute charts-syncer as follows:

```bash
charts-syncer sync --config ./config-save-bundles.yaml
```

Once charts-syncer finishes preparing the wraps you will see them in the directory set in the `path`
configuration property.

The directory will contain a Chart Wrap for every imported Helm Chart.
> :warning: IMPORTANT: The content of the bundle is not meant to be used directly but instead as the input for the step 2 of the process (or consumed by [dt](https://github.com/vmware-labs/distribution-tooling-for-helm)).

### Step 2: Move Chart Wraps directory

Use your method of choice to move all the Chart Wraps from the machine with access to the source repo to the machine with access to the target repo.

### Step 3: Load Chart Wraps

In this final step charts-syncer needs access to the target charts repository and container images registry.
The tool will iterate over each of the Chart Wraps re-tagging the container images and pushing them to the final container registry.
Additionally, it will re-write the chart code to pull the images from the target container registry.

Create a config file like the following and save it to `config-load-bundles.yaml`

```yaml
# Note how the source this time is the directory containing the bundles
source:
  repo:
    kind: LOCAL
    path:  /tmp/chart-bundles-dir
target:
  repo:
    kind: OCI # or as any other supported Helm Chart repository kinds
    url: https://my.harbor.io/my-project/subpath
    # auth:
    #   username: [USERNAME]
    #   password: [PASSWORD]
  ## Container registry authentication
  # containers:
  #   auth: 
  #     registry: [URL] # i.e my.harbor.io
  #     username: [USERNAME]
  #     password: [PASSWORD]   
```

Execute charts-syncer as usual:

```bash
charts-syncer sync --config ./config-load-bundles.yaml
```

Once charts-syncer finishes all your charts and images should be pushed to the configured chart repository and container registry.
