package pkg

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
)

// QoSManager manages Quality of Service policies and enforcement
type QoSManager struct {
	logger            *log.Logger
	strategies        map[string]*QoSStrategy
	violations        []QoSViolation
	complianceHistory map[string][]ComplianceRecord
	mutex             sync.RWMutex
}

// NewQoSManager creates a new QoS manager
func NewQoSManager(logger *log.Logger) *QoSManager {
	return &QoSManager{
		logger:            logger,
		strategies:        make(map[string]*QoSStrategy),
		violations:        make([]QoSViolation, 0),
		complianceHistory: make(map[string][]ComplianceRecord),
	}
}

// ValidateStrategy validates a QoS strategy configuration
func (qm *QoSManager) ValidateStrategy(strategy *QoSStrategy) error {
	if strategy == nil {
		return fmt.Errorf("strategy cannot be nil")
	}

	// Validate strategy type
	switch strategy.Type {
	case QoSStrategyTypeULLC, QoSStrategyTypeEMBB, QoSStrategyTypeMIOT, QoSStrategyTypeCustom:
		// Valid types
	default:
		return fmt.Errorf("invalid strategy type: %s", strategy.Type)
	}

	// Validate bandwidth limits
	for direction, limit := range strategy.BandwidthLimits {
		if err := qm.validateBandwidthLimit(direction, limit); err != nil {
			return fmt.Errorf("invalid bandwidth limit for %s: %w", direction, err)
		}
	}

	// Validate latency targets
	for metric, target := range strategy.LatencyTargets {
		if target <= 0 {
			return fmt.Errorf("invalid latency target for %s: %f", metric, target)
		}
	}

	// Validate traffic classes
	for i, tc := range strategy.TrafficClasses {
		if err := qm.validateTrafficClass(&tc); err != nil {
			return fmt.Errorf("invalid traffic class %d: %w", i, err)
		}
	}

	// Validate scheduling policy
	if err := qm.validateSchedulingPolicy(&strategy.SchedulingPolicy); err != nil {
		return fmt.Errorf("invalid scheduling policy: %w", err)
	}

	return nil
}

// GenerateClusterConfig generates cluster-specific QoS configuration
func (qm *QoSManager) GenerateClusterConfig(strategy *QoSStrategy, clusterName string) *ClusterQoSConfig {
	config := &ClusterQoSConfig{
		ClusterName:       clusterName,
		Strategy:          strategy,
		GeneratedAt:       time.Now(),
		TrafficControlRules: make([]TCRule, 0),
		InterfaceConfigs:  make(map[string]InterfaceQoSConfig),
	}

	// Generate traffic control rules based on strategy
	config.TrafficControlRules = qm.generateTCRules(strategy)

	// Generate interface-specific configurations
	config.InterfaceConfigs = qm.generateInterfaceConfigs(strategy)

	// Add cluster-specific optimizations
	qm.applyClusterOptimizations(config, clusterName)

	return config
}

// ApplyUpdates applies updates to an existing QoS strategy
func (qm *QoSManager) ApplyUpdates(current *QoSStrategy, updates *QoSUpdates) *QoSStrategy {
	updated := *current

	// Apply bandwidth changes
	if updates.BandwidthChanges != nil {
		if updated.BandwidthLimits == nil {
			updated.BandwidthLimits = make(map[string]string)
		}
		for direction, limit := range updates.BandwidthChanges {
			updated.BandwidthLimits[direction] = limit
		}
	}

	// Apply latency changes
	if updates.LatencyChanges != nil {
		if updated.LatencyTargets == nil {
			updated.LatencyTargets = make(map[string]float64)
		}
		for metric, target := range updates.LatencyChanges {
			updated.LatencyTargets[metric] = target
		}
	}

	// Apply priority changes
	if updates.PriorityChanges != nil {
		for className, priority := range updates.PriorityChanges {
			for i := range updated.TrafficClasses {
				if updated.TrafficClasses[i].Name == className {
					updated.TrafficClasses[i].Priority = priority
					break
				}
			}
		}
	}

	// Add new traffic classes
	if updates.AddTrafficClasses != nil {
		updated.TrafficClasses = append(updated.TrafficClasses, updates.AddTrafficClasses...)
	}

	// Remove traffic classes
	if updates.RemoveTrafficClasses != nil {
		removeMap := make(map[string]bool)
		for _, name := range updates.RemoveTrafficClasses {
			removeMap[name] = true
		}

		var filteredClasses []TrafficClass
		for _, tc := range updated.TrafficClasses {
			if !removeMap[tc.Name] {
				filteredClasses = append(filteredClasses, tc)
			}
		}
		updated.TrafficClasses = filteredClasses
	}

	// Update scheduling policy
	if updates.UpdateScheduling != nil {
		updated.SchedulingPolicy = *updates.UpdateScheduling
	}

	return &updated
}

// AdjustForLatency adjusts QoS strategy to compensate for latency issues
func (qm *QoSManager) AdjustForLatency(strategy *QoSStrategy, faultDetails map[string]interface{}) *QoSStrategy {
	adjusted := *strategy

	// Increase priority for latency-sensitive traffic
	for i := range adjusted.TrafficClasses {
		tc := &adjusted.TrafficClasses[i]
		if qm.isLatencySensitive(tc) {
			tc.Priority = min(tc.Priority+1, 10) // Max priority is 10
		}
	}

	// Adjust scheduling to favor low-latency traffic
	if adjusted.SchedulingPolicy.Algorithm == "fair" {
		adjusted.SchedulingPolicy.Algorithm = "priority"
	}

	// Reduce buffer sizes to minimize queuing delay
	for i := range adjusted.SchedulingPolicy.Queues {
		queue := &adjusted.SchedulingPolicy.Queues[i]
		if currentBurst, ok := queue.BurstSize.(string); ok {
			// Reduce burst size by 50%
			queue.BurstSize = qm.reduceBurstSize(currentBurst, 0.5)
		}
	}

	return &adjusted
}

// GetComplianceSummary returns QoS compliance summary
func (qm *QoSManager) GetComplianceSummary() *QoSComplianceSummary {
	qm.mutex.RLock()
	defer qm.mutex.RUnlock()

	summary := &QoSComplianceSummary{
		SliceCompliance: make(map[string]float64),
		Violations:      make([]QoSViolation, len(qm.violations)),
		LastUpdated:     time.Now(),
	}

	copy(summary.Violations, qm.violations)

	// Calculate overall compliance
	totalSlices := len(qm.complianceHistory)
	if totalSlices > 0 {
		var totalCompliance float64
		for sliceID, records := range qm.complianceHistory {
			if len(records) > 0 {
				// Use most recent compliance record
				recentCompliance := records[len(records)-1].CompliancePercent
				summary.SliceCompliance[sliceID] = recentCompliance
				totalCompliance += recentCompliance
			}
		}
		summary.OverallCompliance = totalCompliance / float64(totalSlices)
	}

	return summary
}

// RecordViolation records a QoS violation
func (qm *QoSManager) RecordViolation(violation QoSViolation) {
	qm.mutex.Lock()
	defer qm.mutex.Unlock()

	qm.violations = append(qm.violations, violation)

	// Keep only recent violations (last 1000)
	if len(qm.violations) > 1000 {
		qm.violations = qm.violations[len(qm.violations)-1000:]
	}

	security.SafeLogf(qm.logger, "QoS violation recorded for slice %s: %s",
		security.SanitizeForLog(violation.SliceID), security.SanitizeForLog(violation.MetricType))
}

// RecordCompliance records compliance metrics for a slice
func (qm *QoSManager) RecordCompliance(sliceID string, compliancePercent float64) {
	qm.mutex.Lock()
	defer qm.mutex.Unlock()

	record := ComplianceRecord{
		Timestamp:         time.Now(),
		CompliancePercent: compliancePercent,
	}

	if qm.complianceHistory[sliceID] == nil {
		qm.complianceHistory[sliceID] = make([]ComplianceRecord, 0)
	}

	qm.complianceHistory[sliceID] = append(qm.complianceHistory[sliceID], record)

	// Keep only recent records (last 100 per slice)
	if len(qm.complianceHistory[sliceID]) > 100 {
		qm.complianceHistory[sliceID] = qm.complianceHistory[sliceID][len(qm.complianceHistory[sliceID])-100:]
	}
}

// Helper methods

func (qm *QoSManager) validateBandwidthLimit(direction, limit string) error {
	validDirections := map[string]bool{
		"uplink":   true,
		"downlink": true,
		"bidirectional": true,
	}

	if !validDirections[direction] {
		return fmt.Errorf("invalid direction: %s", direction)
	}

	// Validate bandwidth format (e.g., "100Mbps", "1Gbps")
	if !qm.isValidBandwidthFormat(limit) {
		return fmt.Errorf("invalid bandwidth format: %s", limit)
	}

	return nil
}

func (qm *QoSManager) validateTrafficClass(tc *TrafficClass) error {
	if tc.Name == "" {
		return fmt.Errorf("traffic class name cannot be empty")
	}

	if tc.Priority < 0 || tc.Priority > 10 {
		return fmt.Errorf("priority must be between 0 and 10")
	}

	if tc.Latency < 0 {
		return fmt.Errorf("latency cannot be negative")
	}

	// Validate selector
	if err := qm.validateTrafficSelector(&tc.Selector); err != nil {
		return fmt.Errorf("invalid selector: %w", err)
	}

	// Validate actions
	for i, action := range tc.Actions {
		if err := qm.validateTrafficAction(&action); err != nil {
			return fmt.Errorf("invalid action %d: %w", i, err)
		}
	}

	return nil
}

func (qm *QoSManager) validateTrafficSelector(selector *TrafficSelector) error {
	// Validate protocol
	if selector.Protocol != "" {
		validProtocols := map[string]bool{
			"tcp": true, "udp": true, "icmp": true, "sctp": true,
		}
		if !validProtocols[selector.Protocol] {
			return fmt.Errorf("invalid protocol: %s", selector.Protocol)
		}
	}

	// Validate DSCP
	if selector.DSCP < 0 || selector.DSCP > 63 {
		return fmt.Errorf("DSCP must be between 0 and 63")
	}

	return nil
}

func (qm *QoSManager) validateTrafficAction(action *TrafficAction) error {
	validActions := map[string]bool{
		"mark": true, "police": true, "shape": true, "drop": true,
	}

	if !validActions[action.Type] {
		return fmt.Errorf("invalid action type: %s", action.Type)
	}

	return nil
}

func (qm *QoSManager) validateSchedulingPolicy(policy *SchedulingPolicy) error {
	validAlgorithms := map[string]bool{
		"fifo": true, "fair": true, "priority": true, "cbq": true,
	}

	if !validAlgorithms[policy.Algorithm] {
		return fmt.Errorf("invalid scheduling algorithm: %s", policy.Algorithm)
	}

	// Validate queues
	for i, queue := range policy.Queues {
		if err := qm.validateQueueConfig(&queue); err != nil {
			return fmt.Errorf("invalid queue %d: %w", i, err)
		}
	}

	return nil
}

func (qm *QoSManager) validateQueueConfig(queue *QueueConfig) error {
	if queue.ID == "" {
		return fmt.Errorf("queue ID cannot be empty")
	}

	if queue.Weight < 0 {
		return fmt.Errorf("queue weight cannot be negative")
	}

	if queue.Priority < 0 || queue.Priority > 10 {
		return fmt.Errorf("queue priority must be between 0 and 10")
	}

	return nil
}

func (qm *QoSManager) generateTCRules(strategy *QoSStrategy) []TCRule {
	var rules []TCRule

	// Generate rules for each traffic class
	for _, tc := range strategy.TrafficClasses {
		rule := TCRule{
			Name:     tc.Name,
			Priority: tc.Priority,
			Selector: TCSelector{
				Protocol:   tc.Selector.Protocol,
				SourceIP:   tc.Selector.SourceIP,
				DestIP:     tc.Selector.DestIP,
				SourcePort: tc.Selector.SourcePort,
				DestPort:   tc.Selector.DestPort,
				DSCP:       tc.Selector.DSCP,
			},
			Actions: make([]TCAction, 0),
		}

		// Convert traffic actions to TC actions
		for _, action := range tc.Actions {
			tcAction := TCAction{
				Type:       action.Type,
				Parameters: action.Parameters,
			}
			rule.Actions = append(rule.Actions, tcAction)
		}

		rules = append(rules, rule)
	}

	return rules
}

func (qm *QoSManager) generateInterfaceConfigs(strategy *QoSStrategy) map[string]InterfaceQoSConfig {
	configs := make(map[string]InterfaceQoSConfig)

	// Generate default interface configuration
	defaultConfig := InterfaceQoSConfig{
		Interface:        "default",
		SchedulingPolicy: strategy.SchedulingPolicy,
		BandwidthLimits:  strategy.BandwidthLimits,
		Queues:          make([]QueueConfig, len(strategy.SchedulingPolicy.Queues)),
	}

	copy(defaultConfig.Queues, strategy.SchedulingPolicy.Queues)
	configs["default"] = defaultConfig

	return configs
}

func (qm *QoSManager) applyClusterOptimizations(config *ClusterQoSConfig, clusterName string) {
	// Apply cluster-specific optimizations based on cluster characteristics
	// This could include hardware-specific tuning, network topology optimizations, etc.

	// For now, add basic optimizations
	config.Optimizations = map[string]interface{}{
		"cluster_name":     clusterName,
		"optimization_level": "standard",
		"tuned_for":        "low_latency",
	}
}

func (qm *QoSManager) isLatencySensitive(tc *TrafficClass) bool {
	// Determine if a traffic class is latency-sensitive
	return tc.Latency > 0 && tc.Latency < 10.0 // Less than 10ms is considered latency-sensitive
}

func (qm *QoSManager) reduceBurstSize(currentBurst string, factor float64) string {
	// Parse and reduce burst size
	// This is a simplified implementation
	return fmt.Sprintf("%.0fkb", float64(1024)*factor) // Default to reduced size
}

func (qm *QoSManager) isValidBandwidthFormat(bandwidth string) bool {
	// Validate bandwidth format (simplified)
	validSuffixes := []string{"bps", "kbps", "Mbps", "Gbps"}
	for _, suffix := range validSuffixes {
		if len(bandwidth) > len(suffix) && bandwidth[len(bandwidth)-len(suffix):] == suffix {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Supporting types for QoS management

// ClusterQoSConfig represents QoS configuration for a specific cluster
type ClusterQoSConfig struct {
	ClusterName         string                         `json:"clusterName"`
	Strategy            *QoSStrategy                   `json:"strategy"`
	TrafficControlRules []TCRule                       `json:"trafficControlRules"`
	InterfaceConfigs    map[string]InterfaceQoSConfig  `json:"interfaceConfigs"`
	Optimizations       map[string]interface{}         `json:"optimizations"`
	GeneratedAt         time.Time                      `json:"generatedAt"`
}

// TCRule represents a traffic control rule
type TCRule struct {
	Name     string      `json:"name"`
	Priority int         `json:"priority"`
	Selector TCSelector  `json:"selector"`
	Actions  []TCAction  `json:"actions"`
}

// TCSelector represents traffic selection criteria for TC
type TCSelector struct {
	Protocol   string `json:"protocol,omitempty"`
	SourceIP   string `json:"sourceIp,omitempty"`
	DestIP     string `json:"destIp,omitempty"`
	SourcePort int    `json:"sourcePort,omitempty"`
	DestPort   int    `json:"destPort,omitempty"`
	DSCP       int    `json:"dscp,omitempty"`
}

// TCAction represents a traffic control action
type TCAction struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
}

// InterfaceQoSConfig represents QoS configuration for a network interface
type InterfaceQoSConfig struct {
	Interface        string                 `json:"interface"`
	SchedulingPolicy SchedulingPolicy       `json:"schedulingPolicy"`
	BandwidthLimits  map[string]string      `json:"bandwidthLimits"`
	Queues           []QueueConfig          `json:"queues"`
	Options          map[string]interface{} `json:"options,omitempty"`
}

// ComplianceRecord represents a compliance measurement record
type ComplianceRecord struct {
	Timestamp         time.Time `json:"timestamp"`
	CompliancePercent float64   `json:"compliancePercent"`
}