# TN Manager Multi-stage Dockerfile
# Stage 1: Build
FROM golang:1.24-alpine AS builder

# Security: Use non-root user for build
RUN adduser -D -s /bin/sh appuser

# Install security updates and necessary tools
RUN apk update && apk add --no-cache git ca-certificates tzdata && \
    update-ca-certificates

# Set working directory
WORKDIR /workspace

# Copy go mod files first for better layer caching
COPY tn/go.mod go.mod
COPY tn/go.sum go.sum

# Download dependencies
RUN go mod download

# Copy source code
COPY tn/ .

# Build the binary with security flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o tn-manager cmd/manager/main.go

# Stage 2: Runtime
FROM alpine:3.18

# Install runtime dependencies and security updates
RUN apk update && apk add --no-cache ca-certificates iproute2 iptables && \
    update-ca-certificates && \
    adduser -D -s /bin/sh appuser

# Copy the binary
COPY --from=builder /workspace/tn-manager /usr/local/bin/tn-manager

# Create necessary directories with proper permissions
RUN mkdir -p /var/log/tn-manager && \
    chown appuser:appuser /var/log/tn-manager

# Use non-root user (Note: May need elevated privileges for network operations)
USER appuser

# Create volume for config
VOLUME ["/config", "/var/log/tn-manager"]

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
ENTRYPOINT ["/usr/local/bin/tn-manager"]
CMD ["--config=/config/tn-manager.yaml"]