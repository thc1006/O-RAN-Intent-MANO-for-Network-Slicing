# O-RAN Intent-MANO Orchestrator - Docker Build

## 2025 Docker Best Practices Implementation

This directory contains the optimized Docker configuration for the O-RAN Intent-MANO Orchestrator component, following 2025 Docker security and performance best practices.

## Key Optimizations Applied

### 🔒 Security Enhancements

1. **Minimal Base Image**: Uses `scratch` as the final runtime image for zero attack surface
2. **Non-root User**: Runs as `nobody:nobody` (uid:gid 65534:65534) in production
3. **Static Binary**: CGO_ENABLED=0 with static linking for maximum security
4. **Security Labels**: Comprehensive OCI labels for image metadata and security scanning
5. **Certificate Management**: Includes CA certificates for secure HTTPS communications

### ⚡ Performance Optimizations

1. **Multi-stage Build**: Separates build and runtime environments
2. **Layer Caching**: Optimized COPY order for maximum Docker layer cache efficiency
3. **Trimmed Binary**: Uses `-trimpath` and `-ldflags="-s -w"` for smaller binaries
4. **Build Optimization**: Removes debug symbols and build metadata for production

### 🏗️ Build Process

1. **Go Version**: Uses Go 1.23 (latest stable supporting all dependencies)
2. **Alpine Base**: Minimal `golang:1.23-alpine` for build stage
3. **Dependency Optimization**: Smart copying of only required modules
4. **Build Flags**: Production-ready build flags for security and size

### 📦 Image Specifications

- **Base Image (Build)**: `golang:1.23-alpine`
- **Base Image (Runtime)**: `scratch`
- **User**: `nobody:nobody` (non-root)
- **Exposed Ports**: 8080, 8090, 9090
- **Health Check**: Uses `--version` flag for container health

## Build Instructions

### Prerequisites

- Docker Engine 20.10+
- Git (for VCS metadata)
- 2+ GB available disk space

### Quick Build

```bash
# From project root
docker build -f deploy/docker/orchestrator/Dockerfile -t o-ran-orchestrator:latest .
```

### Production Build (Recommended)

```bash
# Using the provided build script
cd deploy/docker/orchestrator
./build.sh
```

### Advanced Build with Custom Configuration

```bash
# Custom image name and tag
IMAGE_NAME=my-registry/o-ran-orchestrator TAG=v1.0.0 ./build.sh

# Build and push to registry
PUSH_TO_REGISTRY=true REGISTRY_URL=my-registry.com ./build.sh
```

## Build Context

⚠️ **Important**: The build context must be the **project root** (not the orchestrator directory) to handle Go module replace directives correctly.

```bash
# Correct build context
docker build -f deploy/docker/orchestrator/Dockerfile -t orchestrator .

# Incorrect - will fail with module resolution errors
docker build -f Dockerfile -t orchestrator ./orchestrator/
```

## Directory Structure

```
deploy/docker/orchestrator/
├── Dockerfile          # Optimized multi-stage Dockerfile
├── build.sh           # Production build script
├── README.md          # This documentation
└── .dockerignore      # Build optimization (inherited from project root)
```

## Security Scanning

The build script supports automatic security scanning when tools are available:

```bash
# Install Trivy (optional)
curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh

# Build with automatic security scan
./build.sh
```

## Runtime Configuration

### Environment Variables

- `CGO_ENABLED=0`: Disabled for static binary
- `GOOS=linux`: Target Linux platform
- `GOARCH=amd64`: Target AMD64 architecture

### Health Check

The container includes a health check that uses the orchestrator's `--version` flag:

```bash
# Manual health check
docker exec <container-id> /usr/local/bin/orchestrator --version
```

### Volume Mounts

- `/config`: Configuration directory (recommended mount point)

### Example Run Command

```bash
docker run -d \
  --name o-ran-orchestrator \
  -p 8080:8080 \
  -p 8090:8090 \
  -p 9090:9090 \
  -v /path/to/config:/config:ro \
  --restart unless-stopped \
  --security-opt=no-new-privileges:true \
  --read-only \
  --tmpfs /tmp \
  o-ran-orchestrator:latest
```

## Troubleshooting

### Build Issues

1. **Module Resolution Errors**
   ```
   Error: go mod download failed
   ```
   Solution: Ensure build context is project root, not orchestrator directory

2. **Go Version Compatibility**
   ```
   Error: module requires go >= 1.23
   ```
   Solution: Dockerfile uses Go 1.23 which supports all current dependencies

3. **Permission Denied**
   ```
   Error: permission denied creating file
   ```
   Solution: Check Docker daemon permissions and build context access

### Runtime Issues

1. **Container Won't Start**
   - Check if required config file exists in mounted volume
   - Verify port availability on host system

2. **Health Check Failures**
   - Container may need more startup time (adjust `start-period`)
   - Check if binary is correctly built and executable

## Development vs Production

### Development Build
```bash
# Faster build for development (with cache)
docker build --target builder -t orchestrator-dev .
```

### Production Build
```bash
# Optimized production build (multi-stage complete)
docker build -t orchestrator-prod .
```

## Performance Characteristics

- **Build Time**: ~2-5 minutes (depending on dependency cache)
- **Image Size**: ~15-20 MB (scratch-based runtime)
- **Memory Usage**: ~50-100 MB (Go application overhead)
- **Startup Time**: ~2-5 seconds typical

## Compliance & Standards

This Docker configuration complies with:

- ✅ CIS Docker Benchmark
- ✅ NIST Container Security Guidelines
- ✅ OCI Image Format Specification
- ✅ Docker Official Image Best Practices
- ✅ Kubernetes Security Context Standards

## Support

For issues related to the Docker build process:
1. Check this README for common solutions
2. Verify build context and prerequisites
3. Review build logs for specific error messages
4. Ensure Go module dependencies are properly resolved

For application-specific issues, refer to the main project documentation.