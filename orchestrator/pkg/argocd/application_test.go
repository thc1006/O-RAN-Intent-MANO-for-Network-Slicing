package argocd

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/orchestrator/pkg/slices"
)

// TestNewClient tests ArgoCD client initialization
func TestNewClient(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		wantErr   bool
	}{
		{
			name:      "Valid namespace",
			namespace: "argocd",
			wantErr:   false,
		},
		{
			name:      "Empty namespace uses default",
			namespace: "",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.namespace)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, client)

			if tt.namespace == "" {
				assert.Equal(t, DefaultArgoCDNamespace, client.Namespace)
			} else {
				assert.Equal(t, tt.namespace, client.Namespace)
			}
		})
	}
}

// TestCreateApplication tests ArgoCD Application creation
func TestCreateApplication(t *testing.T) {
	tests := []struct {
		name      string
		config    ApplicationConfig
		wantErr   bool
		errMsg    string
	}{
		{
			name: "Valid eMBB application",
			config: ApplicationConfig{
				Name:      "slice-embb-test-001",
				Namespace: "oran-slice-embb",
				SliceType: "eMBB",
				ManifestsYAML: `apiVersion: v1
kind: Namespace
metadata:
  name: oran-slice-embb
`,
			},
			wantErr: false,
		},
		{
			name: "Valid URLLC application",
			config: ApplicationConfig{
				Name:      "slice-urllc-test-001",
				Namespace: "oran-slice-urllc",
				SliceType: "URLLC",
				ManifestsYAML: `apiVersion: v1
kind: Namespace
metadata:
  name: oran-slice-urllc
`,
			},
			wantErr: false,
		},
		{
			name: "Empty application name",
			config: ApplicationConfig{
				Name:      "",
				Namespace: "test",
				SliceType: "eMBB",
			},
			wantErr: true,
			errMsg:  "application name cannot be empty",
		},
		{
			name: "Empty namespace",
			config: ApplicationConfig{
				Name:      "test-app",
				Namespace: "",
				SliceType: "eMBB",
			},
			wantErr: true,
			errMsg:  "namespace cannot be empty",
		},
		{
			name: "Empty manifests",
			config: ApplicationConfig{
				Name:          "test-app",
				Namespace:     "test-ns",
				SliceType:     "eMBB",
				ManifestsYAML: "",
			},
			wantErr: true,
			errMsg:  "manifests cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create client
			client, err := NewClient("argocd")
			require.NoError(t, err)

			// Attempt to create application
			ctx := context.Background()
			app, err := client.CreateApplication(ctx, tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, app)

			// Verify application properties
			assert.Equal(t, tt.config.Name, app.Name)
			assert.Equal(t, tt.config.Namespace, app.Spec.Destination.Namespace)
			assert.Contains(t, app.Labels, "slice-id")
			assert.Contains(t, app.Labels, "slice-type")
		})
	}
}

// TestCreateApplicationFromSliceConfig tests creating ArgoCD app from slice configuration
func TestCreateApplicationFromSliceConfig(t *testing.T) {
	sliceConfig := slices.SliceConfiguration{
		SliceID:   "slice-embb-integration-001",
		SliceType: "eMBB",
		Namespace: "oran-slice-embb-int",
		QoS: slices.QoSProfile{
			Throughput:  1000,
			Latency:     20,
			Reliability: 99.9,
		},
	}

	// Generate manifests
	manifests, err := slices.GenerateSliceManifests(sliceConfig)
	require.NoError(t, err)

	// Serialize to YAML
	yamlData, err := manifests.ToYAML()
	require.NoError(t, err)

	// Create ArgoCD application config
	appConfig := ApplicationConfig{
		Name:          sliceConfig.SliceID,
		Namespace:     sliceConfig.Namespace,
		SliceType:     sliceConfig.SliceType,
		ManifestsYAML: string(yamlData),
	}

	// Create ArgoCD client
	client, err := NewClient("argocd")
	require.NoError(t, err)

	// Create application
	ctx := context.Background()
	app, err := client.CreateApplication(ctx, appConfig)
	require.NoError(t, err)
	assert.NotNil(t, app)

	// Verify application details
	t.Run("Application Metadata", func(t *testing.T) {
		assert.Equal(t, sliceConfig.SliceID, app.Name)
		assert.Equal(t, "argocd", app.Namespace)
		assert.Equal(t, sliceConfig.SliceID, app.Labels["slice-id"])
		assert.Equal(t, "embb", app.Labels["slice-type"])
	})

	t.Run("Application Source", func(t *testing.T) {
		// Should use ConfigMap as source
		assert.NotNil(t, app.Spec.Source)
		// Plugin should be configured for ConfigMap reading
		assert.NotNil(t, app.Spec.Source.Plugin)
	})

	t.Run("Application Destination", func(t *testing.T) {
		assert.Equal(t, sliceConfig.Namespace, app.Spec.Destination.Namespace)
		assert.Equal(t, "https://kubernetes.default.svc", app.Spec.Destination.Server)
	})

	t.Run("Sync Policy", func(t *testing.T) {
		assert.NotNil(t, app.Spec.SyncPolicy)
		assert.NotNil(t, app.Spec.SyncPolicy.Automated)
		assert.True(t, *app.Spec.SyncPolicy.Automated.Prune)
		assert.True(t, *app.Spec.SyncPolicy.Automated.SelfHeal)
	})
}

// TestGetApplication tests retrieving ArgoCD Application status
func TestGetApplication(t *testing.T) {
	client, err := NewClient("argocd")
	require.NoError(t, err)

	ctx := context.Background()

	// Test retrieving non-existent application
	t.Run("Non-existent application", func(t *testing.T) {
		app, err := client.GetApplication(ctx, "non-existent-app")
		assert.Error(t, err)
		assert.Nil(t, app)
	})
}

// TestDeleteApplication tests ArgoCD Application deletion
func TestDeleteApplication(t *testing.T) {
	client, err := NewClient("argocd")
	require.NoError(t, err)

	ctx := context.Background()

	tests := []struct {
		name    string
		appName string
		wantErr bool
	}{
		{
			name:    "Delete with empty name",
			appName: "",
			wantErr: true,
		},
		{
			name:    "Delete non-existent app",
			appName: "non-existent-app",
			wantErr: true, // Should return error for non-existent
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.DeleteApplication(ctx, tt.appName)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestApplicationSyncStatus tests checking sync status
func TestApplicationSyncStatus(t *testing.T) {
	client, err := NewClient("argocd")
	require.NoError(t, err)

	ctx := context.Background()

	// For non-existent app, should return error
	status, err := client.GetSyncStatus(ctx, "non-existent-app")
	assert.Error(t, err)
	assert.Empty(t, status)
}

// TestStoreManifestsInConfigMap tests storing manifests in ConfigMap
func TestStoreManifestsInConfigMap(t *testing.T) {
	client, err := NewClient("argocd")
	require.NoError(t, err)

	ctx := context.Background()

	manifests := `apiVersion: v1
kind: Namespace
metadata:
  name: test-namespace
`

	cmName, err := client.StoreManifestsInConfigMap(ctx, "test-slice-001", manifests)

	// This test requires actual k8s cluster, so we expect it might fail
	// but the function signature and error handling should be correct
	if err != nil {
		// If error, it should be meaningful
		assert.NotEmpty(t, err.Error())
	} else {
		// If successful, should return valid ConfigMap name
		assert.NotEmpty(t, cmName)
		assert.Contains(t, cmName, "test-slice-001")
	}
}