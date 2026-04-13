#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Constants
FAILED_TEST=0
EXPECTED_REGISTRY='localhost:5000/library/containers'

## Check that WordPress deployment is using the expected registry
wordpressImage=$(kubectl get pods --selector=app.kubernetes.io/name=wordpress -ojsonpath='{.items[0].spec.containers[0].image}')
if [[ "${wordpressImage}" =~ ${EXPECTED_REGISTRY} ]]; then
    echo "[PASS] WordPress is using the expected registry: ${EXPECTED_REGISTRY}"
else
    echo "[FAILED] WordPress is not using the expected registry. Got: \"${wordpressImage}\", expected: \"${EXPECTED_REGISTRY}\""
    FAILED_TEST=1
fi

## Check that MariaDB deployment is using the expected registry
mariadbImage=$(kubectl get pods --selector=statefulset.kubernetes.io/pod-name=wordpress-test-mariadb-0 -ojsonpath='{.items[0].spec.containers[0].image}')
if [[ "${mariadbImage}" =~ ${EXPECTED_REGISTRY} ]]; then
    echo "[PASS] MariaDB is using the expected registry: ${EXPECTED_REGISTRY}"
else
    echo "[FAILED] MariaDB is not using the expected registry. Got: \"${mariadbImage}\", expected: \"${EXPECTED_REGISTRY}\""
    FAILED_TEST=1
fi

if [ ${FAILED_TEST} != 0 ]; then
    echo ""
    echo "Please fix above failed tests"
    exit 1
fi

