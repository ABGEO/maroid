# syntax=docker/dockerfile:1

ARG GO_VERSION=1.25.4

FROM golang:${GO_VERSION}

WORKDIR /src
VOLUME /src

COPY build.sh /

ENTRYPOINT ["/build.sh"]
