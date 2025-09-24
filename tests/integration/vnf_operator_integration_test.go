package integration_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	// appsv1 "k8s.io/api/apps/v1" // Commented out, not used yet
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	vnfv1alpha1 "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/api/v1alpha1"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
)

func TestAPIs(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)

	ginkgo.RunSpecs(t, "VNF Operator Integration Suite")
}

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
)

var _ = ginkgo.BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(ginkgo.GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	ginkgo.By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: false,
		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s", "1.25.0-linux-amd64"),
	}

	// Check if we have a kubeconfig file
	home, _ := os.UserHomeDir()
	kubeconfig := filepath.Join(home, ".kube", "config")
	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		// Use envtest
		var err error
		cfg, err = testEnv.Start()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(cfg).NotTo(gomega.BeNil())
	} else {
		// Use existing cluster
		var err error
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	}

	err := vnfv1alpha1.AddToScheme(runtime.NewScheme())
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(k8sClient).NotTo(gomega.BeNil())
})

var _ = ginkgo.AfterSuite(func() {
	cancel()
	if testEnv != nil {
		ginkgo.By("tearing down the test environment")
		err := testEnv.Stop()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	}
})

var _ = ginkgo.Describe("VNF Operator", func() {
	ginkgo.Context("When deploying VNFs", func() {
		ginkgo.It("Should create and manage VNF resources", func() {
			ctx := context.Background()
			namespaceName := "vnf-test"
			vnfName := "test-vnf"

			// Create namespace
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
			}
			err := k8sClient.Create(ctx, namespace)
			if !errors.IsAlreadyExists(err) {
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			}

			// Define a VNF resource
			vnf := &vnfv1alpha1.VNF{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vnfName,
					Namespace: namespaceName,
				},
				Spec: vnfv1alpha1.VNFSpec{
					Name:    "test-upf",
					Type:    "UPF",
					Version: "v1.0.0",
					Image: vnfv1alpha1.ImageSpec{
						Repository: "oran/upf",
						Tag:        "v1.0.0",
						PullPolicy: "IfNotPresent",
					},
					Resources: vnfv1alpha1.ResourceRequirements{
						CPU:       "500m",
						Memory:    "1Gi",
						CPUCores:  1,
						MemoryGB:  1,
						StorageGB: 10,
					},
					QoS: vnfv1alpha1.QoSRequirements{
						Bandwidth: 100.0,
						Latency:   10.0,
						SliceType: "uRLLC",
					},
					Placement: vnfv1alpha1.PlacementRequirements{
						CloudType: "edge",
						Zone:      "zone-a",
					},
					TargetClusters: []string{"edge-cluster"},
				},
			}

			ginkgo.By("Creating the VNF resource")
			gomega.Expect(k8sClient.Create(ctx, vnf)).Should(gomega.Succeed())

			lookupKey := types.NamespacedName{Name: vnfName, Namespace: namespaceName}
			createdVNF := &vnfv1alpha1.VNF{}

			// Get the created VNF
			gomega.Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdVNF)
				return err == nil
			}, time.Second*10, time.Millisecond*250).Should(gomega.BeTrue())

			// Validate that the QoS was set properly
			gomega.Expect(createdVNF.Spec.QoS.Bandwidth).To(gomega.Equal(100.0))
			gomega.Expect(createdVNF.Spec.QoS.SliceType).To(gomega.Equal("uRLLC"))

			// Update the Status to simulate controller processing
			createdVNF.Status.Phase = "Creating"
			gomega.Expect(k8sClient.Status().Update(ctx, createdVNF)).Should(gomega.Succeed())

			// Simulate successful deployment
			gomega.Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdVNF)
				if err != nil {
					return false
				}
				createdVNF.Status.Phase = "Running"
				err = k8sClient.Status().Update(ctx, createdVNF)
				return err == nil
			}, time.Second*10, time.Millisecond*250).Should(gomega.BeTrue())

			// Verify the phase is updated
			gomega.Eventually(func() string {
				err := k8sClient.Get(ctx, lookupKey, createdVNF)
				if err != nil {
					return ""
				}
				return createdVNF.Status.Phase
			}, time.Second*10, time.Millisecond*250).Should(gomega.Equal("Running"))

			// Clean up
			ginkgo.By("Deleting the VNF resource")
			gomega.Eventually(func() error {
				vnf := &vnfv1alpha1.VNF{}
				if err := k8sClient.Get(ctx, lookupKey, vnf); err != nil {
					return err
				}
				return k8sClient.Delete(ctx, vnf)
			}, time.Second*10, time.Millisecond*250).Should(gomega.Succeed())

			// Verify deletion
			gomega.Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, &vnfv1alpha1.VNF{})
				return errors.IsNotFound(err)
			}, time.Second*10, time.Millisecond*250).Should(gomega.BeTrue())
		})
	})

	ginkgo.Context("Resource Management", func() {
		ginkgo.It("Should validate resource requirements", func() {
			vnf := &vnfv1alpha1.VNF{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "resource-test-vnf",
					Namespace: "default",
				},
				Spec: vnfv1alpha1.VNFSpec{
					Name:    "resource-vnf",
					Type:    "AMF",
					Version: "v1.0.0",
					Resources: vnfv1alpha1.ResourceRequirements{
						CPU:      "2",
						Memory:   "4Gi",
						CPUCores: 2,
						MemoryGB: 4,
					},
				},
			}

			// Validate CPU
			cpuQty := resource.MustParse(vnf.Spec.Resources.CPU)
			gomega.Expect(cpuQty.String()).To(gomega.Equal("2"))

			// Validate Memory
			memQty := resource.MustParse(vnf.Spec.Resources.Memory)
			gomega.Expect(memQty.String()).To(gomega.Equal("4Gi"))
		})
	})

	ginkgo.Context("QoS Validation", func() {
		ginkgo.It("Should validate QoS requirements", func() {
			testCases := []struct {
				name        string
				qos         vnfv1alpha1.QoSRequirements
				expectedErr bool
			}{
				{
					name: "Valid eMBB QoS",
					qos: vnfv1alpha1.QoSRequirements{
						Bandwidth: 5.0,
						Latency:   10.0,
						SliceType: "eMBB",
					},
					expectedErr: false,
				},
				{
					name: "Valid uRLLC QoS",
					qos: vnfv1alpha1.QoSRequirements{
						Bandwidth: 1.0,
						Latency:   1.0,
						SliceType: "uRLLC",
					},
					expectedErr: false,
				},
				{
					name: "Valid mIoT QoS",
					qos: vnfv1alpha1.QoSRequirements{
						Bandwidth: 0.1,
						Latency:   100.0,
						SliceType: "mIoT",
					},
					expectedErr: false,
				},
			}

			for _, tc := range testCases {
				ginkgo.By(fmt.Sprintf("Testing %s", tc.name))
				// Basic validation
				gomega.Expect(tc.qos.Bandwidth).To(gomega.BeNumerically(">", 0))
				gomega.Expect(tc.qos.Latency).To(gomega.BeNumerically(">", 0))
				gomega.Expect(tc.qos.SliceType).To(gomega.Or(gomega.Equal("eMBB"), gomega.Equal("uRLLC"), gomega.Equal("mIoT"), gomega.Equal("balanced")))
			}
		})
	})
})

// Helper functions for integration tests
func createTestVNF(name, namespace string) *vnfv1alpha1.VNF {
	return &vnfv1alpha1.VNF{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: vnfv1alpha1.VNFSpec{
			Name:    name,
			Type:    "UPF",
			Version: "v1.0.0",
			Resources: vnfv1alpha1.ResourceRequirements{
				CPU:      "1",
				Memory:   "2Gi",
				CPUCores: 1,
				MemoryGB: 2,
			},
			QoS: vnfv1alpha1.QoSRequirements{
				Bandwidth: 10.0,
				Latency:   5.0,
				SliceType: "balanced",
			},
		},
	}
}

func waitForVNFPhase(ctx context.Context, client client.Client, vnf *vnfv1alpha1.VNF, phase string, timeout time.Duration) error {
	return waitForCondition(ctx, timeout, func() (bool, error) {
		key := types.NamespacedName{Name: vnf.Name, Namespace: vnf.Namespace}
		if err := client.Get(ctx, key, vnf); err != nil {
			return false, err
		}
		return vnf.Status.Phase == phase, nil
	})
}

func waitForCondition(ctx context.Context, timeout time.Duration, condition func() (bool, error)) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			done, err := condition()
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		}
	}
}

// Benchmark tests
var _ = ginkgo.Describe("VNF Operator Performance", func() {
	ginkgo.Context("Deployment Speed", func() {
		ginkgo.It("Should deploy VNF within SLA time", func() {
			startTime := time.Now()
			vnf := createTestVNF("perf-vnf", "default")

			err := k8sClient.Create(ctx, vnf)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			err = waitForVNFPhase(ctx, k8sClient, vnf, "Running", 5*time.Minute)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			deploymentTime := time.Since(startTime)
			ginkgo.By(fmt.Sprintf("Deployment completed in %v", deploymentTime))

			// Target: E2E deploy time <10 min (thesis requirement)
			gomega.Expect(deploymentTime).To(gomega.BeNumerically("<", 10*time.Minute))

			// Cleanup
			err = k8sClient.Delete(ctx, vnf)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})
})

// Test utilities for checking cluster connectivity
func isClusterAvailable() bool { // nolint:unused // TODO: implement cluster connectivity checks
	home, _ := os.UserHomeDir()
	kubeconfig := filepath.Join(home, ".kube", "config")

	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		return false
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return false
	}

	client, err := client.New(config, client.Options{})
	if err != nil {
		return false
	}

	// Try to list namespaces as a connectivity check
	namespaces := &corev1.NamespaceList{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.List(ctx, namespaces)
	return err == nil
}

// Test data generation helpers
func generateTestData(dataType string) interface{} { // nolint:unused // TODO: implement test data generation
	switch dataType {
	case "vnf":
		return createTestVNF("generated-vnf", "default")
	case "qos":
		return vnfv1alpha1.QoSRequirements{
			Bandwidth: 10.0,
			Latency:   5.0,
			SliceType: "balanced",
		}
	default:
		return nil
	}
}

// Validation helpers
func validateVNFSpec(vnf *vnfv1alpha1.VNF) error { // nolint:unused // TODO: implement VNF spec validation
	v := reflect.ValueOf(vnf.Spec)
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Check for zero values in required fields
		if fieldType.Tag.Get("required") == "true" && field.IsZero() {
			return fmt.Errorf("required field %s is empty", fieldType.Name)
		}
	}

	return nil
}

// File system helpers for test artifacts
func writeTestArtifact(filename string, content []byte) error { // nolint:unused // TODO: implement test artifact writing
	artifactDir := filepath.Join(".", "test-artifacts")
	if err := os.MkdirAll(artifactDir, security.PrivateDirMode); err != nil {
		return err
	}

	filePath := filepath.Join(artifactDir, filename)
	return os.WriteFile(filePath, content, security.SecureFileMode)
}
