#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# We will use the Kafka chart as testing asset.
# kafka-11.2.0 chart images (including zookeeper dependencies)
kafkaChartImages=(
    "kafka:2.5.0-debian-10-r66"
    "kubectl:1.17.4-debian-10-r91"
    "minideb:buster"
    "kafka-exporter:1.2.0-debian-10-r140"
    "jmx-exporter:0.13.0-debian-10-r29"
    "zookeeper:3.6.1-debian-10-r37"
)

# Pull images from bitnami, tag it and load it to kind cluster
for image in "${kafkaChartImages[@]}"
do
  echo "----- ${image} ----"
  docker pull bitnami/${image}
  docker tag bitnami/${image} customer.io/library/${image}
  kind load docker-image customer.io/library/${image}
done