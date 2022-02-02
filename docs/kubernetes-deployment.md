# Deploying charts-syncer in Kubernetes

A native way of having two Helm Chart repositories synced is to run charts-syncer periodically using a Kubernetes CronJob.

### Step 0: Retrieve git repository

The [deployment/](/deployment) directory in this repository contains a set of Kubernetes templates that must be used to complete this guide.

```bash
$ git clone https://github.com/bitnami-labs/charts-syncer.git
$ cd charts-syncer
```

### Step 1: Configure charts-syncer

Edit the configuration file from [deployment/config/config.yaml](/deployment/config/config.yaml) and specify your source and target chart repositories. 
You can find a reference example [here](https://github.com/bitnami-labs/charts-syncer/blob/master/charts-syncer.yaml).

### Step 2 (optional): Update deployment options

Edit [deployment/kustomization.yaml](/deployment/kustomization.yaml) and replace `images.NewTag` to point to the latest available release version. For example `v0.14.0`

You can also change the frequency of execution of the cron job by editing the schedule property in [deployment/cronjob.yaml](/deployment/cronjob.yaml). By default, it will be run each 30 minutes.


### Step 3 - Deploy the manifests to your Kubernetes cluster

Charts-syncer will be deployed by default to the `charts-syncer` namespace, so the first step is to create it

```bash
$ kubectl create namespace charts-syncer
```

If none of your Helm Chart or Container repositories require authentication, deploying charts syncer is as simple as executing

```bash
$ kubectl apply -k ./deployment
```

If AuthN is required, a set of credentials need to be provided via one of the following two methods

#### a - Through environment variables

You can provide the desired credentials as environment variables to the `kubectl apply -k` command

```bash
$ SOURCE_REPO_AUTH_USERNAME='my_chart_repo_username' \
SOURCE_REPO_AUTH_PASSWORD='my_chart_repo_password' \
SOURCE_CONTAINERS_AUTH_REGISTRY='registry.test.io' \
SOURCE_CONTAINERS_AUTH_USERNAME='my_container_registry_username' \
SOURCE_CONTAINERS_AUTH_PASSWORD='my_container_registry_password' \
TARGET_REPO_AUTH_USERNAME='my_target_chart_repo_username' \
TARGET_REPO_AUTH_PASSWORD='my_target_chart_repo_password' \
TARGET_CONTAINERS_AUTH_USERNAME='my_target_container_registry_username' \
TARGET_CONTAINERS_AUTH_PASSWORD='my_target_container_registry_password' \
kubectl apply -k ./deployment
```
The full list of credentials and env variables can be found [here](https://github.com/bitnami-labs/charts-syncer#configuration)

#### b - Updating secrets template

Alternatively, you can modify [deployment/config/secrets.env](/deployment/config/secrets.env) 

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

once the file has been changed just execute

```bash
$ kubectl apply -k ./deployment
```

### Step 4 - Try and debug an initial sync

After completing the previous step, a cronjob, a secret and a config map should have been created.

```bash
$ kubectl get secret,configmap,cronjob -n charts-syncer -l app=charts-syncer
NAME                               TYPE     DATA   AGE
secret/charts-syncer-credentials   Opaque   9      65m

NAME                             DATA   AGE
configmap/charts-syncer-config   1      65m

NAME                          SCHEDULE       SUSPEND   ACTIVE   LAST SCHEDULE   AGE
cronjob.batch/charts-syncer   */30 * * * *   False     0        23m             65m
```

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