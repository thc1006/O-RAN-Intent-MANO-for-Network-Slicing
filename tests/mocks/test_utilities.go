package mocks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// MockFactory provides factory methods for creating mock objects
type MockFactory struct{}

// NewMockFactory creates a new mock factory
func NewMockFactory() *MockFactory {
	return &MockFactory{}
}

// Create mock HTTP client with pre-configured responses
func (f *MockFactory) CreateHTTPClientWithResponses(responses map[string]*http.Response) *MockHTTPClient {
	client := &MockHTTPClient{}
	client.DoFunc = func(req *http.Request) (*http.Response, error) {
		key := fmt.Sprintf("%s %s", req.Method, req.URL.Path)
		if response, exists := responses[key]; exists {
			return response, nil
		}
		return CreateHTTPResponse(404, `{"error": "Not found"}`, nil), nil
	}
	return client
}

// Create mock metrics collector with realistic data
func (f *MockFactory) CreateMetricsCollectorWithScenario(scenario string) *MockMetricsCollector {
	collector := &MockMetricsCollector{}

	switch scenario {
	case "optimal":
		collector.CollectLatencyFunc = func(ctx context.Context, target string) (*LatencyMetrics, error) {
			return CreateLowLatencyMetrics(target), nil
		}
		collector.CollectThroughputFunc = func(ctx context.Context, target string) (*ThroughputMetrics, error) {
			return CreateHighThroughputMetrics(target), nil
		}
		collector.CollectResourceFunc = func(ctx context.Context, target string) (*ResourceMetrics, error) {
			return CreateLowResourceUsageMetrics(target), nil
		}

	case "congested":
		collector.CollectLatencyFunc = func(ctx context.Context, target string) (*LatencyMetrics, error) {
			return CreateHighLatencyMetrics(target), nil
		}
		collector.CollectThroughputFunc = func(ctx context.Context, target string) (*ThroughputMetrics, error) {
			return CreateLowThroughputMetrics(target), nil
		}
		collector.CollectResourceFunc = func(ctx context.Context, target string) (*ResourceMetrics, error) {
			return CreateHighResourceUsageMetrics(target), nil
		}

	case "edge":
		collector.CollectResourceFunc = func(ctx context.Context, target string) (*ResourceMetrics, error) {
			return CreateEdgeZoneMetrics(target), nil
		}
		collector.CollectLatencyFunc = func(ctx context.Context, target string) (*LatencyMetrics, error) {
			return CreateLowLatencyMetrics(target), nil
		}

	case "cloud":
		collector.CollectResourceFunc = func(ctx context.Context, target string) (*ResourceMetrics, error) {
			return CreateCloudZoneMetrics(target), nil
		}
		collector.CollectLatencyFunc = func(ctx context.Context, target string) (*LatencyMetrics, error) {
			return CreateDefaultLatencyMetrics(target), nil
		}

	default:
		collector.CollectLatencyFunc = func(ctx context.Context, target string) (*LatencyMetrics, error) {
			return CreateDefaultLatencyMetrics(target), nil
		}
		collector.CollectThroughputFunc = func(ctx context.Context, target string) (*ThroughputMetrics, error) {
			return CreateDefaultThroughputMetrics(target), nil
		}
		collector.CollectResourceFunc = func(ctx context.Context, target string) (*ResourceMetrics, error) {
			return CreateDefaultResourceMetrics(target), nil
		}
	}

	return collector
}

// Create mock Porch client with test packages
func (f *MockFactory) CreatePorchClientWithPackages(packages []*NephioPackage) *MockPorchClient {
	client := &MockPorchClient{}
	packageStore := make(map[string]*NephioPackage)

	// Populate package store
	for _, pkg := range packages {
		packageStore[pkg.Name] = pkg
	}

	client.CreatePackageFunc = func(ctx context.Context, pkg *NephioPackage) error {
		packageStore[pkg.Name] = pkg
		return nil
	}

	client.GetPackageFunc = func(ctx context.Context, name string) (*NephioPackage, error) {
		if pkg, exists := packageStore[name]; exists {
			return pkg, nil
		}
		return nil, fmt.Errorf("package not found: %s", name)
	}

	client.ListPackagesFunc = func(ctx context.Context) ([]*NephioPackage, error) {
		var result []*NephioPackage
		for _, pkg := range packageStore {
			result = append(result, pkg)
		}
		return result, nil
	}

	client.UpdatePackageFunc = func(ctx context.Context, name string, pkg *NephioPackage) error {
		if _, exists := packageStore[name]; exists {
			packageStore[name] = pkg
			return nil
		}
		return fmt.Errorf("package not found: %s", name)
	}

	client.DeletePackageFunc = func(ctx context.Context, name string) error {
		if _, exists := packageStore[name]; exists {
			delete(packageStore, name)
			return nil
		}
		return fmt.Errorf("package not found: %s", name)
	}

	return client
}

// Test scenario builders
type ScenarioBuilder struct {
	name        string
	description string
	setup       func() map[string]interface{}
	validate    func(result interface{}) error
}

func NewScenarioBuilder(name string) *ScenarioBuilder {
	return &ScenarioBuilder{
		name: name,
	}
}

func (b *ScenarioBuilder) WithDescription(desc string) *ScenarioBuilder {
	b.description = desc
	return b
}

func (b *ScenarioBuilder) WithSetup(setup func() map[string]interface{}) *ScenarioBuilder {
	b.setup = setup
	return b
}

func (b *ScenarioBuilder) WithValidation(validate func(result interface{}) error) *ScenarioBuilder {
	b.validate = validate
	return b
}

func (b *ScenarioBuilder) Build() TestScenario {
	return TestScenario{
		Name:        b.name,
		Description: b.description,
		Setup:       b.setup,
		Validate:    b.validate,
	}
}

type TestScenario struct {
	Name        string
	Description string
	Setup       func() map[string]interface{}
	Validate    func(result interface{}) error
}

// Mock response builders
type HTTPResponseBuilder struct {
	statusCode int
	body       string
	headers    map[string]string
}

func NewHTTPResponse() *HTTPResponseBuilder {
	return &HTTPResponseBuilder{
		statusCode: 200,
		headers:    make(map[string]string),
	}
}

func (b *HTTPResponseBuilder) WithStatus(code int) *HTTPResponseBuilder {
	b.statusCode = code
	return b
}

func (b *HTTPResponseBuilder) WithBody(body string) *HTTPResponseBuilder {
	b.body = body
	return b
}

func (b *HTTPResponseBuilder) WithJSONBody(obj interface{}) *HTTPResponseBuilder {
	data, _ := json.Marshal(obj)
	b.body = string(data)
	b.headers["Content-Type"] = "application/json"
	return b
}

func (b *HTTPResponseBuilder) WithHeader(key, value string) *HTTPResponseBuilder {
	b.headers[key] = value
	return b
}

func (b *HTTPResponseBuilder) Build() *http.Response {
	return CreateHTTPResponse(b.statusCode, b.body, b.headers)
}

// Test data generators
type TestDataGenerator struct{}

func NewTestDataGenerator() *TestDataGenerator {
	return &TestDataGenerator{}
}

func (g *TestDataGenerator) GenerateLatencyData(baseLatency time.Duration, variance float64, points int) []LatencyDataPoint {
	data := make([]LatencyDataPoint, points)
	now := time.Now()

	for i := 0; i < points; i++ {
		// Add some realistic variance
		varianceNs := int64(float64(baseLatency.Nanoseconds()) * variance * (0.5 - float64(i%100)/100.0))
		latency := baseLatency + time.Duration(varianceNs)

		data[i] = LatencyDataPoint{
			Timestamp: now.Add(time.Duration(i) * time.Minute),
			Value:     latency,
		}
	}

	return data
}

func (g *TestDataGenerator) GenerateThroughputData(baseThroughput float64, variance float64, points int) []ThroughputDataPoint {
	data := make([]ThroughputDataPoint, points)
	now := time.Now()

	for i := 0; i < points; i++ {
		// Add realistic throughput variance
		downlinkVariance := baseThroughput * variance * (0.5 - float64(i%100)/100.0)
		uplinkVariance := baseThroughput * 0.1 * variance * (0.5 - float64(i%100)/100.0)

		data[i] = ThroughputDataPoint{
			Timestamp: now.Add(time.Duration(i) * time.Minute),
			Downlink:  baseThroughput + downlinkVariance,
			Uplink:    (baseThroughput * 0.1) + uplinkVariance,
		}
	}

	return data
}

func (g *TestDataGenerator) GenerateResourceData(baseCPU, baseMemory, baseNetwork float64, variance float64, points int) []ResourceDataPoint {
	data := make([]ResourceDataPoint, points)
	now := time.Now()

	for i := 0; i < points; i++ {
		cpuVariance := baseCPU * variance * (0.5 - float64(i%100)/100.0)
		memoryVariance := baseMemory * variance * (0.5 - float64(i%100)/100.0)
		networkVariance := baseNetwork * variance * (0.5 - float64(i%100)/100.0)

		data[i] = ResourceDataPoint{
			Timestamp: now.Add(time.Duration(i) * time.Minute),
			CPU:       baseCPU + cpuVariance,
			Memory:    baseMemory + memoryVariance,
			Network:   baseNetwork + networkVariance,
		}
	}

	return data
}

// Mock call verification utilities
type CallVerifier struct {
	expectedCalls map[string]int
	actualCalls   map[string]int
}

func NewCallVerifier() *CallVerifier {
	return &CallVerifier{
		expectedCalls: make(map[string]int),
		actualCalls:   make(map[string]int),
	}
}

func (v *CallVerifier) ExpectCall(method string, times int) *CallVerifier {
	v.expectedCalls[method] = times
	return v
}

func (v *CallVerifier) RecordCall(method string) {
	v.actualCalls[method]++
}

func (v *CallVerifier) Verify() error {
	for method, expectedCount := range v.expectedCalls {
		actualCount := v.actualCalls[method]
		if actualCount != expectedCount {
			return fmt.Errorf("method %s: expected %d calls, got %d", method, expectedCount, actualCount)
		}
	}

	for method, actualCount := range v.actualCalls {
		if _, expected := v.expectedCalls[method]; !expected && actualCount > 0 {
			return fmt.Errorf("unexpected calls to method %s: %d", method, actualCount)
		}
	}

	return nil
}

// Performance testing utilities
type PerformanceTracker struct {
	startTime time.Time
	events    []PerformanceEvent
}

type PerformanceEvent struct {
	Name      string        `json:"name"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

func NewPerformanceTracker() *PerformanceTracker {
	return &PerformanceTracker{
		startTime: time.Now(),
		events:    make([]PerformanceEvent, 0),
	}
}

func (p *PerformanceTracker) StartEvent(name string) func() {
	start := time.Now()
	return func() {
		p.events = append(p.events, PerformanceEvent{
			Name:      name,
			Timestamp: start,
			Duration:  time.Since(start),
		})
	}
}

func (p *PerformanceTracker) RecordEvent(name string, metadata map[string]interface{}) {
	p.events = append(p.events, PerformanceEvent{
		Name:      name,
		Timestamp: time.Now(),
		Metadata:  metadata,
	})
}

func (p *PerformanceTracker) GetTotalDuration() time.Duration {
	return time.Since(p.startTime)
}

func (p *PerformanceTracker) GetEventDuration(name string) time.Duration {
	for _, event := range p.events {
		if event.Name == name && event.Duration > 0 {
			return event.Duration
		}
	}
	return 0
}

func (p *PerformanceTracker) GetSummary() PerformanceSummary {
	return PerformanceSummary{
		TotalDuration: p.GetTotalDuration(),
		Events:        p.events,
		EventCount:    len(p.events),
	}
}

type PerformanceSummary struct {
	TotalDuration time.Duration       `json:"totalDuration"`
	Events        []PerformanceEvent  `json:"events"`
	EventCount    int                 `json:"eventCount"`
}

// Error simulation utilities
type ErrorSimulator struct {
	errorScenarios map[string]ErrorScenario
}

type ErrorScenario struct {
	Probability float64 // 0.0 to 1.0
	ErrorType   string
	Message     string
	Delay       time.Duration
}

func NewErrorSimulator() *ErrorSimulator {
	return &ErrorSimulator{
		errorScenarios: make(map[string]ErrorScenario),
	}
}

func (e *ErrorSimulator) AddScenario(name string, scenario ErrorScenario) {
	e.errorScenarios[name] = scenario
}

func (e *ErrorSimulator) ShouldError(scenarioName string) (bool, error) {
	scenario, exists := e.errorScenarios[scenarioName]
	if !exists {
		return false, nil
	}

	// Simulate probability
	if time.Now().UnixNano()%100 < int64(scenario.Probability*100) {
		if scenario.Delay > 0 {
			time.Sleep(scenario.Delay)
		}
		return true, fmt.Errorf("%s: %s", scenario.ErrorType, scenario.Message)
	}

	return false, nil
}

// Test cleanup utilities
type TestCleaner struct {
	cleanupFuncs []func() error
}

func NewTestCleaner() *TestCleaner {
	return &TestCleaner{
		cleanupFuncs: make([]func() error, 0),
	}
}

func (c *TestCleaner) AddCleanup(f func() error) {
	c.cleanupFuncs = append(c.cleanupFuncs, f)
}

func (c *TestCleaner) Cleanup() []error {
	var errors []error
	for _, f := range c.cleanupFuncs {
		if err := f(); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

// Common test patterns
func CreateTestHTTPResponses() map[string]*http.Response {
	return map[string]*http.Response{
		"GET /o2ims/v1/resourceTypes": CreateHTTPResponse(200, `{
			"resourceTypes": [
				{
					"resourceTypeId": "rt-cucp-001",
					"name": "CU-CP",
					"vendor": "TestVendor",
					"model": "TestModel",
					"version": "1.0.0"
				}
			]
		}`, map[string]string{"Content-Type": "application/json"}),

		"GET /o2ims/v1/resources": CreateHTTPResponse(200, `{
			"resources": [
				{
					"resourceId": "res-cucp-001",
					"resourceTypeId": "rt-cucp-001",
					"description": "Test CU-CP instance",
					"location": "test-zone"
				}
			]
		}`, map[string]string{"Content-Type": "application/json"}),

		"POST /o2ims/v1/subscriptions": CreateHTTPResponse(201, `{
			"subscriptionId": "sub-001",
			"callback": "http://test-callback:8080/notifications"
		}`, map[string]string{"Content-Type": "application/json"}),
	}
}

func CreateTestNephioPackages() []*NephioPackage {
	return []*NephioPackage{
		CreateTestNephioPackage("test-cucp-package"),
		CreateTestNephioPackage("test-cuup-package"),
		CreateTestNephioPackage("test-du-package"),
	}
}

// Utility functions for common test operations
func WaitForCondition(condition func() bool, timeout time.Duration, interval time.Duration) error {
	timer := time.NewTimer(timeout)
	ticker := time.NewTicker(interval)
	defer timer.Stop()
	defer ticker.Stop()

	for {
		select {
		case <-timer.C:
			return fmt.Errorf("timeout waiting for condition")
		case <-ticker.C:
			if condition() {
				return nil
			}
		}
	}
}

func RetryOperation(operation func() error, maxRetries int, delay time.Duration) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := operation(); err == nil {
			return nil
		} else {
			lastErr = err
			if i < maxRetries-1 {
				time.Sleep(delay)
			}
		}
	}
	return fmt.Errorf("operation failed after %d retries: %v", maxRetries, lastErr)
}