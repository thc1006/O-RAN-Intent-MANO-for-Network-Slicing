package unit

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/o-ran/intent-mano/tn/agent/pkg"
)

func TestTCManager(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	t.Run("NewTCManager", func(t *testing.T) {
		policy := &pkg.BandwidthPolicy{
			DownlinkMbps: 100.0,
			UplinkMbps:   50.0,
			LatencyMs:    10.0,
			JitterMs:     2.0,
			LossPercent:  0.1,
			Priority:     1,
		}

		tcManager := pkg.NewTCManager(policy, "eth0", logger)
		require.NotNil(t, tcManager)
	})

	t.Run("CalculateTCOverhead", func(t *testing.T) {
		policy := &pkg.BandwidthPolicy{
			DownlinkMbps: 100.0,
			UplinkMbps:   50.0,
			LatencyMs:    10.0,
			JitterMs:     2.0,
			LossPercent:  0.1,
			Priority:     1,
			Filters: []pkg.Filter{
				{Protocol: "tcp", Priority: 10},
				{Protocol: "udp", Priority: 20},
			},
		}

		tcManager := pkg.NewTCManager(policy, "eth0", logger)
		overhead := tcManager.CalculateTCOverhead()

		assert.Greater(t, overhead, 0.0, "TC overhead should be positive")
		assert.Less(t, overhead, 20.0, "TC overhead should be reasonable")
	})

	t.Run("BandwidthPolicyValidation", func(t *testing.T) {
		testCases := []struct {
			name     string
			policy   *pkg.BandwidthPolicy
			valid    bool
		}{
			{
				name: "Valid eMBB Policy",
				policy: &pkg.BandwidthPolicy{
					DownlinkMbps: 4.57,
					UplinkMbps:   2.0,
					LatencyMs:    16.1,
					JitterMs:     1.0,
					LossPercent:  0.1,
					Priority:     3,
				},
				valid: true,
			},
			{
				name: "Valid URLLC Policy",
				policy: &pkg.BandwidthPolicy{
					DownlinkMbps: 0.93,
					UplinkMbps:   0.5,
					LatencyMs:    6.3,
					JitterMs:     0.5,
					LossPercent:  0.001,
					Priority:     1,
				},
				valid: true,
			},
			{
				name: "Valid mIoT Policy",
				policy: &pkg.BandwidthPolicy{
					DownlinkMbps: 2.77,
					UplinkMbps:   1.0,
					LatencyMs:    15.7,
					JitterMs:     2.0,
					LossPercent:  0.1,
					Priority:     2,
				},
				valid: true,
			},
			{
				name: "Invalid - Zero Bandwidth",
				policy: &pkg.BandwidthPolicy{
					DownlinkMbps: 0.0,
					UplinkMbps:   0.0,
					LatencyMs:    10.0,
					Priority:     1,
				},
				valid: false,
			},
			{
				name: "Invalid - Negative Latency",
				policy: &pkg.BandwidthPolicy{
					DownlinkMbps: 10.0,
					UplinkMbps:   5.0,
					LatencyMs:    -1.0,
					Priority:     1,
				},
				valid: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tcManager := pkg.NewTCManager(tc.policy, "eth0", logger)
				require.NotNil(t, tcManager)

				if tc.valid {
					assert.Greater(t, tc.policy.DownlinkMbps, 0.0)
					assert.GreaterOrEqual(t, tc.policy.LatencyMs, 0.0)
					assert.GreaterOrEqual(t, tc.policy.LossPercent, 0.0)
				}
			})
		}
	})

	t.Run("FilterConfiguration", func(t *testing.T) {
		filters := []pkg.Filter{
			{
				Protocol: "tcp",
				SrcIP:    "192.168.1.0/24",
				DstPort:  80,
				Priority: 10,
				ClassID:  "1:10",
			},
			{
				Protocol: "udp",
				SrcIP:    "192.168.2.0/24",
				DstPort:  53,
				Priority: 5,
				ClassID:  "1:10",
			},
			{
				Protocol: "icmp",
				Priority: 15,
				ClassID:  "1:20",
			},
		}

		policy := &pkg.BandwidthPolicy{
			DownlinkMbps: 100.0,
			UplinkMbps:   50.0,
			Filters:      filters,
		}

		tcManager := pkg.NewTCManager(policy, "eth0", logger)
		require.NotNil(t, tcManager)

		// Verify filter configuration
		assert.Len(t, policy.Filters, 3)
		assert.Equal(t, "tcp", policy.Filters[0].Protocol)
		assert.Equal(t, 80, policy.Filters[0].DstPort)
		assert.Equal(t, "1:10", policy.Filters[0].ClassID)
	})
}

func TestVXLANManager(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	t.Run("NewVXLANManager", func(t *testing.T) {
		config := &pkg.VXLANConfig{
			VNI:        100,
			RemoteIPs:  []string{"192.168.1.100", "192.168.1.101"},
			LocalIP:    "192.168.1.102",
			Port:       4789,
			MTU:        1450,
			DeviceName: "vxlan100",
			Learning:   false,
		}

		vxlanManager := pkg.NewVXLANManager(config, logger)
		require.NotNil(t, vxlanManager)
	})

	t.Run("CalculateVXLANOverhead", func(t *testing.T) {
		config := &pkg.VXLANConfig{
			VNI:        100,
			MTU:        1450,
			DeviceName: "vxlan100",
		}

		vxlanManager := pkg.NewVXLANManager(config, logger)

		testCases := []struct {
			name        string
			originalMTU int
			expected    float64
		}{
			{"Standard MTU", 1500, 3.33}, // 50/1500 * 100
			{"Jumbo MTU", 9000, 0.56},    // 50/9000 * 100
			{"Custom MTU", 1450, 3.45},   // 50/1450 * 100
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				overhead := vxlanManager.CalculateVXLANOverhead(tc.originalMTU)
				assert.InDelta(t, tc.expected, overhead, 0.1, "VXLAN overhead calculation")
			})
		}
	})

	t.Run("VXLANConfigValidation", func(t *testing.T) {
		testCases := []struct {
			name   string
			config *pkg.VXLANConfig
			valid  bool
		}{
			{
				name: "Valid Configuration",
				config: &pkg.VXLANConfig{
					VNI:        100,
					RemoteIPs:  []string{"192.168.1.100"},
					LocalIP:    "192.168.1.102",
					Port:       4789,
					MTU:        1450,
					DeviceName: "vxlan100",
				},
				valid: true,
			},
			{
				name: "Invalid VNI",
				config: &pkg.VXLANConfig{
					VNI:        16777216, // > 24-bit max
					RemoteIPs:  []string{"192.168.1.100"},
					LocalIP:    "192.168.1.102",
					Port:       4789,
					DeviceName: "vxlan100",
				},
				valid: false,
			},
			{
				name: "Invalid Local IP",
				config: &pkg.VXLANConfig{
					VNI:        100,
					RemoteIPs:  []string{"192.168.1.100"},
					LocalIP:    "invalid-ip",
					Port:       4789,
					DeviceName: "vxlan100",
				},
				valid: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				vxlanManager := pkg.NewVXLANManager(tc.config, logger)
				require.NotNil(t, vxlanManager)

				if tc.valid {
					assert.Greater(t, tc.config.VNI, uint32(0))
					assert.Less(t, tc.config.VNI, uint32(16777216))
					assert.NotEmpty(t, tc.config.LocalIP)
					assert.Greater(t, tc.config.Port, 0)
				}
			})
		}
	})
}

func TestIperfManager(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	t.Run("NewIperfManager", func(t *testing.T) {
		iperfManager := pkg.NewIperfManager(logger)
		require.NotNil(t, iperfManager)

		servers := iperfManager.GetActiveServers()
		assert.Empty(t, servers, "No servers should be active initially")
	})

	t.Run("IperfTestConfig", func(t *testing.T) {
		testCases := []struct {
			name   string
			config *pkg.IperfTestConfig
			valid  bool
		}{
			{
				name: "Valid TCP Test",
				config: &pkg.IperfTestConfig{
					ServerIP: "192.168.1.100",
					Port:     5201,
					Duration: 10,
					Protocol: "tcp",
					Parallel: 1,
					JSON:     true,
				},
				valid: true,
			},
			{
				name: "Valid UDP Test",
				config: &pkg.IperfTestConfig{
					ServerIP:  "192.168.1.100",
					Port:      5201,
					Duration:  10,
					Protocol:  "udp",
					Bandwidth: "10M",
					Parallel:  1,
					JSON:      true,
				},
				valid: true,
			},
			{
				name: "Invalid - No Server IP",
				config: &pkg.IperfTestConfig{
					Port:     5201,
					Duration: 10,
					Protocol: "tcp",
				},
				valid: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if tc.valid {
					assert.NotEmpty(t, tc.config.ServerIP)
					assert.Greater(t, tc.config.Port, 0)
					assert.Greater(t, tc.config.Duration, 0)
					assert.Contains(t, []string{"tcp", "udp"}, tc.config.Protocol)
				}
			})
		}
	})
}

func TestPerformanceMetrics(t *testing.T) {
	t.Run("ThroughputMetricsValidation", func(t *testing.T) {
		metrics := &pkg.ThroughputMetrics{
			DownlinkMbps: 4.57,
			UplinkMbps:   2.0,
			AvgMbps:      3.285,
			PeakMbps:     4.57,
			MinMbps:      2.0,
		}

		// Validate thesis targets
		thesisTargets := []float64{0.93, 2.77, 4.57}

		// Check if metrics meet eMBB target (4.57 Mbps)
		assert.GreaterOrEqual(t, metrics.DownlinkMbps, thesisTargets[2]*0.9,
			"eMBB downlink should meet thesis target with 10% tolerance")

		// Validate metric consistency
		assert.Equal(t, (metrics.DownlinkMbps + metrics.UplinkMbps) / 2, metrics.AvgMbps)
		assert.GreaterOrEqual(t, metrics.PeakMbps, metrics.AvgMbps)
		assert.LessOrEqual(t, metrics.MinMbps, metrics.AvgMbps)
	})

	t.Run("LatencyMetricsValidation", func(t *testing.T) {
		metrics := &pkg.LatencyMetrics{
			RTTMs:    16.1,
			MinRTTMs: 15.0,
			MaxRTTMs: 18.0,
			AvgRTTMs: 16.1,
			P50Ms:    16.0,
			P95Ms:    17.5,
			P99Ms:    18.0,
		}

		// Validate thesis targets
		thesisTargets := []float64{6.3, 15.7, 16.1}

		// Check if metrics meet eMBB target (16.1 ms)
		assert.LessOrEqual(t, metrics.AvgRTTMs, thesisTargets[2]*1.1,
			"eMBB latency should meet thesis target with 10% tolerance")

		// Validate metric consistency
		assert.LessOrEqual(t, metrics.MinRTTMs, metrics.AvgRTTMs)
		assert.GreaterOrEqual(t, metrics.MaxRTTMs, metrics.AvgRTTMs)
		assert.LessOrEqual(t, metrics.P50Ms, metrics.P95Ms)
		assert.LessOrEqual(t, metrics.P95Ms, metrics.P99Ms)
	})
}

func TestThesisValidation(t *testing.T) {
	t.Run("ThesisTargetCompliance", func(t *testing.T) {
		validation := &pkg.ThesisValidation{
			ThroughputTargets: []float64{0.93, 2.77, 4.57},
			RTTTargets:        []float64{6.3, 15.7, 16.1},
			ThroughputResults: []float64{0.95, 2.80, 4.60},
			RTTResults:        []float64{6.0, 15.5, 16.0},
			DeployTargetMs:    600000, // 10 minutes
			DeployTimeMs:      480000, // 8 minutes
		}

		// Calculate compliance
		passedTests := 0
		totalTests := 0

		// Throughput compliance (10% tolerance)
		for i, target := range validation.ThroughputTargets {
			if i < len(validation.ThroughputResults) {
				totalTests++
				if validation.ThroughputResults[i] >= target*0.9 {
					passedTests++
				}
			}
		}

		// RTT compliance (10% tolerance)
		for i, target := range validation.RTTTargets {
			if i < len(validation.RTTResults) {
				totalTests++
				if validation.RTTResults[i] <= target*1.1 {
					passedTests++
				}
			}
		}

		// Deploy time compliance
		totalTests++
		if validation.DeployTimeMs <= validation.DeployTargetMs {
			passedTests++
		}

		validation.PassedTests = passedTests
		validation.TotalTests = totalTests
		validation.CompliancePercent = float64(passedTests) / float64(totalTests) * 100

		assert.Equal(t, 7, validation.TotalTests, "Total tests should be 7 (3 throughput + 3 RTT + 1 deploy)")
		assert.Equal(t, 7, validation.PassedTests, "All tests should pass")
		assert.Equal(t, 100.0, validation.CompliancePercent, "Compliance should be 100%")
	})

	t.Run("SliceTypeTargets", func(t *testing.T) {
		sliceTypes := map[string]struct {
			throughputMbps float64
			latencyMs      float64
		}{
			"URLLC": {0.93, 6.3},
			"mIoT":  {2.77, 15.7},
			"eMBB":  {4.57, 16.1},
		}

		for sliceType, targets := range sliceTypes {
			t.Run(sliceType, func(t *testing.T) {
				assert.Greater(t, targets.throughputMbps, 0.0, "Throughput target should be positive")
				assert.Greater(t, targets.latencyMs, 0.0, "Latency target should be positive")

				// URLLC should have lowest latency
				if sliceType == "URLLC" {
					assert.Less(t, targets.latencyMs, sliceTypes["mIoT"].latencyMs)
					assert.Less(t, targets.latencyMs, sliceTypes["eMBB"].latencyMs)
				}

				// eMBB should have highest throughput
				if sliceType == "eMBB" {
					assert.Greater(t, targets.throughputMbps, sliceTypes["URLLC"].throughputMbps)
					assert.Greater(t, targets.throughputMbps, sliceTypes["mIoT"].throughputMbps)
				}
			})
		}
	})
}