# Deploying Charts-syncer in Kubernetes

A simple way of having two Helm Chart repositories synced is to run the tool periodically using a Kubernetes CronJob.

The `/docs/k8s` folder contains a set of Kubernetes templates that can be used to follow the guide below.

## Step 0: Retrieve repository

First step is to retrieve the repository where the k8s deployment templates are placed

```bash
$ git clone https://github.com/bitnami-labs/charts-syncer.git
$ cd charts-syncer
```

### Step 1: Configure charts-syncer

Edit the configuration file from [/docs/k8s/overlays/config.yaml]() and specify your source and target chart repositories. 
You can find a reference example [here](https://github.com/bitnami-labs/charts-syncer/blob/master/charts-syncer.yaml).

### Step 2 (optional): Update deployment options

Edit [/docs/k8s/kustomize.yaml]() and replace images.NewTag to point to the latest available release version. For example v0.14.0

You can also change the frequency of execution of the cron job by editing the schedule property in [/docs/k8s/cronjob.yaml](). By default, it will be run each 30 minutes.

### Step 3: Configure Helm Chart/Container registries credentials

If any of your Helm Chart or Container repositories require authentication
you need to specify the credentials.

The list of available credentials related keys and their meaning can be found [here](https://github.com/bitnami-labs/charts-syncer#configuration)

For k8s, this can be achieved via two different ways

#### a - Updating [/docs/k8s/overlays/secrets.env]()

```diff
 # Source repositories credentials
 # Helm Chart
-SOURCE_REPO_AUTH_USERNAME
-SOURCE_REPO_AUTH_PASSWORD
+SOURCE_REPO_AUTH_USERNAME=my_chart_repo_username
+SOURCE_REPO_AUTH_PASSWORD=my_chart_repo_password
 # Container images
-SOURCE_CONTAINERS_AUTH_REGISTRY
-SOURCE_CONTAINERS_AUTH_USERNAME
-SOURCE_CONTAINERS_AUTH_PASSWORD
+SOURCE_CONTAINERS_AUTH_REGISTRY=container.registry.io
+SOURCE_CONTAINERS_AUTH_USERNAME=my_container_registry_username
+SOURCE_CONTAINERS_AUTH_PASSWORD=my_container_registry_password
```

#### b - Providing environment variables at deployment time

You can provide the desired credentials as environment variables to the `kubectl apply -k` command

```bash
SOURCE_REPO_AUTH_USERNAME='my_chart_repo_username' \
SOURCE_REPO_AUTH_PASSWORD='my_chart_repo_password' \
SOURCE_CONTAINERS_AUTH_REGISTRY='registry.test.io' \
SOURCE_CONTAINERS_AUTH_USERNAME='my_container_registry_username' \
SOURCE_CONTAINERS_AUTH_PASSWORD='my_container_registry_password' \
TARGET_REPO_AUTH_USERNAME='my_target_chart_repo_username' \
TARGET_REPO_AUTH_PASSWORD='my_target_chart_repo_password' \
TARGET_CONTAINERS_AUTH_USERNAME='my_target_container_registry_username' \
TARGET_CONTAINERS_AUTH_PASSWORD='my_target_container_registry_password' \
kubectl apply -k ./docs/k8s
```

### 3 - Deploy the manifests to your Kubernetes cluster

Charts-syncer will be deployed by default to the `charts-syncer` k8s namespace, so the first step is to create it

```bash
$ kubectl create namespace charts-syncer
```

Once the modifications are done, you can generate the templates and deploy it in your k8s cluster by executing

```bash
$ kubectl apply -k ./docs/k8s
```

remember that you can provide credentials overrides as described in te section above i.e

```bash
$ TARGET_REPO_AUTH_PASSWORD='my-password' kubectl apply -k ./docs/k8s
```

The execution of that command will create three kubernetes resources, a cronjob, a secret and a config map

```bash
$ kubectl get secret,configmap,cronjob -n charts-syncer -l app=charts-syncer
NAME                               TYPE     DATA   AGE
secret/charts-syncer-credentials   Opaque   9      65m

NAME                             DATA   AGE
configmap/charts-syncer-config   1      65m

NAME                          SCHEDULE       SUSPEND   ACTIVE   LAST SCHEDULE   AGE
cronjob.batch/charts-syncer   */30 * * * *   False     0        23m             65m
```

### 4 - Try and debug an initial sync

The cronjob will be scheduled based on the schedule frequency, 30 minutes (by default) from now,
but it's possible to run a job based on the cronjob template by executing

```bash
$ kubectl create job test-initial-sync --from cronjob/charts-syncer
job.batch/test-initial-sync created
```

Now we can make sure that the job COMPLETES successfully and retrieve any meaningful information from the logs

```bash
$ kubectl get jobs -l app=charts-syncer
NAME                     COMPLETIONS   DURATION   AGE
test-initial-sync        1/1           105s       2m45s

$ kubectl logs -l app=charts-syncer -f
```

If you ran into any configuration issues just follow the steps 1 to 4 and rinse and repeat