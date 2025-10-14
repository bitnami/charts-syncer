#!/usr/bin/env bash

set -x
set -o nounset
set -o pipefail

helm install --username admin --password dummypassword --wait wordpress-test oci://127.0.0.1:5000/library/wordpress --set service.type=ClusterIP
