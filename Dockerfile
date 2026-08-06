# ---- build stage ----
FROM golang:1.26-alpine AS builder
WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static binary, trimmed for size
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/monitor ./cmd/monitor

# ---- runtime stage ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S monitor && adduser -S -G monitor monitor

WORKDIR /app

COPY --from=builder /out/monitor /app/monitor

USER monitor
EXPOSE 0

ENTRYPOINT ["/app/monitor"]