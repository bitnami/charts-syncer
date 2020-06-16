#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Constants
ROOT_DIR="$(cd "$( dirname "${BASH_SOURCE[0]}" )/.." >/dev/null && pwd)"
RESET='\033[0m'
GREEN='\033[38;5;2m'
RED='\033[38;5;1m'
YELLOW='\033[38;5;3m'

# Load Libraries
# shellcheck disable=SC1090
. "${ROOT_DIR}/script/libtest.sh"
# shellcheck disable=SC1090
. "${ROOT_DIR}/script/liblog.sh"

# Auxiliar functions
print_menu() {
    local script
    script=$(basename "${BASH_SOURCE[0]}")
    log "${RED}NAME${RESET}"
    log "    $(basename -s .sh "${BASH_SOURCE[0]}")"
    log ""
    log "${RED}SYNOPSIS${RESET}"
    log "    $script [${YELLOW}-h${RESET}] [${YELLOW}-u ${GREEN}\"domain\"${RESET}]"
    log ""
    log "${RED}DESCRIPTION${RESET}"
    log "    Script to setup Harbor on your K8s cluster."
    log ""
    log "    The options are as follow:"
    log ""
    log "      ${YELLOW}-u, --domain ${GREEN}[harbor domain]${RESET}          Harbor domain."
    log "      ${YELLOW}-h, --help${RESET}                            Print this help menu."
    log ""
    log "${RED}EXAMPLES${RESET}"
    log "      $script --help"
    log "      $script --domain \"harbor.local\""
    log ""
}

domain=""
help_menu=0
dry_run=0
while [[ "$#" -gt 0 ]]; do
    case "$1" in
        -h|--help)
            help_menu=1
            ;;
        -u|--domain)
            shift; domain="${1:?missing namespace}"
            ;;
        *)
            error "Invalid command line flag $1" >&2
            exit 1
            ;;
    esac
    shift
done

if [[ "$help_menu" -eq 1 ]]; then
    print_menu
    exit 0
fi

# We will use the Kafka chart as testing asset.
# kafka-11.2.0 chart images (including zookeeper dependencies)
kafkaChartImages=(
    "kafka:2.5.0-debian-10-r66"
    "kubectl:1.17.4-debian-10-r91"
    "minideb:buster"
    "kafka-exporter:1.2.0-debian-10-r140"
    "jmx-exporter:0.13.0-debian-10-r29"
    "zookeeper:3.6.1-debian-10-r37"
)

# Pull images from bitnami, tag it and push it to harbor
for image in "${kafkaChartImages[@]}"
do
  echo "----- ${image} ----"
  docker pull bitnami/${image}
  docker tag bitnami/${image} ${domain}/library/${image}
  docker push ${domain}/library/${image}
done