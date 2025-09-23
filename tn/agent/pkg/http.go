package pkg

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// startHTTPServer starts the HTTP API server
func (agent *TNAgent) startHTTPServer() error {
	router := mux.NewRouter()

	// Health check endpoint
	router.HandleFunc("/health", agent.handleHealth).Methods("GET")

	// Status endpoint
	router.HandleFunc("/status", agent.handleStatus).Methods("GET")

	// Configuration endpoints
	router.HandleFunc("/config", agent.handleGetConfig).Methods("GET")
	router.HandleFunc("/config", agent.handleUpdateConfig).Methods("PUT")

	// Slice management
	router.HandleFunc("/slices/{sliceId}", agent.handleConfigureSlice).Methods("POST")
	router.HandleFunc("/slices/{sliceId}", agent.handleDeleteSlice).Methods("DELETE")

	// Performance testing
	router.HandleFunc("/tests", agent.handleRunTest).Methods("POST")
	router.HandleFunc("/tests/{testId}", agent.handleGetTestResult).Methods("GET")

	// VXLAN management
	router.HandleFunc("/vxlan/status", agent.handleVXLANStatus).Methods("GET")
	router.HandleFunc("/vxlan/peers", agent.handleUpdateVXLANPeers).Methods("PUT")
	router.HandleFunc("/vxlan/connectivity", agent.handleTestVXLANConnectivity).Methods("POST")

	// Traffic Control management
	router.HandleFunc("/tc/status", agent.handleTCStatus).Methods("GET")
	router.HandleFunc("/tc/rules", agent.handleApplyTCRules).Methods("POST")
	router.HandleFunc("/tc/rules", agent.handleClearTCRules).Methods("DELETE")

	// Bandwidth monitoring
	router.HandleFunc("/bandwidth", agent.handleBandwidthMetrics).Methods("GET")
	router.HandleFunc("/bandwidth/stream", agent.handleBandwidthStream).Methods("GET")

	// Iperf management
	router.HandleFunc("/iperf/servers", agent.handleIperfServers).Methods("GET")
	router.HandleFunc("/iperf/servers/{port}", agent.handleStartIperfServer).Methods("POST")
	router.HandleFunc("/iperf/servers/{port}", agent.handleStopIperfServer).Methods("DELETE")

	// Metrics and reporting
	router.HandleFunc("/metrics", agent.handleGetMetrics).Methods("GET")
	router.Handle("/prometheus", promhttp.Handler())
	router.HandleFunc("/metrics/export", agent.handleExportMetrics).Methods("GET")

	// Add CORS and logging middleware
	router.Use(corsMiddleware)
	router.Use(loggingMiddleware(agent.logger))

	agent.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", agent.config.MonitoringPort),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		agent.logger.Printf("Starting HTTP server on port %d", agent.config.MonitoringPort)
		if err := agent.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			agent.logger.Printf("HTTP server error: %v", err)
		}
	}()

	return nil
}

// Health check handler
func (agent *TNAgent) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"healthy":   agent.healthy,
		"timestamp": time.Now(),
		"version":   "1.0.0",
		"cluster":   agent.config.ClusterName,
	}

	w.Header().Set("Content-Type", "application/json")
	if agent.healthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(status)
}

// Status handler
func (agent *TNAgent) handleStatus(w http.ResponseWriter, r *http.Request) {
	status, err := agent.GetStatus()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get status: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// Get configuration handler
func (agent *TNAgent) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	agent.mu.RLock()
	config := agent.config
	agent.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// Update configuration handler
func (agent *TNAgent) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var newConfig TNConfig
	if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
		http.Error(w, fmt.Sprintf("Invalid configuration: %v", err), http.StatusBadRequest)
		return
	}

	if err := agent.updateConfiguration(&newConfig); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update configuration: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// Configure slice handler
func (agent *TNAgent) handleConfigureSlice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sliceID := vars["sliceId"]

	var config TNConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, fmt.Sprintf("Invalid slice configuration: %v", err), http.StatusBadRequest)
		return
	}

	if err := agent.ConfigureSlice(sliceID, &config); err != nil {
		http.Error(w, fmt.Sprintf("Failed to configure slice: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"sliceId": sliceID,
		"status":  "configured",
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Delete slice handler
func (agent *TNAgent) handleDeleteSlice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sliceID := vars["sliceId"]

	// Implementation would remove slice configuration
	agent.logger.Printf("Deleting slice: %s", sliceID)

	response := map[string]interface{}{
		"sliceId": sliceID,
		"status":  "deleted",
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Run test handler
func (agent *TNAgent) handleRunTest(w http.ResponseWriter, r *http.Request) {
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
	result, err := agent.RunPerformanceTest(&testConfig)
	if err != nil {
		http.Error(w, fmt.Sprintf("Test execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Get test result handler
func (agent *TNAgent) handleGetTestResult(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testID := vars["testId"]

	// In a real implementation, this would retrieve stored test results
	// For now, return a placeholder response
	response := map[string]interface{}{
		"testId": testID,
		"status": "completed",
		"message": "Test result retrieval not yet implemented",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// VXLAN status handler
func (agent *TNAgent) handleVXLANStatus(w http.ResponseWriter, r *http.Request) {
	if agent.vxlanManager == nil {
		http.Error(w, "VXLAN manager not initialized", http.StatusServiceUnavailable)
		return
	}

	status, err := agent.vxlanManager.GetTunnelStatus()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get VXLAN status: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// Update VXLAN peers handler
func (agent *TNAgent) handleUpdateVXLANPeers(w http.ResponseWriter, r *http.Request) {
	if agent.vxlanManager == nil {
		http.Error(w, "VXLAN manager not initialized", http.StatusServiceUnavailable)
		return
	}

	var peers []string
	if err := json.NewDecoder(r.Body).Decode(&peers); err != nil {
		http.Error(w, fmt.Sprintf("Invalid peer list: %v", err), http.StatusBadRequest)
		return
	}

	if err := agent.vxlanManager.UpdatePeers(peers); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update peers: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"peers": peers,
		"status": "updated",
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Test VXLAN connectivity handler
func (agent *TNAgent) handleTestVXLANConnectivity(w http.ResponseWriter, r *http.Request) {
	if agent.vxlanManager == nil {
		http.Error(w, "VXLAN manager not initialized", http.StatusServiceUnavailable)
		return
	}

	results := agent.vxlanManager.TestConnectivity()

	response := map[string]interface{}{
		"connectivity": results,
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// TC status handler
func (agent *TNAgent) handleTCStatus(w http.ResponseWriter, r *http.Request) {
	if agent.tcManager == nil {
		http.Error(w, "TC manager not initialized", http.StatusServiceUnavailable)
		return
	}

	status, err := agent.tcManager.GetTCStatus()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get TC status: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// Apply TC rules handler
func (agent *TNAgent) handleApplyTCRules(w http.ResponseWriter, r *http.Request) {
	if agent.tcManager == nil {
		http.Error(w, "TC manager not initialized", http.StatusServiceUnavailable)
		return
	}

	var policy BandwidthPolicy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		http.Error(w, fmt.Sprintf("Invalid bandwidth policy: %v", err), http.StatusBadRequest)
		return
	}

	if err := agent.tcManager.UpdateShaping(&policy); err != nil {
		http.Error(w, fmt.Sprintf("Failed to apply TC rules: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status": "applied",
		"timestamp": time.Now(),
		"policy": policy,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Clear TC rules handler
func (agent *TNAgent) handleClearTCRules(w http.ResponseWriter, r *http.Request) {
	if agent.tcManager == nil {
		http.Error(w, "TC manager not initialized", http.StatusServiceUnavailable)
		return
	}

	if err := agent.tcManager.CleanRules(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to clear TC rules: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status": "cleared",
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Bandwidth metrics handler
func (agent *TNAgent) handleBandwidthMetrics(w http.ResponseWriter, r *http.Request) {
	if agent.monitor == nil {
		http.Error(w, "Bandwidth monitor not initialized", http.StatusServiceUnavailable)
		return
	}

	metrics := agent.monitor.GetCurrentMetrics()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// Bandwidth stream handler (Server-Sent Events)
func (agent *TNAgent) handleBandwidthStream(w http.ResponseWriter, r *http.Request) {
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
	metrics := agent.monitor.GetCurrentMetrics()
	data, _ := json.Marshal(metrics)
	fmt.Fprintf(w, "data: %s\n\n", data)

	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Stream metrics every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			metrics := agent.monitor.GetCurrentMetrics()
			data, err := json.Marshal(metrics)
			if err != nil {
				continue
			}

			fmt.Fprintf(w, "data: %s\n\n", data)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}

// Iperf servers handler
func (agent *TNAgent) handleIperfServers(w http.ResponseWriter, r *http.Request) {
	servers := agent.iperfManager.GetActiveServers()

	response := map[string]interface{}{
		"servers": servers,
		"count": len(servers),
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Start iperf server handler
func (agent *TNAgent) handleStartIperfServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	portStr := vars["port"]

	port, err := strconv.Atoi(portStr)
	if err != nil {
		http.Error(w, "Invalid port number", http.StatusBadRequest)
		return
	}

	if err := agent.iperfManager.StartServer(port); err != nil {
		http.Error(w, fmt.Sprintf("Failed to start iperf server: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"port": port,
		"status": "started",
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Stop iperf server handler
func (agent *TNAgent) handleStopIperfServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	portStr := vars["port"]

	port, err := strconv.Atoi(portStr)
	if err != nil {
		http.Error(w, "Invalid port number", http.StatusBadRequest)
		return
	}

	if err := agent.iperfManager.StopServer(port); err != nil {
		http.Error(w, fmt.Sprintf("Failed to stop iperf server: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"port": port,
		"status": "stopped",
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Get metrics handler
func (agent *TNAgent) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	summary := agent.monitor.GetPerformanceSummary()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// Export metrics handler
func (agent *TNAgent) handleExportMetrics(w http.ResponseWriter, r *http.Request) {
	data, err := agent.monitor.ExportMetrics()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to export metrics: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=metrics_%s_%d.json",
		agent.config.ClusterName, time.Now().Unix()))
	w.Write(data)
}

// updateConfiguration updates the agent configuration
func (agent *TNAgent) updateConfiguration(newConfig *TNConfig) error {
	agent.mu.Lock()
	defer agent.mu.Unlock()

	agent.logger.Printf("Updating configuration for cluster: %s", newConfig.ClusterName)

	// Update VXLAN configuration
	if agent.vxlanManager != nil {
		if err := agent.vxlanManager.UpdatePeers(newConfig.VXLANConfig.RemoteIPs); err != nil {
			return fmt.Errorf("failed to update VXLAN peers: %w", err)
		}
	}

	// Update TC configuration
	if agent.tcManager != nil {
		if err := agent.tcManager.UpdateShaping(&newConfig.BWPolicy); err != nil {
			return fmt.Errorf("failed to update traffic shaping: %w", err)
		}
	}

	// Update agent configuration
	agent.config = newConfig

	agent.logger.Println("Configuration updated successfully")
	return nil
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Logging middleware
func loggingMiddleware(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)
			logger.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}