package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tests/framework/dashboard"
)

var (
	configPath    = flag.String("config", "tests/framework/dashboard/config.yaml", "Path to dashboard configuration file")
	outputPath    = flag.String("output", "", "Path to output HTML file (static mode)")
	serve         = flag.Bool("serve", false, "Start HTTP server")
	port          = flag.Int("port", 8080, "HTTP server port")
	debug         = flag.Bool("debug", false, "Enable debug logging")
	version       = flag.Bool("version", false, "Show version information")
	aggregateOnly = flag.Bool("aggregate-only", false, "Run metrics aggregation only")
)

const (
	Version   = "1.0.0"
	BuildTime = "2024-01-15T10:00:00Z"
	GitCommit = "a421f45"
)

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("O-RAN Intent-MANO Test Dashboard\n")
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		return
	}

	if *debug {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Println("Debug logging enabled")
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown gracefully
	setupGracefulShutdown(cancel)

	if *aggregateOnly {
		runAggregationOnly(ctx)
		return
	}

	if *serve {
		runServer(ctx)
		return
	}

	// Default: generate static dashboard
	generateStaticDashboard()
}

// setupGracefulShutdown sets up graceful shutdown handling
func setupGracefulShutdown(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Received shutdown signal, gracefully shutting down...")
		cancel()
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()
}

// runAggregationOnly runs only the metrics aggregation service
func runAggregationOnly(ctx context.Context) {
	log.Println("Starting metrics aggregation service...")

	aggregator, err := dashboard.NewMetricsAggregator(*configPath)
	if err != nil {
		log.Fatalf("Failed to create metrics aggregator: %v", err)
	}

	if err := aggregator.Start(ctx); err != nil {
		log.Fatalf("Failed to start metrics aggregator: %v", err)
	}

	log.Println("Metrics aggregation service started successfully")

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("Shutting down metrics aggregation service...")
	aggregator.Stop()
}

// runServer runs the dashboard HTTP server
func runServer(ctx context.Context) {
	log.Printf("Starting dashboard server on port %d...", *port)

	// Create dashboard instance
	dashboardInstance, err := dashboard.NewDashboard(*configPath)
	if err != nil {
		log.Fatalf("Failed to create dashboard: %v", err)
	}

	// Create metrics aggregator
	aggregator, err := dashboard.NewMetricsAggregator(*configPath)
	if err != nil {
		log.Fatalf("Failed to create metrics aggregator: %v", err)
	}

	// Start metrics aggregator
	if err := aggregator.Start(ctx); err != nil {
		log.Fatalf("Failed to start metrics aggregator: %v", err)
	}

	// Setup HTTP routes
	setupRoutes(dashboardInstance, aggregator)

	// Start HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Dashboard server listening on http://localhost:%d", *port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Shutdown server gracefully
	log.Println("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	// Stop metrics aggregator
	aggregator.Stop()
	log.Println("Server shutdown complete")
}

// setupRoutes sets up HTTP routes for the dashboard
func setupRoutes(dashboardInstance *dashboard.Dashboard, aggregator *dashboard.MetricsAggregator) {
	// Dashboard routes
	http.HandleFunc("/", handleDashboard(dashboardInstance))
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/metrics", handleMetrics(aggregator))
	http.HandleFunc("/alerts", handleAlerts(aggregator))
	http.HandleFunc("/history", handleHistory(aggregator))

	// API routes
	http.HandleFunc("/api/metrics", handleAPIMetrics(aggregator))
	http.HandleFunc("/api/alerts", handleAPIAlerts(aggregator))
	http.HandleFunc("/api/history", handleAPIHistory(aggregator))
	http.HandleFunc("/api/refresh", handleAPIRefresh(dashboardInstance, aggregator))
	http.HandleFunc("/api/config", handleAPIConfig)

	// WebSocket route for real-time updates
	http.HandleFunc("/ws", handleWebSocket(aggregator))

	// Static file serving
	fs := http.FileServer(http.Dir("tests/framework/dashboard/static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Export routes
	http.HandleFunc("/export/json", handleExportJSON(aggregator))
	http.HandleFunc("/export/csv", handleExportCSV(aggregator))
	http.HandleFunc("/export/pdf", handleExportPDF(aggregator))
}

// handleDashboard handles the main dashboard page
func handleDashboard(dashboardInstance *dashboard.Dashboard) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := dashboardInstance.LoadMetrics(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to load metrics: %v", err), http.StatusInternalServerError)
			return
		}

		// Set cache headers
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		// Create secure temporary file for HTML output
		tmpDir := os.TempDir()
		tmpFile, err := security.SecureJoinPath(tmpDir, "o-ran-dashboard.html")
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create secure temp path: %v", err), http.StatusInternalServerError)
			return
		}

		if err := dashboardInstance.GenerateHTML(tmpFile); err != nil {
			http.Error(w, fmt.Sprintf("Failed to generate dashboard: %v", err), http.StatusInternalServerError)
			return
		}

		// Serve the generated HTML file
		http.ServeFile(w, r, tmpFile)
	}
}

// handleHealth handles health check requests
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{
		"status": "healthy",
		"version": "%s",
		"timestamp": "%s"
	}`, Version, time.Now().Format(time.RFC3339))
}

// handleMetrics handles metrics endpoint
func handleMetrics(aggregator *dashboard.MetricsAggregator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics := aggregator.GetCurrentMetrics()
		if metrics == nil {
			http.Error(w, "No metrics available", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(metrics); err != nil {
			http.Error(w, fmt.Sprintf("JSON encoding error: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

// handleAlerts handles alerts endpoint
func handleAlerts(aggregator *dashboard.MetricsAggregator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		alerts := aggregator.GetActiveAlerts()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(alerts); err != nil {
			http.Error(w, fmt.Sprintf("JSON encoding error: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

// handleHistory handles metrics history endpoint
func handleHistory(aggregator *dashboard.MetricsAggregator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limitStr := r.URL.Query().Get("limit")
		limit := 10 // default
		if limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		history := aggregator.GetMetricsHistory(limit)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(history); err != nil {
			http.Error(w, fmt.Sprintf("JSON encoding error: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

// handleAPIMetrics handles API metrics endpoint
func handleAPIMetrics(aggregator *dashboard.MetricsAggregator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleMetrics(aggregator)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// handleAPIAlerts handles API alerts endpoint
func handleAPIAlerts(aggregator *dashboard.MetricsAggregator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleAlerts(aggregator)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// handleAPIHistory handles API history endpoint
func handleAPIHistory(aggregator *dashboard.MetricsAggregator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleHistory(aggregator)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// handleAPIRefresh handles API refresh endpoint
func handleAPIRefresh(dashboardInstance *dashboard.Dashboard, aggregator *dashboard.MetricsAggregator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Trigger metrics refresh
		if err := dashboardInstance.LoadMetrics(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to refresh metrics: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"status": "success",
			"message": "Metrics refreshed successfully",
			"timestamp": "%s"
		}`, time.Now().Format(time.RFC3339))
	}
}

// handleAPIConfig handles API config endpoint
func handleAPIConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	config := map[string]interface{}{
		"version":    Version,
		"build_time": BuildTime,
		"git_commit": GitCommit,
		"features": map[string]bool{
			"real_time":    true,
			"export":       true,
			"alerts":       true,
			"history":      true,
			"aggregation":  true,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(config); err != nil {
		http.Error(w, fmt.Sprintf("JSON encoding error: %v", err), http.StatusInternalServerError)
		return
	}
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// handleWebSocket handles WebSocket connections for real-time updates
func handleWebSocket(aggregator *dashboard.MetricsAggregator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		// Generate unique subscriber ID
		subscriberID := fmt.Sprintf("ws_%d", time.Now().UnixNano())

		// Get filters from query parameters
		filters := r.URL.Query()["filter"]

		// Add subscriber
		aggregator.AddSubscriber(subscriberID, conn, filters)
		defer aggregator.RemoveSubscriber(subscriberID)

		// Send current metrics immediately
		currentMetrics := aggregator.GetCurrentMetrics()
		if currentMetrics != nil {
			if err := conn.WriteJSON(map[string]interface{}{
				"type":      "initial_metrics",
				"timestamp": time.Now(),
				"data":      currentMetrics,
			}); err != nil {
				log.Printf("Error sending initial metrics: %v", err)
				return
			}
		}

		// Keep connection alive and handle ping/pong
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("WebSocket ping failed: %v", err)
					return
				}
			}
		}
	}
}

// Export handlers
func handleExportJSON(aggregator *dashboard.MetricsAggregator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics := aggregator.GetCurrentMetrics()
		if metrics == nil {
			http.Error(w, "No metrics available", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=metrics_%s.json",
			time.Now().Format("20060102_150405")))

		if err := json.NewEncoder(w).Encode(metrics); err != nil {
			http.Error(w, fmt.Sprintf("JSON encoding error: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

func handleExportCSV(aggregator *dashboard.MetricsAggregator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics := aggregator.GetCurrentMetrics()
		if metrics == nil {
			http.Error(w, "No metrics available", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=metrics_%s.csv",
			time.Now().Format("20060102_150405")))

		// Generate CSV content (simplified)
		csv := "Metric,Value,Timestamp\n"
		csv += fmt.Sprintf("Overall Coverage,%.2f%%,%s\n",
			metrics.CoverageResults.OverallCoverage, metrics.Timestamp.Format(time.RFC3339))

		if _, err := w.Write([]byte(csv)); err != nil {
			log.Printf("Error writing CSV: %v", err)
		}
	}
}

func handleExportPDF(aggregator *dashboard.MetricsAggregator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// PDF export would require additional dependencies
		http.Error(w, "PDF export not implemented", http.StatusNotImplemented)
	}
}

// generateStaticDashboard generates a static HTML dashboard file
func generateStaticDashboard() {
	log.Println("Generating static dashboard...")

	dashboardInstance, err := dashboard.NewDashboard(*configPath)
	if err != nil {
		log.Fatalf("Failed to create dashboard: %v", err)
	}

	if err := dashboardInstance.LoadMetrics(); err != nil {
		log.Fatalf("Failed to load metrics: %v", err)
	}

	outputFile := *outputPath
	if outputFile == "" {
		outputFile = "reports/dashboard.html"
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputFile), 0750); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	if err := dashboardInstance.GenerateHTML(outputFile); err != nil {
		log.Fatalf("Failed to generate dashboard: %v", err)
	}

	log.Printf("Static dashboard generated: %s", outputFile)
}