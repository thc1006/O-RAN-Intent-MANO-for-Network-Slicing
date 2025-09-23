package integration

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	vnfv1alpha1 "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/api/v1alpha1"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/controllers"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tests/framework/testutils"
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
	framework *testutils.TestFramework
)

func TestVNFOperatorIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VNF Operator Integration Test Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.Background())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "adapters", "vnf-operator", "config", "crd", "bases"),
		},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = vnfv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Setup test framework
	testConfig := &testutils.TestConfig{
		KubeConfig:    cfg,
		KubeClient:    k8sClient,
		TestEnv:       testEnv,
		Context:       ctx,
		CancelFunc:    cancel,
		LogLevel:      "debug",
		ParallelNodes: 4,
	}

	framework = testutils.NewTestFramework(testConfig)
	err = framework.SetupTestEnvironment()
	Expect(err).NotTo(HaveOccurred())

	// Start the manager
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&controllers.VNFReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())

	if framework != nil {
		err = framework.TeardownTestEnvironment()
		Expect(err).NotTo(HaveOccurred())
	}
})

var _ = Describe("VNF Operator Integration", func() {
	var (
		namespace     *corev1.Namespace
		namespaceName string
	)

	BeforeEach(func() {
		// Create a unique namespace for each test
		namespaceName = fmt.Sprintf("test-vnf-%d", time.Now().UnixNano())
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceName,
			},
		}
		Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
	})

	AfterEach(func() {
		// Clean up namespace
		if namespace != nil {
			Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())
		}
	})

	Context("VNF Custom Resource Lifecycle", func() {
		It("should create and manage a basic VNF", func() {
			vnfName := "test-upf"

			// Create VNF resource
			vnf := &vnfv1alpha1.VNF{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vnfName,
					Namespace: namespaceName,
				},
				Spec: vnfv1alpha1.VNFSpec{
					Type:    "UPF",
					Version: "v1.0.0",
					Image:   "oran/upf:v1.0.0",
					Resources: vnfv1alpha1.ResourceRequirements{
						CPU:     "500m",
						Memory:  "1Gi",
						Storage: "10Gi",
					},
					NetworkSlice: vnfv1alpha1.NetworkSliceSpec{
						SliceID:   "emergency-slice-001",
						SliceType: "urllc",
						QoSProfile: vnfv1alpha1.QoSProfile{
							Priority:            "high",
							MaxLatencyMs:        1,
							MinThroughputMbps:   100,
							ReliabilityPercent:  99.99,
							PacketLossRateMax:   0.001,
						},
					},
					PlacementPolicy: vnfv1alpha1.PlacementPolicy{
						PreferredSites: []string{"edge-site-01"},
						Constraints: map[string]string{
							"region": "us-east-1",
							"type":   "edge",
						},
					},
				},
			}

			By("Creating the VNF resource")
			Expect(k8sClient.Create(ctx, vnf)).To(Succeed())

			By("Checking the VNF is created")
			createdVNF := &vnfv1alpha1.VNF{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      vnfName,
					Namespace: namespaceName,
				}, createdVNF)
			}, time.Minute, time.Second).Should(Succeed())

			By("Verifying VNF specifications")
			Expect(createdVNF.Spec.Type).To(Equal("UPF"))
			Expect(createdVNF.Spec.NetworkSlice.SliceType).To(Equal("urllc"))
			Expect(createdVNF.Spec.NetworkSlice.QoSProfile.Priority).To(Equal("high"))

			By("Waiting for VNF to be reconciled and deployment created")
			deployment := &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      vnfName,
					Namespace: namespaceName,
				}, deployment)
			}, time.Minute*2, time.Second*5).Should(Succeed())

			By("Verifying deployment specifications")
			Expect(deployment.Spec.Template.Spec.Containers).To(HaveLen(1))
			container := deployment.Spec.Template.Spec.Containers[0]
			Expect(container.Image).To(Equal("oran/upf:v1.0.0"))
			Expect(container.Resources.Requests.Cpu().String()).To(Equal("500m"))
			Expect(container.Resources.Requests.Memory().String()).To(Equal("1Gi"))

			By("Checking VNF status updates")
			Eventually(func() vnfv1alpha1.VNFPhase {
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name: vnfName, Namespace: namespaceName,
				}, createdVNF); err != nil {
					return ""
				}
				return createdVNF.Status.Phase
			}, time.Minute*3, time.Second*5).Should(Equal(vnfv1alpha1.VNFPhaseRunning))

			By("Verifying associated services are created")
			service := &corev1.Service{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      vnfName + "-service",
					Namespace: namespaceName,
				}, service)
			}, time.Minute, time.Second*5).Should(Succeed())
		})

		It("should handle VNF updates", func() {
			vnfName := "test-amf-update"

			// Create initial VNF
			vnf := &vnfv1alpha1.VNF{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vnfName,
					Namespace: namespaceName,
				},
				Spec: vnfv1alpha1.VNFSpec{
					Type:    "AMF",
					Version: "v1.0.0",
					Image:   "oran/amf:v1.0.0",
					Resources: vnfv1alpha1.ResourceRequirements{
						CPU:     "300m",
						Memory:  "512Mi",
						Storage: "5Gi",
					},
					NetworkSlice: vnfv1alpha1.NetworkSliceSpec{
						SliceID:   "mobile-slice-001",
						SliceType: "embb",
						QoSProfile: vnfv1alpha1.QoSProfile{
							Priority:            "medium",
							MaxLatencyMs:        10,
							MinThroughputMbps:   50,
							ReliabilityPercent:  99.9,
						},
					},
				},
			}

			By("Creating the initial VNF")
			Expect(k8sClient.Create(ctx, vnf)).To(Succeed())

			By("Waiting for initial deployment")
			deployment := &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name: vnfName, Namespace: namespaceName,
				}, deployment)
			}, time.Minute, time.Second*5).Should(Succeed())

			originalImage := deployment.Spec.Template.Spec.Containers[0].Image
			Expect(originalImage).To(Equal("oran/amf:v1.0.0"))

			By("Updating the VNF image version")
			updatedVNF := &vnfv1alpha1.VNF{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: vnfName, Namespace: namespaceName,
			}, updatedVNF)).To(Succeed())

			updatedVNF.Spec.Version = "v1.1.0"
			updatedVNF.Spec.Image = "oran/amf:v1.1.0"
			Expect(k8sClient.Update(ctx, updatedVNF)).To(Succeed())

			By("Verifying deployment is updated")
			Eventually(func() string {
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name: vnfName, Namespace: namespaceName,
				}, deployment); err != nil {
					return ""
				}
				return deployment.Spec.Template.Spec.Containers[0].Image
			}, time.Minute*2, time.Second*5).Should(Equal("oran/amf:v1.1.0"))

			By("Checking VNF status reflects the update")
			Eventually(func() string {
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name: vnfName, Namespace: namespaceName,
				}, updatedVNF); err != nil {
					return ""
				}
				return updatedVNF.Status.Version
			}, time.Minute, time.Second*5).Should(Equal("v1.1.0"))
		})

		It("should handle VNF deletion gracefully", func() {
			vnfName := "test-smf-delete"

			// Create VNF
			vnf := &vnfv1alpha1.VNF{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vnfName,
					Namespace: namespaceName,
				},
				Spec: vnfv1alpha1.VNFSpec{
					Type:    "SMF",
					Version: "v1.0.0",
					Image:   "oran/smf:v1.0.0",
					Resources: vnfv1alpha1.ResourceRequirements{
						CPU:     "200m",
						Memory:  "256Mi",
						Storage: "2Gi",
					},
				},
			}

			By("Creating the VNF")
			Expect(k8sClient.Create(ctx, vnf)).To(Succeed())

			By("Waiting for resources to be created")
			deployment := &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name: vnfName, Namespace: namespaceName,
				}, deployment)
			}, time.Minute, time.Second*5).Should(Succeed())

			service := &corev1.Service{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name: vnfName + "-service", Namespace: namespaceName,
				}, service)
			}, time.Minute, time.Second*5).Should(Succeed())

			By("Deleting the VNF")
			Expect(k8sClient.Delete(ctx, vnf)).To(Succeed())

			By("Verifying deployment is deleted")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name: vnfName, Namespace: namespaceName,
				}, deployment)
				return err != nil
			}, time.Minute, time.Second*5).Should(BeTrue())

			By("Verifying service is deleted")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name: vnfName + "-service", Namespace: namespaceName,
				}, service)
				return err != nil
			}, time.Minute, time.Second*5).Should(BeTrue())

			By("Verifying VNF resource is deleted")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name: vnfName, Namespace: namespaceName,
				}, vnf)
				return err != nil
			}, time.Minute, time.Second*5).Should(BeTrue())
		})
	})

	Context("Network Slice Integration", func() {
		It("should configure VNFs for different slice types", func() {
			testCases := []struct {
				name      string
				sliceType string
				qos       vnfv1alpha1.QoSProfile
			}{
				{
					name:      "urllc-vnf",
					sliceType: "urllc",
					qos: vnfv1alpha1.QoSProfile{
						Priority:            "critical",
						MaxLatencyMs:        1,
						MinThroughputMbps:   100,
						ReliabilityPercent:  99.999,
						PacketLossRateMax:   0.0001,
					},
				},
				{
					name:      "embb-vnf",
					sliceType: "embb",
					qos: vnfv1alpha1.QoSProfile{
						Priority:            "high",
						MaxLatencyMs:        15.7, // Thesis target
						MinThroughputMbps:   2.77, // Thesis target
						ReliabilityPercent:  99.9,
						PacketLossRateMax:   0.001,
					},
				},
				{
					name:      "mmtc-vnf",
					sliceType: "mmtc",
					qos: vnfv1alpha1.QoSProfile{
						Priority:            "low",
						MaxLatencyMs:        16.1, // Thesis target
						MinThroughputMbps:   0.93, // Thesis target
						ReliabilityPercent:  99.0,
						PacketLossRateMax:   0.01,
					},
				},
			}

			for _, tc := range testCases {
				By(fmt.Sprintf("Creating %s VNF", tc.name))

				vnf := &vnfv1alpha1.VNF{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tc.name,
						Namespace: namespaceName,
					},
					Spec: vnfv1alpha1.VNFSpec{
						Type:    "UPF",
						Version: "v1.0.0",
						Image:   "oran/upf:v1.0.0",
						Resources: vnfv1alpha1.ResourceRequirements{
							CPU:     "500m",
							Memory:  "1Gi",
							Storage: "5Gi",
						},
						NetworkSlice: vnfv1alpha1.NetworkSliceSpec{
							SliceID:    fmt.Sprintf("%s-slice-001", tc.sliceType),
							SliceType:  tc.sliceType,
							QoSProfile: tc.qos,
						},
					},
				}

				Expect(k8sClient.Create(ctx, vnf)).To(Succeed())

				By(fmt.Sprintf("Verifying %s deployment configuration", tc.name))
				deployment := &appsv1.Deployment{}
				Eventually(func() error {
					return k8sClient.Get(ctx, types.NamespacedName{
						Name: tc.name, Namespace: namespaceName,
					}, deployment)
				}, time.Minute, time.Second*5).Should(Succeed())

				// Verify slice-specific annotations
				Expect(deployment.Annotations).To(HaveKey("oran.slice.type"))
				Expect(deployment.Annotations["oran.slice.type"]).To(Equal(tc.sliceType))
				Expect(deployment.Annotations).To(HaveKey("oran.slice.priority"))
				Expect(deployment.Annotations["oran.slice.priority"]).To(Equal(tc.qos.Priority))

				// Verify QoS-related labels
				Expect(deployment.Labels).To(HaveKey("oran.qos.latency"))
				Expect(deployment.Labels).To(HaveKey("oran.qos.throughput"))
				Expect(deployment.Labels).To(HaveKey("oran.qos.reliability"))
			}
		})

		It("should handle slice-specific resource allocation", func() {
			vnfName := "resource-test-vnf"

			// Create VNF with specific resource requirements
			vnf := &vnfv1alpha1.VNF{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vnfName,
					Namespace: namespaceName,
				},
				Spec: vnfv1alpha1.VNFSpec{
					Type:    "UPF",
					Version: "v1.0.0",
					Image:   "oran/upf:v1.0.0",
					Resources: vnfv1alpha1.ResourceRequirements{
						CPU:     "2000m",
						Memory:  "4Gi",
						Storage: "20Gi",
					},
					NetworkSlice: vnfv1alpha1.NetworkSliceSpec{
						SliceID:   "high-perf-slice-001",
						SliceType: "urllc",
						QoSProfile: vnfv1alpha1.QoSProfile{
							Priority:            "critical",
							MaxLatencyMs:        1,
							MinThroughputMbps:   500,
							ReliabilityPercent:  99.999,
						},
					},
					PlacementPolicy: vnfv1alpha1.PlacementPolicy{
						NodeSelector: map[string]string{
							"node-type": "high-performance",
						},
						Tolerations: []corev1.Toleration{
							{
								Key:      "dedicated",
								Operator: corev1.TolerationOpEqual,
								Value:    "vnf",
								Effect:   corev1.TaintEffectNoSchedule,
							},
						},
					},
				},
			}

			By("Creating high-performance VNF")
			Expect(k8sClient.Create(ctx, vnf)).To(Succeed())

			By("Verifying resource specifications in deployment")
			deployment := &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name: vnfName, Namespace: namespaceName,
				}, deployment)
			}, time.Minute, time.Second*5).Should(Succeed())

			container := deployment.Spec.Template.Spec.Containers[0]
			Expect(container.Resources.Requests.Cpu().String()).To(Equal("2"))
			Expect(container.Resources.Requests.Memory().String()).To(Equal("4Gi"))

			// Verify placement constraints
			Expect(deployment.Spec.Template.Spec.NodeSelector).To(HaveKeyWithValue("node-type", "high-performance"))
			Expect(deployment.Spec.Template.Spec.Tolerations).To(HaveLen(1))
			Expect(deployment.Spec.Template.Spec.Tolerations[0].Key).To(Equal("dedicated"))
		})
	})

	Context("Multi-VNF Orchestration", func() {
		It("should deploy and manage multiple VNFs in a slice", func() {
			sliceID := "multi-vnf-slice-001"

			vnfSpecs := []struct {
				name     string
				vnfType  string
				image    string
				priority string
			}{
				{"slice-amf", "AMF", "oran/amf:v1.0.0", "high"},
				{"slice-smf", "SMF", "oran/smf:v1.0.0", "high"},
				{"slice-upf", "UPF", "oran/upf:v1.0.0", "critical"},
				{"slice-pcf", "PCF", "oran/pcf:v1.0.0", "medium"},
			}

			vnfs := make([]*vnfv1alpha1.VNF, len(vnfSpecs))

			By("Creating multiple VNFs for the same slice")
			for i, spec := range vnfSpecs {
				vnf := &vnfv1alpha1.VNF{
					ObjectMeta: metav1.ObjectMeta{
						Name:      spec.name,
						Namespace: namespaceName,
						Labels: map[string]string{
							"slice-id": sliceID,
							"vnf-type": spec.vnfType,
						},
					},
					Spec: vnfv1alpha1.VNFSpec{
						Type:    spec.vnfType,
						Version: "v1.0.0",
						Image:   spec.image,
						Resources: vnfv1alpha1.ResourceRequirements{
							CPU:     "500m",
							Memory:  "1Gi",
							Storage: "5Gi",
						},
						NetworkSlice: vnfv1alpha1.NetworkSliceSpec{
							SliceID:   sliceID,
							SliceType: "embb",
							QoSProfile: vnfv1alpha1.QoSProfile{
								Priority:            spec.priority,
								MaxLatencyMs:        10,
								MinThroughputMbps:   100,
								ReliabilityPercent:  99.9,
							},
						},
					},
				}

				vnfs[i] = vnf
				Expect(k8sClient.Create(ctx, vnf)).To(Succeed())
			}

			By("Verifying all VNFs are deployed")
			for _, vnf := range vnfs {
				deployment := &appsv1.Deployment{}
				Eventually(func() error {
					return k8sClient.Get(ctx, types.NamespacedName{
						Name: vnf.Name, Namespace: namespaceName,
					}, deployment)
				}, time.Minute*2, time.Second*5).Should(Succeed())

				// Verify slice association
				Expect(deployment.Labels).To(HaveKeyWithValue("slice-id", sliceID))
			}

			By("Verifying VNF interdependencies are configured")
			// Check that VNFs have proper service discovery configurations
			for _, vnf := range vnfs {
				service := &corev1.Service{}
				Eventually(func() error {
					return k8sClient.Get(ctx, types.NamespacedName{
						Name: vnf.Name + "-service", Namespace: namespaceName,
					}, service)
				}, time.Minute, time.Second*5).Should(Succeed())

				// Verify service has slice labels
				Expect(service.Labels).To(HaveKeyWithValue("slice-id", sliceID))
			}

			By("Testing slice-wide updates")
			// Update all VNFs in the slice to a new version
			for _, vnf := range vnfs {
				updatedVNF := &vnfv1alpha1.VNF{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: vnf.Name, Namespace: namespaceName,
				}, updatedVNF)).To(Succeed())

				updatedVNF.Spec.Version = "v1.1.0"
				updatedVNF.Spec.Image = fmt.Sprintf("oran/%s:v1.1.0",
					map[string]string{"AMF": "amf", "SMF": "smf", "UPF": "upf", "PCF": "pcf"}[vnf.Spec.Type])

				Expect(k8sClient.Update(ctx, updatedVNF)).To(Succeed())
			}

			By("Verifying coordinated updates")
			for _, vnf := range vnfs {
				Eventually(func() string {
					updatedVNF := &vnfv1alpha1.VNF{}
					if err := k8sClient.Get(ctx, types.NamespacedName{
						Name: vnf.Name, Namespace: namespaceName,
					}, updatedVNF); err != nil {
						return ""
					}
					return updatedVNF.Status.Version
				}, time.Minute*2, time.Second*5).Should(Equal("v1.1.0"))
			}
		})
	})

	Context("Error Handling and Recovery", func() {
		It("should handle invalid VNF configurations", func() {
			vnfName := "invalid-vnf"

			// Create VNF with invalid configuration
			vnf := &vnfv1alpha1.VNF{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vnfName,
					Namespace: namespaceName,
				},
				Spec: vnfv1alpha1.VNFSpec{
					Type:    "INVALID_TYPE",
					Version: "invalid-version",
					Image:   "", // Empty image
					Resources: vnfv1alpha1.ResourceRequirements{
						CPU:     "invalid-cpu",
						Memory:  "invalid-memory",
						Storage: "invalid-storage",
					},
				},
			}

			By("Creating invalid VNF")
			Expect(k8sClient.Create(ctx, vnf)).To(Succeed())

			By("Verifying VNF status shows error")
			createdVNF := &vnfv1alpha1.VNF{}
			Eventually(func() vnfv1alpha1.VNFPhase {
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name: vnfName, Namespace: namespaceName,
				}, createdVNF); err != nil {
					return ""
				}
				return createdVNF.Status.Phase
			}, time.Minute, time.Second*5).Should(Equal(vnfv1alpha1.VNFPhaseFailed))

			By("Verifying error message is set")
			Expect(createdVNF.Status.Message).ToNot(BeEmpty())
			Expect(createdVNF.Status.Message).To(ContainSubstring("invalid"))
		})

		It("should recover from temporary failures", func() {
			vnfName := "recovery-test-vnf"

			// Create VNF with initially invalid image
			vnf := &vnfv1alpha1.VNF{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vnfName,
					Namespace: namespaceName,
				},
				Spec: vnfv1alpha1.VNFSpec{
					Type:    "UPF",
					Version: "v1.0.0",
					Image:   "oran/nonexistent:v1.0.0", // Nonexistent image
					Resources: vnfv1alpha1.ResourceRequirements{
						CPU:     "500m",
						Memory:  "1Gi",
						Storage: "5Gi",
					},
				},
			}

			By("Creating VNF with invalid image")
			Expect(k8sClient.Create(ctx, vnf)).To(Succeed())

			By("Verifying deployment is created but fails")
			deployment := &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name: vnfName, Namespace: namespaceName,
				}, deployment)
			}, time.Minute, time.Second*5).Should(Succeed())

			By("Fixing the VNF configuration")
			updatedVNF := &vnfv1alpha1.VNF{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: vnfName, Namespace: namespaceName,
			}, updatedVNF)).To(Succeed())

			updatedVNF.Spec.Image = "oran/upf:v1.0.0" // Fix the image
			Expect(k8sClient.Update(ctx, updatedVNF)).To(Succeed())

			By("Verifying VNF recovers to running state")
			Eventually(func() vnfv1alpha1.VNFPhase {
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name: vnfName, Namespace: namespaceName,
				}, updatedVNF); err != nil {
					return ""
				}
				return updatedVNF.Status.Phase
			}, time.Minute*3, time.Second*5).Should(Equal(vnfv1alpha1.VNFPhaseRunning))
		})
	})

	Context("Performance and Scalability", func() {
		It("should handle rapid VNF creation and deletion", func() {
			const numVNFs = 10
			vnfNames := make([]string, numVNFs)
			vnfs := make([]*vnfv1alpha1.VNF, numVNFs)

			By(fmt.Sprintf("Creating %d VNFs rapidly", numVNFs))
			startTime := time.Now()

			for i := 0; i < numVNFs; i++ {
				vnfName := fmt.Sprintf("perf-test-vnf-%d", i)
				vnfNames[i] = vnfName

				vnf := &vnfv1alpha1.VNF{
					ObjectMeta: metav1.ObjectMeta{
						Name:      vnfName,
						Namespace: namespaceName,
					},
					Spec: vnfv1alpha1.VNFSpec{
						Type:    "UPF",
						Version: "v1.0.0",
						Image:   "oran/upf:v1.0.0",
						Resources: vnfv1alpha1.ResourceRequirements{
							CPU:     "100m",
							Memory:  "128Mi",
							Storage: "1Gi",
						},
					},
				}

				vnfs[i] = vnf
				Expect(k8sClient.Create(ctx, vnf)).To(Succeed())
			}

			creationTime := time.Since(startTime)

			By("Verifying all VNFs are reconciled")
			for _, vnfName := range vnfNames {
				deployment := &appsv1.Deployment{}
				Eventually(func() error {
					return k8sClient.Get(ctx, types.NamespacedName{
						Name: vnfName, Namespace: namespaceName,
					}, deployment)
				}, time.Minute*2, time.Second*2).Should(Succeed())
			}

			reconciliationTime := time.Since(startTime)

			By(fmt.Sprintf("Deleting %d VNFs", numVNFs))
			deletionStart := time.Now()

			for _, vnf := range vnfs {
				Expect(k8sClient.Delete(ctx, vnf)).To(Succeed())
			}

			By("Verifying all VNFs are deleted")
			for _, vnfName := range vnfNames {
				vnf := &vnfv1alpha1.VNF{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: vnfName, Namespace: namespaceName,
					}, vnf)
					return err != nil
				}, time.Minute*2, time.Second*2).Should(BeTrue())
			}

			deletionTime := time.Since(deletionStart)

			By("Verifying performance metrics")
			avgCreationTime := creationTime / numVNFs
			avgReconciliationTime := reconciliationTime / numVNFs
			avgDeletionTime := deletionTime / numVNFs

			framework.Reporter.ReportTestResult(testutils.TestResult{
				Name:     "VNF Creation Performance",
				Category: "performance",
				Status:   "passed",
				Duration: creationTime,
				Performance: &testutils.PerformanceMetrics{
					DeploymentTime: reconciliationTime,
				},
			})

			// Performance assertions based on thesis requirements
			Expect(avgCreationTime).To(BeNumerically("<", time.Second*5),
				"Average VNF creation should be < 5 seconds")
			Expect(avgReconciliationTime).To(BeNumerically("<", time.Second*30),
				"Average VNF reconciliation should be < 30 seconds")
			Expect(avgDeletionTime).To(BeNumerically("<", time.Second*10),
				"Average VNF deletion should be < 10 seconds")

			By(fmt.Sprintf("Performance Results: Creation: %v, Reconciliation: %v, Deletion: %v",
				avgCreationTime, avgReconciliationTime, avgDeletionTime))
		})
	})

	Context("GitOps Integration", func() {
		It("should integrate with Nephio package generation", func() {
			vnfName := "gitops-test-vnf"

			vnf := &vnfv1alpha1.VNF{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vnfName,
					Namespace: namespaceName,
					Annotations: map[string]string{
						"oran.gitops.enabled":    "true",
						"oran.gitops.repository": "https://github.com/test/packages",
						"oran.gitops.branch":     "main",
					},
				},
				Spec: vnfv1alpha1.VNFSpec{
					Type:    "UPF",
					Version: "v1.0.0",
					Image:   "oran/upf:v1.0.0",
					Resources: vnfv1alpha1.ResourceRequirements{
						CPU:     "500m",
						Memory:  "1Gi",
						Storage: "5Gi",
					},
					NetworkSlice: vnfv1alpha1.NetworkSliceSpec{
						SliceID:   "gitops-slice-001",
						SliceType: "embb",
					},
				},
			}

			By("Creating VNF with GitOps annotations")
			Expect(k8sClient.Create(ctx, vnf)).To(Succeed())

			By("Verifying VNF is created with GitOps metadata")
			createdVNF := &vnfv1alpha1.VNF{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name: vnfName, Namespace: namespaceName,
				}, createdVNF)
			}, time.Minute, time.Second*5).Should(Succeed())

			By("Checking GitOps status annotations")
			Eventually(func() map[string]string {
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name: vnfName, Namespace: namespaceName,
				}, createdVNF); err != nil {
					return nil
				}
				return createdVNF.Status.GitOpsStatus
			}, time.Minute, time.Second*5).ShouldNot(BeEmpty())
		})
	})
})