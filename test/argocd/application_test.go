package argocd_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/argocd"
)

// TestApplicationCreation verifies ArgoCD application creation
func TestApplicationCreation(t *testing.T) {
	t.Run("Create application with valid spec", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		controller := argocd.NewApplicationController()

		appSpec := &argocd.ApplicationSpec{
			Name:      "embb-slice-app",
			Namespace: "argocd",
			Project:   "default",
			Source: argocd.ApplicationSource{
				RepoURL:        "https://github.com/o-ran/network-slices",
				TargetRevision: "main",
				Path:           "slices/embb",
			},
			Destination: argocd.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: "network-slices",
			},
			SyncPolicy: &argocd.SyncPolicy{
				Automated: &argocd.AutomatedSyncPolicy{
					Prune:    true,
					SelfHeal: true,
					AllowEmpty: false,
				},
				SyncOptions: []string{"CreateNamespace=true"},
				Retry: &argocd.RetryStrategy{
					Limit: 5,
					Backoff: &argocd.Backoff{
						Duration:    "5s",
						Factor:      2,
						MaxDuration: "3m",
					},
				},
			},
		}

		// Act
		app, err := controller.CreateApplication(ctx, appSpec)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, app)
		assert.Equal(t, "embb-slice-app", app.Name)
		assert.Equal(t, "argocd", app.Namespace)
		assert.NotEmpty(t, app.UID)
		assert.Equal(t, argocd.HealthStatusUnknown, app.Health.Status)
		assert.Equal(t, argocd.SyncStatusCodeOutOfSync, app.Sync.Status)
	})

	t.Run("Fail to create application with invalid spec", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		controller := argocd.NewApplicationController()

		invalidSpec := &argocd.ApplicationSpec{
			Name: "", // Invalid: empty name
		}

		// Act
		app, err := controller.CreateApplication(ctx, invalidSpec)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, app)
		assert.Contains(t, err.Error(), "application name is required")
	})

	t.Run("Create application with Helm chart source", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		controller := argocd.NewApplicationController()

		appSpec := &argocd.ApplicationSpec{
			Name:      "helm-app",
			Namespace: "argocd",
			Project:   "default",
			Source: argocd.ApplicationSource{
				RepoURL:        "https://charts.bitnami.com/bitnami",
				Chart:          "nginx",
				TargetRevision: "15.0.0",
				Helm: &argocd.HelmParameters{
					ValueFiles: []string{"values.yaml"},
					Parameters: []argocd.HelmParameter{
						{Name: "service.type", Value: "LoadBalancer"},
						{Name: "replicaCount", Value: "3"},
					},
				},
			},
			Destination: argocd.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: "default",
			},
		}

		// Act
		app, err := controller.CreateApplication(ctx, appSpec)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, app)
		assert.Equal(t, "helm-app", app.Name)
		assert.NotNil(t, app.Spec.Source.Helm)
		assert.Len(t, app.Spec.Source.Helm.Parameters, 2)
	})
}

// TestApplicationSync verifies application synchronization
func TestApplicationSync(t *testing.T) {
	t.Run("Sync application successfully", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		controller := argocd.NewApplicationController()

		// First create an application
		appSpec := &argocd.ApplicationSpec{
			Name:      "test-sync-app",
			Namespace: "argocd",
			Project:   "default",
			Source: argocd.ApplicationSource{
				RepoURL:        "https://github.com/test/repo",
				TargetRevision: "main",
				Path:           "manifests",
			},
			Destination: argocd.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: "default",
			},
		}

		app, err := controller.CreateApplication(ctx, appSpec)
		require.NoError(t, err)

		// Act
		syncResult, err := controller.SyncApplication(ctx, app.Name, &argocd.SyncOptions{
			Prune:    true,
			DryRun:   false,
			Strategy: argocd.SyncStrategyApply,
			Resources: []argocd.SyncResource{},
		})

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, syncResult)
		assert.Equal(t, argocd.OperationPhaseSucceeded, syncResult.Phase)
		assert.NotEmpty(t, syncResult.Message)
		assert.NotZero(t, syncResult.StartedAt)
		assert.NotZero(t, syncResult.FinishedAt)
	})

	t.Run("Sync with selective resources", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		controller := argocd.NewApplicationController()

		// Act
		syncResult, err := controller.SyncApplication(ctx, "existing-app", &argocd.SyncOptions{
			Prune:    false,
			DryRun:   false,
			Strategy: argocd.SyncStrategyApply,
			Resources: []argocd.SyncResource{
				{Group: "apps", Kind: "Deployment", Name: "my-app"},
				{Group: "", Kind: "Service", Name: "my-app-svc"},
			},
		})

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, syncResult)
		assert.Len(t, syncResult.Resources, 2)
	})

	t.Run("Dry run sync", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		controller := argocd.NewApplicationController()

		// Act
		syncResult, err := controller.SyncApplication(ctx, "test-app", &argocd.SyncOptions{
			DryRun: true,
		})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, argocd.OperationPhaseDryRun, syncResult.Phase)
	})
}

// TestApplicationRollback verifies application rollback functionality
func TestApplicationRollback(t *testing.T) {
	t.Run("Rollback to previous revision", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		controller := argocd.NewApplicationController()

		// Act
		rollbackResult, err := controller.RollbackApplication(ctx, "test-app", &argocd.RollbackOptions{
			Revision: "abc123def",
			DryRun:   false,
			Prune:    true,
		})

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, rollbackResult)
		assert.Equal(t, argocd.OperationPhaseSucceeded, rollbackResult.Phase)
		assert.Equal(t, "abc123def", rollbackResult.Revision)
	})

	t.Run("Rollback with history ID", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		controller := argocd.NewApplicationController()

		// Act
		rollbackResult, err := controller.RollbackApplication(ctx, "test-app", &argocd.RollbackOptions{
			HistoryID: 5,
			DryRun:    false,
		})

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, rollbackResult)
		assert.Equal(t, int64(5), rollbackResult.HistoryID)
	})
}

// TestMultiClusterDeployment verifies deployment across multiple clusters
func TestMultiClusterDeployment(t *testing.T) {
	t.Run("Deploy to multiple clusters", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		controller := argocd.NewApplicationController()

		clusters := []argocd.ClusterConfig{
			{
				Name:   "production-east",
				Server: "https://k8s-prod-east.example.com",
				Region: "us-east-1",
			},
			{
				Name:   "production-west",
				Server: "https://k8s-prod-west.example.com",
				Region: "us-west-2",
			},
			{
				Name:   "production-eu",
				Server: "https://k8s-prod-eu.example.com",
				Region: "eu-central-1",
			},
		}

		appTemplate := &argocd.ApplicationSpec{
			Project: "network-slices",
			Source: argocd.ApplicationSource{
				RepoURL:        "https://github.com/o-ran/slices",
				TargetRevision: "v1.0.0",
				Path:           "manifests",
			},
		}

		// Act
		deploymentResult, err := controller.DeployToMultipleClusters(ctx, "embb-slice", appTemplate, clusters)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, deploymentResult)
		assert.Len(t, deploymentResult.Applications, 3)
		assert.Equal(t, 3, deploymentResult.SuccessCount)
		assert.Equal(t, 0, deploymentResult.FailureCount)

		// Verify each application
		for _, app := range deploymentResult.Applications {
			assert.Contains(t, app.Name, "embb-slice")
			assert.Equal(t, "network-slices", app.Spec.Project)
			assert.NotEmpty(t, app.Spec.Destination.Server)
		}
	})

	t.Run("Handle partial deployment failures", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		controller := argocd.NewApplicationController()

		clusters := []argocd.ClusterConfig{
			{
				Name:   "working-cluster",
				Server: "https://k8s-working.example.com",
			},
			{
				Name:   "failing-cluster",
				Server: "https://invalid-cluster", // This will fail
			},
		}

		appTemplate := &argocd.ApplicationSpec{
			Project: "default",
			Source: argocd.ApplicationSource{
				RepoURL: "https://github.com/test/repo",
				Path:    ".",
			},
		}

		// Act
		deploymentResult, err := controller.DeployToMultipleClusters(ctx, "test-app", appTemplate, clusters)

		// Assert
		require.NoError(t, err) // Should not error even with partial failures
		assert.Equal(t, 1, deploymentResult.SuccessCount)
		assert.Equal(t, 1, deploymentResult.FailureCount)
		assert.Len(t, deploymentResult.Errors, 1)
	})
}

// TestProgressiveDelivery verifies progressive rollout capabilities
func TestProgressiveDelivery(t *testing.T) {
	t.Run("Canary deployment with traffic splitting", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		controller := argocd.NewApplicationController()

		strategy := &argocd.ProgressiveDeliveryStrategy{
			Type: argocd.CanaryDeployment,
			Steps: []argocd.CanaryStep{
				{Weight: 10, Pause: &argocd.PauseDuration{Duration: 5 * time.Minute}},
				{Weight: 30, Pause: &argocd.PauseDuration{Duration: 10 * time.Minute}},
				{Weight: 50, Analysis: &argocd.AnalysisRun{Template: "success-rate"}},
				{Weight: 100},
			},
			Analysis: &argocd.AnalysisTemplate{
				Metrics: []argocd.Metric{
					{
						Name:         "success-rate",
						Interval:     "30s",
						SuccessCondition: "result >= 0.99",
						FailureCondition: "result < 0.95",
						Provider: argocd.MetricProvider{
							Prometheus: &argocd.PrometheusMetric{
								Address: "http://prometheus:9090",
								Query:   `sum(rate(http_requests_total{status=~"2.."}[5m])) / sum(rate(http_requests_total[5m]))`,
							},
						},
					},
				},
			},
		}

		// Act
		progressiveResult, err := controller.StartProgressiveDelivery(ctx, "test-app", "v2.0.0", strategy)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, progressiveResult)
		assert.Equal(t, argocd.ProgressiveDeliveryStatusInProgress, progressiveResult.Status)
		assert.Equal(t, 0, progressiveResult.CurrentStep)
		assert.Equal(t, 10, progressiveResult.CurrentWeight)
		assert.NotEmpty(t, progressiveResult.RolloutID)
	})

	t.Run("Blue-green deployment", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		controller := argocd.NewApplicationController()

		strategy := &argocd.ProgressiveDeliveryStrategy{
			Type: argocd.BlueGreenDeployment,
			BlueGreen: &argocd.BlueGreenStrategy{
				ActiveService:       "app-active",
				PreviewService:      "app-preview",
				AutoPromotionEnabled: false,
				ScaleDownDelaySeconds: 30,
				PrePromotionAnalysis: &argocd.AnalysisRun{
					Template: "smoke-tests",
				},
			},
		}

		// Act
		progressiveResult, err := controller.StartProgressiveDelivery(ctx, "test-app", "v2.0.0", strategy)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, argocd.ProgressiveDeliveryStatusInProgress, progressiveResult.Status)
		assert.True(t, progressiveResult.BlueGreenActive)
	})

	t.Run("Abort progressive delivery on failure", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		controller := argocd.NewApplicationController()

		// Act
		abortResult, err := controller.AbortProgressiveDelivery(ctx, "rollout-123")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, argocd.ProgressiveDeliveryStatusAborted, abortResult.Status)
		assert.NotEmpty(t, abortResult.Message)
	})

	t.Run("Promote progressive delivery manually", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		controller := argocd.NewApplicationController()

		// Act
		promoteResult, err := controller.PromoteProgressiveDelivery(ctx, "rollout-123", &argocd.PromoteOptions{
			Full: false,
			SkipAnalysis: false,
		})

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, promoteResult)
		assert.Greater(t, promoteResult.CurrentStep, 0)
	})
}

// TestApplicationStatus verifies status monitoring
func TestApplicationStatus(t *testing.T) {
	t.Run("Get application status", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		controller := argocd.NewApplicationController()

		// Act
		status, err := controller.GetApplicationStatus(ctx, "test-app")

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, status)
		assert.NotEmpty(t, status.Health.Status)
		assert.NotEmpty(t, status.Sync.Status)
		assert.NotNil(t, status.Resources)
	})

	t.Run("Watch application events", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		controller := argocd.NewApplicationController()

		// Act
		eventChan, err := controller.WatchApplicationEvents(ctx, "test-app")

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, eventChan)

		// Simulate receiving an event
		select {
		case event := <-eventChan:
			assert.NotNil(t, event)
			assert.NotEmpty(t, event.Type)
		case <-time.After(100 * time.Millisecond):
			// No event received in time, which is okay for test
		}
	})
}
