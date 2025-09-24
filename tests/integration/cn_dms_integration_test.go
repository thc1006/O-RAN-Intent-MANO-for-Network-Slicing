// Package integration provides integration tests for CN-DMS service
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
	cndmsv1 "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/cn-dms/api/v1alpha1"
)

var _ = ginkgo.Describe("CN-DMS Integration Tests", func() {
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

	ginkgo.Context("Core Network Deployment", func() {
		ginkgo.It("should deploy 5G core network functions", func() {
			// Create 5G core deployment
			coreNetwork := &cndmsv1.CoreNetwork{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-5g-core",
					Namespace: testEnv.Namespace,
				},
				Spec: cndmsv1.CoreNetworkSpec{
					Release: "R16",
					Deployment: cndmsv1.DeploymentConfig{
						Architecture: "SBA", // Service-Based Architecture
						Mode:         "standalone",
					},
					NetworkFunctions: []cndmsv1.NetworkFunction{
						{
							Type:    "AMF",
							Version: "v1.0.0",
							Replicas: 2,
							Resources: cndmsv1.ResourceRequirements{
								CPU:    "1000m",
								Memory: "2Gi",
							},
						},
						{
							Type:    "SMF",
							Version: "v1.0.0",
							Replicas: 2,
							Resources: cndmsv1.ResourceRequirements{
								CPU:    "500m",
								Memory: "1Gi",
							},
						},
						{
							Type:    "UPF",
							Version: "v1.0.0",
							Replicas: 3,
							Resources: cndmsv1.ResourceRequirements{
								CPU:    "2000m",
								Memory: "4Gi",
							},
						},
					},
				},
			}

			err := testEnv.Client.Create(ctx, coreNetwork)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Verify core network is deployed
			utils.AssertEventuallyWithTimeout(ginkgo.GinkgoT(), func() bool {
				updatedCore := &cndmsv1.CoreNetwork{}
				err := testEnv.Client.Get(ctx, client.ObjectKeyFromObject(coreNetwork), updatedCore)
				return err == nil && updatedCore.Status.Phase == "Running"
			}, 120*time.Second, "Core network should be running")
		})

		ginkgo.It("should configure network slicing", func() {
			// Create network slice configuration
			networkSlice := &cndmsv1.NetworkSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-network-slice",
					Namespace: testEnv.Namespace,
				},
				Spec: cndmsv1.NetworkSliceSpec{
					SNSSAI: cndmsv1.SNSSAI{
						SST: 1,
						SD:  "000001",
					},
					SliceType: "embb",
					QoSProfile: cndmsv1.QoSProfile{
						ULThroughput: "5Gbps",
						DLThroughput: "10Gbps",
						Latency:      "10ms",
						Reliability:  "99.99%",
					},
					Coverage: []cndmsv1.CoverageArea{
						{
							Type: "city",
							Name: "metropolitan-area",
							Coordinates: cndmsv1.GeographicCoordinates{
								Latitude:  40.7589,
								Longitude: -73.9851,
								Radius:    50000, // 50km radius
							},
						},
					},
				},
			}

			err := testEnv.Client.Create(ctx, networkSlice)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Verify network slice is configured
			utils.AssertEventuallyWithTimeout(ginkgo.GinkgoT(), func() bool {
				updatedSlice := &cndmsv1.NetworkSlice{}
				err := testEnv.Client.Get(ctx, client.ObjectKeyFromObject(networkSlice), updatedSlice)
				return err == nil && updatedSlice.Status.Phase == "Active"
			}, 90*time.Second, "Network slice should be active")
		})
	})

	ginkgo.Context("Service Orchestration", func() {
		ginkgo.It("should orchestrate network services", func() {
			// Create network service
			networkService := &cndmsv1.NetworkService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-network-service",
					Namespace: testEnv.Namespace,
				},
				Spec: cndmsv1.NetworkServiceSpec{
					ServiceID: "ns-001",
					Type:      "voice-call",
					Template:  "voice-service-template",
					Parameters: map[string]string{
						"codec":       "AMR-WB",
						"bitrate":     "12.65kbps",
						"maxDelay":    "150ms",
						"packetLoss":  "1%",
					},
					Endpoints: []cndmsv1.ServiceEndpoint{
						{
							Name: "sip-proxy",
							Port: 5060,
							Protocol: "UDP",
						},
					},
				},
			}

			err := testEnv.Client.Create(ctx, networkService)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Verify service orchestration
			utils.AssertEventuallyWithTimeout(ginkgo.GinkgoT(), func() bool {
				updatedService := &cndmsv1.NetworkService{}
				err := testEnv.Client.Get(ctx, client.ObjectKeyFromObject(networkService), updatedService)
				return err == nil &&
					   updatedService.Status.Phase == "Deployed" &&
					   len(updatedService.Status.ActiveEndpoints) > 0
			}, 60*time.Second, "Network service should be deployed with active endpoints")
		})
	})
})

// CNDMSIntegrationTestSuite provides comprehensive testify-based tests
type CNDMSIntegrationTestSuite struct {
	suite.Suite
	testEnv *utils.TestEnvironment
	ctx     context.Context
}

func TestCNDMSIntegrationSuite(t *testing.T) {
	utils.SkipIfNotIntegration(t)
	suite.Run(t, new(CNDMSIntegrationTestSuite))
}

func (suite *CNDMSIntegrationTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.testEnv = utils.SetupTestEnvironment(suite.T(), scheme)
}

func (suite *CNDMSIntegrationTestSuite) TearDownSuite() {
	suite.testEnv.Cleanup(suite.T())
}

func (suite *CNDMSIntegrationTestSuite) Test5GCoreDeployment() {
	t := suite.T()
	utils.LogTestProgress(t, "Testing 5G core network deployment")

	// Create minimal 5G core configuration
	coreNetwork := &cndmsv1.CoreNetwork{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "minimal-5g-core",
			Namespace: suite.testEnv.Namespace,
		},
		Spec: cndmsv1.CoreNetworkSpec{
			Release: "R17",
			Deployment: cndmsv1.DeploymentConfig{
				Architecture: "SBA",
				Mode:         "standalone",
				Scale:        "small",
			},
			NetworkFunctions: []cndmsv1.NetworkFunction{
				{
					Type:     "AMF",
					Version:  "v1.2.0",
					Replicas: 1,
					Resources: cndmsv1.ResourceRequirements{
						CPU:    "500m",
						Memory: "1Gi",
					},
					Configuration: map[string]string{
						"plmn.mcc": "001",
						"plmn.mnc": "01",
						"tai.tac":  "0001",
					},
				},
				{
					Type:     "SMF",
					Version:  "v1.2.0",
					Replicas: 1,
					Resources: cndmsv1.ResourceRequirements{
						CPU:    "300m",
						Memory: "512Mi",
					},
				},
			},
		},
	}

	err := suite.testEnv.Client.Create(suite.ctx, coreNetwork)
	require.NoError(t, err)

	// Wait for deployment to complete
	utils.AssertEventuallyWithTimeout(t, func() bool {
		updatedCore := &cndmsv1.CoreNetwork{}
		err := suite.testEnv.Client.Get(suite.ctx, client.ObjectKeyFromObject(coreNetwork), updatedCore)
		if err != nil {
			return false
		}

		return updatedCore.Status.Phase == "Running" &&
			   len(updatedCore.Status.DeployedNFs) >= len(coreNetwork.Spec.NetworkFunctions)
	}, 180*time.Second, "5G core should be fully deployed")

	// Verify network function status
	updatedCore := &cndmsv1.CoreNetwork{}
	err = suite.testEnv.Client.Get(suite.ctx, client.ObjectKeyFromObject(coreNetwork), updatedCore)
	require.NoError(t, err)
	assert.Equal(t, "Running", updatedCore.Status.Phase)
	assert.GreaterOrEqual(t, len(updatedCore.Status.DeployedNFs), 2)

	utils.LogTestProgress(t, "5G core deployment test completed")
}

func (suite *CNDMSIntegrationTestSuite) TestNetworkSliceLifecycle() {
	t := suite.T()
	utils.LogTestProgress(t, "Testing network slice lifecycle")

	// Create URLLC network slice
	urllcSlice := &cndmsv1.NetworkSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "urllc-slice",
			Namespace: suite.testEnv.Namespace,
		},
		Spec: cndmsv1.NetworkSliceSpec{
			SNSSAI: cndmsv1.SNSSAI{
				SST: 1,
				SD:  "000100",
			},
			SliceType: "urllc",
			QoSProfile: cndmsv1.QoSProfile{
				ULThroughput: "1Gbps",
				DLThroughput: "1Gbps",
				Latency:      "1ms",
				Reliability:  "99.9999%",
				PacketLoss:   "0.00001%",
			},
			IsolationLevel: "physical",
			Priority:       1,
		},
	}

	err := suite.testEnv.Client.Create(suite.ctx, urllcSlice)
	require.NoError(t, err)

	// Wait for slice activation
	utils.AssertEventuallyWithTimeout(t, func() bool {
		updatedSlice := &cndmsv1.NetworkSlice{}
		err := suite.testEnv.Client.Get(suite.ctx, client.ObjectKeyFromObject(urllcSlice), updatedSlice)
		return err == nil && updatedSlice.Status.Phase == "Active"
	}, 120*time.Second, "URLLC slice should be active")

	// Verify slice configuration
	updatedSlice := &cndmsv1.NetworkSlice{}
	err = suite.testEnv.Client.Get(suite.ctx, client.ObjectKeyFromObject(urllcSlice), updatedSlice)
	require.NoError(t, err)
	assert.Equal(t, "Active", updatedSlice.Status.Phase)
	assert.NotEmpty(t, updatedSlice.Status.AllocatedResources)

	// Test slice modification
	updatedSlice.Spec.QoSProfile.ULThroughput = "2Gbps"
	err = suite.testEnv.Client.Update(suite.ctx, updatedSlice)
	require.NoError(t, err)

	// Wait for slice reconfiguration
	utils.AssertEventuallyWithTimeout(t, func() bool {
		reconfSlice := &cndmsv1.NetworkSlice{}
		err := suite.testEnv.Client.Get(suite.ctx, client.ObjectKeyFromObject(urllcSlice), reconfSlice)
		return err == nil &&
			   reconfSlice.Status.Phase == "Active" &&
			   reconfSlice.Status.LastReconfigured.After(reconfSlice.CreationTimestamp.Time)
	}, 90*time.Second, "Slice should be reconfigured")

	utils.LogTestProgress(t, "Network slice lifecycle test completed")
}

func (suite *CNDMSIntegrationTestSuite) TestMultiTenantIsolation() {
	t := suite.T()
	utils.LogTestProgress(t, "Testing multi-tenant isolation")

	// Create multiple network slices for different tenants
	tenantSlices := []*cndmsv1.NetworkSlice{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "enterprise-slice",
				Namespace: suite.testEnv.Namespace,
				Labels: map[string]string{
					"tenant": "enterprise-corp",
					"sla":    "premium",
				},
			},
			Spec: cndmsv1.NetworkSliceSpec{
				SNSSAI: cndmsv1.SNSSAI{
					SST: 1,
					SD:  "000200",
				},
				SliceType: "embb",
				QoSProfile: cndmsv1.QoSProfile{
					ULThroughput: "10Gbps",
					DLThroughput: "50Gbps",
					Latency:      "5ms",
				},
				IsolationLevel: "logical",
				TenantID:       "enterprise-corp",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mvno-slice",
				Namespace: suite.testEnv.Namespace,
				Labels: map[string]string{
					"tenant": "mvno-partner",
					"sla":    "standard",
				},
			},
			Spec: cndmsv1.NetworkSliceSpec{
				SNSSAI: cndmsv1.SNSSAI{
					SST: 1,
					SD:  "000300",
				},
				SliceType: "embb",
				QoSProfile: cndmsv1.QoSProfile{
					ULThroughput: "5Gbps",
					DLThroughput: "20Gbps",
					Latency:      "10ms",
				},
				IsolationLevel: "logical",
				TenantID:       "mvno-partner",
			},
		},
	}

	// Deploy all tenant slices
	for _, slice := range tenantSlices {
		err := suite.testEnv.Client.Create(suite.ctx, slice)
		require.NoError(t, err)
	}

	// Wait for all slices to be active
	utils.AssertEventuallyWithTimeout(t, func() bool {
		for _, slice := range tenantSlices {
			updatedSlice := &cndmsv1.NetworkSlice{}
			err := suite.testEnv.Client.Get(suite.ctx, client.ObjectKeyFromObject(slice), updatedSlice)
			if err != nil || updatedSlice.Status.Phase != "Active" {
				return false
			}
		}
		return true
	}, 150*time.Second, "All tenant slices should be active")

	// Verify resource isolation
	for _, slice := range tenantSlices {
		updatedSlice := &cndmsv1.NetworkSlice{}
		err := suite.testEnv.Client.Get(suite.ctx, client.ObjectKeyFromObject(slice), updatedSlice)
		require.NoError(t, err)

		// Verify tenant isolation
		assert.Equal(t, slice.Spec.TenantID, updatedSlice.Status.TenantID)
		assert.NotEmpty(t, updatedSlice.Status.IsolationConfig)
		assert.NotNil(t, updatedSlice.Status.AllocatedResources)
	}

	utils.LogTestProgress(t, "Multi-tenant isolation test completed")
}

func (suite *CNDMSIntegrationTestSuite) TestServiceChaining() {
	t := suite.T()
	utils.LogTestProgress(t, "Testing service function chaining")

	// Create service chain
	serviceChain := &cndmsv1.ServiceChain{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "security-chain",
			Namespace: suite.testEnv.Namespace,
		},
		Spec: cndmsv1.ServiceChainSpec{
			ChainID: "sc-001",
			Services: []cndmsv1.ServiceFunction{
				{
					Name:    "firewall",
					Type:    "security",
					Version: "v2.1.0",
					Order:   1,
					Configuration: map[string]string{
						"rules": "allow-http,allow-https,deny-all",
					},
				},
				{
					Name:    "dpi",
					Type:    "inspection",
					Version: "v1.5.0",
					Order:   2,
					Configuration: map[string]string{
						"inspection-depth": "deep",
						"threat-detection": "enabled",
					},
				},
				{
					Name:    "load-balancer",
					Type:    "traffic-management",
					Version: "v3.0.0",
					Order:   3,
					Configuration: map[string]string{
						"algorithm":      "round-robin",
						"health-checks":  "enabled",
						"session-affinity": "source-ip",
					},
				},
			},
			TrafficPolicies: []cndmsv1.TrafficPolicy{
				{
					Type:      "ingress",
					Selector:  "app=web-service",
					ChainRule: "security-chain",
				},
			},
		},
	}

	err := suite.testEnv.Client.Create(suite.ctx, serviceChain)
	require.NoError(t, err)

	// Wait for service chain deployment
	utils.AssertEventuallyWithTimeout(t, func() bool {
		updatedChain := &cndmsv1.ServiceChain{}
		err := suite.testEnv.Client.Get(suite.ctx, client.ObjectKeyFromObject(serviceChain), updatedChain)
		if err != nil {
			return false
		}

		return updatedChain.Status.Phase == "Active" &&
			   len(updatedChain.Status.DeployedServices) == len(serviceChain.Spec.Services)
	}, 120*time.Second, "Service chain should be fully deployed")

	// Verify service chain configuration
	updatedChain := &cndmsv1.ServiceChain{}
	err = suite.testEnv.Client.Get(suite.ctx, client.ObjectKeyFromObject(serviceChain), updatedChain)
	require.NoError(t, err)
	assert.Equal(t, "Active", updatedChain.Status.Phase)
	assert.Len(t, updatedChain.Status.DeployedServices, 3)

	// Verify service ordering
	for i, service := range updatedChain.Status.DeployedServices {
		expectedOrder := i + 1
		assert.Equal(t, expectedOrder, service.Order)
	}

	utils.LogTestProgress(t, "Service chaining test completed")
}

func (suite *CNDMSIntegrationTestSuite) TestPerformanceMonitoring() {
	t := suite.T()
	utils.LogTestProgress(t, "Testing performance monitoring")

	// Create performance monitor for CN services
	perfMonitor := &cndmsv1.PerformanceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cn-perf-monitor",
			Namespace: suite.testEnv.Namespace,
		},
		Spec: cndmsv1.PerformanceMonitorSpec{
			Targets: []cndmsv1.MonitoringTarget{
				{
					Type: "network-function",
					Name: "amf",
					Metrics: []string{"cpu", "memory", "throughput", "latency"},
				},
				{
					Type: "network-slice",
					Name: "urllc-slice",
					Metrics: []string{"packet-loss", "jitter", "availability"},
				},
			},
			SamplingInterval: "10s",
			RetentionPeriod:  "24h",
			Thresholds: []cndmsv1.Threshold{
				{
					Metric:    "latency",
					Condition: "greater-than",
					Value:     "5ms",
					Action:    "alert",
				},
				{
					Metric:    "cpu",
					Condition: "greater-than",
					Value:     "80%",
					Action:    "scale-up",
				},
			},
		},
	}

	err := suite.testEnv.Client.Create(suite.ctx, perfMonitor)
	require.NoError(t, err)

	// Wait for monitoring to start
	utils.AssertEventuallyWithTimeout(t, func() bool {
		updatedMonitor := &cndmsv1.PerformanceMonitor{}
		err := suite.testEnv.Client.Get(suite.ctx, client.ObjectKeyFromObject(perfMonitor), updatedMonitor)
		return err == nil && updatedMonitor.Status.Phase == "Monitoring"
	}, 30*time.Second, "Performance monitoring should start")

	// Wait for some metrics collection
	time.Sleep(15 * time.Second)

	// Verify metrics are being collected
	updatedMonitor := &cndmsv1.PerformanceMonitor{}
	err = suite.testEnv.Client.Get(suite.ctx, client.ObjectKeyFromObject(perfMonitor), updatedMonitor)
	require.NoError(t, err)
	assert.Equal(t, "Monitoring", updatedMonitor.Status.Phase)
	assert.NotEmpty(t, updatedMonitor.Status.LastSampleTime)

	utils.LogTestProgress(t, "Performance monitoring test completed")
}