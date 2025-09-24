# Orchestrator Multi-stage Dockerfile
# Stage 1: Build
FROM golang:1.24.7-alpine AS builder

# Security labels and metadata
LABEL maintainer="O-RAN MANO Team"
LABEL org.opencontainers.image.title="Orchestrator"
LABEL org.opencontainers.image.description="O-RAN Intent-based Orchestrator"
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
    tzdata \
    gcc \
    musl-dev && \
    update-ca-certificates && \
    rm -rf /var/cache/apk/* /tmp/* /var/tmp/*

# Set working directory
WORKDIR /workspace

# Copy go mod files first for better layer caching
COPY orchestrator/go.mod orchestrator/go.sum ./

# Copy pkg/security dependency (required by orchestrator)
# This is needed because go.mod has a replace directive pointing to ../pkg/security
COPY pkg/security/ ../pkg/security/

# Download dependencies with explicit toolchain
ENV GOTOOLCHAIN=go1.24.7
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org

# Disable workspace mode for Docker build
ENV GOWORK=off

RUN go mod download || (sleep 2 && go mod download) || (sleep 5 && go mod download)

# Copy source code
COPY orchestrator/ .

# Build the binary with security flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o orchestrator cmd/orchestrator/main.go

# Stage 2: Runtime
FROM alpine:3.22.1

# Security labels
LABEL org.opencontainers.image.title="Orchestrator Runtime"
LABEL security.scan="trivy,grype,snyk"
LABEL security.user="non-root"
LABEL security.base.image="alpine:3.22.1"
LABEL security.go.version="1.23.6"
LABEL security.scan.date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# Install runtime dependencies and security updates, clean cache
RUN apk update && apk upgrade && apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    dumb-init && \
    update-ca-certificates && \
    adduser -D -s /bin/sh -u 10003 appuser && \
    rm -rf /var/cache/apk/* /tmp/* /var/tmp/*

# Copy the binary
COPY --from=builder /workspace/orchestrator /usr/local/bin/orchestrator

# Use non-root user
USER appuser

# Create volume for config
VOLUME ["/config"]

# Expose ports
EXPOSE 8080 8090 9090

# Health check with improved security
HEALTHCHECK --interval=30s --timeout=10s --start-period=45s --retries=3 \
    CMD curl -f --max-time 5 --retry 2 --retry-delay 1 http://localhost:8080/health || exit 1

# Security: Run as non-root with dumb-init
ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD ["/usr/local/bin/orchestrator", "--config=/config/orchestrator.yaml"]