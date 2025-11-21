# syntax=docker/dockerfile:1

ARG COMPILER_VERSION=latest
ARG HUB_VERSION=latest

FROM ghcr.io/abgeo/maroid-plugin-compiler:${COMPILER_VERSION} AS builder

COPY ./plugins /plugins
COPY .docker/hub-with-plugins/plugins-builder.sh /

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache \
    /plugins-builder.sh

FROM ghcr.io/abgeo/maroid-hub:${HUB_VERSION} AS runtime

COPY --from=builder /plugin-out /home/nonroot/.maroid/plugins
