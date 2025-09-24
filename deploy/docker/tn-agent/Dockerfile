# TN Agent Multi-stage Dockerfile (2025 Best Practices)
# Stage 1: Build
FROM golang:1.22.10-alpine AS builder

# Security and metadata labels
LABEL maintainer="O-RAN MANO Team"
LABEL org.opencontainers.image.title="TN Agent"
LABEL org.opencontainers.image.description="Transport Network Agent"
LABEL org.opencontainers.image.version="1.0.0"
LABEL org.opencontainers.image.vendor="O-RAN Alliance"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.url="https://o-ran.org"
LABEL security.scan="trivy,grype,snyk"
LABEL security.distroless="true"
LABEL security.user="non-root"
LABEL security.capabilities="NET_ADMIN,NET_RAW"

# Install build dependencies with cache mount
RUN --mount=type=cache,target=/var/cache/apk \
    apk add --no-cache \
    git \
    ca-certificates \
    tzdata && \
    update-ca-certificates

# Set working directory
WORKDIR /workspace

# Copy go mod files from project root for layer caching
COPY go.mod go.sum ./

# Copy required source code for tn module
COPY pkg/ pkg/
COPY tn/ tn/

# Set Go environment for reproducible builds
ENV GO111MODULE=on \
    GOPROXY=https://proxy.golang.org,direct \
    GOSUMDB=sum.golang.org \
    GOWORK=off

# Download dependencies with cache mount
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Build static binary with optimizations and trimpath
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w -extldflags '-static'" \
    -a -installsuffix cgo \
    -o /workspace/tn-agent ./tn/agent/main.go

# Stage 2: Minimal runtime (scratch-based)
FROM scratch AS runtime

# Security and runtime labels
LABEL org.opencontainers.image.title="TN Agent Runtime"
LABEL security.scan="trivy,grype,snyk"
LABEL security.user="non-root"
LABEL security.capabilities="NET_ADMIN,NET_RAW"
LABEL security.base.image="scratch"
LABEL security.go.version="1.22.10"

# Copy CA certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy passwd for nobody user
COPY --from=builder /etc/passwd /etc/passwd

# Copy the static binary
COPY --from=builder /workspace/tn-agent /usr/local/bin/tn-agent

# Use nobody user for security
USER nobody:nobody

# Expose application ports
EXPOSE 8080 9090 5201

# Note: TN Agent requires NET_ADMIN,NET_RAW capabilities via K8s security context:
# securityContext:
#   capabilities:
#     add: ["NET_ADMIN", "NET_RAW"]
#   runAsUser: 65534
#   runAsGroup: 65534
#   runAsNonRoot: true

# Run the application
ENTRYPOINT ["/usr/local/bin/tn-agent"]
CMD ["--config=/config/tn-agent.yaml"]