package fixtures

import (
	"encoding/json"
	"time"
)

// O2 DMS API Response structures
type O2DMSInventoryResponse struct {
	ResourceTypes []ResourceType `json:"resourceTypes"`
}

type ResourceType struct {
	ResourceTypeID   string            `json:"resourceTypeId"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	Vendor           string            `json:"vendor"`
	Model            string            `json:"model"`
	Version          string            `json:"version"`
	Extensions       map[string]string `json:"extensions,omitempty"`
}

type O2DMSResourceResponse struct {
	Resources []Resource `json:"resources"`
}

type Resource struct {
	ResourceID       string            `json:"resourceId"`
	ResourceTypeID   string            `json:"resourceTypeId"`
	GlobalAssetID    string            `json:"globalAssetId"`
	Description      string            `json:"description"`
	Location         string            `json:"location"`
	ParentID         string            `json:"parentId,omitempty"`
	Extensions       map[string]string `json:"extensions,omitempty"`
}

type O2DMSConfigurationResponse struct {
	Configuration Configuration `json:"configuration"`
}

type Configuration struct {
	ConfigurationID string            `json:"configurationId"`
	Version         string            `json:"version"`
	Data            map[string]interface{} `json:"data"`
	LastModified    time.Time         `json:"lastModified"`
}

type O2DMSFaultResponse struct {
	Alarms []Alarm `json:"alarms"`
}

type Alarm struct {
	AlarmID          string    `json:"alarmId"`
	ResourceID       string    `json:"resourceId"`
	Severity         string    `json:"severity"`
	ProbableCause    string    `json:"probableCause"`
	AdditionalText   string    `json:"additionalText"`
	RaisedTime       time.Time `json:"raisedTime"`
	ChangedTime      time.Time `json:"changedTime,omitempty"`
	ClearedTime      time.Time `json:"clearedTime,omitempty"`
	PerceivedSeverity string   `json:"perceivedSeverity"`
}

type O2DMSSubscriptionRequest struct {
	ConsumerSubscriptionID string   `json:"consumerSubscriptionId"`
	Filter                 string   `json:"filter"`
	Callback               string   `json:"callback"`
	SystemType             []string `json:"systemType,omitempty"`
}

type O2DMSSubscriptionResponse struct {
	SubscriptionID string `json:"subscriptionId"`
	Callback       string `json:"callback"`
	Filter         string `json:"filter"`
}

// Test fixtures for O2 DMS responses
func ValidInventoryResponse() string {
	response := O2DMSInventoryResponse{
		ResourceTypes: []ResourceType{
			{
				ResourceTypeID: "rt-cucp-001",
				Name:          "CU-CP",
				Description:   "Central Unit Control Plane",
				Vendor:        "Ericsson",
				Model:         "CUCP-5G-v2.1",
				Version:       "2.1.0",
				Extensions: map[string]string{
					"slice-support": "eMBB,URLLC,mMTC",
					"max-slices":   "32",
				},
			},
			{
				ResourceTypeID: "rt-cuup-001",
				Name:          "CU-UP",
				Description:   "Central Unit User Plane",
				Vendor:        "Nokia",
				Model:         "CUUP-5G-v1.5",
				Version:       "1.5.2",
				Extensions: map[string]string{
					"throughput": "10Gbps",
					"latency":    "5ms",
				},
			},
		},
	}
	data, _ := json.Marshal(response)
	return string(data)
}

func ValidResourceResponse() string {
	response := O2DMSResourceResponse{
		Resources: []Resource{
			{
				ResourceID:     "res-cucp-node1",
				ResourceTypeID: "rt-cucp-001",
				GlobalAssetID:  "asset-12345",
				Description:    "CU-CP instance on edge node 1",
				Location:       "edge-zone-a",
				Extensions: map[string]string{
					"status":     "active",
					"allocated":  "true",
					"slice-id":   "slice-embb-001",
				},
			},
			{
				ResourceID:     "res-cuup-node1",
				ResourceTypeID: "rt-cuup-001",
				GlobalAssetID:  "asset-12346",
				Description:    "CU-UP instance on edge node 1",
				Location:       "edge-zone-a",
				ParentID:       "res-cucp-node1",
				Extensions: map[string]string{
					"status":    "active",
					"allocated": "false",
				},
			},
		},
	}
	data, _ := json.Marshal(response)
	return string(data)
}

func ValidConfigurationResponse() string {
	response := O2DMSConfigurationResponse{
		Configuration: Configuration{
			ConfigurationID: "config-cucp-001",
			Version:        "1.0.0",
			Data: map[string]interface{}{
				"amf-endpoint": "http://amf.5gc:8080",
				"smf-endpoint": "http://smf.5gc:8080",
				"slice-config": map[string]interface{}{
					"eMBB": map[string]interface{}{
						"latency":    "20ms",
						"throughput": "1Gbps",
					},
					"URLLC": map[string]interface{}{
						"latency":    "1ms",
						"throughput": "100Mbps",
					},
				},
			},
			LastModified: time.Now(),
		},
	}
	data, _ := json.Marshal(response)
	return string(data)
}

func ValidFaultResponse() string {
	response := O2DMSFaultResponse{
		Alarms: []Alarm{
			{
				AlarmID:          "alarm-001",
				ResourceID:       "res-cucp-node1",
				Severity:         "major",
				ProbableCause:    "communication-failure",
				AdditionalText:   "Connection to AMF lost",
				RaisedTime:       time.Now().Add(-time.Hour),
				ChangedTime:      time.Now().Add(-time.Minute * 30),
				PerceivedSeverity: "major",
			},
			{
				AlarmID:          "alarm-002",
				ResourceID:       "res-cuup-node1",
				Severity:         "minor",
				ProbableCause:    "performance-degradation",
				AdditionalText:   "High CPU utilization",
				RaisedTime:       time.Now().Add(-time.Minute * 15),
				PerceivedSeverity: "minor",
			},
		},
	}
	data, _ := json.Marshal(response)
	return string(data)
}

func ValidSubscriptionResponse() string {
	response := O2DMSSubscriptionResponse{
		SubscriptionID: "sub-001",
		Callback:       "http://oran-controller:8080/notifications",
		Filter:         "resourceType=CU-CP",
	}
	data, _ := json.Marshal(response)
	return string(data)
}

func EmptyInventoryResponse() string {
	response := O2DMSInventoryResponse{
		ResourceTypes: []ResourceType{},
	}
	data, _ := json.Marshal(response)
	return string(data)
}

func ErrorResponse() string {
	return `{
		"error": {
			"code": "INTERNAL_ERROR",
			"message": "Database connection failed",
			"details": "Connection timeout after 30 seconds"
		}
	}`
}

func UnauthorizedResponse() string {
	return `{
		"error": {
			"code": "UNAUTHORIZED",
			"message": "Invalid authentication token",
			"details": "Token expired or invalid"
		}
	}`
}

func ResourceNotFoundResponse() string {
	return `{
		"error": {
			"code": "NOT_FOUND",
			"message": "Resource not found",
			"details": "Resource with ID res-invalid-001 does not exist"
		}
	}`
}

// Helper functions for creating subscription requests
func CreateSubscriptionRequest() O2DMSSubscriptionRequest {
	return O2DMSSubscriptionRequest{
		ConsumerSubscriptionID: "consumer-sub-001",
		Filter:                "resourceType=CU-CP AND location=edge-zone-a",
		Callback:              "http://oran-controller:8080/notifications",
		SystemType:            []string{"O-RAN-CU"},
	}
}

func CreateInvalidSubscriptionRequest() O2DMSSubscriptionRequest {
	return O2DMSSubscriptionRequest{
		ConsumerSubscriptionID: "", // Invalid: empty ID
		Filter:                "invalid filter syntax",
		Callback:              "invalid-url",
	}
}