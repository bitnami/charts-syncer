# Copyright Broadcom, Inc. All Rights Reserved.
# SPDX-License-Identifier: APACHE-2.0

FROM docker.io/bitnami/minideb:bookworm AS builder

SHELL ["/bin/bash", "-o", "errexit", "-o", "nounset", "-o", "pipefail", "-c"]

RUN install_packages ca-certificates
RUN mkdir -p /rootfs/tmp && mkdir /rootfs/.charts-syncer && chmod g+rwX /rootfs/tmp /rootfs/.charts-syncer

######

FROM scratch

ARG TARGETARCH
ARG APP_VERSION
ENV OS_ARCH="${TARGETARCH:-amd64}"

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY dist/release_linux_${OS_ARCH}*/charts-syncer /opt/bitnami/charts-syncer/bin/charts-syncer
COPY --from=builder /rootfs /

ENV APP_VERSION="${APP_VERSION}" \
    BITNAMI_APP_NAME="charts-syncer" \
    PATH="/opt/bitnami/charts-syncer/bin:$PATH"

USER 1001

ENTRYPOINT [ "charts-syncer" ]
