package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	. "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/o2client"
)

// TestO2ClientAuthentication tests O2 IMS authentication flow
func TestO2ClientAuthentication(t *testing.T) {
	tests := []struct {
		name       string
		username   string
		password   string
		response   interface{}
		statusCode int
		wantErr    bool
		errMsg     string
	}{
		{
			name:     "successful authentication",
			username: "admin",
			password: "password",
			response: map[string]string{
				"token":      "test-token-12345",
				"expires_in": "3600",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:     "invalid credentials",
			username: "invalid",
			password: "wrong",
			response: map[string]string{
				"error": "unauthorized",
			},
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
			errMsg:     "authentication failed",
		},
		{
			name:       "empty credentials",
			username:   "",
			password:   "",
			statusCode: http.StatusBadRequest,
			wantErr:    true,
			errMsg:     "credentials cannot be empty",
		},
		{
			name:     "server error",
			username: "admin",
			password: "password",
			response: map[string]string{
				"error": "internal server error",
			},
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
			errMsg:     "server error",
		},
		{
			name:     "malformed response",
			username: "admin",
			password: "password",
			response: "invalid json",
			statusCode: http.StatusOK,
			wantErr:    true,
			errMsg:     "failed to parse response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/auth/token", r.URL.Path)
				assert.Equal(t, "POST", r.Method)

				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					switch v := tt.response.(type) {
					case string:
						w.Write([]byte(v))
					default:
						json.NewEncoder(w).Encode(v)
					}
				}
			}))
			defer server.Close()

			client := NewO2Client(server.URL, tt.username, tt.password)
			err := client.Authenticate()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, client.GetToken())
			}
		})
	}
}

// TestO2ClientGetResource tests resource retrieval
func TestO2ClientGetResource(t *testing.T) {
	tests := []struct {
		name       string
		resourceID string
		response   interface{}
		statusCode int
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "get cloud resource",
			resourceID: "cloud-123",
			response: map[string]interface{}{
				"id":   "cloud-123",
				"type": "CloudRegion",
				"name": "RegionOne",
				"resources": map[string]interface{}{
					"cpu":    100,
					"memory": 512000,
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "get deployment manager",
			resourceID: "dm-456",
			response: map[string]interface{}{
				"id":       "dm-456",
				"type":     "DeploymentManager",
				"endpoint": "http://nephio:8080",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "resource not found",
			resourceID: "nonexistent",
			response: map[string]string{
				"error": "not found",
			},
			statusCode: http.StatusNotFound,
			wantErr:    true,
			errMsg:     "resource not found",
		},
		{
			name:       "empty resource ID",
			resourceID: "",
			statusCode: http.StatusBadRequest,
			wantErr:    true,
			errMsg:     "resource ID cannot be empty",
		},
		{
			name:       "unauthorized request",
			resourceID: "cloud-123",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
			errMsg:     "unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Contains(t, r.URL.Path, "/resources/")

				// Check auth header
				authHeader := r.Header.Get("Authorization")
				if tt.statusCode == http.StatusUnauthorized {
					assert.Empty(t, authHeader)
				}

				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			client := NewO2Client(server.URL, "user", "pass")
			if tt.statusCode != http.StatusUnauthorized {
				client.SetToken("test-token")
			}

			resource, err := client.GetResource(tt.resourceID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resource)
			}
		})
	}
}

// TestO2ClientTimeout tests timeout handling
func TestO2ClientTimeout(t *testing.T) {
	tests := []struct {
		name       string
		timeout    time.Duration
		serverWait time.Duration
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "request within timeout",
			timeout:    2 * time.Second,
			serverWait: 100 * time.Millisecond,
			wantErr:    false,
		},
		{
			name:       "request exceeds timeout",
			timeout:    100 * time.Millisecond,
			serverWait: 2 * time.Second,
			wantErr:    true,
			errMsg:     "context deadline exceeded",
		},
		{
			name:       "immediate timeout",
			timeout:    1 * time.Millisecond,
			serverWait: 1 * time.Second,
			wantErr:    true,
			errMsg:     "deadline exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(tt.serverWait)
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			}))
			defer server.Close()

			client := NewO2Client(server.URL, "user", "pass")
			client.SetTimeout(tt.timeout)

			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout+1*time.Second)
			defer cancel()

			err := client.GetResourceWithContext(ctx, "test-resource")

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestO2ClientListResources tests resource listing
func TestO2ClientListResources(t *testing.T) {
	tests := []struct {
		name       string
		filter     map[string]string
		response   interface{}
		statusCode int
		wantCount  int
		wantErr    bool
	}{
		{
			name:   "list all resources",
			filter: nil,
			response: []map[string]interface{}{
				{"id": "res1", "type": "CloudRegion"},
				{"id": "res2", "type": "DeploymentManager"},
				{"id": "res3", "type": "NetworkSlice"},
			},
			statusCode: http.StatusOK,
			wantCount:  3,
			wantErr:    false,
		},
		{
			name:   "filter by type",
			filter: map[string]string{"type": "CloudRegion"},
			response: []map[string]interface{}{
				{"id": "res1", "type": "CloudRegion"},
			},
			statusCode: http.StatusOK,
			wantCount:  1,
			wantErr:    false,
		},
		{
			name:       "empty result",
			filter:     map[string]string{"type": "NonExistent"},
			response:   []map[string]interface{}{},
			statusCode: http.StatusOK,
			wantCount:  0,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "/resources", r.URL.Path)

				// Check query parameters
				if tt.filter != nil {
					for k, v := range tt.filter {
						assert.Equal(t, v, r.URL.Query().Get(k))
					}
				}

				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			client := NewO2Client(server.URL, "user", "pass")
			client.SetToken("test-token")

			resources, err := client.ListResources(tt.filter)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, resources, tt.wantCount)
			}
		})
	}
}

// TestO2ClientRetry tests retry mechanism
func TestO2ClientRetry(t *testing.T) {
	tests := []struct {
		name          string
		failCount     int
		maxRetries    int
		expectSuccess bool
	}{
		{
			name:          "success on first try",
			failCount:     0,
			maxRetries:    3,
			expectSuccess: true,
		},
		{
			name:          "success after retries",
			failCount:     2,
			maxRetries:    3,
			expectSuccess: true,
		},
		{
			name:          "fail after max retries",
			failCount:     4,
			maxRetries:    3,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempts++
				if attempts <= tt.failCount {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			}))
			defer server.Close()

			client := NewO2Client(server.URL, "user", "pass")
			client.SetMaxRetries(tt.maxRetries)

			_, err := client.GetResource("test-resource")

			if tt.expectSuccess {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			assert.LessOrEqual(t, attempts, tt.maxRetries+1)
		})
	}
}

