# syntax=docker/dockerfile:1
# ---- build stage ----
FROM golang:alpine AS builder
WORKDIR /src
ENV GOTOOLCHAIN=auto

# Cache dependencies with Go module cache mount
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# Static binary compilation with BuildKit compiler and module cache mounts
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/monitor ./cmd/monitor

# ---- runtime stage ----
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S monitor && adduser -S -G monitor monitor \
    && mkdir -p /app/session && chown -R monitor:monitor /app/session

WORKDIR /app

COPY --from=builder /out/monitor /app/monitor

USER monitor
ENTRYPOINT ["/app/monitor"]