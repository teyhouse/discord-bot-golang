# syntax=docker/dockerfile:1

FROM golang:1.26 AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bot ./cmd/bot

FROM scratch

COPY --from=builder /bot /bot
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

USER 65532:65532
WORKDIR /data

# State lives directly in the mounted volume, not a subdirectory of it.
ENV DATA_DIR=/data

ENTRYPOINT ["/bot"]
