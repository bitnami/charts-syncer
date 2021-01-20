FROM bitnami/minideb:buster
ARG IMAGE_VERSION
ENV IMAGE_VERSION=${IMAGE_VERSION}
COPY ./charts-syncer /
CMD [ "/charts-syncer" ]
