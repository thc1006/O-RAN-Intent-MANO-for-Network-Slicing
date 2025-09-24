package o2dms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/o2-client/pkg/models"
)

func TestNewClient(t *testing.T) {
	baseURL := "https://o2dms.example.com"

	t.Run("default configuration", func(t *testing.T) {
		client := NewClient(baseURL)

		if client.baseURL != baseURL {
			t.Errorf("expected baseURL %s, got %s", baseURL, client.baseURL)
		}

		if client.timeout != 30*time.Second {
			t.Errorf("expected timeout 30s, got %v", client.timeout)
		}

		if client.httpClient.Timeout != 30*time.Second {
			t.Errorf("expected HTTP client timeout 30s, got %v", client.httpClient.Timeout)
		}

		if client.authToken != "" {
			t.Errorf("expected empty auth token, got %s", client.authToken)
		}
	})

	t.Run("with custom options", func(t *testing.T) {
		customTimeout := 90 * time.Second
		authToken := "dms-auth-token"

		client := NewClient(baseURL,
			WithTimeout(customTimeout),
			WithAuthToken(authToken),
		)

		if client.timeout != customTimeout {
			t.Errorf("expected timeout %v, got %v", customTimeout, client.timeout)
		}

		if client.authToken != authToken {
			t.Errorf("expected auth token %s, got %s", authToken, client.authToken)
		}
	})
}

func TestListDeploymentManagers(t *testing.T) {
	mockManagers := []*models.DeploymentManager{
		{
			DeploymentManagerID: "dm-1",
			Name:                "Kubernetes DM",
			Description:         "Kubernetes deployment manager",
			SupportedLocations:  []string{"region-a", "region-b"},
			Capabilities:        []string{"helm", "kustomize"},
		},
		{
			DeploymentManagerID: "dm-2",
			Name:                "OpenStack DM",
			Description:         "OpenStack deployment manager",
			SupportedLocations:  []string{"region-c"},
			Capabilities:        []string{"heat", "tosca"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}

		if !strings.Contains(r.URL.Path, "/deploymentManagers") {
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}

		// Convert to items format
		items := make([]interface{}, len(mockManagers))
		for i, mgr := range mockManagers {
			items[i] = mgr
		}

		response := models.ListResponse{Items: items}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	managers, err := client.ListDeploymentManagers(ctx, nil)
	if err != nil {
		t.Fatalf("ListDeploymentManagers failed: %v", err)
	}

	if len(managers) != len(mockManagers) {
		t.Errorf("expected %d managers, got %d", len(mockManagers), len(managers))
	}

	if managers[0].DeploymentManagerID != mockManagers[0].DeploymentManagerID {
		t.Errorf("expected DeploymentManagerID %s, got %s",
			mockManagers[0].DeploymentManagerID, managers[0].DeploymentManagerID)
	}
}

func TestGetDeploymentManager(t *testing.T) {
	deploymentManagerID := "dm-123"
	mockManager := &models.DeploymentManager{
		DeploymentManagerID: deploymentManagerID,
		Name:                "Test DM",
		Description:         "Test deployment manager",
		SupportedLocations:  []string{"region-test"},
		Capabilities:        []string{"test-cap"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := fmt.Sprintf("/o2dms/v1/deploymentManagers/%s", deploymentManagerID)
		if !strings.HasSuffix(r.URL.Path, expectedPath) {
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockManager)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	manager, err := client.GetDeploymentManager(ctx, deploymentManagerID)
	if err != nil {
		t.Fatalf("GetDeploymentManager failed: %v", err)
	}

	if manager.DeploymentManagerID != deploymentManagerID {
		t.Errorf("expected DeploymentManagerID %s, got %s",
			deploymentManagerID, manager.DeploymentManagerID)
	}
}

func TestCreateNFDeployment(t *testing.T) {
	deploymentManagerID := "dm-123"
	request := &DeploymentRequest{
		Name:                     "test-vnf-deployment",
		Description:              "Test VNF deployment",
		NFDeploymentDescriptorID: "desc-456",
		InputParams: map[string]interface{}{
			"cpu":    "2",
			"memory": "4Gi",
		},
		LocationConstraints: []string{"region-a"},
		Extensions: map[string]interface{}{
			"oran.io/slice-type": "eMBB",
		},
	}

	expectedDeployment := &models.NFDeployment{
		ID:          "deployment-789",
		Name:        request.Name,
		Description: request.Description,
		Status:      models.NFDeploymentStatusInstantiating,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		expectedPath := fmt.Sprintf("/o2dms/v1/deploymentManagers/%s/nfDeployments", deploymentManagerID)
		if !strings.HasSuffix(r.URL.Path, expectedPath) {
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}

		// Verify request body
		var receivedRequest DeploymentRequest
		if err := json.NewDecoder(r.Body).Decode(&receivedRequest); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}

		if receivedRequest.Name != request.Name {
			t.Errorf("expected Name %s, got %s", request.Name, receivedRequest.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(expectedDeployment)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	deployment, err := client.CreateNFDeployment(ctx, deploymentManagerID, request)
	if err != nil {
		t.Fatalf("CreateNFDeployment failed: %v", err)
	}

	if deployment.ID != expectedDeployment.ID {
		t.Errorf("expected ID %s, got %s",
			expectedDeployment.ID, deployment.ID)
	}

	if deployment.Status != expectedDeployment.Status {
		t.Errorf("expected Status %s, got %s", expectedDeployment.Status, deployment.Status)
	}
}

func TestListNFDeployments(t *testing.T) {
	deploymentManagerID := "dm-123"
	mockDeployments := []*models.NFDeployment{
		{
			ID:     "deployment-1",
			Name:   "VNF Deployment 1",
			Status: models.NFDeploymentStatusInstantiated,
		},
		{
			ID:     "deployment-2",
			Name:   "VNF Deployment 2",
			Status: models.NFDeploymentStatusInstantiating,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := fmt.Sprintf("/o2dms/v1/deploymentManagers/%s/nfDeployments", deploymentManagerID)
		if !strings.HasSuffix(r.URL.Path, expectedPath) {
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}

		items := make([]interface{}, len(mockDeployments))
		for i, deployment := range mockDeployments {
			items[i] = deployment
		}

		response := models.ListResponse{Items: items}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	deployments, err := client.ListNFDeployments(ctx, deploymentManagerID, nil)
	if err != nil {
		t.Fatalf("ListNFDeployments failed: %v", err)
	}

	if len(deployments) != len(mockDeployments) {
		t.Errorf("expected %d deployments, got %d", len(mockDeployments), len(deployments))
	}

	if deployments[0].ID != mockDeployments[0].ID {
		t.Errorf("expected ID %s, got %s",
			mockDeployments[0].ID, deployments[0].ID)
	}
}

func TestGetNFDeployment(t *testing.T) {
	deploymentManagerID := "dm-123"
	deploymentID := "deployment-456"
	mockDeployment := &models.NFDeployment{
		ID:          deploymentID,
		Name:        "Test VNF",
		Status:      models.NFDeploymentStatusInstantiated,
		Description: "Test VNF deployment",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := fmt.Sprintf("/o2dms/v1/deploymentManagers/%s/nfDeployments/%s", deploymentManagerID, deploymentID)
		if !strings.HasSuffix(r.URL.Path, expectedPath) {
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockDeployment)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	deployment, err := client.GetNFDeployment(ctx, deploymentManagerID, deploymentID)
	if err != nil {
		t.Fatalf("GetNFDeployment failed: %v", err)
	}

	if deployment.ID != deploymentID {
		t.Errorf("expected ID %s, got %s", deploymentID, deployment.ID)
	}
}

func TestUpdateNFDeployment(t *testing.T) {
	deploymentManagerID := "dm-123"
	deploymentID := "deployment-456"
	updateRequest := &DeploymentRequest{
		Name:        "updated-vnf-deployment",
		Description: "Updated VNF deployment",
		InputParams: map[string]interface{}{
			"cpu":    "4",
			"memory": "8Gi",
		},
	}

	updatedDeployment := &models.NFDeployment{
		ID:          deploymentID,
		Name:        updateRequest.Name,
		Description: updateRequest.Description,
		Status:      models.NFDeploymentStatusInstantiated,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT request, got %s", r.Method)
		}

		expectedPath := fmt.Sprintf("/o2dms/v1/deploymentManagers/%s/nfDeployments/%s", deploymentManagerID, deploymentID)
		if !strings.HasSuffix(r.URL.Path, expectedPath) {
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updatedDeployment)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	deployment, err := client.UpdateNFDeployment(ctx, deploymentManagerID, deploymentID, updateRequest)
	if err != nil {
		t.Fatalf("UpdateNFDeployment failed: %v", err)
	}

	if deployment.Name != updateRequest.Name {
		t.Errorf("expected Name %s, got %s", updateRequest.Name, deployment.Name)
	}
}

func TestDeleteNFDeployment(t *testing.T) {
	deploymentManagerID := "dm-123"
	deploymentID := "deployment-456"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE request, got %s", r.Method)
		}

		expectedPath := fmt.Sprintf("/o2dms/v1/deploymentManagers/%s/nfDeployments/%s", deploymentManagerID, deploymentID)
		if !strings.HasSuffix(r.URL.Path, expectedPath) {
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	err := client.DeleteNFDeployment(ctx, deploymentManagerID, deploymentID)
	if err != nil {
		t.Fatalf("DeleteNFDeployment failed: %v", err)
	}
}

func TestDeployVNFWithQoS(t *testing.T) {
	deploymentManagerID := "dm-123"
	vnfType := "cu-vnf"
	qosReq := &models.ORanQoSRequirements{
		Bandwidth: 100.0,
		Latency:   10.0,
		SliceType: "eMBB",
	}
	placement := &models.ORanPlacement{
		CloudType: "distributed",
		Region:    "us-west-1",
		Zone:      "us-west-1a",
	}

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "nfDeploymentDescriptors") {
			// First call - return deployment descriptors
			descriptors := []interface{}{
				map[string]interface{}{
					"id":          "desc-789",
					"name":        "CU VNF Descriptor",
					"vnfType":     vnfType,
					"description": "Central Unit VNF deployment descriptor",
				},
			}
			response := models.ListResponse{Items: descriptors}
			json.NewEncoder(w).Encode(response)
		} else if strings.Contains(r.URL.Path, "nfDeployments") && r.Method == "POST" {
			// Second call - create deployment
			deployment := &models.NFDeployment{
				ID:     "deployment-new",
				Name:   fmt.Sprintf("%s-deployment-%d", vnfType, time.Now().Unix()),
				Status: models.NFDeploymentStatusInstantiating,
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(deployment)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	deployment, err := client.DeployVNFWithQoS(ctx, deploymentManagerID, vnfType, qosReq, placement)
	if err != nil {
		t.Fatalf("DeployVNFWithQoS failed: %v", err)
	}

	if deployment.ID != "deployment-new" {
		t.Errorf("expected ID deployment-new, got %s", deployment.ID)
	}

	if callCount != 2 {
		t.Errorf("expected 2 HTTP calls, got %d", callCount)
	}
}

func TestWaitForDeploymentReady(t *testing.T) {
	deploymentManagerID := "dm-123"
	deploymentID := "deployment-456"

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		deployment := &models.NFDeployment{
			ID:   deploymentID,
			Name: "Test Deployment",
		}

		// First call returns "instantiating", second call returns "instantiated"
		if callCount == 1 {
			deployment.Status = models.NFDeploymentStatusInstantiating
		} else {
			deployment.Status = models.NFDeploymentStatusInstantiated
		}

		json.NewEncoder(w).Encode(deployment)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	deployment, err := client.WaitForDeploymentReady(ctx, deploymentManagerID, deploymentID, 10*time.Second)
	if err != nil {
		t.Fatalf("WaitForDeploymentReady failed: %v", err)
	}

	if deployment.Status != models.NFDeploymentStatusInstantiated {
		t.Errorf("expected status %s, got %s", models.NFDeploymentStatusInstantiated, deployment.Status)
	}

	if callCount < 2 {
		t.Errorf("expected at least 2 calls, got %d", callCount)
	}
}

func TestWaitForDeploymentReadyTimeout(t *testing.T) {
	deploymentManagerID := "dm-123"
	deploymentID := "deployment-456"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		deployment := &models.NFDeployment{
			ID:     deploymentID,
			Name:   "Test Deployment",
			Status: models.NFDeploymentStatusInstantiating, // Always return instantiating
		}
		json.NewEncoder(w).Encode(deployment)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	// Use a very short timeout
	_, err := client.WaitForDeploymentReady(ctx, deploymentManagerID, deploymentID, 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}

	if !strings.Contains(err.Error(), "did not become ready within timeout") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestWaitForDeploymentReadyFailed(t *testing.T) {
	deploymentManagerID := "dm-123"
	deploymentID := "deployment-456"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		deployment := &models.NFDeployment{
			ID:     deploymentID,
			Name:   "Test Deployment",
			Status: models.NFDeploymentStatusFailed,
		}
		json.NewEncoder(w).Encode(deployment)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	_, err := client.WaitForDeploymentReady(ctx, deploymentManagerID, deploymentID, 10*time.Second)
	if err == nil {
		t.Fatal("expected failure error")
	}

	if !strings.Contains(err.Error(), "deployment failed") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGetDeploymentsBySliceType(t *testing.T) {
	deploymentManagerID := "dm-123"
	sliceType := "eMBB"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the filter parameter
		query := r.URL.Query()
		expectedFilter := fmt.Sprintf("sliceType==%s", sliceType)
		if query.Get("filter") != expectedFilter {
			t.Errorf("expected filter %s, got %s", expectedFilter, query.Get("filter"))
		}

		deployments := []interface{}{
			map[string]interface{}{
				"id":        "deployment-1",
				"name":      "eMBB VNF 1",
				"sliceType": sliceType,
			},
			map[string]interface{}{
				"id":        "deployment-2",
				"name":      "eMBB VNF 2",
				"sliceType": sliceType,
			},
		}

		response := models.ListResponse{Items: deployments}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	deployments, err := client.GetDeploymentsBySliceType(ctx, deploymentManagerID, sliceType)
	if err != nil {
		t.Fatalf("GetDeploymentsBySliceType failed: %v", err)
	}

	if len(deployments) != 2 {
		t.Errorf("expected 2 deployments, got %d", len(deployments))
	}
}

func TestBuildQueryParams(t *testing.T) {
	client := NewClient("http://example.com")

	query := &ListQuery{
		Limit:          50,
		Offset:         100,
		Marker:         "marker-456",
		Filter:         "status==instantiated",
		Sort:           "name:desc",
		AllFields:      true,
		Fields:         []string{"name", "status"},
		ExcludeFields:  []string{"metadata"},
		ExcludeDefault: false,
	}

	params := client.buildQueryParams(query)

	if params.Get("limit") != "50" {
		t.Errorf("expected limit=50, got %s", params.Get("limit"))
	}

	if params.Get("offset") != "100" {
		t.Errorf("expected offset=100, got %s", params.Get("offset"))
	}

	if params.Get("filter") != "status==instantiated" {
		t.Errorf("expected filter=status==instantiated, got %s", params.Get("filter"))
	}

	// ExcludeDefault is false, so it should not be in params
	if params.Get("exclude_default") != "" {
		t.Errorf("expected exclude_default to be empty when false, got %s", params.Get("exclude_default"))
	}
}

func TestHealthCheck(t *testing.T) {
	mockHealth := &models.HealthInfo{
		Status: "OK",
		Extensions: map[string]interface{}{
			"database": "connected",
			"storage":  "available",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/o2dms/v1/health") {
			t.Errorf("unexpected health check path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockHealth)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	health, err := client.GetHealthInfo(ctx)
	if err != nil {
		t.Fatalf("GetHealthInfo failed: %v", err)
	}

	if health.Status != mockHealth.Status {
		t.Errorf("expected Status %s, got %s", mockHealth.Status, health.Status)
	}
}