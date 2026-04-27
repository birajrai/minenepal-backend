# Build Stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o minenepal-backend ./cmd/server

# Runtime Stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create cache directories
RUN mkdir -p /app/cache/icons /app/cache/banners /app/cache/status

# Copy binary from builder
COPY --from=builder /app/minenepal-backend .

# Set environment
ENV PORT=10000
ENV HOST=0.0.0.0
ENV CACHE_DIR=/app/cache
ENV CACHE_TTL=15s
ENV BANNER_CACHE_TTL=24h
ENV SERVER_QUERY_TIMEOUT=5s
ENV VOTIFIER_TIMEOUT=5s
ENV LOG_LEVEL=info
ENV LOG_PRETTY=false

# Expose port
EXPOSE 10000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:10000/ || exit 1

# Run the binary
CMD ["./minenepal-backend"]