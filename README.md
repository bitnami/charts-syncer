[![CircleCI](https://circleci.com/gh/bitnami-labs/charts-syncer.svg?style=svg&circle-token=91105ed254723ef1e3af739f6d31dc845136828c)](https://circleci.com/gh/bitnami-labs/charts-syncer/tree/master)


# charts-syncer

Sync chart packages between chart repositories

# Table of Contents

- [Usage](#usage)
    + [Sync all charts](#sync-all-helm-charts)
    + [Sync all charts from specific date](#sync-all-charts-from-specific-date)
- [Advanced Usage](#advanced-usage)
    + [Sync charts and container images](#sync-charts-and-container-images)
    + [Sync charts between repositories without direct connectivity](#sync-charts-between-repositories-without-direct-connectivity)
- [Configuration](#configuration)
  * [Harbor example](#harbor-example)
  * [OCI example](#oci-example)
  * [Local example](#local-example)
- [Requirements](#requirements)
- [Changes performed in a chart](#changes-performed-in-a-chart)
    + [Update *values.yaml* and *values-production.yaml* (if exists)](#update--valuesyaml--and--values-productionyaml---if-exists-)
    + [Update dependencies files](#update-dependencies-files)
    + [Update *README.md*](#update--readmemd-)
    + [values.yaml](#valuesyaml)
    + [requirements.lock (only for Helm v2 charts)](#requirementslock--only-for-helm-v2-charts-)
    + [Chart.lock (only for Helm v3 charts)](#chartlock--only-for-helm-v3-charts-)
    + [README.md](#readmemd)
- [How to build](#how-to-build)
- [Deploy to Kubernetes](#deploy-to-kubernetes)
  * [Manage credentials](#manage-credentials)

## Usage

### Sync all Helm Charts

~~~bash
$ charts-syncer sync
~~~

### Sync Helm Charts from a specific date

~~~bash
$ charts-syncer sync --from-date 2020-05-15
~~~

### Sync latest version of each Helm Chart

~~~bash
$ charts-syncer sync --latest-version-only
~~~

## Advanced Usage

### Sync Helm Charts and Container Images

By default, charts-syncer only sync Helm Charts packages, it does not copy the container images referenced by the chart. This
feature can be enabled by setting the `relocateContainerImages: true` property in the config file i.e

~~~yaml
# leverage .relok8s-images.yaml file inside the Charts to move the container images too
relocateContainerImages: true
source:
   ...
target:
   ...
~~~

In order for this option to work it is required that the source Helm Charts includes a `.relok8s-images.yaml` file with information
about where to find the images inside chart. For more information about this file please refer to [asset-relocation-tool-for-kubernetes readme](https://github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes#image-hints-file).

### Sync Helm Charts and associated container images between disconnected environments

There are scenarios where the source and target Helm Charts repositories are not reachable at the same time from the same location.

For those cases, charts-syncer supports a two steps relocation for offline Chart and container images transport, check the [air gap docs](docs/airgap.md).

----

## Configuration

Below you can find an example configuration file. To know all the available configuration keys see the [charts-syncer](./charts-syncer.yaml) file as it includes explanatory comments for each configuration key.

~~~yaml
#
# Example config file
#
source:
  repo:
    kind: HELM
    url: http://localhost:8080
    # Helm repository authentication, same for other repo types i.e OCI
    # auth:
    #   username: "USERNAME"
    #   password: "PASSWORD"
  # Container images registry authn
  # containers:
  #  auth:
  #     registry: 'REGISTRY'
  #     username: "USERNAME"
  #     password: "PASSWORD"
target:
  repoName: myrepo
  containerRegistry: k8s.container.registry.io
  containerRepository: repository/demo/k8s
  # Container images registry authn
  # containers:
  #   auth:
  #     username: "USERNAME"
  #     password: "PASSWORD"
  repo:
    kind: CHARTMUSEUM
    url: http://localhost:9090
    # Helm repository authentication
    # auth:
    #   username: "USERNAME"
    #   password: "PASSWORD"
charts:
  - redis
  - mariadb
# opt-out counterpart of "charts" property that explicit list the Helm charts to be skipped 
# either "charts" or "skipCharts" can be used at once
# skipCharts:
#  - mariadb
~~~

> Note that the `repo.url` property you need to specify is the same one you would use to add the repo to helm with the `helm repo add command`.
>
> Example: `helm repo add bitnami https://charts.bitnami.com/bitnami`.

Credentials for the Helm Chart repositories and container images registries can be provided using config file or the following environment variables:

Helm Chart repositories

- `SOURCE_REPO_AUTH_USERNAME`
- `SOURCE_REPO_AUTH_PASSWORD`


- `TARGET_REPO_AUTH_USERNAME`
- `TARGET_REPO_AUTH_PASSWORD`

Container images registries

- `SOURCE_CONTAINERS_AUTH_REGISTRY`
- `SOURCE_CONTAINERS_AUTH_USERNAME`
- `SOURCE_CONTAINERS_AUTH_PASSWORD`


- `TARGET_CONTAINERS_AUTH_USERNAME`
- `TARGET_CONTAINERS_AUTH_PASSWORD`

Current available Kinds are `HELM`, `CHARTMUSEUM`, `HARBOR` and `OCI`. Below you can find the compatibility matrix between source and targets repositories.

| Source Repo | Target Repo | Supported          |
|-------------|-------------|--------------------|
| HELM        | HELM        | :x:                |
| HELM        | CHARTMUSEUM | :white_check_mark: |
| HELM        | HARBOR      | :white_check_mark: |
| HELM        | OCI         | :white_check_mark: |
| HELM        | LOCAL       | :white_check_mark: |
| CHARTMUSEUM | HELM        | :x:                |
| CHARTMUSEUM | CHARTMUSEUM | :white_check_mark: |
| CHARTMUSEUM | HARBOR      | :white_check_mark: |
| CHARTMUSEUM | OCI         | :white_check_mark: |
| CHARTMUSEUM | LOCAL       | :white_check_mark: |
| HARBOR      | HELM        | :x:                |
| HARBOR      | CHARTMUSEUM | :white_check_mark: |
| HARBOR      | HARBOR      | :white_check_mark: |
| HARBOR      | OCI         | :white_check_mark: |
| HARBOR      | LOCAL       | :white_check_mark: |
| OCI         | HELM        | :x:                |
| OCI         | CHARTMUSEUM | :white_check_mark: |
| OCI         | HARBOR      | :white_check_mark: |
| OCI         | OCI         | :white_check_mark: |
| OCI         | LOCAL       | :white_check_mark: |
| LOCAL       | HELM        | :x:                |
| LOCAL       | CHARTMUSEUM | :white_check_mark: |
| LOCAL       | HARBOR      | :white_check_mark: |
| LOCAL       | OCI         | :white_check_mark: |
| LOCAL       | LOCAL       | :white_check_mark: |


> The list of charts in the config file is optional except for OCI repositories used as source.
> The rest of chart repositories kinds already support autodiscovery.

### Harbor example

In the case of HARBOR kind repos, be aware that chart repository URLs are:

https://$HARBOR_DOMAIN/chartrepo/$HARBOR_PROJECT

So if HARBOR_DOMAIN=my.harbor.com and HARBOR_PROJECT=my-project, you would need to specify this repo in the config file like:

~~~yaml
target:
 repo:
   kind: HARBOR
   url: https://my.harbor.com/chartrepo/my-project
~~~

### OCI example

Since Harbor 2.0.0, there are two ways of storing charts. The legacy one uses chartmuseum under the hood and it corresponds to the HARBOR kind of this project.
The new one however uses OCI to store helm charts as OCI artifacts. In case you are using Harbor with OCI backend you can use the following example:

~~~yaml
target:
 repo:
   kind: OCI
   url: https://my.harbor.com/my-project/subpath
~~~

`subpath` in the previous url is optional in case your charts are not stored directly under your projects.
It is worth mentioning that you can use Harbor robot accounts using OCI registries as source or target.

Also, take into account that if you use OCI as the source repository you must specify the list of charts to synchronize
or a pointer to a [charts index file](#charts-index-for-oci-based-repositories) in the repository.

#### Charts index for OCI-based repositories

By using a charts index file for OCI-Based repository you won't need to maintain a hardcoded list of chart names in the config file.
charts-syncer will be able to auto-discover what charts need to be synchronized.

By default, the library will look up for a "charts-index:latest" charts index artifact within the source OCI repository.

However, this can be customized using the `chartsIndex` field using the format `REGISTRY/PROJECT/[SUBPATH][:TAG|@sha256:DIGEST]`.

For example, if your URL is `https://my-oci-registry.io/my-project/subpath` and no `chartsIndex` is specified, charts-syncer will try to use 
`my-oci-registry.io/my-project/subpath/charts-index:latest` asset as index if it exists.

An example of the valid index format can be seen directly in its [Protobuf definition](internal/indexer/api/index.proto). Worth to mention 
that the format of the charts index for OCI repositories is a custom one, not a traditional Helm index file.

~~~yaml
source:
 repo:
   kind: OCI
   url: https://my-oci-registry.io/my-project/subpath
   # disableChartsIndex: true
   # Charts index location override, charts-index:latest by default
   chartsIndex: my-oci-registry.io/my-project/my-custom-index:prod
~~~

Finally, if no charts index is found, charts-syncer will require the list of charts in the config file:

~~~yaml
source:
  repo:
    kind: OCI
    url: https://my-oci-registry.io/my-project/subpath
...
# Required if no index is provided or found
charts:
  - redis
  - mariadb
  - ...
~~~

### LOCAL example

In case charts-syncer is not able to directly push the modified charts to the desired target, it would be possible to sync the charts
to a local folder using the LOCAL target kind and then use any other tool or process to upload these charts to the final charts repository.

~~~yaml
target:
 repo:
   kind: LOCAL
   path: your_local_path
~~~

## Requirements

In order for this tool to be able to successfully migrate a chart from a source repository to another it must fulfill the following requirements:

- The images used by the chart must be specified in the following way in the values.yaml file:

~~~yaml
image:
  registry: docker.io
  repository: bitnami/ghost
  tag: 3.22.2-debian-10-r0
~~~

The parent section name does not matter. In the previous example, instead of `image` it could be `mainImage` or whatever other name.

The important thing is that the image name is specified with `registry`, `repository` and `tag`.

The values of the parameteres `containerRegistry` and `containerRepositories` from the configuration file will be used to update the `registry` and `repository` properties in the values.yaml. The `tag` remains unchanged.

> :warning: Be aware that this tool expects the images to be already present in the target container registry.

## Changes performed in a chart

In order to migrate a chart from one repository to another and retrieve the images from a new container registry, this tool performs the following changes in the chart code:

#### Update *values.yaml* and *values-production.yaml* (if exists)

These files are updated with the new container registry where the chart should pull the images from.

#### Update dependencies files

For Helm v2, these files are *requirements.yaml* and *requirements.lock*.
For Helm v3, these files are *Chart.yaml* and *Chart.lock*

If the chart has any dependency, they should be registered in these files that will be updated to retrieve the dependencies from the target repository.

#### Update *README.md*

README files for bitnami charts include a TL;DR; section with instructions to add the helm repository to the helm CLI and a simple command to deploy the chart.

As the chart repository URL and chart repository name should have changed, the instructions in the README should be updated too.

------

Let's see the performed changes with an example. Imagine I sync the Ghost chart from the Bitnami chart repo to a local chartmuseum repo with no authentication.

I would use this config file:

~~~yaml
source:
  repo:
    kind: HELM
    url: https://charts.bitnami.com/bitnami
target:
  repoName: myrepo
  containerRegistry: "my.registry.io"
  containerRepository: "test"
  repo:
    kind: CHARTMUSEUM
    url: http://localhost:8080
~~~

After executing the tool, these are the changes performed to the following files:

#### values.yaml

~~~diff
diff --git a/values.yaml b/values.yaml
index dff53b1..a9d5884 100755
--- a/values.yaml
+++ b/values.yaml
@@ -12,8 +12,8 @@
 ## ref: https://hub.docker.com/r/bitnami/ghost/tags/
 ##
 image:
-  registry: docker.io
-  repository: bitnami/ghost
+  registry: my.registry.io
+  repository: test/ghost
   tag: 3.22.2-debian-10-r0
   ## Specify a imagePullPolicy
   ## Defaults to 'Always' if image tag is 'latest', else set to 'IfNotPresent'
@@ -40,8 +40,8 @@ image:
 ##
 volumePermissions:
   image:
-    registry: docker.io
-    repository: bitnami/minideb
+    registry: my.registry.io
+    repository: test/minideb
     tag: buster
     pullPolicy: Always
     ## Optionally specify an array of imagePullSecrets.
~~~

#### requirements.lock (only for Helm v2 charts)

~~~diff
diff --git a/requirements.lock b/requirements.lock
index ae8a2c5..ea23e53 100755
--- a/requirements.lock
+++ b/requirements.lock
@@ -1,9 +1,9 @@
 dependencies:
 - name: common
-  repository: https://charts.bitnami.com/bitnami
+  repository: http://localhost:8080
   version: 0.3.1
 - name: mariadb
-  repository: https://charts.bitnami.com/bitnami
+  repository: http://localhost:8080
   version: 7.6.1
-digest: sha256:9893236041ef5bdf2e972db020e72e8d68100eac4e280b9066d6e16c4061bcb3
-generated: "2020-07-06T18:13:45.662082005Z"
+digest: sha256:fbd22a3fc7b93ce6875a37902a3c8ccbb5dd3db2611ec9860b99e49d9f23196e
+generated: "2020-07-07T12:57:28.573258+02:00"
~~~

#### Chart.lock (only for Helm v3 charts)

~~~diff
diff --git a/Chart.lock b/Chart.lock
index ae1c198..eeed9a7 100644
--- a/Chart.lock
+++ b/Chart.lock
@@ -1,6 +1,6 @@
 dependencies:
 - name: zookeeper
-  repository: https://charts.bitnami.com/bitnami
+  repository: http://127.0.0.1:8080
   version: 5.21.9
-digest: sha256:3157eeec51b30e4011b34043df2dfac383a4bd11f76c85d07f54414a21dffc19
-generated: "2020-09-29T12:51:56.872354+02:00"
+digest: sha256:8aef6388d327cdf9b8f5714aadfe8112b2e2ff53494e86dbd42946d742d33ff0
+generated: "2020-09-30T16:15:20.548388+02:00"
~~~

#### README.md

~~~diff
diff --git a/README.md b/README.md
index 3fa7d7b..504894e 100755
--- a/README.md
+++ b/README.md
@@ -5,8 +5,8 @@
 ## TL;DR;

 ```console
-$ helm repo add bitnami https://charts.bitnami.com/bitnami
-$ helm install my-release bitnami/ghost
+$ helm repo add myrepo http://localhost:8080
+$ helm install my-release myrepo/ghost
 ```

 ## Introduction
@@ -29,7 +29,7 @@ Bitnami charts can be used with [Kubeapps](https://kubeapps.com/) for deployment
 To install the chart with the release name `my-release`:

 ```console
-$ helm install my-release bitnami/ghost
+$ helm install my-release myrepo/ghost
 ```

 The command deploys Ghost on the Kubernetes cluster in the default configuration. The [Parameters](#parameters) section lists the parameters that can be configured during installation.
@@ -168,7 +168,7 @@ Specify each parameter using the `--set key=value[,key=value]` argument to `helm
 ```console
 $ helm install my-release \
   --set ghostUsername=admin,ghostPassword=password,mariadb.mariadbRootPassword=secretpassword \
-    bitnami/ghost
+    myrepo/ghost
 ```

 The above command sets the Ghost administrator account username and password to `admin` and `password` respectively. Additionally, it sets the MariaDB `root` user password to `secretpassword`.
@@ -176,7 +176,7 @@ The above command sets the Ghost administrator account username and password to
 Alternatively, a YAML file that specifies the values for the above parameters can be provided while installing the chart. For example,

 ```console
-$ helm install my-release -f values.yaml bitnami/ghost
+$ helm install my-release -f values.yaml myrepo/ghost
 ```
~~~

> In order to obtain these diffs check the [developer docs](docs/development.md).

## How to build

> Check the [developer docs](docs/development.md).

## Deploy to Kubernetes

A simple way of having both chart repositories synced is to run the tool periodically using a Kubernetes CronJob.

In order to ease and accelerate the deployment, basic Kubernetes templates have been added to the `/deployment` folder. Follow the next steps to use them:

1. Edit `deployment/cronjob.yaml` and replace the `TAG` placeholder by a valid tag. For example, `v0.6.2`

1. Edit the frequency of execution by editing the `schedule` property. By default, it will be run each 30 minutes.

1. Edit the configuration file from `deployment/configmap.yaml` and specify your source and target chart repositories.

1. (optional) Configure credentials. If any of your source or target chart repository is using basic authentication you need to specify the credentials. See [Manage credentials](#manage-credentials) section to check current options.

1. Deploy the manifests to your Kubernetes cluster.

    ~~~bash
    $ kubectl create -f deployment/
    ~~~

### Manage credentials

The recommended way to specify credentials is using environment variables in the CronJob manifest.

Example credentials for only Helm Chart repositories
~~~yaml
  - name: charts-syncer
    image: IMAGE_NAME:TAG
    env:
      - name: SOURCE_REPO_AUTH_USERNAME
        valueFrom:
          secretKeyRef:
            name: charts-syncer-credentials
            key: source-username
      - name: SOURCE_REPO_AUTH_PASSWORD
        valueFrom:
          secretKeyRef:
            name: charts-syncer-credentials
            key: source-password
      - name: TARGET_REPO_AUTH_USERNAME
        valueFrom:
          secretKeyRef:
            name: charts-syncer-credentials
            key: target-username
      - name: TARGET_REPO_AUTH_PASSWORD
        valueFrom:
          secretKeyRef:
            name: charts-syncer-credentials
            key: target-password
~~~

Example credentials for both Helm Chart repositories and container registries
~~~yaml
  - name: charts-syncer
    image: IMAGE_NAME:TAG
    env:
      # Helm Chart repositories credentials
      - name: SOURCE_REPO_AUTH_USERNAME
        valueFrom:
          secretKeyRef:
            name: charts-syncer-credentials
            key: source-username
      - name: SOURCE_REPO_AUTH_PASSWORD
        valueFrom:
          secretKeyRef:
            name: charts-syncer-credentials
            key: source-password
      - name: TARGET_REPO_AUTH_USERNAME
        valueFrom:
          secretKeyRef:
            name: charts-syncer-credentials
            key: target-username
      - name: TARGET_REPO_AUTH_PASSWORD
        valueFrom:
          secretKeyRef:
            name: charts-syncer-credentials
            key: target-password
      - name: SOURCE_REPO_AUTH_USERNAME
          valueFrom:
            secretKeyRef:
              name: charts-syncer-credentials
              key: source-username

      # Container images registry credentials
      - name: SOURCE_CONTAINERS_AUTH_REGISTRY
        valueFrom:
          secretKeyRef:
            name: charts-syncer-credentials
            key: source-containerauth-registry
      - name: SOURCE_CONTAINERS_AUTH_USERNAME
          valueFrom:
            secretKeyRef:
              name: charts-syncer-credentials
              key: source-containerauth-username
      - name: SOURCE_CONTAINERS_AUTH_PASSWORD
          valueFrom:
            secretKeyRef:
              name: charts-syncer-credentials
              key: source-containerauth-password
      - name: TARGET_CONTAINERS_AUTH_USERNAME
          valueFrom:
            secretKeyRef:
              name: charts-syncer-credentials
              key: target-containerauth-username
      - name: TARGET_CONTAINERS_AUTH_PASSWORD
          valueFrom:
            secretKeyRef:
              name: charts-syncer-credentials
              key: target-containerauth-password
~~~
The above environment variables are retrieved from a secret called `charts-syncer-credentials` that can be created however you prefer, either manually, using sealed-secrets, vault, or any other secrets management solution for Kubernetes.
