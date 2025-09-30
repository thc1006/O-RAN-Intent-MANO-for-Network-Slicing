# Logging Standardization Guide

## Overview

This project uses Go's structured logging package (`log/slog`) for consistent, structured logging across all components. The `pkg/logging` package provides a convenient wrapper around `slog` with sensible defaults and context-aware logging capabilities.

## Key Features

- **Structured Logging**: All log entries are JSON-formatted with key-value pairs
- **Context-Aware**: Store and retrieve loggers from context for request tracing
- **Multiple Log Levels**: Debug, Info, Warn, Error
- **Performance**: Efficient JSON serialization with minimal overhead
- **Type-Safe**: Compile-time type checking for structured fields

## Installation

Import the logging package in your Go files:

```go
import "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/logging"
```

## Basic Usage

### Simple Logging

```go
// Info level - general operational messages
logging.Info("service started",
    "port", 8080,
    "version", "1.0.0")

// Debug level - detailed diagnostic information
logging.Debug("processing request",
    "request_id", "req-123",
    "user_id", "user-456")

// Warn level - warning messages for non-critical issues
logging.Warn("high memory usage",
    "memory_mb", 1500,
    "threshold_mb", 1000)

// Error level - error messages for failures
logging.Error("database connection failed",
    "host", "db.example.com",
    "error", err)
```

### Context-Based Logging

Use context-based logging for request tracing and distributed operations:

```go
func HandleRequest(ctx context.Context, requestID string) {
    // Create logger with request context
    logger := logging.With("request_id", requestID, "handler", "api")
    ctx = logging.NewContext(ctx, logger)

    // Pass context to other functions
    processData(ctx)
}

func processData(ctx context.Context) {
    // Retrieve logger from context
    logger := logging.FromContext(ctx)
    logger.Info("processing data", "records", 100)
}
```

### Component-Specific Loggers

Create loggers with component-specific fields:

```go
// Create logger for orchestrator component
orchestratorLogger := logging.With(
    "component", "orchestrator",
    "version", "2.0.0")

orchestratorLogger.Info("state transition",
    "from", "processing",
    "to", "deployed",
    "slice_id", "slice-001")

// Create logger for O2 client
o2Logger := logging.With(
    "component", "o2_client",
    "api_version", "v1")

o2Logger.Info("API request",
    "endpoint", "/o2ims/v1/resourcePools",
    "duration_ms", 45)
```

## Log Levels

### Setting Log Level

Control verbosity by setting the minimum log level:

```go
import "log/slog"

// Development: verbose logging
logging.SetLevel(slog.LevelDebug)

// Production: standard logging
logging.SetLevel(slog.LevelInfo)

// High-load: minimal logging
logging.SetLevel(slog.LevelWarn)
```

### Log Level Guidelines

| Level | When to Use | Example |
|-------|-------------|---------|
| **Debug** | Detailed diagnostic information | Cache lookups, internal state changes, step-by-step processing |
| **Info** | General operational messages | Service started, request completed, configuration loaded |
| **Warn** | Non-critical issues that don't stop operation | High resource usage, deprecated API usage, retry attempts |
| **Error** | Errors that prevent operation completion | Connection failures, validation errors, deployment failures |

## Structured Fields

### Field Naming Conventions

Use lowercase with underscores for field names:

```go
// ✅ Good
logging.Info("request processed",
    "request_id", "req-123",
    "user_id", "user-456",
    "duration_ms", 150)

// ❌ Avoid
logging.Info("request processed",
    "RequestID", "req-123",
    "userId", "user-456",
    "durationMs", 150)
```

### Common Field Names

Standardized field names for consistency:

- `request_id` - HTTP/gRPC request identifier
- `user_id` - User identifier
- `slice_id` - Network slice identifier
- `component` - Component/service name
- `operation` - Operation being performed
- `duration_ms` - Duration in milliseconds
- `error` - Error value
- `status` - Operation status
- `version` - Version identifier

### Error Logging

Always include the error and relevant context:

```go
err := deployNetworkSlice(sliceID)
if err != nil {
    logging.Error("deployment failed",
        "slice_id", sliceID,
        "operation", "deploy",
        "error", err,
        "retry_count", 3)
}
```

## Performance Monitoring

### Operation Timing

Log operation start and completion with timing:

```go
func DeploySlice(sliceID string) error {
    startTime := time.Now()

    logging.Info("deployment started",
        "slice_id", sliceID)

    // Perform deployment
    err := performDeployment(sliceID)

    duration := time.Since(startTime)

    if err != nil {
        logging.Error("deployment failed",
            "slice_id", sliceID,
            "duration_ms", duration.Milliseconds(),
            "error", err)
        return err
    }

    logging.Info("deployment completed",
        "slice_id", sliceID,
        "duration_ms", duration.Milliseconds())

    return nil
}
```

### Checkpoints

Log checkpoints for long-running operations:

```go
func ProcessWorkflow(workflowID string) {
    logging.Info("workflow started", "workflow_id", workflowID)

    logging.Debug("workflow checkpoint",
        "workflow_id", workflowID,
        "step", "validation")
    validateInputs()

    logging.Debug("workflow checkpoint",
        "workflow_id", workflowID,
        "step", "resource_allocation")
    allocateResources()

    logging.Debug("workflow checkpoint",
        "workflow_id", workflowID,
        "step", "deployment")
    deploy()

    logging.Info("workflow completed", "workflow_id", workflowID)
}
```

## Migration from fmt.Printf/log.Printf

### Before (Old Style)

```go
fmt.Printf("Creating VXLAN interface vxlan%d\n", vxlanID)
log.Printf("Warning: failed to delete tunnel %d: %v", vxlanID, err)
log.Printf("Deployment completed for slice %s in %v", sliceID, duration)
```

### After (Structured Logging)

```go
logging.Info("creating VXLAN interface",
    "interface", fmt.Sprintf("vxlan%d", vxlanID),
    "vxlan_id", vxlanID)

logging.Warn("failed to delete tunnel",
    "vxlan_id", vxlanID,
    "error", err)

logging.Info("deployment completed",
    "slice_id", sliceID,
    "duration_ms", duration.Milliseconds())
```

## Output Format

All logs are output as JSON to stdout:

```json
{
  "time": "2025-09-30T10:15:30.123Z",
  "level": "INFO",
  "msg": "deployment completed",
  "slice_id": "slice-001",
  "duration_ms": 1523,
  "status": "success"
}
```

## Best Practices

### 1. Use Structured Fields

Always use structured key-value pairs instead of string formatting:

```go
// ✅ Good - structured
logging.Info("request processed",
    "method", "POST",
    "path", "/api/slices",
    "status", 200,
    "duration_ms", 45)

// ❌ Bad - formatted string
logging.Info(fmt.Sprintf("Request processed: POST /api/slices - 200 (45ms)"))
```

### 2. Include Relevant Context

Provide enough context to understand the log without additional information:

```go
// ✅ Good - sufficient context
logging.Error("resource allocation failed",
    "slice_id", "slice-001",
    "resource_type", "compute",
    "requested", 8,
    "available", 4,
    "error", err)

// ❌ Bad - insufficient context
logging.Error("allocation failed", "error", err)
```

### 3. Use Appropriate Log Levels

Choose the correct log level based on severity:

```go
// ✅ Good - correct levels
logging.Debug("cache hit", "key", "config:123")
logging.Info("service started", "port", 8080)
logging.Warn("retry attempt", "attempt", 2, "max", 3)
logging.Error("connection failed", "error", err)

// ❌ Bad - everything as Info
logging.Info("cache hit", "key", "config:123")
logging.Info("connection failed", "error", err)
```

### 4. Don't Log Sensitive Data

Never log passwords, tokens, or sensitive information:

```go
// ✅ Good - no sensitive data
logging.Info("authentication successful",
    "user_id", userID,
    "method", "oauth")

// ❌ Bad - logging sensitive data
logging.Info("authentication successful",
    "user_id", userID,
    "password", password,  // DON'T DO THIS!
    "token", token)        // DON'T DO THIS!
```

### 5. Use Context for Request Tracing

Pass loggers through context for distributed tracing:

```go
func HandleAPIRequest(w http.ResponseWriter, r *http.Request) {
    requestID := generateRequestID()

    // Create logger with request context
    logger := logging.With(
        "request_id", requestID,
        "method", r.Method,
        "path", r.URL.Path)

    ctx := logging.NewContext(r.Context(), logger)

    // All downstream functions can use the same logger
    processRequest(ctx, r)
}
```

## Testing

The logging package includes comprehensive tests. Run them with:

```bash
go test -v ./pkg/logging/...
```

Example test coverage:
- Log level filtering
- Structured field output
- Context propagation
- Multiple field types (string, int, float, bool)
- JSON output format validation

## Examples

See `work_dir/examples/logging-example.go` for complete examples demonstrating:

1. Basic structured logging
2. Context-based logging
3. Error logging with context
4. Different log levels
5. Component-specific loggers
6. Performance monitoring
7. Dynamic log level setting
8. Structured error handling

Run the example:

```bash
go run work_dir/examples/logging-example.go
```

## Components Updated

The following components have been migrated to structured logging:

### ✅ Completed
- `pkg/logging` - Logging utility package
- `o2-client/pkg/o2ims` - O2 IMS client
- `o2-client/pkg/o2dms` - O2 DMS client
- `tn/agent/pkg/vxlan` - VXLAN manager

### 🔄 In Progress
- Orchestrator components
- Nephio renderer
- Additional test utilities

## Integration with Monitoring

Structured JSON logs can be easily integrated with log aggregation systems:

### ELK Stack (Elasticsearch, Logstash, Kibana)
```bash
# Logs can be directly ingested by Logstash
./app 2>&1 | logstash -f logstash.conf
```

### Grafana Loki
```bash
# Use promtail to ship logs to Loki
./app 2>&1 | promtail --config.file=promtail.yaml
```

### Cloud Logging (GCP, AWS CloudWatch)
Structured JSON logs are automatically parsed by cloud logging services.

## FAQ

### Q: Can I use fmt.Printf for quick debugging?

A: Use `logging.Debug()` instead. It provides the same immediate feedback but maintains structure:

```go
// Instead of: fmt.Printf("value: %v\n", value)
logging.Debug("debug value", "value", value)
```

### Q: How do I log multi-line error messages?

A: Store the error in a field and add context:

```go
err := complexOperation()
if err != nil {
    logging.Error("complex operation failed",
        "operation", "deploy",
        "error", err,  // Full error message preserved
        "context", "additional details")
}
```

### Q: Can I change log format from JSON?

A: Yes, modify the logger initialization in `pkg/logging/logger.go`:

```go
// For text format
defaultLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))
```

### Q: How do I aggregate logs from multiple services?

A: Use consistent field names (request_id, slice_id, etc.) across services. Log aggregation tools can then correlate logs using these fields.

## References

- [Go slog documentation](https://pkg.go.dev/log/slog)
- [Structured Logging Best Practices](https://www.loggly.com/ultimate-guide/structured-logging/)
- [12-Factor App Logging](https://12factor.net/logs)

## Support

For questions or issues with logging:
1. Check the examples in `work_dir/examples/logging-example.go`
2. Review the tests in `pkg/logging/logger_test.go`
3. Open an issue with the logging tag