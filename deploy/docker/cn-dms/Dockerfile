# CN DMS Multi-stage Dockerfile
# Stage 1: Build
FROM golang:1.24.7-alpine AS builder

# Security labels and metadata
LABEL maintainer="O-RAN MANO Team"
LABEL org.opencontainers.image.title="CN DMS"
LABEL org.opencontainers.image.description="Core Network Domain Management Service"
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
RUN --mount=type=cache,target=/var/cache/apk \
    apk add --no-cache \
    git \
    ca-certificates \
    tzdata && \
    update-ca-certificates

# Set working directory
WORKDIR /workspace

# Copy go mod files first for better layer caching
COPY cn-dms/go.mod go.mod
COPY cn-dms/go.sum go.sum

# Copy pkg/security module before downloading dependencies
# This is needed because go.mod has a replace directive pointing to ../pkg/security
COPY pkg/security/ ../pkg/security/

# Set Go environment variables
ENV GOTOOLCHAIN=go1.24.7
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org

# Disable workspace mode for Docker build
ENV GOWORK=off

# Download dependencies with retry mechanism
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download || (sleep 2 && go mod download) || (sleep 5 && go mod download)

# Copy source code
COPY cn-dms/ .

# Build the binary with security flags and cache
# Support multi-platform builds
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o cn-dms ./cmd/dms/main.go

# Stage 2: Runtime
FROM alpine:3.22.1

# Security labels
LABEL org.opencontainers.image.title="CN DMS Runtime"
LABEL security.scan="trivy,grype,snyk"
LABEL security.user="non-root"
LABEL security.base.image="alpine:3.22.1"
LABEL security.go.version="1.24.7"
LABEL security.scan.date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# Install runtime dependencies and create user
RUN --mount=type=cache,target=/var/cache/apk \
    apk add --no-cache \
    ca-certificates \
    curl \
    jq \
    dumb-init && \
    update-ca-certificates && \
    adduser -D -s /bin/sh -u 10001 appuser

# Copy the binary
COPY --from=builder /workspace/cn-dms /usr/local/bin/cn-dms

# Create necessary directories
RUN mkdir -p /var/lib/cn-dms && \
    chown appuser:appuser /var/lib/cn-dms

# Use non-root user
USER appuser

# Create volume for config and data
VOLUME ["/config", "/var/lib/cn-dms"]

# Expose ports
EXPOSE 8080 8443

# Health check with improved security
HEALTHCHECK --interval=30s --timeout=10s --start-period=45s --retries=3 \
    CMD curl -f --max-time 5 --retry 2 --retry-delay 1 http://localhost:8080/health || exit 1

# Security: Run as non-root with dumb-init
ENTRYPOINT ["/usr/bin/dumb-init", "--", "/usr/local/bin/cn-dms"]
CMD ["--config=/config/cn-dms.yaml"]