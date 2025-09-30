package slices

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
)

// TestGenerateSliceManifests tests the manifest generation for network slices
func TestGenerateSliceManifests(t *testing.T) {
	tests := []struct {
		name      string
		sliceType string
		sliceID   string
		qos       QoSProfile
		wantErr   bool
	}{
		{
			name:      "eMBB slice with high throughput",
			sliceType: "eMBB",
			sliceID:   "slice-embb-test-001",
			qos: QoSProfile{
				Throughput:  1000, // Mbps
				Latency:     20,   // ms
				Reliability: 99.9,
			},
			wantErr: false,
		},
		{
			name:      "URLLC slice with low latency",
			sliceType: "URLLC",
			sliceID:   "slice-urllc-test-001",
			qos: QoSProfile{
				Throughput:  10,
				Latency:     1, // Ultra-low
				Reliability: 99.999,
			},
			wantErr: false,
		},
		{
			name:      "mMTC slice for IoT",
			sliceType: "mMTC",
			sliceID:   "slice-mmtc-test-001",
			qos: QoSProfile{
				Throughput:  1,
				Latency:     100,
				Reliability: 99.0,
				Connections: 10000,
			},
			wantErr: false,
		},
		{
			name:      "Invalid slice type",
			sliceType: "INVALID",
			sliceID:   "slice-invalid-001",
			qos:       QoSProfile{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := SliceConfiguration{
				SliceID:   tt.sliceID,
				SliceType: tt.sliceType,
				Namespace: "oran-slice-" + tt.sliceID,
				QoS:       tt.qos,
			}

			manifests, err := GenerateSliceManifests(config)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, manifests)

			// Verify essential resources are generated
			assert.True(t, len(manifests.Resources) > 0, "Should generate at least one resource")

			// Test specific resource types
			t.Run("Namespace", func(t *testing.T) {
				ns := manifests.Namespace
				require.NotNil(t, ns)
				assert.Equal(t, config.Namespace, ns.Name)
				assert.Contains(t, ns.Labels, "slice-id")
				assert.Equal(t, tt.sliceID, ns.Labels["slice-id"])
			})

			t.Run("ConfigMap with QoS", func(t *testing.T) {
				cm := manifests.QoSConfigMap
				require.NotNil(t, cm)
				assert.Equal(t, config.Namespace, cm.Namespace)
				assert.Contains(t, cm.Data, "qos.yaml")

				// Verify QoS data contains throughput
				assert.Contains(t, cm.Data["qos.yaml"], "throughput")
			})

			t.Run("Deployment", func(t *testing.T) {
				deploy := manifests.Deployment
				require.NotNil(t, deploy)
				assert.Equal(t, config.Namespace, deploy.Namespace)
				assert.Contains(t, deploy.Spec.Template.Spec.Containers[0].Name, "slice")
			})
		})
	}
}

// TestQoSProfileValidation tests QoS profile validation
func TestQoSProfileValidation(t *testing.T) {
	tests := []struct {
		name    string
		qos     QoSProfile
		wantErr bool
	}{
		{
			name: "Valid eMBB QoS",
			qos: QoSProfile{
				Throughput:  100,
				Latency:     50,
				Reliability: 99.5,
			},
			wantErr: false,
		},
		{
			name: "Invalid throughput (negative)",
			qos: QoSProfile{
				Throughput:  -10,
				Latency:     10,
				Reliability: 99.0,
			},
			wantErr: true,
		},
		{
			name: "Invalid latency (zero)",
			qos: QoSProfile{
				Throughput:  100,
				Latency:     0,
				Reliability: 99.0,
			},
			wantErr: true,
		},
		{
			name: "Invalid reliability (>100)",
			qos: QoSProfile{
				Throughput:  100,
				Latency:     10,
				Reliability: 100.1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.qos.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSliceTypeSpecificResources tests slice type specific resource requirements
func TestSliceTypeSpecificResources(t *testing.T) {
	tests := []struct {
		name           string
		sliceType      string
		expectedCPU    string
		expectedMemory string
	}{
		{
			name:           "eMBB requires high resources",
			sliceType:      "eMBB",
			expectedCPU:    "2000m", // 2 cores
			expectedMemory: "4Gi",
		},
		{
			name:           "URLLC requires guaranteed resources",
			sliceType:      "URLLC",
			expectedCPU:    "4000m", // 4 cores (guaranteed)
			expectedMemory: "8Gi",
		},
		{
			name:           "mMTC requires low resources",
			sliceType:      "mMTC",
			expectedCPU:    "500m", // 0.5 core
			expectedMemory: "1Gi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := SliceConfiguration{
				SliceID:   "test-slice-001",
				SliceType: tt.sliceType,
				Namespace: "oran-test",
				QoS: QoSProfile{
					Throughput:  100,
					Latency:     10,
					Reliability: 99.0,
				},
			}

			manifests, err := GenerateSliceManifests(config)
			require.NoError(t, err)

			// Check deployment resource requirements
			deploy := manifests.Deployment
			require.NotNil(t, deploy)

			container := deploy.Spec.Template.Spec.Containers[0]

			// Use MilliValue() for CPU comparison (Kubernetes normalizes format)
			expectedCPU, _ := resource.ParseQuantity(tt.expectedCPU)
			actualCPU := container.Resources.Requests.Cpu()
			assert.Equal(t, expectedCPU.MilliValue(), actualCPU.MilliValue(), "CPU should match")

			// Memory comparison can use direct string match
			assert.Equal(t, tt.expectedMemory, container.Resources.Requests.Memory().String())
		})
	}
}

// TestManifestSerialization tests YAML serialization of manifests
func TestManifestSerialization(t *testing.T) {
	config := SliceConfiguration{
		SliceID:   "slice-test-001",
		SliceType: "eMBB",
		Namespace: "oran-test",
		QoS: QoSProfile{
			Throughput:  1000,
			Latency:     20,
			Reliability: 99.9,
		},
	}

	manifests, err := GenerateSliceManifests(config)
	require.NoError(t, err)

	// Test YAML serialization
	yamlData, err := manifests.ToYAML()
	require.NoError(t, err)
	assert.NotEmpty(t, yamlData)

	// Verify YAML contains essential fields
	assert.Contains(t, string(yamlData), "apiVersion")
	assert.Contains(t, string(yamlData), "kind:")
	assert.Contains(t, string(yamlData), config.SliceID)
}