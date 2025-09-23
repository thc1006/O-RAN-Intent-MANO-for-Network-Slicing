package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
)

// BandwidthMonitor provides real-time bandwidth monitoring capabilities
type BandwidthMonitor struct {
	logger      *log.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	metrics     map[string]*InterfaceMetrics
	collectors  map[string]*MetricCollector
	interval    time.Duration
	running     bool
}

// InterfaceMetrics contains detailed interface statistics
type InterfaceMetrics struct {
	InterfaceName    string            `json:"interfaceName"`
	Timestamp        time.Time         `json:"timestamp"`
	RxBytes          int64             `json:"rxBytes"`
	TxBytes          int64             `json:"txBytes"`
	RxPackets        int64             `json:"rxPackets"`
	TxPackets        int64             `json:"txPackets"`
	RxErrors         int64             `json:"rxErrors"`
	TxErrors         int64             `json:"txErrors"`
	RxDropped        int64             `json:"rxDropped"`
	TxDropped        int64             `json:"txDropped"`
	RxRateMbps       float64           `json:"rxRateMbps"`
	TxRateMbps       float64           `json:"txRateMbps"`
	TotalRateMbps    float64           `json:"totalRateMbps"`
	Utilization      float64           `json:"utilization"`
	QueueLengths     map[string]int64  `json:"queueLengths"`
	PacketSize       float64           `json:"avgPacketSize"`
	ErrorRate        float64           `json:"errorRate"`
	DropRate         float64           `json:"dropRate"`
}

// MetricCollector collects metrics for a specific interface
type MetricCollector struct {
	interfaceName string
	lastMetrics   *InterfaceMetrics
	lastUpdate    time.Time
	logger        *log.Logger
}

// BandwidthSample represents a single bandwidth measurement
type BandwidthSample struct {
	Timestamp     time.Time `json:"timestamp"`
	Interface     string    `json:"interface"`
	RxMbps        float64   `json:"rxMbps"`
	TxMbps        float64   `json:"txMbps"`
	TotalMbps     float64   `json:"totalMbps"`
	Utilization   float64   `json:"utilization"`
	PacketLoss    float64   `json:"packetLoss"`
}

// NetworkStats contains aggregated network statistics
type NetworkStats struct {
	Timestamp       time.Time                    `json:"timestamp"`
	TotalRxMbps     float64                      `json:"totalRxMbps"`
	TotalTxMbps     float64                      `json:"totalTxMbps"`
	TotalMbps       float64                      `json:"totalMbps"`
	AvgUtilization  float64                      `json:"avgUtilization"`
	Interfaces      map[string]*InterfaceMetrics `json:"interfaces"`
	QueueStats      map[string]int64             `json:"queueStats"`
	TotalErrors     int64                        `json:"totalErrors"`
	TotalDropped    int64                        `json:"totalDropped"`
}

// NewBandwidthMonitor creates a new bandwidth monitor
func NewBandwidthMonitor(logger *log.Logger) *BandwidthMonitor {
	return &BandwidthMonitor{
		logger:     logger,
		metrics:    make(map[string]*InterfaceMetrics),
		collectors: make(map[string]*MetricCollector),
		interval:   5 * time.Second,
		running:    false,
	}
}

// Start begins bandwidth monitoring
func (bm *BandwidthMonitor) Start(ctx context.Context) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if bm.running {
		return fmt.Errorf("bandwidth monitor already running")
	}

	bm.ctx, bm.cancel = context.WithCancel(ctx)
	bm.running = true

	// Discover network interfaces
	interfaces, err := bm.discoverInterfaces()
	if err != nil {
		return fmt.Errorf("failed to discover interfaces: %w", err)
	}

	// Initialize collectors for each interface
	for _, iface := range interfaces {
		bm.collectors[iface] = &MetricCollector{
			interfaceName: iface,
			logger:        bm.logger,
		}
	}

	// Start monitoring goroutine
	go bm.monitoringLoop()

	bm.logger.Printf("Bandwidth monitor started, monitoring %d interfaces", len(interfaces))
	return nil
}

// Stop stops bandwidth monitoring
func (bm *BandwidthMonitor) Stop() error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if !bm.running {
		return nil
	}

	bm.cancel()
	bm.running = false

	bm.logger.Println("Bandwidth monitor stopped")
	return nil
}

// SetInterval sets the monitoring interval
func (bm *BandwidthMonitor) SetInterval(interval time.Duration) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.interval = interval
}

// discoverInterfaces discovers available network interfaces
func (bm *BandwidthMonitor) discoverInterfaces() ([]string, error) {
	cmd := exec.Command("ip", "link", "show")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list interfaces: %w", err)
	}

	var interfaces []string
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, ": ") && !strings.Contains(line, "LOOPBACK") {
			parts := strings.Split(line, ": ")
			if len(parts) >= 2 {
				ifaceName := strings.Split(parts[1], "@")[0] // Handle VLAN interfaces
				ifaceName = strings.Split(ifaceName, ":")[0]  // Handle additional formatting
				if ifaceName != "" && ifaceName != "lo" {
					// Validate interface name for security
					if err := security.ValidateNetworkInterface(ifaceName); err != nil {
						bm.logger.Printf("Warning: skipping invalid interface %s: %v", ifaceName, err)
						continue
					}
					interfaces = append(interfaces, ifaceName)
				}
			}
		}
	}

	return interfaces, nil
}

// monitoringLoop runs the main monitoring loop
func (bm *BandwidthMonitor) monitoringLoop() {
	ticker := time.NewTicker(bm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-bm.ctx.Done():
			return
		case <-ticker.C:
			bm.collectMetrics()
		}
	}
}

// collectMetrics collects metrics from all interfaces
func (bm *BandwidthMonitor) collectMetrics() {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	for ifaceName, collector := range bm.collectors {
		metrics, err := collector.collectInterfaceMetrics()
		if err != nil {
			bm.logger.Printf("Failed to collect metrics for %s: %v", ifaceName, err)
			continue
		}

		bm.metrics[ifaceName] = metrics
	}
}

// collectInterfaceMetrics collects metrics for a specific interface
func (mc *MetricCollector) collectInterfaceMetrics() (*InterfaceMetrics, error) {
	metrics := &InterfaceMetrics{
		InterfaceName: mc.interfaceName,
		Timestamp:     time.Now(),
		QueueLengths:  make(map[string]int64),
	}

	// Get basic interface statistics from /proc/net/dev
	if err := mc.getInterfaceStats(metrics); err != nil {
		return nil, fmt.Errorf("failed to get interface stats: %w", err)
	}

	// Calculate rates if we have previous metrics
	if mc.lastMetrics != nil {
		mc.calculateRates(metrics)
	}

	// Get queue statistics
	mc.getQueueStats(metrics)

	// Calculate derived metrics
	mc.calculateDerivedMetrics(metrics)

	// Store as last metrics for next calculation
	mc.lastMetrics = metrics
	mc.lastUpdate = metrics.Timestamp

	return metrics, nil
}

// getInterfaceStats gets basic interface statistics
func (mc *MetricCollector) getInterfaceStats(metrics *InterfaceMetrics) error {
	cmd := exec.Command("cat", "/proc/net/dev")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to read /proc/net/dev: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, mc.interfaceName+":") {
			// Parse the line: interface: rx_bytes rx_packets rx_errs rx_drop ... tx_bytes tx_packets tx_errs tx_drop
			parts := strings.Fields(line)
			if len(parts) >= 17 {
				if rxBytes, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					metrics.RxBytes = rxBytes
				}
				if rxPackets, err := strconv.ParseInt(parts[2], 10, 64); err == nil {
					metrics.RxPackets = rxPackets
				}
				if rxErrors, err := strconv.ParseInt(parts[3], 10, 64); err == nil {
					metrics.RxErrors = rxErrors
				}
				if rxDropped, err := strconv.ParseInt(parts[4], 10, 64); err == nil {
					metrics.RxDropped = rxDropped
				}
				if txBytes, err := strconv.ParseInt(parts[9], 10, 64); err == nil {
					metrics.TxBytes = txBytes
				}
				if txPackets, err := strconv.ParseInt(parts[10], 10, 64); err == nil {
					metrics.TxPackets = txPackets
				}
				if txErrors, err := strconv.ParseInt(parts[11], 10, 64); err == nil {
					metrics.TxErrors = txErrors
				}
				if txDropped, err := strconv.ParseInt(parts[12], 10, 64); err == nil {
					metrics.TxDropped = txDropped
				}
			}
			break
		}
	}

	return nil
}

// calculateRates calculates bandwidth rates
func (mc *MetricCollector) calculateRates(metrics *InterfaceMetrics) {
	if mc.lastMetrics == nil {
		return
	}

	timeDiff := metrics.Timestamp.Sub(mc.lastMetrics.Timestamp).Seconds()
	if timeDiff <= 0 {
		return
	}

	// Calculate byte rates
	rxBytesDiff := metrics.RxBytes - mc.lastMetrics.RxBytes
	txBytesDiff := metrics.TxBytes - mc.lastMetrics.TxBytes

	if rxBytesDiff >= 0 && txBytesDiff >= 0 {
		// Convert to Mbps (bytes/sec * 8 bits/byte / 1,000,000 bits/Mbps)
		metrics.RxRateMbps = float64(rxBytesDiff) / timeDiff * 8 / 1000000
		metrics.TxRateMbps = float64(txBytesDiff) / timeDiff * 8 / 1000000
		metrics.TotalRateMbps = metrics.RxRateMbps + metrics.TxRateMbps
	}
}

// getQueueStats gets queue statistics using tc
func (mc *MetricCollector) getQueueStats(metrics *InterfaceMetrics) {
	// Validate interface name for security
	if err := security.ValidateNetworkInterface(mc.interfaceName); err != nil {
		mc.logger.Printf("Warning: invalid interface name for queue stats: %v", err)
		return
	}

	cmd := exec.Command("tc", "-s", "qdisc", "show", "dev", mc.interfaceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// TC stats not available, skip
		return
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Sent") {
			// Parse line like: "Sent 1234 bytes 5678 pkt (dropped 9, overlimits 0 requeues 0)"
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "Sent" && i+1 < len(parts) {
					if bytes, err := strconv.ParseInt(parts[i+1], 10, 64); err == nil {
						metrics.QueueLengths["sent_bytes"] = bytes
					}
				}
				if part == "pkt" && i+1 < len(parts) {
					if packets, err := strconv.ParseInt(parts[i-1], 10, 64); err == nil {
						metrics.QueueLengths["sent_packets"] = packets
					}
				}
				if strings.Contains(part, "dropped") {
					dropStr := strings.Trim(part, "dropped(),")
					if dropped, err := strconv.ParseInt(dropStr, 10, 64); err == nil {
						metrics.QueueLengths["queue_dropped"] = dropped
					}
				}
			}
		}
	}
}

// calculateDerivedMetrics calculates derived metrics
func (mc *MetricCollector) calculateDerivedMetrics(metrics *InterfaceMetrics) {
	// Calculate average packet size
	if metrics.RxPackets+metrics.TxPackets > 0 {
		metrics.PacketSize = float64(metrics.RxBytes+metrics.TxBytes) / float64(metrics.RxPackets+metrics.TxPackets)
	}

	// Calculate error rates
	totalPackets := metrics.RxPackets + metrics.TxPackets
	if totalPackets > 0 {
		metrics.ErrorRate = float64(metrics.RxErrors+metrics.TxErrors) / float64(totalPackets) * 100
		metrics.DropRate = float64(metrics.RxDropped+metrics.TxDropped) / float64(totalPackets) * 100
	}

	// Estimate utilization (assuming 1 Gbps interface - would need actual link speed)
	linkSpeedMbps := 1000.0 // 1 Gbps default
	if metrics.TotalRateMbps > 0 {
		metrics.Utilization = (metrics.TotalRateMbps / linkSpeedMbps) * 100
		if metrics.Utilization > 100 {
			metrics.Utilization = 100
		}
	}
}

// GetCurrentMetrics returns current metrics for all interfaces
func (bm *BandwidthMonitor) GetCurrentMetrics() map[string]*InterfaceMetrics {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	metrics := make(map[string]*InterfaceMetrics)
	for k, v := range bm.metrics {
		metrics[k] = v
	}

	return metrics
}

// GetInterfaceMetrics returns metrics for a specific interface
func (bm *BandwidthMonitor) GetInterfaceMetrics(interfaceName string) (*InterfaceMetrics, error) {
	// Validate interface name for security
	if err := security.ValidateNetworkInterface(interfaceName); err != nil {
		return nil, fmt.Errorf("invalid interface name: %w", err)
	}

	bm.mu.RLock()
	defer bm.mu.RUnlock()

	metrics, exists := bm.metrics[interfaceName]
	if !exists {
		return nil, fmt.Errorf("no metrics available for interface %s", interfaceName)
	}

	return metrics, nil
}

// GetNetworkStats returns aggregated network statistics
func (bm *BandwidthMonitor) GetNetworkStats() *NetworkStats {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	stats := &NetworkStats{
		Timestamp:  time.Now(),
		Interfaces: make(map[string]*InterfaceMetrics),
		QueueStats: make(map[string]int64),
	}

	var totalUtilization float64
	interfaceCount := 0

	for ifaceName, metrics := range bm.metrics {
		stats.Interfaces[ifaceName] = metrics
		stats.TotalRxMbps += metrics.RxRateMbps
		stats.TotalTxMbps += metrics.TxRateMbps
		stats.TotalErrors += metrics.RxErrors + metrics.TxErrors
		stats.TotalDropped += metrics.RxDropped + metrics.TxDropped

		totalUtilization += metrics.Utilization
		interfaceCount++

		// Aggregate queue stats
		for queueName, queueValue := range metrics.QueueLengths {
			key := fmt.Sprintf("%s_%s", ifaceName, queueName)
			stats.QueueStats[key] = queueValue
		}
	}

	stats.TotalMbps = stats.TotalRxMbps + stats.TotalTxMbps

	if interfaceCount > 0 {
		stats.AvgUtilization = totalUtilization / float64(interfaceCount)
	}

	return stats
}

// GetBandwidthSample returns a bandwidth sample for a specific interface
func (bm *BandwidthMonitor) GetBandwidthSample(interfaceName string) (*BandwidthSample, error) {
	// Validate interface name for security
	if err := security.ValidateNetworkInterface(interfaceName); err != nil {
		return nil, fmt.Errorf("invalid interface name: %w", err)
	}

	metrics, err := bm.GetInterfaceMetrics(interfaceName)
	if err != nil {
		return nil, err
	}

	sample := &BandwidthSample{
		Timestamp:   metrics.Timestamp,
		Interface:   interfaceName,
		RxMbps:      metrics.RxRateMbps,
		TxMbps:      metrics.TxRateMbps,
		TotalMbps:   metrics.TotalRateMbps,
		Utilization: metrics.Utilization,
		PacketLoss:  metrics.DropRate,
	}

	return sample, nil
}

// StartContinuousLogging starts continuous logging of bandwidth metrics
func (bm *BandwidthMonitor) StartContinuousLogging(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-bm.ctx.Done():
				return
			case <-ticker.C:
				stats := bm.GetNetworkStats()
				bm.logger.Printf("Network Stats: Total=%.2f Mbps (RX=%.2f, TX=%.2f), Avg Util=%.1f%%, Errors=%d, Dropped=%d",
					stats.TotalMbps, stats.TotalRxMbps, stats.TotalTxMbps, stats.AvgUtilization, stats.TotalErrors, stats.TotalDropped)

				// Log per-interface details
				for ifaceName, metrics := range stats.Interfaces {
					if metrics.TotalRateMbps > 0.1 { // Only log active interfaces
						bm.logger.Printf("  %s: %.2f Mbps (RX=%.2f, TX=%.2f), Util=%.1f%%, Errors=%d, Dropped=%d",
							ifaceName, metrics.TotalRateMbps, metrics.RxRateMbps, metrics.TxRateMbps,
							metrics.Utilization, metrics.RxErrors+metrics.TxErrors, metrics.RxDropped+metrics.TxDropped)
					}
				}
			}
		}
	}()
}

// ExportMetrics exports metrics to JSON
func (bm *BandwidthMonitor) ExportMetrics() ([]byte, error) {
	stats := bm.GetNetworkStats()
	return json.MarshalIndent(stats, "", "  ")
}

// GetPerformanceSummary returns a performance summary for reporting
func (bm *BandwidthMonitor) GetPerformanceSummary() map[string]interface{} {
	stats := bm.GetNetworkStats()

	summary := map[string]interface{}{
		"timestamp":         stats.Timestamp,
		"total_throughput":  stats.TotalMbps,
		"rx_throughput":     stats.TotalRxMbps,
		"tx_throughput":     stats.TotalTxMbps,
		"avg_utilization":   stats.AvgUtilization,
		"total_errors":      stats.TotalErrors,
		"total_dropped":     stats.TotalDropped,
		"interface_count":   len(stats.Interfaces),
		"active_interfaces": make([]string, 0),
	}

	// List active interfaces
	for ifaceName, metrics := range stats.Interfaces {
		if metrics.TotalRateMbps > 0.1 {
			summary["active_interfaces"] = append(summary["active_interfaces"].([]string), ifaceName)
		}
	}

	return summary
}

// IsHealthy returns true if the monitor is running and collecting metrics
func (bm *BandwidthMonitor) IsHealthy() bool {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	if !bm.running {
		return false
	}

	// Check if we have recent metrics
	for _, metrics := range bm.metrics {
		if time.Since(metrics.Timestamp) < bm.interval*2 {
			return true
		}
	}

	return false
}