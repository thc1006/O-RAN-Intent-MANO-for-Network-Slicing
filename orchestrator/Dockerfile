# Multi-stage build for O-RAN Orchestrator
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o orchestrator ./orchestrator/cmd/orchestrator/

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/orchestrator .

# Expose ports
EXPOSE 8080 9090

# Run the orchestrator
CMD ["./orchestrator"]