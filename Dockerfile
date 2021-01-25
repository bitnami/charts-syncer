FROM scratch
ARG IMAGE_VERSION
ENV IMAGE_VERSION=${IMAGE_VERSION}
COPY ./charts-syncer /
CMD [ "/charts-syncer" ]
