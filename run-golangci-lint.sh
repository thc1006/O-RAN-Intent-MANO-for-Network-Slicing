#!/bin/bash
# Run golangci-lint in Docker container with Go 1.24.7 toolchain

docker run --rm -v $(pwd):/app -w /app \
    -e GOTOOLCHAIN=go1.24.7 \
    -e GOPROXY=https://proxy.golang.org,direct \
    golangci/golangci-lint:v2.5.0 \
    golangci-lint run \
    --timeout=10m \
    --enable=gosec,gocritic,revive,staticcheck,unparam,unused,ineffassign,misspell,goconst,gocyclo \
    ./...