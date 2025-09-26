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

// VNFDMSIntegrationSuite tests integration between VNF Operator and DMS components
type VNFDMSIntegrationSuite struct {
	suite.Suite
	vnfOperatorServer *httptest.Server
	ranDMSServer      *httptest.Server
	cnDMSServer       *httptest.Server
	ctx               context.Context
	cancel            context.CancelFunc
}

// Test data structures for VNF-DMS integration

type VNFLifecycleRequest struct {
	VNFName     string                 `json:"vnfName"`
	VNFType     string                 `json:"vnfType"`
	SliceID     string                 `json:"sliceId"`
	DMSTargets  []DMSTarget           `json:"dmsTargets"`
	Resources   VNFResourceSpec       `json:"resources"`
	Config      map[string]interface{} `json:"config"`
}

type DMSTarget struct {
	Type     string `json:"type"`     // "RAN" or "CN"
	Endpoint string `json:"endpoint"`
	Zone     string `json:"zone"`
}

type VNFResourceSpec struct {
	CPU       string            `json:"cpu"`
	Memory    string            `json:"memory"`
	Storage   string            `json:"storage"`
	Networks  []NetworkBinding  `json:"networks"`
	Policies  []SecurityPolicy  `json:"policies"`
}

type NetworkBinding struct {
	Name      string `json:"name"`
	Interface string `json:"interface"`
	VLAN      string `json:"vlan"`
	Bandwidth string `json:"bandwidth"`
}

type SecurityPolicy struct {
	Type   string            `json:"type"`
	Rules  []string          `json:"rules"`
	Config map[string]string `json:"config"`
}

type DMSRegistrationRequest struct {
	VNFName      string                 `json:"vnfName"`
	VNFEndpoint  string                 `json:"vnfEndpoint"`
	SliceID      string                 `json:"sliceId"`
	Capabilities []string               `json:"capabilities"`
	Metadata     map[string]interface{} `json:"metadata"`
}

type DMSRegistrationResponse struct {
	RegistrationID string    `json:"registrationId"`
	Status         string    `json:"status"`
	DMSEndpoints   []string  `json:"dmsEndpoints"`
	ConfigUpdate   bool      `json:"configUpdate"`
	Timestamp      time.Time `json:"timestamp"`
}

type VNFStatusUpdate struct {
	VNFName     string                 `json:"vnfName"`
	SliceID     string                 `json:"sliceId"`
	Status      string                 `json:"status"`
	Endpoints   []string               `json:"endpoints"`
	Metrics     map[string]interface{} `json:"metrics"`
	Timestamp   time.Time              `json:"timestamp"`
}

type DMSSliceConfiguration struct {
	SliceID       string                 `json:"sliceId"`
	SliceType     string                 `json:"sliceType"`
	VNFInstances  []VNFInstanceConfig    `json:"vnfInstances"`
	NetworkPolicy NetworkPolicyConfig    `json:"networkPolicy"`
	QoSParameters QoSParametersConfig    `json:"qosParameters"`
}

type VNFInstanceConfig struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Endpoints    []string          `json:"endpoints"`
	Config       map[string]string `json:"config"`
	Dependencies []string          `json:"dependencies"`
}

type NetworkPolicyConfig struct {
	BandwidthLimits map[string]string `json:"bandwidthLimits"`
	TrafficShaping  bool              `json:"trafficShaping"`
	QueueManagement string            `json:"queueManagement"`
}

type QoSParametersConfig struct {
	Latency      string `json:"latency"`
	Throughput   string `json:"throughput"`
	Reliability  string `json:"reliability"`
	Availability string `json:"availability"`
}

// Setup and teardown
func (suite *VNFDMSIntegrationSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())

	// Setup mock VNF Operator server
	suite.vnfOperatorServer = httptest.NewServer(http.HandlerFunc(suite.vnfOperatorHandler))

	// Setup mock RAN DMS server
	suite.ranDMSServer = httptest.NewServer(http.HandlerFunc(suite.ranDMSHandler))

	// Setup mock CN DMS server
	suite.cnDMSServer = httptest.NewServer(http.HandlerFunc(suite.cnDMSHandler))
}

func (suite *VNFDMSIntegrationSuite) TearDownSuite() {
	suite.cancel()
	if suite.vnfOperatorServer != nil {
		suite.vnfOperatorServer.Close()
	}
	if suite.ranDMSServer != nil {
		suite.ranDMSServer.Close()
	}
	if suite.cnDMSServer != nil {
		suite.cnDMSServer.Close()
	}
}

// Mock server handlers
func (suite *VNFDMSIntegrationSuite) vnfOperatorHandler(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "POST" && strings.Contains(r.URL.Path, "/vnf/lifecycle"):
		suite.handleVNFLifecycle(w, r)
	case r.Method == "POST" && strings.Contains(r.URL.Path, "/vnf/register"):
		suite.handleVNFRegistration(w, r)
	case r.Method == "PUT" && strings.Contains(r.URL.Path, "/vnf/status"):
		suite.handleVNFStatusUpdate(w, r)
	case r.Method == "GET" && strings.Contains(r.URL.Path, "/vnf/instances"):
		suite.handleVNFInstancesList(w, r)
	case r.Method == "GET" && strings.Contains(r.URL.Path, "/health"):
		suite.handleHealth(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (suite *VNFDMSIntegrationSuite) ranDMSHandler(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "POST" && strings.Contains(r.URL.Path, "/register"):
		suite.handleDMSRegistration(w, r, "RAN")
	case r.Method == "POST" && strings.Contains(r.URL.Path, "/configure"):
		suite.handleDMSConfiguration(w, r, "RAN")
	case r.Method == "GET" && strings.Contains(r.URL.Path, "/status"):
		suite.handleDMSStatus(w, r, "RAN")
	case r.Method == "PUT" && strings.Contains(r.URL.Path, "/vnf/update"):
		suite.handleDMSVNFUpdate(w, r, "RAN")
	case r.Method == "GET" && strings.Contains(r.URL.Path, "/health"):
		suite.handleHealth(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (suite *VNFDMSIntegrationSuite) cnDMSHandler(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "POST" && strings.Contains(r.URL.Path, "/register"):
		suite.handleDMSRegistration(w, r, "CN")
	case r.Method == "POST" && strings.Contains(r.URL.Path, "/configure"):
		suite.handleDMSConfiguration(w, r, "CN")
	case r.Method == "GET" && strings.Contains(r.URL.Path, "/status"):
		suite.handleDMSStatus(w, r, "CN")
	case r.Method == "PUT" && strings.Contains(r.URL.Path, "/vnf/update"):
		suite.handleDMSVNFUpdate(w, r, "CN")
	case r.Method == "GET" && strings.Contains(r.URL.Path, "/health"):
		suite.handleHealth(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (suite *VNFDMSIntegrationSuite) handleVNFLifecycle(w http.ResponseWriter, r *http.Request) {
	var req VNFLifecycleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"vnfName":     req.VNFName,
		"sliceId":     req.SliceID,
		"status":      "deploying",
		"message":     fmt.Sprintf("VNF %s deployment initiated", req.VNFName),
		"dmsTargets":  len(req.DMSTargets),
		"timestamp":   time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (suite *VNFDMSIntegrationSuite) handleVNFRegistration(w http.ResponseWriter, r *http.Request) {
	var req DMSRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response := DMSRegistrationResponse{
		RegistrationID: fmt.Sprintf("reg-%s-%d", req.VNFName, time.Now().Unix()),
		Status:         "registered",
		DMSEndpoints: []string{
			suite.ranDMSServer.URL,
			suite.cnDMSServer.URL,
		},
		ConfigUpdate: true,
		Timestamp:    time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (suite *VNFDMSIntegrationSuite) handleVNFStatusUpdate(w http.ResponseWriter, r *http.Request) {
	var req VNFStatusUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"vnfName":   req.VNFName,
		"sliceId":   req.SliceID,
		"status":    "updated",
		"message":   "VNF status updated successfully",
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (suite *VNFDMSIntegrationSuite) handleVNFInstancesList(w http.ResponseWriter, r *http.Request) {
	sliceID := r.URL.Query().Get("sliceId")

	instances := []VNFInstanceConfig{
		{
			Name:         "amf-001",
			Type:         "AMF",
			Endpoints:    []string{"amf.5g.local:8080"},
			Config:       map[string]string{"slice_id": sliceID},
			Dependencies: []string{"smf-001"},
		},
		{
			Name:         "smf-001",
			Type:         "SMF",
			Endpoints:    []string{"smf.5g.local:8080"},
			Config:       map[string]string{"slice_id": sliceID},
			Dependencies: []string{"upf-001"},
		},
		{
			Name:         "upf-001",
			Type:         "UPF",
			Endpoints:    []string{"upf.5g.local:8080"},
			Config:       map[string]string{"slice_id": sliceID},
			Dependencies: []string{},
		},
	}

	response := map[string]interface{}{
		"sliceId":   sliceID,
		"instances": instances,
		"count":     len(instances),
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (suite *VNFDMSIntegrationSuite) handleDMSRegistration(w http.ResponseWriter, r *http.Request, dmsType string) {
	var req DMSRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response := DMSRegistrationResponse{
		RegistrationID: fmt.Sprintf("%s-reg-%s", strings.ToLower(dmsType), req.VNFName),
		Status:         "registered",
		DMSEndpoints:   []string{fmt.Sprintf("%s-dms.local:8080", strings.ToLower(dmsType))},
		ConfigUpdate:   true,
		Timestamp:      time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (suite *VNFDMSIntegrationSuite) handleDMSConfiguration(w http.ResponseWriter, r *http.Request, dmsType string) {
	var req DMSSliceConfiguration
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"dmsType":    dmsType,
		"sliceId":    req.SliceID,
		"status":     "configured",
		"message":    fmt.Sprintf("%s DMS configured for slice %s", dmsType, req.SliceID),
		"vnfCount":   len(req.VNFInstances),
		"timestamp":  time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (suite *VNFDMSIntegrationSuite) handleDMSStatus(w http.ResponseWriter, r *http.Request, dmsType string) {
	sliceID := r.URL.Query().Get("sliceId")

	response := map[string]interface{}{
		"dmsType":     dmsType,
		"sliceId":     sliceID,
		"status":      "operational",
		"activeVNFs":  3,
		"performance": map[string]interface{}{
			"throughput": "95Mbps",
			"latency":    "5ms",
			"uptime":     "99.9%",
		},
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (suite *VNFDMSIntegrationSuite) handleDMSVNFUpdate(w http.ResponseWriter, r *http.Request, dmsType string) {
	var req VNFStatusUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"dmsType":   dmsType,
		"vnfName":   req.VNFName,
		"sliceId":   req.SliceID,
		"status":    "updated",
		"message":   fmt.Sprintf("VNF %s updated in %s DMS", req.VNFName, dmsType),
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (suite *VNFDMSIntegrationSuite) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   "v1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Integration tests

func (suite *VNFDMSIntegrationSuite) TestVNFDMSRegistrationIntegration() {
	// Test VNF registration across RAN and CN DMS

	registrationReq := DMSRegistrationRequest{
		VNFName:      "test-amf-001",
		VNFEndpoint:  "amf.5g.local:8080",
		SliceID:      "integration-slice-001",
		Capabilities: []string{"N1", "N2", "Namf"},
		Metadata: map[string]interface{}{
			"vendor":  "TestVendor",
			"version": "v1.0.0",
		},
	}

	// Register VNF with VNF Operator
	vnfResponse := suite.registerVNFWithOperator(registrationReq)
	suite.Assert().Equal("registered", vnfResponse.Status)
	suite.Assert().Contains(vnfResponse.DMSEndpoints, suite.ranDMSServer.URL)
	suite.Assert().Contains(vnfResponse.DMSEndpoints, suite.cnDMSServer.URL)

	// Register VNF with RAN DMS
	ranResponse := suite.registerVNFWithDMS(registrationReq, "RAN")
	suite.Assert().Equal("registered", ranResponse.Status)
	suite.Assert().Contains(ranResponse.RegistrationID, "ran-reg")

	// Register VNF with CN DMS
	cnResponse := suite.registerVNFWithDMS(registrationReq, "CN")
	suite.Assert().Equal("registered", cnResponse.Status)
	suite.Assert().Contains(cnResponse.RegistrationID, "cn-reg")
}

func (suite *VNFDMSIntegrationSuite) TestSliceConfigurationPropagation() {
	// Test slice configuration propagation from VNF Operator to DMS components

	sliceConfig := DMSSliceConfiguration{
		SliceID:   "config-slice-001",
		SliceType: "eMBB",
		VNFInstances: []VNFInstanceConfig{
			{
				Name:         "amf-001",
				Type:         "AMF",
				Endpoints:    []string{"amf.5g.local:8080"},
				Config:       map[string]string{"plmn": "001-01"},
				Dependencies: []string{"smf-001"},
			},
			{
				Name:         "upf-001",
				Type:         "UPF",
				Endpoints:    []string{"upf.5g.local:8080"},
				Config:       map[string]string{"dnn": "internet"},
				Dependencies: []string{},
			},
		},
		NetworkPolicy: NetworkPolicyConfig{
			BandwidthLimits: map[string]string{
				"uplink":   "100Mbps",
				"downlink": "150Mbps",
			},
			TrafficShaping:  true,
			QueueManagement: "weighted-fair",
		},
		QoSParameters: QoSParametersConfig{
			Latency:      "10ms",
			Throughput:   "100Mbps",
			Reliability:  "99.99%",
			Availability: "99.9%",
		},
	}

	// Configure RAN DMS
	ranConfigResp := suite.configureDMS(sliceConfig, "RAN")
	suite.Assert().Equal("configured", ranConfigResp["status"])
	suite.Assert().Equal("RAN", ranConfigResp["dmsType"])

	// Configure CN DMS
	cnConfigResp := suite.configureDMS(sliceConfig, "CN")
	suite.Assert().Equal("configured", cnConfigResp["status"])
	suite.Assert().Equal("CN", cnConfigResp["dmsType"])

	// Verify both DMS components have the configuration
	suite.Assert().Equal(sliceConfig.SliceID, ranConfigResp["sliceId"])
	suite.Assert().Equal(sliceConfig.SliceID, cnConfigResp["sliceId"])
}

func (suite *VNFDMSIntegrationSuite) TestVNFLifecycleManagement() {
	// Test complete VNF lifecycle with DMS coordination

	lifecycleReq := VNFLifecycleRequest{
		VNFName: "lifecycle-vnf-001",
		VNFType: "UPF",
		SliceID: "lifecycle-slice-001",
		DMSTargets: []DMSTarget{
			{Type: "RAN", Endpoint: suite.ranDMSServer.URL, Zone: "zone-a"},
			{Type: "CN", Endpoint: suite.cnDMSServer.URL, Zone: "zone-b"},
		},
		Resources: VNFResourceSpec{
			CPU:     "4",
			Memory:  "8Gi",
			Storage: "20Gi",
			Networks: []NetworkBinding{
				{Name: "n3", Interface: "eth0", VLAN: "100", Bandwidth: "1Gbps"},
				{Name: "n4", Interface: "eth1", VLAN: "200", Bandwidth: "500Mbps"},
			},
			Policies: []SecurityPolicy{
				{Type: "firewall", Rules: []string{"allow-n3", "allow-n4"}},
			},
		},
		Config: map[string]interface{}{
			"dnn":     "internet",
			"s_nssai": "1-000001",
		},
	}

	// Initiate VNF lifecycle via VNF Operator
	lifecycleResp := suite.initiateVNFLifecycle(lifecycleReq)
	suite.Assert().Equal("deploying", lifecycleResp["status"])
	suite.Assert().Equal(lifecycleReq.VNFName, lifecycleResp["vnfName"])
	suite.Assert().Equal(len(lifecycleReq.DMSTargets), lifecycleResp["dmsTargets"])

	// Update VNF status
	statusUpdate := VNFStatusUpdate{
		VNFName:   lifecycleReq.VNFName,
		SliceID:   lifecycleReq.SliceID,
		Status:    "running",
		Endpoints: []string{"upf.5g.local:8080"},
		Metrics: map[string]interface{}{
			"cpu_usage":    "45%",
			"memory_usage": "60%",
			"throughput":   "85Mbps",
		},
		Timestamp: time.Now(),
	}

	// Update status in VNF Operator
	updateResp := suite.updateVNFStatus(statusUpdate)
	suite.Assert().Equal("updated", updateResp["status"])

	// Propagate status to DMS components
	ranUpdateResp := suite.updateVNFInDMS(statusUpdate, "RAN")
	suite.Assert().Equal("updated", ranUpdateResp["status"])
	suite.Assert().Equal("RAN", ranUpdateResp["dmsType"])

	cnUpdateResp := suite.updateVNFInDMS(statusUpdate, "CN")
	suite.Assert().Equal("updated", cnUpdateResp["status"])
	suite.Assert().Equal("CN", cnUpdateResp["dmsType"])
}

func (suite *VNFDMSIntegrationSuite) TestVNFScalingCoordination() {
	// Test VNF scaling coordination between VNF Operator and DMS

	sliceID := "scaling-slice-001"

	// Get initial VNF instances
	initialInstances := suite.getVNFInstances(sliceID)
	suite.Assert().Equal(3, initialInstances["count"])

	// Simulate scaling event
	scalingReq := VNFStatusUpdate{
		VNFName:   "upf-001",
		SliceID:   sliceID,
		Status:    "scaling",
		Endpoints: []string{"upf-001.5g.local:8080", "upf-002.5g.local:8080"},
		Metrics: map[string]interface{}{
			"replicas":    2,
			"target_load": "80%",
		},
		Timestamp: time.Now(),
	}

	// Update VNF Operator
	vnfUpdateResp := suite.updateVNFStatus(scalingReq)
	suite.Assert().Equal("updated", vnfUpdateResp["status"])

	// Update DMS components
	ranScaleResp := suite.updateVNFInDMS(scalingReq, "RAN")
	suite.Assert().Equal("updated", ranScaleResp["status"])

	cnScaleResp := suite.updateVNFInDMS(scalingReq, "CN")
	suite.Assert().Equal("updated", cnScaleResp["status"])

	// Verify scaling coordination
	suite.Assert().Equal(scalingReq.VNFName, ranScaleResp["vnfName"])
	suite.Assert().Equal(scalingReq.VNFName, cnScaleResp["vnfName"])
}

func (suite *VNFDMSIntegrationSuite) TestErrorHandlingAndRecovery() {
	// Test error handling and recovery across VNF-DMS integration

	// Invalid registration request
	invalidReq := DMSRegistrationRequest{
		VNFName: "", // Invalid empty name
		SliceID: "error-slice-001",
	}

	resp, err := suite.makeVNFRegistrationRequest(invalidReq)
	if err == nil {
		suite.Assert().Equal(http.StatusBadRequest, resp.StatusCode)
		resp.Body.Close()
	}

	// Invalid DMS configuration
	invalidConfig := DMSSliceConfiguration{
		SliceID:      "", // Invalid empty slice ID
		VNFInstances: []VNFInstanceConfig{},
	}

	ranResp, err := suite.makeDMSConfigRequest(invalidConfig, "RAN")
	if err == nil {
		suite.Assert().Equal(http.StatusBadRequest, ranResp.StatusCode)
		ranResp.Body.Close()
	}
}

func (suite *VNFDMSIntegrationSuite) TestPerformanceMonitoring() {
	// Test performance monitoring integration between VNF Operator and DMS

	sliceID := "perf-slice-001"

	// Get performance metrics from RAN DMS
	ranStatus := suite.getDMSStatus(sliceID, "RAN")
	suite.Assert().Equal("operational", ranStatus["status"])
	suite.Assert().NotNil(ranStatus["performance"])

	ranPerf := ranStatus["performance"].(map[string]interface{})
	suite.Assert().Contains(ranPerf, "throughput")
	suite.Assert().Contains(ranPerf, "latency")
	suite.Assert().Contains(ranPerf, "uptime")

	// Get performance metrics from CN DMS
	cnStatus := suite.getDMSStatus(sliceID, "CN")
	suite.Assert().Equal("operational", cnStatus["status"])
	suite.Assert().NotNil(cnStatus["performance"])

	cnPerf := cnStatus["performance"].(map[string]interface{})
	suite.Assert().Contains(cnPerf, "throughput")
	suite.Assert().Contains(cnPerf, "latency")
	suite.Assert().Contains(cnPerf, "uptime")
}

func (suite *VNFDMSIntegrationSuite) TestConcurrentVNFOperations() {
	// Test concurrent VNF operations across multiple DMS instances

	const numVNFs = 5
	sliceID := "concurrent-slice-001"

	// Deploy multiple VNFs concurrently
	for i := 0; i < numVNFs; i++ {
		go func(id int) {
			vnfReq := VNFLifecycleRequest{
				VNFName: fmt.Sprintf("concurrent-vnf-%03d", id),
				VNFType: "UPF",
				SliceID: sliceID,
				DMSTargets: []DMSTarget{
					{Type: "RAN", Endpoint: suite.ranDMSServer.URL},
					{Type: "CN", Endpoint: suite.cnDMSServer.URL},
				},
			}

			suite.initiateVNFLifecycle(vnfReq)
		}(i)
	}

	// Wait for operations to complete
	time.Sleep(2 * time.Second)

	// Verify VNF instances
	instances := suite.getVNFInstances(sliceID)
	suite.Assert().GreaterOrEqual(instances["count"], 3) // At least the default instances
}

// Helper methods

func (suite *VNFDMSIntegrationSuite) registerVNFWithOperator(req DMSRegistrationRequest) DMSRegistrationResponse {
	resp, err := suite.makeVNFRegistrationRequest(req)
	suite.Require().NoError(err)
	defer resp.Body.Close()
	suite.Require().Equal(http.StatusCreated, resp.StatusCode)

	var response DMSRegistrationResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.Require().NoError(err)

	return response
}

func (suite *VNFDMSIntegrationSuite) makeVNFRegistrationRequest(req DMSRegistrationRequest) (*http.Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	url := suite.vnfOperatorServer.URL + "/vnf/register"
	return http.Post(url, "application/json", strings.NewReader(string(body)))
}

func (suite *VNFDMSIntegrationSuite) registerVNFWithDMS(req DMSRegistrationRequest, dmsType string) DMSRegistrationResponse {
	var server *httptest.Server
	if dmsType == "RAN" {
		server = suite.ranDMSServer
	} else {
		server = suite.cnDMSServer
	}

	body, err := json.Marshal(req)
	suite.Require().NoError(err)

	url := server.URL + "/register"
	resp, err := http.Post(url, "application/json", strings.NewReader(string(body)))
	suite.Require().NoError(err)
	defer resp.Body.Close()
	suite.Require().Equal(http.StatusCreated, resp.StatusCode)

	var response DMSRegistrationResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.Require().NoError(err)

	return response
}

func (suite *VNFDMSIntegrationSuite) configureDMS(config DMSSliceConfiguration, dmsType string) map[string]interface{} {
	resp, err := suite.makeDMSConfigRequest(config, dmsType)
	suite.Require().NoError(err)
	defer resp.Body.Close()
	suite.Require().Equal(http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.Require().NoError(err)

	return response
}

func (suite *VNFDMSIntegrationSuite) makeDMSConfigRequest(config DMSSliceConfiguration, dmsType string) (*http.Response, error) {
	var server *httptest.Server
	if dmsType == "RAN" {
		server = suite.ranDMSServer
	} else {
		server = suite.cnDMSServer
	}

	body, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	url := server.URL + "/configure"
	return http.Post(url, "application/json", strings.NewReader(string(body)))
}

func (suite *VNFDMSIntegrationSuite) initiateVNFLifecycle(req VNFLifecycleRequest) map[string]interface{} {
	body, err := json.Marshal(req)
	suite.Require().NoError(err)

	url := suite.vnfOperatorServer.URL + "/vnf/lifecycle"
	resp, err := http.Post(url, "application/json", strings.NewReader(string(body)))
	suite.Require().NoError(err)
	defer resp.Body.Close()
	suite.Require().Equal(http.StatusCreated, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.Require().NoError(err)

	return response
}

func (suite *VNFDMSIntegrationSuite) updateVNFStatus(update VNFStatusUpdate) map[string]interface{} {
	body, err := json.Marshal(update)
	suite.Require().NoError(err)

	url := suite.vnfOperatorServer.URL + "/vnf/status"
	req, err := http.NewRequest("PUT", url, strings.NewReader(string(body)))
	suite.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().NoError(err)
	defer resp.Body.Close()
	suite.Require().Equal(http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.Require().NoError(err)

	return response
}

func (suite *VNFDMSIntegrationSuite) updateVNFInDMS(update VNFStatusUpdate, dmsType string) map[string]interface{} {
	var server *httptest.Server
	if dmsType == "RAN" {
		server = suite.ranDMSServer
	} else {
		server = suite.cnDMSServer
	}

	body, err := json.Marshal(update)
	suite.Require().NoError(err)

	url := server.URL + "/vnf/update"
	req, err := http.NewRequest("PUT", url, strings.NewReader(string(body)))
	suite.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().NoError(err)
	defer resp.Body.Close()
	suite.Require().Equal(http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.Require().NoError(err)

	return response
}

func (suite *VNFDMSIntegrationSuite) getVNFInstances(sliceID string) map[string]interface{} {
	url := suite.vnfOperatorServer.URL + "/vnf/instances?sliceId=" + sliceID
	resp, err := http.Get(url)
	suite.Require().NoError(err)
	defer resp.Body.Close()
	suite.Require().Equal(http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.Require().NoError(err)

	return response
}

func (suite *VNFDMSIntegrationSuite) getDMSStatus(sliceID, dmsType string) map[string]interface{} {
	var server *httptest.Server
	if dmsType == "RAN" {
		server = suite.ranDMSServer
	} else {
		server = suite.cnDMSServer
	}

	url := server.URL + "/status?sliceId=" + sliceID
	resp, err := http.Get(url)
	suite.Require().NoError(err)
	defer resp.Body.Close()
	suite.Require().Equal(http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.Require().NoError(err)

	return response
}

// Test runner
func TestVNFDMSIntegrationSuite(t *testing.T) {
	suite.Run(t, new(VNFDMSIntegrationSuite))
}

// Benchmark tests for VNF-DMS integration performance

func BenchmarkVNFRegistrationIntegration(b *testing.B) {
	suite := &VNFDMSIntegrationSuite{}
	suite.SetupSuite()
	defer suite.TearDownSuite()

	registrationReq := DMSRegistrationRequest{
		VNFName:      "bench-vnf",
		VNFEndpoint:  "vnf.5g.local:8080",
		SliceID:      "bench-slice",
		Capabilities: []string{"N1", "N2"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registrationReq.VNFName = fmt.Sprintf("bench-vnf-%d", i)
		suite.registerVNFWithOperator(registrationReq)
	}
}

func BenchmarkDMSConfigurationIntegration(b *testing.B) {
	suite := &VNFDMSIntegrationSuite{}
	suite.SetupSuite()
	defer suite.TearDownSuite()

	sliceConfig := DMSSliceConfiguration{
		SliceID:   "bench-config-slice",
		SliceType: "eMBB",
		VNFInstances: []VNFInstanceConfig{
			{Name: "bench-vnf", Type: "UPF"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sliceConfig.SliceID = fmt.Sprintf("bench-config-slice-%d", i)
		suite.configureDMS(sliceConfig, "RAN")
	}
}