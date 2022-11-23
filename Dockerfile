FROM bitnami/minideb:buster as build
RUN install_packages ca-certificates
RUN mkdir /workdir

FROM golang:1.19.2-alpine as syncer

WORKDIR /workdir

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct

COPY . .

RUN CGO_ENABLED=0 GOOS=linux  go build  -o charts-syncer

FROM scratch
ARG IMAGE_VERSION
ENV IMAGE_VERSION=${IMAGE_VERSION}
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Workaround to have a /tmp folder in the scratch container
COPY --from=build /workdir /tmp
COPY --from=syncer /workdir/charts-syncer /
ENTRYPOINT [ "/charts-syncer" ]
