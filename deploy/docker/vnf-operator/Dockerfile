# VNF Operator Multi-stage Dockerfile
# Stage 1: Build
FROM golang:1.24-alpine AS builder

# Security: Use non-root user for build
RUN adduser -D -s /bin/sh appuser

# Install security updates and necessary tools
RUN apk update && apk add --no-cache git ca-certificates tzdata && \
    update-ca-certificates

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

# Stage 2: Runtime
FROM scratch

# Import from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd

# Copy the binary
COPY --from=builder /workspace/manager .

# Use non-root user
USER appuser

# Expose ports
EXPOSE 8080 8081 9443

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD ["/manager", "--health-check"]

# Run the binary
ENTRYPOINT ["/manager"]