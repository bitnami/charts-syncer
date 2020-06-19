#!/usr/bin/env bash

set -o nounset
set -o pipefail

########################
# Waits until a pod log contains certain string
# Arguments:
#   $1 - pod (as a string)
#   $2 - string (as a string)
#   $3 - repetitions. Default: 1
#   $4 - max retries. Default: 12
#   $5 - sleep between retries (in seconds). Default: 5
#########################
wait_for_string_in_pod() {
    local -r pod="${1:?pod is missing}"
    local -r string="${2:?string is missing}"
    local -r repetitions="${3:-1}"
    local -r retries="${4:-12}"
    local -r sleep_time="${5:-5}"

    for ((i = 1 ; i <= retries ; i+=1 )); do
        matches=$(kubectl logs ${pod} | grep "${string}" | wc -l)
        if [ ${matches} -ge ${repetitions} ]; then
            break
        fi
        sleep "$sleep_time"
    done
}

## Wait until Ghost is up and running
ghostPod=$(kubectl get pods --selector=app.kubernetes.io/name=ghost -o  jsonpath='{.items[0].metadata.name}')
wait_for_string_in_pod ${ghostPod} "Your site is now available on" 2
# Even after printing that message in the log, the service is not available yet
sleep 10