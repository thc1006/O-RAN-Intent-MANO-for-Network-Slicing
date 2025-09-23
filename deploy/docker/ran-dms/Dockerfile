# RAN DMS Multi-stage Dockerfile
# Stage 1: Build
FROM golang:1.23-alpine3.19 AS builder

# Security labels and metadata
LABEL maintainer="O-RAN MANO Team"
LABEL org.opencontainers.image.title="RAN DMS"
LABEL org.opencontainers.image.description="Radio Access Network Domain Management Service"
LABEL org.opencontainers.image.version="1.0.0"
LABEL org.opencontainers.image.vendor="O-RAN Alliance"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.url="https://o-ran.org"
LABEL security.scan="trivy,grype,snyk"
LABEL security.distroless="false"
LABEL security.user="non-root"

# Security: Use non-root user for build
RUN adduser -D -s /bin/sh appuser

# Install security updates and necessary tools, clean cache
RUN apk update && apk upgrade && apk add --no-cache \
    git \
    ca-certificates \
    tzdata && \
    update-ca-certificates && \
    rm -rf /var/cache/apk/* /tmp/* /var/tmp/*

# Set working directory
WORKDIR /workspace

# Copy go mod files first for better layer caching
COPY ran-dms/go.mod go.mod
COPY ran-dms/go.sum go.sum

# Download dependencies
RUN go mod download

# Copy source code
COPY ran-dms/ .

# Build the binary with security flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o ran-dms cmd/main.go

# Stage 2: Runtime
FROM alpine:3.19

# Security labels
LABEL org.opencontainers.image.title="RAN DMS Runtime"
LABEL security.scan="trivy,grype,snyk"
LABEL security.user="non-root"

# Install runtime dependencies and security updates, clean cache
RUN apk update && apk upgrade && apk add --no-cache \
    ca-certificates \
    curl \
    jq \
    dumb-init && \
    update-ca-certificates && \
    adduser -D -s /bin/sh -u 10004 appuser && \
    rm -rf /var/cache/apk/* /tmp/* /var/tmp/*

# Copy the binary
COPY --from=builder /workspace/ran-dms /usr/local/bin/ran-dms

# Create necessary directories
RUN mkdir -p /var/lib/ran-dms && \
    chown appuser:appuser /var/lib/ran-dms

# Use non-root user
USER appuser

# Create volume for config and data
VOLUME ["/config", "/var/lib/ran-dms"]

# Expose ports
EXPOSE 8080 8443

# Health check with improved security
HEALTHCHECK --interval=30s --timeout=10s --start-period=45s --retries=3 \
    CMD curl -f --max-time 5 --retry 2 --retry-delay 1 http://localhost:8080/health || exit 1

# Security: Run as non-root with dumb-init
ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD ["/usr/local/bin/ran-dms", "--config=/config/ran-dms.yaml"]