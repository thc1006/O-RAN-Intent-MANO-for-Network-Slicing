package o2ims

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
	baseURL := "https://api.example.com"

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

	t.Run("with options", func(t *testing.T) {
		customTimeout := 60 * time.Second
		authToken := "test-token"
		customHTTPClient := &http.Client{Timeout: 120 * time.Second}

		client := NewClient(baseURL,
			WithTimeout(customTimeout),
			WithAuthToken(authToken),
			WithHTTPClient(customHTTPClient),
		)

		if client.timeout != customTimeout {
			t.Errorf("expected timeout %v, got %v", customTimeout, client.timeout)
		}

		if client.authToken != authToken {
			t.Errorf("expected auth token %s, got %s", authToken, client.authToken)
		}

		if client.httpClient != customHTTPClient {
			t.Errorf("expected custom HTTP client to be set")
		}
	})
}

func TestClientOptions(t *testing.T) {
	client := &Client{
		httpClient: &http.Client{},
	}

	t.Run("WithTimeout", func(t *testing.T) {
		timeout := 45 * time.Second
		option := WithTimeout(timeout)
		option(client)

		if client.timeout != timeout {
			t.Errorf("expected timeout %v, got %v", timeout, client.timeout)
		}

		if client.httpClient.Timeout != timeout {
			t.Errorf("expected HTTP client timeout %v, got %v", timeout, client.httpClient.Timeout)
		}
	})

	t.Run("WithAuthToken", func(t *testing.T) {
		token := "test-auth-token"
		option := WithAuthToken(token)
		option(client)

		if client.authToken != token {
			t.Errorf("expected auth token %s, got %s", token, client.authToken)
		}
	})

	t.Run("WithHTTPClient", func(t *testing.T) {
		customClient := &http.Client{Timeout: 90 * time.Second}
		option := WithHTTPClient(customClient)
		option(client)

		if client.httpClient != customClient {
			t.Errorf("expected custom HTTP client to be set")
		}
	})
}

func TestGetOCloudInfo(t *testing.T) {
	mockOCloudInfo := &models.OCloudInfo{
		OCloudID:    "test-cloud-1",
		Name:        "Test O-Cloud",
		Description: "Test O-Cloud for unit tests",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}

		if !strings.HasSuffix(r.URL.Path, "/o2ims-infrastructureInventory/v1/") {
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}

		// Check headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockOCloudInfo)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	ocloud, err := client.GetOCloudInfo(ctx)
	if err != nil {
		t.Fatalf("GetOCloudInfo failed: %v", err)
	}

	if ocloud.OCloudID != mockOCloudInfo.OCloudID {
		t.Errorf("expected OCloudID %s, got %s", mockOCloudInfo.OCloudID, ocloud.OCloudID)
	}

	if ocloud.Name != mockOCloudInfo.Name {
		t.Errorf("expected Name %s, got %s", mockOCloudInfo.Name, ocloud.Name)
	}
}

func TestGetOCloudInfoWithAuth(t *testing.T) {
	authToken := "Bearer test-token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != authToken {
			t.Errorf("expected Authorization header %s, got %s", authToken, r.Header.Get("Authorization"))
		}

		mockOCloudInfo := &models.OCloudInfo{
			OCloudID: "test-cloud-1",
			Name:     "Test O-Cloud",
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockOCloudInfo)
	}))
	defer server.Close()

	client := NewClient(server.URL, WithAuthToken("test-token"))
	ctx := context.Background()

	_, err := client.GetOCloudInfo(ctx)
	if err != nil {
		t.Fatalf("GetOCloudInfo with auth failed: %v", err)
	}
}

func TestListResourcePools(t *testing.T) {
	mockPools := []*models.O2CloudResourcePool{
		{
			ResourcePoolID: "pool-1",
			Name:           "Test Pool 1",
			Description:    "First test pool",
			Location:       "Region A",
			Extensions:     map[string]interface{}{"capacity": 100.0},
		},
		{
			ResourcePoolID: "pool-2",
			Name:           "Test Pool 2",
			Description:    "Second test pool",
			Location:       "Region B",
			Extensions:     map[string]interface{}{"capacity": 200.0},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}

		if !strings.Contains(r.URL.Path, "/resourcePools") {
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}

		// Convert to items format expected by client
		items := make([]interface{}, len(mockPools))
		for i, pool := range mockPools {
			items[i] = pool
		}

		response := models.ListResponse{
			Items: items,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	pools, err := client.ListResourcePools(ctx, nil)
	if err != nil {
		t.Fatalf("ListResourcePools failed: %v", err)
	}

	if len(pools) != len(mockPools) {
		t.Errorf("expected %d pools, got %d", len(mockPools), len(pools))
	}

	if pools[0].ResourcePoolID != mockPools[0].ResourcePoolID {
		t.Errorf("expected ResourcePoolID %s, got %s", mockPools[0].ResourcePoolID, pools[0].ResourcePoolID)
	}
}

func TestListResourcePoolsWithQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check query parameters
		query := r.URL.Query()
		if query.Get("limit") != "10" {
			t.Errorf("expected limit=10, got %s", query.Get("limit"))
		}
		if query.Get("offset") != "20" {
			t.Errorf("expected offset=20, got %s", query.Get("offset"))
		}
		if query.Get("filter") != "location==RegionA" {
			t.Errorf("expected filter=location==RegionA, got %s", query.Get("filter"))
		}

		response := models.ListResponse{Items: []interface{}{}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	query := &ListQuery{
		Limit:  10,
		Offset: 20,
		Filter: "location==RegionA",
	}

	_, err := client.ListResourcePools(ctx, query)
	if err != nil {
		t.Fatalf("ListResourcePools with query failed: %v", err)
	}
}

func TestGetResourcePool(t *testing.T) {
	resourcePoolID := "pool-123"
	mockPool := &models.O2CloudResourcePool{
		ResourcePoolID: resourcePoolID,
		Name:           "Test Pool",
		Description:    "A test resource pool",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := fmt.Sprintf("/o2ims-infrastructureInventory/v1/resourcePools/%s", resourcePoolID)
		if !strings.HasSuffix(r.URL.Path, expectedPath) {
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockPool)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	pool, err := client.GetResourcePool(ctx, resourcePoolID)
	if err != nil {
		t.Fatalf("GetResourcePool failed: %v", err)
	}

	if pool.ResourcePoolID != resourcePoolID {
		t.Errorf("expected ResourcePoolID %s, got %s", resourcePoolID, pool.ResourcePoolID)
	}
}

func TestBuildQueryParams(t *testing.T) {
	client := NewClient("http://example.com")

	query := &ListQuery{
		Limit:          10,
		Offset:         20,
		Marker:         "marker-123",
		Filter:         "name==test",
		Sort:           "name:asc",
		AllFields:      true,
		Fields:         []string{"name", "description"},
		ExcludeFields:  []string{"extensions"},
		ExcludeDefault: true,
	}

	params := client.buildQueryParams(query)

	if params.Get("limit") != "10" {
		t.Errorf("expected limit=10, got %s", params.Get("limit"))
	}

	if params.Get("offset") != "20" {
		t.Errorf("expected offset=20, got %s", params.Get("offset"))
	}

	if params.Get("marker") != "marker-123" {
		t.Errorf("expected marker=marker-123, got %s", params.Get("marker"))
	}

	if params.Get("filter") != "name==test" {
		t.Errorf("expected filter=name==test, got %s", params.Get("filter"))
	}

	if params.Get("sort") != "name:asc" {
		t.Errorf("expected sort=name:asc, got %s", params.Get("sort"))
	}

	if params.Get("all_fields") != "true" {
		t.Errorf("expected all_fields=true, got %s", params.Get("all_fields"))
	}

	if params.Get("exclude_default") != "true" {
		t.Errorf("expected exclude_default=true, got %s", params.Get("exclude_default"))
	}

	fields := params["fields"]
	if len(fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(fields))
	}

	excludeFields := params["exclude_fields"]
	if len(excludeFields) != 1 {
		t.Errorf("expected 1 exclude field, got %d", len(excludeFields))
	}
}

func TestHTTPErrorHandling(t *testing.T) {
	t.Run("404 Not Found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_ = w.Write([]byte("Not Found"))
		}))
		defer server.Close()

		client := NewClient(server.URL)
		ctx := context.Background()

		_, err := client.GetOCloudInfo(ctx)
		if err == nil {
			t.Fatal("expected error for 404 response")
		}

		if !strings.Contains(err.Error(), "404") {
			t.Errorf("error should mention 404 status: %v", err)
		}
	})

	t.Run("API Error Response", func(t *testing.T) {
		apiError := models.APIError{
			Status: 400,
			Title:  "Bad Request",
			Detail: "Invalid parameter provided",
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(apiError)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		ctx := context.Background()

		_, err := client.GetOCloudInfo(ctx)
		if err == nil {
			t.Fatal("expected error for API error response")
		}

		expectedError := fmt.Sprintf("API error %d: %s - %s", apiError.Status, apiError.Title, apiError.Detail)
		if !strings.Contains(err.Error(), "Bad Request") {
			t.Errorf("expected error to contain '%s', got: %v", expectedError, err)
		}
	})
}

func TestContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	_, err := client.GetOCloudInfo(ctx)
	if err == nil {
		t.Fatal("expected error due to context cancellation")
	}

	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("expected context cancellation error, got: %v", err)
	}
}

func TestFindResourcesByQoS(t *testing.T) {
	qosReq := &models.ORanQoSRequirements{
		Bandwidth: 100.0,
		Latency:   10.0,
		SliceType: "eMBB",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/resourcePools") && !strings.Contains(r.URL.Path, "/resources") {
			// Return resource pools
			pools := []interface{}{
				map[string]interface{}{
					"resourcePoolId": "pool-1",
					"name":           "Test Pool",
				},
			}
			response := models.ListResponse{Items: pools}
			_ = json.NewEncoder(w).Encode(response)
		} else if strings.Contains(r.URL.Path, "/resources") {
			// Return resources for the pool
			resources := []interface{}{
				map[string]interface{}{
					"resourceId":   "resource-1",
					"name":         "Test Resource",
					"resourceType": "compute",
					"extensions": map[string]interface{}{
						"bandwidth": 150.0,
						"latency":   5.0,
					},
				},
			}
			response := models.ListResponse{Items: resources}
			_ = json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	resources, err := client.FindResourcesByQoS(ctx, qosReq)
	if err != nil {
		t.Fatalf("FindResourcesByQoS failed: %v", err)
	}

	if len(resources) == 0 {
		t.Error("expected at least one matching resource")
	}
}

func TestResourceMeetsQoS(t *testing.T) {
	client := NewClient("http://example.com")

	qosReq := &models.ORanQoSRequirements{
		Bandwidth: 100.0,
		Latency:   10.0,
	}

	t.Run("resource meets requirements", func(t *testing.T) {
		resource := &models.Resource{
			Extensions: map[string]interface{}{
				"bandwidth": 150.0,
				"latency":   5.0,
			},
		}

		meets := client.resourceMeetsQoS(resource, qosReq)
		if !meets {
			t.Error("resource should meet QoS requirements")
		}
	})

	t.Run("resource does not meet bandwidth", func(t *testing.T) {
		resource := &models.Resource{
			Extensions: map[string]interface{}{
				"bandwidth": 50.0,
				"latency":   5.0,
			},
		}

		meets := client.resourceMeetsQoS(resource, qosReq)
		if meets {
			t.Error("resource should not meet QoS requirements (insufficient bandwidth)")
		}
	})

	t.Run("resource does not meet latency", func(t *testing.T) {
		resource := &models.Resource{
			Extensions: map[string]interface{}{
				"bandwidth": 150.0,
				"latency":   15.0,
			},
		}

		meets := client.resourceMeetsQoS(resource, qosReq)
		if meets {
			t.Error("resource should not meet QoS requirements (high latency)")
		}
	})

	t.Run("resource without extensions", func(t *testing.T) {
		resource := &models.Resource{
			Extensions: map[string]interface{}{},
		}

		meets := client.resourceMeetsQoS(resource, qosReq)
		if !meets {
			t.Error("resource without extension should pass (default behavior)")
		}
	})
}