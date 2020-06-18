#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Constants
ROOT_DIR="$(cd "$( dirname "${BASH_SOURCE[0]}" )/.." >/dev/null && pwd)"
FAILED_TEST=0

## Wait for Ghost service (Timeout in 30s)
wait-for-port --state=inuse 80

## Check that Ghost service is running
if curl -sI http://127.0.0.1 | grep -q "200 OK" && curl -s http://127.0.0.1 | grep -q "Welcome to Ghost" ; then
    echo "[PASS] Ghost service running."
else
    echo "[FAILED] No Ghost service found"
    FAILED_TEST=1
fi

## Check that Ghost deployment is using the expected image
expectedGhostImage='customer.io/library/ghost:3.20.1-debian-10-r0'
ghostImage=$(kubectl get pods --selector=app.kubernetes.io/name=ghost -o  jsonpath='{.items[0].spec.containers[0].image}')
if [ ${ghostImage} == ${expectedGhostImage} ]; then
    echo "[PASS] Ghost is using the expected image: ${ghostImage}"
else
    echo "[FAILED] Ghost is not using the expected image. Got ${ghostImage}, expected: ${expectedGhostImage}"
    FAILED_TEST=1
fi

## Check that Mariadb deployment is using the expected image
expectedMariadbImage='customer.io/library/mariadb:10.3.23-debian-10-r25"'
mariadbImage=$(kubectl get pods --selector=statefulset.kubernetes.io/pod-name=ghost-test-mariadb-0 -o  jsonpath='{.items[0].spec.containers[0].image}')
if [ ${mariadbImage} == ${expectedMariadbImage} ]; then
    echo "[PASS] Mariadb is using the expected image: ${mariadbImage}"
else
    echo "[FAILED] Mariadb is not using the expected image. Got ${mariadbImage}, expected: ${expectedMariadbImage}"
    FAILED_TEST=1
fi

if [ ${FAILED_TEST} != 0 ]; then
    echo ""
    echo "Please fix above failed tests"
    exit 1
fi

