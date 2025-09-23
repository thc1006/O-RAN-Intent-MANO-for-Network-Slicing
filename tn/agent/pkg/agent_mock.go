package pkg

import (
	"log"
	"net/http"
	"net/http/httptest"
	"time"
)

// MockTNAgent provides a mock implementation of TNAgent for testing
type MockTNAgent struct {
	config *TNConfig
	logger *log.Logger
	server *httptest.Server
	healthy bool
}

// NewMockTNAgent creates a new mock TN agent for testing
func NewMockTNAgent(config *TNConfig, logger *log.Logger) *MockTNAgent {
	agent := &MockTNAgent{
		config:  config,
		logger:  logger,
		healthy: true,
	}

	// Create a test server that responds to health checks
	agent.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			if agent.healthy {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"healthy"}`))
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
		case "/status":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"healthy": true,
				"vxlanStatus": {"tunnelUp": true},
				"tcStatus": {"rulesActive": true, "shapingActive": true},
				"bandwidthUsage": {"eth0": {"rx": 1000, "tx": 1000}}
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	return agent
}

// Start starts the mock agent (no-op for mock)
func (ma *MockTNAgent) Start() error {
	ma.logger.Printf("Mock TN Agent started on %s", ma.server.URL)
	return nil
}

// Stop stops the mock agent
func (ma *MockTNAgent) Stop() error {
	if ma.server != nil {
		ma.server.Close()
	}
	ma.logger.Println("Mock TN Agent stopped")
	return nil
}

// GetStatus returns the status of the mock agent
func (ma *MockTNAgent) GetStatus() (*TNStatus, error) {
	return &TNStatus{
		Healthy:           ma.healthy,
		LastUpdate:        time.Now(),
		ActiveConnections: len(ma.config.VXLANConfig.RemoteIPs),
		BandwidthUsage:    map[string]float64{
			"eth0": 10.0,
		},
		VXLANStatus: VXLANStatus{
			TunnelUp:    true,
			RemotePeers: ma.config.VXLANConfig.RemoteIPs,
			PacketStats: map[string]int64{
				"rx_packets": 1000,
				"tx_packets": 1000,
			},
			LastHeartbeat: time.Now(),
		},
		TCStatus: TCStatus{
			RulesActive:   true,
			ShapingActive: true,
		},
	}, nil
}

// GetHealthEndpoint returns the health check URL for the mock agent
func (ma *MockTNAgent) GetHealthEndpoint() string {
	return ma.server.URL + "/health"
}

// CreateAgentWithMockInCI creates either a real or mock agent based on CI environment
func CreateAgentWithMockInCI(config *TNConfig, logger *log.Logger) interface{} {
	if IsRunningInCI() {
		logger.Println("Creating mock TN agent for CI environment")
		return NewMockTNAgent(config, logger)
	}
	return NewTNAgent(config, logger)
}