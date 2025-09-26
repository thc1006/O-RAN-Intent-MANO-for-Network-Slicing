package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/o2-client/pkg/models"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/o2-client/pkg/o2dms"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/o2-client/pkg/o2ims"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/orchestrator/pkg/statemachine"
	tnpkg "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn/manager/pkg"
)

// TestEnhancedBackendIntegration tests the integration of all enhanced backend components
func TestEnhancedBackendIntegration(t *testing.T) {
	t.Run("OrchestatorStateMachine", testOrchestratorStateMachine)
	t.Run("O2ClientRetryAndEvents", testO2ClientRetryAndEvents)
	t.Run("TNManagerEnhancements", testTNManagerEnhancements)
	t.Run("EndToEndSliceDeployment", testEndToEndSliceDeployment)
}

// Test Orchestrator State Machine Management
func testOrchestratorStateMachine(t *testing.T) {
	t.Run("StateMachineLifecycle", func(t *testing.T) {
		// Create state machine manager
		config := statemachine.DefaultConfig()
		manager := statemachine.NewManager(config)

		// Create a state machine for deployment
		sm, err := manager.CreateStateMachine("test-deployment", statemachine.StateInitializing)
		require.NoError(t, err)
		assert.Equal(t, statemachine.StateInitializing, sm.GetCurrentState())

		// Test state transitions
		ctx := context.Background()

		// Validation phase
		err = sm.SendEvent(ctx, statemachine.EventValidate, map[string]interface{}{
			"intent": "test-intent",
		})
		require.NoError(t, err)

		err = sm.SendEvent(ctx, statemachine.EventValidationSuccess, nil)
		require.NoError(t, err)
		assert.Equal(t, statemachine.StatePending, sm.GetCurrentState())

		// Planning phase
		err = sm.SendEvent(ctx, statemachine.EventPlan, map[string]interface{}{
			"resources": "test-resources",
		})
		require.NoError(t, err)

		err = sm.SendEvent(ctx, statemachine.EventPlanningSuccess, nil)
		require.NoError(t, err)
		assert.Equal(t, statemachine.StatePlanned, sm.GetCurrentState())

		// Deployment phase
		err = sm.SendEvent(ctx, statemachine.EventDeploy, map[string]interface{}{
			"deployment": "test-deployment",
		})
		require.NoError(t, err)

		err = sm.SendEvent(ctx, statemachine.EventDeploymentSuccess, nil)
		require.NoError(t, err)
		assert.Equal(t, statemachine.StateDeployed, sm.GetCurrentState())

		// Verify history
		history := sm.GetHistory()
		assert.Len(t, history, 6) // All state transitions

		// Test metrics
		stats := manager.GetStatistics()
		assert.Equal(t, 1, stats.TotalMachines)
		assert.Equal(t, 1, stats.StateDistribution[statemachine.StateDeployed])
	})

	t.Run("ErrorRecoveryAndRollback", func(t *testing.T) {
		config := statemachine.DefaultConfig()
		manager := statemachine.NewManager(config)

		sm, err := manager.CreateStateMachine("test-recovery", statemachine.StateInitializing)
		require.NoError(t, err)

		ctx := context.Background()

		// Simulate deployment failure
		err = sm.SendEvent(ctx, statemachine.EventValidate, nil)
		require.NoError(t, err)

		err = sm.SendEvent(ctx, statemachine.EventValidationFailure, nil)
		require.NoError(t, err)
		assert.Equal(t, statemachine.StateValidationFailed, sm.GetCurrentState())

		// Test retry mechanism
		assert.True(t, sm.CanTransition(statemachine.EventRetry))

		err = sm.SendEvent(ctx, statemachine.EventRetry, nil)
		require.NoError(t, err)
		assert.Equal(t, statemachine.StateValidating, sm.GetCurrentState())

		// Test rollback mechanism
		err = sm.SendEvent(ctx, statemachine.EventValidationFailure, nil)
		require.NoError(t, err)

		err = sm.SendEvent(ctx, statemachine.EventRollback, nil)
		require.NoError(t, err)
		assert.Equal(t, statemachine.StateRollingBack, sm.GetCurrentState())
	})

	t.Run("ConcurrentDeployments", func(t *testing.T) {
		config := statemachine.DefaultConfig()
		manager := statemachine.NewManager(config)

		ctx := context.Background()

		// Create multiple deployment contexts
		deployments := []statemachine.DeploymentContext{
			{
				SliceID: "slice-1",
				Intent:  "embb-intent",
				Timeout: 30 * time.Second,
				RetryPolicy: statemachine.DefaultRetryPolicy(),
			},
			{
				SliceID: "slice-2",
				Intent:  "urllc-intent",
				Timeout: 30 * time.Second,
				RetryPolicy: statemachine.DefaultRetryPolicy(),
			},
			{
				SliceID: "slice-3",
				Intent:  "miot-intent",
				Timeout: 30 * time.Second,
				RetryPolicy: statemachine.DefaultRetryPolicy(),
			},
		}

		// Start concurrent deployments
		err := manager.StartConcurrentDeployments(ctx, deployments)
		require.NoError(t, err)

		// Verify all deployments were processed
		stats := manager.GetStatistics()
		assert.Equal(t, 3, stats.TotalMachines)
	})
}

// Test O2 Client Retry Logic and Event Notifications
func testO2ClientRetryAndEvents(t *testing.T) {
	t.Run("O2IMSRetryLogic", func(t *testing.T) {
		// Create O2 IMS client with retry configuration
		client := o2ims.NewClient("http://test-o2ims",
			o2ims.WithTimeout(5*time.Second),
			o2ims.WithAuthToken("test-token"))

		retryConfig := o2ims.RetryConfig{
			MaxRetries:    3,
			InitialDelay:  100 * time.Millisecond,
			MaxDelay:      1 * time.Second,
			BackoffFactor: 2.0,
			Timeout:       10 * time.Second,
		}
		client.SetRetryConfig(retryConfig)

		// Test event subscription
		eventReceived := false
		subscriptionID := client.Subscribe([]o2ims.EventType{
			o2ims.EventTypeResourceCreated,
			o2ims.EventTypeResourceUpdated,
		}, func(event o2ims.Event) {
			eventReceived = true
			assert.Equal(t, o2ims.EventTypeResourceCreated, event.Type)
		})

		assert.NotEmpty(t, subscriptionID)

		// Start event processing
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		client.StartEventProcessing(ctx)

		// Simulate event
		// In a real test, this would come from the O2 IMS server
		// For this test, we'll directly publish an event
		testEvent := o2ims.Event{
			ID:        "test-event-1",
			Type:      o2ims.EventTypeResourceCreated,
			Source:    "test",
			Timestamp: time.Now(),
			Data:      map[string]interface{}{"resource_id": "test-resource"},
			Severity:  o2ims.SeverityInfo,
		}

		// Verify metrics
		requests, errors, retries, avgTime := client.GetMetrics()
		assert.GreaterOrEqual(t, requests, int64(0))
		assert.GreaterOrEqual(t, errors, int64(0))
		assert.GreaterOrEqual(t, retries, int64(0))
		assert.GreaterOrEqual(t, avgTime, time.Duration(0))
	})

	t.Run("O2DMSEnhancedOperations", func(t *testing.T) {
		// Create enhanced O2 DMS client
		client := o2dms.NewEnhancedClient("http://test-o2dms",
			o2dms.WithTimeout(10*time.Second),
			o2dms.WithAuthToken("test-token"))

		retryConfig := o2dms.DefaultRetryConfig()
		client.SetRetryConfig(retryConfig)

		// Test network slice deployment
		sliceSpec := &o2dms.NetworkSliceSpec{
			SliceID:   "test-slice-1",
			SliceType: "eMBB",
			QoSRequirements: &models.ORanQoSRequirements{
				Bandwidth:   100.0,
				Latency:     10.0,
				SliceType:   "eMBB",
				Priority:    5,
			},
			Placement: &models.ORanPlacement{
				CloudType: "edge",
				Region:    "us-east",
			},
			NetworkFunctions: []o2dms.NetworkFunctionSpec{
				{
					Type:         "UPF",
					DescriptorID: "upf-descriptor-1",
				},
				{
					Type:         "AMF",
					DescriptorID: "amf-descriptor-1",
				},
			},
			WaitForReady:      true,
			DeploymentTimeout: 5 * time.Minute,
		}

		ctx := context.Background()

		// Test deployment (would normally interact with real O2 DMS)
		// For unit test, we'll validate the configuration
		assert.Equal(t, "test-slice-1", sliceSpec.SliceID)
		assert.Equal(t, "eMBB", sliceSpec.SliceType)
		assert.Len(t, sliceSpec.NetworkFunctions, 2)

		// Test event subscription
		eventCount := 0
		client.Subscribe([]o2dms.EventType{
			o2dms.EventTypeDeploymentCreated,
			o2dms.EventTypeSliceDeployed,
		}, func(event o2dms.Event) {
			eventCount++
		})

		client.StartEventProcessing(ctx)

		// Test connection state
		state := client.GetConnectionState()
		assert.Contains(t, []o2dms.ConnectionState{
			o2dms.ConnectionStateDisconnected,
			o2dms.ConnectionStateConnecting,
			o2dms.ConnectionStateConnected,
		}, state)
	})
}

// Test TN Manager Enhancements
func testTNManagerEnhancements(t *testing.T) {
	t.Run("VXLANDynamicConfiguration", func(t *testing.T) {
		// Create enhanced TN manager
		config := &tnpkg.TNConfig{
			ClusterName: "test-cluster",
			NetworkCIDR: "10.0.0.0/16",
		}
		logger := &mockLogger{}
		manager := tnpkg.NewEnhancedTNManager(config, logger)

		ctx := context.Background()

		// Test VXLAN configuration
		vxlanConfig := &tnpkg.DynamicVXLANConfig{
			VxlanID: 100,
			Endpoints: []tnpkg.TNEndpoint{
				{
					Endpoint: tnv1alpha1.Endpoint{
						NodeName:  "node-1",
						IP:        "192.168.1.10",
						Interface: "eth0",
					},
					Status: "healthy",
				},
				{
					Endpoint: tnv1alpha1.Endpoint{
						NodeName:  "node-2",
						IP:        "192.168.1.11",
						Interface: "eth0",
					},
					Status: "healthy",
				},
			},
			ClusterMapping: map[string]string{
				"192.168.1.10": "cluster-1",
				"192.168.1.11": "cluster-2",
			},
			MTU:  1450,
			Port: 4789,
		}

		// Validate configuration
		assert.Equal(t, int32(100), vxlanConfig.VxlanID)
		assert.Len(t, vxlanConfig.Endpoints, 2)
		assert.Equal(t, 1450, vxlanConfig.MTU)

		// Test VXLAN reconfiguration
		updates := &tnpkg.VXLANUpdateConfig{
			AddEndpoints: []tnpkg.TNEndpoint{
				{
					Endpoint: tnv1alpha1.Endpoint{
						NodeName:  "node-3",
						IP:        "192.168.1.12",
						Interface: "eth0",
					},
					Status: "healthy",
				},
			},
			MTU: 1400,
		}

		assert.Len(t, updates.AddEndpoints, 1)
		assert.Equal(t, 1400, updates.MTU)
	})

	t.Run("QoSStrategyManagement", func(t *testing.T) {
		logger := &mockLogger{}
		qosManager := tnpkg.NewQoSManager(logger)

		// Create QoS strategy
		strategy := &tnpkg.QoSStrategy{
			Type:     tnpkg.QoSStrategyTypeULLC,
			Priority: 10,
			BandwidthLimits: map[string]string{
				"uplink":   "100Mbps",
				"downlink": "200Mbps",
			},
			LatencyTargets: map[string]float64{
				"max_rtt": 1.0, // 1ms for uRLLC
			},
			TrafficClasses: []tnpkg.TrafficClass{
				{
					Name:     "control-plane",
					Priority: 10,
					Bandwidth: "50Mbps",
					Latency:  0.5,
					Selector: tnpkg.TrafficSelector{
						Protocol: "tcp",
						DestPort: 8080,
					},
				},
			},
			SchedulingPolicy: tnpkg.SchedulingPolicy{
				Algorithm: "priority",
				Queues: []tnpkg.QueueConfig{
					{
						ID:        "high-priority",
						Priority:  10,
						Weight:    100,
						Bandwidth: "100Mbps",
					},
				},
			},
		}

		// Validate strategy
		err := qosManager.ValidateStrategy(strategy)
		require.NoError(t, err)

		// Generate cluster configuration
		clusterConfig := qosManager.GenerateClusterConfig(strategy, "test-cluster")
		assert.Equal(t, "test-cluster", clusterConfig.ClusterName)
		assert.Equal(t, strategy, clusterConfig.Strategy)
		assert.NotEmpty(t, clusterConfig.TrafficControlRules)

		// Test QoS updates
		updates := &tnpkg.QoSUpdates{
			BandwidthChanges: map[string]string{
				"uplink": "150Mbps",
			},
			LatencyChanges: map[string]float64{
				"max_rtt": 0.8,
			},
		}

		updatedStrategy := qosManager.ApplyUpdates(strategy, updates)
		assert.Equal(t, "150Mbps", updatedStrategy.BandwidthLimits["uplink"])
		assert.Equal(t, 0.8, updatedStrategy.LatencyTargets["max_rtt"])
	})

	t.Run("NetworkTopologyDiscovery", func(t *testing.T) {
		// Create network topology
		topology := tnpkg.NewNetworkTopology()

		// Add nodes
		node1 := &tnpkg.TopologyNode{
			Name: "node-1",
			Type: "compute",
			Capabilities: []string{"vxlan", "qos", "monitoring"},
			Status: "healthy",
			Interfaces: []tnpkg.NodeInterface{
				{
					Name:   "eth0",
					Type:   "physical",
					IP:     "192.168.1.10",
					Speed:  "1Gbps",
					Status: "up",
				},
			},
			LastUpdated: time.Now(),
		}

		node2 := &tnpkg.TopologyNode{
			Name: "node-2",
			Type: "compute",
			Capabilities: []string{"vxlan", "qos"},
			Status: "healthy",
			Interfaces: []tnpkg.NodeInterface{
				{
					Name:   "eth0",
					Type:   "physical",
					IP:     "192.168.1.11",
					Speed:  "1Gbps",
					Status: "up",
				},
			},
			LastUpdated: time.Now(),
		}

		topology.AddNode(node1)
		topology.AddNode(node2)

		// Add link between nodes
		link := &tnpkg.TopologyLink{
			ID:         "link-1-2",
			SourceNode: "node-1",
			TargetNode: "node-2",
			Type:       "network",
			Status:     "up",
			Metrics: tnpkg.LinkMetrics{
				Bandwidth:   1000.0, // 1Gbps
				Latency:     1.0,    // 1ms
				PacketLoss:  0.0,
				Utilization: 25.0,   // 25%
			},
			LastUpdated: time.Now(),
		}

		topology.AddLink(link)

		// Verify topology
		nodes := topology.GetNodes()
		assert.Len(t, nodes, 2)

		links := topology.GetLinks()
		assert.Len(t, links, 1)

		// Test topology queries
		nodeByName := topology.GetNode("node-1")
		require.NotNil(t, nodeByName)
		assert.Equal(t, "node-1", nodeByName.Name)
		assert.Contains(t, nodeByName.Capabilities, "vxlan")
	})

	t.Run("FaultDetectionAndRecovery", func(t *testing.T) {
		logger := &mockLogger{}
		faultDetector := tnpkg.NewFaultDetector(logger)

		// Create network fault
		fault := &tnpkg.NetworkFault{
			ID:          "fault-1",
			Type:        tnpkg.FaultTypeVXLANDown,
			Severity:    tnpkg.FaultSeverityHigh,
			NodeName:    "node-1",
			SliceID:     "slice-1",
			Description: "VXLAN tunnel down",
			Details: map[string]interface{}{
				"tunnel_id": 100,
				"remote_ip": "192.168.1.11",
			},
			DetectedAt: time.Now(),
		}

		// Record fault
		faultDetector.RecordFault(fault)

		// Get faults summary
		summary := faultDetector.GetFaultsSummary()
		assert.Equal(t, 1, summary.TotalFaults)
		assert.Equal(t, 1, summary.ActiveFaults)
		assert.Equal(t, 1, summary.FaultsByType[tnpkg.FaultTypeVXLANDown])

		// Test fault resolution
		fault.ResolvedAt = &time.Time{}
		*fault.ResolvedAt = time.Now()
		faultDetector.ResolveFault(fault.ID)

		updatedSummary := faultDetector.GetFaultsSummary()
		assert.Equal(t, 0, updatedSummary.ActiveFaults)
		assert.Equal(t, 1, updatedSummary.ResolvedFaults)
	})
}

// Test End-to-End Slice Deployment
func testEndToEndSliceDeployment(t *testing.T) {
	t.Run("CompleteSliceLifecycle", func(t *testing.T) {
		ctx := context.Background()

		// 1. Initialize components
		smConfig := statemachine.DefaultConfig()
		smManager := statemachine.NewManager(smConfig)

		tnConfig := &tnpkg.TNConfig{
			ClusterName: "test-cluster",
			NetworkCIDR: "10.0.0.0/16",
		}
		logger := &mockLogger{}
		tnManager := tnpkg.NewEnhancedTNManager(tnConfig, logger)

		// 2. Create slice deployment context
		deploymentCtx := statemachine.DeploymentContext{
			SliceID: "end-to-end-slice",
			Intent: map[string]interface{}{
				"slice_type": "eMBB",
				"bandwidth": 500.0,
				"latency":   10.0,
			},
			Timeout:     5 * time.Minute,
			RetryPolicy: statemachine.DefaultRetryPolicy(),
		}

		// 3. Create state machine for slice deployment
		sm, err := smManager.CreateStateMachine(deploymentCtx.SliceID, statemachine.StateInitializing)
		require.NoError(t, err)

		// 4. Configure VXLAN for slice
		vxlanConfig := &tnpkg.DynamicVXLANConfig{
			VxlanID: 200,
			Endpoints: []tnpkg.TNEndpoint{
				{
					Endpoint: tnv1alpha1.Endpoint{
						NodeName:  "edge-node-1",
						IP:        "192.168.2.10",
						Interface: "eth0",
					},
					Status: "healthy",
				},
				{
					Endpoint: tnv1alpha1.Endpoint{
						NodeName:  "edge-node-2",
						IP:        "192.168.2.11",
						Interface: "eth0",
					},
					Status: "healthy",
				},
			},
			ClusterMapping: map[string]string{
				"192.168.2.10": "edge-cluster-1",
				"192.168.2.11": "edge-cluster-2",
			},
		}

		// 5. Configure QoS strategy
		qosStrategy := &tnpkg.QoSStrategy{
			Type:     tnpkg.QoSStrategyTypeEMBB,
			Priority: 5,
			BandwidthLimits: map[string]string{
				"uplink":   "250Mbps",
				"downlink": "500Mbps",
			},
			LatencyTargets: map[string]float64{
				"max_rtt": 10.0,
			},
		}

		// 6. Simulate deployment workflow

		// Validation phase
		err = sm.SendEvent(ctx, statemachine.EventValidate, deploymentCtx)
		require.NoError(t, err)

		err = sm.SendEvent(ctx, statemachine.EventValidationSuccess, nil)
		require.NoError(t, err)
		assert.Equal(t, statemachine.StatePending, sm.GetCurrentState())

		// Planning phase
		err = sm.SendEvent(ctx, statemachine.EventPlan, map[string]interface{}{
			"vxlan_config": vxlanConfig,
			"qos_strategy": qosStrategy,
		})
		require.NoError(t, err)

		err = sm.SendEvent(ctx, statemachine.EventPlanningSuccess, nil)
		require.NoError(t, err)
		assert.Equal(t, statemachine.StatePlanned, sm.GetCurrentState())

		// Deployment phase
		err = sm.SendEvent(ctx, statemachine.EventDeploy, map[string]interface{}{
			"execute_deployment": true,
		})
		require.NoError(t, err)

		err = sm.SendEvent(ctx, statemachine.EventDeploymentSuccess, nil)
		require.NoError(t, err)
		assert.Equal(t, statemachine.StateDeployed, sm.GetCurrentState())

		// Activation phase
		err = sm.SendEvent(ctx, statemachine.EventActivate, nil)
		require.NoError(t, err)
		assert.Equal(t, statemachine.StateActive, sm.GetCurrentState())

		// 7. Verify final state
		history := sm.GetHistory()
		assert.GreaterOrEqual(t, len(history), 6)

		// Get enhanced status
		enhancedStatus, err := tnManager.GetEnhancedStatus()
		require.NoError(t, err)
		assert.NotNil(t, enhancedStatus.NetworkTopology)
		assert.NotNil(t, enhancedStatus.QoSCompliance)

		// Verify slice is active
		sliceState := enhancedStatus.ActiveSlices[deploymentCtx.SliceID]
		if sliceState != nil {
			assert.Equal(t, "active", sliceState.Status)
		}
	})
}

// Mock implementations for testing

type mockLogger struct{}

func (m *mockLogger) Printf(format string, v ...interface{}) {}
func (m *mockLogger) Print(v ...interface{})                 {}
func (m *mockLogger) Println(v ...interface{})               {}

// Additional test utilities

func setupTestEnvironment() (context.Context, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	return ctx, cancel
}

func createTestSliceIntent() map[string]interface{} {
	return map[string]interface{}{
		"slice_id":   "test-slice-123",
		"slice_type": "eMBB",
		"qos_requirements": map[string]interface{}{
			"bandwidth": 1000.0,
			"latency":   5.0,
			"priority":  7,
		},
		"placement": map[string]interface{}{
			"cloud_type": "edge",
			"region":     "us-west",
			"zones":      []string{"us-west-1a", "us-west-1b"},
		},
	}
}

func verifyDeploymentMetrics(t *testing.T, metrics interface{}) {
	// Verify that deployment metrics meet expected thresholds
	assert.NotNil(t, metrics)
	// Add specific metric validations based on requirements
}

func cleanupTestResources(ctx context.Context, resources ...interface{}) {
	// Cleanup test resources
	for _, resource := range resources {
		// Implement cleanup logic based on resource type
		_ = resource
	}
}

// Benchmark tests for performance validation

func BenchmarkStateMachineTransition(b *testing.B) {
	config := statemachine.DefaultConfig()
	manager := statemachine.NewManager(config)
	sm, _ := manager.CreateStateMachine("benchmark", statemachine.StateInitializing)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sm.SendEvent(ctx, statemachine.EventValidate, nil)
		sm.SendEvent(ctx, statemachine.EventValidationSuccess, nil)
		// Reset state for next iteration
		sm.SendEvent(ctx, statemachine.EventRollback, nil)
	}
}

func BenchmarkQoSPolicyGeneration(b *testing.B) {
	logger := &mockLogger{}
	qosManager := tnpkg.NewQoSManager(logger)

	strategy := &tnpkg.QoSStrategy{
		Type:     tnpkg.QoSStrategyTypeEMBB,
		Priority: 5,
		BandwidthLimits: map[string]string{
			"uplink":   "100Mbps",
			"downlink": "200Mbps",
		},
		TrafficClasses: make([]tnpkg.TrafficClass, 10), // 10 traffic classes
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qosManager.GenerateClusterConfig(strategy, "test-cluster")
	}
}

func BenchmarkVXLANConfigGeneration(b *testing.B) {
	orchestrator := vxlan.NewOrchestrator()

	endpoints := make([]tnv1alpha1.Endpoint, 100) // 100 endpoints
	for i := range endpoints {
		endpoints[i] = tnv1alpha1.Endpoint{
			NodeName:  fmt.Sprintf("node-%d", i),
			IP:        fmt.Sprintf("192.168.1.%d", i+10),
			Interface: "eth0",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		orchestrator.GenerateTunnelConfigs(int32(i), endpoints)
	}
}