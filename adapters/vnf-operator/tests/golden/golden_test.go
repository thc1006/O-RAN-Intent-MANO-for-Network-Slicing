package golden

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	manov1alpha1 "github.com/o-ran/intent-mano/adapters/vnf-operator/api/v1alpha1"
	"github.com/o-ran/intent-mano/adapters/vnf-operator/pkg/translator"
)

// TestGoldenRANPackage tests RAN VNF to Porch package translation
func TestGoldenRANPackage(t *testing.T) {
	// Load input VNF
	vnf := loadVNFFromFile(t, "testdata/input/ran_vnf.yaml")

	// Translate to Porch package
	translator := translator.NewPorchTranslator()
	pkg, err := translator.TranslateVNF(vnf)
	if err != nil {
		t.Fatalf("Failed to translate VNF: %v", err)
	}

	// Compare with expected output
	expectedKptfile := loadExpectedFile(t, "testdata/expected/ran_package/Kptfile")
	actualKptfile := marshalToYAML(t, pkg.Kptfile)

	if !bytes.Equal(expectedKptfile, actualKptfile) {
		t.Errorf("Kptfile mismatch.\nExpected:\n%s\nActual:\n%s",
			string(expectedKptfile), string(actualKptfile))
	}

	// Verify resources count
	if len(pkg.Resources) != 3 {
		t.Errorf("Expected 3 resources for RAN, got %d", len(pkg.Resources))
	}

	// Verify deployment resource
	deployment := findResourceByKind(pkg.Resources, "Deployment")
	if deployment == nil {
		t.Error("Deployment resource not found")
	} else {
		validateRANDeployment(t, deployment, vnf)
	}
}

// TestGoldenCNPackage tests CN VNF to Porch package translation
func TestGoldenCNPackage(t *testing.T) {
	// Load input VNF
	vnf := loadVNFFromFile(t, "testdata/input/cn_vnf.yaml")

	// Translate to Porch package
	translator := translator.NewPorchTranslator()
	pkg, err := translator.TranslateVNF(vnf)
	if err != nil {
		t.Fatalf("Failed to translate VNF: %v", err)
	}

	// Verify StatefulSet for CN
	statefulset := findResourceByKind(pkg.Resources, "StatefulSet")
	if statefulset == nil {
		t.Error("StatefulSet resource not found for CN VNF")
	} else {
		validateCNStatefulSet(t, statefulset, vnf)
	}
}

// TestGoldenTNPackage tests TN VNF to Porch package translation
func TestGoldenTNPackage(t *testing.T) {
	// Load input VNF
	vnf := loadVNFFromFile(t, "testdata/input/tn_vnf.yaml")

	// Translate to Porch package
	translator := translator.NewPorchTranslator()
	pkg, err := translator.TranslateVNF(vnf)
	if err != nil {
		t.Fatalf("Failed to translate VNF: %v", err)
	}

	// Verify DaemonSet for TN
	daemonset := findResourceByKind(pkg.Resources, "DaemonSet")
	if daemonset == nil {
		t.Error("DaemonSet resource not found for TN VNF")
	} else {
		validateTNDaemonSet(t, daemonset, vnf)
	}
}

// TestQoSParameterMapping tests QoS parameter translation
func TestQoSParameterMapping(t *testing.T) {
	testCases := []struct {
		name              string
		vnfFile          string
		expectedBandwidth string
		expectedLatency   string
	}{
		{
			name:              "High bandwidth eMBB",
			vnfFile:          "testdata/input/ran_vnf.yaml",
			expectedBandwidth: "5.00",
			expectedLatency:   "9.00",
		},
		{
			name:              "Balanced profile",
			vnfFile:          "testdata/input/cn_vnf.yaml",
			expectedBandwidth: "3.00",
			expectedLatency:   "9.00",
		},
		{
			name:              "Low latency uRLLC",
			vnfFile:          "testdata/input/tn_vnf.yaml",
			expectedBandwidth: "1.00",
			expectedLatency:   "1.00",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vnf := loadVNFFromFile(t, tc.vnfFile)
			translator := translator.NewPorchTranslator()
			pkg, err := translator.TranslateVNF(vnf)
			if err != nil {
				t.Fatalf("Failed to translate VNF: %v", err)
			}

			// Find QoS ConfigMap
			configMap := findResourceByKind(pkg.Resources, "ConfigMap")
			if configMap == nil {
				t.Error("ConfigMap resource not found")
				return
			}

			// Verify QoS values
			spec, ok := configMap.Spec["data"].(map[string]interface{})
			if !ok {
				t.Error("ConfigMap data not found")
				return
			}

			if bandwidth, ok := spec["bandwidth"].(string); !ok || bandwidth != tc.expectedBandwidth {
				t.Errorf("Expected bandwidth %s, got %s", tc.expectedBandwidth, bandwidth)
			}

			if latency, ok := spec["latency"].(string); !ok || latency != tc.expectedLatency {
				t.Errorf("Expected latency %s, got %s", tc.expectedLatency, latency)
			}
		})
	}
}

// Helper functions

func loadVNFFromFile(t *testing.T, path string) *manov1alpha1.VNF {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read VNF file: %v", err)
	}

	vnf := &manov1alpha1.VNF{}
	if err := yaml.Unmarshal(data, vnf); err != nil {
		t.Fatalf("Failed to unmarshal VNF: %v", err)
	}

	return vnf
}

func loadExpectedFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		// For initial test runs, create expected files
		if os.IsNotExist(err) {
			t.Logf("Expected file not found: %s (will be created on first run)", path)
			return []byte{}
		}
		t.Fatalf("Failed to read expected file: %v", err)
	}
	return data
}

func marshalToYAML(t *testing.T, obj interface{}) []byte {
	t.Helper()
	data, err := yaml.Marshal(obj)
	if err != nil {
		t.Fatalf("Failed to marshal to YAML: %v", err)
	}
	return data
}

func findResourceByKind(resources []translator.Resource, kind string) *translator.Resource {
	for i := range resources {
		if resources[i].Kind == kind {
			return &resources[i]
		}
	}
	return nil
}

func validateRANDeployment(t *testing.T, deployment *translator.Resource, vnf *manov1alpha1.VNF) {
	t.Helper()

	// Check metadata
	metadata := deployment.Metadata
	if name, ok := metadata["name"].(string); !ok || name != vnf.Name+"-ran" {
		t.Errorf("Deployment name mismatch")
	}

	// Check spec
	spec := deployment.Spec
	if spec == nil {
		t.Error("Deployment spec is nil")
		return
	}

	// Validate container resources
	template, ok := spec["template"].(map[string]interface{})
	if !ok {
		t.Error("Deployment template not found")
		return
	}

	podSpec, ok := template["spec"].(map[string]interface{})
	if !ok {
		t.Error("Pod spec not found")
		return
	}

	containers, ok := podSpec["containers"].([]map[string]interface{})
	if !ok || len(containers) == 0 {
		t.Error("Containers not found")
		return
	}

	container := containers[0]
	resources, ok := container["resources"].(map[string]interface{})
	if !ok {
		t.Error("Container resources not found")
		return
	}

	requests, ok := resources["requests"].(map[string]interface{})
	if !ok {
		t.Error("Resource requests not found")
		return
	}

	if cpu, ok := requests["cpu"].(string); !ok || cpu != vnf.Spec.Resources.CPU {
		t.Errorf("CPU request mismatch: expected %s, got %s", vnf.Spec.Resources.CPU, cpu)
	}
}

func validateCNStatefulSet(t *testing.T, statefulset *translator.Resource, vnf *manov1alpha1.VNF) {
	t.Helper()

	// Check for StatefulSet-specific fields
	spec := statefulset.Spec
	if serviceName, ok := spec["serviceName"].(string); !ok || serviceName != vnf.Name+"-upf" {
		t.Error("StatefulSet serviceName mismatch")
	}
}

func validateTNDaemonSet(t *testing.T, daemonset *translator.Resource, vnf *manov1alpha1.VNF) {
	t.Helper()

	// Check DaemonSet has hostNetwork
	template, ok := daemonset.Spec["template"].(map[string]interface{})
	if !ok {
		t.Error("DaemonSet template not found")
		return
	}

	podSpec, ok := template["spec"].(map[string]interface{})
	if !ok {
		t.Error("Pod spec not found")
		return
	}

	if hostNetwork, ok := podSpec["hostNetwork"].(bool); !ok || !hostNetwork {
		t.Error("TN DaemonSet should have hostNetwork: true")
	}
}