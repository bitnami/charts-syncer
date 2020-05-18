FROM bitnami/minideb:buster

RUN install_packages ca-certificates curl && \
    curl -sL "https://get.helm.sh/helm-v3.2.1-linux-amd64.tar.gz" | tar -xz --strip-components=1 -C /usr/local/bin linux-amd64/helm && \
    chmod +x /usr/local/bin/helm


COPY ./dist/c3tsyncer /

ENV IMAGE_VERSION="0.0.1-r0"
ENTRYPOINT ["/c3tsyncer"]
