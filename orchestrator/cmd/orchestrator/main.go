package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/orchestrator/pkg/placement"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
)

const (
	appName = "orchestrator"
	version = "v0.1.0"
)

var (
	// Prometheus metrics for O-RAN orchestrator
	intentProcessingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "oran_intent_processing_duration_seconds",
			Help: "Time taken to process intent-based requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"intent_type", "status"},
	)

	sliceDeploymentsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oran_slice_deployments_total",
			Help: "Total number of network slice deployments",
		},
		[]string{"slice_type", "status"},
	)

	activeSlicesGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "oran_active_slices",
			Help: "Number of currently active network slices",
		},
		[]string{"slice_type"},
	)

	placementDecisionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oran_placement_decisions_total",
			Help: "Total number of placement decisions made",
		},
		[]string{"placement_type", "result"},
	)
)

func init() {
	// Register metrics with Prometheus
	prometheus.MustRegister(intentProcessingDuration)
	prometheus.MustRegister(sliceDeploymentsTotal)
	prometheus.MustRegister(activeSlicesGauge)
	prometheus.MustRegister(placementDecisionsTotal)
}

// QoSIntent represents a parsed intent with QoS parameters
type QoSIntent struct {
	Bandwidth   float64 `json:"bandwidth"`
	Latency     float64 `json:"latency"`
	SliceType   string  `json:"slice_type,omitempty"`
	Jitter      float64 `json:"jitter,omitempty"`
	PacketLoss  float64 `json:"packet_loss,omitempty"`
	Reliability float64 `json:"reliability,omitempty"`
}

// SliceAllocation represents the orchestrated slice deployment
type SliceAllocation struct {
	SliceID   string             `json:"slice_id"`
	QoS       QoSIntent          `json:"qos"`
	Placement placement.Decision `json:"placement"`
	Resources ResourceAllocation `json:"resources"`
	Status    string             `json:"status"`
}

// ResourceAllocation represents allocated resources
type ResourceAllocation struct {
	RANResources map[string]interface{} `json:"ran_resources"`
	CNResources  map[string]interface{} `json:"cn_resources"`
	TNResources  map[string]interface{} `json:"tn_resources"`
}

// Config holds orchestrator configuration
type Config struct {
	PlanMode     bool
	ApplyMode    bool
	ServerMode   bool
	InputFile    string
	OutputFile   string
	Verbose      bool
	DryRun       bool
	ServerPort   string
	MetricsPort  string
}

func main() {
	config := parseFlags()

	if config.Verbose {
		log.Printf("%s %s starting", appName, version)
	}

	// Server mode for Kubernetes deployment
	if config.ServerMode {
		startServer(config)
		return
	}

	// Load QoS intents from input file
	intents, err := loadQoSIntents(config.InputFile)
	if err != nil {
		log.Fatalf("Failed to load QoS intents: %v", err)
	}

	if config.Verbose {
		log.Printf("Loaded %d QoS intents", len(intents))
	}

	// Process intents based on mode
	if config.PlanMode {
		err = planSliceOrchestration(intents, config)
	} else if config.ApplyMode {
		err = applySliceOrchestration(intents, config)
	} else {
		log.Fatal("Must specify either --plan, --apply, or --server mode")
	}

	if err != nil {
		log.Fatalf("Orchestration failed: %v", err)
	}

	if config.Verbose {
		log.Printf("Orchestration completed successfully")
	}
}

func parseFlags() Config {
	var config Config

	flag.BoolVar(&config.PlanMode, "plan", false, "Generate orchestration plan (dry-run)")
	flag.BoolVar(&config.ApplyMode, "apply", false, "Apply orchestration to deploy slices")
	flag.BoolVar(&config.ServerMode, "server", false, "Run as HTTP server for Kubernetes deployment")
	flag.StringVar(&config.InputFile, "input", "", "Input file with QoS intents (JSONL format)")
	flag.StringVar(&config.OutputFile, "output", "", "Output file for orchestration results")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&config.DryRun, "dry-run", false, "Show what would be done without executing")
	flag.StringVar(&config.ServerPort, "port", "8080", "HTTP server port")
	flag.StringVar(&config.MetricsPort, "metrics-port", "9090", "Metrics server port")

	help := flag.Bool("help", false, "Show help message")
	versionFlag := flag.Bool("version", false, "Show version")

	flag.Parse()

	if *help {
		fmt.Printf("%s %s - O-RAN Intent-Based MANO Orchestrator\n\n", appName, version)
		fmt.Println("Usage:")
		fmt.Printf("  %s --plan <input-file>     # Generate orchestration plan\n", appName)
		fmt.Printf("  %s --apply <input-file>    # Apply orchestration\n", appName)
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Printf("  %s --plan artifacts/qos_intent.jsonl\n", appName)
		fmt.Printf("  %s --apply artifacts/qos_intent.jsonl --verbose\n", appName)
		os.Exit(0)
	}

	if *versionFlag {
		fmt.Printf("%s %s\n", appName, version)
		os.Exit(0)
	}

	// Validation
	if !config.PlanMode && !config.ApplyMode && !config.ServerMode {
		log.Fatal("Must specify either --plan, --apply, or --server mode")
	}

	if (config.PlanMode && config.ApplyMode) || (config.PlanMode && config.ServerMode) || (config.ApplyMode && config.ServerMode) {
		log.Fatal("Cannot specify multiple modes simultaneously")
	}

	// Server mode doesn't need input file
	if !config.ServerMode {
		if config.InputFile == "" {
			if len(flag.Args()) > 0 {
				config.InputFile = flag.Args()[0]
			} else {
				log.Fatal("Input file is required for plan/apply modes")
			}
		}
	}

	// Default output file
	if config.OutputFile == "" {
		base := filepath.Base(config.InputFile)
		ext := filepath.Ext(base)
		name := base[:len(base)-len(ext)]

		if config.PlanMode {
			config.OutputFile = fmt.Sprintf("artifacts/%s_plan.json", name)
		} else {
			config.OutputFile = fmt.Sprintf("artifacts/%s_deployment.json", name)
		}
	}

	return config
}

func loadQoSIntents(filename string) ([]QoSIntent, error) {
	// Create validator for input files
	validator := security.CreateValidatorForConfig(".")

	// Validate file path for security
	if err := validator.ValidateFilePathAndExtension(filename, []string{".jsonl", ".json", ".txt"}); err != nil {
		return nil, fmt.Errorf("input file path validation failed: %w", err)
	}

	file, err := validator.SafeOpenFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Error closing file %s: %v", filename, err)
		}
	}()

	var intents []QoSIntent
	scanner := bufio.NewScanner(file)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		var intent QoSIntent
		if err := json.Unmarshal([]byte(line), &intent); err != nil {
			return nil, fmt.Errorf("failed to parse QoS intent on line %d: %w", lineNum, err)
		}

		intents = append(intents, intent)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return intents, nil
}

func planSliceOrchestration(intents []QoSIntent, config Config) error {
	log.Printf("Planning orchestration for %d intents...", len(intents))

	allocations := make([]SliceAllocation, 0, len(intents))

	for i, intent := range intents {
		sliceID := fmt.Sprintf("slice-%s-%03d", intent.SliceType, i+1)

		if config.Verbose {
			log.Printf("Planning slice %s with QoS: BW=%.2f, Lat=%.1f",
				sliceID, intent.Bandwidth, intent.Latency)
		}

		// Generate placement decision
		nf := &placement.NetworkFunction{
			ID:              sliceID,
			Type:            intent.SliceType,
			Requirements:    generateResourceRequirements(intent),
			QoSRequirements: convertToPlacementQoS(intent),
		}

		// Use mock sites for demonstration
		sites := generateMockSites()
		policy := placement.NewLatencyAwarePolicy(placement.NewMockMetricsProvider())

		placementDecision, err := policy.Place(nf, sites)
		if err != nil {
			return fmt.Errorf("placement generation failed for %s: %w", sliceID, err)
		}

		// Create slice allocation
		allocation := SliceAllocation{
			SliceID:   sliceID,
			QoS:       intent,
			Placement: *placementDecision,
			Resources: generateResourceAllocation(intent),
			Status:    "planned",
		}

		allocations = append(allocations, allocation)
	}

	// Save orchestration plan
	return saveOrchestrationPlan(allocations, config.OutputFile, config.Verbose)
}

func applySliceOrchestration(intents []QoSIntent, config Config) error {
	log.Printf("Applying orchestration for %d intents...", len(intents))

	// First, load the plan if it exists
	planFile := config.OutputFile
	if filepath.Ext(planFile) != ".json" {
		base := filepath.Base(config.InputFile)
		ext := filepath.Ext(base)
		name := base[:len(base)-len(ext)]
		planFile = fmt.Sprintf("artifacts/%s_plan.json", name)
	}

	allocations, err := loadOrchestrationPlan(planFile)
	if err != nil {
		log.Printf("No existing plan found, generating new plan...")
		err = planSliceOrchestration(intents, config)
		if err != nil {
			return fmt.Errorf("failed to generate plan: %w", err)
		}
		allocations, err = loadOrchestrationPlan(planFile)
		if err != nil {
			return fmt.Errorf("failed to load generated plan: %w", err)
		}
	}

	// Apply each allocation
	for i := range allocations {
		allocation := &allocations[i]

		if config.Verbose {
			log.Printf("Deploying slice %s...", allocation.SliceID)
		}

		if config.DryRun {
			log.Printf("DRY RUN: Would deploy slice %s", allocation.SliceID)
			allocation.Status = "dry-run"
		} else {
			deploySlice(allocation, config)
			allocation.Status = "deployed"
			if config.Verbose {
				log.Printf("Successfully deployed slice %s", allocation.SliceID)
			}
		}
	}

	// Save deployment results
	deploymentFile := config.OutputFile
	if filepath.Base(deploymentFile) == filepath.Base(planFile) {
		base := filepath.Base(config.InputFile)
		ext := filepath.Ext(base)
		name := base[:len(base)-len(ext)]
		deploymentFile = fmt.Sprintf("artifacts/%s_deployment.json", name)
	}

	return saveOrchestrationPlan(allocations, deploymentFile, config.Verbose)
}

func convertToPlacementQoS(intent QoSIntent) placement.QoSRequirements {
	// Convert QoS intent to placement requirements
	// For latency, use a higher tolerance for placement (multiply by 2)
	maxLatency := intent.Latency * 2
	if maxLatency < 1 {
		maxLatency = 20 // Default reasonable latency for uRLLC
	}

	return placement.QoSRequirements{
		MaxLatencyMs:      maxLatency,
		MinThroughputMbps: intent.Bandwidth,
		MaxPacketLossRate: 0.01, // 1% default if not specified
		MaxJitterMs:       10.0, // Default reasonable jitter
	}
}

func generateResourceRequirements(intent QoSIntent) placement.ResourceRequirements {
	// Resource scaling based on QoS requirements - use reasonable minimums
	cpuCores := int(intent.Bandwidth * 0.1) // 0.1 cores per Mbps
	if cpuCores < 1 {
		cpuCores = 1
	}

	memory := int(intent.Bandwidth * 0.1) // 100MB per Mbps
	if memory < 1 {
		memory = 1
	}

	storage := int(intent.Bandwidth * 0.5) // 500MB per Mbps
	if storage < 1 {
		storage = 1
	}

	return placement.ResourceRequirements{
		MinCPUCores:      cpuCores,
		MinMemoryGB:      memory,
		MinStorageGB:     storage,
		MinBandwidthMbps: intent.Bandwidth,
	}
}

func generateResourceAllocation(intent QoSIntent) ResourceAllocation {
	// Generate resource allocation based on slice type and QoS
	ranResources := map[string]interface{}{
		"cpu_cores":     intent.Bandwidth * 0.3,
		"memory_mb":     intent.Bandwidth * 128,
		"antennas":      int(intent.Bandwidth),
		"frequency_mhz": 3500 + (intent.Bandwidth * 100),
	}

	cnResources := map[string]interface{}{
		"cpu_cores":    intent.Bandwidth * 0.4,
		"memory_mb":    intent.Bandwidth * 256,
		"storage_gb":   intent.Bandwidth * 2,
		"upf_capacity": intent.Bandwidth * 10,
	}

	tnResources := map[string]interface{}{
		"bandwidth_mbps":    intent.Bandwidth,
		"vlan_id":           1000 + int(intent.Bandwidth*10),
		"qos_class":         intent.SliceType,
		"latency_budget_ms": intent.Latency,
	}

	return ResourceAllocation{
		RANResources: ranResources,
		CNResources:  cnResources,
		TNResources:  tnResources,
	}
}

func deploySlice(allocation *SliceAllocation, config Config) {
	// Simulated deployment - in real implementation this would:
	// 1. Deploy RAN components via O-RAN O2 DMS
	// 2. Deploy CN components via Nephio/GitOps
	// 3. Configure TN with VXLAN/TC bandwidth controls
	// 4. Setup inter-site connectivity with Kube-OVN

	if config.Verbose {
		log.Printf("Deploying RAN resources for %s", allocation.SliceID)
		log.Printf("Deploying CN resources for %s", allocation.SliceID)
		log.Printf("Configuring TN bandwidth controls for %s", allocation.SliceID)
		log.Printf("Setting up inter-site connectivity for %s", allocation.SliceID)
	}

	// Simulate deployment time
	// In real implementation, this would orchestrate actual deployments
}

func saveOrchestrationPlan(allocations []SliceAllocation, filename string, verbose bool) error {
	// Create validator for output files
	validator := security.CreateValidatorForConfig(".")

	// Validate file path for security
	if err := validator.ValidateFilePathAndExtension(filename, []string{".json"}); err != nil {
		return fmt.Errorf("output file path validation failed: %w", err)
	}

	// Ensure output directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, security.SecureDirMode); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := security.SecureCreateFile(filename)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Error closing file %s: %v", filename, err)
		}
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	plan := map[string]interface{}{
		"timestamp":    fmt.Sprintf("%d", os.Getpid()), // Simple timestamp simulation
		"allocations":  allocations,
		"total_slices": len(allocations),
	}

	if err := encoder.Encode(plan); err != nil {
		return fmt.Errorf("failed to encode orchestration plan: %w", err)
	}

	if verbose {
		log.Printf("Orchestration plan saved to: %s", filename)
	}

	return nil
}

func loadOrchestrationPlan(filename string) ([]SliceAllocation, error) {
	// Create validator for plan files
	validator := security.CreateValidatorForConfig(".")

	// Validate file path for security
	if err := validator.ValidateFilePathAndExtension(filename, []string{".json"}); err != nil {
		return nil, fmt.Errorf("plan file path validation failed: %w", err)
	}

	file, err := validator.SafeOpenFile(filename)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Error closing file %s: %v", filename, err)
		}
	}()

	var plan map[string]interface{}
	if err := json.NewDecoder(file).Decode(&plan); err != nil {
		return nil, err
	}

	allocationsData, ok := plan["allocations"]
	if !ok {
		return nil, fmt.Errorf("no allocations found in plan file")
	}

	// Convert back to SliceAllocation structs
	allocationsJSON, err := json.Marshal(allocationsData)
	if err != nil {
		return nil, err
	}

	var allocations []SliceAllocation
	if err := json.Unmarshal(allocationsJSON, &allocations); err != nil {
		return nil, err
	}

	return allocations, nil
}

// generateMockSites creates sample sites for placement testing
func generateMockSites() []*placement.Site {
	return []*placement.Site{
		{
			ID:   "edge01",
			Name: "Edge Site 01",
			Type: placement.CloudTypeEdge,
			Location: placement.Location{
				Latitude:  40.7128,
				Longitude: -74.0060,
				Region:    "us-east",
				Zone:      "us-east-1a",
			},
			Capacity: placement.ResourceCapacity{
				CPUCores:      16,
				MemoryGB:      64,
				StorageGB:     1000,
				BandwidthMbps: 10,
			},
			NetworkProfile: placement.NetworkProfile{
				BaseLatencyMs:     1.0,
				MaxThroughputMbps: 5.0,
				PacketLossRate:    0.001,
				JitterMs:          0.5,
			},
			Available: true,
		},
		{
			ID:   "regional01",
			Name: "Regional Site 01",
			Type: placement.CloudTypeRegional,
			Location: placement.Location{
				Latitude:  40.7589,
				Longitude: -73.9851,
				Region:    "us-east",
				Zone:      "us-east-1b",
			},
			Capacity: placement.ResourceCapacity{
				CPUCores:      64,
				MemoryGB:      256,
				StorageGB:     5000,
				BandwidthMbps: 50,
			},
			NetworkProfile: placement.NetworkProfile{
				BaseLatencyMs:     5.0,
				MaxThroughputMbps: 20.0,
				PacketLossRate:    0.0001,
				JitterMs:          1.0,
			},
			Available: true,
		},
		{
			ID:   "central01",
			Name: "Central Site 01",
			Type: placement.CloudTypeCentral,
			Location: placement.Location{
				Latitude:  39.0458,
				Longitude: -76.6413,
				Region:    "us-east",
				Zone:      "us-east-1c",
			},
			Capacity: placement.ResourceCapacity{
				CPUCores:      256,
				MemoryGB:      1024,
				StorageGB:     20000,
				BandwidthMbps: 100,
			},
			NetworkProfile: placement.NetworkProfile{
				BaseLatencyMs:     10.0,
				MaxThroughputMbps: 50.0,
				PacketLossRate:    0.00001,
				JitterMs:          2.0,
			},
			Available: true,
		},
	}
}

// startServer starts the HTTP server for Kubernetes deployment
func startServer(config Config) {
	log.Printf("Starting %s %s server on port %s", appName, version, config.ServerPort)

	// Create HTTP server mux
	mux := http.NewServeMux()

	// Health check endpoints
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ready", readinessHandler)

	// Intent processing endpoints
	mux.HandleFunc("/api/v1/intents", intentsHandler)
	mux.HandleFunc("/api/v1/slices", slicesHandler)
	mux.HandleFunc("/api/v1/status", statusHandler)

	// Create main HTTP server
	server := &http.Server{
		Addr:    ":" + config.ServerPort,
		Handler: mux,
	}

	// Create metrics server
	metricsServer := &http.Server{
		Addr:    ":" + config.MetricsPort,
		Handler: promhttp.Handler(),
	}

	// Start metrics server in goroutine
	go func() {
		log.Printf("Starting metrics server on port %s", config.MetricsPort)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Metrics server failed: %v", err)
		}
	}()

	// Start main server in goroutine
	go func() {
		log.Printf("Starting HTTP server on port %s", config.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Initialize some mock active slices for demonstration
	activeSlicesGauge.WithLabelValues("embb").Set(2)
	activeSlicesGauge.WithLabelValues("urllc").Set(1)
	activeSlicesGauge.WithLabelValues("mmtc").Set(3)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down servers...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	if err := metricsServer.Shutdown(ctx); err != nil {
		log.Printf("Metrics server shutdown error: %v", err)
	}

	log.Println("Servers stopped")
}

// healthHandler responds to health check requests
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   version,
	})
}

// readinessHandler responds to readiness check requests
func readinessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now().Unix(),
		"version":   version,
	})
}

// intentsHandler handles intent processing requests
func intentsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var status string = "success"

	defer func() {
		duration := time.Since(start).Seconds()
		intentProcessingDuration.WithLabelValues("api", status).Observe(duration)
	}()

	switch r.Method {
	case http.MethodPost:
		handleIntentCreation(w, r)
	case http.MethodGet:
		handleIntentList(w, r)
	default:
		status = "error"
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleIntentCreation processes new intent creation requests
func handleIntentCreation(w http.ResponseWriter, r *http.Request) {
	var intent QoSIntent
	if err := json.NewDecoder(r.Body).Decode(&intent); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Simulate intent processing
	sliceID := fmt.Sprintf("slice-%s-%d", intent.SliceType, time.Now().Unix())

	// Update metrics
	sliceDeploymentsTotal.WithLabelValues(intent.SliceType, "success").Inc()
	placementDecisionsTotal.WithLabelValues("edge", "success").Inc()

	// Increment active slices
	activeSlicesGauge.WithLabelValues(intent.SliceType).Inc()

	response := map[string]interface{}{
		"slice_id": sliceID,
		"status":   "created",
		"qos":      intent,
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleIntentList returns list of active intents
func handleIntentList(w http.ResponseWriter, r *http.Request) {
	// Mock response with current active slices
	slices := []map[string]interface{}{
		{
			"slice_id":   "slice-embb-001",
			"slice_type": "embb",
			"status":     "active",
			"qos": map[string]interface{}{
				"bandwidth": 100.0,
				"latency":   20.0,
			},
		},
		{
			"slice_id":   "slice-urllc-001",
			"slice_type": "urllc",
			"status":     "active",
			"qos": map[string]interface{}{
				"bandwidth": 10.0,
				"latency":   1.0,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"slices": slices,
		"total":  len(slices),
	})
}

// slicesHandler handles slice management requests
func slicesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleSliceList(w, r)
	case http.MethodDelete:
		handleSliceDelete(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSliceList returns list of all slices
func handleSliceList(w http.ResponseWriter, r *http.Request) {
	// This would query actual slice deployments in real implementation
	slices := []map[string]interface{}{
		{
			"slice_id":   "slice-embb-001",
			"slice_type": "embb",
			"status":     "active",
			"deployment_time": time.Now().Add(-1 * time.Hour).Unix(),
		},
		{
			"slice_id":   "slice-urllc-001",
			"slice_type": "urllc",
			"status":     "active",
			"deployment_time": time.Now().Add(-30 * time.Minute).Unix(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"slices": slices,
		"total":  len(slices),
	})
}

// handleSliceDelete handles slice deletion
func handleSliceDelete(w http.ResponseWriter, r *http.Request) {
	sliceID := r.URL.Query().Get("slice_id")
	if sliceID == "" {
		http.Error(w, "slice_id parameter required", http.StatusBadRequest)
		return
	}

	// Mock deletion - would interact with actual slice management in real implementation
	// For demo, just decrement the gauge for a random slice type
	activeSlicesGauge.WithLabelValues("embb").Dec()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"slice_id": sliceID,
		"status":   "deleted",
		"timestamp": time.Now().Unix(),
	})
}

// statusHandler returns overall system status
func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service":   appName,
		"version":   version,
		"status":    "running",
		"timestamp": time.Now().Unix(),
		"uptime":    time.Since(time.Now()).Seconds(), // Mock uptime
	})
}
