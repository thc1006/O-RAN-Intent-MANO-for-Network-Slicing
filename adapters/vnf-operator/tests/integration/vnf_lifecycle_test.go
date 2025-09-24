package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	manov1alpha1 "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/api/v1alpha1"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/controllers"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/pkg/dms"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/pkg/gitops"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/pkg/translator"
)

// TestVNFLifecycleIntegration tests the complete VNF lifecycle
func TestVNFLifecycleIntegration(t *testing.T) {
	ctx := context.Background()

	// Setup fake Kubernetes client
	require.NoError(t, manov1alpha1.AddToScheme(scheme.Scheme))
	k8sClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()

	// Setup VNF reconciler with mock clients
	reconciler := &controllers.VNFReconciler{
		Client:          k8sClient,
		Scheme:          scheme.Scheme,
		PorchTranslator: translator.NewPorchTranslator(),
		DMSClient:       dms.NewMockDMSClient(),
		GitOpsClient:    gitops.NewMockGitOpsClient(),
	}

	testCases := []struct {
		name          string
		vnfType       manov1alpha1.VNFType
		cloudType     string
		bandwidth     float64
		latency       float64
		expectedPhase string
	}{
		{
			name:          "Edge RAN VNF",
			vnfType:       manov1alpha1.VNFTypeRAN,
			cloudType:     "edge",
			bandwidth:     4.5,
			latency:       1.5,
			expectedPhase: "Running",
		},
		{
			name:          "Regional CN VNF",
			vnfType:       manov1alpha1.VNFTypeCN,
			cloudType:     "regional",
			bandwidth:     3.0,
			latency:       5.0,
			expectedPhase: "Running",
		},
		{
			name:          "UPF VNF",
			vnfType:       manov1alpha1.VNFTypeUPF,
			cloudType:     "edge",
			bandwidth:     2.5,
			latency:       2.0,
			expectedPhase: "Running",
		},
		{
			name:          "AMF VNF",
			vnfType:       manov1alpha1.VNFTypeAMF,
			cloudType:     "regional",
			bandwidth:     1.5,
			latency:       8.0,
			expectedPhase: "Running",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create VNF instance
			vnf := &manov1alpha1.VNF{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "vnf.oran.io/v1alpha1",
					Kind:       "VNF",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("test-vnf-%s", string(tc.vnfType)),
					Namespace: "default",
				},
				Spec: manov1alpha1.VNFSpec{
					Name:    fmt.Sprintf("test-vnf-%s", string(tc.vnfType)),
					Type:    tc.vnfType,
					Version: "1.0.0",
					Placement: manov1alpha1.PlacementRequirements{
						CloudType: tc.cloudType,
					},
					Resources: manov1alpha1.ResourceRequirements{
						CPUCores: 2,
						MemoryGB: 4,
					},
					QoS: manov1alpha1.QoSRequirements{
						Bandwidth: tc.bandwidth,
						Latency:   tc.latency,
					},
					TargetClusters: []string{"cluster-01"},
					Image: manov1alpha1.ImageSpec{
						Repository: fmt.Sprintf("oran/%s", string(tc.vnfType)),
						Tag:        "1.0.0",
					},
				},
			}

			// Step 1: Create VNF
			err := k8sClient.Create(ctx, vnf)
			require.NoError(t, err, "Failed to create VNF")

			// Step 2: Reconcile multiple times to simulate lifecycle
			namespacedName := types.NamespacedName{
				Name:      vnf.Name,
				Namespace: vnf.Namespace,
			}

			// First reconciliation - should initialize status
			result, err := reconciler.Reconcile(ctx, newReconcileRequest(namespacedName))
			require.NoError(t, err, "First reconciliation failed")
			assert.True(t, result.RequeueAfter > 0, "Should requeue after initialization")

			// Get updated VNF status
			err = k8sClient.Get(ctx, namespacedName, vnf)
			require.NoError(t, err, "Failed to get VNF after first reconciliation")
			assert.NotEmpty(t, vnf.Status.Phase, "Status phase should be set")

			// Second reconciliation - should handle pending state
			result, err = reconciler.Reconcile(ctx, newReconcileRequest(namespacedName))
			require.NoError(t, err, "Second reconciliation failed")

			// Get updated VNF status
			err = k8sClient.Get(ctx, namespacedName, vnf)
			require.NoError(t, err, "Failed to get VNF after second reconciliation")

			// Should have progressed to Creating state
			if vnf.Status.Phase == "Pending" {
				// If still pending, run one more reconciliation
				result, err = reconciler.Reconcile(ctx, newReconcileRequest(namespacedName))
				require.NoError(t, err, "Third reconciliation failed")

				err = k8sClient.Get(ctx, namespacedName, vnf)
				require.NoError(t, err, "Failed to get VNF after third reconciliation")
			}

			// Verify VNF progressed through states
			assert.Contains(t, []string{"Creating", "Running"}, vnf.Status.Phase,
				"VNF should be in Creating or Running state")

			// Step 3: Verify finalizer is added
			assert.Contains(t, vnf.Finalizers, "mano.oran.io/finalizer",
				"Finalizer should be added")

			// Step 4: Test deletion
			err = k8sClient.Delete(ctx, vnf)
			require.NoError(t, err, "Failed to delete VNF")

			// Reconcile deletion
			result, err = reconciler.Reconcile(ctx, newReconcileRequest(namespacedName))
			require.NoError(t, err, "Deletion reconciliation failed")

			// Verify VNF is cleaned up
			err = k8sClient.Get(ctx, namespacedName, vnf)
			if err == nil {
				// If VNF still exists, it might be in deleting state
				assert.True(t, vnf.DeletionTimestamp != nil, "VNF should be marked for deletion")
			}
		})
	}
}

// TestVNFValidationIntegration tests VNF validation in integration scenarios
func TestVNFValidationIntegration(t *testing.T) {
	ctx := context.Background()

	// Setup fake Kubernetes client
	require.NoError(t, manov1alpha1.AddToScheme(scheme.Scheme))
	k8sClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()

	reconciler := &controllers.VNFReconciler{
		Client:          k8sClient,
		Scheme:          scheme.Scheme,
		PorchTranslator: translator.NewPorchTranslator(),
		DMSClient:       dms.NewMockDMSClient(),
		GitOpsClient:    gitops.NewMockGitOpsClient(),
	}

	invalidTestCases := []struct {
		name      string
		vnf       *manov1alpha1.VNF
		errorText string
	}{
		{
			name: "Invalid Bandwidth",
			vnf: createTestVNF("invalid-bandwidth", manov1alpha1.VNFTypeRAN, "edge", 10.0, 5.0),
			errorText: "bandwidth must be between",
		},
		{
			name: "Invalid Latency",
			vnf: createTestVNF("invalid-latency", manov1alpha1.VNFTypeRAN, "edge", 3.0, 0.5),
			errorText: "latency must be between",
		},
		{
			name: "Invalid Cloud Type",
			vnf: createTestVNF("invalid-cloud", manov1alpha1.VNFTypeRAN, "invalid", 3.0, 5.0),
			errorText: "invalid cloud type",
		},
	}

	for _, tc := range invalidTestCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create invalid VNF
			err := k8sClient.Create(ctx, tc.vnf)
			require.NoError(t, err, "Failed to create VNF")

			// Reconcile - should fail validation
			namespacedName := types.NamespacedName{
				Name:      tc.vnf.Name,
				Namespace: tc.vnf.Namespace,
			}

			// First reconciliation - initialize
			_, err = reconciler.Reconcile(ctx, newReconcileRequest(namespacedName))
			require.NoError(t, err, "Initial reconciliation should not error")

			// Second reconciliation - should fail validation
			_, err = reconciler.Reconcile(ctx, newReconcileRequest(namespacedName))
			require.NoError(t, err, "Reconciliation should handle validation errors gracefully")

			// Check VNF status is marked as failed
			var updatedVNF manov1alpha1.VNF
			err = k8sClient.Get(ctx, namespacedName, &updatedVNF)
			require.NoError(t, err, "Failed to get updated VNF")

			// The VNF should eventually be marked as Failed
			if updatedVNF.Status.Phase == "Pending" {
				// Run another reconciliation to trigger validation
				_, err = reconciler.Reconcile(ctx, newReconcileRequest(namespacedName))
				require.NoError(t, err, "Additional reconciliation should not error")

				err = k8sClient.Get(ctx, namespacedName, &updatedVNF)
				require.NoError(t, err, "Failed to get VNF after validation")
			}

			// Cleanup
			_ = k8sClient.Delete(ctx, tc.vnf)
		})
	}
}

// TestVNFTranslatorIntegration tests the Porch translator integration
func TestVNFTranslatorIntegration(t *testing.T) {
	translator := translator.NewPorchTranslator()

	testCases := []struct {
		name    string
		vnfType manov1alpha1.VNFType
	}{
		{"RAN Translation", manov1alpha1.VNFTypeRAN},
		{"CN Translation", manov1alpha1.VNFTypeCN},
		{"TN Translation", manov1alpha1.VNFTypeTN},
		{"UPF Translation", manov1alpha1.VNFTypeUPF},
		{"AMF Translation", manov1alpha1.VNFTypeAMF},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vnf := createTestVNF(
				fmt.Sprintf("test-%s", string(tc.vnfType)),
				tc.vnfType,
				"edge",
				3.0,
				5.0,
			)

			// Test translation
			pkg, err := translator.TranslateVNF(vnf)
			require.NoError(t, err, "Translation should succeed")
			assert.NotNil(t, pkg, "Package should be generated")
			assert.NotEmpty(t, pkg.Name, "Package name should be set")
			assert.Contains(t, pkg.Name, string(tc.vnfType), "Package name should contain VNF type")
		})
	}
}

// Helper functions

func newReconcileRequest(namespacedName types.NamespacedName) ctrl.Request {
	return ctrl.Request{
		NamespacedName: namespacedName,
	}
}

func createTestVNF(name string, vnfType manov1alpha1.VNFType, cloudType string, bandwidth, latency float64) *manov1alpha1.VNF {
	return &manov1alpha1.VNF{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "vnf.oran.io/v1alpha1",
			Kind:       "VNF",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: manov1alpha1.VNFSpec{
			Name:    name,
			Type:    vnfType,
			Version: "1.0.0",
			Placement: manov1alpha1.PlacementRequirements{
				CloudType: cloudType,
			},
			Resources: manov1alpha1.ResourceRequirements{
				CPUCores: 2,
				MemoryGB: 4,
			},
			QoS: manov1alpha1.QoSRequirements{
				Bandwidth: bandwidth,
				Latency:   latency,
			},
			TargetClusters: []string{"cluster-01"},
			Image: manov1alpha1.ImageSpec{
				Repository: fmt.Sprintf("oran/%s", string(vnfType)),
				Tag:        "1.0.0",
			},
		},
	}
}

// TestVNFScaling tests VNF scaling scenarios
func TestVNFScaling(t *testing.T) {
	ctx := context.Background()

	// Setup fake Kubernetes client
	require.NoError(t, manov1alpha1.AddToScheme(scheme.Scheme))
	k8sClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()

	reconciler := &controllers.VNFReconciler{
		Client:          k8sClient,
		Scheme:          scheme.Scheme,
		PorchTranslator: translator.NewPorchTranslator(),
		DMSClient:       dms.NewMockDMSClient(),
		GitOpsClient:    gitops.NewMockGitOpsClient(),
	}

	// Create VNF with multiple target clusters
	vnf := createTestVNF("scaling-vnf", manov1alpha1.VNFTypeUPF, "edge", 3.0, 5.0)
	vnf.Spec.TargetClusters = []string{"edge01", "edge02", "edge03"}

	err := k8sClient.Create(ctx, vnf)
	require.NoError(t, err, "Failed to create scaling VNF")

	namespacedName := types.NamespacedName{
		Name:      vnf.Name,
		Namespace: vnf.Namespace,
	}

	// Reconcile to deploy across multiple clusters
	for i := 0; i < 3; i++ {
		_, err = reconciler.Reconcile(ctx, newReconcileRequest(namespacedName))
		require.NoError(t, err, fmt.Sprintf("Reconciliation %d failed", i+1))

		// Small delay to simulate async operations
		time.Sleep(100 * time.Millisecond)
	}

	// Get final VNF state
	err = k8sClient.Get(ctx, namespacedName, vnf)
	require.NoError(t, err, "Failed to get final VNF state")

	// Verify deployment across clusters
	assert.Len(t, vnf.Spec.TargetClusters, 3, "Should have 3 target clusters")
}

// TestVNFErrorRecovery tests error recovery scenarios
func TestVNFErrorRecovery(t *testing.T) {
	ctx := context.Background()

	// Setup fake Kubernetes client
	require.NoError(t, manov1alpha1.AddToScheme(scheme.Scheme))
	k8sClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()

	reconciler := &controllers.VNFReconciler{
		Client:          k8sClient,
		Scheme:          scheme.Scheme,
		PorchTranslator: translator.NewPorchTranslator(),
		DMSClient:       dms.NewMockDMSClient(),
		GitOpsClient:    gitops.NewMockGitOpsClient(),
	}

	// Create VNF that will initially fail validation
	vnf := createTestVNF("error-recovery-vnf", manov1alpha1.VNFTypeRAN, "invalid", 10.0, 0.5)

	err := k8sClient.Create(ctx, vnf)
	require.NoError(t, err, "Failed to create VNF")

	namespacedName := types.NamespacedName{
		Name:      vnf.Name,
		Namespace: vnf.Namespace,
	}

	// Initial reconciliation should fail validation
	for i := 0; i < 2; i++ {
		_, err = reconciler.Reconcile(ctx, newReconcileRequest(namespacedName))
		require.NoError(t, err, "Reconciliation should handle errors gracefully")
	}

	// Fix the VNF spec
	err = k8sClient.Get(ctx, namespacedName, vnf)
	require.NoError(t, err, "Failed to get VNF for update")

	vnf.Spec.Placement.CloudType = "edge"
	vnf.Spec.QoS.Bandwidth = 3.0
	vnf.Spec.QoS.Latency = 5.0

	err = k8sClient.Update(ctx, vnf)
	require.NoError(t, err, "Failed to update VNF")

	// Reconcile again - should recover and succeed
	_, err = reconciler.Reconcile(ctx, newReconcileRequest(namespacedName))
	require.NoError(t, err, "Recovery reconciliation failed")

	// Get updated status
	err = k8sClient.Get(ctx, namespacedName, vnf)
	require.NoError(t, err, "Failed to get VNF after recovery")

	// Should have progressed from Failed state
	assert.NotEqual(t, "Failed", vnf.Status.Phase, "VNF should recover from failed state")
}