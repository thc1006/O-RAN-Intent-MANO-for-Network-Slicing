# VNF Operator Multi-stage Dockerfile
# Stage 1: Build
FROM golang:1.24.0-alpine AS builder

# Security labels and metadata
LABEL maintainer="O-RAN MANO Team"
LABEL org.opencontainers.image.title="VNF Operator"
LABEL org.opencontainers.image.description="Virtual Network Function Operator"
LABEL org.opencontainers.image.version="1.0.0"
LABEL org.opencontainers.image.vendor="O-RAN Alliance"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.url="https://o-ran.org"
LABEL security.scan="trivy,grype,snyk"
LABEL security.distroless="true"
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
COPY adapters/vnf-operator/go.mod go.mod
COPY adapters/vnf-operator/go.sum go.sum

# Copy pkg/security module (needed for transitive dependencies)
# The go.mod may have indirect dependencies on pkg/security through other modules
COPY pkg/security ../pkg/security

# Copy api module for local replace directive
COPY adapters/vnf-operator/api /workspace/api

# Download dependencies with explicit toolchain
ENV GOTOOLCHAIN=go1.24.5
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org
RUN go mod download

# Copy source code
COPY adapters/vnf-operator/ .

# Build the binary with security flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o manager cmd/manager/main.go

# Build a minimal health check binary
RUN cat > healthcheck.go <<'EOF' && \
package main
import (
    "fmt"
    "net/http"
    "os"
    "time"
)
func main() {
    client := &http.Client{Timeout: 3 * time.Second}
    resp, err := client.Get("http://localhost:8080/healthz")
    if err != nil {
        fmt.Printf("Health check failed: %v\n", err)
        os.Exit(1)
    }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        fmt.Printf("Health check failed with status: %d\n", resp.StatusCode)
        os.Exit(1)
    }
    fmt.Println("Health check passed")
}
EOF
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
        -ldflags='-w -s -extldflags "-static"' \
        -a -installsuffix cgo \
        -o healthcheck healthcheck.go

# Stage 2: Runtime with distroless for maximum security
FROM gcr.io/distroless/static:nonroot

# Security labels
LABEL org.opencontainers.image.title="VNF Operator Runtime"
LABEL security.scan="trivy,grype,snyk"
LABEL security.user="non-root"
LABEL security.distroless="true"
LABEL security.base.image="gcr.io/distroless/static:nonroot"
LABEL security.go.version="1.24.5"
LABEL security.scan.date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# Copy the binaries
COPY --from=builder /workspace/manager /manager
COPY --from=builder /workspace/healthcheck /healthcheck

# Use non-root user (distroless nonroot user: 65532)
USER 65532:65532

# Expose ports
EXPOSE 8080 8081 9443

# Health check - using custom healthcheck binary for distroless compatibility
# Checks the /healthz endpoint typically provided by controller-runtime
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["/healthcheck"]

# Security: Run with minimal distroless environment
ENTRYPOINT ["/manager"]