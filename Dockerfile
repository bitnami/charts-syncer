FROM bitnami/minideb:buster as build
RUN install_packages ca-certificates
RUN mkdir /workdir

FROM scratch
ARG IMAGE_VERSION
ENV IMAGE_VERSION=${IMAGE_VERSION}
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /workdir /tmp
COPY ./charts-syncer /
ENTRYPOINT [ "/charts-syncer" ]
