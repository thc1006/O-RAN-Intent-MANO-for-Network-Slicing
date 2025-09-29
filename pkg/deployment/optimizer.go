package deployment

import (
	"context"
	"fmt"
	"math"
	"sort"
)

// ResourceOptimizer optimizes resource allocation
type ResourceOptimizer struct {
	threshold float64
}

// NewResourceOptimizer creates a new resource optimizer
func NewResourceOptimizer() *ResourceOptimizer {
	return &ResourceOptimizer{
		threshold: 70.0, // Target utilization threshold
	}
}

// OptimizeResources optimizes resource allocation
func (o *ResourceOptimizer) OptimizeResources(ctx context.Context, state *ResourceState) (*OptimizationPlan, error) {
	if state == nil || len(state.Nodes) == 0 {
		return nil, fmt.Errorf("invalid resource state")
	}

	plan := &OptimizationPlan{
		Migrations: []Migration{},
	}

	// Find overloaded and underutilized nodes
	overloaded := []NodeResource{}
	underutilized := []NodeResource{}

	for _, node := range state.Nodes {
		avgUtilization := (node.CPUUsed + node.MemoryUsed) / 2
		if avgUtilization > o.threshold {
			overloaded = append(overloaded, node)
		} else if avgUtilization < o.threshold-20 {
			underutilized = append(underutilized, node)
		}
	}

	// Create migration plan
	for _, overNode := range overloaded {
		// Find pods on overloaded node
		podsToMigrate := o.findPodsOnNode(state.Pods, overNode.Name)
		
		for _, pod := range podsToMigrate {
			// Find best target node
			targetNode := o.findBestTargetNode(underutilized, pod)
			if targetNode != "" {
				plan.Migrations = append(plan.Migrations, Migration{
					Pod:      pod.Name,
					FromNode: overNode.Name,
					ToNode:   targetNode,
				})
			}
		}
	}

	// Calculate expected improvement
	plan.ExpectedImprovement = o.calculateImprovement(state, plan)

	return plan, nil
}

func (o *ResourceOptimizer) findPodsOnNode(pods []PodResource, nodeName string) []PodResource {
	result := []PodResource{}
	for _, pod := range pods {
		if pod.Node == nodeName {
			result = append(result, pod)
		}
	}
	// Sort by resource consumption (highest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CPU+result[i].Memory > result[j].CPU+result[j].Memory
	})
	return result
}

func (o *ResourceOptimizer) findBestTargetNode(nodes []NodeResource, pod PodResource) string {
	bestNode := ""
	bestScore := math.MaxFloat64

	for _, node := range nodes {
		// Check if node has capacity
		newCPU := node.CPUUsed + pod.CPU
		newMemory := node.MemoryUsed + pod.Memory
		
		if newCPU <= o.threshold && newMemory <= o.threshold {
			// Calculate score (lower is better)
			score := math.Abs(newCPU-o.threshold) + math.Abs(newMemory-o.threshold)
			if score < bestScore {
				bestScore = score
				bestNode = node.Name
			}
		}
	}

	return bestNode
}

func (o *ResourceOptimizer) calculateImprovement(state *ResourceState, plan *OptimizationPlan) float64 {
	if len(plan.Migrations) == 0 {
		return 0.0
	}

	// Calculate current standard deviation
	currentStdDev := o.calculateStdDev(state.Nodes)

	// Simulate migrations and calculate new standard deviation
	newState := o.simulateMigrations(state, plan)
	newStdDev := o.calculateStdDev(newState.Nodes)

	// Improvement is reduction in standard deviation
	improvement := ((currentStdDev - newStdDev) / currentStdDev) * 100
	if improvement < 0 {
		improvement = 0
	}

	return improvement
}

func (o *ResourceOptimizer) calculateStdDev(nodes []NodeResource) float64 {
	if len(nodes) == 0 {
		return 0
	}

	// Calculate mean utilization
	var sum float64
	for _, node := range nodes {
		sum += (node.CPUUsed + node.MemoryUsed) / 2
	}
	mean := sum / float64(len(nodes))

	// Calculate standard deviation
	var variance float64
	for _, node := range nodes {
		utilization := (node.CPUUsed + node.MemoryUsed) / 2
		diff := utilization - mean
		variance += diff * diff
	}

	return math.Sqrt(variance / float64(len(nodes)))
}

func (o *ResourceOptimizer) simulateMigrations(state *ResourceState, plan *OptimizationPlan) *ResourceState {
	// Create a copy of the current state
	newState := &ResourceState{
		Nodes: make([]NodeResource, len(state.Nodes)),
		Pods:  make([]PodResource, len(state.Pods)),
	}

	// Copy nodes
	for i, node := range state.Nodes {
		newState.Nodes[i] = node
	}

	// Copy pods
	for i, pod := range state.Pods {
		newState.Pods[i] = pod
	}

	// Apply migrations
	for _, migration := range plan.Migrations {
		// Find pod
		for i, pod := range newState.Pods {
			if pod.Name == migration.Pod {
				// Update node utilizations
				for j, node := range newState.Nodes {
					if node.Name == migration.FromNode {
						newState.Nodes[j].CPUUsed -= pod.CPU
						newState.Nodes[j].MemoryUsed -= pod.Memory
					}
					if node.Name == migration.ToNode {
						newState.Nodes[j].CPUUsed += pod.CPU
						newState.Nodes[j].MemoryUsed += pod.Memory
					}
				}
				// Update pod's node
				newState.Pods[i].Node = migration.ToNode
				break
			}
		}
	}

	return newState
}

// AutoScaler manages automatic scaling
type AutoScaler struct {
	history []LoadMetrics
}

// NewAutoScaler creates a new auto scaler
func NewAutoScaler() *AutoScaler {
	return &AutoScaler{
		history: make([]LoadMetrics, 0, 100),
	}
}

// MakeScalingDecision makes a scaling decision based on metrics
func (a *AutoScaler) MakeScalingDecision(ctx context.Context, metrics *LoadMetrics, policy *ScalingPolicy) (*ScalingDecision, error) {
	if metrics == nil || policy == nil {
		return nil, fmt.Errorf("invalid input")
	}

	// Add to history for trend analysis
	a.history = append(a.history, *metrics)
	if len(a.history) > 100 {
		a.history = a.history[1:]
	}

	decision := &ScalingDecision{
		Action:          NoChange,
		CurrentReplicas: 3, // Assume current replicas
		NewReplicas:     3,
	}

	// Check if scale up is needed
	if metrics.CPUUtilization > float64(policy.ScaleUpThreshold) ||
		metrics.MemoryUtilization > float64(policy.ScaleUpThreshold) {
		decision.Action = ScaleUp
		decision.NewReplicas = min(decision.CurrentReplicas+1, policy.MaxReplicas)
		decision.Reason = fmt.Sprintf("CPU: %.1f%%, Memory: %.1f%% exceed threshold %d%%",
			metrics.CPUUtilization, metrics.MemoryUtilization, policy.ScaleUpThreshold)
	}

	// Check if scale down is needed
	if metrics.CPUUtilization < float64(policy.ScaleDownThreshold) &&
		metrics.MemoryUtilization < float64(policy.ScaleDownThreshold) {
		// Check if load has been low for sufficient time
		if a.isLoadConsistentlyLow(policy.ScaleDownThreshold) {
			decision.Action = ScaleDown
			decision.NewReplicas = max(decision.CurrentReplicas-1, policy.MinReplicas)
			decision.Reason = fmt.Sprintf("CPU: %.1f%%, Memory: %.1f%% below threshold %d%%",
				metrics.CPUUtilization, metrics.MemoryUtilization, policy.ScaleDownThreshold)
		}
	}

	return decision, nil
}

func (a *AutoScaler) isLoadConsistentlyLow(threshold int) bool {
	if len(a.history) < 5 {
		return false
	}

	// Check last 5 measurements
	for i := len(a.history) - 5; i < len(a.history); i++ {
		if a.history[i].CPUUtilization > float64(threshold) ||
			a.history[i].MemoryUtilization > float64(threshold) {
			return false
		}
	}

	return true
}

func (a *AutoScaler) PredictFutureLoad(ctx context.Context) (*LoadMetrics, error) {
	if len(a.history) < 10 {
		return nil, fmt.Errorf("insufficient history for prediction")
	}

	// Simple moving average prediction
	var sumCPU, sumMemory, sumRequests float64
	count := min(10, len(a.history))
	
	for i := len(a.history) - count; i < len(a.history); i++ {
		sumCPU += a.history[i].CPUUtilization
		sumMemory += a.history[i].MemoryUtilization
		sumRequests += float64(a.history[i].RequestRate)
	}

	predicted := &LoadMetrics{
		CPUUtilization:    sumCPU / float64(count),
		MemoryUtilization: sumMemory / float64(count),
		RequestRate:       int(sumRequests / float64(count)),
		ResponseTime:      150, // Default
	}

	return predicted, nil
}

// Helper functions

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
