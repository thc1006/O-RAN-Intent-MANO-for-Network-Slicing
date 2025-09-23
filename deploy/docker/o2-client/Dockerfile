# O2 Client Multi-stage Dockerfile
# Stage 1: Build
FROM golang:1.21-alpine AS builder

# Security: Use non-root user for build
RUN adduser -D -s /bin/sh appuser

# Install security updates and necessary tools
RUN apk update && apk add --no-cache git ca-certificates tzdata && \
    update-ca-certificates

# Set working directory
WORKDIR /workspace

# Copy go mod files first for better layer caching
COPY o2-client/go.mod go.mod
COPY o2-client/go.sum go.sum

# Download dependencies
RUN go mod download

# Copy source code
COPY o2-client/ .

# Build the binary with security flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o o2-client cmd/main.go

# Stage 2: Runtime
FROM alpine:3.18

# Install runtime dependencies and security updates
RUN apk update && apk add --no-cache ca-certificates curl jq && \
    update-ca-certificates && \
    adduser -D -s /bin/sh appuser

# Copy the binary
COPY --from=builder /workspace/o2-client /usr/local/bin/o2-client

# Use non-root user
USER appuser

# Create volume for config and certs
VOLUME ["/config", "/certs"]

# Expose ports
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run the binary
ENTRYPOINT ["/usr/local/bin/o2-client"]
CMD ["--config=/config/o2-client.yaml"]