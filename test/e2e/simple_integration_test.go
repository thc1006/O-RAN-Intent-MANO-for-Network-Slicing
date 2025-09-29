package e2e_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/argocd"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/claude"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/deployment"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/kpt"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/metrics"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/porch"
)

// TestSimpleIntegration tests basic integration between all components
func TestSimpleIntegration(t *testing.T) {
	t.Run("Natural Language to Deployment Flow", func(t *testing.T) {
		// Arrange
		ctx := context.Background()

		// Test 1: Claude CLI - Natural Language Processing
		claudeConfig := &claude.ClientConfig{
			SessionName: "integration-test",
			Timeout:     5 * time.Second,
			UseFallback: true,
		}
		claudeClient, err := claude.NewClient(ctx, claudeConfig)
		require.NoError(t, err)
		defer claudeClient.Cleanup(ctx)

		intent := &claude.IntentRequest{
			Text: "Deploy an eMBB network slice for video streaming",
		}

		parsedIntent, err := claudeClient.ProcessIntent(ctx, intent)
		require.NoError(t, err)
		assert.Equal(t, "eMBB", parsedIntent.ParsedIntent.SliceType)

		// Test 2: Porch - Package Management
		porchManager := porch.NewPackageLifecycleManager()
		packageSpec := &porch.PackageSpec{
			Name:        "test-slice-package",
			Namespace:   "test-namespace",
			Version:     "v1.0.0",
			Repository:  "test-repo",
			Description: "Test package for eMBB slice",
		}

		pkg, err := porchManager.CreatePackage(ctx, packageSpec)
		require.NoError(t, err)
		assert.NotNil(t, pkg)
		assert.Equal(t, porch.PackageStatusDraft, pkg.Status)

		// Test 3: KPT Functions - Configuration Generation
		nsFunction := kpt.NewSetNamespaceFunction()
		assert.NotNil(t, nsFunction)

		// Test 4: ArgoCD - GitOps Deployment
		argoCDController := argocd.NewApplicationController()
		assert.NotNil(t, argoCDController)

		// Test 5: Real Deployment Management
		deployManager := deployment.NewK8sDeploymentManager()
		ranSpec := &deployment.RANComponentSpec{
			Type:      deployment.ComponentTypeCUCP,
			Name:      "test-cu-cp",
			Namespace: "test-ran",
			SiteID:    "site-001",
			Resources: deployment.ResourceRequirements{
				CPU:    "2",
				Memory: "4Gi",
			},
		}

		ranResult, err := deployManager.DeployRANComponent(ctx, ranSpec)
		require.NoError(t, err)
		assert.Equal(t, deployment.DeploymentStatusRunning, ranResult.Status)

		// Test 6: Metrics Collection
		prometheusClient := metrics.NewPrometheusClient("http://prometheus:9090")
		err = prometheusClient.Connect(ctx)
		require.NoError(t, err)

		query := &metrics.MetricQuery{
			Name:    "test_metric",
			SliceID: "test-slice",
		}

		result, err := prometheusClient.QueryMetrics(ctx, query)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Assert overall integration
		assert.True(t, claudeClient.IsInitialized())
		assert.NotEmpty(t, pkg.UID)
		assert.NotEmpty(t, ranResult.DeploymentName)
		assert.True(t, prometheusClient.IsConnected())
	})
}

// TestComponentCommunication tests inter-component communication
func TestComponentCommunication(t *testing.T) {
	t.Run("Claude to KPT Pipeline", func(t *testing.T) {
		// Arrange
		ctx := context.Background()

		// Process intent with Claude
		claudeClient := createTestClaudeClient(t)
		defer claudeClient.Cleanup(ctx)

		intent := &claude.IntentRequest{
			Text: "Create a URLLC slice with 1ms latency",
		}

		parsedIntent, err := claudeClient.ProcessIntent(ctx, intent)
		require.NoError(t, err)

		// Use parsed intent to configure KPT function
		sliceFunction := kpt.NewGenerateNetworkSliceFunction()
		assert.NotNil(t, sliceFunction)

		// Verify intent was properly parsed
		assert.Equal(t, "URLLC", parsedIntent.ParsedIntent.SliceType)
		assert.Equal(t, 1.0, parsedIntent.ParsedIntent.Requirements.Latency)
	})

	t.Run("Porch to ArgoCD Integration", func(t *testing.T) {
		// Arrange
		ctx := context.Background()

		// Create package with Porch
		porchManager := porch.NewPackageLifecycleManager()
		pkgSpec := &porch.PackageSpec{
			Name:       "argocd-test-pkg",
			Namespace:  "test",
			Version:    "v1.0.0",
			Repository: "test-repo",
		}

		pkg, err := porchManager.CreatePackage(ctx, pkgSpec)
		require.NoError(t, err)

		// Validate package before deployment
		validation, err := porchManager.ValidatePackage(ctx, pkg)
		require.NoError(t, err)
		assert.True(t, validation.IsValid)

		// Create ArgoCD controller for deployment
		argoCDController := argocd.NewApplicationController()
		assert.NotNil(t, argoCDController)
	})
}

// TestEndToEndSliceLifecycle tests complete slice lifecycle
func TestEndToEndSliceLifecycle(t *testing.T) {
	t.Run("Create, Deploy, Monitor, Delete", func(t *testing.T) {
		// Arrange
		ctx := context.Background()

		// CREATE - Parse natural language intent
		claudeClient := createTestClaudeClient(t)
		defer claudeClient.Cleanup(ctx)

		createIntent := &claude.IntentRequest{
			Text: "Create an mIoT slice for smart city sensors",
		}

		parsedCreate, err := claudeClient.ProcessIntent(ctx, createIntent)
		require.NoError(t, err)
		assert.Equal(t, "mIoT", parsedCreate.ParsedIntent.SliceType)

		// DEPLOY - Create deployment specs
		deployManager := deployment.NewK8sDeploymentManager()
		sliceDeployment := &deployment.NetworkSliceDeployment{
			SliceID: "miot-test-001",
			Type:    "mIoT",
			Clusters: []deployment.ClusterTarget{
				{
					Name:       "test-cluster",
					Region:     "test-region",
					Components: []string{"AMF", "SMF"},
				},
			},
		}

		deployResult, err := deployManager.DeployNetworkSlice(ctx, sliceDeployment)
		require.NoError(t, err)
		assert.Equal(t, deployment.SliceStatusActive, deployResult.Status)

		// MONITOR - Setup monitoring
		prometheusClient := metrics.NewPrometheusClient("http://prometheus:9090")
		_ = prometheusClient.Connect(ctx)

		// Create alert rule
		alertRule := &metrics.AlertRule{
			Name:       "miot_device_count",
			Expression: `network_slice_devices{slice_id="miot-test-001"} > 10000`,
			Duration:   5 * time.Minute,
			Severity:   metrics.SeverityWarning,
		}

		err = prometheusClient.CreateAlertRule(ctx, alertRule)
		require.NoError(t, err)

		// Generate dashboard
		dashboardGen := metrics.NewDashboardGenerator()
		dashboard, err := dashboardGen.GenerateSliceDashboard(ctx, "mIoT")
		require.NoError(t, err)
		assert.Contains(t, dashboard.Title, "mIoT")

		// VERIFY - Check all components are working
		assert.True(t, claudeClient.IsInitialized())
		assert.Equal(t, 1, deployResult.DeployedClusters)
		assert.True(t, prometheusClient.IsConnected())
		assert.NotNil(t, dashboard)
	})
}

// TestFailureRecovery tests system resilience
func TestFailureRecovery(t *testing.T) {
	t.Run("Handle deployment failure and rollback", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		deployManager := deployment.NewK8sDeploymentManager()

		// Simulate rollback scenario
		rollbackSpec := &deployment.RollbackSpec{
			DeploymentID:  "failed-deploy-001",
			TargetVersion: "v1.0.0",
			Reason:        "Test rollback",
			Strategy:      "Immediate",
		}

		// Act
		rollbackResult, err := deployManager.RollbackDeployment(ctx, rollbackSpec)

		// Assert
		require.NoError(t, err)
		assert.True(t, rollbackResult.Success)
		assert.Equal(t, "v1.0.0", rollbackResult.CurrentVersion)
	})

	t.Run("Handle invalid natural language intent", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		claudeClient := createTestClaudeClient(t)
		defer claudeClient.Cleanup(ctx)

		emptyIntent := &claude.IntentRequest{
			Text: "",
		}

		// Act
		response, err := claudeClient.ProcessIntent(ctx, emptyIntent)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "empty intent")
	})
}

// TestPerformanceMetrics tests performance and metrics collection
func TestPerformanceMetrics(t *testing.T) {
	t.Run("Collect and export metrics", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		prometheusClient := metrics.NewPrometheusClient("http://prometheus:9090")
		_ = prometheusClient.Connect(ctx)

		// Query historical metrics
		historicalQuery := &metrics.HistoricalQuery{
			Metric: "test_metric",
			Start:  time.Now().Add(-1 * time.Hour),
			End:    time.Now(),
			Step:   5 * time.Minute,
		}

		historicalData, err := prometheusClient.QueryHistorical(ctx, historicalQuery)
		require.NoError(t, err)
		assert.NotEmpty(t, historicalData.DataPoints)

		// Export metrics
		exportConfig := &metrics.ExportConfig{
			Metrics: []string{"test_metric"},
			Format:  metrics.FormatJSON,
			TimeRange: metrics.TimeRange{
				Start: time.Now().Add(-30 * time.Minute),
				End:   time.Now(),
			},
		}

		exportData, err := prometheusClient.ExportMetrics(ctx, exportConfig)
		require.NoError(t, err)
		assert.NotEmpty(t, exportData.Content)
		assert.Contains(t, exportData.Content, "{")
	})

	t.Run("SLA compliance calculation", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		prometheusClient := metrics.NewPrometheusClient("http://prometheus:9090")
		_ = prometheusClient.Connect(ctx)

		slaConfig := &metrics.SLAConfig{
			SliceID:          "test-slice-001",
			LatencyThreshold: 20,
			Availability:     99.9,
			Period:           24 * time.Hour,
		}

		// Act
		compliance, err := prometheusClient.CalculateSLACompliance(ctx, slaConfig)

		// Assert
		require.NoError(t, err)
		assert.GreaterOrEqual(t, compliance.LatencyCompliance, 95.0)
		assert.GreaterOrEqual(t, compliance.AvailabilityMeasured, 99.5)
	})
}

// Helper function
func createTestClaudeClient(t *testing.T) *claude.Client {
	ctx := context.Background()
	config := &claude.ClientConfig{
		SessionName: "test-session",
		Timeout:     5 * time.Second,
		UseFallback: true,
	}

	client, err := claude.NewClient(ctx, config)
	require.NoError(t, err)
	return client
}