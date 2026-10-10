# syntax=docker/dockerfile:1

ARG NODE_VERSION=24.21.0
ARG PNPM_VERSION=12.11.2
ARG DEX_BASE_IMAGE=ghcr.io/dexidp/dex:v2.46.0

FROM --platform=$BUILDPLATFORM node:${NODE_VERSION}-alpine AS builder

ARG PNPM_VERSION
RUN npm install --global pnpm@${PNPM_VERSION}

WORKDIR /src

COPY pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm fetch --store-dir /pnpm/store

COPY . .
RUN pnpm install --store-dir /pnpm/store --offline --frozen-lockfile --filter @maroid/gate... \
  && pnpm --filter @maroid/gate build

FROM ${DEX_BASE_IMAGE} AS runtime

ARG VERSION=dev
ARG REVISION=unknown

LABEL org.opencontainers.image.title="maroid-gate" \
  org.opencontainers.image.description="The identity provider of Maroid: Dex with the Maroid theme." \
  org.opencontainers.image.source="https://github.com/abgeo/maroid" \
  org.opencontainers.image.url="https://github.com/abgeo/maroid" \
  org.opencontainers.image.documentation="https://github.com/abgeo/maroid" \
  org.opencontainers.image.version="${VERSION}" \
  org.opencontainers.image.revision="${REVISION}" \
  org.opencontainers.image.created="" \
  org.opencontainers.image.licenses=""

ENV DEX_FRONTEND_DIR=/srv/dex/web

COPY --chown=root:root --from=builder /src/apps/gate/web /srv/dex/web
