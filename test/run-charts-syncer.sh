#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Constants
ROOT_DIR="$(cd "$( dirname "${BASH_SOURCE[0]}" )/.." >/dev/null && pwd)"

## Wait for chartmuseum service (Timeout in 30s)
wait-for-port --state=inuse 8080

/tmp/dist/charts-syncer --config ${ROOT_DIR}/test/test-config.yaml --from-date 2020-10-01
