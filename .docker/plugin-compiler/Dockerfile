# syntax=docker/dockerfile:1

ARG GO_VERSION=1.26.0

FROM golang:${GO_VERSION}

WORKDIR /src
VOLUME /src

COPY build.sh /

ENTRYPOINT ["/build.sh"]
