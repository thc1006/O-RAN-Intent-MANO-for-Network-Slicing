package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// IntegrationTestSuite provides a test suite for orchestrator-VNF operator integration
type OrchestratorVNFIntegrationSuite struct {
	suite.Suite
	orchestratorServer *httptest.Server
	vnfOperatorServer  *httptest.Server
	ctx                context.Context
	cancel             context.CancelFunc
}

// Test data structures
type SliceRequest struct {
	SliceID     string                 `json:"sliceId"`
	SliceType   string                 `json:"sliceType"`
	Intent      map[string]interface{} `json:"intent"`
	Resources   ResourceSpec           `json:"resources"`
	Placement   PlacementSpec          `json:"placement"`
	QoS         QoSSpec               `json:"qos"`
}

type ResourceSpec struct {
	CPU       string            `json:"cpu"`
	Memory    string            `json:"memory"`
	Storage   string            `json:"storage"`
	Network   NetworkSpec       `json:"network"`
	VNFs      []VNFSpec         `json:"vnfs"`
	Metadata  map[string]string `json:"metadata"`
}

type NetworkSpec struct {
	Bandwidth string   `json:"bandwidth"`
	Latency   string   `json:"latency"`
	VLANs     []string `json:"vlans"`
}

type VNFSpec struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Version   string            `json:"version"`
	Image     string            `json:"image"`
	Resources ResourceRequests  `json:"resources"`
	Config    map[string]string `json:"config"`
}

type ResourceRequests struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

type PlacementSpec struct {
	Strategy   string            `json:"strategy"`
	Zones      []string          `json:"zones"`
	Affinity   map[string]string `json:"affinity"`
	AntiAffinity []string        `json:"antiAffinity"`
}

type QoSSpec struct {
	Throughput string `json:"throughput"`
	Latency    string `json:"latency"`
	Jitter     string `json:"jitter"`
	Priority   string `json:"priority"`
}

type DeploymentResponse struct {
	SliceID      string    `json:"sliceId"`
	Status       string    `json:"status"`
	Message      string    `json:"message"`
	DeploymentID string    `json:"deploymentId"`
	Timestamp    time.Time `json:"timestamp"`
	Resources    []string  `json:"resources"`
}

type VNFDeploymentRequest struct {
	SliceID    string            `json:"sliceId"`
	VNFs       []VNFSpec         `json:"vnfs"`
	Placement  PlacementSpec     `json:"placement"`
	Config     map[string]string `json:"config"`
}

type VNFDeploymentResponse struct {
	DeploymentID string              `json:"deploymentId"`
	Status       string              `json:"status"`
	VNFStatuses  []VNFStatus         `json:"vnfStatuses"`
	Endpoints    map[string]string   `json:"endpoints"`
	Metadata     map[string]interface{} `json:"metadata"`
}

type VNFStatus struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Ready     bool      `json:"ready"`
	Replicas  int       `json:"replicas"`
	Timestamp time.Time `json:"timestamp"`
}

// Setup and teardown
func (suite *OrchestratorVNFIntegrationSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())

	// Setup mock orchestrator server
	suite.orchestratorServer = httptest.NewServer(http.HandlerFunc(suite.orchestratorHandler))

	// Setup mock VNF operator server
	suite.vnfOperatorServer = httptest.NewServer(http.HandlerFunc(suite.vnfOperatorHandler))
}

func (suite *OrchestratorVNFIntegrationSuite) TearDownSuite() {
	suite.cancel()
	if suite.orchestratorServer != nil {
		suite.orchestratorServer.Close()
	}
	if suite.vnfOperatorServer != nil {
		suite.vnfOperatorServer.Close()
	}
}

// Mock server handlers
func (suite *OrchestratorVNFIntegrationSuite) orchestratorHandler(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "POST" && strings.Contains(r.URL.Path, "/slices"):
		suite.handleSliceCreation(w, r)
	case r.Method == "GET" && strings.Contains(r.URL.Path, "/slices"):
		suite.handleSliceStatus(w, r)
	case r.Method == "DELETE" && strings.Contains(r.URL.Path, "/slices"):
		suite.handleSliceDeletion(w, r)
	case r.Method == "GET" && strings.Contains(r.URL.Path, "/health"):
		suite.handleHealth(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (suite *OrchestratorVNFIntegrationSuite) vnfOperatorHandler(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "POST" && strings.Contains(r.URL.Path, "/vnfs/deploy"):
		suite.handleVNFDeployment(w, r)
	case r.Method == "GET" && strings.Contains(r.URL.Path, "/vnfs/status"):
		suite.handleVNFStatus(w, r)
	case r.Method == "DELETE" && strings.Contains(r.URL.Path, "/vnfs"):
		suite.handleVNFUndeploy(w, r)
	case r.Method == "GET" && strings.Contains(r.URL.Path, "/health"):
		suite.handleHealth(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (suite *OrchestratorVNFIntegrationSuite) handleSliceCreation(w http.ResponseWriter, r *http.Request) {
	var req SliceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Simulate slice deployment process
	response := DeploymentResponse{
		SliceID:      req.SliceID,
		Status:       "deploying",
		Message:      "Slice deployment initiated",
		DeploymentID: fmt.Sprintf("deploy-%s", req.SliceID),
		Timestamp:    time.Now(),
		Resources:    []string{"orchestrator", "vnf-operator"},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (suite *OrchestratorVNFIntegrationSuite) handleSliceStatus(w http.ResponseWriter, r *http.Request) {
	sliceID := strings.TrimPrefix(r.URL.Path, "/slices/")

	response := DeploymentResponse{
		SliceID:      sliceID,
		Status:       "active",
		Message:      "Slice is operational",
		DeploymentID: fmt.Sprintf("deploy-%s", sliceID),
		Timestamp:    time.Now(),
		Resources:    []string{"vnf-amf", "vnf-smf", "vnf-upf"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (suite *OrchestratorVNFIntegrationSuite) handleSliceDeletion(w http.ResponseWriter, r *http.Request) {
	sliceID := strings.TrimPrefix(r.URL.Path, "/slices/")

	response := map[string]interface{}{
		"sliceId":   sliceID,
		"status":    "terminating",
		"message":   "Slice termination initiated",
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (suite *OrchestratorVNFIntegrationSuite) handleVNFDeployment(w http.ResponseWriter, r *http.Request) {
	var req VNFDeploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Simulate VNF deployment
	var vnfStatuses []VNFStatus
	for _, vnf := range req.VNFs {
		vnfStatuses = append(vnfStatuses, VNFStatus{
			Name:      vnf.Name,
			Status:    "deploying",
			Ready:     false,
			Replicas:  1,
			Timestamp: time.Now(),
		})
	}

	response := VNFDeploymentResponse{
		DeploymentID: fmt.Sprintf("vnf-deploy-%s", req.SliceID),
		Status:       "deploying",
		VNFStatuses:  vnfStatuses,
		Endpoints:    map[string]string{},
		Metadata:     map[string]interface{}{"sliceId": req.SliceID},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (suite *OrchestratorVNFIntegrationSuite) handleVNFStatus(w http.ResponseWriter, r *http.Request) {
	deploymentID := r.URL.Query().Get("deploymentId")

	vnfStatuses := []VNFStatus{
		{Name: "amf", Status: "running", Ready: true, Replicas: 1, Timestamp: time.Now()},
		{Name: "smf", Status: "running", Ready: true, Replicas: 1, Timestamp: time.Now()},
		{Name: "upf", Status: "running", Ready: true, Replicas: 2, Timestamp: time.Now()},
	}

	response := VNFDeploymentResponse{
		DeploymentID: deploymentID,
		Status:       "running",
		VNFStatuses:  vnfStatuses,
		Endpoints: map[string]string{
			"amf": "amf.5g.local:8080",
			"smf": "smf.5g.local:8080",
			"upf": "upf.5g.local:8080",
		},
		Metadata: map[string]interface{}{"ready": true},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (suite *OrchestratorVNFIntegrationSuite) handleVNFUndeploy(w http.ResponseWriter, r *http.Request) {
	deploymentID := strings.TrimPrefix(r.URL.Path, "/vnfs/")

	response := map[string]interface{}{
		"deploymentId": deploymentID,
		"status":       "terminating",
		"message":      "VNF undeployment initiated",
		"timestamp":    time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (suite *OrchestratorVNFIntegrationSuite) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   "v1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Integration tests

func (suite *OrchestratorVNFIntegrationSuite) TestSliceLifecycleIntegration() {
	// Test complete slice lifecycle: create -> deploy VNFs -> activate -> terminate

	sliceRequest := SliceRequest{
		SliceID:   "integration-slice-001",
		SliceType: "eMBB",
		Intent: map[string]interface{}{
			"bandwidth": "100Mbps",
			"latency":   "10ms",
		},
		Resources: ResourceSpec{
			CPU:     "4",
			Memory:  "8Gi",
			Storage: "20Gi",
			Network: NetworkSpec{
				Bandwidth: "1Gbps",
				Latency:   "5ms",
				VLANs:     []string{"100", "200"},
			},
			VNFs: []VNFSpec{
				{Name: "amf", Type: "core", Version: "v1.0", Image: "5g/amf:latest"},
				{Name: "smf", Type: "core", Version: "v1.0", Image: "5g/smf:latest"},
				{Name: "upf", Type: "user-plane", Version: "v1.0", Image: "5g/upf:latest"},
			},
		},
		Placement: PlacementSpec{
			Strategy: "spread",
			Zones:    []string{"zone-a", "zone-b"},
		},
		QoS: QoSSpec{
			Throughput: "100Mbps",
			Latency:    "10ms",
			Priority:   "high",
		},
	}

	// Step 1: Create slice via orchestrator
	sliceResponse := suite.createSlice(sliceRequest)
	suite.Assert().Equal("deploying", sliceResponse.Status)
	suite.Assert().Equal(sliceRequest.SliceID, sliceResponse.SliceID)

	// Step 2: Deploy VNFs via VNF operator
	vnfRequest := VNFDeploymentRequest{
		SliceID:   sliceRequest.SliceID,
		VNFs:      sliceRequest.Resources.VNFs,
		Placement: sliceRequest.Placement,
		Config:    map[string]string{"slice_type": sliceRequest.SliceType},
	}

	vnfResponse := suite.deployVNFs(vnfRequest)
	suite.Assert().Equal("deploying", vnfResponse.Status)
	suite.Assert().Len(vnfResponse.VNFStatuses, 3)

	// Step 3: Wait for VNFs to become ready
	suite.waitForVNFsReady(vnfResponse.DeploymentID, 30*time.Second)

	// Step 4: Verify slice is active
	sliceStatus := suite.getSliceStatus(sliceRequest.SliceID)
	suite.Assert().Equal("active", sliceStatus.Status)

	// Step 5: Terminate slice
	suite.terminateSlice(sliceRequest.SliceID)
}

func (suite *OrchestratorVNFIntegrationSuite) TestVNFScalingIntegration() {
	sliceID := "scaling-slice-001"

	// Initial VNF deployment
	vnfRequest := VNFDeploymentRequest{
		SliceID: sliceID,
		VNFs: []VNFSpec{
			{Name: "upf", Type: "user-plane", Version: "v1.0", Image: "5g/upf:latest"},
		},
		Placement: PlacementSpec{Strategy: "spread"},
	}

	vnfResponse := suite.deployVNFs(vnfRequest)
	suite.Assert().Equal("deploying", vnfResponse.Status)

	// Wait for initial deployment
	suite.waitForVNFsReady(vnfResponse.DeploymentID, 30*time.Second)

	// Verify initial status
	status := suite.getVNFStatus(vnfResponse.DeploymentID)
	upfStatus := suite.findVNFByName(status.VNFStatuses, "upf")
	suite.Assert().NotNil(upfStatus)
	suite.Assert().True(upfStatus.Ready)

	// Note: Actual scaling would require additional API endpoints
	// This test verifies the integration path for scale operations
}

func (suite *OrchestratorVNFIntegrationSuite) TestMultiSliceConcurrentDeployment() {
	const numSlices = 5
	var sliceIDs []string

	// Create multiple slices concurrently
	for i := 0; i < numSlices; i++ {
		sliceID := fmt.Sprintf("concurrent-slice-%03d", i)
		sliceIDs = append(sliceIDs, sliceID)

		go func(id string) {
			sliceRequest := SliceRequest{
				SliceID:   id,
				SliceType: "eMBB",
				Intent: map[string]interface{}{
					"bandwidth": "50Mbps",
				},
				Resources: ResourceSpec{
					VNFs: []VNFSpec{
						{Name: "amf", Type: "core", Version: "v1.0", Image: "5g/amf:latest"},
					},
				},
			}

			suite.createSlice(sliceRequest)
		}(sliceID)
	}

	// Wait and verify all slices
	time.Sleep(2 * time.Second)

	for _, sliceID := range sliceIDs {
		status := suite.getSliceStatus(sliceID)
		suite.Assert().Contains([]string{"deploying", "active"}, status.Status)
	}
}

func (suite *OrchestratorVNFIntegrationSuite) TestErrorHandlingIntegration() {
	// Test error propagation between orchestrator and VNF operator

	// Invalid slice request
	invalidRequest := SliceRequest{
		SliceID: "", // Invalid empty slice ID
		Resources: ResourceSpec{
			VNFs: []VNFSpec{}, // No VNFs specified
		},
	}

	resp, err := suite.makeSliceRequest(invalidRequest)
	if err == nil {
		suite.Assert().Equal(http.StatusBadRequest, resp.StatusCode)
	}

	// Invalid VNF deployment
	invalidVNFRequest := VNFDeploymentRequest{
		SliceID: "test-slice",
		VNFs:    []VNFSpec{}, // Empty VNF list
	}

	vnfResp, err := suite.makeVNFDeploymentRequest(invalidVNFRequest)
	if err == nil {
		suite.Assert().Equal(http.StatusBadRequest, vnfResp.StatusCode)
	}
}

func (suite *OrchestratorVNFIntegrationSuite) TestResourceAllocationIntegration() {
	// Test resource allocation coordination between orchestrator and VNF operator

	highResourceSlice := SliceRequest{
		SliceID:   "high-resource-slice",
		SliceType: "URLLC",
		Resources: ResourceSpec{
			CPU:     "16",
			Memory:  "32Gi",
			Storage: "100Gi",
			VNFs: []VNFSpec{
				{
					Name:    "upf",
					Type:    "user-plane",
					Version: "v1.0",
					Image:   "5g/upf:latest",
					Resources: ResourceRequests{
						CPU:    "8",
						Memory: "16Gi",
					},
				},
			},
		},
	}

	// Create slice and verify resource allocation
	sliceResponse := suite.createSlice(highResourceSlice)
	suite.Assert().Equal("deploying", sliceResponse.Status)

	// Deploy VNFs with resource constraints
	vnfRequest := VNFDeploymentRequest{
		SliceID: highResourceSlice.SliceID,
		VNFs:    highResourceSlice.Resources.VNFs,
	}

	vnfResponse := suite.deployVNFs(vnfRequest)
	suite.Assert().Equal("deploying", vnfResponse.Status)

	// Verify deployment completes successfully
	suite.waitForVNFsReady(vnfResponse.DeploymentID, 30*time.Second)
}

func (suite *OrchestratorVNFIntegrationSuite) TestHealthMonitoringIntegration() {
	// Test health monitoring across orchestrator and VNF operator

	// Check orchestrator health
	orchHealth := suite.checkOrchestratorHealth()
	suite.Assert().Equal("healthy", orchHealth["status"])

	// Check VNF operator health
	vnfHealth := suite.checkVNFOperatorHealth()
	suite.Assert().Equal("healthy", vnfHealth["status"])

	// Verify both services are responsive
	suite.Assert().NotNil(orchHealth["timestamp"])
	suite.Assert().NotNil(vnfHealth["timestamp"])
}

// Helper methods

func (suite *OrchestratorVNFIntegrationSuite) createSlice(req SliceRequest) DeploymentResponse {
	resp, err := suite.makeSliceRequest(req)
	suite.Require().NoError(err)
	suite.Require().Equal(http.StatusCreated, resp.StatusCode)

	var response DeploymentResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.Require().NoError(err)
	resp.Body.Close()

	return response
}

func (suite *OrchestratorVNFIntegrationSuite) makeSliceRequest(req SliceRequest) (*http.Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	url := suite.orchestratorServer.URL + "/slices"
	return http.Post(url, "application/json", strings.NewReader(string(body)))
}

func (suite *OrchestratorVNFIntegrationSuite) getSliceStatus(sliceID string) DeploymentResponse {
	url := suite.orchestratorServer.URL + "/slices/" + sliceID
	resp, err := http.Get(url)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	var response DeploymentResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.Require().NoError(err)

	return response
}

func (suite *OrchestratorVNFIntegrationSuite) terminateSlice(sliceID string) {
	url := suite.orchestratorServer.URL + "/slices/" + sliceID
	req, err := http.NewRequest("DELETE", url, nil)
	suite.Require().NoError(err)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Assert().Equal(http.StatusOK, resp.StatusCode)
}

func (suite *OrchestratorVNFIntegrationSuite) deployVNFs(req VNFDeploymentRequest) VNFDeploymentResponse {
	resp, err := suite.makeVNFDeploymentRequest(req)
	suite.Require().NoError(err)
	suite.Require().Equal(http.StatusCreated, resp.StatusCode)

	var response VNFDeploymentResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.Require().NoError(err)
	resp.Body.Close()

	return response
}

func (suite *OrchestratorVNFIntegrationSuite) makeVNFDeploymentRequest(req VNFDeploymentRequest) (*http.Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	url := suite.vnfOperatorServer.URL + "/vnfs/deploy"
	return http.Post(url, "application/json", strings.NewReader(string(body)))
}

func (suite *OrchestratorVNFIntegrationSuite) getVNFStatus(deploymentID string) VNFDeploymentResponse {
	url := suite.vnfOperatorServer.URL + "/vnfs/status?deploymentId=" + deploymentID
	resp, err := http.Get(url)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	var response VNFDeploymentResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.Require().NoError(err)

	return response
}

func (suite *OrchestratorVNFIntegrationSuite) waitForVNFsReady(deploymentID string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		status := suite.getVNFStatus(deploymentID)

		allReady := true
		for _, vnfStatus := range status.VNFStatuses {
			if !vnfStatus.Ready {
				allReady = false
				break
			}
		}

		if allReady {
			return
		}

		time.Sleep(1 * time.Second)
	}

	suite.Fail("VNFs did not become ready within timeout")
}

func (suite *OrchestratorVNFIntegrationSuite) findVNFByName(statuses []VNFStatus, name string) *VNFStatus {
	for _, status := range statuses {
		if status.Name == name {
			return &status
		}
	}
	return nil
}

func (suite *OrchestratorVNFIntegrationSuite) checkOrchestratorHealth() map[string]interface{} {
	url := suite.orchestratorServer.URL + "/health"
	resp, err := http.Get(url)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	var health map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&health)
	suite.Require().NoError(err)

	return health
}

func (suite *OrchestratorVNFIntegrationSuite) checkVNFOperatorHealth() map[string]interface{} {
	url := suite.vnfOperatorServer.URL + "/health"
	resp, err := http.Get(url)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	var health map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&health)
	suite.Require().NoError(err)

	return health
}

// Test runner
func TestOrchestratorVNFIntegrationSuite(t *testing.T) {
	suite.Run(t, new(OrchestratorVNFIntegrationSuite))
}

// Benchmark tests for integration performance

func BenchmarkSliceDeploymentIntegration(b *testing.B) {
	suite := &OrchestratorVNFIntegrationSuite{}
	suite.SetupSuite()
	defer suite.TearDownSuite()

	sliceRequest := SliceRequest{
		SliceID:   "bench-slice",
		SliceType: "eMBB",
		Resources: ResourceSpec{
			VNFs: []VNFSpec{
				{Name: "amf", Type: "core", Version: "v1.0", Image: "5g/amf:latest"},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sliceRequest.SliceID = fmt.Sprintf("bench-slice-%d", i)
		suite.createSlice(sliceRequest)
	}
}

func BenchmarkVNFDeploymentIntegration(b *testing.B) {
	suite := &OrchestratorVNFIntegrationSuite{}
	suite.SetupSuite()
	defer suite.TearDownSuite()

	vnfRequest := VNFDeploymentRequest{
		SliceID: "bench-vnf-slice",
		VNFs: []VNFSpec{
			{Name: "upf", Type: "user-plane", Version: "v1.0", Image: "5g/upf:latest"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vnfRequest.SliceID = fmt.Sprintf("bench-vnf-slice-%d", i)
		suite.deployVNFs(vnfRequest)
	}
}

// Test data factory functions

func createTestSliceRequest(sliceID string) SliceRequest {
	return SliceRequest{
		SliceID:   sliceID,
		SliceType: "eMBB",
		Intent: map[string]interface{}{
			"bandwidth": "100Mbps",
			"latency":   "10ms",
		},
		Resources: ResourceSpec{
			CPU:     "4",
			Memory:  "8Gi",
			Storage: "20Gi",
			VNFs: []VNFSpec{
				{Name: "amf", Type: "core", Version: "v1.0", Image: "5g/amf:latest"},
				{Name: "smf", Type: "core", Version: "v1.0", Image: "5g/smf:latest"},
			},
		},
		Placement: PlacementSpec{
			Strategy: "spread",
			Zones:    []string{"zone-a", "zone-b"},
		},
		QoS: QoSSpec{
			Throughput: "100Mbps",
			Latency:    "10ms",
			Priority:   "high",
		},
	}
}

func createTestVNFDeploymentRequest(sliceID string) VNFDeploymentRequest {
	return VNFDeploymentRequest{
		SliceID: sliceID,
		VNFs: []VNFSpec{
			{Name: "amf", Type: "core", Version: "v1.0", Image: "5g/amf:latest"},
			{Name: "smf", Type: "core", Version: "v1.0", Image: "5g/smf:latest"},
		},
		Placement: PlacementSpec{Strategy: "spread"},
		Config:    map[string]string{"slice_type": "eMBB"},
	}
}