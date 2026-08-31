# syntax=docker/dockerfile:1

ARG GO_VERSION=1.26.7

FROM golang:${GO_VERSION}

WORKDIR /src
VOLUME /src

COPY build.sh /

ENTRYPOINT ["/build.sh"]
