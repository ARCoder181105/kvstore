# ─────────────────────────────────────────────────────────────────────────────
# Stage 1: Build the Go binary
# ─────────────────────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Download dependencies first (layer cache)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a statically linked binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o kvstore ./cmd/server

# ─────────────────────────────────────────────────────────────────────────────
# Stage 2: Minimal runtime image
# ─────────────────────────────────────────────────────────────────────────────
FROM alpine:3.19

# ca-certificates needed if the server ever makes outbound TLS calls
RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/kvstore .

# Data directory for AOF log + snapshot (mount a volume here in production)
RUN mkdir -p /app/data

# TCP protocol port | HTTP API + Raft RPC port
EXPOSE 6379 8080

# Override via env vars: NODE_ID, HTTP_ADDR, TCP_ADDR, PEERS, DATA_DIR
ENV NODE_ID=node1 \
    HTTP_ADDR=:8080 \
    TCP_ADDR=:6379 \
    DATA_DIR=/app/data \
    PEERS=""

ENTRYPOINT ["./kvstore"]
