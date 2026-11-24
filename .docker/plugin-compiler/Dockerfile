# syntax=docker/dockerfile:1@sha256:572cebbc2b15e6f84c077742952674889cb63cd5e9b80abdffcbe587b6f19183

ARG GO_VERSION=1.27.4
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
