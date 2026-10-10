# syntax=docker/dockerfile:1

ARG GO_VERSION=1.26.7
ARG DEBIAN_RELEASE=trixie
ARG DISTROLESS_DEBIAN=debian13
ARG DISTROLESS_VERSION=nonroot

FROM golang:${GO_VERSION}-${DEBIAN_RELEASE} AS builder

WORKDIR /src

RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=bind,source=go.mod,target=go.mod \
  --mount=type=bind,source=go.sum,target=go.sum \
  go mod download && go mod verify

RUN --mount=type=bind,target=. \
  --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache \
  CGO_ENABLED=1 go build \
    -ldflags="-s -w" \
    -trimpath \
    -o /usr/bin/hub \
    ./cmd

FROM gcr.io/distroless/base-${DISTROLESS_DEBIAN}:${DISTROLESS_VERSION} AS runtime

ARG VERSION=dev
ARG REVISION=unknown

LABEL org.opencontainers.image.title="maroid-hub" \
  org.opencontainers.image.description="The host process of Maroid. It loads the plugins." \
  org.opencontainers.image.source="https://github.com/abgeo/maroid" \
  org.opencontainers.image.version="${VERSION}" \
  org.opencontainers.image.revision="${REVISION}"

COPY --from=builder /usr/bin/hub /usr/bin/hub

ENTRYPOINT ["/usr/bin/hub"]
