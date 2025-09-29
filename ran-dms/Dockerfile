# syntax=docker/dockerfile:1.4
# Multi-stage build for RAN DMS

# Build arguments for customization
ARG GO_VERSION=1.23
ARG ALPINE_VERSION=3.19

FROM golang:${GO_VERSION}-alpine AS builder

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
    -o dms ./ran-dms/cmd/dms/

# Final stage - use distroless for better security
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

# Metadata
LABEL org.opencontainers.image.title="O-RAN RAN DMS"
LABEL org.opencontainers.image.description="O-RAN RAN Domain Management Service"
LABEL org.opencontainers.image.version="1.0.0"
LABEL org.opencontainers.image.vendor="O-RAN Alliance"
LABEL org.opencontainers.image.licenses="Apache-2.0"

# Copy the binary from builder with proper ownership
COPY --from=builder --chown=nonroot:nonroot /build/dms /usr/local/bin/ran-dms

EXPOSE 8081 9091

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD ["/usr/local/bin/ran-dms", "--version"] || exit 1

# Run the RAN DMS
ENTRYPOINT ["/usr/local/bin/ran-dms"]
CMD ["--config=/config/ran-dms.yaml"]