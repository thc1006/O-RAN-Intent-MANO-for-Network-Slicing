package monitoring

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockAlertManagerClient mocks the AlertManager client interface
type MockAlertManagerClient struct {
	mock.Mock
}

// AlertManagerDeployer handles AlertManager deployment and configuration
type AlertManagerDeployer struct {
	client AlertManagerClientInterface
}

// AlertManagerClientInterface defines the contract for AlertManager operations
type AlertManagerClientInterface interface {
	DeployAlertManager(ctx context.Context, config *AlertManagerConfig) error
	ConfigureRouting(ctx context.Context, routes []Route) error
	ConfigureReceivers(ctx context.Context, receivers []Receiver) error
	SetInhibitionRules(ctx context.Context, rules []InhibitionRule) error
	CreateSilence(ctx context.Context, silence *Silence) (*SilenceResponse, error)
	DeleteSilence(ctx context.Context, silenceID string) error
	GetActiveAlerts(ctx context.Context) ([]Alert, error)
	ValidateConfiguration(config *AlertManagerConfig) error
	ReloadConfiguration(ctx context.Context) error
}

// AlertManagerConfig represents AlertManager configuration
type AlertManagerConfig struct {
	Global      *GlobalConfig    `yaml:"global"`
	Route       *Route           `yaml:"route"`
	Receivers   []Receiver       `yaml:"receivers"`
	InhibitRules []InhibitionRule `yaml:"inhibit_rules"`
	Templates   []string         `yaml:"templates"`
}

// GlobalConfig represents global AlertManager configuration
type GlobalConfig struct {
	SMTPSmartHost    string        `yaml:"smtp_smarthost"`
	SMTPFrom         string        `yaml:"smtp_from"`
	SlackAPIURL      string        `yaml:"slack_api_url"`
	ResolveTimeout   time.Duration `yaml:"resolve_timeout"`
	HTTPConfig       *HTTPConfig   `yaml:"http_config"`
}

// HTTPConfig represents HTTP configuration
type HTTPConfig struct {
	TLSConfig *TLSConfig `yaml:"tls_config"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	CertFile           string `yaml:"cert_file"`
	KeyFile            string `yaml:"key_file"`
	CAFile             string `yaml:"ca_file"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
}

// Route represents routing configuration
type Route struct {
	Receiver       string            `yaml:"receiver"`
	GroupBy        []string          `yaml:"group_by"`
	GroupWait      time.Duration     `yaml:"group_wait"`
	GroupInterval  time.Duration     `yaml:"group_interval"`
	RepeatInterval time.Duration     `yaml:"repeat_interval"`
	Matchers       []Matcher         `yaml:"matchers"`
	Routes         []Route           `yaml:"routes"`
	Continue       bool              `yaml:"continue"`
}

// Matcher represents alert matcher
type Matcher struct {
	Name    string `yaml:"name"`
	Value   string `yaml:"value"`
	IsRegex bool   `yaml:"is_regex"`
	IsEqual bool   `yaml:"is_equal"`
}

// Receiver represents notification receiver
type Receiver struct {
	Name             string             `yaml:"name"`
	EmailConfigs     []EmailConfig      `yaml:"email_configs"`
	SlackConfigs     []SlackConfig      `yaml:"slack_configs"`
	WebhookConfigs   []WebhookConfig    `yaml:"webhook_configs"`
	PagerDutyConfigs []PagerDutyConfig  `yaml:"pagerduty_configs"`
}

// EmailConfig represents email notification configuration
type EmailConfig struct {
	To          string            `yaml:"to"`
	From        string            `yaml:"from"`
	Subject     string            `yaml:"subject"`
	Body        string            `yaml:"body"`
	HTML        string            `yaml:"html"`
	Headers     map[string]string `yaml:"headers"`
	SMTPSmartHost string          `yaml:"smtp_smarthost"`
}

// SlackConfig represents Slack notification configuration
type SlackConfig struct {
	APIURL      string            `yaml:"api_url"`
	Channel     string            `yaml:"channel"`
	Username    string            `yaml:"username"`
	Color       string            `yaml:"color"`
	Title       string            `yaml:"title"`
	TitleLink   string            `yaml:"title_link"`
	Pretext     string            `yaml:"pretext"`
	Text        string            `yaml:"text"`
	Fields      []SlackField      `yaml:"fields"`
	Footer      string            `yaml:"footer"`
	IconEmoji   string            `yaml:"icon_emoji"`
	IconURL     string            `yaml:"icon_url"`
	ImageURL    string            `yaml:"image_url"`
	ThumbURL    string            `yaml:"thumb_url"`
}

// SlackField represents Slack message field
type SlackField struct {
	Title string `yaml:"title"`
	Value string `yaml:"value"`
	Short bool   `yaml:"short"`
}

// WebhookConfig represents webhook notification configuration
type WebhookConfig struct {
	URL        string            `yaml:"url"`
	HTTPConfig *HTTPConfig       `yaml:"http_config"`
	MaxAlerts  int               `yaml:"max_alerts"`
}

// PagerDutyConfig represents PagerDuty notification configuration
type PagerDutyConfig struct {
	ServiceKey  string                 `yaml:"service_key"`
	RoutingKey  string                 `yaml:"routing_key"`
	URL         string                 `yaml:"url"`
	Client      string                 `yaml:"client"`
	ClientURL   string                 `yaml:"client_url"`
	Description string                 `yaml:"description"`
	Severity    string                 `yaml:"severity"`
	Class       string                 `yaml:"class"`
	Component   string                 `yaml:"component"`
	Group       string                 `yaml:"group"`
	Details     map[string]interface{} `yaml:"details"`
}

// InhibitionRule represents alert inhibition rule
type InhibitionRule struct {
	SourceMatchers []Matcher `yaml:"source_matchers"`
	TargetMatchers []Matcher `yaml:"target_matchers"`
	Equal          []string  `yaml:"equal"`
}

// Alert represents an active alert
type Alert struct {
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	State        string            `json:"state"`
	ActiveAt     time.Time         `json:"activeAt"`
	Value        string            `json:"value"`
	GeneratorURL string            `json:"generatorURL"`
}

// Silence represents alert silence
type Silence struct {
	ID        string            `json:"id"`
	Matchers  []Matcher         `json:"matchers"`
	StartsAt  time.Time         `json:"startsAt"`
	EndsAt    time.Time         `json:"endsAt"`
	CreatedBy string            `json:"createdBy"`
	Comment   string            `json:"comment"`
}

// SilenceResponse represents silence creation response
type SilenceResponse struct {
	SilenceID string `json:"silenceID"`
}

// Test cases for AlertManager configuration
func TestAlertManagerConfiguration(t *testing.T) {
	testCases := []struct {
		name          string
		config        *AlertManagerConfig
		expectedError bool
		errorMessage  string
		mockBehavior  func(*MockAlertManagerClient)
	}{
		{
			name: "valid o-ran alertmanager configuration",
			config: &AlertManagerConfig{
				Global: &GlobalConfig{
					SMTPSmartHost:  "smtp.o-ran.local:587",
					SMTPFrom:       "alerts@o-ran.local",
					SlackAPIURL:    "https://hooks.slack.com/services/...",
					ResolveTimeout: 5 * time.Minute,
				},
				Route: &Route{
					Receiver:       "o-ran-default",
					GroupBy:        []string{"alertname", "component", "severity"},
					GroupWait:      10 * time.Second,
					GroupInterval:  5 * time.Minute,
					RepeatInterval: 12 * time.Hour,
					Routes: []Route{
						{
							Receiver: "o-ran-critical",
							Matchers: []Matcher{
								{Name: "severity", Value: "critical", IsEqual: true},
								{Name: "component", Value: "orchestrator|vnf-operator|dms", IsRegex: true, IsEqual: true},
							},
							GroupWait:      5 * time.Second,
							RepeatInterval: 1 * time.Hour,
						},
					},
				},
				Receivers: []Receiver{
					{
						Name: "o-ran-default",
						EmailConfigs: []EmailConfig{
							{
								To:      "o-ran-team@company.com",
								Subject: "[O-RAN] Alert: {{ .GroupLabels.alertname }}",
								Body:    "Alert details: {{ range .Alerts }}{{ .Annotations.summary }}{{ end }}",
							},
						},
					},
					{
						Name: "o-ran-critical",
						SlackConfigs: []SlackConfig{
							{
								Channel:   "#o-ran-alerts",
								Username:  "O-RAN AlertManager",
								Color:     "danger",
								Title:     "🚨 Critical O-RAN Alert",
								Text:      "{{ range .Alerts }}{{ .Annotations.summary }}{{ end }}",
								IconEmoji: ":fire:",
							},
						},
						PagerDutyConfigs: []PagerDutyConfig{
							{
								ServiceKey:  "YOUR_PAGERDUTY_SERVICE_KEY",
								Description: "O-RAN Critical Alert: {{ .GroupLabels.alertname }}",
								Severity:    "critical",
								Component:   "{{ .GroupLabels.component }}",
							},
						},
					},
				},
				InhibitRules: []InhibitionRule{
					{
						SourceMatchers: []Matcher{
							{Name: "severity", Value: "critical", IsEqual: true},
						},
						TargetMatchers: []Matcher{
							{Name: "severity", Value: "warning", IsEqual: true},
						},
						Equal: []string{"component", "instance"},
					},
				},
			},
			expectedError: false,
			mockBehavior: func(m *MockAlertManagerClient) {
				m.On("ValidateConfiguration", mock.AnythingOfType("*monitoring.AlertManagerConfig")).Return(nil)
				m.On("DeployAlertManager", mock.Anything, mock.AnythingOfType("*monitoring.AlertManagerConfig")).Return(nil)
			},
		},
		{
			name: "invalid routing configuration",
			config: &AlertManagerConfig{
				Route: &Route{
					Receiver:       "nonexistent-receiver",
					GroupBy:        []string{},
					GroupWait:      0, // Invalid
					GroupInterval:  0, // Invalid
					RepeatInterval: 0, // Invalid
				},
				Receivers: []Receiver{}, // Empty receivers
			},
			expectedError: true,
			errorMessage:  "invalid routing configuration",
			mockBehavior: func(m *MockAlertManagerClient) {
				m.On("ValidateConfiguration", mock.AnythingOfType("*monitoring.AlertManagerConfig")).Return(errors.New("invalid routing configuration: receiver not found"))
			},
		},
		{
			name: "invalid notification configuration",
			config: &AlertManagerConfig{
				Route: &Route{
					Receiver: "invalid-receiver",
				},
				Receivers: []Receiver{
					{
						Name: "invalid-receiver",
						EmailConfigs: []EmailConfig{
							{
								To:      "invalid-email", // Invalid email format
								Subject: "", // Empty subject
							},
						},
					},
				},
			},
			expectedError: true,
			errorMessage:  "invalid email configuration",
			mockBehavior: func(m *MockAlertManagerClient) {
				m.On("ValidateConfiguration", mock.AnythingOfType("*monitoring.AlertManagerConfig")).Return(errors.New("invalid email configuration"))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockClient := &MockAlertManagerClient{}
			tc.mockBehavior(mockClient)
			deployer := &AlertManagerDeployer{client: mockClient}

			// Act
			err := deployer.ConfigureO_RANAlertManager(context.Background(), tc.config)

			// Assert
			if tc.expectedError {
				assert.Error(t, err)
				if tc.errorMessage != "" {
					assert.Contains(t, err.Error(), tc.errorMessage)
				}
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

// TestRoutingRules tests alert routing rules
func TestRoutingRules(t *testing.T) {
	testCases := []struct {
		name          string
		routes        []Route
		expectedError bool
		errorMessage  string
		mockBehavior  func(*MockAlertManagerClient)
	}{
		{
			name: "valid o-ran component routing",
			routes: []Route{
				{
					Receiver: "orchestrator-team",
					Matchers: []Matcher{
						{Name: "component", Value: "orchestrator", IsEqual: true},
					},
					GroupBy:        []string{"alertname", "severity"},
					GroupWait:      10 * time.Second,
					GroupInterval:  5 * time.Minute,
					RepeatInterval: 1 * time.Hour,
				},
				{
					Receiver: "vnf-operator-team",
					Matchers: []Matcher{
						{Name: "component", Value: "vnf-operator", IsEqual: true},
					},
					GroupBy:        []string{"alertname", "vnf_type"},
					GroupWait:      15 * time.Second,
					GroupInterval:  10 * time.Minute,
					RepeatInterval: 2 * time.Hour,
				},
				{
					Receiver: "dms-team",
					Matchers: []Matcher{
						{Name: "component", Value: "dms", IsEqual: true},
					},
					GroupBy:        []string{"alertname", "dms_role"},
					GroupWait:      20 * time.Second,
					GroupInterval:  15 * time.Minute,
					RepeatInterval: 4 * time.Hour,
				},
			},
			expectedError: false,
			mockBehavior: func(m *MockAlertManagerClient) {
				m.On("ConfigureRouting", mock.Anything, mock.AnythingOfType("[]monitoring.Route")).Return(nil)
			},
		},
		{
			name: "severity-based routing",
			routes: []Route{
				{
					Receiver: "critical-alerts",
					Matchers: []Matcher{
						{Name: "severity", Value: "critical", IsEqual: true},
						{Name: "namespace", Value: "o-ran-.*", IsRegex: true, IsEqual: true},
					},
					GroupBy:        []string{"alertname", "component"},
					GroupWait:      0 * time.Second, // Immediate
					GroupInterval:  1 * time.Minute,
					RepeatInterval: 30 * time.Minute,
					Continue:       true, // Continue to other routes
				},
				{
					Receiver: "warning-alerts",
					Matchers: []Matcher{
						{Name: "severity", Value: "warning", IsEqual: true},
						{Name: "namespace", Value: "o-ran-.*", IsRegex: true, IsEqual: true},
					},
					GroupBy:        []string{"alertname"},
					GroupWait:      5 * time.Minute,
					GroupInterval:  30 * time.Minute,
					RepeatInterval: 24 * time.Hour,
				},
			},
			expectedError: false,
			mockBehavior: func(m *MockAlertManagerClient) {
				m.On("ConfigureRouting", mock.Anything, mock.AnythingOfType("[]monitoring.Route")).Return(nil)
			},
		},
		{
			name: "invalid matcher configuration",
			routes: []Route{
				{
					Receiver: "invalid-route",
					Matchers: []Matcher{
						{Name: "component", Value: "[invalid-regex", IsRegex: true, IsEqual: true}, // Invalid regex
					},
				},
			},
			expectedError: true,
			errorMessage:  "invalid matcher regex",
			mockBehavior: func(m *MockAlertManagerClient) {
				m.On("ConfigureRouting", mock.Anything, mock.AnythingOfType("[]monitoring.Route")).Return(errors.New("invalid matcher regex"))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockClient := &MockAlertManagerClient{}
			tc.mockBehavior(mockClient)
			deployer := &AlertManagerDeployer{client: mockClient}

			// Act
			err := deployer.ConfigureO_RANRouting(context.Background(), tc.routes)

			// Assert
			if tc.expectedError {
				assert.Error(t, err)
				if tc.errorMessage != "" {
					assert.Contains(t, err.Error(), tc.errorMessage)
				}
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

// TestNotificationReceivers tests notification receiver configuration
func TestNotificationReceivers(t *testing.T) {
	testCases := []struct {
		name          string
		receivers     []Receiver
		expectedError bool
		errorMessage  string
		mockBehavior  func(*MockAlertManagerClient)
	}{
		{
			name: "multi-channel o-ran receivers",
			receivers: []Receiver{
				{
					Name: "o-ran-ops-team",
					EmailConfigs: []EmailConfig{
						{
							To:      "ops@o-ran.company.com",
							From:    "alertmanager@o-ran.company.com",
							Subject: "[O-RAN {{ .Status | toUpper }}] {{ .GroupLabels.alertname }}",
							Body:    "{{ range .Alerts }}Component: {{ .Labels.component }}\nSummary: {{ .Annotations.summary }}\nDescription: {{ .Annotations.description }}\n{{ end }}",
							HTML:    "<h2>O-RAN Alert</h2>{{ range .Alerts }}<p><strong>{{ .Labels.component }}</strong>: {{ .Annotations.summary }}</p>{{ end }}",
						},
					},
					SlackConfigs: []SlackConfig{
						{
							Channel:   "#o-ran-operations",
							Username:  "O-RAN AlertManager",
							Color:     "{{ if eq .Status \"firing\" }}danger{{ else }}good{{ end }}",
							Title:     "O-RAN {{ .Status | title }} Alert",
							TitleLink: "{{ (index .Alerts 0).GeneratorURL }}",
							Text:      "{{ range .Alerts }}*{{ .Labels.component }}*: {{ .Annotations.summary }}{{ end }}",
							Fields: []SlackField{
								{Title: "Component", Value: "{{ .GroupLabels.component }}", Short: true},
								{Title: "Severity", Value: "{{ .GroupLabels.severity }}", Short: true},
								{Title: "Namespace", Value: "{{ .GroupLabels.namespace }}", Short: true},
								{Title: "Instance", Value: "{{ .GroupLabels.instance }}", Short: true},
							},
							Footer:    "O-RAN MANO Platform",
							IconEmoji: ":warning:",
						},
					},
					WebhookConfigs: []WebhookConfig{
						{
							URL:       "https://webhook.o-ran.company.com/alerts",
							MaxAlerts: 10,
							HTTPConfig: &HTTPConfig{
								TLSConfig: &TLSConfig{
									InsecureSkipVerify: false,
									CertFile:           "/etc/ssl/certs/webhook.crt",
									KeyFile:            "/etc/ssl/private/webhook.key",
								},
							},
						},
					},
				},
				{
					Name: "o-ran-critical-oncall",
					PagerDutyConfigs: []PagerDutyConfig{
						{
							ServiceKey:  "YOUR_PAGERDUTY_SERVICE_KEY",
							Description: "Critical O-RAN Alert: {{ .GroupLabels.alertname }}",
							Severity:    "critical",
							Class:       "o-ran",
							Component:   "{{ .GroupLabels.component }}",
							Group:       "{{ .GroupLabels.namespace }}",
							Details: map[string]interface{}{
								"summary":     "{{ (index .Alerts 0).Annotations.summary }}",
								"description": "{{ (index .Alerts 0).Annotations.description }}",
								"runbook_url": "{{ (index .Alerts 0).Annotations.runbook_url }}",
							},
						},
					},
				},
			},
			expectedError: false,
			mockBehavior: func(m *MockAlertManagerClient) {
				m.On("ConfigureReceivers", mock.Anything, mock.AnythingOfType("[]monitoring.Receiver")).Return(nil)
			},
		},
		{
			name: "invalid slack configuration",
			receivers: []Receiver{
				{
					Name: "invalid-slack",
					SlackConfigs: []SlackConfig{
						{
							APIURL:  "invalid-url", // Invalid URL
							Channel: "", // Empty channel
						},
					},
				},
			},
			expectedError: true,
			errorMessage:  "invalid slack configuration",
			mockBehavior: func(m *MockAlertManagerClient) {
				m.On("ConfigureReceivers", mock.Anything, mock.AnythingOfType("[]monitoring.Receiver")).Return(errors.New("invalid slack configuration"))
			},
		},
		{
			name: "webhook with invalid URL",
			receivers: []Receiver{
				{
					Name: "invalid-webhook",
					WebhookConfigs: []WebhookConfig{
						{
							URL:       "not-a-valid-url",
							MaxAlerts: -1, // Invalid
						},
					},
				},
			},
			expectedError: true,
			errorMessage:  "invalid webhook URL",
			mockBehavior: func(m *MockAlertManagerClient) {
				m.On("ConfigureReceivers", mock.Anything, mock.AnythingOfType("[]monitoring.Receiver")).Return(errors.New("invalid webhook URL"))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockClient := &MockAlertManagerClient{}
			tc.mockBehavior(mockClient)
			deployer := &AlertManagerDeployer{client: mockClient}

			// Act
			err := deployer.ConfigureO_RANReceivers(context.Background(), tc.receivers)

			// Assert
			if tc.expectedError {
				assert.Error(t, err)
				if tc.errorMessage != "" {
					assert.Contains(t, err.Error(), tc.errorMessage)
				}
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

// TestAlertGrouping tests alert grouping by severity
func TestAlertGrouping(t *testing.T) {
	testCases := []struct {
		name          string
		groupBy       []string
		groupWait     time.Duration
		groupInterval time.Duration
		expectedError bool
		errorMessage  string
		mockBehavior  func(*MockAlertManagerClient)
	}{
		{
			name:          "group by severity and component",
			groupBy:       []string{"severity", "component", "namespace"},
			groupWait:     10 * time.Second,
			groupInterval: 5 * time.Minute,
			expectedError: false,
			mockBehavior: func(m *MockAlertManagerClient) {
				// Mock would validate grouping configuration
			},
		},
		{
			name:          "group by alertname only",
			groupBy:       []string{"alertname"},
			groupWait:     30 * time.Second,
			groupInterval: 10 * time.Minute,
			expectedError: false,
			mockBehavior: func(m *MockAlertManagerClient) {
				// Mock would validate grouping configuration
			},
		},
		{
			name:          "invalid grouping interval",
			groupBy:       []string{"severity"},
			groupWait:     1 * time.Hour, // Too long
			groupInterval: 1 * time.Second, // Too short
			expectedError: true,
			errorMessage:  "invalid grouping intervals",
			mockBehavior: func(m *MockAlertManagerClient) {
				// Mock would reject invalid intervals
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockClient := &MockAlertManagerClient{}
			tc.mockBehavior(mockClient)
			deployer := &AlertManagerDeployer{client: mockClient}

			// Act
			err := deployer.ValidateO_RANGrouping(tc.groupBy, tc.groupWait, tc.groupInterval)

			// Assert
			if tc.expectedError {
				assert.Error(t, err)
				if tc.errorMessage != "" {
					assert.Contains(t, err.Error(), tc.errorMessage)
				}
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

// TestInhibitionRules tests inhibition rule configuration
func TestInhibitionRules(t *testing.T) {
	testCases := []struct {
		name          string
		rules         []InhibitionRule
		expectedError bool
		errorMessage  string
		mockBehavior  func(*MockAlertManagerClient)
	}{
		{
			name: "critical inhibits warning for same component",
			rules: []InhibitionRule{
				{
					SourceMatchers: []Matcher{
						{Name: "severity", Value: "critical", IsEqual: true},
					},
					TargetMatchers: []Matcher{
						{Name: "severity", Value: "warning", IsEqual: true},
					},
					Equal: []string{"component", "instance", "namespace"},
				},
				{
					SourceMatchers: []Matcher{
						{Name: "alertname", Value: "O_RANComponentDown", IsEqual: true},
					},
					TargetMatchers: []Matcher{
						{Name: "alertname", Value: "O_RANHighLatency|O_RANLowThroughput", IsRegex: true, IsEqual: true},
					},
					Equal: []string{"component", "instance"},
				},
			},
			expectedError: false,
			mockBehavior: func(m *MockAlertManagerClient) {
				m.On("SetInhibitionRules", mock.Anything, mock.AnythingOfType("[]monitoring.InhibitionRule")).Return(nil)
			},
		},
		{
			name: "invalid inhibition rule - empty matchers",
			rules: []InhibitionRule{
				{
					SourceMatchers: []Matcher{}, // Empty
					TargetMatchers: []Matcher{}, // Empty
					Equal:          []string{},  // Empty
				},
			},
			expectedError: true,
			errorMessage:  "empty inhibition matchers",
			mockBehavior: func(m *MockAlertManagerClient) {
				m.On("SetInhibitionRules", mock.Anything, mock.AnythingOfType("[]monitoring.InhibitionRule")).Return(errors.New("empty inhibition matchers"))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockClient := &MockAlertManagerClient{}
			tc.mockBehavior(mockClient)
			deployer := &AlertManagerDeployer{client: mockClient}

			// Act
			err := deployer.ConfigureO_RANInhibition(context.Background(), tc.rules)

			// Assert
			if tc.expectedError {
				assert.Error(t, err)
				if tc.errorMessage != "" {
					assert.Contains(t, err.Error(), tc.errorMessage)
				}
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

// TestSilenceManagement tests silence creation and management
func TestSilenceManagement(t *testing.T) {
	testCases := []struct {
		name          string
		silence       *Silence
		expectedError bool
		errorMessage  string
		mockBehavior  func(*MockAlertManagerClient)
	}{
		{
			name: "create maintenance silence",
			silence: &Silence{
				Matchers: []Matcher{
					{Name: "component", Value: "orchestrator", IsEqual: true},
					{Name: "namespace", Value: "o-ran-mano", IsEqual: true},
				},
				StartsAt:  time.Now(),
				EndsAt:    time.Now().Add(2 * time.Hour),
				CreatedBy: "o-ran-ops@company.com",
				Comment:   "Planned maintenance window for orchestrator upgrade",
			},
			expectedError: false,
			mockBehavior: func(m *MockAlertManagerClient) {
				m.On("CreateSilence", mock.Anything, mock.AnythingOfType("*monitoring.Silence")).Return(&SilenceResponse{SilenceID: "silence-123"}, nil)
			},
		},
		{
			name: "create component-specific silence",
			silence: &Silence{
				Matchers: []Matcher{
					{Name: "alertname", Value: "O_RANHighCPUUsage", IsEqual: true},
					{Name: "component", Value: "vnf-operator", IsEqual: true},
					{Name: "instance", Value: "vnf-operator-.*", IsRegex: true, IsEqual: true},
				},
				StartsAt:  time.Now(),
				EndsAt:    time.Now().Add(24 * time.Hour),
				CreatedBy: "vnf-team@company.com",
				Comment:   "Known issue during high VNF deployment load",
			},
			expectedError: false,
			mockBehavior: func(m *MockAlertManagerClient) {
				m.On("CreateSilence", mock.Anything, mock.AnythingOfType("*monitoring.Silence")).Return(&SilenceResponse{SilenceID: "silence-456"}, nil)
			},
		},
		{
			name: "invalid silence - end before start",
			silence: &Silence{
				Matchers: []Matcher{
					{Name: "component", Value: "dms", IsEqual: true},
				},
				StartsAt:  time.Now(),
				EndsAt:    time.Now().Add(-1 * time.Hour), // Invalid: ends before start
				CreatedBy: "user@company.com",
				Comment:   "Invalid silence",
			},
			expectedError: true,
			errorMessage:  "invalid silence duration",
			mockBehavior: func(m *MockAlertManagerClient) {
				m.On("CreateSilence", mock.Anything, mock.AnythingOfType("*monitoring.Silence")).Return(nil, errors.New("invalid silence duration"))
			},
		},
		{
			name: "silence with empty matchers",
			silence: &Silence{
				Matchers:  []Matcher{}, // Empty matchers
				StartsAt:  time.Now(),
				EndsAt:    time.Now().Add(1 * time.Hour),
				CreatedBy: "user@company.com",
				Comment:   "Silence with no matchers",
			},
			expectedError: true,
			errorMessage:  "empty silence matchers",
			mockBehavior: func(m *MockAlertManagerClient) {
				m.On("CreateSilence", mock.Anything, mock.AnythingOfType("*monitoring.Silence")).Return(nil, errors.New("empty silence matchers"))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockClient := &MockAlertManagerClient{}
			tc.mockBehavior(mockClient)
			deployer := &AlertManagerDeployer{client: mockClient}

			// Act
			_, err := deployer.CreateO_RANSilence(context.Background(), tc.silence)

			// Assert
			if tc.expectedError {
				assert.Error(t, err)
				if tc.errorMessage != "" {
					assert.Contains(t, err.Error(), tc.errorMessage)
				}
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

// These methods will need to be implemented in the actual AlertManagerDeployer
// They are defined here to make the tests compile and FAIL (RED phase)
func (amd *AlertManagerDeployer) ConfigureO_RANAlertManager(ctx context.Context, config *AlertManagerConfig) error {
	panic("not implemented - this test should FAIL in RED phase")
}

func (amd *AlertManagerDeployer) ConfigureO_RANRouting(ctx context.Context, routes []Route) error {
	panic("not implemented - this test should FAIL in RED phase")
}

func (amd *AlertManagerDeployer) ConfigureO_RANReceivers(ctx context.Context, receivers []Receiver) error {
	panic("not implemented - this test should FAIL in RED phase")
}

func (amd *AlertManagerDeployer) ValidateO_RANGrouping(groupBy []string, groupWait, groupInterval time.Duration) error {
	panic("not implemented - this test should FAIL in RED phase")
}

func (amd *AlertManagerDeployer) ConfigureO_RANInhibition(ctx context.Context, rules []InhibitionRule) error {
	panic("not implemented - this test should FAIL in RED phase")
}

func (amd *AlertManagerDeployer) CreateO_RANSilence(ctx context.Context, silence *Silence) (*SilenceResponse, error) {
	panic("not implemented - this test should FAIL in RED phase")
}
