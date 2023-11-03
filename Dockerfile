FROM golang:1.17.13-alpine3.15 AS builder
RUN apk add --update --no-cache make
WORKDIR /charts-syncer
COPY . .
RUN make build

FROM bitnami/minideb:buster as system-deps
RUN install_packages ca-certificates
RUN mkdir /workdir

FROM scratch
ARG IMAGE_VERSION
ENV IMAGE_VERSION=${IMAGE_VERSION}
COPY --from=system-deps /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Workaround to have a /tmp folder in the scratch container
COPY --from=system-deps /workdir /tmp
COPY --from=builder /charts-syncer/dist/charts-syncer /
ENTRYPOINT [ "/charts-syncer" ]
