package metrics_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/metrics"
)

// TestPrometheusConnection verifies Prometheus connection
func TestPrometheusConnection(t *testing.T) {
	t.Run("Connect to Prometheus server", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := metrics.NewPrometheusClient("http://prometheus:9090")

		// Act
		err := client.Connect(ctx)

		// Assert
		require.NoError(t, err)
		assert.True(t, client.IsConnected())
	})

	t.Run("Handle connection failure gracefully", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := metrics.NewPrometheusClient("http://invalid-prometheus:9090")

		// Act
		err := client.Connect(ctx)

		// Assert
		// Should not error but handle gracefully
		require.NoError(t, err)
		assert.False(t, client.IsConnected())
	})
}

// TestMetricsCollection verifies metrics collection
func TestMetricsCollection(t *testing.T) {
	t.Run("Collect network slice metrics", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := metrics.NewPrometheusClient("http://prometheus:9090")
		_ = client.Connect(ctx)

		query := &metrics.MetricQuery{
			Name:      "network_slice_throughput",
			SliceID:   "embb-slice-001",
			TimeRange: metrics.TimeRange{
				Start: time.Now().Add(-1 * time.Hour),
				End:   time.Now(),
				Step:  5 * time.Minute,
			},
		}

		// Act
		result, err := client.QueryMetrics(ctx, query)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Values)
		for _, value := range result.Values {
			assert.GreaterOrEqual(t, value.Value, 0.0)
			assert.NotZero(t, value.Timestamp)
		}
	})

	t.Run("Collect RAN component metrics", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := metrics.NewPrometheusClient("http://prometheus:9090")
		_ = client.Connect(ctx)

		ranMetrics := []string{
			"ran_cu_cp_sessions",
			"ran_du_prb_utilization",
			"ran_ru_power_consumption",
			"ran_handover_success_rate",
		}

		// Act & Assert
		for _, metricName := range ranMetrics {
			query := &metrics.MetricQuery{
				Name:      metricName,
				Component: "cu-cp-001",
			}

			result, err := client.QueryMetrics(ctx, query)
			require.NoError(t, err)
			assert.NotNil(t, result)
		}
	})

	t.Run("Collect 5G Core metrics", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := metrics.NewPrometheusClient("http://prometheus:9090")
		_ = client.Connect(ctx)

		// Act
		coreMetrics, err := client.Get5GCoreMetrics(ctx)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, coreMetrics)
		assert.GreaterOrEqual(t, coreMetrics.RegisteredUEs, 0)
		assert.GreaterOrEqual(t, coreMetrics.ActiveSessions, 0)
		assert.GreaterOrEqual(t, coreMetrics.AMFLoad, 0.0)
		assert.GreaterOrEqual(t, coreMetrics.SMFLoad, 0.0)
		assert.GreaterOrEqual(t, coreMetrics.UPFThroughput, 0.0)
	})

	t.Run("Aggregate metrics across sites", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := metrics.NewPrometheusClient("http://prometheus:9090")
		_ = client.Connect(ctx)

		sites := []string{"site-001", "site-002", "site-003"}

		// Act
		aggregated, err := client.AggregateMetrics(ctx, &metrics.AggregationQuery{
			Metric:    "ran_throughput",
			GroupBy:   "site",
			Operation: metrics.AggregationSum,
			Sites:     sites,
		})

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, aggregated)
		assert.GreaterOrEqual(t, aggregated.TotalValue, 0.0)
		assert.Len(t, aggregated.GroupValues, len(sites))
	})
}

// TestAlertingRules verifies alerting rules
func TestAlertingRules(t *testing.T) {
	t.Run("Create alerting rule for high latency", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := metrics.NewPrometheusClient("http://prometheus:9090")
		_ = client.Connect(ctx)

		rule := &metrics.AlertRule{
			Name:        "high_latency_embb",
			Expression:  `network_slice_latency{slice_type="eMBB"} > 20`,
			Duration:    5 * time.Minute,
			Severity:    metrics.SeverityWarning,
			Annotations: map[string]string{
				"summary":     "High latency detected for eMBB slice",
				"description": "Latency exceeds 20ms threshold for {{ $labels.slice_id }}",
			},
		}

		// Act
		err := client.CreateAlertRule(ctx, rule)

		// Assert
		require.NoError(t, err)
	})

	t.Run("Create alerting rule for resource exhaustion", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := metrics.NewPrometheusClient("http://prometheus:9090")
		_ = client.Connect(ctx)

		rule := &metrics.AlertRule{
			Name:       "resource_exhaustion",
			Expression: `(node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) < 0.1`,
			Duration:   10 * time.Minute,
			Severity:   metrics.SeverityCritical,
			Annotations: map[string]string{
				"summary": "Node memory exhaustion",
			},
		}

		// Act
		err := client.CreateAlertRule(ctx, rule)

		// Assert
		require.NoError(t, err)
	})

	t.Run("Query active alerts", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := metrics.NewPrometheusClient("http://prometheus:9090")
		_ = client.Connect(ctx)

		// Act
		alerts, err := client.GetActiveAlerts(ctx)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, alerts)
		for _, alert := range alerts {
			assert.NotEmpty(t, alert.Name)
			assert.NotEmpty(t, alert.State)
		}
	})
}

// TestGrafanaDashboards verifies Grafana dashboard generation
func TestGrafanaDashboards(t *testing.T) {
	t.Run("Generate network slice dashboard", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		generator := metrics.NewDashboardGenerator()

		config := &metrics.DashboardConfig{
			Title:       "Network Slice Performance",
			Description: "Real-time network slice metrics",
			Panels: []metrics.PanelConfig{
				{
					Title:  "Throughput",
					Type:   metrics.PanelTypeGraph,
					Query:  `rate(network_slice_throughput[5m])`,
					Unit:   "Mbps",
				},
				{
					Title:  "Latency",
					Type:   metrics.PanelTypeGraph,
					Query:  `network_slice_latency`,
					Unit:   "ms",
				},
				{
					Title:  "Active UEs",
					Type:   metrics.PanelTypeStat,
					Query:  `sum(network_slice_active_ues)`,
				},
			},
		}

		// Act
		dashboard, err := generator.GenerateDashboard(ctx, config)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, dashboard)
		assert.Equal(t, "Network Slice Performance", dashboard.Title)
		assert.Len(t, dashboard.Panels, 3)
		assert.NotEmpty(t, dashboard.UID)
	})

	t.Run("Generate RAN component dashboard", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		generator := metrics.NewDashboardGenerator()

		// Act
		dashboard, err := generator.GenerateRANDashboard(ctx, "site-001")

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, dashboard)
		assert.Contains(t, dashboard.Title, "RAN")
		assert.NotEmpty(t, dashboard.Panels)
	})

	t.Run("Export dashboard to Grafana", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		grafanaClient := metrics.NewGrafanaClient("http://grafana:3000", "admin", "admin")
		generator := metrics.NewDashboardGenerator()

		config := &metrics.DashboardConfig{
			Title: "Test Dashboard",
			Panels: []metrics.PanelConfig{
				{Title: "Test Panel", Query: "up"},
			},
		}

		dashboard, _ := generator.GenerateDashboard(ctx, config)

		// Act
		err := grafanaClient.ImportDashboard(ctx, dashboard)

		// Assert
		require.NoError(t, err)
	})
}

// TestHistoricalData verifies historical data queries
func TestHistoricalData(t *testing.T) {
	t.Run("Query historical metrics", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := metrics.NewPrometheusClient("http://prometheus:9090")
		_ = client.Connect(ctx)

		// Act
		historical, err := client.QueryHistorical(ctx, &metrics.HistoricalQuery{
			Metric: "network_slice_throughput",
			Start:  time.Now().Add(-24 * time.Hour),
			End:    time.Now(),
			Step:   1 * time.Hour,
		})

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, historical)
		assert.NotEmpty(t, historical.DataPoints)
	})

	t.Run("Calculate SLA compliance", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := metrics.NewPrometheusClient("http://prometheus:9090")
		_ = client.Connect(ctx)

		sla := &metrics.SLAConfig{
			SliceID:          "embb-slice-001",
			LatencyThreshold: 20,  // ms
			Availability:     99.9, // percentage
			Period:           24 * time.Hour,
		}

		// Act
		compliance, err := client.CalculateSLACompliance(ctx, sla)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, compliance)
		assert.GreaterOrEqual(t, compliance.LatencyCompliance, 0.0)
		assert.LessOrEqual(t, compliance.LatencyCompliance, 100.0)
		assert.GreaterOrEqual(t, compliance.AvailabilityMeasured, 0.0)
	})
}

// TestMetricsExport verifies metrics export functionality
func TestMetricsExport(t *testing.T) {
	t.Run("Export metrics in CSV format", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := metrics.NewPrometheusClient("http://prometheus:9090")
		_ = client.Connect(ctx)

		exportConfig := &metrics.ExportConfig{
			Metrics: []string{
				"network_slice_throughput",
				"network_slice_latency",
			},
			Format:    metrics.FormatCSV,
			TimeRange: metrics.TimeRange{
				Start: time.Now().Add(-1 * time.Hour),
				End:   time.Now(),
			},
		}

		// Act
		exportData, err := client.ExportMetrics(ctx, exportConfig)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, exportData)
		assert.NotEmpty(t, exportData.Content)
		assert.Contains(t, exportData.Content, "timestamp")
		assert.Contains(t, exportData.Content, "value")
	})

	t.Run("Export metrics in JSON format", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		client := metrics.NewPrometheusClient("http://prometheus:9090")
		_ = client.Connect(ctx)

		exportConfig := &metrics.ExportConfig{
			Metrics:   []string{"network_slice_throughput"},
			Format:    metrics.FormatJSON,
			TimeRange: metrics.TimeRange{
				Start: time.Now().Add(-30 * time.Minute),
				End:   time.Now(),
			},
		}

		// Act
		exportData, err := client.ExportMetrics(ctx, exportConfig)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, exportData)
		assert.NotEmpty(t, exportData.Content)
		assert.Contains(t, exportData.Content, "{")
	})
}

// TestRealTimeStreaming verifies real-time metrics streaming
func TestRealTimeStreaming(t *testing.T) {
	t.Run("Stream real-time metrics", func(t *testing.T) {
		// Arrange
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		client := metrics.NewPrometheusClient("http://prometheus:9090")
		_ = client.Connect(ctx)

		streamConfig := &metrics.StreamConfig{
			Metrics:  []string{"network_slice_throughput"},
			Interval: 1 * time.Second,
		}

		// Act
		stream, err := client.StreamMetrics(ctx, streamConfig)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, stream)

		// Read at least one metric
		select {
		case metric := <-stream:
			assert.NotNil(t, metric)
			assert.NotEmpty(t, metric.Name)
			assert.GreaterOrEqual(t, metric.Value, 0.0)
		case <-ctx.Done():
			// Timeout is okay for test
		}
	})
}
