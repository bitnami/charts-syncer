#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

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