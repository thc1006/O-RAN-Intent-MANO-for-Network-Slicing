package pkg

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
)

// TCManager manages Traffic Control (TC) operations for bandwidth shaping
type TCManager struct {
	config *BandwidthPolicy
	iface  string
	logger *log.Logger
}

// NewTCManager creates a new Traffic Control manager
func NewTCManager(config *BandwidthPolicy, interfaceName string, logger *log.Logger) *TCManager {
	return &TCManager{
		config: config,
		iface:  interfaceName,
		logger: logger,
	}
}

// CleanRules removes all existing TC rules from the interface
func (tc *TCManager) CleanRules() error {
	// Validate interface name for security
	if err := security.ValidateNetworkInterface(tc.iface); err != nil {
		return fmt.Errorf("invalid interface name: %w", err)
	}

	security.SafeLogf(tc.logger, "Cleaning existing TC rules from interface %s", security.SanitizeForLog(tc.iface))

	// Use secure tc command execution
	tcArgs := []string{"qdisc", "del", "dev", tc.iface, "root"}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := security.SecureExecuteWithValidation(ctx, "tc", security.ValidateTCArgs, tcArgs...); err != nil {
		security.SafeLogError(tc.logger, "Note", err)
	}
	security.SafeLogf(tc.logger, "Cleaned TC rules from interface %s", security.SanitizeForLog(tc.iface))
	return nil
}

// ApplyShaping applies traffic shaping rules to the interface
func (tc *TCManager) ApplyShaping() error {
	// Validate interface name for security
	if err := security.ValidateNetworkInterface(tc.iface); err != nil {
		return fmt.Errorf("invalid interface name: %w", err)
	}

	// Validate bandwidth configuration
	if tc.config.DownlinkMbps <= 0 || tc.config.DownlinkMbps > 100000 {
		return fmt.Errorf("invalid downlink bandwidth: %.2f Mbps", tc.config.DownlinkMbps)
	}

	security.SafeLogf(tc.logger, "Applying traffic shaping to interface %s", security.SanitizeForLog(tc.iface))
	if err := tc.CleanRules(); err != nil {
		security.SafeLogError(tc.logger, "Warning: failed to clean existing rules", err)
	}
	totalKbps := int(tc.config.DownlinkMbps * 1024)

	// Use secure tc command execution
	tcArgs := []string{"qdisc", "add", "dev", tc.iface, "root", "handle", "1:", "htb", "default", "30"}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := security.SecureExecuteWithValidation(ctx, "tc", security.ValidateTCArgs, tcArgs...); err != nil {
		return fmt.Errorf("failed to create root qdisc: %v", err)
	}
	security.SafeLogf(tc.logger, "Traffic shaping applied successfully to %s (rate: %d kbps)", security.SanitizeForLog(tc.iface), totalKbps)
	return nil
}

// MonitorBandwidth continuously monitors bandwidth usage
func (tc *TCManager) MonitorBandwidth(interval time.Duration, stopCh <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-stopCh:
			tc.logger.Println("Stopping bandwidth monitoring")
			return
		case <-ticker.C:
			security.SafeLogf(tc.logger, "Monitoring bandwidth on interface %s", security.SanitizeForLog(tc.iface))
		}
	}
}

// UpdateShaping updates the traffic shaping configuration
func (tc *TCManager) UpdateShaping(newConfig *BandwidthPolicy) error {
	security.SafeLogf(tc.logger, "Updating traffic shaping configuration")
	if err := tc.CleanRules(); err != nil {
		return fmt.Errorf("failed to clean existing rules: %w", err)
	}
	tc.config = newConfig
	return tc.ApplyShaping()
}

// CalculateTCOverhead estimates the overhead introduced by TC processing
func (tc *TCManager) CalculateTCOverhead() float64 {
	overhead := 2.0
	if tc.config.LatencyMs > 0 || tc.config.JitterMs > 0 || tc.config.LossPercent > 0 {
		overhead += 3.0
	}
	overhead += 1.5
	return overhead
}

// GetBandwidthUsage returns current bandwidth utilization
func (tc *TCManager) GetBandwidthUsage() (map[string]float64, error) {
	usage := make(map[string]float64)

	// Use secure file reading
	filePath := "/proc/net/dev"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, err := security.SecureExecute(ctx, "cat", filePath)
	if err != nil {
		return usage, fmt.Errorf("failed to read network statistics: %v", err)
	}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, tc.iface+":") {
			fields := strings.Fields(line)
			if len(fields) >= 10 {
				if rxBytes, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
					usage["rx_bytes"] = float64(rxBytes)
				}
				if txBytes, err := strconv.ParseInt(fields[9], 10, 64); err == nil {
					usage["tx_bytes"] = float64(txBytes)
				}
			}
			break
		}
	}
	return usage, nil
}

// GetTCStatus returns the current TC configuration and statistics
func (tc *TCManager) GetTCStatus() (*TCStatus, error) {
	// Validate interface name for security
	if err := security.ValidateNetworkInterface(tc.iface); err != nil {
		return nil, fmt.Errorf("invalid interface name: %w", err)
	}

	status := &TCStatus{
		RulesActive:   false,
		QueueStats:    make(map[string]int64),
		ShapingActive: false,
		Interfaces:    []string{tc.iface},
	}
	// Use secure tc command execution
	tcArgs := []string{"qdisc", "show", "dev", tc.iface}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	output, err := security.SecureExecuteWithValidation(ctx, "tc", security.ValidateTCArgs, tcArgs...)
	if err != nil {
		return status, fmt.Errorf("failed to get qdisc info: %v", err)
	}
	outputStr := string(output)
	if strings.Contains(outputStr, "htb") {
		status.RulesActive = true
		status.ShapingActive = true
	}
	return status, nil
}

// BandwidthPolicy represents bandwidth policy configuration
type BandwidthPolicy struct {
	DownlinkMbps float64 `json:"downlinkMbps" yaml:"downlinkMbps"`
	UplinkMbps   float64 `json:"uplinkMbps" yaml:"uplinkMbps"`
	LatencyMs    float64 `json:"latencyMs" yaml:"latencyMs"`
	JitterMs     float64 `json:"jitterMs" yaml:"jitterMs"`
	LossPercent  float64 `json:"lossPercent" yaml:"lossPercent"`
	Priority     int     `json:"priority" yaml:"priority"`
	QueueClass   string  `json:"queueClass" yaml:"queueClass"`
	Burst        string    `json:"burst" yaml:"burst"`
	Filters      []Filter  `json:"filters" yaml:"filters"`
}

// TCStatus represents TC status information
type TCStatus struct {
	RulesActive   bool             `json:"rulesActive"`
	QueueStats    map[string]int64 `json:"queueStats"`
	ShapingActive bool             `json:"shapingActive"`
	Interfaces    []string         `json:"interfaces"`
}

// ThesisValidation represents thesis validation metrics
type ThesisValidation struct {
	DeployTimeMs        int64     `json:"deployTimeMs"`
	DLThroughputMbps    float64   `json:"dlThroughputMbps"`
	PingRTTMs           float64   `json:"pingRTTMs"`
	TCOverheadPercent   float64   `json:"tcOverheadPercent"`
	CompliancePercent   float64   `json:"compliancePercent"`
	ThroughputResults   []float64 `json:"throughputResults"`
	ThroughputTargets   []float64 `json:"throughputTargets"`
	RTTResults          []float64 `json:"rttResults"`
	RTTTargets          []float64 `json:"rttTargets"`
}

// Filter represents a traffic classification filter
type Filter struct {
	Protocol   string `json:"protocol" yaml:"protocol"`
	SrcIP      string `json:"srcIP" yaml:"srcIP"`
	DstIP      string `json:"dstIP" yaml:"dstIP"`
	SrcPort    int    `json:"srcPort" yaml:"srcPort"`
	DstPort    int    `json:"dstPort" yaml:"dstPort"`
	FlowID     string `json:"flowID" yaml:"flowID"`
	Action     string `json:"action" yaml:"action"`
	ClassID    string `json:"classID" yaml:"classID"`
	Priority   int    `json:"priority" yaml:"priority"`
}
