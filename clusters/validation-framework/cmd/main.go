// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

// GitOps Validation Framework Main Entry Point
// Provides comprehensive validation for O-RAN Intent-based MANO system
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	validation "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/clusters/validation-framework"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
)

var (
	configPath    = flag.String("config", "config.yaml", "Path to validation configuration file")
	clusterName   = flag.String("cluster", "", "Specific cluster to validate (if empty, validates all clusters)")
	validateOnly  = flag.Bool("validate-only", false, "Run validation only without continuous monitoring")
	outputFormat  = flag.String("output", "json", "Output format: json, yaml, table")
	outputFile    = flag.String("output-file", "", "Output file path (if empty, outputs to stdout)")
	logLevel      = flag.String("log-level", "info", "Log level: debug, info, warn, error")
	enableDrift   = flag.Bool("enable-drift", true, "Enable drift detection")
	enableMetrics = flag.Bool("enable-metrics", true, "Enable metrics collection")
	interval      = flag.Duration("interval", 5*time.Minute, "Validation interval for continuous monitoring")
)

func main() {
	flag.Parse()

	// Setup logging
	setupLogging(*logLevel)

	log.Printf("Starting O-RAN GitOps Validation Framework")
	log.Printf("Config: %s", *configPath)
	log.Printf("Cluster: %s", *clusterName)
	log.Printf("Validate Only: %v", *validateOnly)
	log.Printf("Metrics Enabled: %v", *enableMetrics)

	// Initialize validation framework
	framework, err := validation.NewValidationFramework(*configPath)
	if err != nil {
		log.Printf("Failed to initialize validation framework: %v", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var exitCode int
	if *validateOnly {
		// Run validation once and exit
		if err := runValidation(ctx, framework, *clusterName); err != nil {
			log.Printf("Validation failed: %v", err)
			exitCode = 1
		}
	} else {
		// Run continuous monitoring
		if err := runContinuousMonitoring(ctx, framework, *clusterName, *interval, sigChan); err != nil {
			log.Printf("Continuous monitoring failed: %v", err)
			exitCode = 1
		}
	}

	if exitCode == 0 {
		log.Printf("GitOps Validation Framework completed successfully")
	}
	os.Exit(exitCode)
}

// setupLogging configures logging based on log level
func setupLogging(level string) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	switch level {
	case "debug":
		log.SetOutput(os.Stdout)
	case "info":
		log.SetOutput(os.Stdout)
	case "warn":
		log.SetOutput(os.Stderr)
	case "error":
		log.SetOutput(os.Stderr)
	default:
		log.SetOutput(os.Stdout)
	}
}

// runValidation performs a single validation run
func runValidation(ctx context.Context, framework *validation.ValidationFramework, clusterName string) error {
	log.Printf("Running validation...")

	var results map[string]*validation.ValidationResult
	var err error

	if clusterName != "" {
		// Validate specific cluster
		result, err := framework.ValidateCluster(ctx, clusterName)
		if err != nil {
			return fmt.Errorf("cluster validation failed: %w", err)
		}
		results = map[string]*validation.ValidationResult{clusterName: result}
	} else {
		// Validate all clusters
		results, err = framework.ValidateAll(ctx)
		if err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	// Output results
	if err := outputResults(results); err != nil {
		return fmt.Errorf("failed to output results: %w", err)
	}

	// Check if any validations failed
	for cluster, result := range results {
		if !result.Success {
			log.Printf("Validation failed for cluster %s: %v", cluster, result.Errors)
			return fmt.Errorf("validation failed for cluster %s", cluster)
		}
	}

	log.Printf("All validations passed successfully")
	return nil
}

// runContinuousMonitoring runs continuous validation and monitoring
func runContinuousMonitoring(ctx context.Context, framework *validation.ValidationFramework, clusterName string, interval time.Duration, sigChan chan os.Signal) error {
	log.Printf("Starting continuous monitoring (interval: %v)", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run initial validation
	if err := runValidation(ctx, framework, clusterName); err != nil {
		log.Printf("Initial validation failed: %v", err)
	}

	// Start drift detection if enabled
	var driftTicker *time.Ticker
	if *enableDrift {
		driftTicker = time.NewTicker(30 * time.Second) // More frequent drift checks
		defer driftTicker.Stop()
	}

	for {
		select {
		case <-ctx.Done():
			log.Printf("Context canceled, stopping monitoring")
			return nil

		case <-sigChan:
			log.Printf("Received signal, stopping monitoring")
			return nil

		case <-ticker.C:
			log.Printf("Running scheduled validation...")
			if err := runValidation(ctx, framework, clusterName); err != nil {
				log.Printf("Scheduled validation failed: %v", err)

				// Try to trigger rollback if configured
				if err := handleValidationFailure(ctx, framework, clusterName, err); err != nil {
					log.Printf("Failed to handle validation failure: %v", err)
				}
			}

		case <-func() <-chan time.Time {
			if driftTicker != nil {
				return driftTicker.C
			}
			return make(chan time.Time) // Never triggers if drift detection is disabled
		}():
			log.Printf("Running drift detection...")
			if err := runDriftDetection(ctx, framework); err != nil {
				log.Printf("Drift detection failed: %v", err)
			}
		}
	}
}

// runDriftDetection performs drift detection across all clusters
func runDriftDetection(_ context.Context, framework *validation.ValidationFramework) error {
	// This would iterate through all clusters and run drift detection
	// For now, we'll implement a placeholder
	log.Printf("Drift detection completed (placeholder)")
	return nil
}

// handleValidationFailure handles validation failures
func handleValidationFailure(_ context.Context, framework *validation.ValidationFramework, clusterName string, validationErr error) error {
	log.Printf("Handling validation failure for cluster %s: %v", clusterName, validationErr)

	// This could trigger rollback, alerting, or other remediation actions
	// For now, just log the failure
	return nil
}

// outputResults outputs validation results in the specified format
func outputResults(results map[string]*validation.ValidationResult) error {
	var output []byte
	var err error

	switch *outputFormat {
	case "json":
		output, err = json.MarshalIndent(results, "", "  ")
	case "yaml":
		output, err = marshalYAML(results)
	case "table":
		output = []byte(formatTable(results))
	default:
		return fmt.Errorf("unsupported output format: %s", *outputFormat)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	// Output to file or stdout
	if *outputFile != "" {
		if err := writeToFile(*outputFile, output); err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}
		log.Printf("Results written to %s", *outputFile)
	} else {
		fmt.Print(string(output))
	}

	return nil
}

// marshalYAML marshals results to YAML format
func marshalYAML(results map[string]*validation.ValidationResult) ([]byte, error) {
	// Simple YAML marshaling - in production, use a proper YAML library
	output := "validation_results:\n"
	for cluster, result := range results {
		output += fmt.Sprintf("  %s:\n", cluster)
		output += fmt.Sprintf("    success: %v\n", result.Success)
		output += fmt.Sprintf("    timestamp: %s\n", result.Timestamp.Format(time.RFC3339))
		output += fmt.Sprintf("    duration: %s\n", result.Duration.String())

		if len(result.Errors) > 0 {
			output += "    errors:\n"
			for _, err := range result.Errors {
				output += fmt.Sprintf("      - %s\n", err)
			}
		}

		if len(result.Resources) > 0 {
			output += "    resources:\n"
			for _, resource := range result.Resources {
				output += fmt.Sprintf("      - name: %s\n", resource.Name)
				output += fmt.Sprintf("        kind: %s\n", resource.Kind)
				output += fmt.Sprintf("        ready: %v\n", resource.Ready)
				output += fmt.Sprintf("        status: %s\n", resource.Status)
			}
		}
	}
	return []byte(output), nil
}

// formatTable formats results as a table
func formatTable(results map[string]*validation.ValidationResult) string {
	output := "┌─────────────────┬─────────┬─────────────────────┬──────────┬─────────────┐\n"
	output += "│ Cluster         │ Success │ Timestamp           │ Duration │ Resources   │\n"
	output += "├─────────────────┼─────────┼─────────────────────┼──────────┼─────────────┤\n"

	for cluster, result := range results {
		status := "✓"
		if !result.Success {
			status = "✗"
		}

		output += fmt.Sprintf("│ %-15s │ %-7s │ %-19s │ %-8s │ %-11s │\n",
			truncateString(cluster, 15),
			status,
			result.Timestamp.Format("2006-01-02 15:04:05"),
			result.Duration.Truncate(time.Second).String(),
			fmt.Sprintf("%d", len(result.Resources)))

		// Add error details if any
		if !result.Success && len(result.Errors) > 0 {
			output += "│                 │         │                     │          │             │\n"
			for _, err := range result.Errors {
				output += fmt.Sprintf("│                 │ Error:  │ %-47s │\n",
					truncateString(err, 47))
			}
		}
	}

	output += "└─────────────────┴─────────┴─────────────────────┴──────────┴─────────────┘\n"
	return output
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// writeToFile writes data to a file
func writeToFile(filename string, data []byte) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, security.SecureDirMode); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(filename, data, security.SecureFileMode); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Version information
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

func init() {
	// Add version flag
	var version = flag.Bool("version", false, "Show version information")

	flag.Parse()

	if *version {
		fmt.Printf("O-RAN GitOps Validation Framework\n")
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		fmt.Printf("Build Time: %s\n", BuildTime)
		os.Exit(0)
	}
}
