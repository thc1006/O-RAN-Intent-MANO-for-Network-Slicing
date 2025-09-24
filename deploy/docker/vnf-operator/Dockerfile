# VNF Operator Multi-stage Dockerfile
# Stage 1: Build with Go 1.22.10-alpine (match workflow requirements)
FROM golang:1.22.10-alpine AS builder

# Set Go environment variables
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org

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

# Install build dependencies and certificates
RUN apk add --no-cache \
    ca-certificates \
    git \
    tzdata && \
    update-ca-certificates && \
    rm -rf /var/cache/apk/* /tmp/* /var/tmp/*

# Set working directory to project root
WORKDIR /workspace

# Copy go.mod and go.sum from project root for dependency layer caching
COPY go.mod go.sum ./

# Download dependencies first for better layer caching
RUN go mod download

# Copy entire source code (build context should be project root)
COPY . .

# Build the VNF operator manager binary with optimization flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w -extldflags '-static'" \
    -a -installsuffix cgo \
    -o manager ./adapters/vnf-operator/cmd/manager/main.go

# Build optimized health check binary
RUN printf 'package main\nimport ("fmt";"net/http";"os";"time")\nfunc main() {\nclient := &http.Client{Timeout: 3 * time.Second}\nresp, err := client.Get("http://localhost:8081/healthz")\nif err != nil { fmt.Printf("Health check failed: %%v\\n", err); os.Exit(1) }\ndefer resp.Body.Close()\nif resp.StatusCode != 200 { fmt.Printf("Health check failed with status: %%d\\n", resp.StatusCode); os.Exit(1) }\nfmt.Println("Health check passed")\n}' > healthcheck.go

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w -extldflags '-static'" \
    -a -installsuffix cgo \
    -o healthcheck healthcheck.go

# Stage 2: Runtime with scratch for maximum security and minimal size
FROM scratch

# Security and runtime labels
LABEL org.opencontainers.image.title="VNF Operator Runtime"
LABEL org.opencontainers.image.description="Production VNF Operator Runtime"
LABEL org.opencontainers.image.vendor="O-RAN Alliance"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL security.scan="trivy,grype,snyk"
LABEL security.user="non-root"
LABEL security.distroless="true"
LABEL security.base.image="scratch"
LABEL security.go.version="1.22.10"
LABEL security.binary="static"

# Copy CA certificates for HTTPS calls
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data for proper time handling
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy passwd file for nobody user
COPY --from=builder /etc/passwd /etc/passwd

# Copy the optimized binaries
COPY --from=builder --chown=65534:65534 /workspace/manager /manager
COPY --from=builder --chown=65534:65534 /workspace/healthcheck /healthcheck

# Use nobody user for security (UID 65534)
USER 65534:65534

# Expose standard Kubernetes operator ports
# 8080: metrics endpoint
# 8081: health/readiness probes
# 9443: webhook server (if enabled)
EXPOSE 8080 8081 9443

# Health check using optimized binary
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
  CMD ["/healthcheck"]

# Run the VNF operator manager
ENTRYPOINT ["/manager"]