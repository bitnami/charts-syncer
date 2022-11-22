# Simple Chart

This example shows how relok8s can be used to move a MariaDB Helm Chart along with their container images to a target registry.

## Inputs

### Helm Chart

In this example, we are using the [Bitnami MariaDB](https://github.com/bitnami/charts/tree/master/bitnami/mariadb) Helm Chart.
This chart references three images and does not contain any subcharts*.

> NOTE: This chart actually does contain a subchart, `bitnami/common`, but that chart does not itself contain any image references, so it effectively makes no difference to the example.

### Image patterns hints file

`relok8s` requires an image hints file to know how the Helm C
hart encodes the image references. These "hints" entries are used during both finding the Container images to relocate as well as knowing how to re-write back the new location of these.
Specifically, the main MariaDB image is referenced in the chart like this:

```yaml
image:
  registry: docker.io
  repository: bitnami/mariadb
  tag: 10.5.12-debian-10-r0
```

So, our hints file includes this line:

```yaml
- "{{ .image.registry }}/{{ .image.repository }}:{{ .image.tag }}"
```

This is repeated for the other two images.

## Running `relok8s`

To relocate the chart we will run this command:

```bash
relok8s chart move mariadb-chart --image-patterns image-hints.yaml --registry projects.registry.vmware.com --repo-prefix relocated/example1
```

Breaking down this command:

```bash
relok8s chart move ...
```

indicates that we want to relocate a Helm chart

```bash
... mariadb-chart ...
```

this part is the path to the chart

```bash
... --image-patterns image-hints.yaml ...
```

this part is the path to the image patterns hints file that we created for this chart

```bash
... --registry harbor-repo.vmware.com ...
```

this flag says that we want to change the image registry

```bash
... --repo-prefix relocated/example1
```

... and prepend the image path with `relocated/example1`.

When the command runs, it will:

1. Resolve the image URLs by using the image hints file
1. Pull the resolved images
1. Check the remote registry
1. Calculate push and rewrite operations
1. Prompt for confirmation
1. Push the rewritten images
1. Create a the modified Helm Chart pointing to the new image references
1. Package the resulting Helm Chart

```bash
$ relok8s chart move mariadb-chart --image-patterns image-hints.yaml --registry projects.registry.vmware.com --repo-prefix relocated/example1

Images to be pushed:
  projects.registry.vmware.com/relocated/example1/mariadb:10.5.12-debian-10-r0 (sha256:8e2c533c786b7c921c75a4f26f1779362bdbb3a38ffd0e7771a93078eb641692)
  projects.registry.vmware.com/relocated/example1/mysqld-exporter:0.13.0-debian-10-r56 (sha256:a591b9f9c08b328efd3bd5815c275c348e1a631688aec984e32185946d84126f)
  projects.registry.vmware.com/relocated/example1/bitnami-shell:10-debian-10-r153 (sha256:53891aea23bdc9fe2d1aa248ea260c867b51eb07e26debc01c00fb7169208950)

Changes written to mariadb/values.yaml:
  .image.registry: projects.registry.vmware.com
  .image.repository: relocated/example1/mariadb
  .metrics.image.registry: projects.registry.vmware.com
  .metrics.image.repository: relocated/example1/mysqld-exporter
  .volumePermissions.image.registry: projects.registry.vmware.com
  .volumePermissions.image.repository: relocated/example1/bitnami-shell
Would you like to proceed? (y/N)
y
Pushing projects.registry.vmware.com/relocated/example1/mariadb:10.5.12-debian-10-r0...
Done
Pushing projects.registry.vmware.com/relocated/example1/mysqld-exporter:0.13.0-debian-10-r56...
Done
Pushing projects.registry.vmware.com/relocated/example1/bitnami-shell:10-debian-10-r153...
Done
Writing chart files... Done

```

## Outputs

### Modified Helm Chart

The output of the command is a rewritten Helm chart, with the values of the image references put into the chart's values.yaml file:

```bash
$ ls *relocated.tgz
mariadb-9.4.2.relocated.tgz
```
```diff
$ diff -u mariadb-chart/values.yaml <(tar zxfO *.relocated.tgz mariadb/values.yaml)
diff -u mariadb-chart/values.yaml <(tar zxfO *.relocated.tgz mariadb/values.yaml)
--- mariadb-chart/values.yaml   2021-08-11 14:36:20.615731277 -0700
+++ /dev/fd/63  2021-08-11 15:12:04.320853143 -0700
@@ -68,8 +68,8 @@
 ## @param image.debug Specify if debug logs should be enabled
 ##
 image:
-  registry: docker.io
-  repository: bitnami/mariadb
+  registry: projects.registry.vmware.com
+  repository: relocated/example1/mariadb
   tag: 10.5.12-debian-10-r0
   ## Specify a imagePullPolicy
   ## Defaults to 'Always' if image tag is 'latest', else set to 'IfNotPresent'
@@ -795,8 +795,8 @@
   ## @param volumePermissions.image.pullSecrets Specify docker-registry secret names as an array
   ##
   image:
-    registry: docker.io
-    repository: bitnami/bitnami-shell
+    registry: projects.registry.vmware.com
+    repository: relocated/example1/bitnami-shell
     tag: 10-debian-10-r153
     pullPolicy: Always
     ## Optionally specify an array of imagePullSecrets (secrets must be manually created in the namespace)
@@ -828,8 +828,8 @@
   ## @param metrics.image.pullSecrets Specify docker-registry secret names as an array
   ##
   image:
-    registry: docker.io
-    repository: bitnami/mysqld-exporter
+    registry: projects.registry.vmware.com
+    repository: relocated/example1/mysqld-exporter
     tag: 0.13.0-debian-10-r56
     pullPolicy: IfNotPresent
     ## Optionally specify an array of imagePullSecrets (secrets must be manually created in the namespace)
```
