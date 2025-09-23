// Package testutils provides common testing utilities for the O-RAN Intent-MANO system
package testutils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	TestTimeout          = 30 * time.Second
	TestPollingInterval  = 250 * time.Millisecond
	TestEventualTimeout  = 60 * time.Second
)

// TestSuite represents a comprehensive test suite configuration
type TestSuite struct {
	Name        string
	Description string
	Categories  []TestCategory
	Timeout     time.Duration
	Parallel    bool
	Setup       func() error
	Teardown    func() error
	Config      *TestConfig
}

// TestCategory defines different types of tests
type TestCategory string

const (
	UnitTest        TestCategory = "unit"
	IntegrationTest TestCategory = "integration"
	E2ETest         TestCategory = "e2e"
	PerformanceTest TestCategory = "performance"
	ContractTest    TestCategory = "contract"
	ChaosTest       TestCategory = "chaos"
	SecurityTest    TestCategory = "security"
)

// TestConfig holds configuration for test execution
type TestConfig struct {
	KubeConfig         *rest.Config
	KubeClient         client.Client
	KubernetesClient   kubernetes.Interface
	TestEnv            *envtest.Environment
	Context            context.Context
	CancelFunc         context.CancelFunc
	TestDataDir        string
	TempDir            string
	LogLevel           string
	MockServices       map[string]interface{}
	TestMetrics        *TestMetrics
	ParallelNodes      int
	EnableCoverage     bool
	CoverageThreshold  float64
}

// TestMetrics tracks test execution metrics
type TestMetrics struct {
	StartTime         time.Time
	EndTime           time.Time
	Duration          time.Duration
	TestsRun          int
	TestsPassed       int
	TestsFailed       int
	TestsSkipped      int
	CoveragePercent   float64
	PerformanceData   map[string]interface{}
	ResourceUsage     map[string]interface{}
}

// TestFramework provides a comprehensive testing framework
type TestFramework struct {
	Config   *TestConfig
	Suites   map[string]*TestSuite
	Reporter *TestReporter
}

// NewTestFramework creates a new test framework instance
func NewTestFramework(config *TestConfig) *TestFramework {
	if config == nil {
		config = &TestConfig{
			LogLevel:          "info",
			ParallelNodes:     4,
			EnableCoverage:    true,
			CoverageThreshold: 90.0,
		}
	}

	return &TestFramework{
		Config:   config,
		Suites:   make(map[string]*TestSuite),
		Reporter: NewTestReporter(),
	}
}

// SetupTestEnvironment initializes the test environment
func (tf *TestFramework) SetupTestEnvironment() error {
	// Set up logging
	logf.SetLogger(zap.New(zap.WriteTo(ginkgo.GinkgoWriter), zap.UseDevMode(true)))

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	tf.Config.Context = ctx
	tf.Config.CancelFunc = cancel

	// Set up test directories
	tempDir, err := os.MkdirTemp("", "oran-mano-test-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	tf.Config.TempDir = tempDir

	// Set up test data directory
	testDataDir := filepath.Join(".", "testdata")
	if _, err := os.Stat(testDataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(testDataDir, security.SecureDirMode); err != nil {
			return fmt.Errorf("failed to create test data directory: %w", err)
		}
	}
	tf.Config.TestDataDir = testDataDir

	// Initialize test metrics
	tf.Config.TestMetrics = &TestMetrics{
		StartTime:       time.Now(),
		PerformanceData: make(map[string]interface{}),
		ResourceUsage:   make(map[string]interface{}),
	}

	// Set up mock services
	tf.Config.MockServices = make(map[string]interface{})

	return nil
}

// SetupKubernetesTestEnvironment sets up a Kubernetes test environment using envtest
func (tf *TestFramework) SetupKubernetesTestEnvironment() error {
	tf.Config.TestEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "adapters", "vnf-operator", "config", "crd", "bases"),
			filepath.Join("..", "..", "tn", "manager", "config", "crd", "bases"),
		},
		ErrorIfCRDPathMissing: false,
	}

	cfg, err := tf.Config.TestEnv.Start()
	if err != nil {
		return fmt.Errorf("failed to start test environment: %w", err)
	}

	tf.Config.KubeConfig = cfg

	// Create Kubernetes clients
	scheme := runtime.NewScheme()
	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}
	tf.Config.KubeClient = k8sClient

	kubernetesClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}
	tf.Config.KubernetesClient = kubernetesClient

	return nil
}

// TeardownTestEnvironment cleans up the test environment
func (tf *TestFramework) TeardownTestEnvironment() error {
	// Cancel context
	if tf.Config.CancelFunc != nil {
		tf.Config.CancelFunc()
	}

	// Stop test environment
	if tf.Config.TestEnv != nil {
		if err := tf.Config.TestEnv.Stop(); err != nil {
			return fmt.Errorf("failed to stop test environment: %w", err)
		}
	}

	// Clean up temp directory
	if tf.Config.TempDir != "" {
		if err := os.RemoveAll(tf.Config.TempDir); err != nil {
			return fmt.Errorf("failed to remove temp directory: %w", err)
		}
	}

	// Finalize metrics
	tf.Config.TestMetrics.EndTime = time.Now()
	tf.Config.TestMetrics.Duration = tf.Config.TestMetrics.EndTime.Sub(tf.Config.TestMetrics.StartTime)

	return nil
}

// RegisterTestSuite registers a test suite with the framework
func (tf *TestFramework) RegisterTestSuite(suite *TestSuite) {
	tf.Suites[suite.Name] = suite
}

// RunTestSuite executes a specific test suite
func (tf *TestFramework) RunTestSuite(suiteName string, t *testing.T) error {
	suite, exists := tf.Suites[suiteName]
	if !exists {
		return fmt.Errorf("test suite %s not found", suiteName)
	}

	// Setup
	if suite.Setup != nil {
		if err := suite.Setup(); err != nil {
			return fmt.Errorf("suite setup failed: %w", err)
		}
	}

	// Run tests
	if suite.Parallel {
		t.Parallel()
	}

	// Configure timeout
	timeout := suite.Timeout
	if timeout == 0 {
		timeout = TestTimeout
	}

	ctx, cancel := context.WithTimeout(tf.Config.Context, timeout)
	defer cancel()

	// Execute test categories
	for _, category := range suite.Categories {
		if err := tf.runTestCategory(ctx, category, suite); err != nil {
			return fmt.Errorf("failed to run %s tests: %w", category, err)
		}
	}

	// Teardown
	if suite.Teardown != nil {
		if err := suite.Teardown(); err != nil {
			return fmt.Errorf("suite teardown failed: %w", err)
		}
	}

	return nil
}

// runTestCategory executes tests for a specific category
func (tf *TestFramework) runTestCategory(ctx context.Context, category TestCategory, suite *TestSuite) error {
	tf.Reporter.ReportTestStart(string(category))

	switch category {
	case UnitTest:
		return tf.runUnitTests(ctx, suite)
	case IntegrationTest:
		return tf.runIntegrationTests(ctx, suite)
	case E2ETest:
		return tf.runE2ETests(ctx, suite)
	case PerformanceTest:
		return tf.runPerformanceTests(ctx, suite)
	case ContractTest:
		return tf.runContractTests(ctx, suite)
	case ChaosTest:
		return tf.runChaosTests(ctx, suite)
	case SecurityTest:
		return tf.runSecurityTests(ctx, suite)
	default:
		return fmt.Errorf("unknown test category: %s", category)
	}
}

// Helper methods for different test types
func (tf *TestFramework) runUnitTests(ctx context.Context, suite *TestSuite) error {
	// Implementation for unit tests
	return nil
}

func (tf *TestFramework) runIntegrationTests(ctx context.Context, suite *TestSuite) error {
	// Implementation for integration tests
	return nil
}

func (tf *TestFramework) runE2ETests(ctx context.Context, suite *TestSuite) error {
	// Implementation for E2E tests
	return nil
}

func (tf *TestFramework) runPerformanceTests(ctx context.Context, suite *TestSuite) error {
	// Implementation for performance tests
	return nil
}

func (tf *TestFramework) runContractTests(ctx context.Context, suite *TestSuite) error {
	// Implementation for contract tests
	return nil
}

func (tf *TestFramework) runChaosTests(ctx context.Context, suite *TestSuite) error {
	// Implementation for chaos tests
	return nil
}

func (tf *TestFramework) runSecurityTests(ctx context.Context, suite *TestSuite) error {
	// Implementation for security tests
	return nil
}

// WaitForCondition waits for a condition to be met within a timeout
func (tf *TestFramework) WaitForCondition(condition func() bool, timeout time.Duration, interval time.Duration) error {
	ctx, cancel := context.WithTimeout(tf.Config.Context, timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for condition")
		case <-ticker.C:
			if condition() {
				return nil
			}
		}
	}
}

// AssertExpectations sets up Gomega expectations for tests
func (tf *TestFramework) AssertExpectations() {
	gomega.RegisterFailHandler(ginkgo.Fail)
	gomega.SetDefaultEventuallyTimeout(TestEventualTimeout)
	gomega.SetDefaultEventuallyPollingInterval(TestPollingInterval)
}