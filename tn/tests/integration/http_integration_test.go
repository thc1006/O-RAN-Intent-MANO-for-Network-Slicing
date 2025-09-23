package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn/agent/pkg"
)

// IntegrationTestSuite for HTTP endpoints
type IntegrationTestSuite struct {
	server   *httptest.Server
	agent    *pkg.TNAgent
	client   *http.Client
	baseURL  string
}

func setupIntegrationTest(t *testing.T) *IntegrationTestSuite {
	// Create a test agent with real configuration
	config := &pkg.TNConfig{
		ClusterName:    "integration-test-cluster",
		NodeID:         "test-node-1",
		MonitoringPort: 0, // Let test server choose port
		VXLANConfig: pkg.VXLANConfig{
			Interface: "eth0",
			LocalIP:   "10.0.1.1",
			RemoteIPs: []string{"10.0.1.2", "10.0.1.3"},
			VxlanID:   100,
		},
		BWPolicy: pkg.BandwidthPolicy{
			DownlinkMbps: 100.0,
			UplinkMbps:   50.0,
		},
	}

	agent, err := pkg.NewTNAgent(config)
	require.NoError(t, err)

	// Create router with all handlers
	router := mux.NewRouter()
	setupRoutes(router, agent)

	// Create test server
	server := httptest.NewServer(router)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &IntegrationTestSuite{
		server:  server,
		agent:   agent,
		client:  client,
		baseURL: server.URL,
	}
}

func setupRoutes(router *mux.Router, agent *pkg.TNAgent) {
	// Health and status
	router.HandleFunc("/health", agent.HandleHealth).Methods("GET")
	router.HandleFunc("/status", agent.HandleStatus).Methods("GET")

	// Configuration
	router.HandleFunc("/config", agent.HandleGetConfig).Methods("GET")
	router.HandleFunc("/config", agent.HandleUpdateConfig).Methods("PUT")

	// Slice management
	router.HandleFunc("/slices/{sliceId}", agent.HandleConfigureSlice).Methods("POST")
	router.HandleFunc("/slices/{sliceId}", agent.HandleDeleteSlice).Methods("DELETE")

	// Performance testing
	router.HandleFunc("/tests", agent.HandleRunTest).Methods("POST")
	router.HandleFunc("/tests/{testId}", agent.HandleGetTestResult).Methods("GET")

	// VXLAN management
	router.HandleFunc("/vxlan/status", agent.HandleVXLANStatus).Methods("GET")
	router.HandleFunc("/vxlan/peers", agent.HandleUpdateVXLANPeers).Methods("PUT")
	router.HandleFunc("/vxlan/connectivity", agent.HandleTestVXLANConnectivity).Methods("POST")

	// Traffic Control
	router.HandleFunc("/tc/status", agent.HandleTCStatus).Methods("GET")
	router.HandleFunc("/tc/rules", agent.HandleApplyTCRules).Methods("POST")
	router.HandleFunc("/tc/rules", agent.HandleClearTCRules).Methods("DELETE")

	// Monitoring
	router.HandleFunc("/bandwidth", agent.HandleBandwidthMetrics).Methods("GET")
	router.HandleFunc("/bandwidth/stream", agent.HandleBandwidthStream).Methods("GET")

	// Iperf management
	router.HandleFunc("/iperf/servers", agent.HandleIperfServers).Methods("GET")
	router.HandleFunc("/iperf/servers/{port}", agent.HandleStartIperfServer).Methods("POST")
	router.HandleFunc("/iperf/servers/{port}", agent.HandleStopIperfServer).Methods("DELETE")

	// Metrics
	router.HandleFunc("/metrics", agent.HandleGetMetrics).Methods("GET")
	router.HandleFunc("/metrics/export", agent.HandleExportMetrics).Methods("GET")
}

func (suite *IntegrationTestSuite) cleanup() {
	if suite.server != nil {
		suite.server.Close()
	}
}

func TestHTTPIntegration_HealthEndpoint(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	// Test healthy agent
	resp, err := suite.client.Get(suite.baseURL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var health map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&health)
	require.NoError(t, err)

	assert.Equal(t, true, health["healthy"])
	assert.Equal(t, "integration-test-cluster", health["cluster"])
	assert.Equal(t, "1.0.0", health["version"])
	assert.NotNil(t, health["timestamp"])
}

func TestHTTPIntegration_StatusEndpoint(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	resp, err := suite.client.Get(suite.baseURL + "/status")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var status map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&status)
	require.NoError(t, err)

	// Status should contain basic information
	assert.NotNil(t, status)
}

func TestHTTPIntegration_ConfigurationEndpoints(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	// Test GET config
	resp, err := suite.client.Get(suite.baseURL + "/config")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var config pkg.TNConfig
	err = json.NewDecoder(resp.Body).Decode(&config)
	require.NoError(t, err)

	assert.Equal(t, "integration-test-cluster", config.ClusterName)
	assert.Equal(t, "test-node-1", config.NodeID)

	// Test PUT config with valid data
	newConfig := pkg.TNConfig{
		ClusterName:    "updated-cluster",
		NodeID:         "updated-node",
		MonitoringPort: 8081,
		VXLANConfig: pkg.VXLANConfig{
			Interface: "eth1",
			LocalIP:   "10.0.2.1",
			RemoteIPs: []string{"10.0.2.2"},
			VxlanID:   200,
		},
		BWPolicy: pkg.BandwidthPolicy{
			DownlinkMbps: 200.0,
			UplinkMbps:   100.0,
		},
	}

	configData, err := json.Marshal(newConfig)
	require.NoError(t, err)

	resp, err = suite.client.Put(suite.baseURL+"/config", bytes.NewReader(configData))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var updateResponse map[string]string
	err = json.NewDecoder(resp.Body).Decode(&updateResponse)
	require.NoError(t, err)

	assert.Equal(t, "updated", updateResponse["status"])

	// Test PUT config with invalid JSON
	resp, err = suite.client.Put(suite.baseURL+"/config", strings.NewReader(`{invalid json}`))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestHTTPIntegration_SliceManagement(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	sliceID := "test-slice-1"

	// Test POST slice configuration
	sliceConfig := pkg.TNConfig{
		ClusterName: "slice-cluster",
		NodeID:      "slice-node",
	}

	configData, err := json.Marshal(sliceConfig)
	require.NoError(t, err)

	resp, err := suite.client.Post(suite.baseURL+"/slices/"+sliceID, "application/json", bytes.NewReader(configData))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, sliceID, response["sliceId"])
	assert.Equal(t, "configured", response["status"])
	assert.NotNil(t, response["timestamp"])

	// Test DELETE slice
	req, err := http.NewRequest("DELETE", suite.baseURL+"/slices/"+sliceID, nil)
	require.NoError(t, err)

	resp, err = suite.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, sliceID, response["sliceId"])
	assert.Equal(t, "deleted", response["status"])
}

func TestHTTPIntegration_PerformanceTests(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	// Test POST performance test
	testConfig := pkg.PerformanceTestConfig{
		TestID:   "integration-test-1",
		Duration: 10,
		Type:     "bandwidth",
	}

	configData, err := json.Marshal(testConfig)
	require.NoError(t, err)

	resp, err := suite.client.Post(suite.baseURL+"/tests", "application/json", bytes.NewReader(configData))
	require.NoError(t, err)
	defer resp.Body.Close()

	// Might succeed or fail depending on environment, but should respond
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusInternalServerError)

	// Test GET test result
	resp, err = suite.client.Get(suite.baseURL + "/tests/integration-test-1")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "integration-test-1", result["testId"])
}

func TestHTTPIntegration_VXLANManagement(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	// Test VXLAN status (might fail if manager not initialized)
	resp, err := suite.client.Get(suite.baseURL + "/vxlan/status")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should either return status or service unavailable
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusServiceUnavailable)

	if resp.StatusCode == http.StatusServiceUnavailable {
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "VXLAN manager not initialized")
		return // Skip other VXLAN tests if manager not available
	}

	// Test update VXLAN peers
	peers := []string{"10.0.1.10", "10.0.1.11"}
	peersData, err := json.Marshal(peers)
	require.NoError(t, err)

	resp, err = suite.client.Put(suite.baseURL+"/vxlan/peers", bytes.NewReader(peersData))
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should either succeed or fail gracefully
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusInternalServerError)

	// Test VXLAN connectivity
	resp, err = suite.client.Post(suite.baseURL+"/vxlan/connectivity", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusServiceUnavailable)
}

func TestHTTPIntegration_TrafficControl(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	// Test TC status
	resp, err := suite.client.Get(suite.baseURL + "/tc/status")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should either return status or service unavailable
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusServiceUnavailable)

	if resp.StatusCode == http.StatusServiceUnavailable {
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "TC manager not initialized")
		return // Skip other TC tests if manager not available
	}

	// Test apply TC rules
	policy := pkg.BandwidthPolicy{
		DownlinkMbps: 50.0,
		UplinkMbps:   25.0,
	}

	policyData, err := json.Marshal(policy)
	require.NoError(t, err)

	resp, err = suite.client.Post(suite.baseURL+"/tc/rules", bytes.NewReader(policyData))
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should either succeed or fail gracefully
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusInternalServerError)

	// Test clear TC rules
	req, err := http.NewRequest("DELETE", suite.baseURL+"/tc/rules", nil)
	require.NoError(t, err)

	resp, err = suite.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusInternalServerError)
}

func TestHTTPIntegration_IperfManagement(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	// Test get iperf servers
	resp, err := suite.client.Get(suite.baseURL + "/iperf/servers")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var serversResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&serversResponse)
	require.NoError(t, err)

	assert.NotNil(t, serversResponse["servers"])
	assert.NotNil(t, serversResponse["count"])

	// Test start iperf server (use high port to avoid conflicts)
	port := 15001
	resp, err = suite.client.Post(suite.baseURL+fmt.Sprintf("/iperf/servers/%d", port), nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should either succeed or fail gracefully
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusInternalServerError)

	if resp.StatusCode == http.StatusOK {
		var startResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&startResponse)
		require.NoError(t, err)

		assert.Equal(t, float64(port), startResponse["port"])
		assert.Equal(t, "started", startResponse["status"])

		// Test stop iperf server
		req, err := http.NewRequest("DELETE", suite.baseURL+fmt.Sprintf("/iperf/servers/%d", port), nil)
		require.NoError(t, err)

		resp, err = suite.client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var stopResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&stopResponse)
		require.NoError(t, err)

		assert.Equal(t, float64(port), stopResponse["port"])
		assert.Equal(t, "stopped", stopResponse["status"])
	}
}

func TestHTTPIntegration_MonitoringEndpoints(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	// Test bandwidth metrics
	resp, err := suite.client.Get(suite.baseURL + "/bandwidth")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should either return metrics or service unavailable
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusServiceUnavailable)

	// Test metrics endpoint
	resp, err = suite.client.Get(suite.baseURL + "/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Test metrics export
	resp, err = suite.client.Get(suite.baseURL + "/metrics/export")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should either return export or fail gracefully
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusInternalServerError)

	if resp.StatusCode == http.StatusOK {
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		assert.Contains(t, resp.Header.Get("Content-Disposition"), "attachment")
	}
}

func TestHTTPIntegration_BandwidthStream(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	// Test bandwidth stream (Server-Sent Events)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", suite.baseURL+"/bandwidth/stream", nil)
	require.NoError(t, err)

	resp, err := suite.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should either return stream or service unavailable
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusServiceUnavailable)

	if resp.StatusCode == http.StatusOK {
		assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
		assert.Equal(t, "no-cache", resp.Header.Get("Cache-Control"))

		// Read some data from the stream
		buffer := make([]byte, 1024)
		n, err := resp.Body.Read(buffer)

		// Should read something or timeout
		if err == nil && n > 0 {
			data := string(buffer[:n])
			assert.Contains(t, data, "data:")
		}
	}
}

func TestHTTPIntegration_ConcurrentRequests(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	// Test concurrent requests to different endpoints
	var wg sync.WaitGroup
	requestCount := 20
	errorChan := make(chan error, requestCount)

	endpoints := []string{
		"/health",
		"/status",
		"/config",
		"/iperf/servers",
		"/metrics",
	}

	for i := 0; i < requestCount; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			endpoint := endpoints[i%len(endpoints)]
			resp, err := suite.client.Get(suite.baseURL + endpoint)
			if err != nil {
				errorChan <- fmt.Errorf("request %d to %s failed: %w", i, endpoint, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 500 {
				errorChan <- fmt.Errorf("request %d to %s returned %d", i, endpoint, resp.StatusCode)
			}
		}(i)
	}

	wg.Wait()
	close(errorChan)

	// Collect errors
	var errors []error
	for err := range errorChan {
		errors = append(errors, err)
	}

	// Should handle concurrent requests well
	assert.True(t, len(errors) < 5, "Too many errors in concurrent requests: %v", errors)
}

func TestHTTPIntegration_LargePayloads(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	// Test large configuration payload
	largeConfig := pkg.TNConfig{
		ClusterName: strings.Repeat("large-cluster-name-", 100),
		NodeID:      strings.Repeat("large-node-id-", 100),
		VXLANConfig: pkg.VXLANConfig{
			Interface: "eth0",
			LocalIP:   "10.0.1.1",
			RemoteIPs: make([]string, 1000),
		},
	}

	// Fill with valid IP addresses
	for i := range largeConfig.VXLANConfig.RemoteIPs {
		largeConfig.VXLANConfig.RemoteIPs[i] = fmt.Sprintf("10.%d.%d.%d",
			(i/65536)%256, (i/256)%256, i%256)
	}

	configData, err := json.Marshal(largeConfig)
	require.NoError(t, err)

	resp, err := suite.client.Put(suite.baseURL+"/config", bytes.NewReader(configData))
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should handle large payloads (might succeed or fail based on validation)
	assert.True(t, resp.StatusCode == http.StatusOK ||
		resp.StatusCode == http.StatusBadRequest ||
		resp.StatusCode == http.StatusInternalServerError)
}

func TestHTTPIntegration_ErrorRecovery(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	// Test malformed JSON handling
	malformedPayloads := []string{
		`{`,
		`{"invalid": }`,
		`{invalid json}`,
		``,
		`null`,
		strings.Repeat(`{"nested":`, 1000) + `"value"` + strings.Repeat(`}`, 1000),
	}

	for i, payload := range malformedPayloads {
		t.Run(fmt.Sprintf("malformed_payload_%d", i), func(t *testing.T) {
			resp, err := suite.client.Put(suite.baseURL+"/config", strings.NewReader(payload))
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should reject malformed JSON
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			// Server should still be responsive after error
			healthResp, err := suite.client.Get(suite.baseURL + "/health")
			require.NoError(t, err)
			defer healthResp.Body.Close()

			assert.Equal(t, http.StatusOK, healthResp.StatusCode)
		})
	}
}

func TestHTTPIntegration_ConnectionHandling(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	// Test connection limits and handling
	client := &http.Client{
		Timeout: 1 * time.Second, // Short timeout
		Transport: &http.Transport{
			MaxIdleConns:        1,
			MaxConnsPerHost:     1,
			IdleConnTimeout:     1 * time.Second,
			DisableKeepAlives:   false,
		},
	}

	// Make multiple requests to test connection reuse
	for i := 0; i < 10; i++ {
		resp, err := client.Get(suite.baseURL + "/health")
		require.NoError(t, err)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, string(body), "healthy")
	}
}

func TestHTTPIntegration_ContentNegotiation(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	// Test different Accept headers
	req, err := http.NewRequest("GET", suite.baseURL+"/health", nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")

	resp, err := suite.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	// Test unsupported content type for POST requests
	req, err = http.NewRequest("POST", suite.baseURL+"/slices/test", strings.NewReader(`{"test": true}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "text/plain")

	resp, err = suite.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should still process as the handler looks at the body content
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest)
}

func TestHTTPIntegration_SecurityHeaders(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	// Test CORS headers
	req, err := http.NewRequest("OPTIONS", suite.baseURL+"/health", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://example.com")

	resp, err := suite.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Contains(t, resp.Header.Get("Access-Control-Allow-Methods"), "GET")
	assert.Contains(t, resp.Header.Get("Access-Control-Allow-Methods"), "POST")
}

func TestHTTPIntegration_LoadTesting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	// Simulate load testing
	var wg sync.WaitGroup
	requestCount := 100
	concurrency := 10
	sem := make(chan struct{}, concurrency)

	startTime := time.Now()
	errorCount := 0
	var mu sync.Mutex

	for i := 0; i < requestCount; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			resp, err := suite.client.Get(suite.baseURL + "/health")
			if err != nil {
				mu.Lock()
				errorCount++
				mu.Unlock()
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				mu.Lock()
				errorCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("Load test completed: %d requests in %v", requestCount, duration)
	t.Logf("Request rate: %.2f req/s", float64(requestCount)/duration.Seconds())
	t.Logf("Error rate: %.2f%%", float64(errorCount)/float64(requestCount)*100)

	// Should handle load reasonably well
	assert.True(t, float64(errorCount)/float64(requestCount) < 0.1, "Error rate too high: %d/%d", errorCount, requestCount)
	assert.True(t, duration < 30*time.Second, "Load test took too long: %v", duration)
}

// Test network-level errors and resilience
func TestHTTPIntegration_NetworkErrors(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	// Test timeout handling
	client := &http.Client{
		Timeout: 100 * time.Millisecond, // Very short timeout
	}

	// Test with timeout (might work if server is very fast)
	resp, err := client.Get(suite.baseURL + "/health")
	if err != nil {
		// Timeout is expected with very short timeout
		assert.Contains(t, err.Error(), "timeout")
	} else {
		defer resp.Body.Close()
		// If it succeeds, it should be valid
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	// Test connection to invalid port (should fail)
	invalidURL := "http://localhost:99999/health"
	_, err = client.Get(invalidURL)
	assert.Error(t, err)
	assert.True(t,
		strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "no connection could be made"),
	)
}

// Helper function to check if port is available
func isPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}