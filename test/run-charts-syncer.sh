#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Constants
ROOT_DIR="$(cd "$( dirname "${BASH_SOURCE[0]}" )/.." >/dev/null && pwd)"

## Wait for harbor service (Timeout in 60s)
wait-for-port --state=inuse --timeout=60 8080
sleep 30

/tmp/dist/charts-syncer --config "${ROOT_DIR}/test/test-config.yaml" sync --latest-version-only --insecure --use-plain-http
