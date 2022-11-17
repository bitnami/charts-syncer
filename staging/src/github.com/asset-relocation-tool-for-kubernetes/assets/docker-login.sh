#!/bin/bash
# Copyright 2021 VMware, Inc.
# SPDX-License-Identifier: BSD-2-Clause

set -euo pipefail

REGISTRY=$1
USERNAME=$2
PASSWORD=$3
DOCKER_CONFIG_FILE=${4:-~/.docker/config.json}

if [ ! -f "${DOCKER_CONFIG_FILE}" ] ; then
  mkdir -p "$(dirname "${DOCKER_CONFIG_FILE}")"
  echo '{}' > "${DOCKER_CONFIG_FILE}"
fi

AUTH_ENTRY=$(jq -n \
  --arg registry "${REGISTRY}" \
  --arg username "${USERNAME}" \
  --arg password "${PASSWORD}" \
  --arg auth "$(echo -n "${USERNAME}:${PASSWORD}" | base64)" \
  '{
    "auths": {
      ($registry): {
        "auth": $auth,
        "username": $username,
        "password": $password
      }
    }
  }')

jq -s '.[0] * .[1] | del(.credsStore)' "${DOCKER_CONFIG_FILE}" <(echo "${AUTH_ENTRY}") > "${DOCKER_CONFIG_FILE}.temp"
mv "${DOCKER_CONFIG_FILE}.temp" "${DOCKER_CONFIG_FILE}"
