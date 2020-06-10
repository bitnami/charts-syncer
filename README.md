# c3tsyncer

Sync chart packages between chart repositories

## Usage

#### Sync a specific chart

~~~bash
$ c3tsyncer syncChart --name nginx --version 1.0.0 --config ./c3tsyncer.yaml
~~~

#### Sync all versions for a specific chart

~~~bash
$ c3tsyncer syncChart --name nginx --all-versions --config ./c3tsyncer.yaml
~~~

#### Sync all charts and versions

~~~bash
$ c3tsyncer sync --config ./c3tsyncer.yaml
~~~

#### Sync all charts and versions from specific date

~~~bash
$ c3tsyncer sync --config --from-date 2020-05-15 ./c3tsyncer.yaml
~~~

 > Date should be in format YYYY-MM-DD

----

## Configuration

Below you can find an example configuration file:

~~~yaml
#
# Example config file
#
source:
  repo:
    kind: "HELM"
    url: "http://localhost:8080" # local test source repo
    # auth:
    #   username: "USERNAME"
    #   password: "PASSWORD"
target:
  containerRegistry: "k8s.container.registry.io"
  containerRepository: "repository/demo/k8s"
  repo:
    kind: "CHARTMUSEUM"
    url: "http://localhost:9090" # local test target repo
    # auth:
    #   username: "USERNAME"
    #   password: "PASSWORD"#
~~~

Credentials can be provided using config file or the following environment variables:

- `SOURCE_AUTH_USERNAME`
- `SOURCE_AUTH_PASSWORD`
- `TARGET_AUTH_USERNAME`
- `TARGET_AUTH_PASSWORD`

Current available Kinds are `HELM` and `CHARTMUSEUM`

## How to build

You need go and the Go protocol buffers pluging:

~~~bash
make gen # To generate Go code from protobuff definition
make build # To actually build the binary
~~~~

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
  - name: c3styncer
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
