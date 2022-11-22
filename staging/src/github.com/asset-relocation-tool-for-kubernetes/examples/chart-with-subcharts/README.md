# Chart with Subcharts

This example shows how relok8s can be used to relocate a Helm chart that contains subcharts.
This will not go into as much detail as the [Simple Chart](../simple-chart) example, but will highlight the differences.

## Inputs

### Helm Chart

In this example, we are using the [Bitnami WordPress](https://github.com/bitnami/charts/tree/master/bitnami/wordpress) chart.
This chart depends on the [Bitnami MariaDB](https://github.com/bitnami/charts/tree/master/bitnami/mariadb) and [Bitnami Memcached](https://github.com/bitnami/charts/tree/master/bitnami/memcached) charts. Each chart references three images.

### Image pattern hints file

`relok8s` only requires a single image hints file to find all of the images in the chart and subcharts.
Images in the subcharts are pre-pended with the subchart name.

If you inspect the [hints file](./image-hints.yaml), you will see that referencing images in a subchart is as simple as pre-pending the subchart name to the path, for example, the following line will in fact inspect `$parent-chart/charts/mariadb/values.yaml`  

```yaml
- {{ .mariadb.image.registry }}/{{ .mariadb.image.repository }}:{{ .mariadb.image.tag }}
...
```
## Running `relok8s`

To relocate the chart we will run this command:

```bash
$ relok8s chart move wordpress-chart --image-patterns image-hints.yaml --registry projects.registry.vmware.com --repo-prefix relocated/example2

Images to be pushed:
  projects.registry.vmware.com/relocated/example2/wordpress:5.7.2-debian-10-r45 (sha256:187d539c69e4da11706d63fead255a870cae79a34719b67f8d1cbfd8f7653ff8)
  projects.registry.vmware.com/relocated/example2/apache-exporter:0.9.0-debian-10-r33 (sha256:c64fa482ae9cbadbb53487d1155a0a135141116bbfecd8fded0cd365f9646407)
  projects.registry.vmware.com/relocated/example2/bitnami-shell:10-debian-10-r134 (sha256:1d385c55c7d8efddc1ac7c9a1d847e3b040803b8bcbf58fba41715e77706add7)
  projects.registry.vmware.com/relocated/example2/mariadb:10.5.11-debian-10-r0 (sha256:160902dddb9c7d9640dcfc33ae0dbbed9346f786eeb653fa1b427c76f4673126)
  projects.registry.vmware.com/relocated/example2/mysqld-exporter:0.13.0-debian-10-r19 (sha256:ad0993ebdf34a6b6ee0ec469384a3c92ed020ff7b23277c90d150ddd4ed01020)
  projects.registry.vmware.com/relocated/example2/bitnami-shell:10-debian-10-r115 (sha256:400d6b412a753845c65c656b311a1d032b605cf1c63e14c25929c1d9e9c423c8)
  projects.registry.vmware.com/relocated/example2/memcached:1.6.9-debian-10-r194 (sha256:3dcf3a49f162f55ae9f7407d022ae53021cccf49f8d43276080708bd56857e78)
  projects.registry.vmware.com/relocated/example2/memcached-exporter:0.9.0-debian-10-r85 (sha256:979154c8afa2027194fe2721351b112d4daacf96ccf6ff853525e4794d1bbe49)
  projects.registry.vmware.com/relocated/example2/bitnami-shell:10-debian-10-r120 (sha256:9eeeefd2e9abeed0ee111a43e4c5c19b2983b74a2a38adb641649a77929ac59b)
                                                        
Changes written to wordpress/values.yaml:               
  .image.registry: projects.registry.vmware.com                                                                  
  .image.repository: relocated/example2/wordpress
  .metrics.image.registry: projects.registry.vmware.com
  .metrics.image.repository: relocated/example2/apache-exporter
  .volumePermissions.image.registry: projects.registry.vmware.com
  .volumePermissions.image.repository: relocated/example2/bitnami-shell

Changes written to wordpress/charts/mariadb/values.yaml: 
  .mariadb.image.registry: projects.registry.vmware.com
  .mariadb.image.repository: relocated/example2/mariadb
  .mariadb.metrics.image.registry: projects.registry.vmware.com
  .mariadb.metrics.image.repository: relocated/example2/mysqld-exporter
  .mariadb.volumePermissions.image.registry: projects.registry.vmware.com
  .mariadb.volumePermissions.image.repository: relocated/example2/bitnami-shell

Changes written to wordpress/charts/memcached/values.yaml:
  .memcached.image.registry: projects.registry.vmware.com
  .memcached.image.repository: relocated/example2/memcached
  .memcached.metrics.image.registry: projects.registry.vmware.com
  .memcached.metrics.image.repository: relocated/example2/memcached-exporter
  .memcached.volumePermissions.image.registry: projects.registry.vmware.com
  .memcached.volumePermissions.image.repository: relocated/example2/bitnami-shell
Would you like to proceed? (y/N)
y
Pushing projects.registry.vmware.com/relocated/example2/wordpress:5.7.2-debian-10-r45...
Done
Pushing projects.registry.vmware.com/relocated/example2/apache-exporter:0.9.0-debian-10-r33...
Done
Pushing projects.registry.vmware.com/relocated/example2/bitnami-shell:10-debian-10-r134...
Done
Pushing projects.registry.vmware.com/relocated/example2/mariadb:10.5.11-debian-10-r0...
Done
Pushing projects.registry.vmware.com/relocated/example2/mysqld-exporter:0.13.0-debian-10-r19...
Done
Pushing projects.registry.vmware.com/relocated/example2/bitnami-shell:10-debian-10-r115...
Done
Pushing projects.registry.vmware.com/relocated/example2/memcached:1.6.9-debian-10-r194...
Done
Pushing projects.registry.vmware.com/relocated/example2/memcached-exporter:0.9.0-debian-10-r85...
Done                                                    
Pushing projects.registry.vmware.com/relocated/example2/bitnami-shell:10-debian-10-r120...
Done                                                    
Writing chart files... Done                             
```
## Outputs

### Modified Helm Chart

The output of the command is a rewritten Helm chart, with the values of the image references put into the chart's and subchart's values.yaml files:

```bash
$ ls *relocated.tgz
wordpress-11.1.5.relocated.tgz
```
```diff
RELOCATED_DIR=/tmp/wordpress-relocated && \
rm -r $RELOCATED_DIR && mkdir $RELOCATED_DIR && tar zxf *.relocated.tgz -C $RELOCATED_DIR && \
diff -ur wordpress-chart $RELOCATED_DIR/wordpress
diff -ur wordpress-chart/charts/mariadb/values.yaml /tmp/wordpress-relocated/wordpress/charts/mariadb/values.yaml 
--- wordpress-chart/charts/mariadb/values.yaml  2021-08-11 15:39:51.471458979 -0700
+++ /tmp/wordpress-relocated/wordpress/charts/mariadb/values.yaml       2021-08-11 16:02:59.000000000 -0700
@@ -53,8 +53,8 @@            
 ## @param image.debug Specify if debug logs should be enabled
 ##                
 image:
-  registry: docker.io
-  repository: bitnami/mariadb
+  registry: projects.registry.vmware.com
+  repository: relocated/example2/mariadb
   tag: 10.5.11-debian-10-r0               
   ## Specify a imagePullPolicy                  
   ## Defaults to 'Always' if image tag is 'latest', else set to 'IfNotPresent'
@@ -780,8 +780,8 @@              
   ## @param volumePermissions.image.pullSecrets Specify docker-registry secret names as an array
   ##                                                                                                            
   image:                                                                                                        
-    registry: docker.io                                                                                         
-    repository: bitnami/bitnami-shell
+    registry: projects.registry.vmware.com   
+    repository: relocated/example2/bitnami-shell
     tag: 10-debian-10-r115
     pullPolicy: Always
     ## Optionally specify an array of imagePullSecrets (secrets must be manually created in the namespace)
@@ -813,8 +813,8 @@                      
   ## @param metrics.image.pullSecrets Specify docker-registry secret names as an array
   ##                      
   image:                      
-    registry: docker.io                                                                                         
-    repository: bitnami/mysqld-exporter
+    registry: projects.registry.vmware.com                                                                      
+    repository: relocated/example2/mysqld-exporter
     tag: 0.13.0-debian-10-r19
     pullPolicy: IfNotPresent
     ## Optionally specify an array of imagePullSecrets (secrets must be manually created in the namespace)
diff -ur wordpress-chart/charts/memcached/values.yaml /tmp/wordpress-relocated/wordpress/charts/memcached/values.yaml
--- wordpress-chart/charts/memcached/values.yaml        2021-08-11 15:39:51.475458907 -0700
+++ /tmp/wordpress-relocated/wordpress/charts/memcached/values.yaml     2021-08-11 16:02:59.000000000 -0700
@@ -12,8 +12,8 @@      
 ## ref: https://hub.docker.com/r/bitnami/memcached/tags/
 ##                
 image:                                                                                                          
-  registry: docker.io
-  repository: bitnami/memcached
+  registry: projects.registry.vmware.com
+  repository: relocated/example2/memcached
   tag: 1.6.9-debian-10-r194               
   ## Specify a imagePullPolicy                    
   ## Defaults to 'Always' if image tag is 'latest', else set to 'IfNotPresent'
@@ -298,8 +298,8 @@          
   ## ref: https://hub.docker.com/r/bitnami/memcached-exporter/tags/
   ##                                                                                                            
   image:
-    registry: docker.io
-    repository: bitnami/memcached-exporter
+    registry: projects.registry.vmware.com
+    repository: relocated/example2/memcached-exporter
     tag: 0.9.0-debian-10-r85
     pullPolicy: IfNotPresent
     ## Optionally specify an array of imagePullSecrets. 
@@ -397,8 +397,8 @@
 ##
 volumePermissions:
   image:
-    registry: docker.io
-    repository: bitnami/bitnami-shell
+    registry: projects.registry.vmware.com
+    repository: relocated/example2/bitnami-shell
     tag: 10-debian-10-r120
     ## Specify a imagePullPolicy
     ## Defaults to 'Always' if image tag is 'latest', else set to 'IfNotPresent'
diff -ur wordpress-chart/values.yaml /tmp/wordpress-relocated/wordpress/values.yaml
--- wordpress-chart/values.yaml 2021-08-11 15:39:51.467459052 -0700
+++ /tmp/wordpress-relocated/wordpress/values.yaml      2021-08-11 16:02:59.000000000 -0700
@@ -52,8 +52,8 @@
 ## @param image.debug Enable image debug mode
 ##
 image:
-  registry: docker.io
-  repository: bitnami/wordpress
+  registry: projects.registry.vmware.com
+  repository: relocated/example2/wordpress
   tag: 5.7.2-debian-10-r45
   ## Specify a imagePullPolicy
   ## Defaults to 'Always' if image tag is 'latest', else set to 'IfNotPresent'
@@ -626,8 +626,8 @@
   ## @param volumePermissions.image.pullSecrets Bitnami Shell image pull secrets
   ##
   image:
-    registry: docker.io
-    repository: bitnami/bitnami-shell
+    registry: projects.registry.vmware.com
+    repository: relocated/example2/bitnami-shell
     tag: 10-debian-10-r134
     pullPolicy: Always
     ## Optionally specify an array of imagePullSecrets. 
@@ -700,8 +700,8 @@
   ## @param metrics.image.pullSecrets Apache Exporter image pull secrets
   ##
   image:
-    registry: docker.io
-    repository: bitnami/apache-exporter
+    registry: projects.registry.vmware.com
+    repository: relocated/example2/apache-exporter
     tag: 0.9.0-debian-10-r33
     pullPolicy: IfNotPresent
     ## Optionally specify an array of imagePullSecrets. 
```
