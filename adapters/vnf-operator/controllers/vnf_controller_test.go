package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	manov1alpha1 "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/api/v1alpha1"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/pkg/dms"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/pkg/gitops"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/pkg/translator"
)

var _ = Describe("VNF Controller", func() {
	Context("When reconciling a VNF", func() {
		const (
			vnfName      = "test-vnf"
			vnfNamespace = "default"
			timeout      = time.Second * 10
			interval     = time.Millisecond * 250
		)

		var vnfReconciler *VNFReconciler

		BeforeEach(func() {
			// Initialize mock clients for the reconciler
			vnfReconciler = &VNFReconciler{
				Client:          k8sClient,
				Scheme:          k8sClient.Scheme(),
				PorchTranslator: translator.NewPorchTranslator(),
				DMSClient:       dms.NewMockDMSClient(),
				GitOpsClient:    gitops.NewMockGitOpsClient(),
			}
		})

		AfterEach(func() {
			// Cleanup VNF resources
			vnf := &manov1alpha1.VNF{}
			err := k8sClient.Get(context.Background(),
				types.NamespacedName{Name: vnfName, Namespace: vnfNamespace}, vnf)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), vnf)
			}
		})

		It("Should create VNF with RAN type successfully", func() {
			ctx := context.Background()

			// Create a RAN VNF
			vnf := &manov1alpha1.VNF{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "vnf.oran.io/v1alpha1",
					Kind:       "VNF",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      vnfName,
					Namespace: vnfNamespace,
				},
				Spec: manov1alpha1.VNFSpec{
					Name:    vnfName,
					Type:    manov1alpha1.VNFTypeRAN,
					Version: "1.0.0",
					Placement: manov1alpha1.PlacementRequirements{
						CloudType: "edge",
					},
					Resources: manov1alpha1.ResourceRequirements{
						CPUCores: 2,
						MemoryGB: 4,
						CPU:      "2", // Legacy field for compatibility
						Memory:   "4Gi",
					},
					QoS: manov1alpha1.QoSRequirements{
						Bandwidth: 4.5,
						Latency:   1.5,
					},
					TargetClusters: []string{"edge01", "edge02"},
					Image: manov1alpha1.ImageSpec{
						Repository: "oran/ran",
						Tag:        "1.0.0",
					},
				},
			}

			Expect(k8sClient.Create(ctx, vnf)).Should(Succeed())

			// Wait for VNF to be created
			createdVNF := &manov1alpha1.VNF{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx,
					types.NamespacedName{Name: vnfName, Namespace: vnfNamespace},
					createdVNF)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Verify VNF spec
			Expect(createdVNF.Spec.Type).Should(Equal(manov1alpha1.VNFTypeRAN))
			Expect(createdVNF.Spec.QoS.Bandwidth).Should(Equal(4.5))
			Expect(createdVNF.Spec.QoS.Latency).Should(Equal(1.5))
		})

		It("Should create VNF with CN type successfully", func() {
			ctx := context.Background()

			// Create a CN VNF
			vnf := &manov1alpha1.VNF{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "vnf.oran.io/v1alpha1",
					Kind:       "VNF",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      vnfName + "-cn",
					Namespace: vnfNamespace,
				},
				Spec: manov1alpha1.VNFSpec{
					Name:    vnfName + "-cn",
					Type:    manov1alpha1.VNFTypeCN,
					Version: "2.0.0",
					Placement: manov1alpha1.PlacementRequirements{
						CloudType: "regional",
					},
					Resources: manov1alpha1.ResourceRequirements{
						CPUCores: 4,
						MemoryGB: 8,
						CPU:      "4",
						Memory:   "8Gi",
					},
					QoS: manov1alpha1.QoSRequirements{
						Bandwidth: 3.0,
						Latency:   9.0,
					},
					TargetClusters: []string{"regional"},
					Image: manov1alpha1.ImageSpec{
						Repository: "oran/core-network",
						Tag:        "2.0.0",
					},
				},
			}

			Expect(k8sClient.Create(ctx, vnf)).Should(Succeed())

			// Verify creation
			createdVNF := &manov1alpha1.VNF{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx,
					types.NamespacedName{Name: vnfName + "-cn", Namespace: vnfNamespace},
					createdVNF)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdVNF.Spec.Type).Should(Equal(manov1alpha1.VNFTypeCN))
			Expect(createdVNF.Spec.Placement.CloudType).Should(Equal("regional"))
		})

		It("Should reconcile VNF through pending to creating state", func() {
			ctx := context.Background()

			// Create a valid VNF
			vnf := &manov1alpha1.VNF{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "vnf.oran.io/v1alpha1",
					Kind:       "VNF",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      vnfName + "-reconcile",
					Namespace: vnfNamespace,
				},
				Spec: manov1alpha1.VNFSpec{
					Name:    vnfName + "-reconcile",
					Type:    manov1alpha1.VNFTypeUPF,
					Version: "1.0.0",
					Placement: manov1alpha1.PlacementRequirements{
						CloudType: "edge",
					},
					Resources: manov1alpha1.ResourceRequirements{
						CPUCores: 1,
						MemoryGB: 2,
					},
					QoS: manov1alpha1.QoSRequirements{
						Bandwidth: 2.0,
						Latency:   5.0,
					},
					TargetClusters: []string{"edge01"},
					Image: manov1alpha1.ImageSpec{
						Repository: "oran/upf",
						Tag:        "1.0.0",
					},
				},
			}

			Expect(k8sClient.Create(ctx, vnf)).Should(Succeed())

			// Trigger reconciliation
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      vnfName + "-reconcile",
					Namespace: vnfNamespace,
				},
			}

			_, err := vnfReconciler.Reconcile(ctx, req)
			Expect(err).ShouldNot(HaveOccurred())

			// Check that VNF status is updated
			reconciledVNF := &manov1alpha1.VNF{}
			Eventually(func() string {
				_ = k8sClient.Get(ctx,
					types.NamespacedName{Name: vnfName + "-reconcile", Namespace: vnfNamespace},
					reconciledVNF)
				return reconciledVNF.Status.Phase
			}, timeout, interval).Should(Not(BeEmpty()))
		})

		It("Should validate QoS parameters and fail with invalid values", func() {
			ctx := context.Background()

			// Create VNF with invalid QoS parameters
			vnf := &manov1alpha1.VNF{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "vnf.oran.io/v1alpha1",
					Kind:       "VNF",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      vnfName + "-invalid",
					Namespace: vnfNamespace,
				},
				Spec: manov1alpha1.VNFSpec{
					Name:    vnfName + "-invalid",
					Type:    manov1alpha1.VNFTypeRAN,
					Version: "1.0.0",
					Placement: manov1alpha1.PlacementRequirements{
						CloudType: "edge",
					},
					Resources: manov1alpha1.ResourceRequirements{
						CPUCores: 1,
						MemoryGB: 1,
					},
					QoS: manov1alpha1.QoSRequirements{
						Bandwidth: 10.0, // Invalid: > 5
						Latency:   0.5,  // Invalid: < 1
					},
					Image: manov1alpha1.ImageSpec{
						Repository: "oran/ran",
						Tag:        "1.0.0",
					},
				},
			}

			Expect(k8sClient.Create(ctx, vnf)).Should(Succeed())

			// Test validation directly
			err := vnfReconciler.validateVNF(vnf)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("bandwidth must be between"))
		})

		It("Should handle VNF deletion with finalizer", func() {
			ctx := context.Background()

			// Create a VNF
			vnf := &manov1alpha1.VNF{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "vnf.oran.io/v1alpha1",
					Kind:       "VNF",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      vnfName + "-delete",
					Namespace: vnfNamespace,
				},
				Spec: manov1alpha1.VNFSpec{
					Name:    vnfName + "-delete",
					Type:    manov1alpha1.VNFTypeTN,
					Version: "1.0.0",
					Placement: manov1alpha1.PlacementRequirements{
						CloudType: "edge",
					},
					Resources: manov1alpha1.ResourceRequirements{
						CPUCores: 1,
						MemoryGB: 1,
					},
					QoS: manov1alpha1.QoSRequirements{
						Bandwidth: 1.0,
						Latency:   1.0,
					},
					Image: manov1alpha1.ImageSpec{
						Repository: "oran/tn",
						Tag:        "1.0.0",
					},
				},
			}

			Expect(k8sClient.Create(ctx, vnf)).Should(Succeed())

			// Wait for creation
			createdVNF := &manov1alpha1.VNF{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx,
					types.NamespacedName{Name: vnfName + "-delete", Namespace: vnfNamespace},
					createdVNF)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Delete the VNF
			Expect(k8sClient.Delete(ctx, createdVNF)).Should(Succeed())

			// Verify deletion
			Eventually(func() bool {
				err := k8sClient.Get(ctx,
					types.NamespacedName{Name: vnfName + "-delete", Namespace: vnfNamespace},
					&manov1alpha1.VNF{})
				return errors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})

		It("Should handle different VNF types", func() {
			ctx := context.Background()

			testCases := []struct {
				name    string
				vnfType manov1alpha1.VNFType
			}{
				{"AMF", manov1alpha1.VNFTypeAMF},
				{"SMF", manov1alpha1.VNFTypeSMF},
				{"UPF", manov1alpha1.VNFTypeUPF},
				{"gNB", manov1alpha1.VNFTypegNB},
			}

			for _, tc := range testCases {
				vnf := &manov1alpha1.VNF{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "vnf.oran.io/v1alpha1",
						Kind:       "VNF",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      vnfName + "-" + tc.name,
						Namespace: vnfNamespace,
					},
					Spec: manov1alpha1.VNFSpec{
						Name:    vnfName + "-" + tc.name,
						Type:    tc.vnfType,
						Version: "1.0.0",
						Placement: manov1alpha1.PlacementRequirements{
							CloudType: "edge",
						},
						Resources: manov1alpha1.ResourceRequirements{
							CPUCores: 2,
							MemoryGB: 4,
						},
						QoS: manov1alpha1.QoSRequirements{
							Bandwidth: 3.0,
							Latency:   5.0,
						},
						TargetClusters: []string{"edge01"},
						Image: manov1alpha1.ImageSpec{
							Repository: "oran/" + tc.name,
							Tag:        "1.0.0",
						},
					},
				}

				Expect(k8sClient.Create(ctx, vnf)).Should(Succeed())

				// Verify VNF type is set correctly
				createdVNF := &manov1alpha1.VNF{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx,
						types.NamespacedName{Name: vnfName + "-" + tc.name, Namespace: vnfNamespace},
						createdVNF)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				Expect(createdVNF.Spec.Type).Should(Equal(tc.vnfType))

				// Cleanup
				_ = k8sClient.Delete(ctx, createdVNF)
			}
		})

		It("Should test controller phases and transitions", func() {

			// Test pending phase validation
			vnf := &manov1alpha1.VNF{
				Spec: manov1alpha1.VNFSpec{
					QoS: manov1alpha1.QoSRequirements{
						Bandwidth: 3.0,
						Latency:   5.0,
					},
					Placement: manov1alpha1.PlacementRequirements{
						CloudType: "edge",
					},
				},
				Status: manov1alpha1.VNFStatus{
					Phase: "Pending",
				},
			}

			err := vnfReconciler.validateVNF(vnf)
			Expect(err).ShouldNot(HaveOccurred())

			// Test with invalid cloud type
			vnf.Spec.Placement.CloudType = "invalid"
			err = vnfReconciler.validateVNF(vnf)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("invalid cloud type"))
		})
	})
})