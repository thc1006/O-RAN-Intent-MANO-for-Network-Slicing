# syntax=docker/dockerfile:1.4
# Multi-stage build for O-RAN Orchestrator

# Build arguments for customization
ARG GO_VERSION=1.23
ARG ALPINE_VERSION=3.19

FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum* ./
# Download dependencies with BuildKit cache mount
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download 2>/dev/null || true

# Copy source code
COPY . .

# Build the binary with BuildKit cache mount and security flags
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -buildid= -extldflags=-static" \
    -tags=netgo \
    -a -installsuffix cgo \
    -o orchestrator ./orchestrator/cmd/orchestrator/

# Final stage - use distroless for better security
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

# Metadata
LABEL org.opencontainers.image.title="O-RAN Orchestrator"
LABEL org.opencontainers.image.description="O-RAN Intent-based MANO Orchestrator"
LABEL org.opencontainers.image.version="1.0.0"
LABEL org.opencontainers.image.vendor="O-RAN Alliance"
LABEL org.opencontainers.image.licenses="Apache-2.0"

# Copy the binary from builder with proper ownership
COPY --from=builder --chown=nonroot:nonroot /build/orchestrator /usr/local/bin/orchestrator

# Copy Web UI static files with proper ownership
COPY --from=builder --chown=nonroot:nonroot /build/web/static /app/web/static

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD ["/usr/local/bin/orchestrator", "--version"] || exit 1

# Run the orchestrator
ENTRYPOINT ["/usr/local/bin/orchestrator"]
CMD ["--config=/config/orchestrator.yaml"]