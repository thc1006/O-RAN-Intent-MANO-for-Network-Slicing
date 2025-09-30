package nlp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client := NewClient("")
	assert.Equal(t, "http://localhost:8082", client.BaseURL)
	assert.NotNil(t, client.HTTPClient)
}

func TestParseIntent(t *testing.T) {
	// Mock NLP service
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/parse", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var req IntentRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		assert.Equal(t, "Deploy video streaming slice", req.Intent)

		resp := IntentResponse{
			Success:   true,
			SliceType: "eMBB",
			QoSProfile: QoSProfile{
				SliceType:      "eMBB",
				ThroughputMbps: 50.0,
				LatencyMs:      10.0,
				PacketLossRate: 0.001,
				Priority:       5,
			},
			RawIntent:        req.Intent,
			SessionID:        req.SessionID,
			Timestamp:        "2025-10-01T00:50:00Z",
			ProcessingTimeMs: 12.5,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	result, err := client.ParseIntent(ctx, "Deploy video streaming slice", "test-session")
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "eMBB", result.SliceType)
	assert.Equal(t, 50.0, result.QoSProfile.ThroughputMbps)
	assert.Equal(t, 10.0, result.QoSProfile.LatencyMs)
}

func TestHealthCheck(t *testing.T) {
	// Mock NLP service
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/health", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		resp := HealthResponse{
			Status:                "healthy",
			Version:               "1.0.0",
			UptimeSeconds:         123.45,
			TotalIntentsProcessed: 42,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	result, err := client.HealthCheck(ctx)
	require.NoError(t, err)
	assert.Equal(t, "healthy", result.Status)
	assert.Equal(t, "1.0.0", result.Version)
	assert.Equal(t, 42, result.TotalIntentsProcessed)
}

func TestParseIntentError(t *testing.T) {
	// Mock NLP service returning error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Invalid intent"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	_, err := client.ParseIntent(ctx, "", "test-session")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NLP service returned error")
}
