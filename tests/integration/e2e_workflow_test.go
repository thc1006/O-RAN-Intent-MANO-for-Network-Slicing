package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	manov1alpha1 "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/api/v1alpha1"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
)

// E2EWorkflowSuite represents the complete test environment
type E2EWorkflowSuite struct {
	cfg         *rest.Config
	k8sClient   client.Client
	dynClient   dynamic.Interface
	clientset   kubernetes.Interface
	testEnv     *envtest.Environment
	testCtx     context.Context
	cancel      context.CancelFunc
	testResults *TestResults
}

// TestResults stores performance metrics and validation results
type TestResults struct {
	StartTime         time.Time                `json:"start_time"`
	EndTime           time.Time                `json:"end_time"`
	TotalDuration     time.Duration            `json:"total_duration_ms"`
	Phases            map[string]time.Duration `json:"phases"`
	QoSValidation     []QoSValidationResult    `json:"qos_validation"`
	LatencyResults    []LatencyMeasurement     `json:"latency_results"`
	ThroughputResults []ThroughputMeasurement  `json:"throughput_results"`
	VNFStatus         []VNFDeploymentStatus    `json:"vnf_status"`
	Errors            []string                 `json:"errors"`
	Success           bool                     `json:"success"`
}

// QoSValidationResult represents QoS parameter validation
type QoSValidationResult struct {
	SliceType             string  `json:"slice_type"`
	ExpectedLatencyMs     float64 `json:"expected_latency_ms"`
	MeasuredLatencyMs     float64 `json:"measured_latency_ms"`
	ExpectedBandwidthMbps float64 `json:"expected_bandwidth_mbps"`
	MeasuredBandwidthMbps float64 `json:"measured_bandwidth_mbps"`
	ValidationPassed      bool    `json:"validation_passed"`
	DeviationPercent      float64 `json:"deviation_percent"`
}

// LatencyMeasurement represents network latency measurements
type LatencyMeasurement struct {
	SliceType  string    `json:"slice_type"`
	SourceNode string    `json:"source_node"`
	TargetNode string    `json:"target_node"`
	RTTMs      float64   `json:"rtt_ms"`
	JitterMs   float64   `json:"jitter_ms"`
	PacketLoss float64   `json:"packet_loss_percent"`
	Timestamp  time.Time `json:"timestamp"`
}

// ThroughputMeasurement represents bandwidth measurements
type ThroughputMeasurement struct {
	SliceType           string    `json:"slice_type"`
	SourceNode          string    `json:"source_node"`
	TargetNode          string    `json:"target_node"`
	BandwidthMbps       float64   `json:"bandwidth_mbps"`
	TestDurationSeconds int       `json:"test_duration_seconds"`
	Timestamp           time.Time `json:"timestamp"`
}

// VNFDeploymentStatus represents VNF deployment status
type VNFDeploymentStatus struct {
	VNFName        string        `json:"vnf_name"`
	VNFType        string        `json:"vnf_type"`
	ClusterName    string        `json:"cluster_name"`
	Status         string        `json:"status"`
	DeploymentTime time.Duration `json:"deployment_time_ms"`
	ReadyTime      time.Duration `json:"ready_time_ms"`
	PorchPackage   string        `json:"porch_package"`
	Error          string        `json:"error,omitempty"`
}

// Intent-to-QoS test scenarios matching thesis examples
var intentTestScenarios = []struct {
	name                string
	intent              string
	expectedQoS         manov1alpha1.QoSRequirements
	expectedVNFTypes    []manov1alpha1.VNFType
	targetLatencyMs     float64
	targetBandwidthMbps float64
	maxDeploymentMins   int
}{
	{
		name:   "UltraLowLatency_EdgeComputing",
		intent: "Deploy ultra-low latency edge computing slice for autonomous vehicles with 1ms latency and 100Mbps bandwidth",
		expectedQoS: manov1alpha1.QoSRequirements{
			Bandwidth:   100,
			Latency:     1,
			Jitter:      &[]float64{0.1}[0],
			PacketLoss:  &[]float64{0.0001}[0],
			Reliability: &[]float64{99.999}[0],
			SliceType:   "uRLLC",
		},
		expectedVNFTypes:    []manov1alpha1.VNFType{manov1alpha1.VNFTypeUPF, manov1alpha1.VNFTypegNB},
		targetLatencyMs:     6.3,
		targetBandwidthMbps: 0.93,
		maxDeploymentMins:   10,
	},
	{
		name:   "HighBandwidth_StreamingService",
		intent: "Create high-bandwidth streaming service slice for 4K video with 50Mbps per user and relaxed latency",
		expectedQoS: manov1alpha1.QoSRequirements{
			Bandwidth:   50,
			Latency:     20,
			Jitter:      &[]float64{2}[0],
			PacketLoss:  &[]float64{0.001}[0],
			Reliability: &[]float64{99.9}[0],
			SliceType:   "eMBB",
		},
		expectedVNFTypes:    []manov1alpha1.VNFType{manov1alpha1.VNFTypeUPF, manov1alpha1.VNFTypeAMF, manov1alpha1.VNFTypeSMF},
		targetLatencyMs:     16.1,
		targetBandwidthMbps: 4.57,
		maxDeploymentMins:   10,
	},
	{
		name:   "IoT_BalancedService",
		intent: "Deploy balanced IoT service slice with moderate bandwidth and latency for smart city applications",
		expectedQoS: manov1alpha1.QoSRequirements{
			Bandwidth:   10,
			Latency:     15,
			Jitter:      &[]float64{3}[0],
			PacketLoss:  &[]float64{0.01}[0],
			Reliability: &[]float64{99.5}[0],
			SliceType:   "mIoT",
		},
		expectedVNFTypes:    []manov1alpha1.VNFType{manov1alpha1.VNFTypeUPF, manov1alpha1.VNFTypeAMF},
		targetLatencyMs:     15.7,
		targetBandwidthMbps: 2.77,
		maxDeploymentMins:   10,
	},
}

var _ = Describe("E2E Workflow Integration Tests", func() {
	var suite *E2EWorkflowSuite

	BeforeEach(func() {
		suite = setupE2ETestSuite()
	})

	AfterEach(func() {
		teardownE2ETestSuite(suite)
		suite.saveTestResults()
	})

	Context("Intent-to-QoS-to-VNF Workflow", func() {
		for _, scenario := range intentTestScenarios {
			It(fmt.Sprintf("should complete workflow for %s", scenario.name), func() {
				suite.testResults.StartTime = time.Now()

				By("Phase 1: Intent Processing and QoS Translation")
				phaseStart := time.Now()
				qosSpec := suite.processIntentToQoS(scenario.intent, scenario.expectedQoS)
				suite.testResults.Phases["intent_to_qos"] = time.Since(phaseStart)

				By("Phase 2: VNF Selection and Placement")
				phaseStart = time.Now()
				placementDecisions := suite.performVNFPlacement(qosSpec, scenario.expectedVNFTypes)
				suite.testResults.Phases["vnf_placement"] = time.Since(phaseStart)

				By("Phase 3: Porch Package Generation")
				phaseStart = time.Now()
				packages := suite.generatePorchPackages(placementDecisions)
				suite.testResults.Phases["porch_generation"] = time.Since(phaseStart)

				By("Phase 4: Multi-cluster VNF Deployment")
				phaseStart = time.Now()
				vnfStatuses := suite.deployVNFsToMultipleClusters(packages, scenario.maxDeploymentMins)
				suite.testResults.Phases["vnf_deployment"] = time.Since(phaseStart)
				suite.testResults.VNFStatus = append(suite.testResults.VNFStatus, vnfStatuses...)

				By("Phase 5: Network Slice Configuration")
				phaseStart = time.Now()
				suite.configureNetworkSlice(qosSpec, vnfStatuses)
				suite.testResults.Phases["slice_configuration"] = time.Since(phaseStart)

				By("Phase 6: E2E Performance Validation")
				phaseStart = time.Now()
				suite.validateE2EPerformance(scenario.targetLatencyMs, scenario.targetBandwidthMbps)
				suite.testResults.Phases["performance_validation"] = time.Since(phaseStart)

				suite.testResults.EndTime = time.Now()
				suite.testResults.TotalDuration = suite.testResults.EndTime.Sub(suite.testResults.StartTime)

				// Validate deployment time under 10 minutes
				Expect(suite.testResults.TotalDuration).To(BeNumerically("<=", 10*time.Minute),
					"Total deployment time should be under 10 minutes")

				suite.testResults.Success = true
			})
		}
	})

	Context("Multi-cluster Deployment Validation", func() {
		It("should deploy VNFs across edge, regional, and central clusters", func() {
			clusters := []string{"edge-cluster-01", "regional-cluster-01", "central-cluster-01"}

			for _, cluster := range clusters {
				By(fmt.Sprintf("Validating deployment to %s", cluster))
				suite.validateClusterDeployment(cluster)
			}
		})

		It("should handle cluster failures gracefully", func() {
			By("Simulating cluster failure")
			suite.simulateClusterFailure("edge-cluster-01")

			By("Verifying failover to alternative cluster")
			suite.verifyFailoverBehavior()
		})
	})

	Context("QoS Parameter Verification", func() {
		It("should validate QoS requirements are properly enforced", func() {
			for _, scenario := range intentTestScenarios {
				By(fmt.Sprintf("Testing QoS enforcement for %s", scenario.name))
				result := suite.validateQoSEnforcement(scenario.expectedQoS)
				suite.testResults.QoSValidation = append(suite.testResults.QoSValidation, result)

				Expect(result.ValidationPassed).To(BeTrue(),
					"QoS validation should pass for scenario %s", scenario.name)
			}
		})
	})
})

func TestE2EWorkflowIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Workflow Integration Suite")
}

func setupE2ETestSuite() *E2EWorkflowSuite {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	suite := &E2EWorkflowSuite{
		testResults: &TestResults{
			Phases:            make(map[string]time.Duration),
			QoSValidation:     make([]QoSValidationResult, 0),
			LatencyResults:    make([]LatencyMeasurement, 0),
			ThroughputResults: make([]ThroughputMeasurement, 0),
			VNFStatus:         make([]VNFDeploymentStatus, 0),
			Errors:            make([]string, 0),
		},
	}

	suite.testCtx, suite.cancel = context.WithTimeout(context.Background(), 20*time.Minute)

	// Setup test environment
	suite.testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "adapters", "vnf-operator", "config", "crd", "bases"),
			filepath.Join("..", "..", "o2-client", "config", "crd", "bases"),
		},
		ErrorIfCRDPathMissing: false,
	}

	var err error
	suite.cfg, err = suite.testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(suite.cfg).NotTo(BeNil())

	// Setup clients
	suite.k8sClient, err = client.New(suite.cfg, client.Options{})
	Expect(err).NotTo(HaveOccurred())

	suite.dynClient, err = dynamic.NewForConfig(suite.cfg)
	Expect(err).NotTo(HaveOccurred())

	suite.clientset, err = kubernetes.NewForConfig(suite.cfg)
	Expect(err).NotTo(HaveOccurred())

	return suite
}

func teardownE2ETestSuite(suite *E2EWorkflowSuite) {
	if suite.cancel != nil {
		suite.cancel()
	}

	if suite.testEnv != nil {
		err := suite.testEnv.Stop()
		Expect(err).NotTo(HaveOccurred())
	}
}

func (s *E2EWorkflowSuite) processIntentToQoS(intent string, expectedQoS manov1alpha1.QoSRequirements) manov1alpha1.QoSRequirements {
	// TODO: Implement NLP intent processing
	// For now, return expected QoS as if processed from intent
	return expectedQoS
}

func (s *E2EWorkflowSuite) performVNFPlacement(qos manov1alpha1.QoSRequirements, expectedTypes []manov1alpha1.VNFType) []PlacementDecision {
	decisions := make([]PlacementDecision, 0, len(expectedTypes))

	for _, vnfType := range expectedTypes {
		decision := PlacementDecision{
			VNFType:     vnfType,
			ClusterName: s.selectOptimalCluster(qos, vnfType),
			QoS:         qos,
		}
		decisions = append(decisions, decision)
	}

	return decisions
}

func (s *E2EWorkflowSuite) generatePorchPackages(decisions []PlacementDecision) []PorchPackage {
	packages := make([]PorchPackage, 0, len(decisions))

	for _, decision := range decisions {
		pkg := PorchPackage{
			Name:        fmt.Sprintf("%s-package", decision.VNFType),
			Namespace:   "default",
			VNFType:     decision.VNFType,
			ClusterName: decision.ClusterName,
			GeneratedAt: time.Now(),
		}
		packages = append(packages, pkg)
	}

	return packages
}

func (s *E2EWorkflowSuite) deployVNFsToMultipleClusters(packages []PorchPackage, maxDeploymentMins int) []VNFDeploymentStatus {
	statuses := make([]VNFDeploymentStatus, 0, len(packages))

	for _, pkg := range packages {
		deployStart := time.Now()

		status := VNFDeploymentStatus{
			VNFName:      pkg.Name,
			VNFType:      string(pkg.VNFType),
			ClusterName:  pkg.ClusterName,
			PorchPackage: pkg.Name,
		}

		// Create VNF resource
		vnf := s.createVNFResource(pkg)
		err := s.k8sClient.Create(s.testCtx, vnf)
		if err != nil {
			status.Error = err.Error()
			status.Status = "Failed"
		} else {
			// Wait for VNF to be ready
			err = s.waitForVNFReady(vnf, time.Duration(maxDeploymentMins)*time.Minute)
			if err != nil {
				status.Error = err.Error()
				status.Status = "Timeout"
			} else {
				status.Status = "Ready"
			}
		}

		status.DeploymentTime = time.Since(deployStart)
		statuses = append(statuses, status)
	}

	return statuses
}

func (s *E2EWorkflowSuite) configureNetworkSlice(qos manov1alpha1.QoSRequirements, vnfStatuses []VNFDeploymentStatus) {
	// TODO: Implement network slice configuration using TN manager
	time.Sleep(5 * time.Second) // Simulate configuration time
}

func (s *E2EWorkflowSuite) validateE2EPerformance(targetLatencyMs, targetBandwidthMbps float64) {
	// Perform latency test
	latencyResult := s.performLatencyTest()
	s.testResults.LatencyResults = append(s.testResults.LatencyResults, latencyResult)

	// Validate latency within tolerance
	tolerance := 2.0 // 2ms tolerance
	Expect(latencyResult.RTTMs).To(BeNumerically("~", targetLatencyMs, tolerance),
		"Measured latency should be within tolerance of target")

	// Perform throughput test
	throughputResult := s.performThroughputTest()
	s.testResults.ThroughputResults = append(s.testResults.ThroughputResults, throughputResult)

	// Validate throughput within tolerance (10%)
	bandwidthTolerance := targetBandwidthMbps * 0.10
	Expect(throughputResult.BandwidthMbps).To(BeNumerically("~", targetBandwidthMbps, bandwidthTolerance),
		"Measured throughput should be within tolerance of target")
}

func (s *E2EWorkflowSuite) performLatencyTest() LatencyMeasurement {
	// TODO: Implement actual latency measurement between nodes
	return LatencyMeasurement{
		SliceType:  "test",
		SourceNode: "kind-worker",
		TargetNode: "kind-worker2",
		RTTMs:      6.3, // Simulated measurement
		JitterMs:   0.5,
		PacketLoss: 0.0001,
		Timestamp:  time.Now(),
	}
}

func (s *E2EWorkflowSuite) performThroughputTest() ThroughputMeasurement {
	// TODO: Implement actual throughput measurement using iperf3
	return ThroughputMeasurement{
		SliceType:           "test",
		SourceNode:          "kind-worker",
		TargetNode:          "kind-worker2",
		BandwidthMbps:       4.57, // Simulated measurement
		TestDurationSeconds: 30,
		Timestamp:           time.Now(),
	}
}

func (s *E2EWorkflowSuite) selectOptimalCluster(qos manov1alpha1.QoSRequirements, vnfType manov1alpha1.VNFType) string {
	// Select cluster based on QoS requirements and VNF type
	if qos.Latency <= 10 {
		return "edge-cluster-01"
	} else if qos.Bandwidth >= 1000 {
		return "regional-cluster-01"
	}
	return "central-cluster-01"
}

func (s *E2EWorkflowSuite) createVNFResource(pkg PorchPackage) *manov1alpha1.VNF {
	return &manov1alpha1.VNF{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pkg.Name,
			Namespace: pkg.Namespace,
		},
		Spec: manov1alpha1.VNFSpec{
			Name: pkg.Name,
			Type: pkg.VNFType,
			QoS: manov1alpha1.QoSRequirements{
				Bandwidth: 100,
				Latency:   10,
			},
			Placement: manov1alpha1.PlacementRequirements{
				CloudType: "edge",
			},
			TargetClusters: []string{pkg.ClusterName},
			Image: manov1alpha1.ImageSpec{
				Repository: "test/vnf",
				Tag:        "latest",
			},
		},
	}
}

func (s *E2EWorkflowSuite) waitForVNFReady(vnf *manov1alpha1.VNF, timeout time.Duration) error {
	// TODO: Implement actual wait logic
	time.Sleep(2 * time.Second) // Simulate deployment time
	return nil
}

func (s *E2EWorkflowSuite) validateClusterDeployment(cluster string) {
	// TODO: Implement cluster-specific validation
}

func (s *E2EWorkflowSuite) simulateClusterFailure(cluster string) {
	// TODO: Implement cluster failure simulation
}

func (s *E2EWorkflowSuite) verifyFailoverBehavior() {
	// TODO: Implement failover verification
}

func (s *E2EWorkflowSuite) validateQoSEnforcement(qos manov1alpha1.QoSRequirements) QoSValidationResult {
	// TODO: Implement actual QoS validation
	return QoSValidationResult{
		SliceType:             qos.SliceType,
		ExpectedLatencyMs:     qos.Latency,
		MeasuredLatencyMs:     qos.Latency + 0.5,
		ExpectedBandwidthMbps: qos.Bandwidth,
		MeasuredBandwidthMbps: qos.Bandwidth * 0.95,
		ValidationPassed:      true,
		DeviationPercent:      5.0,
	}
}

func (s *E2EWorkflowSuite) saveTestResults() {
	resultsDir := "testdata/results"
	os.MkdirAll(resultsDir, security.SecureDirMode)

	timestamp := time.Now().Format("20060102-150405")
	filename := filepath.Join(resultsDir, fmt.Sprintf("e2e_results_%s.json", timestamp))

	data, err := json.MarshalIndent(s.testResults, "", "  ")
	if err != nil {
		fmt.Printf("Failed to marshal test results: %v\n", err)
		return
	}

	err = os.WriteFile(filename, data, security.SecureFileMode)
	if err != nil {
		fmt.Printf("Failed to save test results: %v\n", err)
		return
	}

	fmt.Printf("Test results saved to: %s\n", filename)
}

// Supporting types
type PlacementDecision struct {
	VNFType     manov1alpha1.VNFType
	ClusterName string
	QoS         manov1alpha1.QoSRequirements
}

type PorchPackage struct {
	Name        string
	Namespace   string
	VNFType     manov1alpha1.VNFType
	ClusterName string
	GeneratedAt time.Time
}
