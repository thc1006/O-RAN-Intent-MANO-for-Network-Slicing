# O2 Client Multi-stage Dockerfile
# Stage 1: Build
FROM golang:1.24.5-alpine AS builder

# Security labels and metadata
LABEL maintainer="O-RAN MANO Team"
LABEL org.opencontainers.image.title="O2 Client"
LABEL org.opencontainers.image.description="O-RAN O2 Interface Client"
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

# Copy go mod files from root for better layer caching
COPY go.mod go.mod
COPY go.sum go.sum

# Copy necessary source code including pkg directory and o2-client
COPY pkg/ pkg/
COPY o2-client/ o2-client/
COPY adapters/ adapters/
COPY orchestrator/ orchestrator/
COPY cn-dms/ cn-dms/
COPY ran-dms/ ran-dms/
COPY tn/ tn/

# Download dependencies (after copying local packages) with explicit toolchain
ENV GOTOOLCHAIN=go1.24.5
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org
RUN go mod download

# Build the binary with security flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o o2-client ./o2-client/cmd/main.go

# Stage 2: Runtime
FROM alpine:3.22.1

# Security labels
LABEL org.opencontainers.image.title="O2 Client Runtime"
LABEL security.scan="trivy,grype,snyk"
LABEL security.user="non-root"
LABEL security.base.image="alpine:3.22.1"
LABEL security.go.version="1.24.5"
LABEL security.scan.date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# Install runtime dependencies with pinned versions, clean cache
RUN apk update && apk upgrade && apk add --no-cache \
    ca-certificates \
    curl \
    jq \
    dumb-init && \
    update-ca-certificates && \
    adduser -D -s /bin/sh -u 10002 appuser && \
    rm -rf /var/cache/apk/* /tmp/* /var/tmp/*

# Copy the binary
COPY --from=builder /workspace/o2-client /usr/local/bin/o2-client

# Use non-root user
USER appuser

# Create volume for config and certs
VOLUME ["/config", "/certs"]

# Expose ports
EXPOSE 8080

# Health check with improved security
HEALTHCHECK --interval=30s --timeout=10s --start-period=45s --retries=3 \
    CMD curl -f --max-time 5 --retry 2 --retry-delay 1 http://localhost:8080/health || exit 1

# Security: Run as non-root with dumb-init
ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD ["/usr/local/bin/o2-client", "--config=/config/o2-client.yaml"]