package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
)

// IperfManager manages iperf3 testing operations
type IperfManager struct {
	logger   *log.Logger
	servers  map[string]*IperfServer
	mu       sync.RWMutex
}

// IperfServer represents an iperf3 server instance
type IperfServer struct {
	Port     int             `json:"port"`
	PID      int             `json:"pid"`
	Started  time.Time       `json:"started"`
	Context  context.Context `json:"-"`
	Cancel   context.CancelFunc `json:"-"`
}

// IperfTestConfig defines parameters for iperf3 tests
type IperfTestConfig struct {
	ServerIP   string        `json:"serverIP"`
	Port       int           `json:"port"`
	Duration   time.Duration `json:"duration"`
	Protocol   string        `json:"protocol"`   // "tcp" or "udp"
	Bandwidth  string        `json:"bandwidth"`  // for UDP tests, e.g., "10M"
	Parallel   int           `json:"parallel"`   // number of parallel streams
	WindowSize string        `json:"windowSize"` // TCP window size
	Interval   time.Duration `json:"interval"`   // reporting interval
	Reverse    bool          `json:"reverse"`    // reverse mode (server sends)
	Bidir      bool          `json:"bidir"`      // bidirectional test
	JSON       bool          `json:"json"`       // JSON output
}

// IperfResult contains iperf3 test results
type IperfResult struct {
	TestID        string            `json:"testId"`
	Timestamp     time.Time         `json:"timestamp"`
	Duration      float64           `json:"duration"`
	Protocol      string            `json:"protocol"`
	Streams       []IperfStream     `json:"streams"`
	Summary       IperfSummary      `json:"summary"`
	ServerInfo    IperfServerInfo   `json:"serverInfo"`
	ErrorMessages []string          `json:"errorMessages,omitempty"`
	RawOutput     string            `json:"rawOutput,omitempty"`
}

// IperfStream represents individual stream results
type IperfStream struct {
	StreamID    int     `json:"streamId"`
	Bytes       int64   `json:"bytes"`
	BitsPerSec  float64 `json:"bitsPerSec"`
	Retransmits int     `json:"retransmits,omitempty"`
	RTTMin      float64 `json:"rttMin,omitempty"`
	RTTMax      float64 `json:"rttMax,omitempty"`
	RTTMean     float64 `json:"rttMean,omitempty"`
	RTTVar      float64 `json:"rttVar,omitempty"`
}

// IperfSummary contains aggregated test results
type IperfSummary struct {
	Sent          IperfStreamSummary `json:"sent"`
	Received      IperfStreamSummary `json:"received"`
	CPUUtil       IperfCPUUtil       `json:"cpuUtil"`
	LostPackets   int                `json:"lostPackets,omitempty"`
	LostPercent   float64            `json:"lostPercent,omitempty"`
	Jitter        float64            `json:"jitter,omitempty"`
}

// IperfStreamSummary contains summary for sent/received data
type IperfStreamSummary struct {
	Bytes      int64   `json:"bytes"`
	BitsPerSec float64 `json:"bitsPerSec"`
	MbitsPerSec float64 `json:"mbitsPerSec"`
	Retransmits int    `json:"retransmits,omitempty"`
}

// IperfCPUUtil contains CPU utilization information
type IperfCPUUtil struct {
	HostTotal   float64 `json:"hostTotal"`
	HostUser    float64 `json:"hostUser"`
	HostSystem  float64 `json:"hostSystem"`
	RemoteTotal float64 `json:"remoteTotal"`
	RemoteUser  float64 `json:"remoteUser"`
	RemoteSystem float64 `json:"remoteSystem"`
}

// IperfServerInfo contains server information
type IperfServerInfo struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Version  string `json:"version"`
	Platform string `json:"platform"`
}

// NewIperfManager creates a new iperf3 manager
func NewIperfManager(logger *log.Logger) *IperfManager {
	return &IperfManager{
		logger:  logger,
		servers: make(map[string]*IperfServer),
	}
}

// StartServer starts an iperf3 server on the specified port
func (im *IperfManager) StartServer(port int) error {
	// Validate port for security
	if err := security.ValidatePort(port); err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	im.mu.Lock()
	defer im.mu.Unlock()

	serverKey := fmt.Sprintf("port_%d", port)

	// Check if server is already running
	if server, exists := im.servers[serverKey]; exists {
		security.SafeLogf(im.logger, "Iperf3 server already running on port %d (PID: %d)", port, server.PID)
		return nil
	}

	security.SafeLogf(im.logger, "Starting iperf3 server on port %d", port)

	ctx, cancel := context.WithCancel(context.Background())

	// Start iperf3 server
	cmd := exec.CommandContext(ctx, "iperf3", "-s", "-p", strconv.Itoa(port), "-D")

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("failed to start iperf3 server: %w", err)
	}

	server := &IperfServer{
		Port:    port,
		PID:     cmd.Process.Pid,
		Started: time.Now(),
		Context: ctx,
		Cancel:  cancel,
	}

	im.servers[serverKey] = server

	// Wait a moment to ensure server is ready
	time.Sleep(1 * time.Second)

	// Verify server is listening
	if !im.isServerListening(port) {
		im.StopServer(port)
		return fmt.Errorf("iperf3 server failed to start listening on port %d", port)
	}

	security.SafeLogf(im.logger, "Iperf3 server started successfully on port %d (PID: %d)", port, server.PID)
	return nil
}

// StopServer stops the iperf3 server on the specified port
func (im *IperfManager) StopServer(port int) error {
	// Validate port for security
	if err := security.ValidatePort(port); err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	im.mu.Lock()
	defer im.mu.Unlock()

	serverKey := fmt.Sprintf("port_%d", port)

	server, exists := im.servers[serverKey]
	if !exists {
		return fmt.Errorf("no iperf3 server running on port %d", port)
	}

	security.SafeLogf(im.logger, "Stopping iperf3 server on port %d (PID: %d)", port, server.PID)

	// Cancel context to stop the server
	server.Cancel()

	// Kill the process if it's still running
	cmd := exec.Command("pkill", "-f", fmt.Sprintf("iperf3.*-p %d", port))
	if output, err := cmd.CombinedOutput(); err != nil {
		security.SafeLogf(im.logger, "Warning: failed to kill iperf3 server process: %s, output: %s", security.SanitizeErrorForLog(err), security.SanitizeForLog(string(output)))
	}

	delete(im.servers, serverKey)

	security.SafeLogf(im.logger, "Iperf3 server stopped on port %d", port)
	return nil
}

// isServerListening checks if a server is listening on the specified port
func (im *IperfManager) isServerListening(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 3*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// RunTest executes an iperf3 client test
func (im *IperfManager) RunTest(config *IperfTestConfig) (*IperfResult, error) {
	// Validate inputs for security
	if err := security.ValidateIPAddress(config.ServerIP); err != nil {
		return nil, fmt.Errorf("invalid server IP: %w", err)
	}
	if err := security.ValidatePort(config.Port); err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}
	if config.Duration > 3600*time.Second {
		return nil, fmt.Errorf("duration too long: %v (max: 1 hour)", config.Duration)
	}
	if config.Parallel < 1 || config.Parallel > 128 {
		return nil, fmt.Errorf("invalid parallel streams: %d (must be 1-128)", config.Parallel)
	}
	if config.Bandwidth != "" {
		if err := security.ValidateBandwidth(config.Bandwidth); err != nil {
			return nil, fmt.Errorf("invalid bandwidth: %w", err)
		}
	}

	security.SafeLogf(im.logger, "Running iperf3 test to %s:%d", security.SanitizeIPForLog(config.ServerIP), config.Port)

	testID := fmt.Sprintf("test_%d", time.Now().Unix())
	startTime := time.Now()

	// Build iperf3 command arguments
	args := []string{"-c", config.ServerIP, "-p", strconv.Itoa(config.Port)}

	// Test duration
	if config.Duration > 0 {
		args = append(args, "-t", strconv.Itoa(int(config.Duration.Seconds())))
	}

	// Protocol
	if strings.ToLower(config.Protocol) == "udp" {
		args = append(args, "-u")
		if config.Bandwidth != "" {
			args = append(args, "-b", config.Bandwidth)
		}
	}

	// Parallel streams
	if config.Parallel > 1 {
		args = append(args, "-P", strconv.Itoa(config.Parallel))
	}

	// Window size
	if config.WindowSize != "" {
		// Validate window size format (e.g., "64K", "1M")
		if err := security.ValidateCommandArgument(config.WindowSize); err != nil {
			return nil, fmt.Errorf("invalid window size: %w", err)
		}
		args = append(args, "-w", config.WindowSize)
	}

	// Reporting interval
	if config.Interval > 0 {
		args = append(args, "-i", strconv.Itoa(int(config.Interval.Seconds())))
	}

	// Reverse mode
	if config.Reverse {
		args = append(args, "-R")
	}

	// Bidirectional mode
	if config.Bidir {
		args = append(args, "--bidir")
	}

	// JSON output
	if config.JSON {
		args = append(args, "-J")
	}

	// Execute iperf3 client
	cmd := exec.Command("iperf3", args...)
	output, err := cmd.CombinedOutput()

	result := &IperfResult{
		TestID:    testID,
		Timestamp: startTime,
		Duration:  time.Since(startTime).Seconds(),
		Protocol:  config.Protocol,
		RawOutput: string(output),
	}

	if err != nil {
		result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("iperf3 command failed: %s", err))
		security.SafeLogError(im.logger, "Iperf3 test failed", err)
		return result, fmt.Errorf("iperf3 test failed: %w", err)
	}

	// Parse results based on output format
	if config.JSON {
		if err := im.parseJSONOutput(string(output), result); err != nil {
			result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("Failed to parse JSON output: %s", err))
			security.SafeLogError(im.logger, "Failed to parse iperf3 JSON output", err)
		}
	} else {
		if err := im.parseTextOutput(string(output), result); err != nil {
			result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("Failed to parse text output: %s", err))
			security.SafeLogError(im.logger, "Failed to parse iperf3 text output", err)
		}
	}

	security.SafeLogf(im.logger, "Iperf3 test completed: %.2f Mbps", result.Summary.Received.MbitsPerSec)
	return result, nil
}

// parseJSONOutput parses iperf3 JSON output
func (im *IperfManager) parseJSONOutput(output string, result *IperfResult) error {
	var jsonResult map[string]interface{}

	if err := json.Unmarshal([]byte(output), &jsonResult); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Extract summary information
	if end, ok := jsonResult["end"].(map[string]interface{}); ok {
		// Sum sent
		if sumSent, ok := end["sum_sent"].(map[string]interface{}); ok {
			result.Summary.Sent = IperfStreamSummary{
				Bytes:       int64(sumSent["bytes"].(float64)),
				BitsPerSec:  sumSent["bits_per_second"].(float64),
				MbitsPerSec: sumSent["bits_per_second"].(float64) / 1000000,
			}
			if retrans, ok := sumSent["retransmits"]; ok {
				result.Summary.Sent.Retransmits = int(retrans.(float64))
			}
		}

		// Sum received
		if sumReceived, ok := end["sum_received"].(map[string]interface{}); ok {
			result.Summary.Received = IperfStreamSummary{
				Bytes:       int64(sumReceived["bytes"].(float64)),
				BitsPerSec:  sumReceived["bits_per_second"].(float64),
				MbitsPerSec: sumReceived["bits_per_second"].(float64) / 1000000,
			}
		}

		// CPU utilization
		if cpuUtil, ok := end["cpu_utilization_percent"].(map[string]interface{}); ok {
			result.Summary.CPUUtil = IperfCPUUtil{
				HostTotal:   cpuUtil["host_total"].(float64),
				HostUser:    cpuUtil["host_user"].(float64),
				HostSystem:  cpuUtil["host_system"].(float64),
				RemoteTotal: cpuUtil["remote_total"].(float64),
				RemoteUser:  cpuUtil["remote_user"].(float64),
				RemoteSystem: cpuUtil["remote_system"].(float64),
			}
		}
	}

	// Extract server information
	if start, ok := jsonResult["start"].(map[string]interface{}); ok {
		if connecting, ok := start["connecting_to"].(map[string]interface{}); ok {
			result.ServerInfo.Host = connecting["host"].(string)
			result.ServerInfo.Port = int(connecting["port"].(float64))
		}
		if version, ok := start["version"]; ok {
			result.ServerInfo.Version = version.(string)
		}
	}

	return nil
}

// parseTextOutput parses iperf3 text output
func (im *IperfManager) parseTextOutput(output string, result *IperfResult) error {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse final summary line
		if strings.Contains(line, "sender") || strings.Contains(line, "receiver") {
			parts := strings.Fields(line)
			if len(parts) >= 7 {
				// Extract bandwidth (usually in format like "1.23 Mbits/sec")
				for i, part := range parts {
					if strings.Contains(part, "bits/sec") && i > 0 {
						if bw, err := strconv.ParseFloat(parts[i-1], 64); err == nil {
							if strings.Contains(line, "sender") {
								result.Summary.Sent.MbitsPerSec = bw
								result.Summary.Sent.BitsPerSec = bw * 1000000
							} else if strings.Contains(line, "receiver") {
								result.Summary.Received.MbitsPerSec = bw
								result.Summary.Received.BitsPerSec = bw * 1000000
							}
						}
						break
					}
				}
			}
		}

		// Parse server connection info
		if strings.Contains(line, "Connecting to host") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "host" && i+1 < len(parts) {
					result.ServerInfo.Host = strings.Trim(parts[i+1], ",")
				}
				if part == "port" && i+1 < len(parts) {
					if port, err := strconv.Atoi(parts[i+1]); err == nil {
						result.ServerInfo.Port = port
					}
				}
			}
		}
	}

	return nil
}

// RunThroughputTest runs a comprehensive throughput test
func (im *IperfManager) RunThroughputTest(serverIP string, port int, duration time.Duration) (*ThroughputMetrics, error) {
	// Validate inputs for security
	if err := security.ValidateIPAddress(serverIP); err != nil {
		return nil, fmt.Errorf("invalid server IP: %w", err)
	}
	if err := security.ValidatePort(port); err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}
	if duration > 3600*time.Second {
		return nil, fmt.Errorf("duration too long: %v (max: 1 hour)", duration)
	}

	security.SafeLogf(im.logger, "Running throughput test to %s:%d for %v", security.SanitizeIPForLog(serverIP), port, duration)

	metrics := &ThroughputMetrics{}

	// TCP Download test
	tcpConfig := &IperfTestConfig{
		ServerIP: serverIP,
		Port:     port,
		Duration: duration,
		Protocol: "tcp",
		JSON:     true,
		Parallel: 1,
	}

	tcpResult, err := im.RunTest(tcpConfig)
	if err != nil {
		return metrics, fmt.Errorf("TCP download test failed: %w", err)
	}

	metrics.DownlinkMbps = tcpResult.Summary.Received.MbitsPerSec

	// TCP Upload test (reverse)
	tcpConfig.Reverse = true
	tcpUpResult, err := im.RunTest(tcpConfig)
	if err != nil {
		security.SafeLogError(im.logger, "TCP upload test failed", err)
	} else {
		metrics.UplinkMbps = tcpUpResult.Summary.Sent.MbitsPerSec
	}

	// Bidirectional test
	tcpConfig.Reverse = false
	tcpConfig.Bidir = true
	bidirResult, err := im.RunTest(tcpConfig)
	if err != nil {
		security.SafeLogError(im.logger, "Bidirectional test failed", err)
	} else {
		metrics.BiDirMbps = bidirResult.Summary.Received.MbitsPerSec
	}

	// Calculate statistics
	metrics.AvgMbps = (metrics.DownlinkMbps + metrics.UplinkMbps) / 2
	metrics.PeakMbps = metrics.DownlinkMbps
	if metrics.UplinkMbps > metrics.PeakMbps {
		metrics.PeakMbps = metrics.UplinkMbps
	}
	metrics.MinMbps = metrics.DownlinkMbps
	if metrics.UplinkMbps < metrics.MinMbps {
		metrics.MinMbps = metrics.UplinkMbps
	}

	security.SafeLogf(im.logger, "Throughput test completed: DL=%.2f Mbps, UL=%.2f Mbps, Avg=%.2f Mbps",
		metrics.DownlinkMbps, metrics.UplinkMbps, metrics.AvgMbps)

	return metrics, nil
}

// RunLatencyTest runs a latency test using iperf3 with small packets
func (im *IperfManager) RunLatencyTest(serverIP string, port int, duration time.Duration) (*LatencyMetrics, error) {
	// Validate inputs for security
	if err := security.ValidateIPAddress(serverIP); err != nil {
		return nil, fmt.Errorf("invalid server IP: %w", err)
	}
	if err := security.ValidatePort(port); err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}
	if duration > 600*time.Second {
		return nil, fmt.Errorf("duration too long: %v (max: 10 minutes)", duration)
	}

	security.SafeLogf(im.logger, "Running latency test to %s:%d", security.SanitizeIPForLog(serverIP), port)

	metrics := &LatencyMetrics{}

	// Use ping for more accurate latency measurements
	cmd := exec.Command("ping", "-c", "10", "-i", "0.1", serverIP)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return metrics, fmt.Errorf("ping test failed: %w", err)
	}

	// Parse ping output
	lines := strings.Split(string(output), "\n")
	var rtts []float64

	for _, line := range lines {
		if strings.Contains(line, "time=") {
			parts := strings.Split(line, "time=")
			if len(parts) > 1 {
				timeStr := strings.Fields(parts[1])[0]
				if rtt, err := strconv.ParseFloat(timeStr, 64); err == nil {
					rtts = append(rtts, rtt)
				}
			}
		}
	}

	if len(rtts) > 0 {
		// Calculate statistics
		var sum, min, max float64
		min = rtts[0]
		max = rtts[0]

		for _, rtt := range rtts {
			sum += rtt
			if rtt < min {
				min = rtt
			}
			if rtt > max {
				max = rtt
			}
		}

		metrics.AvgRTTMs = sum / float64(len(rtts))
		metrics.MinRTTMs = min
		metrics.MaxRTTMs = max
		metrics.RTTMs = metrics.AvgRTTMs

		// Calculate standard deviation
		var variance float64
		for _, rtt := range rtts {
			variance += (rtt - metrics.AvgRTTMs) * (rtt - metrics.AvgRTTMs)
		}
		metrics.StdDevMs = variance / float64(len(rtts))

		// Approximate percentiles (simple implementation)
		if len(rtts) >= 2 {
			metrics.P50Ms = rtts[len(rtts)/2]
			metrics.P95Ms = rtts[int(float64(len(rtts))*0.95)]
			metrics.P99Ms = rtts[int(float64(len(rtts))*0.99)]
		}
	}

	security.SafeLogf(im.logger, "Latency test completed: Avg=%.2f ms, Min=%.2f ms, Max=%.2f ms",
		metrics.AvgRTTMs, metrics.MinRTTMs, metrics.MaxRTTMs)

	return metrics, nil
}

// GetActiveServers returns information about running iperf3 servers
func (im *IperfManager) GetActiveServers() map[string]*IperfServer {
	im.mu.RLock()
	defer im.mu.RUnlock()

	servers := make(map[string]*IperfServer)
	for k, v := range im.servers {
		servers[k] = v
	}

	return servers
}

// StopAllServers stops all running iperf3 servers
func (im *IperfManager) StopAllServers() error {
	im.mu.Lock()
	defer im.mu.Unlock()

	var errors []string

	for serverKey, server := range im.servers {
		if err := im.StopServer(server.Port); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to stop server %s: %v", serverKey, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors stopping servers: %s", strings.Join(errors, "; "))
	}

	return nil
}

// ThroughputMetrics contains detailed throughput measurements (from types.go)
type ThroughputMetrics struct {
	DownlinkMbps  float64 `json:"downlinkMbps"`
	UplinkMbps    float64 `json:"uplinkMbps"`
	BiDirMbps     float64 `json:"biDirMbps"`
	TargetMbps    float64 `json:"targetMbps"`
	AchievedRatio float64 `json:"achievedRatio"`
	PeakMbps      float64 `json:"peakMbps"`
	MinMbps       float64 `json:"minMbps"`
	AvgMbps       float64 `json:"avgMbps"`
	StdDevMbps    float64 `json:"stdDevMbps"`
}

// LatencyMetrics contains detailed latency measurements (from types.go)
type LatencyMetrics struct {
	RTTMs     float64 `json:"rttMs"`
	MinRTTMs  float64 `json:"minRttMs"`
	MaxRTTMs  float64 `json:"maxRttMs"`
	AvgRTTMs  float64 `json:"avgRttMs"`
	StdDevMs  float64 `json:"stdDevMs"`
	P50Ms     float64 `json:"p50Ms"`
	P95Ms     float64 `json:"p95Ms"`
	P99Ms     float64 `json:"p99Ms"`
	TargetMs  float64 `json:"targetMs"`
}