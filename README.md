[![CircleCI](https://circleci.com/gh/bitnami-labs/charts-syncer.svg?style=svg&circle-token=91105ed254723ef1e3af739f6d31dc845136828c)](https://circleci.com/gh/bitnami-labs/charts-syncer/tree/master)


# charts-syncer

Sync chart packages between chart repositories

## Usage

#### Sync a specific chart

~~~bash
$ charts-syncer syncChart --name nginx --version 1.0.0 --config ./charts-syncer.yaml
~~~

#### Sync all versions for a specific chart

~~~bash
$ charts-syncer syncChart --name nginx --all-versions --config ./charts-syncer.yaml
~~~

#### Sync all charts and versions

~~~bash
$ charts-syncer sync --config ./charts-syncer.yaml
~~~

#### Sync all charts and versions from specific date

~~~bash
$ charts-syncer sync --from-date 2020-05-15 --config ./charts-syncer.yaml
~~~

 > Date should be in format YYYY-MM-DD

----

## Configuration

Below you can find an example configuration file. To all the available entries see the [example-config.yaml](./charts-syncer.yaml) as it includes explanatory comments for each configuration key.

~~~yaml
#
# Example config file
#
source:
  repo:
    kind: HELM
    url: http://localhost:8080 # local test source repo
    # auth:
    #   username: "USERNAME"
    #   password: "PASSWORD"
target:
  repoName: myrepo
  containerRegistry: k8s.container.registry.io
  containerRepository: repository/demo/k8s
  repo:
    kind: CHARTMUSEUM
    url: http://localhost:9090 # local test target repo
    # auth:
    #   username: "USERNAME"
    #   password: "PASSWORD"
~~~

> Note the `repo.url` property you need to specify is the same one you would use to add the repo to helm with the `helm repo add command`.
>
> Example: `helm repo add bitnami https://charts.bitnami.com/bitnami`.

Credentials can be provided using config file or the following environment variables:

- `SOURCE_AUTH_USERNAME`
- `SOURCE_AUTH_PASSWORD`
- `TARGET_AUTH_USERNAME`
- `TARGET_AUTH_PASSWORD`

Current available Kinds are `HELM`, `CHARTMUSEUM` and `HARBOR`. Below you can find the compatibility matrix betweeen source and targets repositories.

| Source Repo | Target Repo | Supported          |
|-------------|-------------|--------------------|
| HELM        | HELM        | :white_check_mark: |
| HELM        | CHARTMUSEUM | :white_check_mark: |
| HELM        | HARBOR      | :white_check_mark: |
| CHARTMUSEUM | HELM        | :x:                |
| CHARTMUSEUM | CHARTMUSEUM | :white_check_mark: |
| CHARTMUSEUM | HARBOR      | :white_check_mark: |
| HARBOR      | HELM        | :x:                |
| HARBOR      | CHARTMUSEUM | :white_check_mark: |
| HARBOR      | HARBOR      | :white_check_mark: |


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

## Requirements

In order for this tool to be able to successfully migrate a chart from a source repository to another it must fulfill the following requirements:

- If the chart has dependencies they are specified in a *requirements.yaml* file. Currently, if the dependencies are specified in the Chart.yaml file the tool won't be able to manage them.
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

> :warning: Be aware that this tool expect the images to be already present in the target container registry.

## Changes performed in a chart

In order to migrate a chart from one repository to another and retrieve the images from a new container registry, this tools performs the following changes in the chart code:

#### Update *values.yaml* and *values-production.yaml* (if exists)

These files are updated with the new container registry where the chart should pull the images from.

#### Update *requirements.yaml* and *requirements.lock*

If the chart has any dependency, they should be registered in this file. The *requirements.yaml* and requirements.lock file will be updated to retrieve the dependencies from the target repository.

#### Update *README.md*

README files for bitnami charts includes a TL;DR; section with instructions to add the helm repository to the helm cli and a simple command to deploy the chart.

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

#### requirements.yaml

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

1. Build a docker image containing the tool and push it to a docker registry so later on it can be consumed from Kubernetes.

    ~~~bash
    $ docker build -t IMAGE_NAME:TAG .
    $ docker push IMAGE_NAME:TAG
    ~~~

1. Edit `deployment/cronjob.yaml` and replace the `IMAGE_NAME:TAG` placeholder by the real image name and tag.

1. Edit the frequenty of execution by editing the `schedule` property. By default it will be run each 30 minutes.

1. Edit the configuration file from `deployment/configmap.yaml` and specify your source and target chart repositories.

1. (optional) Configure credentials. If any of your source or target chart repository is using basic authentication you need to specify the credentials. See [Manage credentials](#manage-credentials) section to check current options.

1. Deploy the manifests to your Kubernetes cluster.

    ~~~bash
    $ kubectl create -f deployment/
    ~~~

### Manage credentials

The recommended way to specify credentials is using environment variables in the CronJob manifest.

~~~yaml
  - name: charts-syncer
    image: IMAGE_NAME:TAG
    env:
      - name: SOURCE_REPO_USERNAME
        valueFrom:
          secretKeyRef:
            name: chart-syncer-credentials
            key: source-username
      - name: SOURCE_REPO_PASSWORD
        valueFrom:
          secretKeyRef:
            name: chart-syncer-credentials
            key: source-password
      - name: TARGET_REPO_USERNAME
        valueFrom:
          secretKeyRef:
            name: chart-syncer-credentials
            key: target-username
      - name: TARGET_REPO_PASSWORD
        valueFrom:
          secretKeyRef:
            name: chart-syncer-credentials
            key: target-password
~~~

The above environment variables are retrieved from a secret called `chart-syncer-credentials` that can be created however you prefer, either manually, using sealed-secrets, vault, or any other secrets management solution for Kubernetes.

