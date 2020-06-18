#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Constants
ROOT_DIR="$(cd "$( dirname "${BASH_SOURCE[0]}" )/.." >/dev/null && pwd)"

## Wait for chartmuseum service (Timeout in 30s)
wait-for-port --state=inuse 8080

/tmp/dist/c3tsyncer syncChart --name mariadb --version 7.5.1 --config ${ROOT_DIR}/test/test-config.yaml
/tmp/dist/c3tsyncer syncChart --name common --version 0.3.1 --config ${ROOT_DIR}/test/test-config.yaml
/tmp/dist/c3tsyncer syncChart --name ghost --version 10.0.10 --config ${ROOT_DIR}/test/test-config.yaml