// Package integration provides integration tests for RAN-DMS service
package integration

import (
	"context"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tests/utils"
	randmsv1 "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/ran-dms/api/v1alpha1"
)

var _ = ginkgo.Describe("RAN-DMS Integration Tests", func() {
	var (
		testEnv *utils.TestEnvironment
		ctx     context.Context
	)

	ginkgo.BeforeEach(func() {
		ctx = context.Background()
		testEnv = utils.SetupTestEnvironment(ginkgo.GinkgoT(), scheme)
	})

	ginkgo.AfterEach(func() {
		testEnv.Cleanup(ginkgo.GinkgoT())
	})

	ginkgo.Context("RAN Resource Management", func() {
		ginkgo.It("should create and manage RAN resources", func() {
			// Create RAN resource
			ranResource := &randmsv1.RANResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ran-resource",
					Namespace: testEnv.Namespace,
				},
				Spec: randmsv1.RANResourceSpec{
					Type:     "CU",
					Location: "edge01",
					Capacity: randmsv1.ResourceCapacity{
						CPU:    "4000m",
						Memory: "8Gi",
						Storage: "100Gi",
					},
					RadioConfig: randmsv1.RadioConfiguration{
						Frequency: "3.5GHz",
						Bandwidth: "100MHz",
						TxPower:   "20dBm",
					},
				},
			}

			err := testEnv.Client.Create(ctx, ranResource)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Verify resource is created and ready
			utils.AssertEventuallyWithTimeout(ginkgo.GinkgoT(), func() bool {
				updatedResource := &randmsv1.RANResource{}
				err := testEnv.Client.Get(ctx, client.ObjectKeyFromObject(ranResource), updatedResource)
				return err == nil && updatedResource.Status.Phase == "Ready"
			}, 30*time.Second, "RAN resource should be ready")
		})

		ginkgo.It("should configure radio parameters", func() {
			// Create RAN slice with specific radio configuration
			ranSlice := &randmsv1.RANSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ran-slice",
					Namespace: testEnv.Namespace,
				},
				Spec: randmsv1.RANSliceSpec{
					SliceID:   "slice-001",
					SliceType: "embb",
					QoSProfile: randmsv1.QoSProfile{
						Priority:   1,
						Throughput: "5Gbps",
						Latency:    "10ms",
						Reliability: "99.999%",
					},
					ResourceAllocation: randmsv1.ResourceAllocation{
						PRBs:    100,
						Antennas: 4,
						Carriers: []randmsv1.CarrierConfig{
							{
								Frequency: "3.5GHz",
								Bandwidth: "20MHz",
							},
						},
					},
				},
			}

			err := testEnv.Client.Create(ctx, ranSlice)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Verify slice configuration
			utils.AssertEventuallyWithTimeout(ginkgo.GinkgoT(), func() bool {
				updatedSlice := &randmsv1.RANSlice{}
				err := testEnv.Client.Get(ctx, client.ObjectKeyFromObject(ranSlice), updatedSlice)
				return err == nil && updatedSlice.Status.ConfiguredPRBs > 0
			}, 45*time.Second, "RAN slice should be configured")
		})
	})

	ginkgo.Context("Network Function Lifecycle", func() {
		ginkgo.It("should deploy and manage gNodeB functions", func() {
			// Create gNodeB deployment
			gnodeb := &randmsv1.GNodeB{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-gnodeb",
					Namespace: testEnv.Namespace,
				},
				Spec: randmsv1.GNodeBSpec{
					NodeID: "gnb-001",
					PLMNs: []randmsv1.PLMN{
						{
							MCC: "001",
							MNC: "01",
						},
					},
					Cells: []randmsv1.CellConfig{
						{
							CellID:      1,
							PCI:         100,
							TAC:         "0001",
							Frequency:   "3.5GHz",
							Bandwidth:   "20MHz",
							TxPower:     "20dBm",
						},
					},
					AMFConnections: []randmsv1.AMFConnection{
						{
							AMF_IP:   "10.0.1.100",
							AMF_Port: 38412,
						},
					},
				},
			}

			err := testEnv.Client.Create(ctx, gnodeb)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Verify gNodeB is deployed and connected
			utils.AssertEventuallyWithTimeout(ginkgo.GinkgoT(), func() bool {
				updatedGNodeB := &randmsv1.GNodeB{}
				err := testEnv.Client.Get(ctx, client.ObjectKeyFromObject(gnodeb), updatedGNodeB)
				return err == nil &&
					   updatedGNodeB.Status.Phase == "Running" &&
					   updatedGNodeB.Status.ConnectedCells > 0
			}, 60*time.Second, "gNodeB should be running with connected cells")
		})
	})
})

// RANDMSIntegrationTestSuite provides comprehensive testify-based tests
type RANDMSIntegrationTestSuite struct {
	suite.Suite
	testEnv *utils.TestEnvironment
	ctx     context.Context
}

func TestRANDMSIntegrationSuite(t *testing.T) {
	utils.SkipIfNotIntegration(t)
	suite.Run(t, new(RANDMSIntegrationTestSuite))
}

func (suite *RANDMSIntegrationTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.testEnv = utils.SetupTestEnvironment(suite.T(), scheme)
}

func (suite *RANDMSIntegrationTestSuite) TearDownSuite() {
	suite.testEnv.Cleanup(suite.T())
}

func (suite *RANDMSIntegrationTestSuite) TestRANResourceAllocation() {
	t := suite.T()
	utils.LogTestProgress(t, "Testing RAN resource allocation")

	// Create RAN slice with specific resource requirements
	ranSlice := &randmsv1.RANSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allocation-test-slice",
			Namespace: suite.testEnv.Namespace,
		},
		Spec: randmsv1.RANSliceSpec{
			SliceID:   "slice-allocation-001",
			SliceType: "urllc",
			QoSProfile: randmsv1.QoSProfile{
				Priority:    1,
				Throughput:  "1Gbps",
				Latency:     "1ms",
				Reliability: "99.9999%",
			},
			ResourceAllocation: randmsv1.ResourceAllocation{
				PRBs:     50,
				Antennas: 2,
				Carriers: []randmsv1.CarrierConfig{
					{
						Frequency: "28GHz",
						Bandwidth: "400MHz",
					},
				},
			},
		},
	}

	err := suite.testEnv.Client.Create(suite.ctx, ranSlice)
	require.NoError(t, err)

	// Wait for resource allocation
	utils.AssertEventuallyWithTimeout(t, func() bool {
		updatedSlice := &randmsv1.RANSlice{}
		err := suite.testEnv.Client.Get(suite.ctx, client.ObjectKeyFromObject(ranSlice), updatedSlice)
		if err != nil {
			return false
		}

		return updatedSlice.Status.AllocatedResources != nil &&
			   updatedSlice.Status.AllocatedResources.PRBs >= ranSlice.Spec.ResourceAllocation.PRBs
	}, 30*time.Second, "RAN resources should be allocated")

	// Verify allocation details
	updatedSlice := &randmsv1.RANSlice{}
	err = suite.testEnv.Client.Get(suite.ctx, client.ObjectKeyFromObject(ranSlice), updatedSlice)
	require.NoError(t, err)
	assert.NotNil(t, updatedSlice.Status.AllocatedResources)
	assert.GreaterOrEqual(t, updatedSlice.Status.AllocatedResources.PRBs, ranSlice.Spec.ResourceAllocation.PRBs)

	utils.LogTestProgress(t, "RAN resource allocation test completed")
}

func (suite *RANDMSIntegrationTestSuite) TestMultiSliceIsolation() {
	t := suite.T()
	utils.LogTestProgress(t, "Testing multi-slice isolation")

	// Create multiple slices with different QoS requirements
	slices := []*randmsv1.RANSlice{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "embb-slice",
				Namespace: suite.testEnv.Namespace,
			},
			Spec: randmsv1.RANSliceSpec{
				SliceID:   "slice-embb-001",
				SliceType: "embb",
				QoSProfile: randmsv1.QoSProfile{
					Priority:   3,
					Throughput: "5Gbps",
					Latency:    "10ms",
				},
				ResourceAllocation: randmsv1.ResourceAllocation{
					PRBs: 80,
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "urllc-slice",
				Namespace: suite.testEnv.Namespace,
			},
			Spec: randmsv1.RANSliceSpec{
				SliceID:   "slice-urllc-001",
				SliceType: "urllc",
				QoSProfile: randmsv1.QoSProfile{
					Priority:   1,
					Throughput: "1Gbps",
					Latency:    "1ms",
				},
				ResourceAllocation: randmsv1.ResourceAllocation{
					PRBs: 40,
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mmtc-slice",
				Namespace: suite.testEnv.Namespace,
			},
			Spec: randmsv1.RANSliceSpec{
				SliceID:   "slice-mmtc-001",
				SliceType: "mmtc",
				QoSProfile: randmsv1.QoSProfile{
					Priority:   5,
					Throughput: "100Mbps",
					Latency:    "100ms",
				},
				ResourceAllocation: randmsv1.ResourceAllocation{
					PRBs: 20,
				},
			},
		},
	}

	// Create all slices
	for _, slice := range slices {
		err := suite.testEnv.Client.Create(suite.ctx, slice)
		require.NoError(t, err)
	}

	// Wait for all slices to be configured
	utils.AssertEventuallyWithTimeout(t, func() bool {
		for _, slice := range slices {
			updatedSlice := &randmsv1.RANSlice{}
			err := suite.testEnv.Client.Get(suite.ctx, client.ObjectKeyFromObject(slice), updatedSlice)
			if err != nil || updatedSlice.Status.Phase != "Active" {
				return false
			}
		}
		return true
	}, 45*time.Second, "All slices should be active")

	// Verify resource isolation (no overlap in allocated PRBs)
	allocatedPRBs := make(map[int]string)
	for _, slice := range slices {
		updatedSlice := &randmsv1.RANSlice{}
		err := suite.testEnv.Client.Get(suite.ctx, client.ObjectKeyFromObject(slice), updatedSlice)
		require.NoError(t, err)

		if updatedSlice.Status.AllocatedResources != nil {
			for i := 0; i < updatedSlice.Status.AllocatedResources.PRBs; i++ {
				startPRB := updatedSlice.Status.AllocatedResources.StartPRB
				prbIndex := startPRB + i

				if existingSlice, exists := allocatedPRBs[prbIndex]; exists {
					t.Errorf("PRB %d allocated to both %s and %s", prbIndex, existingSlice, slice.Name)
				}
				allocatedPRBs[prbIndex] = slice.Name
			}
		}
	}

	utils.LogTestProgress(t, "Multi-slice isolation test completed")
}

func (suite *RANDMSIntegrationTestSuite) TestPerformanceMetrics() {
	t := suite.T()
	utils.LogTestProgress(t, "Testing performance metrics collection")

	// Create performance monitoring configuration
	perfMonitor := &randmsv1.PerformanceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ran-perf-monitor",
			Namespace: suite.testEnv.Namespace,
		},
		Spec: randmsv1.PerformanceMonitorSpec{
			Targets: []randmsv1.MonitorTarget{
				{
					Type: "slice",
					Name: "embb-slice",
				},
			},
			Metrics: []randmsv1.MetricConfig{
				{
					Name:     "throughput",
					Interval: "5s",
					Unit:     "Mbps",
				},
				{
					Name:     "latency",
					Interval: "1s",
					Unit:     "ms",
				},
				{
					Name:     "prb_utilization",
					Interval: "10s",
					Unit:     "percent",
				},
			},
			Duration: "60s",
		},
	}

	err := suite.testEnv.Client.Create(suite.ctx, perfMonitor)
	require.NoError(t, err)

	// Wait for metrics collection to start
	utils.AssertEventuallyWithTimeout(t, func() bool {
		updatedMonitor := &randmsv1.PerformanceMonitor{}
		err := suite.testEnv.Client.Get(suite.ctx, client.ObjectKeyFromObject(perfMonitor), updatedMonitor)
		return err == nil && updatedMonitor.Status.Phase == "Collecting"
	}, 20*time.Second, "Performance monitoring should start")

	// Wait for metrics to be collected
	time.Sleep(10 * time.Second)

	// Verify metrics are available
	updatedMonitor := &randmsv1.PerformanceMonitor{}
	err = suite.testEnv.Client.Get(suite.ctx, client.ObjectKeyFromObject(perfMonitor), updatedMonitor)
	require.NoError(t, err)
	assert.NotEmpty(t, updatedMonitor.Status.CollectedMetrics)

	utils.LogTestProgress(t, "Performance metrics test completed")
}

func (suite *RANDMSIntegrationTestSuite) TestFailureRecovery() {
	t := suite.T()
	utils.LogTestProgress(t, "Testing failure recovery mechanisms")

	// Create RAN resource with failure injection
	ranResource := &randmsv1.RANResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "failure-test-resource",
			Namespace: suite.testEnv.Namespace,
			Annotations: map[string]string{
				"test.ran-dms.io/inject-failure": "connection-loss",
			},
		},
		Spec: randmsv1.RANResourceSpec{
			Type:     "DU",
			Location: "edge02",
			Capacity: randmsv1.ResourceCapacity{
				CPU:    "2000m",
				Memory: "4Gi",
			},
			HighAvailability: randmsv1.HAConfig{
				Enabled:       true,
				BackupSites:   []string{"edge03"},
				FailoverTime:  "30s",
				SyncInterval:  "5s",
			},
		},
	}

	err := suite.testEnv.Client.Create(suite.ctx, ranResource)
	require.NoError(t, err)

	// Wait for initial deployment
	utils.AssertEventuallyWithTimeout(t, func() bool {
		updatedResource := &randmsv1.RANResource{}
		err := suite.testEnv.Client.Get(suite.ctx, client.ObjectKeyFromObject(ranResource), updatedResource)
		return err == nil && updatedResource.Status.Phase == "Ready"
	}, 30*time.Second, "RAN resource should be ready initially")

	// Simulate failure and verify recovery
	utils.AssertEventuallyWithTimeout(t, func() bool {
		updatedResource := &randmsv1.RANResource{}
		err := suite.testEnv.Client.Get(suite.ctx, client.ObjectKeyFromObject(ranResource), updatedResource)
		if err != nil {
			return false
		}

		// Check if failover occurred
		return updatedResource.Status.ActiveSite != ranResource.Spec.Location &&
			   updatedResource.Status.Phase == "Ready"
	}, 60*time.Second, "Failover should occur and resource should recover")

	utils.LogTestProgress(t, "Failure recovery test completed")
}