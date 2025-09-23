package pkg

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusMetrics contains all Prometheus metrics for the TN agent
type PrometheusMetrics struct {
	// Network throughput metrics
	ThroughputMbps *prometheus.GaugeVec

	// Latency metrics
	LatencyMs *prometheus.GaugeVec

	// Packet metrics
	PacketsTotal *prometheus.CounterVec
	PacketsDropped *prometheus.CounterVec
	PacketsErrors *prometheus.CounterVec

	// VXLAN metrics
	VXLANTunnelUp *prometheus.GaugeVec
	VXLANOverhead *prometheus.GaugeVec

	// TC metrics
	TCRulesActive *prometheus.GaugeVec
	TCOverhead *prometheus.GaugeVec
	BandwidthUtilization *prometheus.GaugeVec

	// Test metrics
	TestDuration *prometheus.HistogramVec
	TestSuccess *prometheus.CounterVec
	TestFailure *prometheus.CounterVec

	// Agent health metrics
	AgentUp *prometheus.GaugeVec
	AgentConnections *prometheus.GaugeVec

	// Performance target compliance
	ThesisCompliance *prometheus.GaugeVec
	SLACompliance *prometheus.GaugeVec
}

// NewPrometheusMetrics creates new Prometheus metrics
func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		ThroughputMbps: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tn_throughput_mbps",
				Help: "Network throughput in Mbps",
			},
			[]string{"cluster", "interface", "direction", "slice_type"},
		),

		LatencyMs: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tn_latency_ms",
				Help: "Network latency in milliseconds",
			},
			[]string{"cluster", "target", "slice_type"},
		),

		PacketsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tn_packets_total",
				Help: "Total number of network packets",
			},
			[]string{"cluster", "interface", "direction"},
		),

		PacketsDropped: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tn_packets_dropped_total",
				Help: "Total number of dropped packets",
			},
			[]string{"cluster", "interface", "reason"},
		),

		PacketsErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tn_packets_errors_total",
				Help: "Total number of packet errors",
			},
			[]string{"cluster", "interface", "error_type"},
		),

		VXLANTunnelUp: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tn_vxlan_tunnel_up",
				Help: "VXLAN tunnel status (1 = up, 0 = down)",
			},
			[]string{"cluster", "tunnel_id", "remote_ip"},
		),

		VXLANOverhead: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tn_vxlan_overhead_percent",
				Help: "VXLAN encapsulation overhead percentage",
			},
			[]string{"cluster", "tunnel_id"},
		),

		TCRulesActive: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tn_tc_rules_active",
				Help: "Number of active TC rules",
			},
			[]string{"cluster", "interface"},
		),

		TCOverhead: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tn_tc_overhead_percent",
				Help: "TC processing overhead percentage",
			},
			[]string{"cluster", "interface"},
		),

		BandwidthUtilization: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tn_bandwidth_utilization_percent",
				Help: "Bandwidth utilization percentage",
			},
			[]string{"cluster", "interface", "qos_class"},
		),

		TestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "tn_test_duration_seconds",
				Help: "Performance test duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"cluster", "test_type", "slice_type"},
		),

		TestSuccess: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tn_test_success_total",
				Help: "Total number of successful tests",
			},
			[]string{"cluster", "test_type", "slice_type"},
		),

		TestFailure: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tn_test_failure_total",
				Help: "Total number of failed tests",
			},
			[]string{"cluster", "test_type", "slice_type"},
		),

		AgentUp: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tn_agent_up",
				Help: "TN agent status (1 = up, 0 = down)",
			},
			[]string{"cluster"},
		),

		AgentConnections: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tn_agent_connections",
				Help: "Number of active connections",
			},
			[]string{"cluster"},
		),

		ThesisCompliance: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tn_thesis_compliance_percent",
				Help: "Thesis target compliance percentage",
			},
			[]string{"cluster", "metric_type"},
		),

		SLACompliance: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tn_sla_compliance",
				Help: "SLA compliance status (1 = compliant, 0 = non-compliant)",
			},
			[]string{"cluster", "slice_type"},
		),
	}
}

// UpdateThroughputMetrics updates throughput metrics
func (pm *PrometheusMetrics) UpdateThroughputMetrics(clusterName string, metrics *ThroughputMetrics, sliceType string) {
	pm.ThroughputMbps.WithLabelValues(clusterName, "total", "downlink", sliceType).Set(metrics.DownlinkMbps)
	pm.ThroughputMbps.WithLabelValues(clusterName, "total", "uplink", sliceType).Set(metrics.UplinkMbps)
	pm.ThroughputMbps.WithLabelValues(clusterName, "total", "average", sliceType).Set(metrics.AvgMbps)
	pm.ThroughputMbps.WithLabelValues(clusterName, "total", "peak", sliceType).Set(metrics.PeakMbps)
}

// UpdateLatencyMetrics updates latency metrics
func (pm *PrometheusMetrics) UpdateLatencyMetrics(clusterName string, metrics *LatencyMetrics, target, sliceType string) {
	pm.LatencyMs.WithLabelValues(clusterName, target, sliceType).Set(metrics.AvgRTTMs)
}

// UpdateVXLANMetrics updates VXLAN metrics
func (pm *PrometheusMetrics) UpdateVXLANMetrics(clusterName string, status *VXLANStatus, overhead float64) {
	tunnelStatus := 0.0
	if status.TunnelUp {
		tunnelStatus = 1.0
	}

	for _, peer := range status.RemotePeers {
		pm.VXLANTunnelUp.WithLabelValues(clusterName, "vxlan0", peer).Set(tunnelStatus)
	}

	pm.VXLANOverhead.WithLabelValues(clusterName, "vxlan0").Set(overhead)
}

// UpdateTCMetrics updates Traffic Control metrics
func (pm *PrometheusMetrics) UpdateTCMetrics(clusterName, interfaceName string, status *TCStatus, overhead float64) {
	rulesActive := 0.0
	if status.RulesActive {
		rulesActive = 1.0
	}

	pm.TCRulesActive.WithLabelValues(clusterName, interfaceName).Set(rulesActive)
	pm.TCOverhead.WithLabelValues(clusterName, interfaceName).Set(overhead)
}

// UpdateBandwidthMetrics updates bandwidth utilization metrics
func (pm *PrometheusMetrics) UpdateBandwidthMetrics(clusterName string, interfaceMetrics map[string]*InterfaceMetrics) {
	for interfaceName, metrics := range interfaceMetrics {
		pm.BandwidthUtilization.WithLabelValues(clusterName, interfaceName, "total").Set(metrics.Utilization)

		// Update packet counters
		pm.PacketsTotal.WithLabelValues(clusterName, interfaceName, "rx").Add(float64(metrics.RxPackets))
		pm.PacketsTotal.WithLabelValues(clusterName, interfaceName, "tx").Add(float64(metrics.TxPackets))

		pm.PacketsDropped.WithLabelValues(clusterName, interfaceName, "queue").Add(float64(metrics.RxDropped + metrics.TxDropped))
		pm.PacketsErrors.WithLabelValues(clusterName, interfaceName, "crc").Add(float64(metrics.RxErrors + metrics.TxErrors))
	}
}

// UpdateTestMetrics updates test execution metrics
func (pm *PrometheusMetrics) UpdateTestMetrics(clusterName, testType, sliceType string, duration float64, success bool) {
	pm.TestDuration.WithLabelValues(clusterName, testType, sliceType).Observe(duration)

	if success {
		pm.TestSuccess.WithLabelValues(clusterName, testType, sliceType).Inc()
	} else {
		pm.TestFailure.WithLabelValues(clusterName, testType, sliceType).Inc()
	}
}

// UpdateAgentMetrics updates agent health metrics
func (pm *PrometheusMetrics) UpdateAgentMetrics(clusterName string, healthy bool, connections int) {
	agentStatus := 0.0
	if healthy {
		agentStatus = 1.0
	}

	pm.AgentUp.WithLabelValues(clusterName).Set(agentStatus)
	pm.AgentConnections.WithLabelValues(clusterName).Set(float64(connections))
}

// UpdateThesisCompliance updates thesis validation metrics
func (pm *PrometheusMetrics) UpdateThesisCompliance(clusterName string, validation *ThesisValidation) {
	pm.ThesisCompliance.WithLabelValues(clusterName, "overall").Set(validation.CompliancePercent)

	// Calculate individual metric compliance
	if len(validation.ThroughputResults) > 0 && len(validation.ThroughputTargets) > 0 {
		throughputCompliance := 0.0
		for i, result := range validation.ThroughputResults {
			if i < len(validation.ThroughputTargets) {
				target := validation.ThroughputTargets[i]
				if result >= target*0.9 { // 10% tolerance
					throughputCompliance += 100.0 / float64(len(validation.ThroughputResults))
				}
			}
		}
		pm.ThesisCompliance.WithLabelValues(clusterName, "throughput").Set(throughputCompliance)
	}

	if len(validation.RTTResults) > 0 && len(validation.RTTTargets) > 0 {
		latencyCompliance := 0.0
		for i, result := range validation.RTTResults {
			if i < len(validation.RTTTargets) {
				target := validation.RTTTargets[i]
				if result <= target*1.1 { // 10% tolerance
					latencyCompliance += 100.0 / float64(len(validation.RTTResults))
				}
			}
		}
		pm.ThesisCompliance.WithLabelValues(clusterName, "latency").Set(latencyCompliance)
	}
}

// UpdateSLACompliance updates SLA compliance metrics
func (pm *PrometheusMetrics) UpdateSLACompliance(clusterName, sliceType string, compliant bool) {
	compliance := 0.0
	if compliant {
		compliance = 1.0
	}

	pm.SLACompliance.WithLabelValues(clusterName, sliceType).Set(compliance)
}

// RecordPerformanceTest records a complete performance test
func (pm *PrometheusMetrics) RecordPerformanceTest(clusterName string, metrics *PerformanceMetrics) {
	// Record test execution
	pm.UpdateTestMetrics(clusterName, metrics.TestType, metrics.QoSClass,
		metrics.Duration.Seconds(), len(metrics.ErrorDetails) == 0)

	// Record throughput and latency
	pm.UpdateThroughputMetrics(clusterName, &metrics.Throughput, metrics.QoSClass)
	pm.UpdateLatencyMetrics(clusterName, &metrics.Latency, "target", metrics.QoSClass)

	// Record overhead metrics
	pm.VXLANOverhead.WithLabelValues(clusterName, "vxlan0").Set(metrics.VXLANOverhead)
	pm.TCOverhead.WithLabelValues(clusterName, "primary").Set(metrics.TCOverhead)

	// Record bandwidth utilization
	pm.BandwidthUtilization.WithLabelValues(clusterName, "primary", metrics.QoSClass).Set(metrics.BandwidthUtilization)
}

// RecordAgentStatus records complete agent status
func (pm *PrometheusMetrics) RecordAgentStatus(clusterName string, status *TNStatus) {
	pm.UpdateAgentMetrics(clusterName, status.Healthy, status.ActiveConnections)
	pm.UpdateVXLANMetrics(clusterName, &status.VXLANStatus, 0) // Overhead calculated separately
	pm.UpdateTCMetrics(clusterName, "primary", &status.TCStatus, 0) // Overhead calculated separately
}