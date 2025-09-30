package main

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/logging"
)

func main() {
	// Example 1: Basic logging with structured fields
	basicLoggingExample()

	// Example 2: Context-based logging
	contextLoggingExample()

	// Example 3: Error logging with structured fields
	errorLoggingExample()

	// Example 4: Log levels
	logLevelsExample()

	// Example 5: Component-specific logger
	componentLoggerExample()

	// Example 6: Performance monitoring
	performanceLoggingExample()
}

// Example 1: Basic structured logging
func basicLoggingExample() {
	logging.Info("application started",
		"version", "1.0.0",
		"environment", "production",
		"port", 8080)

	logging.Info("database connected",
		"host", "localhost",
		"database", "oran_db",
		"pool_size", 10)

	logging.Info("orchestrator initialized",
		"workers", 4,
		"queue_size", 100,
		"timeout", "30s")
}

// Example 2: Context-based logging
func contextLoggingExample() {
	ctx := context.Background()

	// Create a logger with request-specific fields
	requestLogger := logging.With(
		"request_id", "req-12345",
		"user_id", "user-67890",
		"ip", "192.168.1.100",
	)

	// Store logger in context
	ctx = logging.NewContext(ctx, requestLogger)

	// Use logger from context in different functions
	processRequest(ctx, "deploy_network_slice")
}

func processRequest(ctx context.Context, operation string) {
	logger := logging.FromContext(ctx)

	logger.Info("processing request",
		"operation", operation,
		"timestamp", time.Now())

	// Simulate work
	result := performOperation(ctx, operation)

	logger.Info("request completed",
		"operation", operation,
		"duration_ms", 150,
		"result", result)
}

func performOperation(ctx context.Context, operation string) string {
	logger := logging.FromContext(ctx)

	logger.Debug("starting operation",
		"operation", operation,
		"step", "validation")

	// Simulate validation and processing
	return "success"
}

// Example 3: Error logging with context
func errorLoggingExample() {
	err := deployNetworkSlice("slice-001")
	if err != nil {
		logging.Error("deployment failed",
			"slice_id", "slice-001",
			"error", err,
			"retry_count", 3,
			"last_attempt", time.Now())
	}
}

func deployNetworkSlice(sliceID string) error {
	logging.Info("deploying network slice",
		"slice_id", sliceID,
		"qos_profile", "ultra-low-latency")

	// Simulate error
	err := errors.New("insufficient resources")

	if err != nil {
		logging.Warn("deployment attempt failed",
			"slice_id", sliceID,
			"error", err,
			"will_retry", true)
		return err
	}

	return nil
}

// Example 4: Different log levels
func logLevelsExample() {
	// Debug: Detailed information for debugging
	logging.Debug("cache lookup",
		"key", "config:slice:123",
		"cache_hit", true,
		"ttl_seconds", 300)

	// Info: General informational messages
	logging.Info("service health check",
		"status", "healthy",
		"uptime_seconds", 3600,
		"active_connections", 42)

	// Warn: Warning messages for non-critical issues
	logging.Warn("resource usage high",
		"component", "orchestrator",
		"cpu_percent", 85,
		"memory_mb", 1500,
		"threshold_percent", 80)

	// Error: Error messages for failures
	logging.Error("API request failed",
		"endpoint", "/api/v1/slices",
		"method", "POST",
		"status_code", 500,
		"error", "connection timeout",
		"retry_after_seconds", 30)
}

// Example 5: Component-specific logger
func componentLoggerExample() {
	// Create loggers for different components
	orchestratorLogger := logging.With(
		"component", "orchestrator",
		"version", "2.0.0",
	)

	o2ClientLogger := logging.With(
		"component", "o2_client",
		"api_version", "v1",
	)

	vxlanLogger := logging.With(
		"component", "vxlan_manager",
		"interface_count", 10,
	)

	// Use component-specific loggers
	orchestratorLogger.Info("state transition",
		"from", "processing",
		"to", "deployed",
		"slice_id", "slice-001")

	o2ClientLogger.Info("API request",
		"endpoint", "/o2ims/v1/resourcePools",
		"method", "GET",
		"duration_ms", 45)

	vxlanLogger.Info("tunnel created",
		"interface", "vxlan100",
		"vni", 100,
		"local_ip", "10.0.0.1",
		"remote_ip", "10.0.0.2")
}

// Example 6: Performance monitoring with logging
func performanceLoggingExample() {
	operation := "network_slice_deployment"
	startTime := time.Now()

	// Log operation start
	logging.Info("operation started",
		"operation", operation,
		"slice_id", "slice-002")

	// Simulate work with checkpoints
	checkpointLogging(operation, "resource_allocation", startTime)
	checkpointLogging(operation, "configuration", startTime)
	checkpointLogging(operation, "validation", startTime)

	duration := time.Since(startTime)

	// Log operation completion with metrics
	logging.Info("operation completed",
		"operation", operation,
		"duration_ms", duration.Milliseconds(),
		"total_steps", 3,
		"success", true)
}

func checkpointLogging(operation, step string, startTime time.Time) {
	stepDuration := time.Since(startTime)

	logging.Debug("operation checkpoint",
		"operation", operation,
		"step", step,
		"elapsed_ms", stepDuration.Milliseconds())

	// Simulate step work
	time.Sleep(10 * time.Millisecond)
}

// Example 7: Setting log level dynamically
func logLevelExample() {
	// Set to debug level for troubleshooting
	logging.SetLevel(slog.LevelDebug)

	logging.Debug("detailed debug info", "data", "some value")
	logging.Info("normal operation", "status", "ok")

	// Set to info level for production
	logging.SetLevel(slog.LevelInfo)

	logging.Debug("this won't be logged")
	logging.Info("this will be logged", "status", "ok")

	// Set to warn level for high-load scenarios
	logging.SetLevel(slog.LevelWarn)

	logging.Info("this won't be logged")
	logging.Warn("this will be logged", "issue", "high load")
	logging.Error("this will always be logged", "error", "critical")
}

// Example 8: Structured error handling with logging
func structuredErrorHandling() {
	// Create context with trace ID
	ctx := context.Background()
	traceID := "trace-abc123"
	logger := logging.With("trace_id", traceID)
	ctx = logging.NewContext(ctx, logger)

	// Chain of operations with structured logging
	if err := step1(ctx); err != nil {
		logging.FromContext(ctx).Error("step1 failed",
			"error", err,
			"recovery_action", "rollback")
		rollback(ctx)
		return
	}

	if err := step2(ctx); err != nil {
		logging.FromContext(ctx).Error("step2 failed",
			"error", err,
			"recovery_action", "retry")
		return
	}

	logging.FromContext(ctx).Info("workflow completed successfully",
		"total_steps", 2)
}

func step1(ctx context.Context) error {
	logging.FromContext(ctx).Debug("executing step1",
		"action", "validate_resources")
	return nil
}

func step2(ctx context.Context) error {
	logging.FromContext(ctx).Debug("executing step2",
		"action", "deploy_configuration")
	return nil
}

func rollback(ctx context.Context) {
	logging.FromContext(ctx).Warn("initiating rollback",
		"reason", "step1 failure")
}