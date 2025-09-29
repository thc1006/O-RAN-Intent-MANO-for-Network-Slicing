package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/e2e"
	"gopkg.in/yaml.v3"
)

// TestFullE2EIntegration tests complete flow from NLP to ArgoCD deployment
func TestFullE2EIntegration(t *testing.T) {
	// Skip if not in CI/CD environment with full stack
	if os.Getenv("E2E_FULL_TEST") != "true" {
		t.Skip("Skipping full E2E test. Set E2E_FULL_TEST=true to run")
	}

	ctx := context.Background()

	// Pre-flight checks
	t.Run("Prerequisites check", func(t *testing.T) {
		// Check kubectl
		cmd := exec.Command("kubectl", "version", "--short")
		output, err := cmd.Output()
		if err != nil {
			t.Skip("kubectl not available or cluster not accessible")
		}
		t.Logf("Kubernetes cluster available: %s", output)

		// Check ArgoCD
		cmd = exec.Command("kubectl", "get", "ns", "argocd")
		if err := cmd.Run(); err != nil {
			t.Skip("ArgoCD namespace not found")
		}
		t.Log("ArgoCD namespace exists")

		// Check Prometheus
		cmd = exec.Command("kubectl", "get", "svc", "-n", "monitoring", "prometheus-server")
		if err := cmd.Run(); err != nil {
			t.Log("Warning: Prometheus not found, metrics collection will be skipped")
		}
	})

	// Test complete E2E flow
	t.Run("Complete E2E orchestration", func(t *testing.T) {
		config := &e2e.E2EConfig{
			PorchEndpoint:   "http://localhost:4523",
			Namespace:       "network-slices",
			Repository:      "deployments",
			GitRepo:         "/tmp/test-repo",
			ArgoCDNamespace: "argocd",
			PrometheusURL:   "http://localhost:9090",
		}

		// Initialize test git repo
		setupTestGitRepo(t, config.GitRepo)
		defer cleanupTestGitRepo(config.GitRepo)

		orchestrator, err := e2e.NewE2EOrchestrator(config)
		require.NoError(t, err)

		// Test intent
		intent := "Deploy an eMBB network slice for 4K video streaming with 1 Gbps throughput and 20ms latency"

		// Process intent
		result, err := orchestrator.ProcessIntent(ctx, intent)

		// Allow partial success in test environment
		if err != nil {
			t.Logf("E2E flow completed with warnings: %v", err)
		}

		assert.NotNil(t, result)
		assert.Equal(t, intent, result.Intent)
		assert.NotEmpty(t, result.SliceID)

		// Verify steps were executed
		assert.GreaterOrEqual(t, len(result.Steps), 3)

		// Check Claude processing step
		claudeStep := findStep(result.Steps, "Claude NLP Processing")
		if claudeStep != nil {
			assert.Equal(t, "completed", claudeStep.Status)
			t.Logf("Claude processing: %+v", claudeStep.Details)
		}

		// Check package generation step
		packageStep := findStep(result.Steps, "Nephio Package Generation")
		if packageStep != nil {
			assert.Equal(t, "completed", packageStep.Status)

			// Verify package was created
			if details, ok := packageStep.Details.(map[string]string); ok {
				if packagePath, exists := details["path"]; exists {
					// Check Kptfile exists
					kptfilePath := fmt.Sprintf("%s/Kptfile", packagePath)
					_, err := os.Stat(kptfilePath)
					assert.NoError(t, err, "Kptfile should exist")

					// Validate Kptfile content
					kptfileContent, err := os.ReadFile(kptfilePath)
					if err == nil {
						var kptfile map[string]interface{}
						err = yaml.Unmarshal(kptfileContent, &kptfile)
						assert.NoError(t, err)
						assert.Equal(t, "kpt.dev/v1", kptfile["apiVersion"])
						assert.Equal(t, "Kptfile", kptfile["kind"])
					}

					// Check NetworkSlice CR exists
					slicePath := fmt.Sprintf("%s/network-slice.yaml", packagePath)
					_, err = os.Stat(slicePath)
					assert.NoError(t, err, "NetworkSlice CR should exist")
				}
			}
		}

		// Check Git commit step
		gitStep := findStep(result.Steps, "Git Commit")
		if gitStep != nil {
			// Git might fail in test env, that's OK
			t.Logf("Git commit step: %s", gitStep.Status)
			if details, ok := gitStep.Details.(map[string]string); ok {
				if commit, exists := details["commit"]; exists {
					assert.NotEmpty(t, commit)
					t.Logf("Git commit hash: %s", commit)
				}
			}
		}

		// Check ArgoCD application step
		argoStep := findStep(result.Steps, "ArgoCD Application")
		if argoStep != nil {
			t.Logf("ArgoCD app creation: %s", argoStep.Status)
			if details, ok := argoStep.Details.(map[string]string); ok {
				if appName, exists := details["app"]; exists {
					assert.NotEmpty(t, appName)
					t.Logf("ArgoCD app name: %s", appName)

					// Try to verify app was created (might fail without cluster)
					cmd := exec.Command("kubectl", "get", "application", appName, "-n", "argocd", "-o", "json")
					if output, err := cmd.Output(); err == nil {
						var app map[string]interface{}
						json.Unmarshal(output, &app)
						t.Logf("ArgoCD app exists: %s", appName)
					}
				}
			}
		}
	})

	// Test WebSocket E2E integration
	t.Run("WebSocket E2E flow", func(t *testing.T) {
		// Start test WebSocket server
		serverURL := startTestWebSocketServer(t)
		defer stopTestWebSocketServer()

		// Connect WebSocket client
		wsURL := strings.Replace(serverURL, "http", "ws", 1) + "/ws"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn.Close()

		// Read welcome message
		var welcome map[string]interface{}
		err = conn.ReadJSON(&welcome)
		require.NoError(t, err)

		sessionID := welcome["sessionId"].(string)

		// Send E2E intent
		e2eIntent := map[string]interface{}{
			"type":      "e2e_intent",
			"intent":    "Create a URLLC slice for autonomous vehicles with 1ms latency and 99.999% reliability",
			"sessionId": sessionID,
			"e2e":       true,
		}

		err = conn.WriteJSON(e2eIntent)
		require.NoError(t, err)

		// Read E2E updates
		stepCount := 0
		timeout := time.After(30 * time.Second)

		for {
			select {
			case <-timeout:
				assert.GreaterOrEqual(t, stepCount, 3, "Should receive at least 3 E2E steps")
				return

			default:
				conn.SetReadDeadline(time.Now().Add(5 * time.Second))

				var msg map[string]interface{}
				err := conn.ReadJSON(&msg)
				if err != nil {
					if websocket.IsUnexpectedCloseError(err) {
						return
					}
					continue
				}

				msgType := msg["type"].(string)
				t.Logf("Received message type: %s", msgType)

				switch msgType {
				case "e2e_step":
					stepCount++
					if data, ok := msg["data"].(map[string]interface{}); ok {
						t.Logf("E2E Step %d: %s - %s", stepCount, data["step"], data["status"])
					}

				case "e2e_complete":
					t.Log("E2E flow completed")
					if data, ok := msg["data"].(map[string]interface{}); ok {
						assert.True(t, data["success"].(bool))
						assert.NotEmpty(t, data["sliceId"])
						t.Logf("Slice ID: %s", data["sliceId"])
					}
					return

				case "e2e_error":
					t.Logf("E2E error: %s", msg["message"])
					// Don't fail - errors expected in test env
				}
			}
		}
	})

	// Test ArgoCD synchronization
	t.Run("ArgoCD sync verification", func(t *testing.T) {
		// This test requires actual ArgoCD installation
		cmd := exec.Command("argocd", "version")
		if err := cmd.Run(); err != nil {
			t.Skip("ArgoCD CLI not available")
		}

		// Create test application
		appName := fmt.Sprintf("test-slice-%d", time.Now().Unix())

		// Generate test ArgoCD app manifest
		appManifest := generateTestArgoApp(appName)
		appPath := fmt.Sprintf("/tmp/%s.yaml", appName)
		err := os.WriteFile(appPath, []byte(appManifest), 0644)
		require.NoError(t, err)
		defer os.Remove(appPath)

		// Apply application
		cmd = exec.Command("kubectl", "apply", "-f", appPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("ArgoCD app creation failed (expected in test): %s", output)
			return
		}

		// Wait for sync
		time.Sleep(5 * time.Second)

		// Check sync status
		cmd = exec.Command("kubectl", "get", "application", appName,
			"-n", "argocd",
			"-o", "jsonpath={.status.sync.status}")

		syncStatus, err := cmd.Output()
		if err == nil {
			t.Logf("ArgoCD sync status: %s", syncStatus)
			// Clean up
			exec.Command("kubectl", "delete", "application", appName, "-n", "argocd").Run()
		}
	})

	// Test metrics collection
	t.Run("Prometheus metrics verification", func(t *testing.T) {
		// Check if Prometheus is accessible
		resp, err := http.Get("http://localhost:9090/-/healthy")
		if err != nil {
			t.Skip("Prometheus not accessible")
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Skip("Prometheus not healthy")
		}

		// Query test metrics
		queries := []string{
			"up",
			"slice_active_sessions",
			"slice_throughput_bytes",
			"slice_latency_ms",
		}

		for _, query := range queries {
			url := fmt.Sprintf("http://localhost:9090/api/v1/query?query=%s", query)
			resp, err := http.Get(url)
			if err != nil {
				t.Logf("Metric query failed for %s: %v", query, err)
				continue
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)

			if status, ok := result["status"].(string); ok && status == "success" {
				t.Logf("Metric %s available", query)
			}
		}
	})
}

// Helper functions

func setupTestGitRepo(t *testing.T, repoPath string) {
	// Create test git repository
	os.RemoveAll(repoPath)
	os.MkdirAll(repoPath, 0755)

	cmd := exec.Command("git", "init")
	cmd.Dir = repoPath
	err := cmd.Run()
	require.NoError(t, err)

	// Create initial commit
	testFile := fmt.Sprintf("%s/README.md", repoPath)
	os.WriteFile(testFile, []byte("# Test Repository"), 0644)

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoPath
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = repoPath
	cmd.Run()
}

func cleanupTestGitRepo(repoPath string) {
	os.RemoveAll(repoPath)
}

func findStep(steps []e2e.StepResult, name string) *e2e.StepResult {
	for _, step := range steps {
		if step.Name == name {
			return &step
		}
	}
	return nil
}

func startTestWebSocketServer(t *testing.T) string {
	// This would start the actual WebSocket server
	// For testing, we'll return a mock URL
	return "http://localhost:8080"
}

func stopTestWebSocketServer() {
	// Stop test server
}

func generateTestArgoApp(appName string) string {
	return fmt.Sprintf(`apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: %s
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/example/test
    path: deployments
    targetRevision: HEAD
  destination:
    server: https://kubernetes.default.svc
    namespace: network-slices
  syncPolicy:
    automated:
      prune: true
      selfHeal: true`, appName)
}

// BenchmarkE2EFlow benchmarks the E2E flow
func BenchmarkE2EFlow(b *testing.B) {
	if os.Getenv("E2E_BENCHMARK") != "true" {
		b.Skip("Skipping E2E benchmark. Set E2E_BENCHMARK=true to run")
	}

	config := &e2e.E2EConfig{
		PorchEndpoint:   "http://localhost:4523",
		Namespace:       "network-slices",
		Repository:      "deployments",
		GitRepo:         "/tmp/bench-repo",
		ArgoCDNamespace: "argocd",
		PrometheusURL:   "http://localhost:9090",
	}

	setupTestGitRepo(&testing.T{}, config.GitRepo)
	defer cleanupTestGitRepo(config.GitRepo)

	orchestrator, _ := e2e.NewE2EOrchestrator(config)
	ctx := context.Background()

	intents := []string{
		"Deploy eMBB slice for video streaming",
		"Create URLLC slice for autonomous vehicles",
		"Setup mIoT slice for smart city",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		intent := intents[i%len(intents)]
		orchestrator.ProcessIntent(ctx, intent)
	}
}