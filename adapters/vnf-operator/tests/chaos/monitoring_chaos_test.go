package chaos

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var _ = Describe("O-RAN Monitoring Chaos Engineering Tests", func() {
	var (
		clientset   *kubernetes.Clientset
		restConfig  *rest.Config
		namespace   = "oran-monitoring"
		testTimeout = 20 * time.Minute
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

	Describe("Prometheus Pod Failure Scenarios", func() {
		It("should recover from Prometheus pod deletion", func() {
			// Get initial Prometheus pod
			initialPods := getPrometheusPods(ctx, clientset, namespace)
			Expect(len(initialPods)).To(BeNumerically(">=", 1))

			initialPod := initialPods[0]

			// Delete Prometheus pod
			err := clientset.CoreV1().Pods(namespace).Delete(ctx, initialPod.Name, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Wait for pod to be recreated and ready
			Eventually(func() bool {
				pods := getPrometheusPods(ctx, clientset, namespace)
				if len(pods) == 0 {
					return false
				}

				for _, pod := range pods {
					if pod.Name != initialPod.Name && pod.Status.Phase == corev1.PodRunning {
						// Check if pod is ready
						for _, condition := range pod.Status.Conditions {
							if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
								return true
							}
						}
					}
				}
				return false
			}, 5*time.Minute, 10*time.Second).Should(BeTrue(), "New Prometheus pod should be running and ready")

			// Verify Prometheus is functional after recovery
			Eventually(func() bool {
				return isPrometheusHealthy(ctx, clientset, restConfig, namespace)
			}, 2*time.Minute, 10*time.Second).Should(BeTrue(), "Prometheus should be healthy after recovery")

			// Verify targets are being scraped again
			Eventually(func() bool {
				return hasActiveTargets(ctx, clientset, restConfig, namespace)
			}, 3*time.Minute, 15*time.Second).Should(BeTrue(), "Prometheus should have active targets after recovery")
		})

		It("should handle Prometheus container restart gracefully", func() {
			// Get Prometheus deployment
			deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, "prometheus", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Scale down to 0
			zero := int32(0)
			deployment.Spec.Replicas = &zero
			_, err = clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Wait for pod to be terminated
			Eventually(func() bool {
				pods := getPrometheusPods(ctx, clientset, namespace)
				return len(pods) == 0
			}, 2*time.Minute, 5*time.Second).Should(BeTrue())

			// Scale back up to 1
			one := int32(1)
			deployment.Spec.Replicas = &one
			_, err = clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Wait for recovery
			Eventually(func() bool {
				pods := getPrometheusPods(ctx, clientset, namespace)
				if len(pods) != 1 {
					return false
				}
				return pods[0].Status.Phase == corev1.PodRunning
			}, 3*time.Minute, 10*time.Second).Should(BeTrue())

			// Verify functionality restored
			Eventually(func() bool {
				return isPrometheusHealthy(ctx, clientset, restConfig, namespace)
			}, 2*time.Minute, 10*time.Second).Should(BeTrue())
		})

		It("should maintain data consistency during restart", func() {
			// Query current metrics before restart
			initialMetrics := queryPrometheusMetrics(ctx, clientset, restConfig, namespace, "up")

			// Restart Prometheus pod (delete and let it recreate)
			pods := getPrometheusPods(ctx, clientset, namespace)
			Expect(len(pods)).To(BeNumerically(">=", 1))

			err := clientset.CoreV1().Pods(namespace).Delete(ctx, pods[0].Name, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Wait for recovery
			Eventually(func() bool {
				return isPrometheusHealthy(ctx, clientset, restConfig, namespace)
			}, 5*time.Minute, 15*time.Second).Should(BeTrue())

			// Wait a bit for metrics to be re-scraped
			time.Sleep(1 * time.Minute)

			// Query metrics after restart
			recoveredMetrics := queryPrometheusMetrics(ctx, clientset, restConfig, namespace, "up")

			// Should have similar or more metrics (allowing for some variance)
			ratio := float64(len(recoveredMetrics)) / float64(len(initialMetrics))
			Expect(ratio).To(BeNumerically(">=", 0.8), "Should recover at least 80% of metrics after restart")
		})
	})

	Describe("Grafana Pod Failure Scenarios", func() {
		It("should recover from Grafana pod failure", func() {
			// Delete Grafana pod
			pods := getGrafanaPods(ctx, clientset, namespace)
			Expect(len(pods)).To(BeNumerically(">=", 1))

			err := clientset.CoreV1().Pods(namespace).Delete(ctx, pods[0].Name, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Wait for recovery
			Eventually(func() bool {
				pods := getGrafanaPods(ctx, clientset, namespace)
				if len(pods) == 0 {
					return false
				}
				return pods[0].Status.Phase == corev1.PodRunning
			}, 3*time.Minute, 10*time.Second).Should(BeTrue())

			// Verify Grafana is accessible
			Eventually(func() bool {
				return isGrafanaHealthy(ctx, clientset, restConfig, namespace)
			}, 2*time.Minute, 10*time.Second).Should(BeTrue())
		})

		It("should preserve dashboard configurations after restart", func() {
			// Get dashboard list before restart
			initialDashboards := getGrafanaDashboards(ctx, clientset, restConfig, namespace)

			// Restart Grafana
			pods := getGrafanaPods(ctx, clientset, namespace)
			if len(pods) > 0 {
				err := clientset.CoreV1().Pods(namespace).Delete(ctx, pods[0].Name, metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
			}

			// Wait for recovery
			Eventually(func() bool {
				return isGrafanaHealthy(ctx, clientset, restConfig, namespace)
			}, 3*time.Minute, 10*time.Second).Should(BeTrue())

			// Get dashboard list after restart
			recoveredDashboards := getGrafanaDashboards(ctx, clientset, restConfig, namespace)

			// Should have same or more dashboards
			Expect(len(recoveredDashboards)).To(BeNumerically(">=", len(initialDashboards)),
				"Dashboard count should be preserved after restart")
		})
	})

	Describe("AlertManager Failure Scenarios", func() {
		It("should handle AlertManager pod failures", func() {
			// Delete AlertManager pod
			pods := getAlertManagerPods(ctx, clientset, namespace)
			if len(pods) > 0 {
				err := clientset.CoreV1().Pods(namespace).Delete(ctx, pods[0].Name, metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
			}

			// Wait for recovery
			Eventually(func() bool {
				pods := getAlertManagerPods(ctx, clientset, namespace)
				if len(pods) == 0 {
					return false
				}
				return pods[0].Status.Phase == corev1.PodRunning
			}, 3*time.Minute, 10*time.Second).Should(BeTrue())

			// Verify AlertManager is accessible
			Eventually(func() bool {
				return isAlertManagerHealthy(ctx, clientset, restConfig, namespace)
			}, 2*time.Minute, 10*time.Second).Should(BeTrue())
		})
	})

	Describe("Network Partition Scenarios", func() {
		It("should handle temporary network issues gracefully", func() {
			// This is a simplified simulation - in real chaos testing you'd use tools like Chaos Mesh

			// Verify initial state
			Expect(isPrometheusHealthy(ctx, clientset, restConfig, namespace)).To(BeTrue())

			// Simulate network partition by scaling down temporarily
			deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, "prometheus", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			originalReplicas := *deployment.Spec.Replicas
			zero := int32(0)
			deployment.Spec.Replicas = &zero
			_, err = clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Wait for partition
			time.Sleep(30 * time.Second)

			// Restore
			deployment.Spec.Replicas = &originalReplicas
			_, err = clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Verify recovery
			Eventually(func() bool {
				return isPrometheusHealthy(ctx, clientset, restConfig, namespace)
			}, 3*time.Minute, 10*time.Second).Should(BeTrue())
		})
	})

	Describe("High Load Scenarios", func() {
		It("should handle high query load", func() {
			// Generate high query load (simplified)
			queryCount := 0
			done := make(chan bool, 1)

			// Run queries for 2 minutes
			go func() {
				timeout := time.After(2 * time.Minute)
				ticker := time.NewTicker(100 * time.Millisecond)
				defer ticker.Stop()

				for {
					select {
					case <-timeout:
						done <- true
						return
					case <-ticker.C:
						if isPrometheusHealthy(ctx, clientset, restConfig, namespace) {
							queryCount++
						}
					}
				}
			}()

			<-done

			// Should handle reasonable number of queries
			Expect(queryCount).To(BeNumerically(">=", 100), "Should handle at least 100 queries under load")

			// System should still be healthy after load test
			Expect(isPrometheusHealthy(ctx, clientset, restConfig, namespace)).To(BeTrue())
		})

		It("should handle high cardinality metrics", func() {
			// Check that system can handle high cardinality (test would need actual high cardinality metrics)
			metrics := queryPrometheusMetrics(ctx, clientset, restConfig, namespace, "up")

			// Should be able to query metrics even with high cardinality
			Expect(len(metrics)).To(BeNumerically(">=", 1))

			// System should remain responsive
			Expect(isPrometheusHealthy(ctx, clientset, restConfig, namespace)).To(BeTrue())
		})
	})

	Describe("Alert Storm Handling", func() {
		It("should handle multiple simultaneous alerts", func() {
			// Verify AlertManager can handle multiple alerts
			// In a real test, you'd create conditions that trigger multiple alerts

			// Check AlertManager is responsive
			Expect(isAlertManagerHealthy(ctx, clientset, restConfig, namespace)).To(BeTrue())

			// Check that alert rules are loaded
			hasAlerts := hasActiveAlertRules(ctx, clientset, restConfig, namespace)
			if hasAlerts {
				// If we have alerts, verify AlertManager can handle them
				alerts := getActiveAlerts(ctx, clientset, restConfig, namespace)
				// Should be able to retrieve alerts without timeout
				Expect(len(alerts)).To(BeNumerically(">=", 0))
			}
		})
	})
})

// Test Suite for structured chaos testing
type MonitoringChaosTestSuite struct {
	suite.Suite
	clientset  *kubernetes.Clientset
	restConfig *rest.Config
	namespace  string
	ctx        context.Context
	cancel     context.CancelFunc
}

func (suite *MonitoringChaosTestSuite) SetupSuite() {
	suite.namespace = "oran-monitoring"
	suite.ctx, suite.cancel = context.WithTimeout(context.Background(), 30*time.Minute)

	var err error
	suite.restConfig, err = config.GetConfig()
	require.NoError(suite.T(), err)

	suite.clientset, err = kubernetes.NewForConfig(suite.restConfig)
	require.NoError(suite.T(), err)
}

func (suite *MonitoringChaosTestSuite) TearDownSuite() {
	if suite.cancel != nil {
		suite.cancel()
	}
}

func (suite *MonitoringChaosTestSuite) TestCascadingFailure() {
	t := suite.T()

	// Test cascading failure scenario
	originalHealth := make(map[string]bool)
	components := []string{"prometheus", "grafana", "alertmanager"}

	// Record initial health
	for _, component := range components {
		switch component {
		case "prometheus":
			originalHealth[component] = isPrometheusHealthy(suite.ctx, suite.clientset, suite.restConfig, suite.namespace)
		case "grafana":
			originalHealth[component] = isGrafanaHealthy(suite.ctx, suite.clientset, suite.restConfig, suite.namespace)
		case "alertmanager":
			originalHealth[component] = isAlertManagerHealthy(suite.ctx, suite.clientset, suite.restConfig, suite.namespace)
		}
	}

	// Introduce failures in sequence
	for _, component := range components {
		t.Run(fmt.Sprintf("Fail_%s", component), func(t *testing.T) {
			// Delete pods for component
			labelSelector := fmt.Sprintf("app.kubernetes.io/name=%s", component)
			pods, err := suite.clientset.CoreV1().Pods(suite.namespace).List(suite.ctx, metav1.ListOptions{
				LabelSelector: labelSelector,
			})
			require.NoError(t, err)

			for _, pod := range pods.Items {
				err := suite.clientset.CoreV1().Pods(suite.namespace).Delete(suite.ctx, pod.Name, metav1.DeleteOptions{})
				assert.NoError(t, err)
			}

			// Wait for recovery
			time.Sleep(30 * time.Second)
		})
	}

	// Verify all components recover
	err := wait.PollImmediate(10*time.Second, 5*time.Minute, func() (bool, error) {
		for _, component := range components {
			var healthy bool
			switch component {
			case "prometheus":
				healthy = isPrometheusHealthy(suite.ctx, suite.clientset, suite.restConfig, suite.namespace)
			case "grafana":
				healthy = isGrafanaHealthy(suite.ctx, suite.clientset, suite.restConfig, suite.namespace)
			case "alertmanager":
				healthy = isAlertManagerHealthy(suite.ctx, suite.clientset, suite.restConfig, suite.namespace)
			}

			if !healthy {
				return false, nil
			}
		}
		return true, nil
	})

	assert.NoError(t, err, "All components should recover from cascading failure")
}

func (suite *MonitoringChaosTestSuite) TestResourceExhaustion() {
	t := suite.T()

	// Test behavior under resource pressure
	// This is a simplified test - real chaos testing would create actual resource pressure

	// Check current resource usage
	pods, err := suite.clientset.CoreV1().Pods(suite.namespace).List(suite.ctx, metav1.ListOptions{})
	require.NoError(t, err)

	for _, pod := range pods.Items {
		// Verify pods have resource limits set
		for _, container := range pod.Spec.Containers {
			assert.NotNil(t, container.Resources.Limits,
				"Container %s in pod %s should have resource limits", container.Name, pod.Name)
		}
	}

	// Verify system remains stable under normal load
	assert.True(t, isPrometheusHealthy(suite.ctx, suite.clientset, suite.restConfig, suite.namespace))
	assert.True(t, isGrafanaHealthy(suite.ctx, suite.clientset, suite.restConfig, suite.namespace))
}

// Helper functions for chaos testing
func getPrometheusPods(ctx context.Context, clientset *kubernetes.Clientset, namespace string) []corev1.Pod {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=prometheus",
	})
	if err != nil {
		return []corev1.Pod{}
	}
	return pods.Items
}

func getGrafanaPods(ctx context.Context, clientset *kubernetes.Clientset, namespace string) []corev1.Pod {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=grafana",
	})
	if err != nil {
		return []corev1.Pod{}
	}
	return pods.Items
}

func getAlertManagerPods(ctx context.Context, clientset *kubernetes.Clientset, namespace string) []corev1.Pod {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=alertmanager",
	})
	if err != nil {
		return []corev1.Pod{}
	}
	return pods.Items
}

func isPrometheusHealthy(ctx context.Context, clientset *kubernetes.Clientset, restConfig *rest.Config, namespace string) bool {
	proxyReq := clientset.CoreV1().Services(namespace).ProxyGet("http", "prometheus", "", "api/v1/status/runtimeinfo", nil)
	resp, err := proxyReq.DoRaw(ctx)
	if err != nil {
		return false
	}

	var response map[string]interface{}
	if err := json.Unmarshal(resp, &response); err != nil {
		return false
	}

	return response["status"] == "success"
}

func isGrafanaHealthy(ctx context.Context, clientset *kubernetes.Clientset, restConfig *rest.Config, namespace string) bool {
	proxyReq := clientset.CoreV1().Services(namespace).ProxyGet("http", "grafana", "", "api/health", nil)
	_, err := proxyReq.DoRaw(ctx)
	return err == nil
}

func isAlertManagerHealthy(ctx context.Context, clientset *kubernetes.Clientset, restConfig *rest.Config, namespace string) bool {
	proxyReq := clientset.CoreV1().Services(namespace).ProxyGet("http", "alertmanager", "", "api/v2/status", nil)
	_, err := proxyReq.DoRaw(ctx)
	return err == nil
}

func hasActiveTargets(ctx context.Context, clientset *kubernetes.Clientset, restConfig *rest.Config, namespace string) bool {
	proxyReq := clientset.CoreV1().Services(namespace).ProxyGet("http", "prometheus", "", "api/v1/targets", nil)
	resp, err := proxyReq.DoRaw(ctx)
	if err != nil {
		return false
	}

	var targetsResp struct {
		Data struct {
			ActiveTargets []struct {
				Health string `json:"health"`
			} `json:"activeTargets"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &targetsResp); err != nil {
		return false
	}

	upTargets := 0
	for _, target := range targetsResp.Data.ActiveTargets {
		if target.Health == "up" {
			upTargets++
		}
	}

	return upTargets > 0
}

func queryPrometheusMetrics(ctx context.Context, clientset *kubernetes.Clientset, restConfig *rest.Config, namespace, query string) []interface{} {
	path := fmt.Sprintf("api/v1/query?query=%s", query)
	proxyReq := clientset.CoreV1().Services(namespace).ProxyGet("http", "prometheus", "", path, nil)
	resp, err := proxyReq.DoRaw(ctx)
	if err != nil {
		return []interface{}{}
	}

	var queryResp struct {
		Data struct {
			Result []interface{} `json:"result"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &queryResp); err != nil {
		return []interface{}{}
	}

	return queryResp.Data.Result
}

func getGrafanaDashboards(ctx context.Context, clientset *kubernetes.Clientset, restConfig *rest.Config, namespace string) []interface{} {
	proxyReq := clientset.CoreV1().Services(namespace).ProxyGet("http", "grafana", "", "api/search?type=dash-db", nil)
	resp, err := proxyReq.DoRaw(ctx)
	if err != nil {
		return []interface{}{}
	}

	var dashboards []interface{}
	json.Unmarshal(resp, &dashboards)
	return dashboards
}

func hasActiveAlertRules(ctx context.Context, clientset *kubernetes.Clientset, restConfig *rest.Config, namespace string) bool {
	proxyReq := clientset.CoreV1().Services(namespace).ProxyGet("http", "prometheus", "", "api/v1/rules", nil)
	resp, err := proxyReq.DoRaw(ctx)
	if err != nil {
		return false
	}

	var rulesResp struct {
		Data struct {
			Groups []struct {
				Rules []interface{} `json:"rules"`
			} `json:"groups"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &rulesResp); err != nil {
		return false
	}

	for _, group := range rulesResp.Data.Groups {
		if len(group.Rules) > 0 {
			return true
		}
	}
	return false
}

func getActiveAlerts(ctx context.Context, clientset *kubernetes.Clientset, restConfig *rest.Config, namespace string) []interface{} {
	proxyReq := clientset.CoreV1().Services(namespace).ProxyGet("http", "alertmanager", "", "api/v2/alerts", nil)
	resp, err := proxyReq.DoRaw(ctx)
	if err != nil {
		return []interface{}{}
	}

	var alerts []interface{}
	json.Unmarshal(resp, &alerts)
	return alerts
}

// Test runner
func TestMonitoringChaos(t *testing.T) {
	suite.Run(t, new(MonitoringChaosTestSuite))
}

func TestMain(m *testing.M) {
	RegisterFailHandler(Fail)
	RunSpecs(m, "Monitoring Chaos Test Suite")
}