# Test Framework Multi-stage Dockerfile
FROM golang:1.24.7-alpine AS go-builder

# Security labels and metadata
LABEL maintainer="O-RAN MANO Team"
LABEL org.opencontainers.image.title="Test Framework Builder"
LABEL org.opencontainers.image.description="O-RAN Test Framework Builder"
LABEL org.opencontainers.image.version="1.0.0"
LABEL org.opencontainers.image.vendor="O-RAN Alliance"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL security.scan="trivy,grype,snyk"
LABEL security.user="non-root"

# Install testing tools and clean cache
RUN apk update && apk upgrade && apk add --no-cache \
    git \
    make \
    gcc \
    musl-dev && \
    rm -rf /var/cache/apk/* /tmp/* /var/tmp/*

WORKDIR /workspace
COPY tests/go.mod tests/go.sum ./
RUN go mod download

COPY tests/ .
RUN go build -o test-runner cmd/test-runner/main.go

# Final stage - Using Debian slim instead of Ubuntu for better security
FROM debian:12.8-slim

# Security labels
LABEL org.opencontainers.image.title="Test Framework Runtime"
LABEL security.scan="trivy,grype,snyk"
LABEL security.user="non-root"
LABEL security.distroless="false"
LABEL security.base.image="debian:12.8-slim"
LABEL security.go.version="1.24.7"
LABEL security.scan.date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
LABEL security.risk="high-testing-tools"

# Install runtime dependencies with security updates, clean cache
RUN apt-get update && apt-get upgrade -y && apt-get install -y \
    curl \
    wget \
    jq \
    iproute2 \
    iperf3 \
    netcat-openbsd \
    dnsutils \
    tcpdump \
    python3 \
    python3-pip \
    ca-certificates \
    dumb-init \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Install kubectl with enhanced security verification
RUN curl -LO "https://dl.k8s.io/release/v1.28.0/bin/linux/amd64/kubectl" && \
    curl -LO "https://dl.k8s.io/v1.28.0/bin/linux/amd64/kubectl.sha256" && \
    echo "$(cat kubectl.sha256)  kubectl" | sha256sum --check && \
    # Verify kubectl signature (GPG verification would be ideal but adds complexity)
    chmod 755 kubectl && \
    ./kubectl version --client=true || (echo "kubectl verification failed" && exit 1) && \
    install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl && \
    rm kubectl kubectl.sha256

# Install Kind with enhanced security verification
RUN curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64 && \
    curl -Lo ./kind.sha256sum https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64.sha256sum && \
    echo "$(cat kind.sha256sum)  kind" | sha256sum --check && \
    chmod +x ./kind && \
    # Basic verification that kind binary works
    ./kind version || (echo "kind verification failed" && exit 1) && \
    mv ./kind /usr/local/bin/kind && \
    rm kind.sha256sum

# Install Helm with enhanced security verification
RUN curl -fsSL https://get.helm.sh/helm-v3.13.0-linux-amd64.tar.gz -o helm.tar.gz && \
    curl -fsSL https://get.helm.sh/helm-v3.13.0-linux-amd64.tar.gz.sha256sum -o helm.sha256sum && \
    echo "$(cat helm.sha256sum)  helm.tar.gz" | sha256sum --check && \
    tar -xzf helm.tar.gz && \
    # Verify helm binary works
    ./linux-amd64/helm version --client || (echo "helm verification failed" && exit 1) && \
    mv linux-amd64/helm /usr/local/bin/helm && \
    rm -rf linux-amd64 helm.tar.gz helm.sha256sum

# Install Python testing dependencies with pinned versions
COPY deploy/docker/test-framework/requirements.txt /tmp/requirements.txt
RUN echo "pytest==7.4.3\npytest-html==4.1.1\npytest-json-report==1.5.0\nrequests==2.31.0\nPyYAML==6.0.1" > /tmp/requirements.txt && \
    pip3 install --no-cache-dir -r /tmp/requirements.txt && \
    rm /tmp/requirements.txt

# Copy test binaries and scripts
COPY --from=go-builder /workspace/test-runner /usr/local/bin/test-runner
COPY tests/ /tests/
COPY experiments/ /experiments/

# Create non-root user with specific UID
RUN useradd -u 10005 -m -s /bin/bash tester

# Set proper permissions
RUN chown -R tester:tester /tests /experiments

# Set working directory
WORKDIR /tests

# Health check with improved security
HEALTHCHECK --interval=30s --timeout=10s --start-period=45s --retries=3 \
    CMD test -f /usr/local/bin/test-runner || exit 1

# Switch to non-root user
USER tester

# Security: Run with dumb-init
ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD ["/usr/local/bin/test-runner"]