# Test Framework Multi-stage Dockerfile
FROM golang:1.24-alpine AS go-builder

# Install testing tools
RUN apk add --no-cache git make gcc musl-dev

WORKDIR /workspace
COPY tests/go.mod tests/go.sum ./
RUN go mod download

COPY tests/ .
RUN go build -o test-runner cmd/test-runner/main.go

# Final stage
FROM ubuntu:22.04

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    curl \
    wget \
    jq \
    iproute2 \
    iperf3 \
    netcat \
    dnsutils \
    tcpdump \
    python3 \
    python3-pip \
    kubectl \
    docker.io \
    && rm -rf /var/lib/apt/lists/*

# Install Kind
RUN curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64 && \
    chmod +x ./kind && \
    mv ./kind /usr/local/bin/kind

# Install Helm
RUN curl https://get.helm.sh/helm-v3.13.0-linux-amd64.tar.gz | tar -xz && \
    mv linux-amd64/helm /usr/local/bin/helm && \
    rm -rf linux-amd64

# Install Python testing dependencies
RUN pip3 install pytest pytest-html pytest-json-report requests pyyaml

# Copy test binaries and scripts
COPY --from=go-builder /workspace/test-runner /usr/local/bin/test-runner
COPY tests/ /tests/
COPY experiments/ /experiments/

# Create non-root user
RUN useradd -m -s /bin/bash tester && \
    usermod -aG docker tester

# Set working directory
WORKDIR /tests

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD test -f /usr/local/bin/test-runner || exit 1

# Switch to non-root user
USER tester

# Default command
CMD ["/usr/local/bin/test-runner"]