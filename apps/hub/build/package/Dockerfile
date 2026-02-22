# syntax=docker/dockerfile:1

ARG GO_VERSION=1.26.0
ARG DISTROLESS_VERSION=nonroot

FROM golang:${GO_VERSION} AS builder

WORKDIR /src

RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=bind,source=go.mod,target=go.mod \
  --mount=type=bind,source=go.sum,target=go.sum \
  go mod download -x && go mod verify

RUN --mount=type=bind,target=. \
  --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache \
  CGO_ENABLED=1 go build \
    -v \
    -ldflags="-s -w" \
    -trimpath \
    -o /usr/bin/hub \
    ./cmd

FROM gcr.io/distroless/base:${DISTROLESS_VERSION} AS runtime

COPY --from=builder /usr/bin/hub /usr/bin/hub

ENTRYPOINT ["/usr/bin/hub"]
