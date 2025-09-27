package monitoring

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/api/core/v1"
)

// MockServiceMonitorClient mocks the ServiceMonitor client interface
type MockServiceMonitorClient struct {
	mock.Mock
}

// ServiceMonitorManager handles ServiceMonitor CRD operations
type ServiceMonitorManager struct {
	client ServiceMonitorClientInterface
}

// ServiceMonitorClientInterface defines the contract for ServiceMonitor operations
type ServiceMonitorClientInterface interface {
	CreateServiceMonitor(ctx context.Context, serviceMonitor *monitoringv1.ServiceMonitor) error
	UpdateServiceMonitor(ctx context.Context, serviceMonitor *monitoringv1.ServiceMonitor) error
	DeleteServiceMonitor(ctx context.Context, namespace, name string) error
	GetServiceMonitor(ctx context.Context, namespace, name string) (*monitoringv1.ServiceMonitor, error)
	ListServiceMonitors(ctx context.Context, namespace string) ([]monitoringv1.ServiceMonitor, error)
	ValidateServiceMonitor(serviceMonitor *monitoringv1.ServiceMonitor) error
	TestEndpointConnectivity(ctx context.Context, endpoint *monitoringv1.Endpoint, selector *metav1.LabelSelector) error
	ValidateTLSConfig(tlsConfig *monitoringv1.TLSConfig) error
	TestAuthentication(ctx context.Context, authConfig *AuthenticationConfig) error
}

// AuthenticationConfig represents authentication configuration
type AuthenticationConfig struct {
	Type         string // "bearer", "basic", "oauth2"
	BearerToken  string
	Username     string
	Password     string
	OAuth2Config *OAuth2Config
	TLSConfig    *monitoringv1.TLSConfig
}

// OAuth2Config represents OAuth2 configuration
type OAuth2Config struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	Scopes       []string
}

// Test cases for ServiceMonitor CRD creation
func TestServiceMonitorCreation(t *testing.T) {
	testCases := []struct {
		name           string
		serviceMonitor *monitoringv1.ServiceMonitor
		expectedError  bool
		errorMessage   string
		mockBehavior   func(*MockServiceMonitorClient)
	}{
		{
			name: "create orchestrator service monitor",
			serviceMonitor: &monitoringv1.ServiceMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "orchestrator-metrics",
					Namespace: "o-ran-mano",
					Labels: map[string]string{
						"app.kubernetes.io/name":      "orchestrator",
						"app.kubernetes.io/component": "o-ran-mano",
						"app.kubernetes.io/part-of":   "o-ran-platform",
						"prometheus":                  "o-ran",
					},
				},
				Spec: monitoringv1.ServiceMonitorSpec{
					Selector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/name": "orchestrator",
						},
					},
					Endpoints: []monitoringv1.Endpoint{
						{
							Port:     "metrics",
							Path:     "/metrics",
							Scheme:   "http",
							Interval: "30s",
							ScrapeTimeout: "10s",
							HonorLabels: false,
							Relabelings: []*monitoringv1.RelabelConfig{
								{
									SourceLabels: []monitoringv1.LabelName{"__meta_kubernetes_service_name"},
									TargetLabel:  "service",
								},
								{
									SourceLabels: []monitoringv1.LabelName{"__meta_kubernetes_namespace"},
									TargetLabel:  "namespace",
								},
							},
						},
					},
					NamespaceSelector: monitoringv1.NamespaceSelector{
						MatchNames: []string{"o-ran-mano"},
					},
				},
			},
			expectedError: false,
			mockBehavior: func(m *MockServiceMonitorClient) {
				m.On("ValidateServiceMonitor", mock.AnythingOfType("*v1.ServiceMonitor")).Return(nil)
				m.On("CreateServiceMonitor", mock.Anything, mock.AnythingOfType("*v1.ServiceMonitor")).Return(nil)
			},
		},
		{
			name: "create vnf-operator service monitor with TLS",
			serviceMonitor: &monitoringv1.ServiceMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vnf-operator-metrics",
					Namespace: "o-ran-mano",
					Labels: map[string]string{
						"app.kubernetes.io/name":      "vnf-operator",
						"app.kubernetes.io/component": "o-ran-mano",
						"prometheus":                  "o-ran",
					},
				},
				Spec: monitoringv1.ServiceMonitorSpec{
					Selector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/name": "vnf-operator",
						},
					},
					Endpoints: []monitoringv1.Endpoint{
						{
							Port:     "metrics",
							Path:     "/metrics",
							Scheme:   "https",
							Interval: "15s",
							ScrapeTimeout: "5s",
							TLSConfig: &monitoringv1.TLSConfig{
								SafeT LSConfig: monitoringv1.SafeTLSConfig{
									InsecureSkipVerify: false,
									ServerName:         "vnf-operator.o-ran-mano.svc.cluster.local",
								},
								CertFile: "/etc/ssl/certs/client.crt",
								KeyFile:  "/etc/ssl/private/client.key",
								CAFile:   "/etc/ssl/certs/ca.crt",
							},
							BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
						},
					},
					NamespaceSelector: monitoringv1.NamespaceSelector{
						MatchNames: []string{"o-ran-mano"},
					},
				},
			},
			expectedError: false,
			mockBehavior: func(m *MockServiceMonitorClient) {
				m.On("ValidateServiceMonitor", mock.AnythingOfType("*v1.ServiceMonitor")).Return(nil)
				m.On("ValidateTLSConfig", mock.AnythingOfType("*v1.TLSConfig")).Return(nil)
				m.On("CreateServiceMonitor", mock.Anything, mock.AnythingOfType("*v1.ServiceMonitor")).Return(nil)
			},
		},
		{
			name: "create dms service monitor with custom relabeling",
			serviceMonitor: &monitoringv1.ServiceMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dms-metrics",
					Namespace: "o-ran-mano",
					Labels: map[string]string{
						"app.kubernetes.io/name": "dms",
						"prometheus":             "o-ran",
					},
				},
				Spec: monitoringv1.ServiceMonitorSpec{
					Selector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/name": "dms",
						},
					},
					Endpoints: []monitoringv1.Endpoint{
						{
							Port:     "metrics",
							Path:     "/metrics",
							Scheme:   "http",
							Interval: "30s",
							ScrapeTimeout: "10s",
							Relabelings: []*monitoringv1.RelabelConfig{
								{
									SourceLabels: []monitoringv1.LabelName{"__name__"},
									Regex:        "go_.*",
									Action:       "drop",
								},
								{
									SourceLabels: []monitoringv1.LabelName{"__meta_kubernetes_pod_name"},
									TargetLabel:  "pod",
								},
								{
									SourceLabels: []monitoringv1.LabelName{"__meta_kubernetes_pod_label_dms_role"},
									TargetLabel:  "dms_role",
								},
							},
							MetricRelabelings: []*monitoringv1.RelabelConfig{
								{
									SourceLabels: []monitoringv1.LabelName{"__name__"},
									Regex:        "o_ran_(.*)",
									TargetLabel:  "__name__",
									Replacement:  "oran_${1}",
								},
							},
						},
					},
					NamespaceSelector: monitoringv1.NamespaceSelector{
						MatchNames: []string{"o-ran-mano"},
					},
				},
			},
			expectedError: false,
			mockBehavior: func(m *MockServiceMonitorClient) {
				m.On("ValidateServiceMonitor", mock.AnythingOfType("*v1.ServiceMonitor")).Return(nil)
				m.On("CreateServiceMonitor", mock.Anything, mock.AnythingOfType("*v1.ServiceMonitor")).Return(nil)
			},
		},
		{
			name: "invalid service monitor - empty selector",
			serviceMonitor: &monitoringv1.ServiceMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-monitor",
					Namespace: "o-ran-mano",
				},
				Spec: monitoringv1.ServiceMonitorSpec{
					Selector: metav1.LabelSelector{}, // Empty selector
					Endpoints: []monitoringv1.Endpoint{
						{
							Port: "metrics",
							Path: "/metrics",
						},
					},
				},
			},
			expectedError: true,
			errorMessage:  "empty selector",
			mockBehavior: func(m *MockServiceMonitorClient) {
				m.On("ValidateServiceMonitor", mock.AnythingOfType("*v1.ServiceMonitor")).Return(errors.New("validation failed: empty selector labels"))
			},
		},
		{
			name: "invalid service monitor - no endpoints",
			serviceMonitor: &monitoringv1.ServiceMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "no-endpoints",
					Namespace: "o-ran-mano",
				},
				Spec: monitoringv1.ServiceMonitorSpec{
					Selector: metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
					Endpoints: []monitoringv1.Endpoint{}, // No endpoints
				},
			},
			expectedError: true,
			errorMessage:  "no endpoints defined",
			mockBehavior: func(m *MockServiceMonitorClient) {
				m.On("ValidateServiceMonitor", mock.AnythingOfType("*v1.ServiceMonitor")).Return(errors.New("validation failed: no endpoints defined"))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockClient := &MockServiceMonitorClient{}
			tc.mockBehavior(mockClient)
			manager := &ServiceMonitorManager{client: mockClient}

			// Act
			err := manager.CreateO_RANServiceMonitor(context.Background(), tc.serviceMonitor)

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

// TestEndpointSelection tests endpoint selection by labels
func TestEndpointSelection(t *testing.T) {
	testCases := []struct {
		name          string
		selector      *metav1.LabelSelector
		expectedError bool
		errorMessage  string
		mockBehavior  func(*MockServiceMonitorClient)
	}{
		{
			name: "select by component label",
			selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name":      "orchestrator",
					"app.kubernetes.io/component": "o-ran-mano",
				},
			},
			expectedError: false,
			mockBehavior: func(m *MockServiceMonitorClient) {
				m.On("TestEndpointConnectivity", mock.Anything, mock.AnythingOfType("*v1.Endpoint"), mock.AnythingOfType("*v1.LabelSelector")).Return(nil)
			},
		},
		{
			name: "select by multiple labels with expressions",
			selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/part-of": "o-ran-platform",
				},
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "app.kubernetes.io/name",
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{"orchestrator", "vnf-operator", "dms"},
					},
					{
						Key:      "environment",
						Operator: metav1.LabelSelectorOpNotIn,
						Values:   []string{"development"},
					},
				},
			},
			expectedError: false,
			mockBehavior: func(m *MockServiceMonitorClient) {
				m.On("TestEndpointConnectivity", mock.Anything, mock.AnythingOfType("*v1.Endpoint"), mock.AnythingOfType("*v1.LabelSelector")).Return(nil)
			},
		},
		{
			name: "select by version constraint",
			selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": "vnf-operator",
				},
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "app.kubernetes.io/version",
						Operator: metav1.LabelSelectorOpExists,
					},
					{
						Key:      "beta",
						Operator: metav1.LabelSelectorOpDoesNotExist,
					},
				},
			},
			expectedError: false,
			mockBehavior: func(m *MockServiceMonitorClient) {
				m.On("TestEndpointConnectivity", mock.Anything, mock.AnythingOfType("*v1.Endpoint"), mock.AnythingOfType("*v1.LabelSelector")).Return(nil)
			},
		},
		{
			name:          "no matching endpoints",
			selector:      &metav1.LabelSelector{MatchLabels: map[string]string{"nonexistent": "label"}},
			expectedError: true,
			errorMessage:  "no matching endpoints",
			mockBehavior: func(m *MockServiceMonitorClient) {
				m.On("TestEndpointConnectivity", mock.Anything, mock.AnythingOfType("*v1.Endpoint"), mock.AnythingOfType("*v1.LabelSelector")).Return(errors.New("no matching endpoints found"))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockClient := &MockServiceMonitorClient{}
			tc.mockBehavior(mockClient)
			manager := &ServiceMonitorManager{client: mockClient}

			// Act
			err := manager.TestO_RANEndpointSelection(context.Background(), tc.selector)

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

// TestTLSConfiguration tests TLS configuration for secure metrics
func TestTLSConfiguration(t *testing.T) {
	testCases := []struct {
		name          string
		tlsConfig     *monitoringv1.TLSConfig
		expectedError bool
		errorMessage  string
		mockBehavior  func(*MockServiceMonitorClient)
	}{
		{
			name: "valid TLS configuration with certificates",
			tlsConfig: &monitoringv1.TLSConfig{
				SafeTLSConfig: monitoringv1.SafeTLSConfig{
					InsecureSkipVerify: false,
					ServerName:         "orchestrator.o-ran-mano.svc.cluster.local",
				},
				CertFile: "/etc/ssl/certs/client.crt",
				KeyFile:  "/etc/ssl/private/client.key",
				CAFile:   "/etc/ssl/certs/ca.crt",
			},
			expectedError: false,
			mockBehavior: func(m *MockServiceMonitorClient) {
				m.On("ValidateTLSConfig", mock.AnythingOfType("*v1.TLSConfig")).Return(nil)
			},
		},
		{
			name: "TLS with insecure skip verify",
			tlsConfig: &monitoringv1.TLSConfig{
				SafeTLSConfig: monitoringv1.SafeTLSConfig{
					InsecureSkipVerify: true,
					ServerName:         "vnf-operator.o-ran-mano.svc.cluster.local",
				},
			},
			expectedError: false,
			mockBehavior: func(m *MockServiceMonitorClient) {
				m.On("ValidateTLSConfig", mock.AnythingOfType("*v1.TLSConfig")).Return(nil)
			},
		},
		{
			name: "TLS with secret references",
			tlsConfig: &monitoringv1.TLSConfig{
				SafeTLSConfig: monitoringv1.SafeTLSConfig{
					InsecureSkipVerify: false,
					ServerName:         "dms.o-ran-mano.svc.cluster.local",
					Cert: monitoringv1.SecretOrConfigMap{
						Secret: &v1.SecretKeySelector{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "dms-tls-cert",
							},
							Key: "tls.crt",
						},
					},
					KeySecret: &v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{
							Name: "dms-tls-cert",
						},
						Key: "tls.key",
					},
					CA: monitoringv1.SecretOrConfigMap{
						ConfigMap: &v1.ConfigMapKeySelector{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "o-ran-ca-bundle",
							},
							Key: "ca.crt",
						},
					},
				},
			},
			expectedError: false,
			mockBehavior: func(m *MockServiceMonitorClient) {
				m.On("ValidateTLSConfig", mock.AnythingOfType("*v1.TLSConfig")).Return(nil)
			},
		},
		{
			name: "invalid TLS - missing certificate",
			tlsConfig: &monitoringv1.TLSConfig{
				SafeTLSConfig: monitoringv1.SafeTLSConfig{
					InsecureSkipVerify: false,
					ServerName:         "secure-service.o-ran-mano.svc.cluster.local",
				},
				// Missing certificate configuration
			},
			expectedError: true,
			errorMessage:  "missing TLS certificate",
			mockBehavior: func(m *MockServiceMonitorClient) {
				m.On("ValidateTLSConfig", mock.AnythingOfType("*v1.TLSConfig")).Return(errors.New("missing TLS certificate configuration"))
			},
		},
		{
			name: "invalid TLS - mismatched server name",
			tlsConfig: &monitoringv1.TLSConfig{
				SafeTLSConfig: monitoringv1.SafeTLSConfig{
					InsecureSkipVerify: false,
					ServerName:         "wrong-hostname", // Doesn't match actual service
				},
				CertFile: "/etc/ssl/certs/client.crt",
				KeyFile:  "/etc/ssl/private/client.key",
				CAFile:   "/etc/ssl/certs/ca.crt",
			},
			expectedError: true,
			errorMessage:  "server name mismatch",
			mockBehavior: func(m *MockServiceMonitorClient) {
				m.On("ValidateTLSConfig", mock.AnythingOfType("*v1.TLSConfig")).Return(errors.New("server name mismatch"))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockClient := &MockServiceMonitorClient{}
			tc.mockBehavior(mockClient)
			manager := &ServiceMonitorManager{client: mockClient}

			// Act
			err := manager.ValidateO_RANTLSConfig(tc.tlsConfig)

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

// TestAuthentication tests authentication configuration
func TestAuthentication(t *testing.T) {
	testCases := []struct {
		name          string
		authConfig    *AuthenticationConfig
		expectedError bool
		errorMessage  string
		mockBehavior  func(*MockServiceMonitorClient)
	}{
		{
			name: "bearer token authentication",
			authConfig: &AuthenticationConfig{
				Type:        "bearer",
				BearerToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
			},
			expectedError: false,
			mockBehavior: func(m *MockServiceMonitorClient) {
				m.On("TestAuthentication", mock.Anything, mock.AnythingOfType("*monitoring.AuthenticationConfig")).Return(nil)
			},
		},
		{
			name: "basic authentication",
			authConfig: &AuthenticationConfig{
				Type:     "basic",
				Username: "prometheus",
				Password: "secure-password-123",
			},
			expectedError: false,
			mockBehavior: func(m *MockServiceMonitorClient) {
				m.On("TestAuthentication", mock.Anything, mock.AnythingOfType("*monitoring.AuthenticationConfig")).Return(nil)
			},
		},
		{
			name: "oauth2 authentication",
			authConfig: &AuthenticationConfig{
				Type: "oauth2",
				OAuth2Config: &OAuth2Config{
					ClientID:     "o-ran-prometheus-client",
					ClientSecret: "oauth2-client-secret",
					TokenURL:     "https://auth.o-ran.company.com/oauth2/token",
					Scopes:       []string{"metrics:read", "o-ran:monitor"},
				},
			},
			expectedError: false,
			mockBehavior: func(m *MockServiceMonitorClient) {
				m.On("TestAuthentication", mock.Anything, mock.AnythingOfType("*monitoring.AuthenticationConfig")).Return(nil)
			},
		},
		{
			name: "bearer with TLS authentication",
			authConfig: &AuthenticationConfig{
				Type:        "bearer",
				BearerToken: "service-account-token",
				TLSConfig: &monitoringv1.TLSConfig{
					SafeTLSConfig: monitoringv1.SafeTLSConfig{
						InsecureSkipVerify: false,
						ServerName:         "secure-metrics.o-ran-mano.svc.cluster.local",
					},
					CAFile: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
				},
			},
			expectedError: false,
			mockBehavior: func(m *MockServiceMonitorClient) {
				m.On("TestAuthentication", mock.Anything, mock.AnythingOfType("*monitoring.AuthenticationConfig")).Return(nil)
			},
		},
		{
			name: "invalid authentication - empty credentials",
			authConfig: &AuthenticationConfig{
				Type: "bearer",
				// Missing bearer token
			},
			expectedError: true,
			errorMessage:  "missing credentials",
			mockBehavior: func(m *MockServiceMonitorClient) {
				m.On("TestAuthentication", mock.Anything, mock.AnythingOfType("*monitoring.AuthenticationConfig")).Return(errors.New("missing credentials for bearer authentication"))
			},
		},
		{
			name: "invalid authentication - unsupported type",
			authConfig: &AuthenticationConfig{
				Type:     "unsupported",
				Username: "user",
				Password: "pass",
			},
			expectedError: true,
			errorMessage:  "unsupported authentication type",
			mockBehavior: func(m *MockServiceMonitorClient) {
				m.On("TestAuthentication", mock.Anything, mock.AnythingOfType("*monitoring.AuthenticationConfig")).Return(errors.New("unsupported authentication type: unsupported"))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockClient := &MockServiceMonitorClient{}
			tc.mockBehavior(mockClient)
			manager := &ServiceMonitorManager{client: mockClient}

			// Act
			err := manager.TestO_RANAuthentication(context.Background(), tc.authConfig)

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

// TestScrapeFailureHandling tests scrape failure handling
func TestScrapeFailureHandling(t *testing.T) {
	testCases := []struct {
		name          string
		failureType   string
		expectedError bool
		errorMessage  string
		mockBehavior  func(*MockServiceMonitorClient)
	}{
		{
			name:        "connection timeout",
			failureType: "timeout",
			expectedError: false, // Should handle gracefully
			mockBehavior: func(m *MockServiceMonitorClient) {
				// Mock timeout handling
			},
		},
		{
			name:        "endpoint not found",
			failureType: "not_found",
			expectedError: false, // Should handle gracefully
			mockBehavior: func(m *MockServiceMonitorClient) {
				// Mock 404 handling
			},
		},
		{
			name:        "authentication failure",
			failureType: "auth_failed",
			expectedError: true,
			errorMessage: "authentication failed",
			mockBehavior: func(m *MockServiceMonitorClient) {
				// Mock auth failure
			},
		},
		{
			name:        "TLS certificate error",
			failureType: "tls_error",
			expectedError: true,
			errorMessage: "TLS certificate verification failed",
			mockBehavior: func(m *MockServiceMonitorClient) {
				// Mock TLS error
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockClient := &MockServiceMonitorClient{}
			tc.mockBehavior(mockClient)
			manager := &ServiceMonitorManager{client: mockClient}

			// Act
			err := manager.TestO_RANScrapeFailureHandling(tc.failureType)

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

// These methods will need to be implemented in the actual ServiceMonitorManager
// They are defined here to make the tests compile and FAIL (RED phase)
func (smm *ServiceMonitorManager) CreateO_RANServiceMonitor(ctx context.Context, serviceMonitor *monitoringv1.ServiceMonitor) error {
	panic("not implemented - this test should FAIL in RED phase")
}

func (smm *ServiceMonitorManager) TestO_RANEndpointSelection(ctx context.Context, selector *metav1.LabelSelector) error {
	panic("not implemented - this test should FAIL in RED phase")
}

func (smm *ServiceMonitorManager) ValidateO_RANTLSConfig(tlsConfig *monitoringv1.TLSConfig) error {
	panic("not implemented - this test should FAIL in RED phase")
}

func (smm *ServiceMonitorManager) TestO_RANAuthentication(ctx context.Context, authConfig *AuthenticationConfig) error {
	panic("not implemented - this test should FAIL in RED phase")
}

func (smm *ServiceMonitorManager) TestO_RANScrapeFailureHandling(failureType string) error {
	panic("not implemented - this test should FAIL in RED phase")
}
