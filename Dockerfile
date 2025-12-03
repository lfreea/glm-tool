# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=${VERSION}" \
    -o glm-tool ./cmd/server

# Runtime stage
FROM alpine:3.21

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates tzdata

# Create data directory for cache
RUN mkdir -p /data

# Copy binary from builder
COPY --from=builder /app/glm-tool .

# Set default cache path
ENV CACHE_PATH=/data/image_cache.db

# Expose port
EXPOSE 8080

# Run
ENTRYPOINT ["./glm-tool"]
