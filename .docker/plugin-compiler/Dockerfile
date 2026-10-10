# syntax=docker/dockerfile:1

ARG GO_VERSION=1.26.7
ARG DEBIAN_RELEASE=trixie

FROM golang:${GO_VERSION}-${DEBIAN_RELEASE}

ARG VERSION=dev
ARG REVISION=unknown

LABEL org.opencontainers.image.title="maroid-plugin-compiler" \
  org.opencontainers.image.description="Compiles a Maroid plugin into a shared object that the hub loads." \
  org.opencontainers.image.source="https://github.com/abgeo/maroid" \
  org.opencontainers.image.version="${VERSION}" \
  org.opencontainers.image.revision="${REVISION}"

WORKDIR /src
VOLUME /src

COPY build.sh /

ENTRYPOINT ["/build.sh"]
