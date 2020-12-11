FROM bitnami/minideb:buster
ARG IMAGE_VERSION
ENV IMAGE_VERSION=${IMAGE_VERSION}

RUN install_packages ca-certificates curl && \
    curl -sL "https://get.helm.sh/helm-v3.2.1-linux-amd64.tar.gz" | tar -xz --strip-components=1 -C /usr/local/bin linux-amd64/helm && \
    chmod +x /usr/local/bin/helm

COPY ./dist/charts-syncer /
RUN chmod +x /charts-syncer

CMD [ "/charts-syncer" ]
