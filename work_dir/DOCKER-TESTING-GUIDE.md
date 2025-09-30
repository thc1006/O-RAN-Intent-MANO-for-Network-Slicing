# Docker Testing Environment - Quick Reference Guide

## 🐳 O-RAN Intent MANO - Docker-Based Testing

This guide provides quick commands and workflows for running tests in Docker containers.

---

## 📋 Prerequisites

- Docker 20.10+
- Docker Compose 2.0+
- 4GB+ available memory
- Internet connection (for first build)

---

## 🚀 Quick Start

### 1. Build Test Environment

```bash
# From project root
cd C:/Users/thc1006/Desktop/dev/O-RAN-Intent-MANO-for-Network-Slicing

# Build test image
docker build -f work_dir/Dockerfile.test -t oran-intent-mano-test:latest .
```

### 2. Run All Tests

```bash
# Using Docker Compose (recommended)
docker-compose -f work_dir/docker-compose.test.yml up test-all

# Or using the test runner script
docker-compose -f work_dir/docker-compose.test.yml up test-runner
```

---

## 📦 Individual Test Suites

### Run Specific Component Tests

```bash
# Nephio Renderer Tests
docker-compose -f work_dir/docker-compose.test.yml up test-nephio

# O2 Client Tests
docker-compose -f work_dir/docker-compose.test.yml up test-o2client

# ConfigWatcher Tests
docker-compose -f work_dir/docker-compose.test.yml up test-configwatcher

# ParseFloat64 Tests
docker-compose -f work_dir/docker-compose.test.yml up test-parsefloat

# VXLAN Optimization Tests (stubs)
docker-compose -f work_dir/docker-compose.test.yml up test-vxlan
```

---

## 📊 Coverage Reports

### Generate HTML Coverage Report

```bash
# Run tests with coverage first
docker-compose -f work_dir/docker-compose.test.yml up test-all

# Generate HTML report
docker-compose -f work_dir/docker-compose.test.yml up coverage-report

# View report (open in browser)
# File: work_dir/reports/coverage.html
```

### View Coverage in Terminal

```bash
# Build and run with coverage output
docker run --rm -v "$(pwd)/work_dir:/workspace/work_dir" \
  oran-intent-mano-test:latest \
  sh -c "go test -coverprofile=/workspace/work_dir/reports/coverage.out ./work_dir/tests/... && \
         go tool cover -func=/workspace/work_dir/reports/coverage.out"
```

---

## 🔄 Parallel Test Execution

### Run All Test Suites in Parallel

```bash
docker-compose -f work_dir/docker-compose.test.yml up \
  test-nephio \
  test-o2client \
  test-configwatcher \
  test-parsefloat \
  test-vxlan
```

### Run Tests with Race Detection

```bash
docker run --rm -v "$(pwd)/work_dir:/workspace/work_dir" \
  oran-intent-mano-test:latest \
  go test -race -v ./work_dir/tests/...
```

---

## 🛠️ Advanced Usage

### Interactive Test Container

```bash
# Start interactive shell in test container
docker run -it --rm \
  -v "$(pwd)/work_dir:/workspace/work_dir" \
  oran-intent-mano-test:latest \
  /bin/sh

# Inside container:
cd /workspace/work_dir/tests
go test -v nephio_renderer_test.go
```

### Custom Test Commands

```bash
# Run specific test function
docker run --rm -v "$(pwd)/work_dir:/workspace/work_dir" \
  oran-intent-mano-test:latest \
  go test -v -run TestO2ClientAuthentication ./work_dir/tests/

# Run tests with verbose output and coverage
docker run --rm -v "$(pwd)/work_dir:/workspace/work_dir" \
  oran-intent-mano-test:latest \
  go test -v -coverprofile=/workspace/work_dir/reports/coverage.out \
  -covermode=atomic ./work_dir/tests/...

# Run tests with JSON output
docker run --rm -v "$(pwd)/work_dir:/workspace/work_dir" \
  oran-intent-mano-test:latest \
  go test -json ./work_dir/tests/... > work_dir/reports/test-results.json
```

---

## 🧹 Cleanup

### Remove Test Containers and Volumes

```bash
# Stop and remove all test containers
docker-compose -f work_dir/docker-compose.test.yml down

# Remove with volumes
docker-compose -f work_dir/docker-compose.test.yml down -v

# Remove test image
docker rmi oran-intent-mano-test:latest
```

### Clean Test Reports

```bash
# Remove all test reports and coverage files
rm -rf work_dir/reports/*.out
rm -rf work_dir/reports/*.html
rm -rf work_dir/reports/*.log
```

---

## 📝 Test Runner Script

### Using the Bash Test Runner

```bash
# Make script executable (first time only)
chmod +x work_dir/run-tests.sh

# Run in Docker
docker run --rm \
  -v "$(pwd)/work_dir:/workspace/work_dir" \
  oran-intent-mano-test:latest \
  /workspace/work_dir/run-tests.sh
```

---

## 🐛 Troubleshooting

### Test Failures

```bash
# View test logs
docker-compose -f work_dir/docker-compose.test.yml logs test-all

# View specific service logs
docker-compose -f work_dir/docker-compose.test.yml logs test-nephio
```

### Build Issues

```bash
# Clean build with no cache
docker build --no-cache -f work_dir/Dockerfile.test -t oran-intent-mano-test:latest .

# Verify Go modules
docker run --rm \
  -v "$(pwd):/workspace" \
  oran-intent-mano-test:latest \
  go mod verify
```

### Network Issues

```bash
# Check Docker network
docker network ls

# Inspect test network
docker network inspect work_dir_test-network
```

---

## 📊 Test Metrics

### Performance Benchmarks

```bash
# Run benchmarks
docker run --rm -v "$(pwd)/work_dir:/workspace/work_dir" \
  oran-intent-mano-test:latest \
  go test -bench=. -benchmem ./work_dir/tests/...
```

### Memory Profiling

```bash
# Generate memory profile
docker run --rm -v "$(pwd)/work_dir:/workspace/work_dir" \
  oran-intent-mano-test:latest \
  go test -memprofile=/workspace/work_dir/reports/mem.prof ./work_dir/tests/...

# Analyze with pprof
go tool pprof work_dir/reports/mem.prof
```

---

## 🔧 Dockerfile Stages

### Available Build Stages

1. **test-base**: Base testing environment with Go and tools
2. **test-runner**: Test execution with comprehensive runner script
3. **coverage-reporter**: HTML coverage report generator

### Build Specific Stage

```bash
# Build only test-base
docker build --target test-base -f work_dir/Dockerfile.test -t oran-test-base:latest .

# Build test-runner
docker build --target test-runner -f work_dir/Dockerfile.test -t oran-test-runner:latest .

# Build coverage-reporter
docker build --target coverage-reporter -f work_dir/Dockerfile.test -t oran-coverage:latest .
```

---

## 🎯 CI/CD Integration

### GitHub Actions Example

```yaml
name: Docker Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Build test image
        run: docker build -f work_dir/Dockerfile.test -t oran-test .

      - name: Run tests
        run: docker-compose -f work_dir/docker-compose.test.yml up --abort-on-container-exit test-all

      - name: Generate coverage
        run: docker-compose -f work_dir/docker-compose.test.yml up coverage-report

      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          files: ./work_dir/reports/coverage.out
```

---

## 📚 Additional Resources

- **Test Results Report**: `work_dir/reports/test-results.md`
- **Coverage Files**: `work_dir/reports/*.out`
- **Test Logs**: `work_dir/reports/*.log`
- **Dockerfile**: `work_dir/Dockerfile.test`
- **Compose Config**: `work_dir/docker-compose.test.yml`
- **Runner Script**: `work_dir/run-tests.sh`

---

## 💡 Pro Tips

1. **Use Docker Compose** for orchestrated testing
2. **Volume mount** your code for fast iteration
3. **Run parallel tests** with compose to save time
4. **Generate coverage** on every test run
5. **Use test cache** volume to speed up subsequent runs
6. **Interactive containers** for debugging test failures
7. **JSON output** for automated result parsing

---

**Last Updated**: 2025-09-30
**Project**: O-RAN Intent MANO for Network Slicing
**Docker Version**: 20.10+
**Go Version**: 1.23-alpine