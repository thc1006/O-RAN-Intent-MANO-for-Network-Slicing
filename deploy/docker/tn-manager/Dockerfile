# TN Manager Multi-stage Dockerfile
# Stage 1: Build
FROM golang:1.24.0-alpine AS builder

# Security labels and metadata
LABEL maintainer="O-RAN MANO Team"
LABEL org.opencontainers.image.title="TN Manager"
LABEL org.opencontainers.image.description="Transport Network Manager"
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
COPY tn/go.mod tn/go.sum ./

# Copy the manager API module (required by replace directive)
COPY tn/manager/api/v1alpha1/ ./manager/api/v1alpha1/

# Copy pkg/security module before downloading dependencies
# This is needed because go.mod has a replace directive pointing to ../pkg/security
COPY pkg/security/ ../pkg/security/

# Download dependencies with explicit toolchain
ENV GOTOOLCHAIN=auto
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org
RUN go mod download

# Copy full source code
COPY tn/ .

# Build the binary with security flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o tn-manager cmd/manager/main.go

# Stage 2: Runtime
FROM alpine:3.22.1

# Security labels
LABEL org.opencontainers.image.title="TN Manager Runtime"
LABEL security.scan="trivy,grype,snyk"
LABEL security.user="non-root"
LABEL security.base.image="alpine:3.22.1"
LABEL security.go.version="1.24.0"
LABEL security.scan.date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# Install runtime dependencies, clean cache
RUN apk update && apk upgrade && apk add --no-cache \
    ca-certificates \
    iproute2 \
    iptables \
    curl \
    dumb-init && \
    update-ca-certificates && \
    adduser -D -s /bin/sh -u 10007 appuser && \
    rm -rf /var/cache/apk/* /tmp/* /var/tmp/*

# Copy the binary
COPY --from=builder /workspace/tn-manager /usr/local/bin/tn-manager

# Create necessary directories with proper permissions
RUN mkdir -p /var/log/tn-manager && \
    chown appuser:appuser /var/log/tn-manager

# Use non-root user (Note: May need elevated privileges for network operations via K8s security context)
USER appuser

# Create volume for config
VOLUME ["/config", "/var/log/tn-manager"]

# Expose ports
EXPOSE 8080 9090

# Health check with improved security
HEALTHCHECK --interval=30s --timeout=10s --start-period=45s --retries=3 \
    CMD curl -f --max-time 5 --retry 2 --retry-delay 1 http://localhost:8080/health || exit 1

# Security: Run as non-root with dumb-init
ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD ["/usr/local/bin/tn-manager", "--config=/config/tn-manager.yaml"]