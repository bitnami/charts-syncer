#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# We will use the Ghost chart as testing asset.
# ghost-10.0.10 chart images (including common and mariadb dependencies)
ghostChartImages=(
    "ghost:3.20.1-debian-10-r0"
    "minideb:buster"
    "mariadb:10.3.23-debian-10-r25"
)

# Pull images from bitnami, tag it and load it to kind cluster
for image in "${ghostChartImages[@]}"
do
  echo "----- ${image} ----"
  docker pull bitnami/${image}
  docker tag bitnami/${image} customer.io/library/${image}
  kind load docker-image customer.io/library/${image}
done