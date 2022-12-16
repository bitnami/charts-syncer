FROM bitnami/minideb:buster as build
RUN install_packages ca-certificates
RUN mkdir /workdir

FROM alpine:3.15
ARG IMAGE_VERSION
ENV IMAGE_VERSION=${IMAGE_VERSION}
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY ./charts-syncer /bin/
ENTRYPOINT [ "/bin/charts-syncer" ]
