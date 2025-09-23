package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLogger provides a test logger implementation
type TestLogger struct {
	messages []string
}

func (tl *TestLogger) Write(p []byte) (n int, err error) {
	tl.messages = append(tl.messages, string(p))
	return len(p), nil
}

func (tl *TestLogger) GetMessages() []string {
	return tl.messages
}

// Test setup helper
func setupTestAgent() *TNAgent {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	config := &TNConfig{
		ClusterName: "test-cluster",
		NetworkCIDR: "10.0.0.0/16",
		VXLANConfig: VXLANConfig{
			VNI:        100,
			RemoteIPs:  []string{"192.168.1.1"},
			LocalIP:    "192.168.1.2",
			Port:       4789,
			MTU:        1450,
			DeviceName: "vxlan0",
			Learning:   true,
		},
		BWPolicy: BandwidthPolicy{
			DownlinkMbps: 1000.0,
			UplinkMbps:   1000.0,
			LatencyMs:    10.0,
			JitterMs:     1.0,
			LossPercent:  0.1,
			Priority:     1,
			QueueClass:   "root",
		},
		QoSClass:       "besteffort",
		MonitoringPort: 8080,
	}

	agent := &TNAgent{
		config: config,
		logger: logger,
	}

	return agent
}

func TestHTTPHandlers_HealthCheck(t *testing.T) {
	agent := setupTestAgent()

	t.Run("successful_health_check", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		agent.handleHealth(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "healthy")
	})

	t.Run("health_check_with_malformed_request", func(t *testing.T) {
		// Test with invalid method
		req := httptest.NewRequest("POST", "/health", nil)
		w := httptest.NewRecorder()

		agent.handleHealth(w, req)

		// Should still respond (health checks should be permissive)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHTTPHandlers_Status(t *testing.T) {
	agent := setupTestAgent()

	t.Run("successful_status_request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/status", nil)
		w := httptest.NewRecorder()

		agent.handleStatus(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "status")
	})

	t.Run("status_with_invalid_method", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/status", nil)
		w := httptest.NewRecorder()

		agent.handleStatus(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestHTTPHandlers_Configuration(t *testing.T) {
	agent := setupTestAgent()

	t.Run("successful_config_update", func(t *testing.T) {
		config := TNConfig{
			ClusterName: "updated-cluster",
			NetworkCIDR: "10.1.0.0/16",
			VXLANConfig: VXLANConfig{
				VNI:        200,
				RemoteIPs:  []string{"192.168.2.1"},
				LocalIP:    "192.168.2.2",
				Port:       4789,
				MTU:        1450,
				DeviceName: "vxlan1",
				Learning:   true,
			},
			QoSClass:       "guaranteed",
			MonitoringPort: 8080,
		}

		configJSON, err := json.Marshal(config)
		require.NoError(t, err)

		req := httptest.NewRequest("PUT", "/config", bytes.NewBuffer(configJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		agent.handleUpdateConfig(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("malformed_json_config", func(t *testing.T) {
		malformedJSON := `{"clusterName": "test", "invalidField": `

		req := httptest.NewRequest("PUT", "/config", strings.NewReader(malformedJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		agent.handleUpdateConfig(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "error")
	})

	t.Run("empty_request_body", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/config", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		agent.handleUpdateConfig(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid_content_type", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/config", strings.NewReader("invalid"))
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()

		agent.handleUpdateConfig(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("get_current_config", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/config", nil)
		w := httptest.NewRecorder()

		agent.handleGetConfig(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response TNConfig
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "test-cluster", response.ClusterName)
	})
}

func TestHTTPHandlers_SliceManagement(t *testing.T) {
	agent := setupTestAgent()

	t.Run("create_slice_success", func(t *testing.T) {
		sliceConfig := map[string]interface{}{
			"id":          "slice-1",
			"qosClass":    "guaranteed",
			"bandwidth":   "100Mbps",
		}

		configJSON, err := json.Marshal(sliceConfig)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/slices", bytes.NewBuffer(configJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		agent.handleConfigureSlice(w, req)

		// Depending on implementation, this might be 201 or 200
		assert.True(t, w.Code == http.StatusCreated || w.Code == http.StatusOK)
	})

	t.Run("create_slice_invalid_json", func(t *testing.T) {
		invalidJSON := `{"id": "slice-1", "invalid"`

		req := httptest.NewRequest("POST", "/slices", strings.NewReader(invalidJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		agent.handleConfigureSlice(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("get_slices_list", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/slices", nil)
		w := httptest.NewRecorder()

		agent.handleConfigureSlice(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHTTPHandlers_PerformanceTesting(t *testing.T) {
	agent := setupTestAgent()

	t.Run("start_performance_test_success", func(t *testing.T) {
		testConfig := map[string]interface{}{
			"duration":   "30s",
			"bandwidth":  "100M",
			"protocol":   "tcp",
			"target":     "192.168.1.100",
		}

		configJSON, err := json.Marshal(testConfig)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/performance", bytes.NewBuffer(configJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		agent.handleRunTest(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusAccepted)
	})

	t.Run("performance_test_invalid_config", func(t *testing.T) {
		invalidConfig := map[string]interface{}{
			"duration": -1,  // Invalid duration
		}

		configJSON, err := json.Marshal(invalidConfig)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/performance", bytes.NewBuffer(configJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		agent.handleRunTest(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("get_performance_results", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/performance", nil)
		w := httptest.NewRecorder()

		agent.handleRunTest(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHTTPHandlers_ErrorHandling(t *testing.T) {
	agent := setupTestAgent()

	t.Run("large_request_body", func(t *testing.T) {
		// Create a large request body (>1MB)
		largeData := strings.Repeat("x", 2*1024*1024)

		req := httptest.NewRequest("PUT", "/config", strings.NewReader(largeData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		agent.handleUpdateConfig(w, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	})

	t.Run("unsupported_endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/unsupported", nil)
		w := httptest.NewRecorder()

		// Create a router to test 404 handling
		router := mux.NewRouter()
		// Add a simple route to test 404
		router.HandleFunc("/status", agent.handleStatus).Methods("GET")

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("request_timeout", func(t *testing.T) {
		// Test with a request that should timeout
		req := httptest.NewRequest("GET", "/status", nil)
		w := httptest.NewRecorder()

		// Simulate timeout by setting a very short deadline
		ctx := req.Context()
		ctx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
		defer cancel()
		req = req.WithContext(ctx)

		time.Sleep(2 * time.Millisecond) // Ensure timeout

		agent.handleStatus(w, req)

		// The handler should detect the timeout
		assert.True(t, w.Code >= 400)
	})
}

func TestHTTPHandlers_SecurityValidation(t *testing.T) {
	agent := setupTestAgent()

	t.Run("sql_injection_attempt", func(t *testing.T) {
		maliciousConfig := map[string]interface{}{
			"clusterName": "'; DROP TABLE users; --",
			"networkCIDR": "10.0.0.0/16",
		}

		configJSON, err := json.Marshal(maliciousConfig)
		require.NoError(t, err)

		req := httptest.NewRequest("PUT", "/config", bytes.NewBuffer(configJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		agent.handleUpdateConfig(w, req)

		// Should reject malicious input
		assert.True(t, w.Code >= 400)
	})

	t.Run("xss_attempt", func(t *testing.T) {
		maliciousConfig := map[string]interface{}{
			"clusterName": "<script>alert('xss')</script>",
			"networkCIDR": "10.0.0.0/16",
		}

		configJSON, err := json.Marshal(maliciousConfig)
		require.NoError(t, err)

		req := httptest.NewRequest("PUT", "/config", bytes.NewBuffer(configJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		agent.handleUpdateConfig(w, req)

		// Should handle malicious input safely
		assert.True(t, w.Code >= 400)
	})

	t.Run("command_injection_attempt", func(t *testing.T) {
		maliciousConfig := map[string]interface{}{
			"clusterName": "test && rm -rf /",
			"networkCIDR": "10.0.0.0/16",
		}

		configJSON, err := json.Marshal(maliciousConfig)
		require.NoError(t, err)

		req := httptest.NewRequest("PUT", "/config", bytes.NewBuffer(configJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		agent.handleUpdateConfig(w, req)

		// Should reject command injection attempts
		assert.True(t, w.Code >= 400)
	})
}

func TestHTTPHandlers_ConcurrentRequests(t *testing.T) {
	agent := setupTestAgent()

	t.Run("concurrent_status_requests", func(t *testing.T) {
		const numRequests = 50
		responses := make(chan int, numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				req := httptest.NewRequest("GET", "/status", nil)
				w := httptest.NewRecorder()
				agent.handleStatus(w, req)
				responses <- w.Code
			}()
		}

		// Collect all responses
		for i := 0; i < numRequests; i++ {
			select {
			case code := <-responses:
				assert.Equal(t, http.StatusOK, code)
			case <-time.After(5 * time.Second):
				t.Fatal("Timeout waiting for response")
			}
		}
	})

	t.Run("concurrent_config_updates", func(t *testing.T) {
		const numRequests = 10
		responses := make(chan int, numRequests)

		for i := 0; i < numRequests; i++ {
			go func(id int) {
				config := TNConfig{
					ClusterName: fmt.Sprintf("cluster-%d", id),
					NetworkCIDR: "10.0.0.0/16",
					VXLANConfig: VXLANConfig{
						VNI:        uint32(100 + id),
						RemoteIPs:  []string{"192.168.1.1"},
						LocalIP:    "192.168.1.2",
						Port:       4789,
						MTU:        1450,
						DeviceName: fmt.Sprintf("vxlan%d", id),
						Learning:   true,
					},
					QoSClass:       "besteffort",
					MonitoringPort: 8080,
				}

				configJSON, _ := json.Marshal(config)
				req := httptest.NewRequest("PUT", "/config", bytes.NewBuffer(configJSON))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				agent.handleUpdateConfig(w, req)
				responses <- w.Code
			}(i)
		}

		// Collect all responses
		successCount := 0
		for i := 0; i < numRequests; i++ {
			select {
			case code := <-responses:
				if code == http.StatusOK {
					successCount++
				}
			case <-time.After(10 * time.Second):
				t.Fatal("Timeout waiting for response")
			}
		}

		// At least some requests should succeed
		assert.True(t, successCount > 0)
	})
}

// Benchmark tests
func BenchmarkHTTPHandlers_StatusRequest(b *testing.B) {
	agent := setupTestAgent()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/status", nil)
		w := httptest.NewRecorder()
		agent.handleStatus(w, req)
	}
}

func BenchmarkHTTPHandlers_ConfigUpdate(b *testing.B) {
	agent := setupTestAgent()

	config := TNConfig{
		ClusterName: "benchmark-cluster",
		NetworkCIDR: "10.0.0.0/16",
		VXLANConfig: VXLANConfig{
			VNI:        100,
			RemoteIPs:  []string{"192.168.1.1"},
			LocalIP:    "192.168.1.2",
			Port:       4789,
			MTU:        1450,
			DeviceName: "vxlan0",
			Learning:   true,
		},
		QoSClass:       "besteffort",
		MonitoringPort: 8080,
	}

	configJSON, _ := json.Marshal(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("PUT", "/config", bytes.NewBuffer(configJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		agent.handleUpdateConfig(w, req)
	}
}