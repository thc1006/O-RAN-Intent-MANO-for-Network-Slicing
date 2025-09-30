# O2 Client Multi-stage Dockerfile (2025 Best Practices)
# Stage 1: Build
# Using Alpine 3.20 (stable) to fix CVE-2025-46394 and CVE-2024-58251
FROM golang:1.24.7-alpine3.20 AS builder

# Security and metadata labels
LABEL maintainer="O-RAN MANO Team"
LABEL org.opencontainers.image.title="O2 Client"
LABEL org.opencontainers.image.description="O-RAN O2 Interface Client"
LABEL org.opencontainers.image.version="1.0.0"
LABEL org.opencontainers.image.vendor="O-RAN Alliance"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.url="https://o-ran.org"
LABEL security.scan="trivy,grype,snyk"
LABEL security.distroless="true"
LABEL security.user="non-root"

# Install build dependencies with cache mount and create passwd file
RUN --mount=type=cache,target=/var/cache/apk \
    apk add --no-cache \
    git \
    ca-certificates \
    tzdata && \
    update-ca-certificates && \
    echo "nobody:x:65534:65534:nobody:/:/sbin/nologin" > /etc/passwd && \
    echo "nobody:x:65534:" > /etc/group

# Set working directory
WORKDIR /workspace

# Copy workspace configuration and root module files
COPY go.work go.work.sum* ./
COPY go.mod go.sum ./

# Copy required source code maintaining workspace structure
COPY pkg/ pkg/
COPY o2-client/ o2-client/
COPY adapters/ adapters/
COPY orchestrator/ orchestrator/

# Set Go environment for workspace builds
ENV GO111MODULE=on \
    GOPROXY=https://proxy.golang.org,direct \
    GOSUMDB=sum.golang.org

# Download dependencies with increased timeout and retry logic
# Split into smaller steps to avoid timeout issues
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    set -e; \
    echo "Downloading o2-client dependencies..."; \
    cd o2-client && \
    timeout 300 go mod download 2>&1 || { \
        echo "First attempt timed out or failed, retrying..."; \
        timeout 300 go mod download 2>&1 || echo "Warning: go mod download failed, continuing..."; \
    }

# Sync workspace (optional, will use cached modules)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    timeout 120 go work sync 2>&1 || echo "Warning: go work sync failed, but cached modules should be sufficient"

# Build static binary from workspace root
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -trimpath \
    -ldflags="-s -w -extldflags '-static'" \
    -a -installsuffix cgo \
    -o /usr/local/bin/o2-client \
    ./o2-client/cmd/client/main.go

# Stage 2: Minimal runtime (scratch-based)
FROM scratch AS runtime

# Security and runtime labels
LABEL org.opencontainers.image.title="O2 Client Runtime"
LABEL security.scan="trivy,grype,snyk"
LABEL security.user="non-root"
LABEL security.base.image="scratch"
LABEL security.go.version="1.22.10"

# Copy CA certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy passwd and group for nobody user
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy the static binary
COPY --from=builder /usr/local/bin/o2-client /usr/local/bin/o2-client

# Use nobody user for security
USER nobody:nobody

# Expose application port
EXPOSE 8080

# HEALTHCHECK to monitor container health
# Checks HTTP health endpoint every 30s with 3s timeout
# Allows 3 retries before marking unhealthy (90s startup time)
HEALTHCHECK --interval=30s --timeout=3s --start-period=90s --retries=3 \
    CMD ["/usr/local/bin/o2-client", "--health-check"] || exit 1

# Run the application
ENTRYPOINT ["/usr/local/bin/o2-client"]
CMD ["--config=/config/o2-client.yaml"]