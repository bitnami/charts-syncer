#!/usr/bin/env bash

set -o nounset
set -o pipefail

helm repo update
helm search repo target/ghost
helm install --wait ghost-test target/ghost --set ghostHost=127.0.0.1 --set service.type=ClusterIP
