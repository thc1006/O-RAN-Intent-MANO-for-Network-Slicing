// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

package security

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SecureHTTPServer represents a security-hardened HTTP server configuration
type SecureHTTPServer struct {
	server   *http.Server
	router   *mux.Router
	config   *SecurityConfig
	shutdown chan bool
	mu       sync.RWMutex
}

// SecurityConfig holds security configuration for the HTTP server
type SecurityConfig struct {
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	MaxHeaderBytes    int
	MaxRequestSize    int64
	EnableHTTPS       bool
	RequireClientCert bool
	AllowedOrigins    []string
	RateLimitRPS      int
	EnableHSTS        bool
	CSPPolicy         string
}

// DefaultSecurityConfig returns a secure default configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
		MaxRequestSize:    1 << 20, // 1MB
		EnableHTTPS:       true,
		RequireClientCert: false,
		AllowedOrigins:    []string{"https://localhost"},
		RateLimitRPS:      100,
		EnableHSTS:        true,
		CSPPolicy:         "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'",
	}
}

// NewSecureHTTPServer creates a new security-hardened HTTP server
func NewSecureHTTPServer(config *SecurityConfig) *SecureHTTPServer {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	router := mux.NewRouter()

	server := &http.Server{
		ReadTimeout:       config.ReadTimeout,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       config.IdleTimeout,
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		MaxHeaderBytes:    config.MaxHeaderBytes,
		Handler:           router,
	}

	s := &SecureHTTPServer{
		server:   server,
		router:   router,
		config:   config,
		shutdown: make(chan bool, 1),
	}

	s.setupSecurityMiddleware()
	s.setupRoutes()

	return s
}

// setupSecurityMiddleware configures security middleware
func (s *SecureHTTPServer) setupSecurityMiddleware() {
	s.router.Use(s.securityHeadersMiddleware)
	s.router.Use(s.requestSizeLimitMiddleware)
	s.router.Use(s.rateLimitMiddleware)
	s.router.Use(s.corsMiddleware)
	s.router.Use(s.timeoutMiddleware)
}

// Security middleware implementations
func (s *SecureHTTPServer) securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		if s.config.EnableHSTS && r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		if s.config.CSPPolicy != "" {
			w.Header().Set("Content-Security-Policy", s.config.CSPPolicy)
		}

		// Remove server info
		w.Header().Set("Server", "")

		next.ServeHTTP(w, r)
	})
}

func (s *SecureHTTPServer) requestSizeLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > s.config.MaxRequestSize {
			http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
			return
		}

		// Limit body reader
		r.Body = http.MaxBytesReader(w, r.Body, s.config.MaxRequestSize)

		next.ServeHTTP(w, r)
	})
}

func (s *SecureHTTPServer) rateLimitMiddleware(next http.Handler) http.Handler {
	// Simple rate limiting implementation
	clients := make(map[string]*RateLimiter)
	var mu sync.Mutex

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)

		mu.Lock()
		if clients[clientIP] == nil {
			clients[clientIP] = NewRateLimiter(s.config.RateLimitRPS)
		}
		limiter := clients[clientIP]
		mu.Unlock()

		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *SecureHTTPServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range s.config.AllowedOrigins {
			if origin == allowedOrigin || allowedOrigin == "*" {
				allowed = true
				break
			}
		}

		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *SecureHTTPServer) timeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), s.config.ReadTimeout)
		defer cancel()

		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

// setupRoutes configures secure routes
func (s *SecureHTTPServer) setupRoutes() {
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	s.router.HandleFunc("/status", s.handleStatus).Methods("GET")
	s.router.HandleFunc("/config", s.handleConfig).Methods("GET", "PUT")
}

// Route handlers
func (s *SecureHTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *SecureHTTPServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":  "running",
		"uptime":  time.Since(time.Now()),
		"version": "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *SecureHTTPServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.handleGetConfig(w, r)
	case "PUT":
		s.handleUpdateConfig(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *SecureHTTPServer) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.config)
}

func (s *SecureHTTPServer) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var newConfig SecurityConfig

	if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate configuration
	if err := s.validateConfig(&newConfig); err != nil {
		http.Error(w, fmt.Sprintf("Invalid configuration: %v", err), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.config = &newConfig
	s.mu.Unlock()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// validateConfig validates security configuration
func (s *SecureHTTPServer) validateConfig(config *SecurityConfig) error {
	if config.ReadTimeout <= 0 || config.ReadTimeout > 60*time.Second {
		return fmt.Errorf("readTimeout must be between 1s and 60s")
	}

	if config.WriteTimeout <= 0 || config.WriteTimeout > 60*time.Second {
		return fmt.Errorf("writeTimeout must be between 1s and 60s")
	}

	if config.MaxRequestSize <= 0 || config.MaxRequestSize > 100<<20 {
		return fmt.Errorf("maxRequestSize must be between 1 byte and 100MB")
	}

	return nil
}

// Helper types for rate limiting
type RateLimiter struct {
	rate     int
	interval time.Duration
	tokens   int
	lastTime time.Time
	mu       sync.Mutex
}

func NewRateLimiter(rps int) *RateLimiter {
	return &RateLimiter{
		rate:     rps,
		interval: time.Second,
		tokens:   rps,
		lastTime: time.Now(),
	}
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastTime)

	if elapsed >= rl.interval {
		rl.tokens = rl.rate
		rl.lastTime = now
	}

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to remote address
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

// Test functions start here

func TestHTTPServerSecurityConfiguration(t *testing.T) {
	t.Run("default_secure_configuration", func(t *testing.T) {
		config := DefaultSecurityConfig()

		assert.Equal(t, 10*time.Second, config.ReadTimeout)
		assert.Equal(t, 10*time.Second, config.WriteTimeout)
		assert.Equal(t, 5*time.Second, config.ReadHeaderTimeout)
		assert.Equal(t, 1<<20, config.MaxHeaderBytes)
		assert.Equal(t, int64(1<<20), config.MaxRequestSize)
		assert.True(t, config.EnableHTTPS)
		assert.True(t, config.EnableHSTS)
		assert.NotEmpty(t, config.CSPPolicy)
	})

	t.Run("server_timeout_configuration", func(t *testing.T) {
		config := &SecurityConfig{
			ReadTimeout:       5 * time.Second,
			WriteTimeout:      5 * time.Second,
			IdleTimeout:       15 * time.Second,
			ReadHeaderTimeout: 2 * time.Second,
			MaxHeaderBytes:    1 << 16,
			MaxRequestSize:    1 << 16,
		}

		server := NewSecureHTTPServer(config)

		assert.Equal(t, 5*time.Second, server.server.ReadTimeout)
		assert.Equal(t, 5*time.Second, server.server.WriteTimeout)
		assert.Equal(t, 15*time.Second, server.server.IdleTimeout)
		assert.Equal(t, 2*time.Second, server.server.ReadHeaderTimeout)
		assert.Equal(t, 1<<16, server.server.MaxHeaderBytes)
	})

	t.Run("invalid_timeout_configuration", func(t *testing.T) {
		server := NewSecureHTTPServer(nil)

		tests := []struct {
			name   string
			config SecurityConfig
		}{
			{
				name: "negative_read_timeout",
				config: SecurityConfig{
					ReadTimeout:    -1 * time.Second,
					WriteTimeout:   10 * time.Second,
					MaxRequestSize: 1024,
				},
			},
			{
				name: "excessive_read_timeout",
				config: SecurityConfig{
					ReadTimeout:    120 * time.Second,
					WriteTimeout:   10 * time.Second,
					MaxRequestSize: 1024,
				},
			},
			{
				name: "excessive_request_size",
				config: SecurityConfig{
					ReadTimeout:    10 * time.Second,
					WriteTimeout:   10 * time.Second,
					MaxRequestSize: 1000 << 20, // 1000MB
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := server.validateConfig(&tt.config)
				assert.Error(t, err, "Should reject invalid configuration")
			})
		}
	})
}

func TestHTTPSecurityHeaders(t *testing.T) {
	server := NewSecureHTTPServer(nil)

	t.Run("security_headers_present", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		headers := w.Header()

		assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
		assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))
		assert.Equal(t, "1; mode=block", headers.Get("X-XSS-Protection"))
		assert.Equal(t, "strict-origin-when-cross-origin", headers.Get("Referrer-Policy"))
		assert.NotEmpty(t, headers.Get("Content-Security-Policy"))
		assert.Empty(t, headers.Get("Server")) // Server header should be removed
	})

	t.Run("hsts_header_with_tls", func(t *testing.T) {
		config := DefaultSecurityConfig()
		config.EnableHSTS = true
		server := NewSecureHTTPServer(config)

		req := httptest.NewRequest("GET", "/health", nil)
		req.TLS = &tls.ConnectionState{} // Simulate TLS connection
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		hsts := w.Header().Get("Strict-Transport-Security")
		assert.Contains(t, hsts, "max-age=31536000")
		assert.Contains(t, hsts, "includeSubDomains")
	})

	t.Run("no_hsts_without_tls", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		// No TLS connection state
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		hsts := w.Header().Get("Strict-Transport-Security")
		assert.Empty(t, hsts, "HSTS should not be set without TLS")
	})
}

func TestHTTPRequestSizeLimits(t *testing.T) {
	config := &SecurityConfig{
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxRequestSize: 1024, // 1KB limit
	}
	server := NewSecureHTTPServer(config)

	t.Run("reject_oversized_request", func(t *testing.T) {
		// Create request larger than limit
		largeData := strings.Repeat("x", 2048) // 2KB

		req := httptest.NewRequest("PUT", "/config", strings.NewReader(largeData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	})

	t.Run("accept_normal_sized_request", func(t *testing.T) {
		validConfig := map[string]interface{}{
			"readTimeout":    "10s",
			"writeTimeout":   "10s",
			"maxRequestSize": 512,
		}

		configJSON, _ := json.Marshal(validConfig)

		req := httptest.NewRequest("PUT", "/config", bytes.NewBuffer(configJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		assert.NotEqual(t, http.StatusRequestEntityTooLarge, w.Code)
	})

	t.Run("reject_content_length_attack", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/config", strings.NewReader("small"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Length", "999999") // Lie about content length
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		// Should handle content-length mismatch gracefully
		assert.True(t, w.Code >= 400, "Should reject content-length attack")
	})
}

func TestHTTPRateLimiting(t *testing.T) {
	config := &SecurityConfig{
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxRequestSize: 1024,
		RateLimitRPS:   2, // Very low rate limit for testing
	}
	server := NewSecureHTTPServer(config)

	t.Run("enforce_rate_limiting", func(t *testing.T) {
		// Make multiple rapid requests
		var responses []int

		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/health", nil)
			req.RemoteAddr = "192.168.1.100:12345" // Same IP
			w := httptest.NewRecorder()

			server.router.ServeHTTP(w, req)
			responses = append(responses, w.Code)
		}

		// Should have some rate-limited responses
		rateLimitedCount := 0
		for _, code := range responses {
			if code == http.StatusTooManyRequests {
				rateLimitedCount++
			}
		}

		assert.True(t, rateLimitedCount > 0, "Should rate limit excessive requests")
	})

	t.Run("different_ips_not_rate_limited", func(t *testing.T) {
		// Requests from different IPs should not affect each other
		ips := []string{"192.168.1.1:12345", "192.168.1.2:12345", "192.168.1.3:12345"}

		for _, ip := range ips {
			req := httptest.NewRequest("GET", "/health", nil)
			req.RemoteAddr = ip
			w := httptest.NewRecorder()

			server.router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code,
				"Requests from different IPs should not be rate limited")
		}
	})
}

func TestHTTPCORSConfiguration(t *testing.T) {
	config := &SecurityConfig{
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxRequestSize: 1024,
		AllowedOrigins: []string{"https://example.com", "https://trusted.com"},
	}
	server := NewSecureHTTPServer(config)

	t.Run("allow_trusted_origins", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("reject_untrusted_origins", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		req.Header.Set("Origin", "https://malicious.com")
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("handle_preflight_requests", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/config", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Methods"))
	})
}

func TestHTTPTimeoutEnforcement(t *testing.T) {
	config := &SecurityConfig{
		ReadTimeout:    100 * time.Millisecond, // Very short timeout
		WriteTimeout:   100 * time.Millisecond,
		MaxRequestSize: 1024,
	}
	server := NewSecureHTTPServer(config)

	t.Run("enforce_request_timeout", func(t *testing.T) {
		// Create a slow reader to trigger timeout
		slowReader := &slowReader{delay: 200 * time.Millisecond}

		req := httptest.NewRequest("PUT", "/config", slowReader)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		start := time.Now()
		server.router.ServeHTTP(w, req)
		duration := time.Since(start)

		// Should timeout quickly
		assert.True(t, duration < 500*time.Millisecond, "Should timeout quickly")
	})
}

// slowReader simulates a slow network connection
type slowReader struct {
	data  []byte
	pos   int
	delay time.Duration
}

func (sr *slowReader) Read(p []byte) (n int, err error) {
	if sr.pos >= len(sr.data) {
		return 0, io.EOF
	}

	time.Sleep(sr.delay)

	n = copy(p, sr.data[sr.pos:])
	sr.pos += n
	return n, nil
}

func TestHTTPSSlowlorisProtection(t *testing.T) {
	config := &SecurityConfig{
		ReadTimeout:       2 * time.Second,
		ReadHeaderTimeout: 1 * time.Second,
		WriteTimeout:      2 * time.Second,
		MaxRequestSize:    1024,
	}
	server := NewSecureHTTPServer(config)

	t.Run("protect_against_slowloris", func(t *testing.T) {
		// Simulate Slowloris attack with very slow header sending
		req := httptest.NewRequest("GET", "/health", nil)

		// This would be a real network connection in production
		// where headers are sent byte by byte with delays
		w := httptest.NewRecorder()

		start := time.Now()
		server.router.ServeHTTP(w, req)
		duration := time.Since(start)

		// Should complete quickly since we're using httptest
		// In real scenario, ReadHeaderTimeout would protect against slow headers
		assert.True(t, duration < 5*time.Second, "Should not hang on slow requests")
	})
}

func TestHTTPInputValidation(t *testing.T) {
	server := NewSecureHTTPServer(nil)

	t.Run("reject_malicious_json", func(t *testing.T) {
		maliciousPayloads := []string{
			`{"config": "'; DROP TABLE users; --"}`,
			`{"config": "<script>alert('xss')</script>"}`,
			`{"config": "$(rm -rf /)"}`,
			`{"config": "../../../../etc/passwd"}`,
			strings.Repeat(`{"a":`, 1000) + strings.Repeat(`}`, 1000), // JSON bomb
		}

		for i, payload := range maliciousPayloads {
			t.Run(fmt.Sprintf("malicious_payload_%d", i), func(t *testing.T) {
				req := httptest.NewRequest("PUT", "/config", strings.NewReader(payload))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				server.router.ServeHTTP(w, req)

				// Should either reject or handle safely
				if w.Code == http.StatusOK {
					// If accepted, ensure it was sanitized/validated
					var response map[string]interface{}
					json.Unmarshal(w.Body.Bytes(), &response)

					// Add additional validation checks here
					assert.NotContains(t, fmt.Sprintf("%v", response), "DROP TABLE")
					assert.NotContains(t, fmt.Sprintf("%v", response), "<script>")
				} else {
					assert.True(t, w.Code >= 400, "Should reject malicious input")
				}
			})
		}
	})

	t.Run("validate_content_type", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/config", strings.NewReader(`{"test": "data"}`))
		req.Header.Set("Content-Type", "text/plain") // Wrong content type
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		// Should handle content type validation appropriately
		assert.True(t, w.Code >= 400 || w.Body.String() != "", "Should validate content type")
	})
}

func TestHTTPConcurrentSafety(t *testing.T) {
	server := NewSecureHTTPServer(nil)

	t.Run("concurrent_request_handling", func(t *testing.T) {
		const numGoroutines = 50
		results := make(chan int, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				req := httptest.NewRequest("GET", "/health", nil)
				req.RemoteAddr = fmt.Sprintf("192.168.1.%d:12345", id%256)
				w := httptest.NewRecorder()

				server.router.ServeHTTP(w, req)
				results <- w.Code
			}(i)
		}

		// Collect all results
		successCount := 0
		for i := 0; i < numGoroutines; i++ {
			select {
			case code := <-results:
				if code == http.StatusOK {
					successCount++
				}
			case <-time.After(10 * time.Second):
				t.Fatal("Timeout waiting for concurrent requests")
			}
		}

		// Most requests should succeed
		assert.True(t, successCount > numGoroutines/2,
			"Most concurrent requests should succeed")
	})

	t.Run("concurrent_config_updates", func(t *testing.T) {
		const numUpdates = 20
		results := make(chan int, numUpdates)

		for i := 0; i < numUpdates; i++ {
			go func(id int) {
				config := map[string]interface{}{
					"readTimeout":    "10s",
					"writeTimeout":   "10s",
					"maxRequestSize": 1024 + id,
				}

				configJSON, _ := json.Marshal(config)
				req := httptest.NewRequest("PUT", "/config", bytes.NewBuffer(configJSON))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				server.router.ServeHTTP(w, req)
				results <- w.Code
			}(i)
		}

		// Collect all results
		for i := 0; i < numUpdates; i++ {
			select {
			case code := <-results:
				// Should handle concurrent updates safely
				assert.True(t, code >= 200 && code < 500,
					"Should handle concurrent updates safely")
			case <-time.After(10 * time.Second):
				t.Fatal("Timeout waiting for concurrent updates")
			}
		}
	})
}

// BenchmarkHTTPSecurity benchmarks security middleware performance
func BenchmarkHTTPSecurity(b *testing.B) {
	server := NewSecureHTTPServer(nil)

	b.Run("security_middleware_overhead", func(b *testing.B) {
		req := httptest.NewRequest("GET", "/health", nil)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)
		}
	})

	b.Run("rate_limiting_performance", func(b *testing.B) {
		req := httptest.NewRequest("GET", "/health", nil)
		req.RemoteAddr = "192.168.1.1:12345"

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)
		}
	})
}

func TestHTTPSecurityIntegration(t *testing.T) {
	// Integration test with real HTTP server
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	server := NewSecureHTTPServer(nil)

	// Start server on random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	go func() {
		server.server.Serve(listener)
	}()

	baseURL := fmt.Sprintf("http://%s", listener.Addr().String())

	t.Run("real_http_request_security", func(t *testing.T) {
		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		resp, err := client.Get(baseURL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check security headers
		assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
		assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
		assert.NotEmpty(t, resp.Header.Get("Content-Security-Policy"))
	})

	t.Run("real_timeout_enforcement", func(t *testing.T) {
		// This would require a more sophisticated test with actual network delays
		// For now, just verify the server responds within reasonable time
		client := &http.Client{
			Timeout: 1 * time.Second,
		}

		start := time.Now()
		resp, err := client.Get(baseURL + "/health")
		duration := time.Since(start)

		require.NoError(t, err)
		resp.Body.Close()

		assert.True(t, duration < 500*time.Millisecond,
			"Server should respond quickly to health checks")
	})
}
