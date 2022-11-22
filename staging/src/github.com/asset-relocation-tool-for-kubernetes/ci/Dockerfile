# Copyright 2021 VMware, Inc.
# SPDX-License-Identifier: BSD-2-Clause
FROM harbor-repo.vmware.com/dockerhub-proxy-cache/library/photon:3.0-20210716

WORKDIR /

# Install required packages
RUN yum -y install build-essential coreutils gawk gcc git go-1.17.2-1.ph3.x86_64 jq make
ENV PATH="/root/go/bin:${PATH}"

# Install go packages used for building and testing
RUN go get -u github.com/maxbrunsfeld/counterfeiter/v6 \
    github.com/onsi/ginkgo/ginkgo \
    github.com/onsi/gomega

# Install Helm
RUN curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash

COPY assets/docker-login.sh /usr/local/bin/docker-login.sh
