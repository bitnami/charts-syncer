#!/usr/bin/env bash

set -x
set -o nounset
set -o pipefail

helm install --username admin --password dummypassword --wait ghost-test oci://127.0.0.1:8080/library/ghost --set ghostHost=127.0.0.1 --set service.type=ClusterIP



