#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Constants
ROOT_DIR="$(cd "$( dirname "${BASH_SOURCE[0]}" )/.." >/dev/null && pwd)"
FAILED_TEST=0
EXPECTED_REGISTRY='docker.io/bitnami'

## Check that Ghost deployment is using the expected registry
ghostImage=$(kubectl get pods --selector=app.kubernetes.io/name=ghost -ojsonpath='{.items[0].spec.containers[0].image}')
if [[ "${ghostImage}" =~ "${EXPECTED_REGISTRY}" ]]; then
    echo "[PASS] Ghost is using the expected registry: ${EXPECTED_REGISTRY}"
else
    echo "[FAILED] Ghost is not using the expected registry. Got: \"${ghostImage}\", expected: \"${EXPECTED_REGISTRY}\""
    FAILED_TEST=1
fi

## Check that MySQL deployment is using the expected registry
mysqlImage=$(kubectl get pods --selector=statefulset.kubernetes.io/pod-name=ghost-test-mysql-0 -ojsonpath='{.items[0].spec.containers[0].image}')
if [[ "${mysqlImage}" =~ "${EXPECTED_REGISTRY}" ]]; then
    echo "[PASS] MySQL is using the expected registry: ${EXPECTED_REGISTRY}"
else
    echo "[FAILED] MySQL is not using the expected registry. Got: \"${mysqlImage}\", expected: \"${EXPECTED_REGISTRY}\""
    FAILED_TEST=1
fi

if [ ${FAILED_TEST} != 0 ]; then
    echo ""
    echo "Please fix above failed tests"
    exit 1
fi

