// Package main provides comprehensive test runner for O-RAN Intent-MANO system
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

// TestConfig defines test execution configuration
type TestConfig struct {
	TestSuite      string            `json:"test_suite"`
	Parallel       bool              `json:"parallel"`
	Verbose        bool              `json:"verbose"`
	Timeout        time.Duration     `json:"timeout"`
	Coverage       bool              `json:"coverage"`
	Environment    map[string]string `json:"environment"`
	KubeConfig     string            `json:"kube_config"`
	ExcludePatterns []string         `json:"exclude_patterns"`
}

// TestResult captures test execution results
type TestResult struct {
	Suite     string        `json:"suite"`
	Status    string        `json:"status"`
	Duration  time.Duration `json:"duration"`
	Coverage  float64       `json:"coverage"`
	Tests     int           `json:"tests"`
	Failures  int           `json:"failures"`
	Errors    []string      `json:"errors"`
	Output    string        `json:"output"`
	Timestamp time.Time     `json:"timestamp"`
}

// TestRunner orchestrates test execution
type TestRunner struct {
	config  TestConfig
	results []TestResult
}

func main() {
	var (
		suite      = flag.String("suite", "all", "Test suite to run (unit|integration|e2e|performance|security|all)")
		parallel   = flag.Bool("parallel", true, "Run tests in parallel")
		verbose    = flag.Bool("verbose", false, "Verbose output")
		timeout    = flag.Duration("timeout", 30*time.Minute, "Test timeout")
		coverage   = flag.Bool("coverage", true, "Generate coverage reports")
		configFile = flag.String("config", "", "Path to test configuration file")
		output     = flag.String("output", "test-results.json", "Output file for test results")
	)
	flag.Parse()

	runner := &TestRunner{
		config: TestConfig{
			TestSuite:   *suite,
			Parallel:    *parallel,
			Verbose:     *verbose,
			Timeout:     *timeout,
			Coverage:    *coverage,
			Environment: make(map[string]string),
		},
	}

	// Load config file if provided
	if *configFile != "" {
		if err := runner.loadConfig(*configFile); err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
	}

	// Set up environment
	runner.setupEnvironment()

	// Execute tests
	ctx, cancel := context.WithTimeout(context.Background(), runner.config.Timeout)
	defer cancel()

	if err := runner.runTests(ctx); err != nil {
		log.Fatalf("Test execution failed: %v", err)
	}

	// Generate report
	if err := runner.generateReport(*output); err != nil {
		log.Fatalf("Failed to generate report: %v", err)
	}

	// Check if any tests failed
	if runner.hasFailures() {
		os.Exit(1)
	}
}

func (r *TestRunner) loadConfig(configFile string) error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	return json.Unmarshal(data, &r.config)
}

func (r *TestRunner) setupEnvironment() {
	// Set Go environment
	os.Setenv("GO_VERSION", "1.24.7")
	os.Setenv("CGO_ENABLED", "0")

	// Set test environment
	os.Setenv("GINKGO_PARALLEL_NODE", "1")
	os.Setenv("GINKGO_PARALLEL_TOTAL", "1")

	// Apply custom environment variables
	for key, value := range r.config.Environment {
		os.Setenv(key, value)
	}

	// Verify required tools
	requiredTools := []string{"go", "kubectl", "ginkgo"}
	for _, tool := range requiredTools {
		if _, err := exec.LookPath(tool); err != nil {
			log.Printf("Warning: %s not found in PATH", tool)
		}
	}
}

func (r *TestRunner) runTests(ctx context.Context) error {
	suites := r.getTestSuites()

	for _, suite := range suites {
		if r.shouldSkipSuite(suite) {
			continue
		}

		result := TestResult{
			Suite:     suite,
			Timestamp: time.Now(),
		}

		start := time.Now()
		output, err := r.executeSuite(ctx, suite)
		result.Duration = time.Since(start)
		result.Output = output

		if err != nil {
			result.Status = "failed"
			result.Errors = []string{err.Error()}
		} else {
			result.Status = "passed"
		}

		// Parse test results for detailed metrics
		r.parseTestMetrics(&result, output)

		r.results = append(r.results, result)

		if r.config.Verbose {
			r.printSuiteResult(result)
		}
	}

	return nil
}

func (r *TestRunner) getTestSuites() []string {
	switch r.config.TestSuite {
	case "all":
		return []string{"unit", "integration", "e2e", "performance", "security"}
	case "unit":
		return []string{"unit"}
	case "integration":
		return []string{"integration"}
	case "e2e":
		return []string{"e2e"}
	case "performance":
		return []string{"performance"}
	case "security":
		return []string{"security"}
	default:
		return strings.Split(r.config.TestSuite, ",")
	}
}

func (r *TestRunner) shouldSkipSuite(suite string) bool {
	for _, pattern := range r.config.ExcludePatterns {
		if strings.Contains(suite, pattern) {
			return true
		}
	}
	return false
}

func (r *TestRunner) executeSuite(ctx context.Context, suite string) (string, error) {
	var cmd *exec.Cmd

	switch suite {
	case "unit":
		cmd = r.buildUnitTestCmd()
	case "integration":
		cmd = r.buildIntegrationTestCmd()
	case "e2e":
		cmd = r.buildE2ETestCmd()
	case "performance":
		cmd = r.buildPerformanceTestCmd()
	case "security":
		cmd = r.buildSecurityTestCmd()
	default:
		return "", fmt.Errorf("unknown test suite: %s", suite)
	}

	cmd = cmd.WithContext(ctx)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (r *TestRunner) buildUnitTestCmd() *exec.Cmd {
	args := []string{"test", "./..."}

	if r.config.Parallel {
		args = append(args, "-parallel", "4")
	}

	if r.config.Coverage {
		args = append(args, "-coverprofile=coverage.out", "-covermode=atomic")
	}

	if r.config.Verbose {
		args = append(args, "-v")
	}

	args = append(args, "-race", "-short")

	return exec.Command("go", args...)
}

func (r *TestRunner) buildIntegrationTestCmd() *exec.Cmd {
	return exec.Command("ginkgo", "-r", "./integration", "-v", "--trace")
}

func (r *TestRunner) buildE2ETestCmd() *exec.Cmd {
	return exec.Command("ginkgo", "-r", "./e2e", "-v", "--trace", "--timeout=20m")
}

func (r *TestRunner) buildPerformanceTestCmd() *exec.Cmd {
	return exec.Command("go", "test", "./performance", "-v", "-bench=.", "-benchmem")
}

func (r *TestRunner) buildSecurityTestCmd() *exec.Cmd {
	return exec.Command("go", "test", "./security", "-v", "-race")
}

func (r *TestRunner) parseTestMetrics(result *TestResult, output string) {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		// Parse test counts
		if strings.Contains(line, "PASS") || strings.Contains(line, "FAIL") {
			// Extract test metrics from output
			// This is a simplified parser - could be enhanced
		}

		// Parse coverage
		if strings.Contains(line, "coverage:") {
			// Extract coverage percentage
		}
	}
}

func (r *TestRunner) printSuiteResult(result TestResult) {
	fmt.Printf("\n=== %s Tests ===\n", strings.ToUpper(result.Suite))
	fmt.Printf("Status: %s\n", result.Status)
	fmt.Printf("Duration: %v\n", result.Duration)
	fmt.Printf("Coverage: %.1f%%\n", result.Coverage)

	if len(result.Errors) > 0 {
		fmt.Printf("Errors:\n")
		for _, err := range result.Errors {
			fmt.Printf("  - %s\n", err)
		}
	}
}

func (r *TestRunner) generateReport(outputFile string) error {
	summary := map[string]interface{}{
		"timestamp":    time.Now(),
		"total_suites": len(r.results),
		"passed":       r.countPassed(),
		"failed":       r.countFailed(),
		"duration":     r.totalDuration(),
		"results":      r.results,
	}

	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write results: %w", err)
	}

	fmt.Printf("\nTest results written to: %s\n", outputFile)
	r.printSummary()

	return nil
}

func (r *TestRunner) countPassed() int {
	count := 0
	for _, result := range r.results {
		if result.Status == "passed" {
			count++
		}
	}
	return count
}

func (r *TestRunner) countFailed() int {
	count := 0
	for _, result := range r.results {
		if result.Status == "failed" {
			count++
		}
	}
	return count
}

func (r *TestRunner) totalDuration() time.Duration {
	var total time.Duration
	for _, result := range r.results {
		total += result.Duration
	}
	return total
}

func (r *TestRunner) hasFailures() bool {
	return r.countFailed() > 0
}

func (r *TestRunner) printSummary() {
	fmt.Printf("\n=== TEST SUMMARY ===\n")
	fmt.Printf("Total Suites: %d\n", len(r.results))
	fmt.Printf("Passed: %d\n", r.countPassed())
	fmt.Printf("Failed: %d\n", r.countFailed())
	fmt.Printf("Total Duration: %v\n", r.totalDuration())

	if r.hasFailures() {
		fmt.Printf("\n❌ TESTS FAILED\n")
	} else {
		fmt.Printf("\n✅ ALL TESTS PASSED\n")
	}
}