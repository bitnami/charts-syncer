# charts-syncer

Sync chart packages and associated container images between chart repositories

# Table of Contents

- [Usage](#usage)
    + [Sync all charts](#sync-all-helm-charts)
    + [Sync all charts from specific date](#sync-all-charts-from-specific-date)
- [Advanced Usage](#advanced-usage)
    + [Skip syncing artifacts](#skip-syncing-artifacts)
    + [Sync only specific container platforms](#sync-only-specific-container-platforms)
    + [Sync Helm Charts and Container Images to different registries](#sync-helm-charts-and-container-images-to-different-registries)
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

## Usage

### Sync all Helm Charts

```console
$ charts-syncer sync
```

### Sync Helm Charts from a specific date

```console
$ charts-syncer sync --from-date 2020-05-15
```

### Sync latest version of each Helm Chart

```console
$ charts-syncer sync --latest-version-only
```

## Advanced Usage

### Sync only specific container platforms

By default, all container platforms are sync-ed to the destination registry, but this behavior can by tweaked by defining a list of platforms to sync:

```yaml
#
# Example config file
#
source:
  repo:
    kind: OCI
    url: http://localhost:8080
target:
  # Container images registry authn
  repo:
    kind: OCI
    url: http://localhost:9090/charts

containerPlatforms:
  - linux/amd64

charts:
  - redis
  - mariadb
```

### Skip syncing artifacts

If your chart and docker images include artifacts such as signatures or metadata, they will be synced to the destination repository. If you want to disable this behavior, you can opt out by setting `skipArtifacts` to true:

```yaml
source:
  repo:
    kind: OCI
    url: http://localhost:8080
target:
  repo:
    kind: OCI
    url: http://localhost:9090/charts
charts:
  - redis

skipArtifacts: true
```

This is especially useful when you filter the container platforms to sync, which would invalidate the signatures. Using `skipArtifacts: true` will prevent syncing the now invalid signatures:

```yaml
source:
  repo:
    kind: OCI
    url: http://localhost:8080
target:
  repo:
    kind: OCI
    url: http://localhost:9090/charts
charts:
  - redis
skipArtifacts: true
containerPlatforms:
  - linux/amd64
```


### Sync Helm Charts and Container Images to different registries

By default, charts-syncer syncs Helm Charts packages and their container images to the same registry specified in the `target.repo.url` property. If you require to configure a different destination registry for the images, this can be configured in the `target.containers.url` property:

```yaml
#
# Example config file
#
source:
  repo:
    kind: OCI
    url: http://localhost:8080
target:
  # Container images registry authn
  containers:
    url: http://localhost:9090/containers
    auth:
      username: "USERNAME"
      password: "PASSWORD"
  repo:
    kind: OCI
    url: http://localhost:9090/charts
    # Helm repository authentication
    # auth:
    #   username: "USERNAME"
    #   password: "PASSWORD"
charts:
  - redis
  - mariadb
```

### Sync Helm Charts and associated container images between disconnected environments

There are scenarios where the source and target Helm Charts repositories are not reachable at the same time from the same location.

For those cases, charts-syncer supports a two steps relocation for offline Chart and container images transport, check the [air gap docs](docs/airgap.md).

----

## Configuration

Below you can find an example configuration file. To know all the available configuration keys see the [charts-syncer](./charts-syncer.yaml) file as it includes explanatory comments for each configuration key.

```yaml
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
  repo:
    kind: OCI
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
```

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

Current available Kinds are `LOCAL`, `HELM`, `CHARTMUSEUM`, `HARBOR` and `OCI` for the Source Repo and `OCI` and `LOCAL` for the Target Repo.

> The list of charts in the config file is optional except for OCI repositories used as source.
> The rest of chart repositories kinds already support autodiscovery.

### Harbor example

In the case of HARBOR kind repos, be aware that chart repository URLs are:

https://$HARBOR_DOMAIN/chartrepo/$HARBOR_PROJECT

So if HARBOR_DOMAIN=my.harbor.com and HARBOR_PROJECT=my-project, you would need to specify this repo in the config file like:

```yaml
source:
 repo:
   kind: HARBOR
   url: https://my.harbor.com/chartrepo/my-project
```

### OCI example

Since Harbor 2.0.0, there are two ways of storing charts. The legacy one uses chartmuseum under the hood and it corresponds to the HARBOR kind of this project.
The new one however uses OCI to store helm charts as OCI artifacts. In case you are using Harbor with OCI backend you can use the following example:

```yaml
target:
 repo:
   kind: OCI
   url: https://my.harbor.com/my-project/subpath
```

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

```yaml
source:
 repo:
   kind: OCI
   url: https://my-oci-registry.io/my-project/subpath
   # disableChartsIndex: true
   # Charts index location override, charts-index:latest by default
   chartsIndex: my-oci-registry.io/my-project/my-custom-index:prod
```

Finally, if no charts index is found, charts-syncer will require the list of charts in the config file:

```yaml
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
```

#### Amazon Elastic Container Registry (ECR)
Amazon Elastic Container Registry (ECR) is an OCI registry, but it has two peculiarities that should be taken into account when interacting with charts-syncer.

The first peculiarity relates to authentication, usually you would have an IAM user with access_key_id and secret_access_key credentials. These **are not** the credentials that should be entered in the config file.
To obtain the proper credentials, you need to obtain a temporary password to operate the ECR registry. By using the following command, you can get the password:

```console
$ aws ecr get-login-password --region REGION
```

In the previous command remember to update the REGION placeholder by a proper value.
The username will always be `AWS`.

The second peculiarity only affects when using ECR as a target registry since you need to create the repositories in advance.
If the repositories don't exist when executing charts-syncer you will get an error like the following:

```
failed to do request: Post "https://AWS_ACCOUNT.dkr.ecr.AWS_REGION.amazonaws.com/v2/charts-syncer-test/charts/common/blobs/uploads/": EOF
failed to do request: Post "https://AWS_ACCOUNT.dkr.ecr.AWS_REGION.amazonaws.com/v2/charts-syncer-test/charts/apache/blobs/uploads/": EOF
failed to do request: Post "https://AWS_ACCOUNT.dkr.ecr.AWS_REGION.amazonaws.com/v2/charts-syncer-test/charts/mariadb/blobs/uploads/": EOF
failed to do request: Post "https://AWS_ACCOUNT.dkr.ecr.AWS_REGION.amazonaws.com/v2/charts-syncer-test/charts/mysql/blobs/uploads/": EOF
failed to do request: Post "https://AWS_ACCOUNT.dkr.ecr.AWS_REGION.amazonaws.com/v2/charts-syncer-test/charts/redis/blobs/uploads/": EOF
```

In that case make sure all the missing repositories are created before executing charts-syner again. Please refer to [AWS documentation](https://docs.aws.amazon.com/AmazonECR/latest/userguide/repository-create.html) to see how to do it.

### LOCAL example

In case charts-syncer is not able to directly push the modified charts to the desired target, it would be possible to sync the charts
to a local folder using the LOCAL target kind and then use any other tool or process to upload these charts to the final charts repository.

```yaml
target:
 repo:
   kind: LOCAL
   path: your_local_path
```

## Requirements

In order for this tool to be able to successfully migrate a chart from a source repository to another it must fulfill the following requirements:

- The images used by the chart must be specified in the following way in the values.yaml file:

```yaml
image:
  registry: docker.io
  repository: bitnami/ghost
  tag: 3.22.2-debian-10-r0
```

The parent section name does not matter. In the previous example, instead of `image` it could be `mainImage` or whatever other name.

The important thing is that the image name is specified with `registry`, `repository` and `tag`.

The values of the parameters `containerRegistry` and `containerRepositories` from the configuration file will be used to update the `registry` and `repository` properties in the values.yaml. If these parameters are unset, the associated properties won't be modified. The `tag` property remains unchanged.

> :warning: Be aware that this tool expects the images to be already present in the target container registry.

## Changes performed in a chart

In order to migrate a chart from one repository to another and retrieve the images from a new container registry, this tool performs the following changes in the chart code:

#### Update *values.yaml*

This file is updated with the new container registry where the chart should pull the images from.

### Update *Chart.yaml*

This file will get its `images` annotation rewritten to point to the new relocated container images.

### Update *Images.lock*

If present, the [Images.lock](https://github.com/vmware-labs/distribution-tooling-for-helm/tree/main?tab=readme-ov-file#creating-an-images-lock)
 will be relocated to point to the new container images.

 If the file does not exist, it will be created.

------

Let's see the performed changes with an example. Imagine I sync the Ghost chart from the Bitnami chart repo to a local chartmuseum repo with no authentication.

I would use this config file:

```yaml
source:
  repo:
    kind: HELM
    url: https://charts.bitnami.com/bitnami
target:
  repo:
    kind: OCI
    url: http://localhost:8080
```

After executing the tool, these are the changes performed to the following files:

#### values.yaml

```diff
diff --git a/values.yaml b/values.yaml
index dff53b1..a9d5884 100755
--- a/values.yaml
+++ b/values.yaml
@@ -68,8 +68,8 @@
 ## @param image.debug Enable image debug mode
 ##
 image:
-  registry: docker.io
-  repository: bitnami/ghost
+  registry: localhost:80
+  repository: library/bitnami/ghost
   tag: 5.79.4-debian-12-r2
   digest: ""
   ## Specify a imagePullPolicy
@@ -608,8 +608,8 @@
   ## @param volumePermissions.image.pullSecrets OS Shell + Utility image pull secrets
   ##
   image:
-    registry: docker.io
-    repository: bitnami/os-shell
+    registry: localhost:80
+    repository: library/bitnami/os-shell
     tag: 12-debian-12-r15
     digest: ""
     pullPolicy: IfNotPresent
```


#### Chart.yaml

```diff
diff --git a/Chart.yaml b/Chart.yaml
--- a/Chart.yaml
+++ b/Chart.yaml
@@ -2,9 +2,9 @@
   category: CMS
   images: |
     - name: ghost
-      image: docker.io/bitnami/ghost:5.79.4-debian-12-r2
+      image: localhost:80/library/bitnami/ghost:5.79.4-debian-12-r2
     - name: os-shell
-      image: docker.io/bitnami/os-shell:12-debian-12-r15
+      image: localhost:80/library/bitnami/os-shell:12-debian-12-r15
   licenses: Apache-2.0
 apiVersion: v2
 appVersion: 5.79.4
```

#### Images.lock

```diff
--- /dev/null	2024-02-23 14:30:30
+++ b/Images.lock	2024-02-22 10:51:59
@@ -0,0 +1,50 @@
+apiVersion: v0
+kind: ImagesLock
+metadata:
+  generatedAt: "2024-02-22T09:46:03.681760496Z"
+  generatedBy: Distribution Tooling for Helm
+chart:
+  name: ghost
+  version: 19.10.2
+  appVersion: 5.79.4
+images:
+- name: ghost
+  image: localhost:80/library/bitnami/ghost:5.79.4-debian-12-r2
+  chart: ghost
+  digests:
+  - digest: sha256:950c0bcbdcd9e97fb6db96c70cde4408ab30e658e5568445f4a1a9734cc9cc68
+    arch: linux/amd64
+  - digest: sha256:c7ba15d98097bc06baf80091fba8f0b0c2a05fd4d94f3bec4ea4c92c3202e3b6
+    arch: linux/arm64
+- name: os-shell
+  image: localhost:80/library/bitnami/os-shell:12-debian-12-r15
+  chart: ghost
+  digests:
+  - digest: sha256:fbb2bf7afc15ff68e89b36c24cf3210a47729246f1943db056ae3c9a0c2f278d
+    arch: linux/amd64
+  - digest: sha256:051cc71e48d8d901f2958e3f323977964c2373a153a9e2b6183c3dbd2cd2075c
+    arch: linux/arm64
+- name: mysql
+  image: localhost:80/library/bitnami/mysql:8.0.36-debian-12-r7
+  chart: mysql
+  digests:
+  - digest: sha256:af4f8a296ed5081a5c91d262f06c897ac956714009a71192ee36b22742f23b9d
+    arch: linux/amd64
+  - digest: sha256:1b15bdfd66ad9acc14a6a81d570c39eb607e53a73242f13e91d63c64496b2b2f
+    arch: linux/arm64
+- name: mysqld-exporter
+  image: localhost:80/library/bitnami/mysqld-exporter:0.15.1-debian-12-r7
+  chart: mysql
+  digests:
+  - digest: sha256:cc417c3577774bd439bc8cef6aeced1ef1019192964b763116072fafad16b73a
+    arch: linux/amd64
+  - digest: sha256:81ead00a80c63f562ff028c3fc2772354f8354085fe0ffb5ba29b04f6d4a2f4a
+    arch: linux/arm64
+- name: os-shell
+  image: localhost:80/library/bitnami/os-shell:12-debian-12-r15
+  chart: mysql
+  digests:
+  - digest: sha256:fbb2bf7afc15ff68e89b36c24cf3210a47729246f1943db056ae3c9a0c2f278d
+    arch: linux/amd64
+  - digest: sha256:051cc71e48d8d901f2958e3f323977964c2373a153a9e2b6183c3dbd2cd2075c
+    arch: linux/arm64
```


## Deploy to Kubernetes

Visit [this guide](docs/kubernetes-deployment.md) to deploy a Kubernetes CronJob that will keep two Helm Chart repositories synced.

## How to build

> Check the [developer docs](docs/development.md).

## License

Copyright &copy; 2024 Broadcom. The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and limitations under the License.
