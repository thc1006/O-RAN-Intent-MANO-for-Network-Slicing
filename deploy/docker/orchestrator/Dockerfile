# Orchestrator Multi-stage Dockerfile
# 2025 Docker Best Practices - Optimized for O-RAN Intent-MANO Orchestrator
# Build context: orchestrator directory (./orchestrator) with minimal dependencies

# Stage 1: Build
FROM golang:1.24-alpine AS builder

# Security and build metadata
LABEL org.opencontainers.image.title="O-RAN Orchestrator"
LABEL org.opencontainers.image.description="O-RAN Intent-based MANO Orchestrator"
LABEL org.opencontainers.image.version="1.0.0"
LABEL org.opencontainers.image.vendor="O-RAN Alliance"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.url="https://o-ran.org"
LABEL org.opencontainers.image.source="https://github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing"
LABEL org.opencontainers.image.documentation="https://github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing"

# Install build dependencies (minimal set for security)
RUN apk update && apk upgrade && apk add --no-cache \
    git \
    ca-certificates \
    tzdata && \
    update-ca-certificates && \
    rm -rf /var/cache/apk/* /tmp/* /var/tmp/*

# Set working directory for orchestrator build
WORKDIR /workspace

# Copy orchestrator module files for layer caching
COPY orchestrator/go.mod orchestrator/go.sum ./

# Copy only required security dependency
COPY pkg/security/ ../pkg/security/

# Set Go environment for optimized builds
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org
ENV GOWORK=off

# Download dependencies (orchestrator module level)
RUN go mod download && go mod verify

# Copy orchestrator source code
COPY orchestrator/ .

# Build optimized binary with 2025 security flags
RUN go build \
    -trimpath \
    -ldflags="-s -w -buildid= -extldflags=-static" \
    -tags=netgo \
    -a \
    -o /tmp/orchestrator \
    ./cmd/orchestrator

# Stage 2: Runtime (scratch for minimal attack surface)
FROM scratch AS runtime

# Runtime metadata
LABEL org.opencontainers.image.title="O-RAN Orchestrator Runtime"
LABEL org.opencontainers.image.base.name="scratch"
LABEL security.user="nobody"
LABEL security.go.version="1.22.10"

# Copy required certificates for HTTPS calls
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data for proper time handling
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy passwd and group files for non-root user
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy optimized binary
COPY --from=builder /tmp/orchestrator /usr/local/bin/orchestrator

# Use nobody user (uid:gid 65534:65534) for security
USER nobody:nobody

# Expose application ports
EXPOSE 8080 8090 9090

# Health check using version flag (since orchestrator supports --version)
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD ["/usr/local/bin/orchestrator", "--version"] || exit 1

# Optimized entrypoint
ENTRYPOINT ["/usr/local/bin/orchestrator"]

# Default configuration
CMD ["--config=/config/orchestrator.yaml", "--verbose"]