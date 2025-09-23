package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	tnv1alpha1 "github.com/o-ran/intent-mano/tn/manager/api/v1alpha1"
	"github.com/o-ran/intent-mano/tn/manager/pkg/tc"
	"github.com/o-ran/intent-mano/tn/manager/pkg/vxlan"
)

const (
	tnSliceFinalizer = "tn.oran.io/finalizer"
	agentConfigMap   = "tn-agent-config"
)

// TNSliceReconciler reconciles a TNSlice object
type TNSliceReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	TCCalculator   *tc.Calculator
	VXLANOrchestrator *vxlan.Orchestrator
}

// AgentConfig represents the configuration passed to TN agents
type AgentConfig struct {
	SliceID   string           `json:"sliceId"`
	VxlanID   int32            `json:"vxlanId"`
	TCRules   []tc.Rule        `json:"tcRules"`
	Tunnels   []vxlan.TunnelConfig `json:"tunnels"`
	Priority  int32            `json:"priority"`
}

//+kubebuilder:rbac:groups=tn.oran.io,resources=tnslices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=tn.oran.io,resources=tnslices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=tn.oran.io,resources=tnslices/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop
func (r *TNSliceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the TNSlice instance
	slice := &tnv1alpha1.TNSlice{}
	if err := r.Get(ctx, req.NamespacedName, slice); err != nil {
		if errors.IsNotFound(err) {
			log.Info("TNSlice resource not found")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion
	if slice.ObjectMeta.DeletionTimestamp != nil {
		if controllerutil.ContainsFinalizer(slice, tnSliceFinalizer) {
			if err := r.finalizeTNSlice(ctx, slice); err != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(slice, tnSliceFinalizer)
			if err := r.Update(ctx, slice); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer
	if !controllerutil.ContainsFinalizer(slice, tnSliceFinalizer) {
		controllerutil.AddFinalizer(slice, tnSliceFinalizer)
		if err := r.Update(ctx, slice); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Initialize status if needed
	if slice.Status.Phase == "" {
		slice.Status.Phase = "Pending"
		slice.Status.ObservedGeneration = slice.Generation
		if err := r.Status().Update(ctx, slice); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Main reconciliation
	switch slice.Status.Phase {
	case "Pending":
		return r.handlePending(ctx, slice)
	case "Configuring":
		return r.handleConfiguring(ctx, slice)
	case "Active":
		return r.handleActive(ctx, slice)
	case "Failed":
		return r.handleFailed(ctx, slice)
	}

	return ctrl.Result{}, nil
}

func (r *TNSliceReconciler) handlePending(ctx context.Context, slice *tnv1alpha1.TNSlice) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling Pending TNSlice", "sliceId", slice.Spec.SliceID)

	// Apply profile-based defaults if specified
	if slice.Spec.Profile != "" {
		r.applyProfileDefaults(slice)
	}

	// Validate slice configuration
	if err := r.validateSlice(slice); err != nil {
		return r.updateStatusWithError(ctx, slice, "ValidationFailed", err)
	}

	// Calculate TC parameters
	tcRules := r.TCCalculator.CalculateRules(slice.Spec.Bandwidth, slice.Spec.Latency,
		slice.Spec.Jitter, slice.Spec.PacketLoss, slice.Spec.Priority)

	// Generate VXLAN tunnel configurations
	tunnels := r.VXLANOrchestrator.GenerateTunnelConfigs(slice.Spec.VxlanID, slice.Spec.Endpoints)

	// Create agent configuration
	agentConfig := AgentConfig{
		SliceID:  slice.Spec.SliceID,
		VxlanID:  slice.Spec.VxlanID,
		TCRules:  tcRules,
		Tunnels:  tunnels,
		Priority: slice.Spec.Priority,
	}

	// Deploy configuration to agents via ConfigMap
	if err := r.deployAgentConfig(ctx, slice, agentConfig); err != nil {
		return r.updateStatusWithError(ctx, slice, "ConfigDeploymentFailed", err)
	}

	// Update status
	slice.Status.Phase = "Configuring"
	slice.Status.LastConfigTime = &metav1.Time{Time: time.Now()}
	r.setCondition(slice, "ConfigDeployed", metav1.ConditionTrue, "Success",
		fmt.Sprintf("Configuration deployed for slice %s", slice.Spec.SliceID))

	if err := r.Status().Update(ctx, slice); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

func (r *TNSliceReconciler) handleConfiguring(ctx context.Context, slice *tnv1alpha1.TNSlice) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Checking configuration status", "sliceId", slice.Spec.SliceID)

	// Check if agents have applied the configuration
	configured, err := r.checkAgentStatus(ctx, slice)
	if err != nil {
		log.Error(err, "Failed to check agent status")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if !configured {
		// Still waiting for agents to configure
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// Update tunnel status
	tunnelStatus := r.generateTunnelStatus(slice)
	slice.Status.ActiveTunnels = tunnelStatus

	// Update configured nodes
	slice.Status.ConfiguredNodes = r.getConfiguredNodes(slice)

	// Move to Active phase
	slice.Status.Phase = "Active"
	r.setCondition(slice, "SliceActive", metav1.ConditionTrue, "Success",
		fmt.Sprintf("Slice %s is active with %d tunnels", slice.Spec.SliceID, len(tunnelStatus)))

	if err := r.Status().Update(ctx, slice); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *TNSliceReconciler) handleActive(ctx context.Context, slice *tnv1alpha1.TNSlice) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.V(1).Info("Monitoring active slice", "sliceId", slice.Spec.SliceID)

	// Check if spec has changed
	if slice.Generation != slice.Status.ObservedGeneration {
		log.Info("Spec changed, reconfiguring slice")
		slice.Status.Phase = "Pending"
		slice.Status.ObservedGeneration = slice.Generation
		if err := r.Status().Update(ctx, slice); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// TODO: Collect metrics from agents and update status.MeasuredMetrics

	// Periodic reconciliation
	return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
}

func (r *TNSliceReconciler) handleFailed(ctx context.Context, slice *tnv1alpha1.TNSlice) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("TNSlice in Failed state", "sliceId", slice.Spec.SliceID)

	// Check if generation changed (user updated spec)
	if slice.Generation != slice.Status.ObservedGeneration {
		log.Info("Spec updated, retrying configuration")
		slice.Status.Phase = "Pending"
		slice.Status.ObservedGeneration = slice.Generation
		if err := r.Status().Update(ctx, slice); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *TNSliceReconciler) deployAgentConfig(ctx context.Context, slice *tnv1alpha1.TNSlice, config AgentConfig) error {
	configData, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create ConfigMap for each node
	for _, endpoint := range slice.Spec.Endpoints {
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", agentConfigMap, slice.Spec.SliceID),
				Namespace: slice.Namespace,
				Labels: map[string]string{
					"tn-slice": slice.Spec.SliceID,
					"node":     endpoint.NodeName,
				},
			},
			Data: map[string]string{
				"config.json": string(configData),
				"node":        endpoint.NodeName,
			},
		}

		// Set owner reference
		if err := controllerutil.SetControllerReference(slice, configMap, r.Scheme); err != nil {
			return err
		}

		// Create or update ConfigMap
		foundCM := &corev1.ConfigMap{}
		err := r.Get(ctx, types.NamespacedName{
			Name:      configMap.Name,
			Namespace: configMap.Namespace,
		}, foundCM)

		if err != nil && errors.IsNotFound(err) {
			if err := r.Create(ctx, configMap); err != nil {
				return fmt.Errorf("failed to create configmap: %w", err)
			}
		} else if err == nil {
			foundCM.Data = configMap.Data
			if err := r.Update(ctx, foundCM); err != nil {
				return fmt.Errorf("failed to update configmap: %w", err)
			}
		} else {
			return err
		}
	}

	return nil
}

func (r *TNSliceReconciler) applyProfileDefaults(slice *tnv1alpha1.TNSlice) {
	switch slice.Spec.Profile {
	case "eMBB":
		if slice.Spec.Bandwidth == 0 {
			slice.Spec.Bandwidth = 4.57
		}
		if slice.Spec.Latency == 0 {
			slice.Spec.Latency = 16.1
		}
	case "uRLLC":
		if slice.Spec.Bandwidth == 0 {
			slice.Spec.Bandwidth = 0.93
		}
		if slice.Spec.Latency == 0 {
			slice.Spec.Latency = 6.3
		}
	case "mIoT":
		if slice.Spec.Bandwidth == 0 {
			slice.Spec.Bandwidth = 2.77
		}
		if slice.Spec.Latency == 0 {
			slice.Spec.Latency = 15.7
		}
	}
}

func (r *TNSliceReconciler) validateSlice(slice *tnv1alpha1.TNSlice) error {
	if len(slice.Spec.Endpoints) < 2 {
		return fmt.Errorf("at least 2 endpoints required")
	}

	if slice.Spec.Bandwidth < 0.1 || slice.Spec.Bandwidth > 10 {
		return fmt.Errorf("bandwidth must be between 0.1 and 10 Mbps")
	}

	if slice.Spec.Latency < 1 || slice.Spec.Latency > 100 {
		return fmt.Errorf("latency must be between 1 and 100 ms")
	}

	return nil
}

func (r *TNSliceReconciler) checkAgentStatus(ctx context.Context, slice *tnv1alpha1.TNSlice) (bool, error) {
	// TODO: Implement actual agent status checking
	// For now, assume configuration is applied after a delay
	if slice.Status.LastConfigTime != nil {
		elapsed := time.Since(slice.Status.LastConfigTime.Time)
		return elapsed > 10*time.Second, nil
	}
	return false, nil
}

func (r *TNSliceReconciler) generateTunnelStatus(slice *tnv1alpha1.TNSlice) []tnv1alpha1.TunnelStatus {
	var tunnels []tnv1alpha1.TunnelStatus

	// Generate tunnel status for each endpoint pair
	for i, src := range slice.Spec.Endpoints {
		for j, dst := range slice.Spec.Endpoints {
			if i >= j {
				continue // Avoid duplicates
			}

			tunnel := tnv1alpha1.TunnelStatus{
				TunnelID:      fmt.Sprintf("vxlan%d", slice.Spec.VxlanID),
				SourceIP:      src.IP,
				DestinationIP: dst.IP,
				State:         "up",
			}
			tunnels = append(tunnels, tunnel)
		}
	}

	return tunnels
}

func (r *TNSliceReconciler) getConfiguredNodes(slice *tnv1alpha1.TNSlice) []string {
	nodes := make([]string, 0, len(slice.Spec.Endpoints))
	for _, endpoint := range slice.Spec.Endpoints {
		nodes = append(nodes, endpoint.NodeName)
	}
	return nodes
}

func (r *TNSliceReconciler) finalizeTNSlice(ctx context.Context, slice *tnv1alpha1.TNSlice) error {
	log := log.FromContext(ctx)
	log.Info("Finalizing TNSlice", "sliceId", slice.Spec.SliceID)

	// Delete agent configurations
	for range slice.Spec.Endpoints {
		configMapName := fmt.Sprintf("%s-%s", agentConfigMap, slice.Spec.SliceID)
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: slice.Namespace,
			},
		}

		if err := r.Delete(ctx, configMap); err != nil && !errors.IsNotFound(err) {
			log.Error(err, "Failed to delete configmap", "name", configMapName)
		}
	}

	slice.Status.Phase = "Deleting"
	return nil
}

func (r *TNSliceReconciler) updateStatusWithError(ctx context.Context, slice *tnv1alpha1.TNSlice,
	reason string, err error) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Error(err, "TNSlice reconciliation failed", "reason", reason)

	slice.Status.Phase = "Failed"
	r.setCondition(slice, reason, metav1.ConditionFalse, "Error", err.Error())

	if statusErr := r.Status().Update(ctx, slice); statusErr != nil {
		return ctrl.Result{}, statusErr
	}

	return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
}

func (r *TNSliceReconciler) setCondition(slice *tnv1alpha1.TNSlice, conditionType string,
	status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&slice.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: slice.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *TNSliceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tnv1alpha1.TNSlice{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}