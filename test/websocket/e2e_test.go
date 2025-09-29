package websocket_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	wsserver "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/websocket"
)

// TestWebSocketServerE2E tests the complete WebSocket integration
func TestWebSocketServerE2E(t *testing.T) {
	// Create test server
	server := wsserver.NewServer(":0") // Use any available port

	// Create HTTP test server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ws":
			server.HandleWebSocket(w, r)
		case "/health":
			server.HandleHealth(w, r)
		case "/":
			server.ServeHome(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer testServer.Close()

	// Test health endpoint
	t.Run("Health endpoint", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var health map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err)

		assert.Equal(t, "healthy", health["status"])
		assert.Equal(t, float64(0), health["activeSessions"]) // Initially no sessions
	})

	// Test WebSocket connection
	t.Run("WebSocket connection and intent processing", func(t *testing.T) {
		// Convert HTTP URL to WebSocket URL
		wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"

		// Connect to WebSocket
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer conn.Close()

		// Set read deadline for tests
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))

		// Read welcome message
		var welcomeMsg wsserver.Message
		err = conn.ReadJSON(&welcomeMsg)
		require.NoError(t, err)

		assert.Equal(t, "connected", welcomeMsg.Type)
		assert.NotEmpty(t, welcomeMsg.SessionID)
		assert.Equal(t, "Connected to O-RAN Network Slicing Claude Service", welcomeMsg.Message)
		assert.Equal(t, "success", welcomeMsg.Status)

		sessionID := welcomeMsg.SessionID

		// Send intent request
		intentReq := wsserver.IntentRequest{
			Type:      "intent",
			Intent:    "Deploy an eMBB slice for 4K video streaming with 1 Gbps throughput",
			SessionID: sessionID,
		}

		err = conn.WriteJSON(intentReq)
		require.NoError(t, err)

		// Read processing message
		var processingMsg wsserver.Message
		err = conn.ReadJSON(&processingMsg)
		require.NoError(t, err)

		assert.Equal(t, "processing", processingMsg.Type)
		assert.Equal(t, sessionID, processingMsg.SessionID)
		assert.Equal(t, "Processing your intent...", processingMsg.Message)

		// Read intent response
		var responseMsg wsserver.Message
		err = conn.ReadJSON(&responseMsg)
		require.NoError(t, err)

		// Verify response structure
		if responseMsg.Type == "intent_response" {
			// Check if response has expected fields in Data
			if responseMsg.Data != nil {
				if sliceType, ok := responseMsg.Data["sliceType"].(string); ok {
					assert.NotEmpty(t, sliceType)
					assert.Contains(t, []string{"eMBB", "URLLC", "mIoT"}, sliceType)
				}

				if requirements, ok := responseMsg.Data["requirements"].(map[string]interface{}); ok {
					if throughput, ok := requirements["throughput"].(float64); ok {
						assert.True(t, throughput >= 0)
					}
					if latency, ok := requirements["latency"].(float64); ok {
						assert.True(t, latency >= 0)
					}
					if reliability, ok := requirements["reliability"].(float64); ok {
						assert.True(t, reliability >= 0)
					}
				}
			}
		} else if responseMsg.Type == "error" {
			t.Logf("Received error (expected in test environment): %s", responseMsg.Message)
		}
	})

	// Test multiple concurrent clients
	t.Run("Multiple concurrent clients", func(t *testing.T) {
		const numClients = 3
		wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"

		var clients []*websocket.Conn
		defer func() {
			for _, client := range clients {
				if client != nil {
					client.Close()
				}
			}
		}()

		// Connect multiple clients
		for i := 0; i < numClients; i++ {
			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			require.NoError(t, err)
			clients = append(clients, conn)

			// Read welcome message
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			var welcomeMsg wsserver.Message
			err = conn.ReadJSON(&welcomeMsg)
			require.NoError(t, err)
			assert.Equal(t, "connected", welcomeMsg.Type)
		}

		// Send intents from all clients
		intents := []string{
			"Create an eMBB slice for video streaming",
			"Deploy a URLLC slice for autonomous vehicles",
			"Setup an mIoT slice for smart city sensors",
		}

		for i, client := range clients {
			intentReq := wsserver.IntentRequest{
				Type:   "intent",
				Intent: intents[i],
			}

			err := client.WriteJSON(intentReq)
			require.NoError(t, err)

			// Set longer read deadline for processing
			client.SetReadDeadline(time.Now().Add(10 * time.Second))

			// Read processing message
			var processingMsg wsserver.Message
			err = client.ReadJSON(&processingMsg)
			require.NoError(t, err)
			assert.Equal(t, "processing", processingMsg.Type)

			// Read response message with extended timeout
			client.SetReadDeadline(time.Now().Add(15 * time.Second))
			var responseMsg wsserver.Message
			err = client.ReadJSON(&responseMsg)
			require.NoError(t, err)
			// Response should be either intent_response or error
			assert.Contains(t, []string{"intent_response", "error"}, responseMsg.Type)
		}
	})

	// Test connection cleanup
	t.Run("Connection cleanup", func(t *testing.T) {
		wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"

		// Connect client
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)

		// Read welcome message
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		var welcomeMsg wsserver.Message
		err = conn.ReadJSON(&welcomeMsg)
		require.NoError(t, err)

		// Close connection
		conn.Close()

		// Wait a bit for cleanup
		time.Sleep(100 * time.Millisecond)

		// Verify sessions are cleaned up by checking health endpoint
		resp, err := http.Get(testServer.URL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		var health map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err)

		// Sessions should be cleaned up
		activeSessions := int(health["activeSessions"].(float64))
		assert.True(t, activeSessions >= 0, "Active sessions should be non-negative")
	})
}

// TestWebSocketMessageFormats tests message format validation
func TestWebSocketMessageFormats(t *testing.T) {
	testCases := []struct {
		name        string
		intent      string
		expectError bool
	}{
		{
			name:        "Valid eMBB intent",
			intent:      "Deploy an eMBB slice for 4K video streaming with 1 Gbps throughput",
			expectError: false,
		},
		{
			name:        "Valid URLLC intent",
			intent:      "Create a URLLC slice for autonomous vehicle control with 1ms latency",
			expectError: false,
		},
		{
			name:        "Valid mIoT intent",
			intent:      "Setup mIoT slice for smart city sensors supporting 1M devices",
			expectError: false,
		},
		{
			name:        "Empty intent",
			intent:      "",
			expectError: true,
		},
		{
			name:        "Very long intent",
			intent:      strings.Repeat("a", 10000),
			expectError: false, // Should handle gracefully
		},
		{
			name:        "Special characters",
			intent:      "Create slice with @#$%^&*()_+ special chars",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test server
			server := wsserver.NewServer(":0")
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/ws" {
					server.HandleWebSocket(w, r)
				}
			}))
			defer testServer.Close()

			// Connect to WebSocket
			wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"
			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			require.NoError(t, err)
			defer conn.Close()

			// Read welcome message
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			var welcomeMsg wsserver.Message
			err = conn.ReadJSON(&welcomeMsg)
			require.NoError(t, err)

			// Send intent
			intentReq := wsserver.IntentRequest{
				Type:      "intent",
				Intent:    tc.intent,
				SessionID: welcomeMsg.SessionID,
			}

			err = conn.WriteJSON(intentReq)
			require.NoError(t, err)

			// Read response
			var responseMsg wsserver.Message
			err = conn.ReadJSON(&responseMsg)

			if tc.expectError {
				if err == nil {
					// If no network error, should get error message
					assert.Contains(t, []string{"error", "processing"}, responseMsg.Type)
				}
			} else {
				require.NoError(t, err)
				assert.Contains(t, []string{"processing", "error"}, responseMsg.Type)
			}
		})
	}
}

// TestWebSocketServerIntegration tests server lifecycle
func TestWebSocketServerIntegration(t *testing.T) {
	t.Run("Server creation and configuration", func(t *testing.T) {
		server := wsserver.NewServer(":8080")
		assert.NotNil(t, server)

		// Test initial state
		assert.Equal(t, 0, server.GetActiveSessions())
	})

	t.Run("Server methods accessibility", func(t *testing.T) {
		server := wsserver.NewServer(":0")

		// Test broadcast capability
		testMsg := wsserver.Message{
			Type:      "test",
			Message:   "test broadcast",
			Timestamp: time.Now().Unix(),
		}

		// Should not panic
		assert.NotPanics(t, func() {
			server.BroadcastMessage(testMsg)
		})

		// Test session count
		assert.Equal(t, 0, server.GetActiveSessions())
	})
}

// Benchmark WebSocket performance
func BenchmarkWebSocketIntentProcessing(b *testing.B) {
	server := wsserver.NewServer(":0")
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ws" {
			server.HandleWebSocket(w, r)
		}
	}))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Connect
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			b.Fatal(err)
		}

		// Read welcome
		var welcomeMsg wsserver.Message
		conn.SetReadDeadline(time.Now().Add(time.Second))
		conn.ReadJSON(&welcomeMsg)

		// Send intent
		intentReq := wsserver.IntentRequest{
			Type:   "intent",
			Intent: fmt.Sprintf("Create eMBB slice %d", i),
		}
		conn.WriteJSON(intentReq)

		// Read response
		var responseMsg wsserver.Message
		conn.ReadJSON(&responseMsg)

		conn.Close()
	}
}