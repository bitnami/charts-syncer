# Copyright 2021 VMware, Inc.
# SPDX-License-Identifier: BSD-2-Clause

# Distributes the relok8s binary previously crafted by the release process
# as part of the CI release action
FROM photon:4.0
ARG VERSION
ENV VERSION=${VERSION}

LABEL description="Asset Relocation Tool for Kubernetes"
LABEL maintainer="tanzu-isv-engineering@groups.vmware.com"
LABEL org.opencontainers.image.source https://github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes

# Deps required for docker-login and the additional testing performed in the CI using this image
# TODO: remove these dependencies
RUN yum -y install diffutils jq

COPY assets/docker-login.sh /usr/local/bin/docker-login.sh
COPY ./relok8s /usr/local/bin
ENTRYPOINT ["/usr/local/bin/relok8s"]

