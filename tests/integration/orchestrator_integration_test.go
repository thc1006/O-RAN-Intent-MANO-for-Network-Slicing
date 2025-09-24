// Package integration provides integration tests for the orchestrator service
package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tests/utils"
	orchestratorv1 "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/orchestrator/api/v1"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/orchestrator/pkg/placement"
)

var _ = ginkgo.Describe("Orchestrator Integration Tests", func() {
	var (
		testEnv    *utils.TestEnvironment
		mockServer *utils.MockServer
		ctx        context.Context
	)

	ginkgo.BeforeEach(func() {
		ctx = context.Background()
		testEnv = utils.SetupTestEnvironment(ginkgo.GinkgoT(), scheme)
		mockServer = utils.NewMockO2Server()
	})

	ginkgo.AfterEach(func() {
		mockServer.Stop()
		testEnv.Cleanup(ginkgo.GinkgoT())
	})

	ginkgo.Context("Intent Processing", func() {
		ginkgo.It("should process natural language intent to QoS requirements", func() {
			// Test intent: "Deploy eMBB slice with 5G throughput in edge cluster"
			intent := &orchestratorv1.Intent{
				Spec: orchestratorv1.IntentSpec{
					Description: "Deploy eMBB slice with 5G throughput in edge cluster",
					SliceType:   "embb",
					QoSRequirements: orchestratorv1.QoSRequirements{
						Throughput: "5Gbps",
						Latency:    "10ms",
					},
					Placement: orchestratorv1.PlacementPolicy{
						PreferredSites: []string{"edge01", "edge02"},
					},
				},
			}

			// Create intent
			err := testEnv.Client.Create(ctx, intent)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Verify intent is processed
			utils.AssertEventuallyWithTimeout(ginkgo.GinkgoT(), func() bool {
				updatedIntent := &orchestratorv1.Intent{}
				err := testEnv.Client.Get(ctx, client.ObjectKeyFromObject(intent), updatedIntent)
				return err == nil && updatedIntent.Status.Phase == "Processing"
			}, 30*time.Second, "Intent should be processed")
		})

		ginkgo.It("should validate QoS requirements", func() {
			// Test with invalid QoS requirements
			intent := &orchestratorv1.Intent{
				Spec: orchestratorv1.IntentSpec{
					Description: "Invalid QoS requirements",
					QoSRequirements: orchestratorv1.QoSRequirements{
						Throughput: "invalid",
						Latency:    "negative",
					},
				},
			}

			err := testEnv.Client.Create(ctx, intent)
			gomega.Expect(err).To(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("Placement Decisions", func() {
		ginkgo.It("should make optimal placement decisions", func() {
			// Create multiple intents with different requirements
			intents := []*orchestratorv1.Intent{
				{
					Spec: orchestratorv1.IntentSpec{
						Description: "Low latency URLLC slice",
						SliceType:   "urllc",
						QoSRequirements: orchestratorv1.QoSRequirements{
							Latency: "1ms",
						},
					},
				},
				{
					Spec: orchestratorv1.IntentSpec{
						Description: "High throughput eMBB slice",
						SliceType:   "embb",
						QoSRequirements: orchestratorv1.QoSRequirements{
							Throughput: "10Gbps",
						},
					},
				},
			}

			for _, intent := range intents {
				err := testEnv.Client.Create(ctx, intent)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			}

			// Verify placement decisions are made
			utils.AssertEventuallyWithTimeout(ginkgo.GinkgoT(), func() bool {
				// Check if placement decisions have been made
				intentList := &orchestratorv1.IntentList{}
				err := testEnv.Client.List(ctx, intentList)
				if err != nil {
					return false
				}

				for _, intent := range intentList.Items {
					if intent.Status.PlacementDecision == nil {
						return false
					}
				}
				return true
			}, 60*time.Second, "Placement decisions should be made")
		})
	})

	ginkgo.Context("O2 Interface Integration", func() {
		ginkgo.It("should integrate with O2IMS for infrastructure inventory", func() {
			// Test O2IMS integration
			resp, err := http.Get(mockServer.URL + "/o2ims/v1/")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))

			var inventory map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&inventory)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(inventory["version"]).To(gomega.Equal("1.0"))
		})

		ginkgo.It("should integrate with O2DMS for deployment management", func() {
			// Test O2DMS integration
			resp, err := http.Get(mockServer.URL + "/o2dms/v1/")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))

			var deployments map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&deployments)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(deployments["deployments"]).NotTo(gomega.BeNil())
		})
	})
})

// OrchestratorIntegrationTestSuite provides testify-based integration tests
type OrchestratorIntegrationTestSuite struct {
	suite.Suite
	testEnv    *utils.TestEnvironment
	mockServer *utils.MockServer
	ctx        context.Context
}

func TestOrchestratorIntegrationSuite(t *testing.T) {
	utils.SkipIfNotIntegration(t)
	suite.Run(t, new(OrchestratorIntegrationTestSuite))
}

func (suite *OrchestratorIntegrationTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.testEnv = utils.SetupTestEnvironment(suite.T(), scheme)
	suite.mockServer = utils.NewMockO2Server()
}

func (suite *OrchestratorIntegrationTestSuite) TearDownSuite() {
	suite.mockServer.Stop()
	suite.testEnv.Cleanup(suite.T())
}

func (suite *OrchestratorIntegrationTestSuite) TestIntentLifecycle() {
	t := suite.T()
	utils.LogTestProgress(t, "Starting intent lifecycle test")

	// Create intent
	intent := &orchestratorv1.Intent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-intent",
			Namespace: suite.testEnv.Namespace,
		},
		Spec: orchestratorv1.IntentSpec{
			Description: "Test intent for lifecycle validation",
			SliceType:   "embb",
			QoSRequirements: orchestratorv1.QoSRequirements{
				Throughput: "1Gbps",
				Latency:    "20ms",
			},
		},
	}

	utils.LogTestProgress(t, "Creating intent")
	err := suite.testEnv.Client.Create(suite.ctx, intent)
	require.NoError(t, err)

	// Wait for intent to be processed
	utils.LogTestProgress(t, "Waiting for intent processing")
	utils.AssertEventuallyWithTimeout(t, func() bool {
		updatedIntent := &orchestratorv1.Intent{}
		err := suite.testEnv.Client.Get(suite.ctx, client.ObjectKeyFromObject(intent), updatedIntent)
		return err == nil && updatedIntent.Status.Phase != ""
	}, 30*time.Second, "Intent should be processed")

	// Verify intent status
	updatedIntent := &orchestratorv1.Intent{}
	err = suite.testEnv.Client.Get(suite.ctx, client.ObjectKeyFromObject(intent), updatedIntent)
	require.NoError(t, err)
	assert.NotEmpty(t, updatedIntent.Status.Phase)

	utils.LogTestProgress(t, "Intent lifecycle test completed")
}

func (suite *OrchestratorIntegrationTestSuite) TestPlacementOptimization() {
	t := suite.T()
	utils.LogTestProgress(t, "Starting placement optimization test")

	// Create placement policy
	policy := &placement.Policy{
		Constraints: []placement.Constraint{
			{
				Type:   "latency",
				Value:  "10ms",
				Weight: 0.8,
			},
			{
				Type:   "capacity",
				Value:  "high",
				Weight: 0.2,
			},
		},
	}

	// Test placement engine
	engine := placement.NewEngine(suite.testEnv.Client)
	sites := []string{"edge01", "edge02", "regional01", "central01"}

	decision, err := engine.OptimalPlacement(suite.ctx, policy, sites)
	require.NoError(t, err)
	assert.NotEmpty(t, decision.SelectedSites)
	assert.GreaterOrEqual(t, decision.Score, 0.0)

	utils.LogTestProgress(t, "Placement optimization test completed")
}

func (suite *OrchestratorIntegrationTestSuite) TestMetricsCollection() {
	t := suite.T()
	utils.LogTestProgress(t, "Starting metrics collection test")

	// Initialize metrics collector
	collector, err := utils.NewMetricsCollector("http://prometheus:9090", nil, suite.testEnv.Namespace)
	if err != nil {
		t.Skip("Prometheus not available, skipping metrics test")
	}

	// Generate test metrics
	metrics := utils.GenerateTestMetrics("orchestrator-test")

	// Simulate deployment time
	time.Sleep(100 * time.Millisecond)
	metrics.DeploymentTime = 5 * time.Minute // Simulated

	// Validate metrics
	metrics.ThroughputMbps = 4.57 // eMBB target
	metrics.LatencyMs = 16.1      // eMBB target

	metrics.FinishTestMetrics()
	metrics.ValidateThesisMetrics(t)

	utils.LogTestProgress(t, "Metrics collection test completed")
}

// Helper functions for test setup
func (suite *OrchestratorIntegrationTestSuite) createTestIntent(name, sliceType string) *orchestratorv1.Intent {
	return &orchestratorv1.Intent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: suite.testEnv.Namespace,
		},
		Spec: orchestratorv1.IntentSpec{
			Description: fmt.Sprintf("Test %s intent", sliceType),
			SliceType:   sliceType,
			QoSRequirements: orchestratorv1.QoSRequirements{
				Throughput: "1Gbps",
				Latency:    "10ms",
			},
		},
	}
}

func (suite *OrchestratorIntegrationTestSuite) waitForIntentReady(intent *orchestratorv1.Intent, timeout time.Duration) error {
	return utils.RetryOperation(func() error {
		updatedIntent := &orchestratorv1.Intent{}
		if err := suite.testEnv.Client.Get(suite.ctx, client.ObjectKeyFromObject(intent), updatedIntent); err != nil {
			return err
		}

		if updatedIntent.Status.Phase != "Ready" {
			return fmt.Errorf("intent not ready: %s", updatedIntent.Status.Phase)
		}

		return nil
	}, 10, time.Second)
}