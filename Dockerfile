FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o hotdealworker .

# Create final minimal image
FROM scratch

# Copy timezone data for proper time handling
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
# Copy CA certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /build/hotdealworker /app/hotdealworker

# Set default environment variables
ENV TZ=Asia/Seoul
ENV REDIS_ADDR=redis:6379
ENV REDIS_DB=0
ENV REDIS_STREAM=streamHotdeals
ENV REDIS_STREAM_COUNT=1
ENV REDIS_STREAM_MAX_LENGTH=500
ENV MEMCACHE_ADDR=memcached:11211
ENV CRAWL_INTERVAL_SECONDS=60
ENV HOTDEAL_ENVIRONMENT=production
ENV LOG_LEVEL=info

# Run as non-root user (UID 1000)
USER 1000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/hotdealworker", "-health"]

# Run the application
ENTRYPOINT ["/app/hotdealworker"]
