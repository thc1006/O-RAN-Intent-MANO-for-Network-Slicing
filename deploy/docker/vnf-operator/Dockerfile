# VNF Operator Multi-stage Dockerfile
# Stage 1: Build
FROM golang:1.24.5-alpine3.19 AS builder

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

# Download dependencies
RUN go mod download

# Copy source code
COPY adapters/vnf-operator/ .

# Build the binary with security flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o manager cmd/manager/main.go

# Stage 2: Runtime with distroless for maximum security
FROM gcr.io/distroless/static:nonroot

# Security labels
LABEL org.opencontainers.image.title="VNF Operator Runtime"
LABEL security.scan="trivy,grype,snyk"
LABEL security.user="non-root"
LABEL security.distroless="true"

# Copy the binary
COPY --from=builder /workspace/manager /manager

# Use non-root user (distroless nonroot user: 65532)
USER 65532:65532

# Expose ports
EXPOSE 8080 8081 9443

# Security: Run with minimal distroless environment
ENTRYPOINT ["/manager"]