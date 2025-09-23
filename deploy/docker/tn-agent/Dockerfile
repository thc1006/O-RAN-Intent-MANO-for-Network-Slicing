# TN Agent Multi-stage Dockerfile
# Stage 1: Build
FROM golang:1.24-alpine AS builder

# Security labels and metadata
LABEL maintainer="O-RAN MANO Team"
LABEL org.opencontainers.image.title="TN Agent"
LABEL org.opencontainers.image.description="Transport Network Agent"
LABEL org.opencontainers.image.version="1.0.0"
LABEL org.opencontainers.image.vendor="O-RAN Alliance"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.url="https://o-ran.org"
LABEL security.scan="trivy,grype,snyk"
LABEL security.distroless="false"
LABEL security.user="non-root"
LABEL security.capabilities="NET_ADMIN,NET_RAW"

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
COPY tn/go.mod go.mod
COPY tn/go.sum go.sum

# Download dependencies
RUN go mod download

# Copy source code and pkg/security
COPY tn/ .
COPY pkg/security/ ../pkg/security/

# Build the binary with security flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o tn-agent cmd/agent/main.go

# Stage 2: Runtime
FROM alpine:3.20

# Security labels
LABEL org.opencontainers.image.title="TN Agent Runtime"
LABEL security.scan="trivy,grype,snyk"
LABEL security.user="non-root"
LABEL security.capabilities="NET_ADMIN,NET_RAW"

# Install runtime dependencies and security updates for network operations, clean cache
RUN apk update && apk upgrade && apk add --no-cache \
    ca-certificates \
    iproute2 \
    iptables \
    bridge-utils \
    ethtool \
    iperf3 \
    curl \
    dumb-init && \
    update-ca-certificates && \
    adduser -D -s /bin/sh -u 10006 appuser && \
    rm -rf /var/cache/apk/* /tmp/* /var/tmp/*

# Copy the binary
COPY --from=builder /workspace/tn-agent /usr/local/bin/tn-agent

# Create necessary directories with proper permissions
RUN mkdir -p /var/log/tn-agent /var/lib/tn-agent && \
    chown -R appuser:appuser /var/log/tn-agent /var/lib/tn-agent

# Create volume for config and data
VOLUME ["/config", "/var/log/tn-agent", "/var/lib/tn-agent"]

# Expose ports
EXPOSE 8080 9090 5201

# Health check with improved security
HEALTHCHECK --interval=30s --timeout=10s --start-period=45s --retries=3 \
    CMD curl -f --max-time 5 --retry 2 --retry-delay 1 http://localhost:8080/health || exit 1

# Note: TN Agent needs network admin capabilities for traffic control
# This will be handled at runtime via Kubernetes security context:
# securityContext:
#   capabilities:
#     add: ["NET_ADMIN", "NET_RAW"]
#   runAsUser: 10006
#   runAsGroup: 10006
#   runAsNonRoot: true

# Switch to non-root user
USER appuser

# Security: Run as non-root with dumb-init
ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD ["/usr/local/bin/tn-agent", "--config=/config/tn-agent.yaml"]