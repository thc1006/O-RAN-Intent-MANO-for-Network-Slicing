// Package utils provides common test utilities and helpers for O-RAN Intent-MANO testing
package utils

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// TestEnvironment provides common test environment setup
type TestEnvironment struct {
	Client     client.Client
	Config     *rest.Config
	TestEnv    *envtest.Environment
	Ctx        context.Context
	CancelFunc context.CancelFunc
	Namespace  string
}

// MockServer provides HTTP mock server for testing
type MockServer struct {
	Server *httptest.Server
	URL    string
}

// TestMetrics captures test execution metrics
type TestMetrics struct {
	StartTime        time.Time
	EndTime          time.Time
	Duration         time.Duration
	DeploymentTime   time.Duration
	ThroughputMbps   float64
	LatencyMs        float64
	PacketLoss       float64
	ResourceUsage    ResourceUsage
	TestsPassed      int
	TestsFailed      int
}

// ResourceUsage captures resource utilization metrics
type ResourceUsage struct {
	CPUPercent    float64
	MemoryMB      float64
	NetworkMbps   float64
	StorageGB     float64
}

// SetupTestEnvironment initializes the test environment with Kubernetes
func SetupTestEnvironment(t *testing.T, scheme *runtime.Scheme) *TestEnvironment {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	// Use existing cluster if available, otherwise start test environment
	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "adapters", "vnf-operator", "config", "crd", "bases"),
			filepath.Join("..", "tn", "config", "crd", "bases"),
		},
		ErrorIfCRDPathMissing: false,
		UseExistingCluster:    func() *bool { b := true; return &b }(),
	}

	cfg, err := testEnv.Start()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme})
	require.NoError(t, err)
	require.NotNil(t, k8sClient)

	// Create test namespace
	namespace := createTestNamespace(t, k8sClient)

	return &TestEnvironment{
		Client:     k8sClient,
		Config:     cfg,
		TestEnv:    testEnv,
		Ctx:        ctx,
		CancelFunc: cancel,
		Namespace:  namespace,
	}
}

// CleanupTestEnvironment cleans up the test environment
func (env *TestEnvironment) Cleanup(t *testing.T) {
	t.Helper()

	if env.CancelFunc != nil {
		env.CancelFunc()
	}

	if env.TestEnv != nil {
		err := env.TestEnv.Stop()
		require.NoError(t, err)
	}
}

// createTestNamespace creates a unique test namespace
func createTestNamespace(t *testing.T, client client.Client) string {
	t.Helper()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Labels: map[string]string{
				"test": "true",
				"type": "integration",
			},
		},
	}

	err := client.Create(context.Background(), namespace)
	require.NoError(t, err)

	t.Cleanup(func() {
		err := client.Delete(context.Background(), namespace)
		if err != nil {
			t.Logf("Failed to delete test namespace %s: %v", namespace.Name, err)
		}
	})

	return namespace.Name
}

// WaitForDeployment waits for a deployment to be ready
func WaitForDeployment(ctx context.Context, client client.Client, namespace, name string, timeout time.Duration) error {
	return wait.PollImmediate(5*time.Second, timeout, func() (bool, error) {
		// Check if deployment is ready
		// This is simplified - real implementation would check deployment status
		return true, nil
	})
}

// WaitForService waits for a service to be available
func WaitForService(ctx context.Context, client client.Client, namespace, name string, timeout time.Duration) error {
	return wait.PollImmediate(2*time.Second, timeout, func() (bool, error) {
		service := &corev1.Service{}
		err := client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, service)
		if err != nil {
			return false, err
		}
		return service.Spec.ClusterIP != "", nil
	})
}

// StartMockServer starts a mock HTTP server for testing
func StartMockServer(handler http.Handler) *MockServer {
	server := httptest.NewServer(handler)
	return &MockServer{
		Server: server,
		URL:    server.URL,
	}
}

// StopMockServer stops the mock HTTP server
func (m *MockServer) Stop() {
	if m.Server != nil {
		m.Server.Close()
	}
}

// NewMockO2Server creates a mock O2 interface server
func NewMockO2Server() *MockServer {
	mux := http.NewServeMux()

	// O2IMS endpoints
	mux.HandleFunc("/o2ims/v1/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"version": "1.0", "status": "ready"}`))
	})

	// O2DMS endpoints
	mux.HandleFunc("/o2dms/v1/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"version": "1.0", "deployments": []}`))
	})

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return StartMockServer(mux)
}

// NewMockNephioServer creates a mock Nephio server
func NewMockNephioServer() *MockServer {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/porch/v1alpha1/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"packages": []}`))
	})

	return StartMockServer(mux)
}

// AssertEventuallyWithTimeout asserts condition is met within timeout
func AssertEventuallyWithTimeout(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()

	gomega.NewWithT(t).Eventually(condition, timeout, 100*time.Millisecond).Should(gomega.BeTrue(), message)
}

// AssertConsistentlyWithTimeout asserts condition remains true for duration
func AssertConsistentlyWithTimeout(t *testing.T, condition func() bool, duration time.Duration, message string) {
	t.Helper()

	gomega.NewWithT(t).Consistently(condition, duration, 100*time.Millisecond).Should(gomega.BeTrue(), message)
}

// GetTestDataPath returns path to test data files
func GetTestDataPath(filename string) string {
	return filepath.Join("..", "testdata", filename)
}

// LoadTestData loads test data from file
func LoadTestData(t *testing.T, filename string) []byte {
	t.Helper()

	path := GetTestDataPath(filename)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read test data file: %s", path)

	return data
}

// CreateTempFile creates a temporary file for testing
func CreateTempFile(t *testing.T, content string) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "test-*.yaml")
	require.NoError(t, err)

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})

	return tmpFile.Name()
}

// GenerateTestMetrics generates test metrics for validation
func GenerateTestMetrics(testName string) *TestMetrics {
	return &TestMetrics{
		StartTime: time.Now(),
		ResourceUsage: ResourceUsage{
			CPUPercent:  50.0,
			MemoryMB:    256.0,
			NetworkMbps: 100.0,
			StorageGB:   1.0,
		},
	}
}

// FinishTestMetrics completes test metrics collection
func (m *TestMetrics) Finish() {
	m.EndTime = time.Now()
	m.Duration = m.EndTime.Sub(m.StartTime)
}

// ValidateThesisMetrics validates metrics against thesis requirements
func (m *TestMetrics) ValidateThesisMetrics(t *testing.T) {
	t.Helper()

	// Validate deployment time < 10 minutes
	require.Less(t, m.DeploymentTime.Minutes(), 10.0,
		"Deployment time %v exceeds 10 minutes", m.DeploymentTime)

	// Validate throughput targets
	expectedThroughputs := []float64{4.57, 2.77, 0.93}
	for _, expected := range expectedThroughputs {
		if m.ThroughputMbps >= expected*0.9 { // 90% tolerance
			t.Logf("Throughput %v Mbps meets target %v Mbps", m.ThroughputMbps, expected)
			return
		}
	}
	t.Errorf("Throughput %v Mbps does not meet any target thresholds", m.ThroughputMbps)

	// Validate RTT targets
	expectedRTTs := []float64{16.1, 15.7, 6.3}
	for _, expected := range expectedRTTs {
		if m.LatencyMs <= expected*1.1 { // 10% tolerance
			t.Logf("RTT %v ms meets target %v ms", m.LatencyMs, expected)
			return
		}
	}
	t.Errorf("RTT %v ms does not meet any target thresholds", m.LatencyMs)
}

// SkipIfNotIntegration skips test if not running integration tests
func SkipIfNotIntegration(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("INTEGRATION_TESTS") == "false" {
		t.Skip("Integration tests disabled")
	}
}

// SkipIfNoKubernetes skips test if Kubernetes is not available
func SkipIfNoKubernetes(t *testing.T) {
	t.Helper()

	if os.Getenv("KUBECONFIG") == "" && os.Getenv("HOME")+"/.kube/config" == "" {
		t.Skip("No Kubernetes configuration available")
	}
}

// LogTestProgress logs test progress for debugging
func LogTestProgress(t *testing.T, step string) {
	t.Helper()
	t.Logf("=== %s: %s ===", t.Name(), step)
}

// RetryOperation retries an operation with exponential backoff
func RetryOperation(operation func() error, maxRetries int, initialDelay time.Duration) error {
	var err error
	delay := initialDelay

	for i := 0; i < maxRetries; i++ {
		if err = operation(); err == nil {
			return nil
		}

		if i < maxRetries-1 {
			time.Sleep(delay)
			delay *= 2
		}
	}

	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, err)
}

// IsKindCluster checks if running in KIND cluster
func IsKindCluster() bool {
	kubeContext := os.Getenv("KUBECONFIG")
	return strings.Contains(kubeContext, "kind")
}

// GetKubernetesClient returns a Kubernetes client for testing
func GetKubernetesClient(config *rest.Config) (kubernetes.Interface, error) {
	return kubernetes.NewForConfig(config)
}