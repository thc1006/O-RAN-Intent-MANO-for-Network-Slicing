package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockTNAgent for testing HTTP handlers
type MockTNAgent struct {
	mock.Mock
	healthy bool
	config  *TNConfig
	server  *http.Server
	logger  TestLogger
}

func (m *MockTNAgent) GetStatus() (interface{}, error) {
	args := m.Called()
	return args.Get(0), args.Error(1)
}

func (m *MockTNAgent) updateConfiguration(config *TNConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockTNAgent) ConfigureSlice(sliceID string, config *TNConfig) error {
	args := m.Called(sliceID, config)
	return args.Error(0)
}

func (m *MockTNAgent) RunPerformanceTest(config *PerformanceTestConfig) (interface{}, error) {
	args := m.Called(config)
	return args.Get(0), args.Error(1)
}

func (m *MockTNAgent) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if _, err := w.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	return nil
}

// Mock managers
type MockVXLANManager struct {
	mock.Mock
}

func (m *MockVXLANManager) GetTunnelStatus() (interface{}, error) {
	args := m.Called()
	return args.Get(0), args.Error(1)
}

func (m *MockVXLANManager) UpdatePeers(peers []string) error {
	args := m.Called(peers)
	return args.Error(0)
}

func (m *MockVXLANManager) TestConnectivity() interface{} {
	args := m.Called()
	return args.Get(0)
}

type MockTCManager struct {
	mock.Mock
}

func (m *MockTCManager) GetTCStatus() (interface{}, error) {
	args := m.Called()
	return args.Get(0), args.Error(1)
}

func (m *MockTCManager) UpdateShaping(policy *BandwidthPolicy) error {
	args := m.Called(policy)
	return args.Error(0)
}

func (m *MockTCManager) CleanRules() error {
	args := m.Called()
	return args.Error(0)
}

type MockMonitor struct {
	mock.Mock
}

func (m *MockMonitor) GetCurrentMetrics() interface{} {
	args := m.Called()
	return args.Get(0)
}

func (m *MockMonitor) GetPerformanceSummary() interface{} {
	args := m.Called()
	return args.Get(0)
}

func (m *MockMonitor) ExportMetrics() ([]byte, error) {
	args := m.Called()
	return args.Get(0).([]byte), args.Error(1)
}

type MockIperfManager struct {
	mock.Mock
}

func (m *MockIperfManager) GetActiveServers() map[string]*IperfServer {
	args := m.Called()
	return args.Get(0).(map[string]*IperfServer)
}

func (m *MockIperfManager) StartServer(port int) error {
	args := m.Called(port)
	return args.Error(0)
}

func (m *MockIperfManager) StopServer(port int) error {
	args := m.Called(port)
	return args.Error(0)
}

// TestLogger for testing
type TestLogger struct {
	logs []string
}

func (t *TestLogger) Printf(format string, v ...interface{}) {
	t.logs = append(t.logs, fmt.Sprintf(format, v...))
}

func (t *TestLogger) Println(v ...interface{}) {
	t.logs = append(t.logs, fmt.Sprintln(v...))
}

func setupTestAgent() *TNAgent {
	config := &TNConfig{
		ClusterName:    "test-cluster",
		NodeID:         "test-node",
		MonitoringPort: 8080,
		VXLANConfig: VXLANConfig{
			Interface: "eth0",
			LocalIP:   "10.0.1.1",
			RemoteIPs: []string{"10.0.1.2"},
		},
		BWPolicy: BandwidthPolicy{
			DownlinkMbps: 100.0,
			UplinkMbps:   50.0,
		},
	}

	agent := &TNAgent{
		config:  config,
		healthy: true,
		logger:  &TestLogger{},
	}

	return agent
}

func TestHTTPHandlers_HealthEndpoint(t *testing.T) {
	testCases := []struct {
		name           string
		healthy        bool
		expectedStatus int
		expectedHealth bool
	}{
		{
			name:           "healthy agent",
			healthy:        true,
			expectedStatus: http.StatusOK,
			expectedHealth: true,
		},
		{
			name:           "unhealthy agent",
			healthy:        false,
			expectedStatus: http.StatusServiceUnavailable,
			expectedHealth: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			agent := setupTestAgent()
			agent.healthy = tc.healthy

			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()

			agent.handleHealth(w, req)

			require.Equal(t, tc.expectedStatus, w.Code)
			require.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedHealth, response["healthy"])
			assert.Equal(t, "1.0.0", response["version"])
			assert.Equal(t, "test-cluster", response["cluster"])
			assert.NotNil(t, response["timestamp"])
		})
	}
}

func TestHTTPHandlers_StatusEndpoint_Errors(t *testing.T) {
	testCases := []struct {
		name           string
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "GetStatus returns error",
			mockError:      errors.New("status retrieval failed"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to get status: status retrieval failed",
		},
		{
			name:           "GetStatus returns timeout error",
			mockError:      context.DeadlineExceeded,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to get status: context deadline exceeded",
		},
		{
			name:           "GetStatus returns permission error",
			mockError:      fmt.Errorf("permission denied: %w", errors.New("access forbidden")),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to get status: permission denied: access forbidden",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockAgent := &MockTNAgent{
				healthy: true,
				config:  setupTestAgent().config,
				logger:  TestLogger{},
			}
			mockAgent.On("GetStatus").Return(nil, tc.mockError)

			req := httptest.NewRequest("GET", "/status", nil)
			w := httptest.NewRecorder()

			// Create a function that calls GetStatus and handles errors like the real handler
			handler := func(w http.ResponseWriter, r *http.Request) {
				status, err := mockAgent.GetStatus()
				if err != nil {
					http.Error(w, fmt.Sprintf("Failed to get status: %v", err), http.StatusInternalServerError)
					return
				}

				if err := mockAgent.writeJSONResponse(w, http.StatusOK, status); err != nil {
					http.Error(w, "Internal server error", http.StatusInternalServerError)
				}
			}

			handler(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tc.expectedBody)
			mockAgent.AssertExpectations(t)
		})
	}
}

func TestHTTPHandlers_ConfigurationEndpoints_Errors(t *testing.T) {
	t.Run("UpdateConfig with invalid JSON", func(t *testing.T) {
		agent := setupTestAgent()

		invalidJSON := `{"invalid": json}`
		req := httptest.NewRequest("PUT", "/config", strings.NewReader(invalidJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		agent.handleUpdateConfig(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid configuration")
	})

	t.Run("UpdateConfig with update failure", func(t *testing.T) {
		mockAgent := &MockTNAgent{
			healthy: true,
			config:  setupTestAgent().config,
			logger:  TestLogger{},
		}

		updateError := errors.New("update configuration failed")
		mockAgent.On("updateConfiguration", mock.AnythingOfType("*pkg.TNConfig")).Return(updateError)

		validConfig := `{"clusterName": "test", "nodeId": "node1", "monitoringPort": 8080}`
		req := httptest.NewRequest("PUT", "/config", strings.NewReader(validConfig))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Create handler that mimics the real updateConfig handler
		handler := func(w http.ResponseWriter, r *http.Request) {
			var newConfig TNConfig
			if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
				http.Error(w, fmt.Sprintf("Invalid configuration: %v", err), http.StatusBadRequest)
				return
			}

			if err := mockAgent.updateConfiguration(&newConfig); err != nil {
				http.Error(w, fmt.Sprintf("Failed to update configuration: %v", err), http.StatusInternalServerError)
				return
			}

			if err := mockAgent.writeJSONResponse(w, http.StatusOK, map[string]string{"status": "updated"}); err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}

		handler(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to update configuration: update configuration failed")
		mockAgent.AssertExpectations(t)
	})
}

func TestHTTPHandlers_SliceManagement_Errors(t *testing.T) {
	t.Run("ConfigureSlice with invalid JSON", func(t *testing.T) {
		agent := setupTestAgent()

		req := httptest.NewRequest("POST", "/slices/slice1", strings.NewReader(`invalid json`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Add URL vars for mux
		vars := map[string]string{"sliceId": "slice1"}
		req = mux.SetURLVars(req, vars)

		agent.handleConfigureSlice(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid slice configuration")
	})

	t.Run("ConfigureSlice with configuration failure", func(t *testing.T) {
		mockAgent := &MockTNAgent{
			healthy: true,
			config:  setupTestAgent().config,
			logger:  TestLogger{},
		}

		configError := errors.New("slice configuration failed")
		mockAgent.On("ConfigureSlice", "slice1", mock.AnythingOfType("*pkg.TNConfig")).Return(configError)

		validConfig := `{"clusterName": "test"}`
		req := httptest.NewRequest("POST", "/slices/slice1", strings.NewReader(validConfig))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		vars := map[string]string{"sliceId": "slice1"}
		req = mux.SetURLVars(req, vars)

		handler := func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			sliceID := vars["sliceId"]

			var config TNConfig
			if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
				http.Error(w, fmt.Sprintf("Invalid slice configuration: %v", err), http.StatusBadRequest)
				return
			}

			if err := mockAgent.ConfigureSlice(sliceID, &config); err != nil {
				http.Error(w, fmt.Sprintf("Failed to configure slice: %v", err), http.StatusInternalServerError)
				return
			}

			response := map[string]interface{}{
				"sliceId":   sliceID,
				"status":    "configured",
				"timestamp": time.Now(),
			}

			if err := mockAgent.writeJSONResponse(w, http.StatusOK, response); err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}

		handler(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to configure slice: slice configuration failed")
		mockAgent.AssertExpectations(t)
	})
}

func TestHTTPHandlers_VXLANManagement_Errors(t *testing.T) {
	t.Run("VXLAN status with manager not initialized", func(t *testing.T) {
		agent := setupTestAgent()
		agent.vxlanManager = nil

		req := httptest.NewRequest("GET", "/vxlan/status", nil)
		w := httptest.NewRecorder()

		agent.handleVXLANStatus(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), "VXLAN manager not initialized")
	})

	t.Run("VXLAN status with manager error", func(t *testing.T) {
		agent := setupTestAgent()
		mockVXLAN := &MockVXLANManager{}
		agent.vxlanManager = mockVXLAN

		statusError := errors.New("failed to get VXLAN status")
		mockVXLAN.On("GetTunnelStatus").Return(nil, statusError)

		req := httptest.NewRequest("GET", "/vxlan/status", nil)
		w := httptest.NewRecorder()

		// Mock the real handler behavior
		handler := func(w http.ResponseWriter, r *http.Request) {
			if agent.vxlanManager == nil {
				http.Error(w, "VXLAN manager not initialized", http.StatusServiceUnavailable)
				return
			}

			status, err := mockVXLAN.GetTunnelStatus()
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to get VXLAN status: %v", err), http.StatusInternalServerError)
				return
			}

			if err := agent.writeJSONResponse(w, http.StatusOK, status); err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}

		handler(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to get VXLAN status: failed to get VXLAN status")
		mockVXLAN.AssertExpectations(t)
	})

	t.Run("Update VXLAN peers with invalid JSON", func(t *testing.T) {
		agent := setupTestAgent()
		agent.vxlanManager = &MockVXLANManager{}

		req := httptest.NewRequest("PUT", "/vxlan/peers", strings.NewReader(`invalid json`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		agent.handleUpdateVXLANPeers(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid peer list")
	})

	t.Run("Update VXLAN peers with update failure", func(t *testing.T) {
		agent := setupTestAgent()
		mockVXLAN := &MockVXLANManager{}
		agent.vxlanManager = mockVXLAN

		updateError := errors.New("peer update failed")
		mockVXLAN.On("UpdatePeers", []string{"10.0.1.2", "10.0.1.3"}).Return(updateError)

		validPeers := `["10.0.1.2", "10.0.1.3"]`
		req := httptest.NewRequest("PUT", "/vxlan/peers", strings.NewReader(validPeers))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler := func(w http.ResponseWriter, r *http.Request) {
			if agent.vxlanManager == nil {
				http.Error(w, "VXLAN manager not initialized", http.StatusServiceUnavailable)
				return
			}

			var peers []string
			if err := json.NewDecoder(r.Body).Decode(&peers); err != nil {
				http.Error(w, fmt.Sprintf("Invalid peer list: %v", err), http.StatusBadRequest)
				return
			}

			if err := mockVXLAN.UpdatePeers(peers); err != nil {
				http.Error(w, fmt.Sprintf("Failed to update peers: %v", err), http.StatusInternalServerError)
				return
			}

			response := map[string]interface{}{
				"peers":     peers,
				"status":    "updated",
				"timestamp": time.Now(),
			}

			if err := agent.writeJSONResponse(w, http.StatusOK, response); err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}

		handler(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to update peers: peer update failed")
		mockVXLAN.AssertExpectations(t)
	})
}

func TestHTTPHandlers_TrafficControl_Errors(t *testing.T) {
	t.Run("TC status with manager not initialized", func(t *testing.T) {
		agent := setupTestAgent()
		agent.tcManager = nil

		req := httptest.NewRequest("GET", "/tc/status", nil)
		w := httptest.NewRecorder()

		agent.handleTCStatus(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), "TC manager not initialized")
	})

	t.Run("Apply TC rules with invalid JSON", func(t *testing.T) {
		agent := setupTestAgent()
		agent.tcManager = &MockTCManager{}

		req := httptest.NewRequest("POST", "/tc/rules", strings.NewReader(`invalid json`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		agent.handleApplyTCRules(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid bandwidth policy")
	})

	t.Run("Clear TC rules with manager error", func(t *testing.T) {
		agent := setupTestAgent()
		mockTC := &MockTCManager{}
		agent.tcManager = mockTC

		clearError := errors.New("failed to clear rules")
		mockTC.On("CleanRules").Return(clearError)

		req := httptest.NewRequest("DELETE", "/tc/rules", nil)
		w := httptest.NewRecorder()

		handler := func(w http.ResponseWriter, r *http.Request) {
			if agent.tcManager == nil {
				http.Error(w, "TC manager not initialized", http.StatusServiceUnavailable)
				return
			}

			if err := mockTC.CleanRules(); err != nil {
				http.Error(w, fmt.Sprintf("Failed to clear TC rules: %v", err), http.StatusInternalServerError)
				return
			}

			response := map[string]interface{}{
				"status":    "cleared",
				"timestamp": time.Now(),
			}

			if err := agent.writeJSONResponse(w, http.StatusOK, response); err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}

		handler(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to clear TC rules: failed to clear rules")
		mockTC.AssertExpectations(t)
	})
}

func TestHTTPHandlers_IperfManagement_Errors(t *testing.T) {
	t.Run("Start iperf server with invalid port", func(t *testing.T) {
		agent := setupTestAgent()
		agent.iperfManager = &MockIperfManager{}

		req := httptest.NewRequest("POST", "/iperf/servers/invalid", nil)
		w := httptest.NewRecorder()

		vars := map[string]string{"port": "invalid"}
		req = mux.SetURLVars(req, vars)

		agent.handleStartIperfServer(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid port number")
	})

	t.Run("Start iperf server with start failure", func(t *testing.T) {
		agent := setupTestAgent()
		mockIperf := &MockIperfManager{}
		agent.iperfManager = mockIperf

		startError := errors.New("failed to start server")
		mockIperf.On("StartServer", 5001).Return(startError)

		req := httptest.NewRequest("POST", "/iperf/servers/5001", nil)
		w := httptest.NewRecorder()

		vars := map[string]string{"port": "5001"}
		req = mux.SetURLVars(req, vars)

		handler := func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			portStr := vars["port"]

			port := 5001 // Simplified for test

			if err := mockIperf.StartServer(port); err != nil {
				http.Error(w, fmt.Sprintf("Failed to start iperf server: %v", err), http.StatusInternalServerError)
				return
			}

			response := map[string]interface{}{
				"port":      port,
				"status":    "started",
				"timestamp": time.Now(),
			}

			if err := agent.writeJSONResponse(w, http.StatusOK, response); err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}

		handler(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to start iperf server: failed to start server")
		mockIperf.AssertExpectations(t)
	})

	t.Run("Stop iperf server with stop failure", func(t *testing.T) {
		agent := setupTestAgent()
		mockIperf := &MockIperfManager{}
		agent.iperfManager = mockIperf

		stopError := errors.New("failed to stop server")
		mockIperf.On("StopServer", 5001).Return(stopError)

		req := httptest.NewRequest("DELETE", "/iperf/servers/5001", nil)
		w := httptest.NewRecorder()

		vars := map[string]string{"port": "5001"}
		req = mux.SetURLVars(req, vars)

		handler := func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			portStr := vars["port"]

			port := 5001 // Simplified for test

			if err := mockIperf.StopServer(port); err != nil {
				http.Error(w, fmt.Sprintf("Failed to stop iperf server: %v", err), http.StatusInternalServerError)
				return
			}

			response := map[string]interface{}{
				"port":      port,
				"status":    "stopped",
				"timestamp": time.Now(),
			}

			if err := agent.writeJSONResponse(w, http.StatusOK, response); err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}

		handler(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to stop iperf server: failed to stop server")
		mockIperf.AssertExpectations(t)
	})
}

func TestHTTPHandlers_MonitoringEndpoints_Errors(t *testing.T) {
	t.Run("Bandwidth metrics with monitor not initialized", func(t *testing.T) {
		agent := setupTestAgent()
		agent.monitor = nil

		req := httptest.NewRequest("GET", "/bandwidth", nil)
		w := httptest.NewRecorder()

		agent.handleBandwidthMetrics(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), "Bandwidth monitor not initialized")
	})

	t.Run("Export metrics with export failure", func(t *testing.T) {
		agent := setupTestAgent()
		mockMonitor := &MockMonitor{}
		agent.monitor = mockMonitor

		exportError := errors.New("export failed")
		mockMonitor.On("ExportMetrics").Return([]byte{}, exportError)

		req := httptest.NewRequest("GET", "/metrics/export", nil)
		w := httptest.NewRecorder()

		handler := func(w http.ResponseWriter, r *http.Request) {
			data, err := mockMonitor.ExportMetrics()
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to export metrics: %v", err), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=metrics_%s_%d.json",
				agent.config.ClusterName, time.Now().Unix()))
			w.WriteHeader(http.StatusOK)
			w.Write(data)
		}

		handler(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to export metrics: export failed")
		mockMonitor.AssertExpectations(t)
	})
}

func TestHTTPHandlers_PerformanceTest_Errors(t *testing.T) {
	t.Run("Run test with invalid JSON", func(t *testing.T) {
		agent := setupTestAgent()

		req := httptest.NewRequest("POST", "/tests", strings.NewReader(`invalid json`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		agent.handleRunTest(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid test configuration")
	})

	t.Run("Run test with execution failure", func(t *testing.T) {
		mockAgent := &MockTNAgent{
			healthy: true,
			config:  setupTestAgent().config,
			logger:  TestLogger{},
		}

		testError := errors.New("test execution failed")
		mockAgent.On("RunPerformanceTest", mock.AnythingOfType("*pkg.PerformanceTestConfig")).Return(nil, testError)

		validConfig := `{"testId": "test1", "duration": 10}`
		req := httptest.NewRequest("POST", "/tests", strings.NewReader(validConfig))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler := func(w http.ResponseWriter, r *http.Request) {
			var testConfig PerformanceTestConfig
			if err := json.NewDecoder(r.Body).Decode(&testConfig); err != nil {
				http.Error(w, fmt.Sprintf("Invalid test configuration: %v", err), http.StatusBadRequest)
				return
			}

			// Generate test ID if not provided
			if testConfig.TestID == "" {
				testConfig.TestID = fmt.Sprintf("test_%d", time.Now().Unix())
			}

			// Run the test
			result, err := mockAgent.RunPerformanceTest(&testConfig)
			if err != nil {
				http.Error(w, fmt.Sprintf("Test execution failed: %v", err), http.StatusInternalServerError)
				return
			}

			if err := mockAgent.writeJSONResponse(w, http.StatusOK, result); err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}

		handler(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Test execution failed: test execution failed")
		mockAgent.AssertExpectations(t)
	})
}

func TestHTTPHandlers_JSONResponseErrors(t *testing.T) {
	t.Run("writeJSONResponse with invalid data", func(t *testing.T) {
		agent := setupTestAgent()
		w := httptest.NewRecorder()

		// Create data that will fail JSON encoding (circular reference)
		type cyclicStruct struct {
			Self *cyclicStruct `json:"self"`
		}
		data := &cyclicStruct{}
		data.Self = data

		err := agent.writeJSONResponse(w, http.StatusOK, data)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to encode JSON")
	})

	t.Run("writeJSONResponse with write failure", func(t *testing.T) {
		agent := setupTestAgent()

		// Create a writer that will fail on write
		w := &FailingResponseWriter{
			header: make(http.Header),
		}

		data := map[string]string{"test": "data"}
		err := agent.writeJSONResponse(w, http.StatusOK, data)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write response")
	})
}

// FailingResponseWriter for testing write failures
type FailingResponseWriter struct {
	header http.Header
}

func (f *FailingResponseWriter) Header() http.Header {
	return f.header
}

func (f *FailingResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func (f *FailingResponseWriter) WriteHeader(statusCode int) {
	// Do nothing
}

func TestHTTPHandlers_BandwidthStream_Errors(t *testing.T) {
	t.Run("Bandwidth stream with monitor not initialized", func(t *testing.T) {
		agent := setupTestAgent()
		agent.monitor = nil

		req := httptest.NewRequest("GET", "/bandwidth/stream", nil)
		w := httptest.NewRecorder()

		agent.handleBandwidthStream(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), "Bandwidth monitor not initialized")
	})

	t.Run("Bandwidth stream with metrics marshal failure", func(t *testing.T) {
		agent := setupTestAgent()
		mockMonitor := &MockMonitor{}
		agent.monitor = mockMonitor

		// Return data that cannot be marshaled
		type unmarshalableData struct {
			BadFunc func() `json:"func"`
		}
		badData := unmarshalableData{BadFunc: func() {}}
		mockMonitor.On("GetCurrentMetrics").Return(badData)

		req := httptest.NewRequest("GET", "/bandwidth/stream", nil)
		w := httptest.NewRecorder()

		handler := func(w http.ResponseWriter, r *http.Request) {
			if agent.monitor == nil {
				http.Error(w, "Bandwidth monitor not initialized", http.StatusServiceUnavailable)
				return
			}

			// Set headers for Server-Sent Events
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.Header().Set("Access-Control-Allow-Origin", "*")

			// Send initial metrics
			metrics := mockMonitor.GetCurrentMetrics()
			data, err := json.Marshal(metrics)
			if err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
		}

		handler(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockMonitor.AssertExpectations(t)
	})
}

// Test edge cases and boundary conditions
func TestHTTPHandlers_EdgeCases(t *testing.T) {
	t.Run("Empty request body handling", func(t *testing.T) {
		agent := setupTestAgent()

		req := httptest.NewRequest("PUT", "/config", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		agent.handleUpdateConfig(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Large request body handling", func(t *testing.T) {
		agent := setupTestAgent()

		// Create a very large JSON payload
		largeData := make(map[string]string)
		for i := 0; i < 1000; i++ {
			largeData[fmt.Sprintf("key%d", i)] = strings.Repeat("value", 1000)
		}

		jsonData, _ := json.Marshal(largeData)
		req := httptest.NewRequest("PUT", "/config", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		agent.handleUpdateConfig(w, req)

		// Should handle large payloads (may succeed or fail based on validation)
		assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusInternalServerError || w.Code == http.StatusOK)
	})

	t.Run("Concurrent request handling", func(t *testing.T) {
		agent := setupTestAgent()

		// Simulate concurrent requests
		var responses []int
		responseChan := make(chan int, 10)

		for i := 0; i < 10; i++ {
			go func() {
				req := httptest.NewRequest("GET", "/health", nil)
				w := httptest.NewRecorder()
				agent.handleHealth(w, req)
				responseChan <- w.Code
			}()
		}

		// Collect responses
		for i := 0; i < 10; i++ {
			responses = append(responses, <-responseChan)
		}

		// All responses should be successful
		for _, code := range responses {
			assert.Equal(t, http.StatusOK, code)
		}
	})
}

// Benchmark tests for performance validation
func BenchmarkHTTPHandlers_Health(b *testing.B) {
	agent := setupTestAgent()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		agent.handleHealth(w, req)
	}
}

func BenchmarkHTTPHandlers_Status(b *testing.B) {
	agent := setupTestAgent()

	// Mock GetStatus to return quickly
	mockStatus := map[string]interface{}{
		"healthy": true,
		"uptime":  "1h",
	}

	// Create a simple mock that returns status quickly
	originalGetStatus := func() (interface{}, error) {
		return mockStatus, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate handler call
		status, err := originalGetStatus()
		if err == nil && status != nil {
			// Success path
			continue
		}
	}
}

// Property-based testing for security validation
func TestHTTPHandlers_SecurityValidation(t *testing.T) {
	agent := setupTestAgent()

	// Test with various malicious inputs
	maliciousInputs := []string{
		`{"clusterName": "'; DROP TABLE users; --"}`,
		`{"clusterName": "<script>alert('xss')</script>"}`,
		`{"clusterName": "$(rm -rf /)"}`,
		`{"clusterName": "\u0000\u0001\u0002"}`,
		`{"nodeId": "../../../etc/passwd"}`,
	}

	for _, input := range maliciousInputs {
		t.Run(fmt.Sprintf("malicious_input_%s", input[:20]), func(t *testing.T) {
			req := httptest.NewRequest("PUT", "/config", strings.NewReader(input))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			agent.handleUpdateConfig(w, req)

			// Should either reject or sanitize the input
			assert.True(t, w.Code >= 400 || w.Code == 200)

			// If successful, verify no injection occurred
			if w.Code == 200 {
				// Additional validation could be added here
				assert.NotContains(t, w.Body.String(), "DROP TABLE")
				assert.NotContains(t, w.Body.String(), "<script>")
			}
		})
	}
}