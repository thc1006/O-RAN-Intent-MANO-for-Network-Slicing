// Package utils provides metrics collection and validation utilities
package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

// MetricsCollector collects and validates system metrics
type MetricsCollector struct {
	promClient     v1.API
	k8sClient      kubernetes.Interface
	metricsClient  versioned.Interface
	namespace      string
}

// NetworkMetrics captures network performance metrics
type NetworkMetrics struct {
	Throughput    float64   `json:"throughput_mbps"`
	Latency       float64   `json:"latency_ms"`
	PacketLoss    float64   `json:"packet_loss_percent"`
	Jitter        float64   `json:"jitter_ms"`
	Timestamp     time.Time `json:"timestamp"`
}

// DeploymentMetrics captures deployment performance metrics
type DeploymentMetrics struct {
	DeploymentTime   time.Duration `json:"deployment_time"`
	ReadyTime        time.Duration `json:"ready_time"`
	ResourcesCreated int           `json:"resources_created"`
	Errors           []string      `json:"errors"`
}

// ResourceMetrics captures resource utilization
type ResourceMetrics struct {
	CPUUsage      float64 `json:"cpu_usage_percent"`
	MemoryUsage   float64 `json:"memory_usage_mb"`
	NetworkIO     float64 `json:"network_io_mbps"`
	StorageIO     float64 `json:"storage_io_mbps"`
	PodCount      int     `json:"pod_count"`
}

// ThesisValidationMetrics contains all metrics required for thesis validation
type ThesisValidationMetrics struct {
	DeploymentTime time.Duration    `json:"deployment_time"`
	Network        NetworkMetrics   `json:"network"`
	Resources      ResourceMetrics  `json:"resources"`
	QoSSlice       []QoSMetrics     `json:"qos_slices"`
	Timestamp      time.Time        `json:"timestamp"`
}

// QoSMetrics captures QoS-specific metrics for different slice types
type QoSMetrics struct {
	SliceType     string  `json:"slice_type"`
	Throughput    float64 `json:"throughput_mbps"`
	Latency       float64 `json:"latency_ms"`
	Reliability   float64 `json:"reliability_percent"`
	PacketLoss    float64 `json:"packet_loss_percent"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(promEndpoint string, k8sClient kubernetes.Interface, namespace string) (*MetricsCollector, error) {
	promClient, err := api.NewClient(api.Config{
		Address: promEndpoint,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	metricsClient, err := versioned.NewForConfig(k8sClient.CoreV1().RESTClient().Get().URL().Host)
	if err != nil {
		// Metrics client is optional
		metricsClient = nil
	}

	return &MetricsCollector{
		promClient:    v1.NewAPI(promClient),
		k8sClient:     k8sClient,
		metricsClient: metricsClient,
		namespace:     namespace,
	}, nil
}

// CollectNetworkMetrics collects network performance metrics using iperf3
func (mc *MetricsCollector) CollectNetworkMetrics(ctx context.Context, targetIP string, duration time.Duration) (*NetworkMetrics, error) {
	// Run iperf3 client test
	cmd := exec.CommandContext(ctx, "iperf3", "-c", targetIP, "-t", fmt.Sprintf("%d", int(duration.Seconds())), "-J")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run iperf3: %w", err)
	}

	// Parse iperf3 JSON output
	var result struct {
		End struct {
			SumSent struct {
				BitsPerSecond float64 `json:"bits_per_second"`
			} `json:"sum_sent"`
			SumReceived struct {
				BitsPerSecond float64 `json:"bits_per_second"`
			} `json:"sum_received"`
		} `json:"end"`
		Intervals []struct {
			Sum struct {
				RTTMean float64 `json:"rtt_mean"`
				Jitter  float64 `json:"jitter_ms"`
			} `json:"sum"`
		} `json:"intervals"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse iperf3 output: %w", err)
	}

	throughputBps := result.End.SumReceived.BitsPerSecond
	throughputMbps := throughputBps / 1_000_000

	// Measure latency with ping
	latency, err := mc.measureLatency(targetIP)
	if err != nil {
		latency = 0 // Use 0 if ping fails
	}

	// Measure packet loss
	packetLoss, err := mc.measurePacketLoss(targetIP)
	if err != nil {
		packetLoss = 0
	}

	return &NetworkMetrics{
		Throughput: throughputMbps,
		Latency:    latency,
		PacketLoss: packetLoss,
		Jitter:     0, // TODO: Extract from iperf3 output
		Timestamp:  time.Now(),
	}, nil
}

// measureLatency measures RTT latency using ping
func (mc *MetricsCollector) measureLatency(targetIP string) (float64, error) {
	cmd := exec.Command("ping", "-c", "10", targetIP)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to ping %s: %w", targetIP, err)
	}

	// Parse ping output for average RTT
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "rtt min/avg/max/mdev") {
			parts := strings.Split(line, " = ")
			if len(parts) > 1 {
				rttParts := strings.Split(parts[1], "/")
				if len(rttParts) >= 2 {
					avg, err := strconv.ParseFloat(rttParts[1], 64)
					if err == nil {
						return avg, nil
					}
				}
			}
		}
	}

	return 0, fmt.Errorf("could not parse ping output")
}

// measurePacketLoss measures packet loss percentage
func (mc *MetricsCollector) measurePacketLoss(targetIP string) (float64, error) {
	cmd := exec.Command("ping", "-c", "100", targetIP)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to ping %s: %w", targetIP, err)
	}

	// Parse ping output for packet loss
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "packet loss") {
			parts := strings.Split(line, ",")
			for _, part := range parts {
				if strings.Contains(part, "packet loss") {
					lossStr := strings.TrimSpace(strings.Split(part, "%")[0])
					// Extract the number before %
					words := strings.Fields(lossStr)
					if len(words) > 0 {
						loss, err := strconv.ParseFloat(words[len(words)-1], 64)
						if err == nil {
							return loss, nil
						}
					}
				}
			}
		}
	}

	return 0, nil
}

// CollectDeploymentMetrics collects deployment timing metrics
func (mc *MetricsCollector) CollectDeploymentMetrics(ctx context.Context, deploymentName string) (*DeploymentMetrics, error) {
	start := time.Now()

	// Wait for deployment to be ready
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			deployment, err := mc.k8sClient.AppsV1().Deployments(mc.namespace).Get(ctx, deploymentName, metav1.GetOptions{})
			if err != nil {
				continue
			}

			if deployment.Status.ReadyReplicas == deployment.Status.Replicas && deployment.Status.Replicas > 0 {
				return &DeploymentMetrics{
					DeploymentTime:   time.Since(start),
					ReadyTime:        time.Since(start),
					ResourcesCreated: int(deployment.Status.Replicas),
				}, nil
			}

			time.Sleep(5 * time.Second)
		}
	}
}

// CollectResourceMetrics collects resource utilization metrics
func (mc *MetricsCollector) CollectResourceMetrics(ctx context.Context) (*ResourceMetrics, error) {
	pods, err := mc.k8sClient.CoreV1().Pods(mc.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	metrics := &ResourceMetrics{
		PodCount: len(pods.Items),
	}

	// Collect CPU and memory metrics if metrics server is available
	if mc.metricsClient != nil {
		podMetrics, err := mc.metricsClient.MetricsV1beta1().PodMetricses(mc.namespace).List(ctx, metav1.ListOptions{})
		if err == nil {
			var totalCPU, totalMemory int64
			for _, pm := range podMetrics.Items {
				for _, container := range pm.Containers {
					totalCPU += container.Usage.Cpu().MilliValue()
					totalMemory += container.Usage.Memory().Value()
				}
			}

			metrics.CPUUsage = float64(totalCPU) / 1000 // Convert millicores to cores
			metrics.MemoryUsage = float64(totalMemory) / (1024 * 1024) // Convert bytes to MB
		}
	}

	return metrics, nil
}

// ValidateThesisRequirements validates metrics against thesis requirements
func (mc *MetricsCollector) ValidateThesisRequirements(metrics *ThesisValidationMetrics) []string {
	var violations []string

	// Validate E2E deployment time < 10 minutes
	if metrics.DeploymentTime > 10*time.Minute {
		violations = append(violations, fmt.Sprintf("Deployment time %v exceeds 10 minutes", metrics.DeploymentTime))
	}

	// Define expected QoS targets based on thesis
	expectedQoS := map[string]QoSMetrics{
		"embb": {Throughput: 4.57, Latency: 16.1},
		"urllc": {Throughput: 2.77, Latency: 15.7},
		"mmtc": {Throughput: 0.93, Latency: 6.3},
	}

	// Validate QoS metrics for each slice type
	for _, qos := range metrics.QoSSlice {
		expected, exists := expectedQoS[strings.ToLower(qos.SliceType)]
		if !exists {
			continue
		}

		// Validate throughput (allow 10% tolerance)
		if qos.Throughput < expected.Throughput*0.9 {
			violations = append(violations,
				fmt.Sprintf("Slice %s throughput %.2f Mbps below expected %.2f Mbps",
					qos.SliceType, qos.Throughput, expected.Throughput))
		}

		// Validate latency (allow 10% tolerance)
		if qos.Latency > expected.Latency*1.1 {
			violations = append(violations,
				fmt.Sprintf("Slice %s latency %.2f ms above expected %.2f ms",
					qos.SliceType, qos.Latency, expected.Latency))
		}

		// Validate packet loss < 1%
		if qos.PacketLoss > 1.0 {
			violations = append(violations,
				fmt.Sprintf("Slice %s packet loss %.2f%% exceeds 1%%",
					qos.SliceType, qos.PacketLoss))
		}
	}

	return violations
}

// GenerateMetricsReport generates a comprehensive metrics report
func (mc *MetricsCollector) GenerateMetricsReport(metrics *ThesisValidationMetrics) map[string]interface{} {
	violations := mc.ValidateThesisRequirements(metrics)

	return map[string]interface{}{
		"timestamp":       time.Now(),
		"deployment_time": metrics.DeploymentTime.String(),
		"network_metrics": metrics.Network,
		"resource_metrics": metrics.Resources,
		"qos_metrics":     metrics.QoSSlice,
		"violations":      violations,
		"compliance":      len(violations) == 0,
		"summary": map[string]interface{}{
			"total_violations": len(violations),
			"deployment_compliant": metrics.DeploymentTime <= 10*time.Minute,
			"qos_compliant":       len(violations) == 0,
		},
	}
}

// MeasureEndToEndLatency measures latency between two network endpoints
func MeasureEndToEndLatency(source, destination string, count int) (float64, error) {
	conn, err := net.Dial("tcp", destination)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to %s: %w", destination, err)
	}
	defer conn.Close()

	var totalLatency time.Duration
	validMeasurements := 0

	for i := 0; i < count; i++ {
		start := time.Now()
		_, err := conn.Write([]byte("ping"))
		if err != nil {
			continue
		}

		buffer := make([]byte, 4)
		_, err = conn.Read(buffer)
		if err != nil {
			continue
		}

		latency := time.Since(start)
		totalLatency += latency
		validMeasurements++

		time.Sleep(100 * time.Millisecond)
	}

	if validMeasurements == 0 {
		return 0, fmt.Errorf("no valid measurements obtained")
	}

	avgLatency := totalLatency / time.Duration(validMeasurements)
	return float64(avgLatency.Nanoseconds()) / 1_000_000, nil // Convert to milliseconds
}