# syntax=docker/dockerfile:1.4
# Multi-stage build for CN DMS

# Build arguments for customization
ARG GO_VERSION=1.24.7
ARG ALPINE_VERSION=3.20

# Using Alpine 3.20 (stable) to fix CVE-2025-46394 and CVE-2024-58251
FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

WORKDIR /build

COPY go.mod go.sum* ./
# Download dependencies with BuildKit cache mount
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download 2>/dev/null || true

COPY . .

# Build the binary with BuildKit cache mount and security flags
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -buildid= -extldflags=-static" \
    -tags=netgo \
    -a -installsuffix cgo \
    -o dms ./cn-dms/cmd/dms/

# Final stage - use distroless for better security
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

# Metadata
LABEL org.opencontainers.image.title="O-RAN CN DMS"
LABEL org.opencontainers.image.description="O-RAN Core Network Domain Management Service"
LABEL org.opencontainers.image.version="1.0.0"
LABEL org.opencontainers.image.vendor="O-RAN Alliance"
LABEL org.opencontainers.image.licenses="Apache-2.0"

# Copy the binary from builder with proper ownership
COPY --from=builder --chown=nonroot:nonroot /build/dms /usr/local/bin/cn-dms

EXPOSE 8082 9092

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD ["/usr/local/bin/cn-dms", "--version"] || exit 1

# Run the CN DMS
ENTRYPOINT ["/usr/local/bin/cn-dms"]
CMD ["--config=/config/cn-dms.yaml"]