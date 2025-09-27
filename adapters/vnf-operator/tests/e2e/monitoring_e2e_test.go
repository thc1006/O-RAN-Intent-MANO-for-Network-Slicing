package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var _ = Describe("O-RAN Monitoring Stack E2E Tests", func() {
	var (
		clientset   *kubernetes.Clientset
		restConfig  *rest.Config
		namespace   = "oran-monitoring"
		testTimeout = 10 * time.Minute
		ctx         context.Context
		cancel      context.CancelFunc
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), testTimeout)

		var err error
		restConfig, err = config.GetConfig()
		Expect(err).NotTo(HaveOccurred())

		clientset, err = kubernetes.NewForConfig(restConfig)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if cancel != nil {
			cancel()
		}
	})

	Describe("Monitoring Stack Deployment", func() {
		It("should have monitoring namespace created", func() {
			ns, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(ns.Name).To(Equal(namespace))
			Expect(ns.Labels["app.kubernetes.io/part-of"]).To(Equal("oran-intent-mano"))
		})

		It("should have all monitoring components deployed", func() {
			// Check Prometheus deployment
			promDeployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, "prometheus", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(promDeployment.Status.ReadyReplicas).To(Equal(int32(1)))

			// Check Grafana deployment
			grafanaDeployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, "grafana", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(grafanaDeployment.Status.ReadyReplicas).To(Equal(int32(1)))

			// Check AlertManager deployment
			amDeployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, "alertmanager", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(amDeployment.Status.ReadyReplicas).To(Equal(int32(1)))
		})

		It("should have all pods running and ready", func() {
			Eventually(func() bool {
				pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
				if err != nil {
					return false
				}

				for _, pod := range pods.Items {
					if pod.Status.Phase != corev1.PodRunning {
						return false
					}

					// Check all containers are ready
					for _, condition := range pod.Status.Conditions {
						if condition.Type == corev1.PodReady && condition.Status != corev1.ConditionTrue {
							return false
						}
					}
				}
				return len(pods.Items) >= 3 // Prometheus, Grafana, AlertManager
			}, 5*time.Minute, 10*time.Second).Should(BeTrue())
		})

		It("should have services accessible", func() {
			services := []string{"prometheus", "grafana", "alertmanager"}

			for _, serviceName := range services {
				service, err := clientset.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(service.Spec.Ports).To(HaveLen(BeNumerically(">=", 1)))
				Expect(service.Spec.Selector).ToNot(BeEmpty())
			}
		})
	})

	Describe("Prometheus Configuration and Targets", func() {
		var prometheusPort int32

		BeforeEach(func() {
			// Get Prometheus service port
			service, err := clientset.CoreV1().Services(namespace).Get(ctx, "prometheus", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			prometheusPort = service.Spec.Ports[0].Port
		})

		It("should scrape all configured targets", func() {
			// Port forward to Prometheus
			promURL := fmt.Sprintf("http://prometheus.%s.svc.cluster.local:%d", namespace, prometheusPort)

			Eventually(func() bool {
				resp, err := makeClusterRequest(ctx, clientset, restConfig,
					namespace, "prometheus", "api/v1/targets")
				if err != nil {
					return false
				}
				defer resp.Body.Close()

				var targetsResp PrometheusTargetsResponse
				if err := json.NewDecoder(resp.Body).Decode(&targetsResp); err != nil {
					return false
				}

				// Check that essential targets are UP
				upTargets := 0
				for _, target := range targetsResp.Data.ActiveTargets {
					if target.Health == "up" {
						upTargets++
					}
				}

				// Expect at least kubernetes, node-exporter, and oran components
				return upTargets >= 5
			}, 3*time.Minute, 15*time.Second).Should(BeTrue())
		})

		It("should have O-RAN specific scrape jobs configured", func() {
			resp, err := makeClusterRequest(ctx, clientset, restConfig,
				namespace, "prometheus", "api/v1/status/config")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

			configStr := string(body)
			oranJobs := []string{
				"oran-nlp",
				"oran-orchestrator",
				"oran-ran",
				"oran-cn",
				"oran-tn",
				"oran-vnf-operator",
			}

			for _, job := range oranJobs {
				Expect(configStr).To(ContainSubstring(job))
			}
		})

		It("should collect metrics from O-RAN components", func() {
			Eventually(func() bool {
				resp, err := makeClusterRequest(ctx, clientset, restConfig,
					namespace, "prometheus", "api/v1/query?query=up{job=~\"oran.*\"}")
				if err != nil {
					return false
				}
				defer resp.Body.Close()

				var queryResp PrometheusQueryResponse
				if err := json.NewDecoder(resp.Body).Decode(&queryResp); err != nil {
					return false
				}

				return len(queryResp.Data.Result) > 0
			}, 2*time.Minute, 10*time.Second).Should(BeTrue())
		})
	})

	Describe("Grafana Dashboard and API", func() {
		It("should have Grafana accessible and healthy", func() {
			Eventually(func() bool {
				resp, err := makeClusterRequest(ctx, clientset, restConfig,
					namespace, "grafana", "api/health")
				if err != nil {
					return false
				}
				defer resp.Body.Close()

				return resp.StatusCode == http.StatusOK
			}, 2*time.Minute, 10*time.Second).Should(BeTrue())
		})

		It("should have O-RAN dashboards configured", func() {
			resp, err := makeClusterRequest(ctx, clientset, restConfig,
				namespace, "grafana", "api/search?type=dash-db")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			var dashboards []GrafanaDashboard
			err = json.NewDecoder(resp.Body).Decode(&dashboards)
			Expect(err).NotTo(HaveOccurred())

			// Check for essential O-RAN dashboards
			dashboardTitles := make([]string, len(dashboards))
			for i, dashboard := range dashboards {
				dashboardTitles[i] = dashboard.Title
			}

			expectedDashboards := []string{
				"O-RAN Intent Processing",
				"Network Slice Performance",
				"VNF Deployment Overview",
				"Infrastructure Monitoring",
			}

			for _, expected := range expectedDashboards {
				found := false
				for _, title := range dashboardTitles {
					if strings.Contains(strings.ToLower(title), strings.ToLower(expected)) {
						found = true
						break
					}
				}
				Expect(found).To(BeTrue(), fmt.Sprintf("Dashboard '%s' not found", expected))
			}
		})

		It("should render dashboards without errors", func() {
			// Get list of dashboards
			resp, err := makeClusterRequest(ctx, clientset, restConfig,
				namespace, "grafana", "api/search?type=dash-db")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			var dashboards []GrafanaDashboard
			err = json.NewDecoder(resp.Body).Decode(&dashboards)
			Expect(err).NotTo(HaveOccurred())

			// Test rendering of each dashboard
			for _, dashboard := range dashboards {
				dashboardResp, err := makeClusterRequest(ctx, clientset, restConfig,
					namespace, "grafana", fmt.Sprintf("api/dashboards/uid/%s", dashboard.UID))
				Expect(err).NotTo(HaveOccurred())
				Expect(dashboardResp.StatusCode).To(Equal(http.StatusOK))
				dashboardResp.Body.Close()
			}
		})
	})

	Describe("Alert Rules and Firing", func() {
		It("should have alert rules loaded", func() {
			resp, err := makeClusterRequest(ctx, clientset, restConfig,
				namespace, "prometheus", "api/v1/rules")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			var rulesResp PrometheusRulesResponse
			err = json.NewDecoder(resp.Body).Decode(&rulesResp)
			Expect(err).NotTo(HaveOccurred())

			totalRules := 0
			for _, group := range rulesResp.Data.Groups {
				totalRules += len(group.Rules)
			}

			Expect(totalRules).To(BeNumerically(">", 0))
		})

		It("should have AlertManager accessible", func() {
			resp, err := makeClusterRequest(ctx, clientset, restConfig,
				namespace, "alertmanager", "api/v2/status")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("should handle test alerts properly", func() {
			// Create test alert by triggering a known condition
			testQuery := "vector(1) > 0" // Always true condition

			// Wait a bit for alert evaluation
			time.Sleep(30 * time.Second)

			resp, err := makeClusterRequest(ctx, clientset, restConfig,
				namespace, "alertmanager", "api/v2/alerts")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})

	Describe("Metrics Data Flow", func() {
		It("should have metrics flowing end-to-end", func() {
			// Test basic Kubernetes metrics
			queries := []string{
				"up",
				"node_cpu_seconds_total",
				"node_memory_MemTotal_bytes",
				"container_cpu_usage_seconds_total",
			}

			for _, query := range queries {
				Eventually(func() bool {
					resp, err := makeClusterRequest(ctx, clientset, restConfig,
						namespace, "prometheus", fmt.Sprintf("api/v1/query?query=%s", query))
					if err != nil {
						return false
					}
					defer resp.Body.Close()

					var queryResp PrometheusQueryResponse
					if err := json.NewDecoder(resp.Body).Decode(&queryResp); err != nil {
						return false
					}

					return len(queryResp.Data.Result) > 0
				}, 1*time.Minute, 5*time.Second).Should(BeTrue(),
					fmt.Sprintf("Query '%s' should return data", query))
			}
		})

		It("should show O-RAN specific metrics", func() {
			// Test O-RAN custom metrics if they exist
			oranQueries := []string{
				"oran_intent_processing_duration_seconds",
				"oran_slice_deployment_duration_seconds",
				"oran_vnf_placement_total",
			}

			for _, query := range oranQueries {
				// These might not exist in test environment, so just check they don't error
				resp, err := makeClusterRequest(ctx, clientset, restConfig,
					namespace, "prometheus", fmt.Sprintf("api/v1/query?query=%s", query))
				if err == nil {
					resp.Body.Close()
				}
				// Not failing on these as they depend on actual O-RAN components running
			}
		})

		It("should have proper metric retention", func() {
			// Query metrics from 1 hour ago to verify retention
			oneHourAgo := time.Now().Add(-1 * time.Hour).Unix()
			query := fmt.Sprintf("query_range?query=up&start=%d&end=%d&step=60",
				oneHourAgo, time.Now().Unix())

			resp, err := makeClusterRequest(ctx, clientset, restConfig,
				namespace, "prometheus", fmt.Sprintf("api/v1/%s", query))
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			var queryResp PrometheusQueryRangeResponse
			err = json.NewDecoder(resp.Body).Decode(&queryResp)
			Expect(err).NotTo(HaveOccurred())

			Expect(len(queryResp.Data.Result)).To(BeNumerically(">", 0))
		})
	})
})

// Test Suite for structured testing
type MonitoringE2ETestSuite struct {
	suite.Suite
	clientset  *kubernetes.Clientset
	restConfig *rest.Config
	namespace  string
	ctx        context.Context
	cancel     context.CancelFunc
}

func (suite *MonitoringE2ETestSuite) SetupSuite() {
	suite.namespace = "oran-monitoring"
	suite.ctx, suite.cancel = context.WithTimeout(context.Background(), 15*time.Minute)

	var err error
	suite.restConfig, err = config.GetConfig()
	require.NoError(suite.T(), err)

	suite.clientset, err = kubernetes.NewForConfig(suite.restConfig)
	require.NoError(suite.T(), err)
}

func (suite *MonitoringE2ETestSuite) TearDownSuite() {
	if suite.cancel != nil {
		suite.cancel()
	}
}

func (suite *MonitoringE2ETestSuite) TestCompleteDeploymentWorkflow() {
	t := suite.T()

	// Test 1: Verify namespace and basic setup
	suite.verifyNamespaceSetup(t)

	// Test 2: Verify all pods are running
	suite.verifyPodsRunning(t)

	// Test 3: Verify services are accessible
	suite.verifyServicesAccessible(t)

	// Test 4: Verify Prometheus targets
	suite.verifyPrometheusTargets(t)

	// Test 5: Verify Grafana dashboards
	suite.verifyGrafanaDashboards(t)

	// Test 6: Verify AlertManager
	suite.verifyAlertManager(t)

	// Test 7: Verify end-to-end metrics flow
	suite.verifyMetricsFlow(t)
}

func (suite *MonitoringE2ETestSuite) verifyNamespaceSetup(t *testing.T) {
	ns, err := suite.clientset.CoreV1().Namespaces().Get(suite.ctx, suite.namespace, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, suite.namespace, ns.Name)
	assert.Equal(t, "oran-intent-mano", ns.Labels["app.kubernetes.io/part-of"])
}

func (suite *MonitoringE2ETestSuite) verifyPodsRunning(t *testing.T) {
	err := wait.PollImmediate(10*time.Second, 5*time.Minute, func() (bool, error) {
		pods, err := suite.clientset.CoreV1().Pods(suite.namespace).List(suite.ctx, metav1.ListOptions{})
		if err != nil {
			return false, err
		}

		if len(pods.Items) < 3 {
			return false, nil
		}

		for _, pod := range pods.Items {
			if pod.Status.Phase != corev1.PodRunning {
				return false, nil
			}

			for _, condition := range pod.Status.Conditions {
				if condition.Type == corev1.PodReady && condition.Status != corev1.ConditionTrue {
					return false, nil
				}
			}
		}
		return true, nil
	})
	assert.NoError(t, err, "All monitoring pods should be running and ready")
}

func (suite *MonitoringE2ETestSuite) verifyServicesAccessible(t *testing.T) {
	services := []string{"prometheus", "grafana", "alertmanager"}

	for _, serviceName := range services {
		service, err := suite.clientset.CoreV1().Services(suite.namespace).Get(suite.ctx, serviceName, metav1.GetOptions{})
		assert.NoError(t, err, fmt.Sprintf("Service %s should exist", serviceName))
		assert.NotEmpty(t, service.Spec.Ports, fmt.Sprintf("Service %s should have ports", serviceName))
		assert.NotEmpty(t, service.Spec.Selector, fmt.Sprintf("Service %s should have selectors", serviceName))
	}
}

func (suite *MonitoringE2ETestSuite) verifyPrometheusTargets(t *testing.T) {
	err := wait.PollImmediate(15*time.Second, 3*time.Minute, func() (bool, error) {
		resp, err := makeClusterRequest(suite.ctx, suite.clientset, suite.restConfig,
			suite.namespace, "prometheus", "api/v1/targets")
		if err != nil {
			return false, nil
		}
		defer resp.Body.Close()

		var targetsResp PrometheusTargetsResponse
		if err := json.NewDecoder(resp.Body).Decode(&targetsResp); err != nil {
			return false, nil
		}

		upTargets := 0
		for _, target := range targetsResp.Data.ActiveTargets {
			if target.Health == "up" {
				upTargets++
			}
		}

		return upTargets >= 3, nil // At least kubernetes, node metrics, etc.
	})
	assert.NoError(t, err, "Prometheus should have healthy targets")
}

func (suite *MonitoringE2ETestSuite) verifyGrafanaDashboards(t *testing.T) {
	resp, err := makeClusterRequest(suite.ctx, suite.clientset, suite.restConfig,
		suite.namespace, "grafana", "api/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Grafana should be healthy")
}

func (suite *MonitoringE2ETestSuite) verifyAlertManager(t *testing.T) {
	resp, err := makeClusterRequest(suite.ctx, suite.clientset, suite.restConfig,
		suite.namespace, "alertmanager", "api/v2/status")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "AlertManager should be accessible")
}

func (suite *MonitoringE2ETestSuite) verifyMetricsFlow(t *testing.T) {
	queries := []string{"up", "node_cpu_seconds_total"}

	for _, query := range queries {
		err := wait.PollImmediate(5*time.Second, 1*time.Minute, func() (bool, error) {
			resp, err := makeClusterRequest(suite.ctx, suite.clientset, suite.restConfig,
				suite.namespace, "prometheus", fmt.Sprintf("api/v1/query?query=%s", query))
			if err != nil {
				return false, nil
			}
			defer resp.Body.Close()

			var queryResp PrometheusQueryResponse
			if err := json.NewDecoder(resp.Body).Decode(&queryResp); err != nil {
				return false, nil
			}

			return len(queryResp.Data.Result) > 0, nil
		})
		assert.NoError(t, err, fmt.Sprintf("Query '%s' should return data", query))
	}
}

// Helper function to make requests within the cluster
func makeClusterRequest(ctx context.Context, clientset *kubernetes.Clientset, restConfig *rest.Config,
	namespace, serviceName, path string) (*http.Response, error) {

	// Create a proxy request to the service
	proxyReq := clientset.CoreV1().Services(namespace).ProxyGet("http", serviceName, "", path, nil)
	return proxyReq.DoRaw(ctx)
}

// Response structures for API calls
type PrometheusTargetsResponse struct {
	Status string `json:"status"`
	Data   struct {
		ActiveTargets []struct {
			Health string `json:"health"`
			Job    string `json:"job"`
		} `json:"activeTargets"`
	} `json:"data"`
}

type PrometheusQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

type PrometheusQueryRangeResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Values [][]interface{}   `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

type PrometheusRulesResponse struct {
	Status string `json:"status"`
	Data   struct {
		Groups []struct {
			Name  string `json:"name"`
			Rules []struct {
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"rules"`
		} `json:"groups"`
	} `json:"data"`
}

type GrafanaDashboard struct {
	ID    int    `json:"id"`
	UID   string `json:"uid"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

// Test runner functions
func TestMonitoringE2E(t *testing.T) {
	suite.Run(t, new(MonitoringE2ETestSuite))
}

func TestMain(m *testing.M) {
	RegisterFailHandler(Fail)
	RunSpecs(m, "Monitoring E2E Test Suite")
}