#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Constants
ROOT_DIR="$(cd "$( dirname "${BASH_SOURCE[0]}" )/.." >/dev/null && pwd)"
FAILED_TEST=0

########################
# Waits until a pod log contains certain string
# Arguments:
#   $1 - pod (as a string)
#   $2 - string (as a string)
#   $3 - max retries. Default: 12
#   $4 - sleep between retries (in seconds). Default: 5
#########################
wait_for_string_in_pod() {
    local -r pod="${1:?pod is missing}"
    local -r string="${2:?string is missing}"
    local -r retries="${3:-12}"
    local -r sleep_time="${4:-5}"

    for ((i = 1 ; i <= retries ; i+=1 )); do
        ghostLog=$(kubectl logs ${pod})
        if echo ${ghostLog} | grep -q "${string}"; then
            break
        fi
        sleep "$sleep_time"
    done
}


## Wait until Ghost is up and running
ghostPod=$(kubectl get pods --selector=app.kubernetes.io/name=ghost -o  jsonpath='{.items[0].metadata.name}')
wait_for_string_in_pod ${ghostPod} "Your site is now available on"

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

## TODO (tompizmor): Enable this test once issue is fixed in the tool
## Check that Mariadb deployment is using the expected image
# expectedMariadbImage='customer.io/library/mariadb:10.3.23-debian-10-r25"'
# mariadbImage=$(kubectl get pods --selector=statefulset.kubernetes.io/pod-name=ghost-test-mariadb-0 -o  jsonpath='{.items[0].spec.containers[0].image}')
# if [ ${mariadbImage} == ${expectedMariadbImage} ]; then
#     echo "[PASS] Mariadb is using the expected image: ${mariadbImage}"
# else
#     echo "[FAILED] Mariadb is not using the expected image. Got ${mariadbImage}, expected: ${expectedMariadbImage}"
#     FAILED_TEST=1
# fi

if [ ${FAILED_TEST} != 0 ]; then
    echo ""
    echo "Please fix above failed tests"
    exit 1
fi

