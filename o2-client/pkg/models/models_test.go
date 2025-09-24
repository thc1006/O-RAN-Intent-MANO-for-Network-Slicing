package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestOCloudInfoSerialization(t *testing.T) {
	ocloud := &OCloudInfo{
		OCloudID:          "ocloud-123",
		GlobalCloudID:     "global-456",
		Name:              "Test O-Cloud",
		Description:       "Test O-Cloud for unit tests",
		ServiceURI:        "https://api.ocloud.example.com",
		SupportedFeatures: []string{"feature1", "feature2"},
		Extensions: map[string]interface{}{
			"customField": "customValue",
			"version":     1.0,
		},
	}

	// Test marshaling
	data, err := json.Marshal(ocloud)
	if err != nil {
		t.Fatalf("failed to marshal OCloudInfo: %v", err)
	}

	// Test unmarshaling
	var unmarshaled OCloudInfo
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal OCloudInfo: %v", err)
	}

	// Verify fields
	if unmarshaled.OCloudID != ocloud.OCloudID {
		t.Errorf("expected OCloudID %s, got %s", ocloud.OCloudID, unmarshaled.OCloudID)
	}

	if unmarshaled.Name != ocloud.Name {
		t.Errorf("expected Name %s, got %s", ocloud.Name, unmarshaled.Name)
	}

	if len(unmarshaled.SupportedFeatures) != len(ocloud.SupportedFeatures) {
		t.Errorf("expected %d features, got %d", len(ocloud.SupportedFeatures), len(unmarshaled.SupportedFeatures))
	}

	if unmarshaled.Extensions["customField"] != "customValue" {
		t.Errorf("expected customField=customValue, got %v", unmarshaled.Extensions["customField"])
	}
}

func TestO2CloudResourcePoolSerialization(t *testing.T) {
	pool := &O2CloudResourcePool{
		ResourcePoolID: "pool-123",
		OCloudID:       "ocloud-456",
		GlobalCloudID:  "global-789",
		Name:           "Test Resource Pool",
		Description:    "Test resource pool for unit tests",
		Location:       "us-west-1",
		State:          ResourcePoolStateEnabled,
		Resources: []Resource{
			{
				ResourceID:     "resource-001",
				ResourcePoolID: "pool-123",
				OCloudID:       "ocloud-456",
				ResourceTypeID: "compute",
				Name:           "Compute Resource 1",
				Description:    "First compute resource",
			},
		},
		Extensions: map[string]interface{}{
			"capacity":     100,
			"utilization":  75.5,
			"maintenance":  false,
		},
	}

	// Test marshaling
	data, err := json.Marshal(pool)
	if err != nil {
		t.Fatalf("failed to marshal O2CloudResourcePool: %v", err)
	}

	// Test unmarshaling
	var unmarshaled O2CloudResourcePool
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal O2CloudResourcePool: %v", err)
	}

	// Verify fields
	if unmarshaled.ResourcePoolID != pool.ResourcePoolID {
		t.Errorf("expected ResourcePoolID %s, got %s", pool.ResourcePoolID, unmarshaled.ResourcePoolID)
	}

	if unmarshaled.State != pool.State {
		t.Errorf("expected State %s, got %s", pool.State, unmarshaled.State)
	}

	if len(unmarshaled.Resources) != len(pool.Resources) {
		t.Errorf("expected %d resources, got %d", len(pool.Resources), len(unmarshaled.Resources))
	}

	if unmarshaled.Resources[0].ResourceID != pool.Resources[0].ResourceID {
		t.Errorf("expected ResourceID %s, got %s", pool.Resources[0].ResourceID, unmarshaled.Resources[0].ResourceID)
	}
}

func TestResourceTypeSerialization(t *testing.T) {
	typeInfo := &ResourceTypeInfo{
		ResourceTypeID:  "type-123",
		Name:            "Compute Node",
		Description:     "High-performance compute node",
		Vendor:          "TestVendor",
		Model:           "TestModel-X1",
		Version:         "v1.2.3",
		ResourceKind:    "physical",
		ResourceClass:   "compute",
		Extensions: map[string]interface{}{
			"cores":       64,
			"memory":      "512GB",
			"accelerators": []string{"GPU", "FPGA"},
		},
		AlarmDictionary: AlarmDictionary{
			ID:         "alarm-dict-1",
			Name:       "Compute Node Alarms",
			EntityType: "ComputeNode",
			AlarmDefinition: []AlarmDef{
				{
					AlarmCode:        "CPU_HIGH",
					AlarmName:        "High CPU Utilization",
					AlarmDescription: "CPU utilization exceeds 90%",
					ProposedRepairActions: "Scale out or reduce load",
					ClearingType:     "automatic",
				},
			},
		},
	}

	// Test marshaling
	data, err := json.Marshal(typeInfo)
	if err != nil {
		t.Fatalf("failed to marshal ResourceTypeInfo: %v", err)
	}

	// Test unmarshaling
	var unmarshaled ResourceTypeInfo
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal ResourceTypeInfo: %v", err)
	}

	// Verify fields
	if unmarshaled.ResourceTypeID != typeInfo.ResourceTypeID {
		t.Errorf("expected ResourceTypeID %s, got %s", typeInfo.ResourceTypeID, unmarshaled.ResourceTypeID)
	}

	if unmarshaled.Vendor != typeInfo.Vendor {
		t.Errorf("expected Vendor %s, got %s", typeInfo.Vendor, unmarshaled.Vendor)
	}

	if len(unmarshaled.AlarmDictionary.AlarmDefinition) != len(typeInfo.AlarmDictionary.AlarmDefinition) {
		t.Errorf("expected %d alarm definitions, got %d",
			len(typeInfo.AlarmDictionary.AlarmDefinition),
			len(unmarshaled.AlarmDictionary.AlarmDefinition))
	}

	if unmarshaled.AlarmDictionary.AlarmDefinition[0].AlarmCode != "CPU_HIGH" {
		t.Errorf("expected AlarmCode CPU_HIGH, got %s",
			unmarshaled.AlarmDictionary.AlarmDefinition[0].AlarmCode)
	}
}

func TestDeploymentManagerSerialization(t *testing.T) {
	manager := &DeploymentManager{
		DeploymentManagerID: "dm-123",
		OCloudID:           "ocloud-456",
		Name:               "Kubernetes Deployment Manager",
		Description:        "Manages Kubernetes-based deployments",
		DeploymentManagementServiceEndpoint: "https://k8s.example.com",
		CapacityInfo:       "100 pods, 1000 containers",
		State:              DeploymentManagerStateEnabled,
		SupportedLocations: []string{"us-west-1", "us-east-1", "eu-west-1"},
		Capabilities:       []string{"helm", "kustomize", "yaml"},
		Extensions: map[string]interface{}{
			"k8sVersion":     "1.24.0",
			"clusterNodes":   5,
			"storageClasses": []string{"fast-ssd", "standard"},
		},
	}

	// Test marshaling
	data, err := json.Marshal(manager)
	if err != nil {
		t.Fatalf("failed to marshal DeploymentManager: %v", err)
	}

	// Test unmarshaling
	var unmarshaled DeploymentManager
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal DeploymentManager: %v", err)
	}

	// Verify fields
	if unmarshaled.DeploymentManagerID != manager.DeploymentManagerID {
		t.Errorf("expected DeploymentManagerID %s, got %s",
			manager.DeploymentManagerID, unmarshaled.DeploymentManagerID)
	}

	if unmarshaled.State != manager.State {
		t.Errorf("expected State %s, got %s", manager.State, unmarshaled.State)
	}

	if len(unmarshaled.SupportedLocations) != len(manager.SupportedLocations) {
		t.Errorf("expected %d locations, got %d",
			len(manager.SupportedLocations), len(unmarshaled.SupportedLocations))
	}

	if len(unmarshaled.Capabilities) != len(manager.Capabilities) {
		t.Errorf("expected %d capabilities, got %d",
			len(manager.Capabilities), len(unmarshaled.Capabilities))
	}
}

func TestNFDeploymentSerialization(t *testing.T) {
	now := time.Now().UTC()
	deployment := &NFDeployment{
		ID:                       "nf-deploy-123",
		Name:                     "CU VNF Deployment",
		Description:              "Central Unit VNF deployment for 5G",
		NFDeploymentDescriptorID: "desc-456",
		ParentDeploymentID:       "parent-789",
		DeploymentManagerID:      "dm-101",
		Status:                   NFDeploymentStatusInstantiated,
		InputParams: map[string]interface{}{
			"cpu":         "4",
			"memory":      "8Gi",
			"replicas":    3,
			"sliceType":   "eMBB",
		},
		OutputParams: map[string]interface{}{
			"serviceEndpoint": "https://cu-vnf.example.com",
			"healthStatus":    "healthy",
		},
		CreationTime:   now,
		LastUpdateTime: now.Add(1 * time.Hour),
		Extensions: map[string]interface{}{
			"helmRelease":  "cu-vnf-v1.2.3",
			"namespace":    "ran-functions",
			"monitoring":   true,
		},
	}

	// Test marshaling
	data, err := json.Marshal(deployment)
	if err != nil {
		t.Fatalf("failed to marshal NFDeployment: %v", err)
	}

	// Test unmarshaling
	var unmarshaled NFDeployment
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal NFDeployment: %v", err)
	}

	// Verify fields
	if unmarshaled.ID != deployment.ID {
		t.Errorf("expected ID %s, got %s", deployment.ID, unmarshaled.ID)
	}

	if unmarshaled.Status != deployment.Status {
		t.Errorf("expected Status %s, got %s", deployment.Status, unmarshaled.Status)
	}

	if unmarshaled.InputParams["cpu"] != "4" {
		t.Errorf("expected cpu=4, got %v", unmarshaled.InputParams["cpu"])
	}

	if unmarshaled.OutputParams["serviceEndpoint"] != "https://cu-vnf.example.com" {
		t.Errorf("expected serviceEndpoint https://cu-vnf.example.com, got %v",
			unmarshaled.OutputParams["serviceEndpoint"])
	}

	// Verify time fields (with some tolerance for JSON serialization)
	if unmarshaled.CreationTime.Unix() != deployment.CreationTime.Unix() {
		t.Errorf("expected CreationTime %v, got %v",
			deployment.CreationTime, unmarshaled.CreationTime)
	}
}

func TestSubscriptionSerialization(t *testing.T) {
	now := time.Now().UTC()
	subscription := &Subscription{
		ID:                     "sub-123",
		SubscriptionID:         "sub-456",
		Status:                 "active",
		Callback:               "https://webhook.example.com/notifications",
		ConsumerSubscriptionID: "consumer-789",
		Filter:                 "resourceType==compute",
		SystemType:             []string{"O2IMS", "O2DMS"},
		CreatedAt:              now,
		Spec: SubscriptionSpec{
			Filter: EventFilter{
				EventTypes: []string{"ResourceCreated", "ResourceUpdated", "ResourceDeleted"},
				Source:     "o2ims",
			},
			CallbackURL: "https://webhook.example.com/notifications",
			ExpiryTime:  now.Add(24 * time.Hour),
		},
		Extensions: map[string]interface{}{
			"retryPolicy": "exponential",
			"maxRetries":  5,
		},
	}

	// Test marshaling
	data, err := json.Marshal(subscription)
	if err != nil {
		t.Fatalf("failed to marshal Subscription: %v", err)
	}

	// Test unmarshaling
	var unmarshaled Subscription
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal Subscription: %v", err)
	}

	// Verify fields
	if unmarshaled.SubscriptionID != subscription.SubscriptionID {
		t.Errorf("expected SubscriptionID %s, got %s",
			subscription.SubscriptionID, unmarshaled.SubscriptionID)
	}

	if len(unmarshaled.Spec.Filter.EventTypes) != len(subscription.Spec.Filter.EventTypes) {
		t.Errorf("expected %d event types, got %d",
			len(subscription.Spec.Filter.EventTypes),
			len(unmarshaled.Spec.Filter.EventTypes))
	}

	if unmarshaled.Spec.Filter.Source != subscription.Spec.Filter.Source {
		t.Errorf("expected Source %s, got %s",
			subscription.Spec.Filter.Source, unmarshaled.Spec.Filter.Source)
	}
}

func TestORanQoSRequirementsSerialization(t *testing.T) {
	qos := &ORanQoSRequirements{
		Bandwidth:    100.5,
		Latency:      10.2,
		Jitter:       1.5,
		PacketLoss:   0.01,
		Reliability:  99.99,
		SliceType:    "eMBB",
		Priority:     5,
	}

	// Test marshaling
	data, err := json.Marshal(qos)
	if err != nil {
		t.Fatalf("failed to marshal ORanQoSRequirements: %v", err)
	}

	// Test unmarshaling
	var unmarshaled ORanQoSRequirements
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal ORanQoSRequirements: %v", err)
	}

	// Verify fields
	if unmarshaled.Bandwidth != qos.Bandwidth {
		t.Errorf("expected Bandwidth %f, got %f", qos.Bandwidth, unmarshaled.Bandwidth)
	}

	if unmarshaled.Latency != qos.Latency {
		t.Errorf("expected Latency %f, got %f", qos.Latency, unmarshaled.Latency)
	}

	if unmarshaled.SliceType != qos.SliceType {
		t.Errorf("expected SliceType %s, got %s", qos.SliceType, unmarshaled.SliceType)
	}

	if unmarshaled.Priority != qos.Priority {
		t.Errorf("expected Priority %d, got %d", qos.Priority, unmarshaled.Priority)
	}
}

func TestORanPlacementSerialization(t *testing.T) {
	placement := &ORanPlacement{
		CloudType:     "edge",
		Region:        "us-west-1",
		Zone:          "us-west-1a",
		Site:          "edge-site-001",
		AffinityRules: []string{"anti-affinity:cu-du", "affinity:cu-cp"},
	}

	// Test marshaling
	data, err := json.Marshal(placement)
	if err != nil {
		t.Fatalf("failed to marshal ORanPlacement: %v", err)
	}

	// Test unmarshaling
	var unmarshaled ORanPlacement
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal ORanPlacement: %v", err)
	}

	// Verify fields
	if unmarshaled.CloudType != placement.CloudType {
		t.Errorf("expected CloudType %s, got %s", placement.CloudType, unmarshaled.CloudType)
	}

	if unmarshaled.Region != placement.Region {
		t.Errorf("expected Region %s, got %s", placement.Region, unmarshaled.Region)
	}

	if len(unmarshaled.AffinityRules) != len(placement.AffinityRules) {
		t.Errorf("expected %d affinity rules, got %d",
			len(placement.AffinityRules), len(unmarshaled.AffinityRules))
	}
}

func TestORanSliceInfoSerialization(t *testing.T) {
	slice := &ORanSliceInfo{
		SliceID:     "slice-123",
		ServiceType: "eMBB",
		QoSRequirements: ORanQoSRequirements{
			Bandwidth: 1000.0,
			Latency:   5.0,
			SliceType: "eMBB",
			Priority:  8,
		},
		Placement: ORanPlacement{
			CloudType: "edge",
			Region:    "us-west-1",
		},
		NetworkFunctions: []string{"cu", "du", "ue"},
		Capacity: map[string]interface{}{
			"maxUsers":      10000,
			"maxThroughput": "10Gbps",
		},
		SLA: map[string]interface{}{
			"availability": 99.99,
			"mttr":         "< 5 minutes",
		},
	}

	// Test marshaling
	data, err := json.Marshal(slice)
	if err != nil {
		t.Fatalf("failed to marshal ORanSliceInfo: %v", err)
	}

	// Test unmarshaling
	var unmarshaled ORanSliceInfo
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal ORanSliceInfo: %v", err)
	}

	// Verify fields
	if unmarshaled.SliceID != slice.SliceID {
		t.Errorf("expected SliceID %s, got %s", slice.SliceID, unmarshaled.SliceID)
	}

	if unmarshaled.QoSRequirements.Bandwidth != slice.QoSRequirements.Bandwidth {
		t.Errorf("expected QoS Bandwidth %f, got %f",
			slice.QoSRequirements.Bandwidth, unmarshaled.QoSRequirements.Bandwidth)
	}

	if unmarshaled.Placement.CloudType != slice.Placement.CloudType {
		t.Errorf("expected Placement CloudType %s, got %s",
			slice.Placement.CloudType, unmarshaled.Placement.CloudType)
	}

	if len(unmarshaled.NetworkFunctions) != len(slice.NetworkFunctions) {
		t.Errorf("expected %d network functions, got %d",
			len(slice.NetworkFunctions), len(unmarshaled.NetworkFunctions))
	}
}

func TestAPIErrorSerialization(t *testing.T) {
	apiError := &APIError{
		Type:     "https://example.com/errors/validation",
		Title:    "Validation Error",
		Status:   400,
		Detail:   "The request body contains invalid parameters",
		Instance: "/api/v1/deployments/123",
	}

	// Test marshaling
	data, err := json.Marshal(apiError)
	if err != nil {
		t.Fatalf("failed to marshal APIError: %v", err)
	}

	// Test unmarshaling
	var unmarshaled APIError
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal APIError: %v", err)
	}

	// Verify fields
	if unmarshaled.Status != apiError.Status {
		t.Errorf("expected Status %d, got %d", apiError.Status, unmarshaled.Status)
	}

	if unmarshaled.Title != apiError.Title {
		t.Errorf("expected Title %s, got %s", apiError.Title, unmarshaled.Title)
	}

	if unmarshaled.Detail != apiError.Detail {
		t.Errorf("expected Detail %s, got %s", apiError.Detail, unmarshaled.Detail)
	}
}

func TestListResponseSerialization(t *testing.T) {
	// Create some test items
	items := []interface{}{
		map[string]interface{}{
			"id":   "item-1",
			"name": "First Item",
		},
		map[string]interface{}{
			"id":   "item-2",
			"name": "Second Item",
		},
	}

	listResp := &ListResponse{
		Items:      items,
		Total:      100,
		NextMarker: "marker-next",
		PrevMarker: "marker-prev",
		HasMore:    true,
	}

	// Test marshaling
	data, err := json.Marshal(listResp)
	if err != nil {
		t.Fatalf("failed to marshal ListResponse: %v", err)
	}

	// Test unmarshaling
	var unmarshaled ListResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal ListResponse: %v", err)
	}

	// Verify fields
	if len(unmarshaled.Items) != len(listResp.Items) {
		t.Errorf("expected %d items, got %d", len(listResp.Items), len(unmarshaled.Items))
	}

	if unmarshaled.Total != listResp.Total {
		t.Errorf("expected Total %d, got %d", listResp.Total, unmarshaled.Total)
	}

	if unmarshaled.HasMore != listResp.HasMore {
		t.Errorf("expected HasMore %v, got %v", listResp.HasMore, unmarshaled.HasMore)
	}

	// Verify item content
	firstItem := unmarshaled.Items[0].(map[string]interface{})
	if firstItem["id"] != "item-1" {
		t.Errorf("expected first item id=item-1, got %v", firstItem["id"])
	}
}

func TestResourceTypesConstantValues(t *testing.T) {
	// Test that resource type constants have expected values
	expectedTypes := map[ResourceType]string{
		ResourceTypeNode:        "Node",
		ResourceTypeCPU:         "CPU",
		ResourceTypeMemory:      "Memory",
		ResourceTypeStorage:     "Storage",
		ResourceTypeNetwork:     "Network",
		ResourceTypeAccelerator: "Accelerator",
	}

	for resourceType, expectedValue := range expectedTypes {
		if string(resourceType) != expectedValue {
			t.Errorf("expected %s, got %s", expectedValue, string(resourceType))
		}
	}
}

func TestNFDeploymentStatusConstants(t *testing.T) {
	// Test that NFDeploymentStatus constants have expected values
	expectedStatuses := map[NFDeploymentStatus]string{
		NFDeploymentStatusNotInstantiated: "NOT_INSTANTIATED",
		NFDeploymentStatusInstantiated:    "INSTANTIATED",
		NFDeploymentStatusFailed:          "FAILED",
	}

	for status, expectedValue := range expectedStatuses {
		if string(status) != expectedValue {
			t.Errorf("expected %s, got %s", expectedValue, string(status))
		}
	}
}

func TestResourcePoolStateConstants(t *testing.T) {
	// Test ResourcePoolState constants
	if string(ResourcePoolStateEnabled) != "enabled" {
		t.Errorf("expected 'enabled', got %s", string(ResourcePoolStateEnabled))
	}

	if string(ResourcePoolStateDisabled) != "disabled" {
		t.Errorf("expected 'disabled', got %s", string(ResourcePoolStateDisabled))
	}
}

func TestDeploymentManagerStateConstants(t *testing.T) {
	// Test DeploymentManagerState constants
	if string(DeploymentManagerStateEnabled) != "enabled" {
		t.Errorf("expected 'enabled', got %s", string(DeploymentManagerStateEnabled))
	}

	if string(DeploymentManagerStateDisabled) != "disabled" {
		t.Errorf("expected 'disabled', got %s", string(DeploymentManagerStateDisabled))
	}
}