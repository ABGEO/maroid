# syntax=docker/dockerfile:1

ARG HUB_VERSION=latest
ARG COMPILER_VERSION=${HUB_VERSION}
ARG NODE_VERSION=24.15.0
ARG PNPM_VERSION=11.25.0

FROM --platform=$BUILDPLATFORM node:${NODE_VERSION}-alpine AS ui

ARG PNPM_VERSION
RUN npm install --global pnpm@${PNPM_VERSION}

WORKDIR /src

COPY pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm fetch --store-dir /pnpm/store

COPY . .
RUN pnpm install --store-dir /pnpm/store --offline --frozen-lockfile --filter '{./plugins/*/ui}...' \
  && pnpm --filter './plugins/*/ui' build \
  && mkdir /out \
  && for dist in plugins/*/ui/dist; do mkdir -p "/out/${dist%/dist}" && cp -R "$dist" "/out/$dist"; done

FROM ghcr.io/abgeo/maroid-plugin-compiler:${COMPILER_VERSION} AS builder

COPY ./plugins /plugins
COPY --from=ui /out/plugins /plugins
COPY .docker/hub-with-plugins/plugins-builder.sh /

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache \
    /plugins-builder.sh

FROM ghcr.io/abgeo/maroid-hub:${HUB_VERSION} AS runtime

ARG VERSION=dev
ARG REVISION=unknown

LABEL org.opencontainers.image.title="maroid-hub-with-plugins" \
  org.opencontainers.image.description="The host process of Maroid with every plugin of this repository." \
  org.opencontainers.image.source="https://github.com/abgeo/maroid" \
  org.opencontainers.image.version="${VERSION}" \
  org.opencontainers.image.revision="${REVISION}"

COPY --from=builder /plugin-out /home/nonroot/.maroid/plugins
