# syntax=docker/dockerfile:1@sha256:572cebbc2b15e6f84c077742952674889cb63cd5e9b80abdffcbe587b6f19183

ARG NODE_VERSION=26.13.1
ARG PNPM_VERSION=12.23.0
ARG DISTROLESS_VERSION=nonroot@sha256:675c532423b908f9e56f9d60a4e62309b63cd2a3175acb90a46b96f7ee68e121

FROM --platform=$BUILDPLATFORM node:${NODE_VERSION}-alpine AS builder

ARG PNPM_VERSION
ENV PNPM_HOME=/pnpm
ENV PATH=$PNPM_HOME:$PATH
RUN npm install --global pnpm@${PNPM_VERSION}

WORKDIR /src

COPY pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm fetch --store-dir /pnpm/store

COPY . .
RUN pnpm install --store-dir /pnpm/store --prefer-offline --frozen-lockfile --filter @maroid/deck... \
  && pnpm --config.verify-deps-before-run=false --filter @maroid/deck build

RUN mkdir /out \
  && cp -R apps/deck/build /out/build \
  && printf '{"type":"module","private":true}\n' > /out/package.json

FROM gcr.io/distroless/nodejs24-debian13:${DISTROLESS_VERSION} AS runtime

ARG VERSION=dev
ARG REVISION=unknown

LABEL org.opencontainers.image.title="maroid-deck" \
  org.opencontainers.image.description="The web interface of Maroid." \
  org.opencontainers.image.source="https://github.com/abgeo/maroid" \
  org.opencontainers.image.version="${VERSION}" \
  org.opencontainers.image.revision="${REVISION}"

ENV NODE_ENV=production \
  PORT=3000

WORKDIR /app

COPY --from=builder /out ./

EXPOSE 3000

CMD ["build/index.js"]
