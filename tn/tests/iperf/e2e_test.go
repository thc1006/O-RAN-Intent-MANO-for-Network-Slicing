package iperf

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	tnv1alpha1 "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn/manager/api/v1alpha1"
)

// TestProfile represents a test configuration for a specific QoS profile
type TestProfile struct {
	Name              string
	SliceID          string
	ExpectedBandwidth float64 // Mbps
	ExpectedLatency  float64 // ms
	Tolerance        float64 // percentage (e.g., 0.1 for 10%)
}

var testProfiles = []TestProfile{
	{
		Name:              "eMBB-HighBandwidth",
		SliceID:          "embb-slice",
		ExpectedBandwidth: 4.57,
		ExpectedLatency:  16.1,
		Tolerance:        0.10,
	},
	{
		Name:              "mIoT-Balanced",
		SliceID:          "miot-slice",
		ExpectedBandwidth: 2.77,
		ExpectedLatency:  15.7,
		Tolerance:        0.10,
	},
	{
		Name:              "uRLLC-LowLatency",
		SliceID:          "urllc-slice",
		ExpectedBandwidth: 0.93,
		ExpectedLatency:  6.3,
		Tolerance:        0.10,
	},
}

// IPerf3Result represents the JSON output from iperf3
type IPerf3Result struct {
	End struct {
		SumSent struct {
			BitsPerSecond float64 `json:"bits_per_second"`
		} `json:"sum_sent"`
		SumReceived struct {
			BitsPerSecond float64 `json:"bits_per_second"`
		} `json:"sum_received"`
	} `json:"end"`
}

func TestTNSlicePerformance(t *testing.T) {
	// Setup Kubernetes client
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		t.Fatalf("Failed to build config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("Failed to create clientset: %v", err)
	}

	// Run tests for each profile
	for _, profile := range testProfiles {
		t.Run(profile.Name, func(t *testing.T) {
			// Deploy test slice
			slice := createTestSlice(profile)
			if err := deploySlice(slice); err != nil {
				t.Fatalf("Failed to deploy slice: %v", err)
			}
			defer cleanupSlice(slice)

			// Wait for slice to become active
			if err := waitForSliceActive(slice, 60*time.Second); err != nil {
				t.Fatalf("Slice did not become active: %v", err)
			}

			// Deploy iPerf3 server and client
			serverPod, clientPod := deployIPerf3Pods(t, clientset, profile.SliceID)
			defer cleanupPods(clientset, serverPod, clientPod)

			// Wait for pods to be ready
			if err := waitForPodsReady(clientset, []*corev1.Pod{serverPod, clientPod}, 60*time.Second); err != nil {
				t.Fatalf("Pods did not become ready: %v", err)
			}

			// Run throughput test
			throughput := runThroughputTest(t, clientset, clientPod, serverPod)

			// Run latency test
			latency := runLatencyTest(t, clientset, clientPod, serverPod)

			// Validate results
			validateThroughput(t, profile, throughput)
			validateLatency(t, profile, latency)
		})
	}
}

func createTestSlice(profile TestProfile) *tnv1alpha1.TNSlice {
	return &tnv1alpha1.TNSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      profile.SliceID,
			Namespace: "default",
		},
		Spec: tnv1alpha1.TNSliceSpec{
			SliceID:   profile.SliceID,
			Bandwidth: float32(profile.ExpectedBandwidth),
			Latency:   float32(profile.ExpectedLatency),
			VxlanID:   hashSliceID(profile.SliceID),
			Priority:  5,
			Endpoints: []tnv1alpha1.Endpoint{
				{
					NodeName:  "kind-worker",
					IP:        "172.18.0.3",
					Interface: "eth0",
					Role:      "source",
				},
				{
					NodeName:  "kind-worker2",
					IP:        "172.18.0.4",
					Interface: "eth0",
					Role:      "destination",
				},
			},
		},
	}
}

func deployIPerf3Pods(t *testing.T, clientset kubernetes.Interface, sliceID string) (*corev1.Pod, *corev1.Pod) {
	namespace := "default"

	// Create iPerf3 server pod
	serverPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("iperf3-server-%s", sliceID),
			Namespace: namespace,
			Labels: map[string]string{
				"app":   "iperf3-server",
				"slice": sliceID,
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "kind-worker2",
			Containers: []corev1.Container{
				{
					Name:    "iperf3",
					Image:   "networkstatic/iperf3:latest",
					Command: []string{"iperf3", "-s"},
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 5201,
							Protocol:      corev1.ProtocolTCP,
						},
					},
				},
			},
		},
	}

	// Create iPerf3 client pod
	clientPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("iperf3-client-%s", sliceID),
			Namespace: namespace,
			Labels: map[string]string{
				"app":   "iperf3-client",
				"slice": sliceID,
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "kind-worker",
			Containers: []corev1.Container{
				{
					Name:    "iperf3",
					Image:   "networkstatic/iperf3:latest",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}

	// Deploy pods
	ctx := context.Background()
	createdServer, err := clientset.CoreV1().Pods(namespace).Create(ctx, serverPod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create server pod: %v", err)
	}

	createdClient, err := clientset.CoreV1().Pods(namespace).Create(ctx, clientPod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create client pod: %v", err)
	}

	return createdServer, createdClient
}

func runThroughputTest(t *testing.T, clientset kubernetes.Interface, clientPod, serverPod *corev1.Pod) float64 {
	// Get server pod IP
	ctx := context.Background()
	server, err := clientset.CoreV1().Pods(serverPod.Namespace).Get(ctx, serverPod.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get server pod: %v", err)
	}

	serverIP := server.Status.PodIP

	// Run iperf3 test for 30 seconds
	cmd := fmt.Sprintf("iperf3 -c %s -t 30 -J", serverIP)

	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(clientPod.Name).
		Namespace(clientPod.Namespace).
		SubResource("exec").
		Param("container", "iperf3").
		Param("command", "sh").
		Param("command", "-c").
		Param("command", cmd).
		Param("stdout", "true").
		Param("stderr", "true")

	// Execute command
	output, err := executeInPod(req)
	if err != nil {
		t.Fatalf("Failed to run iperf3: %v", err)
	}

	// Parse JSON output
	var result IPerf3Result
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse iperf3 output: %v", err)
	}

	// Convert to Mbps
	throughputMbps := result.End.SumSent.BitsPerSecond / 1000000

	t.Logf("Measured throughput: %.2f Mbps", throughputMbps)
	return throughputMbps
}

func runLatencyTest(t *testing.T, clientset kubernetes.Interface, clientPod, serverPod *corev1.Pod) float64 {
	// Get server pod IP
	ctx := context.Background()
	server, err := clientset.CoreV1().Pods(serverPod.Namespace).Get(ctx, serverPod.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get server pod: %v", err)
	}

	serverIP := server.Status.PodIP

	// Run ping test (1000 packets)
	cmd := fmt.Sprintf("ping -c 1000 -i 0.01 %s | tail -1 | awk '{print $4}' | cut -d '/' -f 2", serverIP)

	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(clientPod.Name).
		Namespace(clientPod.Namespace).
		SubResource("exec").
		Param("container", "iperf3").
		Param("command", "sh").
		Param("command", "-c").
		Param("command", cmd).
		Param("stdout", "true")

	// Execute command
	output, err := executeInPod(req)
	if err != nil {
		t.Fatalf("Failed to run ping: %v", err)
	}

	// Parse average RTT
	avgRTT, err := strconv.ParseFloat(strings.TrimSpace(output), 64)
	if err != nil {
		t.Fatalf("Failed to parse ping output: %v", err)
	}

	t.Logf("Measured average RTT: %.2f ms", avgRTT)
	return avgRTT
}

func validateThroughput(t *testing.T, profile TestProfile, measuredMbps float64) {
	lowerBound := profile.ExpectedBandwidth * (1 - profile.Tolerance)
	upperBound := profile.ExpectedBandwidth * (1 + profile.Tolerance)

	if measuredMbps < lowerBound || measuredMbps > upperBound {
		t.Errorf("Throughput validation failed for %s: expected %.2f±%.0f%% Mbps, got %.2f Mbps",
			profile.Name, profile.ExpectedBandwidth, profile.Tolerance*100, measuredMbps)
	} else {
		t.Logf("✓ Throughput validation passed for %s: %.2f Mbps (expected %.2f±%.0f%%)",
			profile.Name, measuredMbps, profile.ExpectedBandwidth, profile.Tolerance*100)
	}

	// Calculate percentage difference
	percentDiff := math.Abs(measuredMbps-profile.ExpectedBandwidth) / profile.ExpectedBandwidth * 100
	t.Logf("  Throughput deviation: %.1f%%", percentDiff)
}

func validateLatency(t *testing.T, profile TestProfile, measuredMs float64) {
	// Allow 2ms tolerance for latency
	latencyTolerance := 2.0

	if math.Abs(measuredMs-profile.ExpectedLatency) > latencyTolerance {
		t.Errorf("Latency validation failed for %s: expected %.1f±%.1f ms, got %.2f ms",
			profile.Name, profile.ExpectedLatency, latencyTolerance, measuredMs)
	} else {
		t.Logf("✓ Latency validation passed for %s: %.2f ms (expected %.1f±%.1f ms)",
			profile.Name, measuredMs, profile.ExpectedLatency, latencyTolerance)
	}
}

func waitForSliceActive(slice *tnv1alpha1.TNSlice, timeout time.Duration) error {
	// TODO: Implement actual wait logic checking slice status
	time.Sleep(10 * time.Second)
	return nil
}

func waitForPodsReady(clientset kubernetes.Interface, pods []*corev1.Pod, timeout time.Duration) error {
	ctx := context.Background()
	deadline := time.Now().Add(timeout)

	for _, pod := range pods {
		for time.Now().Before(deadline) {
			p, err := clientset.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if p.Status.Phase == corev1.PodRunning {
				ready := true
				for _, cond := range p.Status.Conditions {
					if cond.Type == corev1.PodReady && cond.Status != corev1.ConditionTrue {
						ready = false
						break
					}
				}
				if ready {
					break
				}
			}

			time.Sleep(2 * time.Second)
		}
	}

	return nil
}

func executeInPod(req *rest.Request) (string, error) {
	// TODO: Implement actual pod exec
	// This would use remotecommand.NewSPDYExecutor
	return "", fmt.Errorf("not implemented")
}

func deploySlice(slice *tnv1alpha1.TNSlice) error {
	// TODO: Deploy slice using dynamic client
	return nil
}

func cleanupSlice(slice *tnv1alpha1.TNSlice) {
	// TODO: Cleanup slice
}

func cleanupPods(clientset kubernetes.Interface, pods ...*corev1.Pod) {
	ctx := context.Background()
	for _, pod := range pods {
		_ = clientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
	}
}

func hashSliceID(sliceID string) int32 {
	hash := int32(0)
	for _, c := range sliceID {
		hash = hash*31 + int32(c)
	}
	return hash % 16777215 // Max VXLAN ID
}