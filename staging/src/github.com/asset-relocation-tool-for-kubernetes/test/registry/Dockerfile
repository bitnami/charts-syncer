# Copyright 2021 VMware, Inc.
# SPDX-License-Identifier: BSD-2-Clause
FROM golang:1.17 as compiled
RUN cd /go && \
    GO111MODULE=off go get -u filippo.io/mkcert && \
    cd src/filippo.io/mkcert && \
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /go/bin/mkcert
COPY . /app/

RUN cd /app && \
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go test \
  -c /app/pkg/mover/registry_test.go -o /app/registry_test

FROM registry:2 as registry

FROM photon:4.0
RUN mkdir /data && tdnf install -y jq httpd
WORKDIR /data
ENV REGISTRY_TEST true
ENV CAROOT /data
ENV HOME /data
COPY --from=compiled /go/bin/mkcert /bin/mkcert
COPY --from=compiled /app/registry_test /bin/registry-test
COPY --from=compiled /app/assets/docker-login.sh /bin/docker-login.sh
COPY --from=compiled /app/examples/ /data/examples
COPY --from=registry /bin/registry /bin/registry
COPY --from=registry /etc/docker/registry/config.yml /data/registry-config.yml
COPY test/registry/local-registry.sh /bin/local-registry.sh
ENTRYPOINT ["/bin/local-registry.sh"]
CMD ["/bin/registry-test", "-test.v"]
