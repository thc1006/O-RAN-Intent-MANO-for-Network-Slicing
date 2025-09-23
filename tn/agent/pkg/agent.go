package pkg

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// TNAgent represents the Transport Network agent
type TNAgent struct {
	config        *TNConfig
	vxlanManager  *VXLANManager
	tcManager     *TCManager
	iperfManager  *IperfManager
	monitor       *BandwidthMonitor
	server        *http.Server
	logger        *log.Logger
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
	healthy       bool
}

// NewTNAgent creates a new Transport Network agent
func NewTNAgent(config *TNConfig, logger *log.Logger) *TNAgent {
	ctx, cancel := context.WithCancel(context.Background())

	return &TNAgent{
		config:       config,
		logger:       logger,
		ctx:          ctx,
		cancel:       cancel,
		healthy:      false,
		iperfManager: NewIperfManager(logger),
		monitor:      NewBandwidthMonitor(logger),
	}
}

// Start initializes and starts the TN agent
func (agent *TNAgent) Start() error {
	agent.logger.Printf("Starting TN Agent for cluster: %s", agent.config.ClusterName)

	// Initialize VXLAN manager
	if err := agent.initializeVXLAN(); err != nil {
		return fmt.Errorf("failed to initialize VXLAN: %w", err)
	}

	// Initialize TC manager
	if err := agent.initializeTC(); err != nil {
		return fmt.Errorf("failed to initialize TC: %w", err)
	}

	// Start iperf3 servers
	if err := agent.startIperfServers(); err != nil {
		return fmt.Errorf("failed to start iperf servers: %w", err)
	}

	// Start bandwidth monitoring
	if err := agent.monitor.Start(agent.ctx); err != nil {
		return fmt.Errorf("failed to start bandwidth monitoring: %w", err)
	}

	// Start HTTP server for API
	if err := agent.startHTTPServer(); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	// Start monitoring loop
	go agent.monitoringLoop()

	agent.healthy = true
	agent.logger.Printf("TN Agent started successfully on port %d", agent.config.MonitoringPort)

	return nil
}

// Stop gracefully shuts down the TN agent
func (agent *TNAgent) Stop() error {
	agent.logger.Println("Stopping TN Agent...")

	agent.cancel()

	// Stop HTTP server
	if agent.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := agent.server.Shutdown(ctx); err != nil {
			agent.logger.Printf("HTTP server shutdown error: %v", err)
		}
	}

	// Stop iperf servers
	if err := agent.iperfManager.StopAllServers(); err != nil {
		agent.logger.Printf("Error stopping iperf servers: %v", err)
	}

	// Clean TC rules
	if agent.tcManager != nil {
		if err := agent.tcManager.CleanRules(); err != nil {
			agent.logger.Printf("Error cleaning TC rules: %v", err)
		}
	}

	// Delete VXLAN tunnel
	if agent.vxlanManager != nil {
		if err := agent.vxlanManager.DeleteTunnel(); err != nil {
			agent.logger.Printf("Error deleting VXLAN tunnel: %v", err)
		}
	}

	// Stop bandwidth monitoring
	if err := agent.monitor.Stop(); err != nil {
		agent.logger.Printf("Error stopping bandwidth monitor: %v", err)
	}

	agent.healthy = false
	agent.logger.Println("TN Agent stopped")

	return nil
}

// initializeVXLAN sets up VXLAN tunnel management
func (agent *TNAgent) initializeVXLAN() error {
	agent.logger.Println("Initializing VXLAN manager")

	agent.vxlanManager = NewVXLANManager(&agent.config.VXLANConfig, agent.logger)

	// Create VXLAN tunnel
	if err := agent.vxlanManager.CreateTunnel(); err != nil {
		return fmt.Errorf("failed to create VXLAN tunnel: %w", err)
	}

	// Start tunnel monitoring
	go agent.vxlanManager.MonitorTunnel(30*time.Second, agent.ctx.Done())

	agent.logger.Println("VXLAN manager initialized successfully")
	return nil
}

// initializeTC sets up Traffic Control management
func (agent *TNAgent) initializeTC() error {
	agent.logger.Println("Initializing TC manager")

	// Use the VXLAN interface for traffic control
	interfaceName := agent.config.VXLANConfig.DeviceName
	if interfaceName == "" {
		// Fallback to primary network interface
		interfaceName = agent.getPrimaryInterface()
	}

	agent.tcManager = NewTCManager(&agent.config.BWPolicy, interfaceName, agent.logger)

	// Apply traffic shaping
	if err := agent.tcManager.ApplyShaping(); err != nil {
		return fmt.Errorf("failed to apply traffic shaping: %w", err)
	}

	// Start bandwidth monitoring for TC
	go agent.tcManager.MonitorBandwidth(10*time.Second, agent.ctx.Done())

	agent.logger.Println("TC manager initialized successfully")
	return nil
}

// getPrimaryInterface gets the primary network interface name
func (agent *TNAgent) getPrimaryInterface() string {
	// This is a simplified implementation
	// In a real deployment, you'd use more sophisticated network interface detection
	return "eth0"
}

// startIperfServers starts iperf3 servers for performance testing
func (agent *TNAgent) startIperfServers() error {
	agent.logger.Println("Starting iperf3 servers")

	// Start iperf3 server on monitoring port + 1
	iperfPort := agent.config.MonitoringPort + 1
	if err := agent.iperfManager.StartServer(iperfPort); err != nil {
		return fmt.Errorf("failed to start iperf3 server: %w", err)
	}

	// Start additional servers for different test types
	for i, port := range []int{iperfPort + 1, iperfPort + 2} {
		if err := agent.iperfManager.StartServer(port); err != nil {
			agent.logger.Printf("Warning: failed to start additional iperf3 server %d: %v", i, err)
		}
	}

	agent.logger.Println("Iperf3 servers started successfully")
	return nil
}

// ConfigureSlice configures the network slice
func (agent *TNAgent) ConfigureSlice(sliceID string, config *TNConfig) error {
	agent.mu.Lock()
	defer agent.mu.Unlock()

	agent.logger.Printf("Configuring network slice: %s", sliceID)

	// Update configuration
	agent.config = config

	// Reconfigure VXLAN if needed
	if agent.vxlanManager != nil {
		if err := agent.vxlanManager.UpdatePeers(config.VXLANConfig.RemoteIPs); err != nil {
			agent.logger.Printf("Warning: failed to update VXLAN peers: %v", err)
		}
	}

	// Reconfigure TC
	if agent.tcManager != nil {
		if err := agent.tcManager.UpdateShaping(&config.BWPolicy); err != nil {
			return fmt.Errorf("failed to update traffic shaping: %w", err)
		}
	}

	agent.logger.Printf("Network slice %s configured successfully", sliceID)
	return nil
}

// RunPerformanceTest executes a comprehensive performance test
func (agent *TNAgent) RunPerformanceTest(config *PerformanceTestConfig) (*PerformanceMetrics, error) {
	agent.logger.Printf("Running performance test: %s", config.TestID)

	startTime := time.Now()

	metrics := &PerformanceMetrics{
		Timestamp:   startTime,
		ClusterName: agent.config.ClusterName,
		TestID:      config.TestID,
		TestType:    config.TestType,
		QoSClass:    agent.config.QoSClass,
	}

	// Run throughput test
	if throughput, err := agent.runThroughputTest(config); err != nil {
		metrics.ErrorDetails = append(metrics.ErrorDetails, fmt.Sprintf("Throughput test failed: %v", err))
	} else {
		metrics.Throughput = *throughput
	}

	// Run latency test
	if latency, err := agent.runLatencyTest(config); err != nil {
		metrics.ErrorDetails = append(metrics.ErrorDetails, fmt.Sprintf("Latency test failed: %v", err))
	} else {
		metrics.Latency = *latency
	}

	// Calculate overhead metrics
	metrics.VXLANOverhead = agent.vxlanManager.CalculateVXLANOverhead(agent.config.VXLANConfig.MTU)
	metrics.TCOverhead = agent.tcManager.CalculateTCOverhead()

	// Get bandwidth utilization
	if usage, err := agent.tcManager.GetBandwidthUsage(); err == nil {
		totalBytes := usage["rx_bytes"] + usage["tx_bytes"]
		maxBytes := float64(agent.config.BWPolicy.DownlinkMbps) * 1024 * 1024 / 8 * metrics.Duration.Seconds()
		if maxBytes > 0 {
			metrics.BandwidthUtilization = (totalBytes / maxBytes) * 100
		}
	}

	metrics.Duration = time.Since(startTime)

	agent.logger.Printf("Performance test completed: %.2f Mbps throughput, %.2f ms latency",
		metrics.Throughput.AvgMbps, metrics.Latency.AvgRTTMs)

	return metrics, nil
}

// runThroughputTest runs throughput measurements
func (agent *TNAgent) runThroughputTest(config *PerformanceTestConfig) (*ThroughputMetrics, error) {
	// Determine target server
	targetIP := config.TargetCluster
	if targetIP == "" {
		// Use first remote peer
		if len(agent.config.VXLANConfig.RemoteIPs) > 0 {
			targetIP = agent.config.VXLANConfig.RemoteIPs[0]
		} else {
			return nil, fmt.Errorf("no target IP specified")
		}
	}

	// Use iperf port
	port := agent.config.MonitoringPort + 1

	return agent.iperfManager.RunThroughputTest(targetIP, port, config.Duration)
}

// runLatencyTest runs latency measurements
func (agent *TNAgent) runLatencyTest(config *PerformanceTestConfig) (*LatencyMetrics, error) {
	// Determine target server
	targetIP := config.TargetCluster
	if targetIP == "" {
		// Use first remote peer
		if len(agent.config.VXLANConfig.RemoteIPs) > 0 {
			targetIP = agent.config.VXLANConfig.RemoteIPs[0]
		} else {
			return nil, fmt.Errorf("no target IP specified")
		}
	}

	port := agent.config.MonitoringPort + 1

	return agent.iperfManager.RunLatencyTest(targetIP, port, config.Duration)
}

// GetStatus returns the current status of the agent
func (agent *TNAgent) GetStatus() (*TNStatus, error) {
	agent.mu.RLock()
	defer agent.mu.RUnlock()

	status := &TNStatus{
		Healthy:           agent.healthy,
		LastUpdate:        time.Now(),
		ActiveConnections: len(agent.config.VXLANConfig.RemoteIPs),
		BandwidthUsage:    make(map[string]float64),
	}

	// Get VXLAN status
	if agent.vxlanManager != nil {
		if vxlanStatus, err := agent.vxlanManager.GetTunnelStatus(); err == nil {
			status.VXLANStatus = *vxlanStatus
		}
	}

	// Get TC status
	if agent.tcManager != nil {
		if tcStatus, err := agent.tcManager.GetTCStatus(); err == nil {
			status.TCStatus = *tcStatus
		}

		// Get bandwidth usage
		if usage, err := agent.tcManager.GetBandwidthUsage(); err == nil {
			status.BandwidthUsage = usage
		}
	}

	return status, nil
}

// monitoringLoop runs continuous health monitoring
func (agent *TNAgent) monitoringLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-agent.ctx.Done():
			return
		case <-ticker.C:
			agent.performHealthCheck()
		}
	}
}

// performHealthCheck performs periodic health checks
func (agent *TNAgent) performHealthCheck() {
	agent.mu.Lock()
	defer agent.mu.Unlock()

	healthy := true

	// Check VXLAN tunnel
	if agent.vxlanManager != nil {
		if status, err := agent.vxlanManager.GetTunnelStatus(); err != nil || !status.TunnelUp {
			healthy = false
			agent.logger.Printf("VXLAN tunnel unhealthy: %v", err)
		}
	}

	// Check TC rules
	if agent.tcManager != nil {
		if status, err := agent.tcManager.GetTCStatus(); err != nil || !status.RulesActive {
			healthy = false
			agent.logger.Printf("TC rules unhealthy: %v", err)
		}
	}

	// Check iperf servers
	servers := agent.iperfManager.GetActiveServers()
	if len(servers) == 0 {
		healthy = false
		agent.logger.Println("No iperf servers running")
	}

	agent.healthy = healthy

	if !healthy {
		agent.logger.Println("TN Agent health check failed")
	}
}

// Export relevant types from manager package
type TNConfig struct {
	ClusterName     string            `json:"clusterName" yaml:"clusterName"`
	NetworkCIDR     string            `json:"networkCIDR" yaml:"networkCIDR"`
	VXLANConfig     VXLANConfig       `json:"vxlan" yaml:"vxlan"`
	BWPolicy        BandwidthPolicy   `json:"bandwidthPolicy" yaml:"bandwidthPolicy"`
	QoSClass        string            `json:"qosClass" yaml:"qosClass"`
	Interfaces      []NetworkInterface `json:"interfaces" yaml:"interfaces"`
	MonitoringPort  int               `json:"monitoringPort" yaml:"monitoringPort"`
}

type NetworkInterface struct {
	Name      string `json:"name" yaml:"name"`
	Type      string `json:"type" yaml:"type"`
	IP        string `json:"ip" yaml:"ip"`
	Netmask   string `json:"netmask" yaml:"netmask"`
	Gateway   string `json:"gateway" yaml:"gateway"`
	MTU       int    `json:"mtu" yaml:"mtu"`
	State     string `json:"state" yaml:"state"`
}

type PerformanceMetrics struct {
	Timestamp        time.Time       `json:"timestamp"`
	ClusterName      string          `json:"clusterName"`
	TestID           string          `json:"testId"`
	TestType         string          `json:"testType"`
	Duration         time.Duration   `json:"duration"`
	Throughput       ThroughputMetrics `json:"throughput"`
	Latency          LatencyMetrics    `json:"latency"`
	PacketLoss       float64         `json:"packetLoss"`
	Jitter           float64         `json:"jitter"`
	BandwidthUtilization float64     `json:"bandwidthUtilization"`
	QoSClass         string          `json:"qosClass"`
	VXLANOverhead    float64         `json:"vxlanOverhead"`
	TCOverhead       float64         `json:"tcOverhead"`
	NetworkPath      []string        `json:"networkPath"`
	ErrorDetails     []string        `json:"errorDetails,omitempty"`
}

type TNStatus struct {
	Healthy          bool                `json:"healthy"`
	LastUpdate       time.Time           `json:"lastUpdate"`
	ActiveConnections int                `json:"activeConnections"`
	BandwidthUsage   map[string]float64  `json:"bandwidthUsage"`
	VXLANStatus      VXLANStatus         `json:"vxlanStatus"`
	TCStatus         TCStatus            `json:"tcStatus"`
	ErrorMessages    []string            `json:"errorMessages,omitempty"`
}

type PerformanceTestConfig struct {
	TestID       string        `json:"testId"`
	SliceID      string        `json:"sliceId"`
	SliceType    string        `json:"sliceType"`
	Duration     time.Duration `json:"duration"`
	TestType     string        `json:"testType"`     // "iperf3", "ping", "custom"
	SourceCluster string       `json:"sourceCluster"`
	TargetCluster string       `json:"targetCluster"`
	Protocol     string        `json:"protocol"`     // "tcp", "udp"
	Parallel     int           `json:"parallel"`
	WindowSize   string        `json:"windowSize"`
	Interval     time.Duration `json:"interval"`
}