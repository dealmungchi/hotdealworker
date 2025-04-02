FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/hotdealworker

# Create final minimal image
FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/hotdealworker /app/hotdealworker

# Set environment variables (can be overridden)
ENV REDIS_ADDR=redis:6379
ENV REDIS_DB=0
ENV REDIS_STREAM_COUNT=1
ENV MEMCACHE_ADDR=memcached:11211
ENV CRAWL_INTERVAL_SECONDS=60
ENV HOTDEAL_ENVIRONMENT=production

# Run the application
CMD ["/app/hotdealworker"]