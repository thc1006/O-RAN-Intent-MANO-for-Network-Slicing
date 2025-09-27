package monitoring

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// MockObservabilityStack mocks the entire observability stack
type MockObservabilityStack struct {
	mock.Mock
}

// E2EObservabilityTestSuite defines the test suite for end-to-end observability
type E2EObservabilityTestSuite struct {
	suite.Suite
	mockStack *MockObservabilityStack
	stack     *ObservabilityStack
}

// ObservabilityStack represents the complete monitoring stack
type ObservabilityStack struct {
	stack ObservabilityStackInterface
}

// ObservabilityStackInterface defines the contract for observability operations
type ObservabilityStackInterface interface {
	// Application to Prometheus flow
	GenerateMetrics(component string, metrics []Metric) error
	ExposeMetricsEndpoint(component string, port int) error
	ScrapeMetrics(component string) (*ScrapeResult, error)
	
	// Prometheus to Grafana flow
	QueryMetrics(query string, timeRange TimeRange) (*QueryResult, error)
	RenderDashboard(dashboardUID string, timeRange TimeRange) (*DashboardRender, error)
	
	// Alert flow
	EvaluateAlerts(rules []AlertRule) ([]Alert, error)
	TriggerAlert(alert Alert) error
	PropagateAlert(alert Alert) (*AlertPropagation, error)
	
	// Performance testing
	MeasureQueryPerformance(query string, timeRange TimeRange) (*PerformanceMetrics, error)
	TestHighCardinality(metricCount int) (*CardinalityTestResult, error)
	
	// Integration testing
	DeployFullStack(config *StackConfig) error
	TestEndToEndFlow(scenario *E2EScenario) (*E2EResult, error)
	CleanupStack() error
}

// ScrapeResult represents metrics scraping result
type ScrapeResult struct {
	Component     string
	MetricsCount  int
	ScrapeTime    time.Duration
	Success       bool
	ErrorMessages []string
	LastScrapeAt  time.Time
}

// QueryResult represents Prometheus query result
type QueryResult struct {
	Query         string
	ResultType    string
	Data          interface{}
	ExecutionTime time.Duration
	Status        string
	Warnings      []string
}

// DashboardRender represents Grafana dashboard render result
type DashboardRender struct {
	DashboardUID  string
	RenderTime    time.Duration
	PanelCount    int
	DataPoints    int
	Success       bool
	Errors        []string
}

// AlertRule represents alert rule
type AlertRule struct {
	Name        string
	Expression  string
	Duration    time.Duration
	Severity    string
	Annotations map[string]string
	Labels      map[string]string
}

// AlertPropagation represents alert propagation result
type AlertPropagation struct {
	AlertID       string
	Notifications []NotificationResult
	TotalTime     time.Duration
	Success       bool
}

// NotificationResult represents notification delivery result
type NotificationResult struct {
	Receiver    string
	Channel     string
	Delivered   bool
	DeliveryTime time.Duration
	Error       string
}

// PerformanceMetrics represents query performance metrics
type PerformanceMetrics struct {
	Query         string
	ExecutionTime time.Duration
	MemoryUsage   int64
	SamplesCount  int
	SeriesCount   int
	ChunksCount   int
}

// CardinalityTestResult represents cardinality test result
type CardinalityTestResult struct {
	MetricCount      int
	SeriesCount      int
	IngestionRate    float64
	QueryPerformance map[string]time.Duration
	MemoryUsage      int64
	Success          bool
	Errors           []string
}

// StackConfig represents stack configuration
type StackConfig struct {
	Namespace    string
	Components   []string
	Prometheus   *PrometheusConfig
	Grafana      *GrafanaConfig
	AlertManager *AlertManagerConfig
}

// E2EScenario represents end-to-end test scenario
type E2EScenario struct {
	Name        string
	Steps       []E2EStep
	Timeout     time.Duration
	RetryCount  int
	Expectation *E2EExpectation
}

// E2EStep represents a single step in E2E scenario
type E2EStep struct {
	Name        string
	Action      string
	Component   string
	Parameters  map[string]interface{}
	WaitTime    time.Duration
}

// E2EExpectation represents expected E2E results
type E2EExpectation struct {
	MetricsExposed   bool
	AlertsTriggered  int
	DashboardRenders bool
	NotificationsSent int
	MaxLatency       time.Duration
}

// E2EResult represents end-to-end test result
type E2EResult struct {
	Scenario      string
	Success       bool
	StepResults   []StepResult
	TotalDuration time.Duration
	Errors        []string
	Metrics       map[string]interface{}
}

// StepResult represents individual step result
type StepResult struct {
	Step     string
	Success  bool
	Duration time.Duration
	Error    string
	Output   interface{}
}

// SetupSuite initializes the test suite
func (suite *E2EObservabilityTestSuite) SetupSuite() {
	suite.mockStack = &MockObservabilityStack{}
	suite.stack = &ObservabilityStack{stack: suite.mockStack}
}

// TearDownSuite cleans up after test suite
func (suite *E2EObservabilityTestSuite) TearDownSuite() {
	// Cleanup resources
}

// TestEndToEndMetricFlow tests complete metric flow from app to Grafana
func (suite *E2EObservabilityTestSuite) TestEndToEndMetricFlow() {
	testCases := []struct {
		name          string
		component     string
		metrics       []Metric
		query         string
		timeRange     TimeRange
		expectedError bool
		errorMessage  string
		mockBehavior  func(*MockObservabilityStack)
	}{
		{
			name:      "orchestrator metrics flow",
			component: "orchestrator",
			metrics: []Metric{
				{
					Name: "o_ran_requests_total",
					Type: CounterType,
					Labels: map[string]string{
						"component": "orchestrator",
						"namespace": "o-ran-mano",
						"method":    "POST",
					},
					Value: 100,
				},
				{
					Name: "o_ran_cpu_usage",
					Type: GaugeType,
					Labels: map[string]string{
						"component": "orchestrator",
						"namespace": "o-ran-mano",
					},
					Value: 0.75,
				},
			},
			query:     "rate(o_ran_requests_total{component=\"orchestrator\"}[5m])",
			timeRange: TimeRange{From: "now-1h", To: "now"},
			expectedError: false,
			mockBehavior: func(m *MockObservabilityStack) {
				// Mock the complete flow
				m.On("GenerateMetrics", "orchestrator", mock.AnythingOfType("[]monitoring.Metric")).Return(nil)
				m.On("ExposeMetricsEndpoint", "orchestrator", 8080).Return(nil)
				m.On("ScrapeMetrics", "orchestrator").Return(&ScrapeResult{
					Component:    "orchestrator",
					MetricsCount: 2,
					ScrapeTime:   100 * time.Millisecond,
					Success:      true,
					LastScrapeAt: time.Now(),
				}, nil)
				m.On("QueryMetrics", "rate(o_ran_requests_total{component=\"orchestrator\"}[5m])", mock.AnythingOfType("monitoring.TimeRange")).Return(&QueryResult{
					Query:         "rate(o_ran_requests_total{component=\"orchestrator\"}[5m])",
					ResultType:    "vector",
					ExecutionTime: 50 * time.Millisecond,
					Status:        "success",
				}, nil)
				m.On("RenderDashboard", "o-ran-orchestrator", mock.AnythingOfType("monitoring.TimeRange")).Return(&DashboardRender{
					DashboardUID: "o-ran-orchestrator",
					RenderTime:   200 * time.Millisecond,
					PanelCount:   5,
					DataPoints:   1000,
					Success:      true,
				}, nil)
			},
		},
		{
			name:      "vnf-operator metrics flow",
			component: "vnf-operator",
			metrics: []Metric{
				{
					Name: "o_ran_vnf_deployments_total",
					Type: CounterType,
					Labels: map[string]string{
						"component": "vnf-operator",
						"namespace": "o-ran-mano",
						"vnf_type":  "du",
					},
					Value: 25,
				},
			},
			query:     "o_ran_vnf_deployments_total{component=\"vnf-operator\"}",
			timeRange: TimeRange{From: "now-6h", To: "now"},
			expectedError: false,
			mockBehavior: func(m *MockObservabilityStack) {
				m.On("GenerateMetrics", "vnf-operator", mock.AnythingOfType("[]monitoring.Metric")).Return(nil)
				m.On("ExposeMetricsEndpoint", "vnf-operator", 8080).Return(nil)
				m.On("ScrapeMetrics", "vnf-operator").Return(&ScrapeResult{
					Component:    "vnf-operator",
					MetricsCount: 1,
					ScrapeTime:   75 * time.Millisecond,
					Success:      true,
					LastScrapeAt: time.Now(),
				}, nil)
				m.On("QueryMetrics", "o_ran_vnf_deployments_total{component=\"vnf-operator\"}", mock.AnythingOfType("monitoring.TimeRange")).Return(&QueryResult{
					Query:         "o_ran_vnf_deployments_total{component=\"vnf-operator\"}",
					ResultType:    "vector",
					ExecutionTime: 30 * time.Millisecond,
					Status:        "success",
				}, nil)
				m.On("RenderDashboard", "o-ran-vnf-operator", mock.AnythingOfType("monitoring.TimeRange")).Return(&DashboardRender{
					DashboardUID: "o-ran-vnf-operator",
					RenderTime:   150 * time.Millisecond,
					PanelCount:   3,
					DataPoints:   500,
					Success:      true,
				}, nil)
			},
		},
		{
			name:          "metric generation failure",
			component:     "failing-component",
			metrics:       []Metric{},
			query:         "",
			timeRange:     TimeRange{},
			expectedError: true,
			errorMessage:  "metric generation failed",
			mockBehavior: func(m *MockObservabilityStack) {
				m.On("GenerateMetrics", "failing-component", mock.AnythingOfType("[]monitoring.Metric")).Return(errors.New("metric generation failed"))
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Arrange
			tc.mockBehavior(suite.mockStack)

			// Act
			err := suite.stack.TestCompleteMetricFlow(tc.component, tc.metrics, tc.query, tc.timeRange)

			// Assert
			if tc.expectedError {
				suite.Error(err)
				if tc.errorMessage != "" {
					suite.Contains(err.Error(), tc.errorMessage)
				}
			} else {
				suite.NoError(err)
			}

			suite.mockStack.AssertExpectations(suite.T())
		})
	}
}

// TestAlertTriggering tests alert triggering and propagation
func (suite *E2EObservabilityTestSuite) TestAlertTriggering() {
	testCases := []struct {
		name          string
		alertRules    []AlertRule
		expectedError bool
		errorMessage  string
		mockBehavior  func(*MockObservabilityStack)
	}{
		{
			name: "critical orchestrator alert",
			alertRules: []AlertRule{
				{
					Name:       "O_RANOrchestratorDown",
					Expression: "up{component=\"orchestrator\"} == 0",
					Duration:   1 * time.Minute,
					Severity:   "critical",
					Annotations: map[string]string{
						"summary":     "O-RAN Orchestrator is down",
						"description": "O-RAN Orchestrator has been down for more than 1 minute",
						"runbook_url": "https://docs.o-ran.company.com/runbooks/orchestrator-down",
					},
					Labels: map[string]string{
						"component": "orchestrator",
						"team":      "o-ran-ops",
					},
				},
			},
			expectedError: false,
			mockBehavior: func(m *MockObservabilityStack) {
				m.On("EvaluateAlerts", mock.AnythingOfType("[]monitoring.AlertRule")).Return([]Alert{
					{
						Labels: map[string]string{
							"alertname": "O_RANOrchestratorDown",
							"component": "orchestrator",
							"severity":  "critical",
						},
						Annotations: map[string]string{
							"summary": "O-RAN Orchestrator is down",
						},
						State:    "firing",
						ActiveAt: time.Now(),
					},
				}, nil)
				m.On("TriggerAlert", mock.AnythingOfType("monitoring.Alert")).Return(nil)
				m.On("PropagateAlert", mock.AnythingOfType("monitoring.Alert")).Return(&AlertPropagation{
					AlertID: "alert-123",
					Notifications: []NotificationResult{
						{Receiver: "o-ran-critical", Channel: "slack", Delivered: true, DeliveryTime: 100 * time.Millisecond},
						{Receiver: "o-ran-critical", Channel: "email", Delivered: true, DeliveryTime: 500 * time.Millisecond},
						{Receiver: "o-ran-critical", Channel: "pagerduty", Delivered: true, DeliveryTime: 200 * time.Millisecond},
					},
					TotalTime: 800 * time.Millisecond,
					Success:   true,
				}, nil)
			},
		},
		{
			name: "warning vnf-operator alert",
			alertRules: []AlertRule{
				{
					Name:       "O_RANVNFOperatorHighCPU",
					Expression: "o_ran_cpu_usage{component=\"vnf-operator\"} > 0.8",
					Duration:   5 * time.Minute,
					Severity:   "warning",
					Annotations: map[string]string{
						"summary":     "VNF Operator high CPU usage",
						"description": "VNF Operator CPU usage is above 80% for more than 5 minutes",
					},
				},
			},
			expectedError: false,
			mockBehavior: func(m *MockObservabilityStack) {
				m.On("EvaluateAlerts", mock.AnythingOfType("[]monitoring.AlertRule")).Return([]Alert{
					{
						Labels: map[string]string{
							"alertname": "O_RANVNFOperatorHighCPU",
							"component": "vnf-operator",
							"severity":  "warning",
						},
						State:    "firing",
						ActiveAt: time.Now(),
					},
				}, nil)
				m.On("TriggerAlert", mock.AnythingOfType("monitoring.Alert")).Return(nil)
				m.On("PropagateAlert", mock.AnythingOfType("monitoring.Alert")).Return(&AlertPropagation{
					AlertID: "alert-456",
					Notifications: []NotificationResult{
						{Receiver: "o-ran-warning", Channel: "slack", Delivered: true, DeliveryTime: 150 * time.Millisecond},
						{Receiver: "o-ran-warning", Channel: "email", Delivered: true, DeliveryTime: 400 * time.Millisecond},
					},
					TotalTime: 550 * time.Millisecond,
					Success:   true,
				}, nil)
			},
		},
		{
			name:          "alert evaluation failure",
			alertRules:    []AlertRule{{Name: "InvalidAlert", Expression: "invalid_expr{{"}},
			expectedError: true,
			errorMessage:  "alert evaluation failed",
			mockBehavior: func(m *MockObservabilityStack) {
				m.On("EvaluateAlerts", mock.AnythingOfType("[]monitoring.AlertRule")).Return(nil, errors.New("alert evaluation failed: invalid expression"))
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Arrange
			tc.mockBehavior(suite.mockStack)

			// Act
			err := suite.stack.TestAlertFlow(tc.alertRules)

			// Assert
			if tc.expectedError {
				suite.Error(err)
				if tc.errorMessage != "" {
					suite.Contains(err.Error(), tc.errorMessage)
				}
			} else {
				suite.NoError(err)
			}

			suite.mockStack.AssertExpectations(suite.T())
		})
	}
}

// TestQueryPerformance tests query performance requirements
func (suite *E2EObservabilityTestSuite) TestQueryPerformance() {
	testCases := []struct {
		name           string
		query          string
		timeRange      TimeRange
		maxDuration    time.Duration
		expectedError  bool
		errorMessage   string
		mockBehavior   func(*MockObservabilityStack)
	}{
		{
			name:        "fast 1h range query",
			query:       "rate(o_ran_requests_total[5m])",
			timeRange:   TimeRange{From: "now-1h", To: "now"},
			maxDuration: 1 * time.Second,
			expectedError: false,
			mockBehavior: func(m *MockObservabilityStack) {
				m.On("MeasureQueryPerformance", "rate(o_ran_requests_total[5m])", mock.AnythingOfType("monitoring.TimeRange")).Return(&PerformanceMetrics{
					Query:         "rate(o_ran_requests_total[5m])",
					ExecutionTime: 500 * time.Millisecond,
					MemoryUsage:   1024 * 1024, // 1MB
					SamplesCount:  1000,
					SeriesCount:   10,
					ChunksCount:   100,
				}, nil)
			},
		},
		{
			name:        "complex aggregation query",
			query:       "sum by (component) (rate(o_ran_requests_total[5m]))",
			timeRange:   TimeRange{From: "now-6h", To: "now"},
			maxDuration: 2 * time.Second,
			expectedError: false,
			mockBehavior: func(m *MockObservabilityStack) {
				m.On("MeasureQueryPerformance", "sum by (component) (rate(o_ran_requests_total[5m]))", mock.AnythingOfType("monitoring.TimeRange")).Return(&PerformanceMetrics{
					Query:         "sum by (component) (rate(o_ran_requests_total[5m]))",
					ExecutionTime: 1500 * time.Millisecond,
					MemoryUsage:   5 * 1024 * 1024, // 5MB
					SamplesCount:  10000,
					SeriesCount:   50,
					ChunksCount:   500,
				}, nil)
			},
		},
		{
			name:          "slow query exceeds limit",
			query:         "slow_query_with_huge_cardinality",
			timeRange:     TimeRange{From: "now-24h", To: "now"},
			maxDuration:   1 * time.Second,
			expectedError: true,
			errorMessage:  "query exceeded time limit",
			mockBehavior: func(m *MockObservabilityStack) {
				m.On("MeasureQueryPerformance", "slow_query_with_huge_cardinality", mock.AnythingOfType("monitoring.TimeRange")).Return(&PerformanceMetrics{
					Query:         "slow_query_with_huge_cardinality",
					ExecutionTime: 5 * time.Second, // Exceeds limit
					MemoryUsage:   100 * 1024 * 1024, // 100MB
					SamplesCount:  1000000,
					SeriesCount:   1000,
				}, nil)
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Arrange
			tc.mockBehavior(suite.mockStack)

			// Act
			err := suite.stack.TestQueryPerformance(tc.query, tc.timeRange, tc.maxDuration)

			// Assert
			if tc.expectedError {
				suite.Error(err)
				if tc.errorMessage != "" {
					suite.Contains(err.Error(), tc.errorMessage)
				}
			} else {
				suite.NoError(err)
			}

			suite.mockStack.AssertExpectations(suite.T())
		})
	}
}

// TestHighCardinalityScenario tests high cardinality metric handling
func (suite *E2EObservabilityTestSuite) TestHighCardinalityScenario() {
	testCases := []struct {
		name          string
		metricCount   int
		expectedError bool
		errorMessage  string
		mockBehavior  func(*MockObservabilityStack)
	}{
		{
			name:        "normal cardinality - 1000 metrics",
			metricCount: 1000,
			expectedError: false,
			mockBehavior: func(m *MockObservabilityStack) {
				m.On("TestHighCardinality", 1000).Return(&CardinalityTestResult{
					MetricCount:   1000,
					SeriesCount:   1000,
					IngestionRate: 1000.0, // metrics per second
					QueryPerformance: map[string]time.Duration{
						"simple_query":  100 * time.Millisecond,
						"complex_query": 500 * time.Millisecond,
					},
					MemoryUsage: 50 * 1024 * 1024, // 50MB
					Success:     true,
				}, nil)
			},
		},
		{
			name:        "high cardinality - 10000 metrics",
			metricCount: 10000,
			expectedError: false,
			mockBehavior: func(m *MockObservabilityStack) {
				m.On("TestHighCardinality", 10000).Return(&CardinalityTestResult{
					MetricCount:   10000,
					SeriesCount:   10000,
					IngestionRate: 8000.0, // Slightly degraded
					QueryPerformance: map[string]time.Duration{
						"simple_query":  200 * time.Millisecond,
						"complex_query": 1000 * time.Millisecond,
					},
					MemoryUsage: 200 * 1024 * 1024, // 200MB
					Success:     true,
				}, nil)
			},
		},
		{
			name:          "extreme cardinality - system overload",
			metricCount:   100000,
			expectedError: true,
			errorMessage:  "cardinality limit exceeded",
			mockBehavior: func(m *MockObservabilityStack) {
				m.On("TestHighCardinality", 100000).Return(&CardinalityTestResult{
					MetricCount:   100000,
					SeriesCount:   100000,
					IngestionRate: 100.0, // Severely degraded
					QueryPerformance: map[string]time.Duration{
						"simple_query":  5 * time.Second, // Too slow
						"complex_query": 30 * time.Second, // Unacceptable
					},
					MemoryUsage: 2 * 1024 * 1024 * 1024, // 2GB
					Success:     false,
					Errors:      []string{"memory exhausted", "query timeout"},
				}, nil)
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Arrange
			tc.mockBehavior(suite.mockStack)

			// Act
			err := suite.stack.TestHighCardinalityHandling(tc.metricCount)

			// Assert
			if tc.expectedError {
				suite.Error(err)
				if tc.errorMessage != "" {
					suite.Contains(err.Error(), tc.errorMessage)
				}
			} else {
				suite.NoError(err)
			}

			suite.mockStack.AssertExpectations(suite.T())
		})
	}
}

// TestFullStackIntegration tests complete stack deployment and E2E scenarios
func (suite *E2EObservabilityTestSuite) TestFullStackIntegration() {
	testCases := []struct {
		name          string
		scenario      *E2EScenario
		expectedError bool
		errorMessage  string
		mockBehavior  func(*MockObservabilityStack)
	}{
		{
			name: "complete o-ran stack deployment",
			scenario: &E2EScenario{
				Name:    "O-RAN Full Stack E2E",
				Timeout: 10 * time.Minute,
				Steps: []E2EStep{
					{Name: "Deploy Prometheus", Action: "deploy", Component: "prometheus"},
					{Name: "Deploy Grafana", Action: "deploy", Component: "grafana"},
					{Name: "Deploy AlertManager", Action: "deploy", Component: "alertmanager"},
					{Name: "Deploy O-RAN Components", Action: "deploy", Component: "o-ran"},
					{Name: "Wait for Metrics", Action: "wait", WaitTime: 30 * time.Second},
					{Name: "Validate Dashboards", Action: "validate", Component: "grafana"},
					{Name: "Trigger Test Alert", Action: "trigger", Component: "alertmanager"},
				},
				Expectation: &E2EExpectation{
					MetricsExposed:    true,
					AlertsTriggered:   1,
					DashboardRenders:  true,
					NotificationsSent: 3, // Slack, email, PagerDuty
					MaxLatency:        2 * time.Second,
				},
			},
			expectedError: false,
			mockBehavior: func(m *MockObservabilityStack) {
				m.On("DeployFullStack", mock.AnythingOfType("*monitoring.StackConfig")).Return(nil)
				m.On("TestEndToEndFlow", mock.AnythingOfType("*monitoring.E2EScenario")).Return(&E2EResult{
					Scenario:      "O-RAN Full Stack E2E",
					Success:       true,
					TotalDuration: 5 * time.Minute,
					StepResults: []StepResult{
						{Step: "Deploy Prometheus", Success: true, Duration: 30 * time.Second},
						{Step: "Deploy Grafana", Success: true, Duration: 45 * time.Second},
						{Step: "Deploy AlertManager", Success: true, Duration: 20 * time.Second},
						{Step: "Deploy O-RAN Components", Success: true, Duration: 2 * time.Minute},
						{Step: "Wait for Metrics", Success: true, Duration: 30 * time.Second},
						{Step: "Validate Dashboards", Success: true, Duration: 10 * time.Second},
						{Step: "Trigger Test Alert", Success: true, Duration: 5 * time.Second},
					},
					Metrics: map[string]interface{}{
						"metrics_exposed":     true,
						"alerts_triggered":    1,
						"dashboard_renders":   true,
						"notifications_sent":  3,
						"max_latency_seconds": 1.5,
					},
				}, nil)
				m.On("CleanupStack").Return(nil)
			},
		},
		{
			name: "integration test with real cluster",
			scenario: &E2EScenario{
				Name:    "Real Cluster Integration",
				Timeout: 15 * time.Minute,
				Steps: []E2EStep{
					{Name: "Connect to Cluster", Action: "connect"},
					{Name: "Deploy Full Stack", Action: "deploy"},
					{Name: "Generate Load", Action: "load", Parameters: map[string]interface{}{"requests_per_second": 100}},
					{Name: "Monitor Performance", Action: "monitor", WaitTime: 2 * time.Minute},
					{Name: "Validate Results", Action: "validate"},
				},
				Expectation: &E2EExpectation{
					MetricsExposed:    true,
					DashboardRenders:  true,
					MaxLatency:        1 * time.Second,
				},
			},
			expectedError: false,
			mockBehavior: func(m *MockObservabilityStack) {
				m.On("TestEndToEndFlow", mock.AnythingOfType("*monitoring.E2EScenario")).Return(&E2EResult{
					Scenario:      "Real Cluster Integration",
					Success:       true,
					TotalDuration: 8 * time.Minute,
					StepResults: []StepResult{
						{Step: "Connect to Cluster", Success: true, Duration: 10 * time.Second},
						{Step: "Deploy Full Stack", Success: true, Duration: 3 * time.Minute},
						{Step: "Generate Load", Success: true, Duration: 30 * time.Second},
						{Step: "Monitor Performance", Success: true, Duration: 2 * time.Minute},
						{Step: "Validate Results", Success: true, Duration: 20 * time.Second},
					},
				}, nil)
			},
		},
		{
			name:          "deployment failure scenario",
			scenario:      &E2EScenario{Name: "Failing Deployment", Steps: []E2EStep{{Name: "Deploy", Action: "deploy"}}},
			expectedError: true,
			errorMessage:  "deployment failed",
			mockBehavior: func(m *MockObservabilityStack) {
				m.On("TestEndToEndFlow", mock.AnythingOfType("*monitoring.E2EScenario")).Return(nil, errors.New("deployment failed: insufficient resources"))
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Arrange
			tc.mockBehavior(suite.mockStack)

			// Act
			err := suite.stack.RunE2EIntegrationTest(tc.scenario)

			// Assert
			if tc.expectedError {
				suite.Error(err)
				if tc.errorMessage != "" {
					suite.Contains(err.Error(), tc.errorMessage)
				}
			} else {
				suite.NoError(err)
			}

			suite.mockStack.AssertExpectations(suite.T())
		})
	}
}

// Run the test suite
func TestE2EObservabilityTestSuite(t *testing.T) {
	suite.Run(t, new(E2EObservabilityTestSuite))
}

// These methods will need to be implemented in the actual ObservabilityStack
// They are defined here to make the tests compile and FAIL (RED phase)
func (os *ObservabilityStack) TestCompleteMetricFlow(component string, metrics []Metric, query string, timeRange TimeRange) error {
	panic("not implemented - this test should FAIL in RED phase")
}

func (os *ObservabilityStack) TestAlertFlow(alertRules []AlertRule) error {
	panic("not implemented - this test should FAIL in RED phase")
}

func (os *ObservabilityStack) TestQueryPerformance(query string, timeRange TimeRange, maxDuration time.Duration) error {
	panic("not implemented - this test should FAIL in RED phase")
}

func (os *ObservabilityStack) TestHighCardinalityHandling(metricCount int) error {
	panic("not implemented - this test should FAIL in RED phase")
}

func (os *ObservabilityStack) RunE2EIntegrationTest(scenario *E2EScenario) error {
	panic("not implemented - this test should FAIL in RED phase")
}
