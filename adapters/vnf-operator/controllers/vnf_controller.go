package controllers

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	manov1alpha1 "github.com/o-ran/intent-mano/adapters/vnf-operator/api/v1alpha1"
	"github.com/o-ran/intent-mano/adapters/vnf-operator/pkg/dms"
	"github.com/o-ran/intent-mano/adapters/vnf-operator/pkg/gitops"
	"github.com/o-ran/intent-mano/adapters/vnf-operator/pkg/translator"
)

const (
	vnfFinalizer = "mano.oran.io/finalizer"
)

// VNFReconciler reconciles a VNF object
type VNFReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	PorchTranslator  *translator.PorchTranslator
	DMSClient        dms.Client
	GitOpsClient     gitops.Client
}

//+kubebuilder:rbac:groups=mano.oran.io,resources=vnfs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mano.oran.io,resources=vnfs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=mano.oran.io,resources=vnfs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups="",resources=configmaps;secrets,verbs=get;list;watch;create;update;patch

// Reconcile is part of the main kubernetes reconciliation loop
func (r *VNFReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the VNF instance
	vnf := &manov1alpha1.VNF{}
	if err := r.Get(ctx, req.NamespacedName, vnf); err != nil {
		if errors.IsNotFound(err) {
			log.Info("VNF resource not found, possibly deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get VNF")
		return ctrl.Result{}, err
	}

	// Check if the VNF instance is marked for deletion
	if vnf.ObjectMeta.DeletionTimestamp != nil {
		if controllerutil.ContainsFinalizer(vnf, vnfFinalizer) {
			// Run finalization logic
			if err := r.finalizeVNF(ctx, vnf); err != nil {
				return ctrl.Result{}, err
			}

			// Remove finalizer
			controllerutil.RemoveFinalizer(vnf, vnfFinalizer)
			if err := r.Update(ctx, vnf); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(vnf, vnfFinalizer) {
		controllerutil.AddFinalizer(vnf, vnfFinalizer)
		if err := r.Update(ctx, vnf); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Initialize status if needed
	if vnf.Status.Phase == "" {
		vnf.Status.Phase = "Pending"
		vnf.Status.ObservedGeneration = vnf.Generation
		if err := r.Status().Update(ctx, vnf); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Main reconciliation logic
	switch vnf.Status.Phase {
	case "Pending":
		return r.handlePending(ctx, vnf)
	case "Creating":
		return r.handleCreating(ctx, vnf)
	case "Running":
		return r.handleRunning(ctx, vnf)
	case "Failed":
		return r.handleFailed(ctx, vnf)
	default:
		log.Info("Unknown phase", "phase", vnf.Status.Phase)
		return ctrl.Result{}, nil
	}
}

func (r *VNFReconciler) handlePending(ctx context.Context, vnf *manov1alpha1.VNF) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling Pending VNF", "name", vnf.Name)

	// Validate VNF spec
	if err := r.validateVNF(vnf); err != nil {
		return r.updateStatusWithError(ctx, vnf, "ValidationFailed", err)
	}

	// Translate VNF to Porch package
	pkg, err := r.PorchTranslator.TranslateVNF(vnf)
	if err != nil {
		return r.updateStatusWithError(ctx, vnf, "TranslationFailed", err)
	}

	// Push package to Porch repository
	revision, err := r.GitOpsClient.PushPackage(ctx, pkg)
	if err != nil {
		return r.updateStatusWithError(ctx, vnf, "PorchPushFailed", err)
	}

	// Update status
	vnf.Status.Phase = "Creating"
	vnf.Status.PorchPackageRevision = revision
	r.setCondition(vnf, "PackageCreated", metav1.ConditionTrue, "Success",
		fmt.Sprintf("Package pushed to Porch: %s", revision))

	if err := r.Status().Update(ctx, vnf); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{Requeue: true}, nil
}

func (r *VNFReconciler) handleCreating(ctx context.Context, vnf *manov1alpha1.VNF) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling Creating VNF", "name", vnf.Name)

	// Create DMS deployment request
	deploymentID, err := r.DMSClient.CreateDeployment(ctx, vnf)
	if err != nil {
		return r.updateStatusWithError(ctx, vnf, "DMSDeploymentFailed", err)
	}

	// Update status with DMS deployment ID
	vnf.Status.DMSDeploymentID = deploymentID
	vnf.Status.Phase = "Running"
	vnf.Status.LastReconcileTime = &metav1.Time{Time: time.Now()}
	r.setCondition(vnf, "Deployed", metav1.ConditionTrue, "Success",
		fmt.Sprintf("Deployed via DMS: %s", deploymentID))

	// Update deployed clusters based on target clusters
	vnf.Status.DeployedClusters = vnf.Spec.TargetClusters

	if err := r.Status().Update(ctx, vnf); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *VNFReconciler) handleRunning(ctx context.Context, vnf *manov1alpha1.VNF) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Checking Running VNF status", "name", vnf.Name)

	// Check DMS deployment status
	status, err := r.DMSClient.GetDeploymentStatus(ctx, vnf.Status.DMSDeploymentID)
	if err != nil {
		log.Error(err, "Failed to get DMS deployment status")
		// Don't fail the VNF, just requeue
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	// Update last reconcile time
	vnf.Status.LastReconcileTime = &metav1.Time{Time: time.Now()}

	// Check if deployment has issues
	if status == "Failed" {
		vnf.Status.Phase = "Failed"
		r.setCondition(vnf, "DeploymentFailed", metav1.ConditionTrue, "DMSFailure",
			"DMS deployment reported failure")
	}

	if err := r.Status().Update(ctx, vnf); err != nil {
		return ctrl.Result{}, err
	}

	// Periodic reconciliation
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *VNFReconciler) handleFailed(ctx context.Context, vnf *manov1alpha1.VNF) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("VNF in Failed state", "name", vnf.Name)

	// Check if generation has changed (user updated the spec)
	if vnf.Generation != vnf.Status.ObservedGeneration {
		log.Info("Spec updated, retrying deployment")
		vnf.Status.Phase = "Pending"
		vnf.Status.ObservedGeneration = vnf.Generation
		if err := r.Status().Update(ctx, vnf); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Stay in failed state until user intervention
	return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil
}

func (r *VNFReconciler) finalizeVNF(ctx context.Context, vnf *manov1alpha1.VNF) error {
	log := log.FromContext(ctx)
	log.Info("Finalizing VNF", "name", vnf.Name)

	// Delete DMS deployment
	if vnf.Status.DMSDeploymentID != "" {
		if err := r.DMSClient.DeleteDeployment(ctx, vnf.Status.DMSDeploymentID); err != nil {
			log.Error(err, "Failed to delete DMS deployment")
			// Continue cleanup even if DMS deletion fails
		}
	}

	// Remove Porch package
	if vnf.Status.PorchPackageRevision != "" {
		if err := r.GitOpsClient.DeletePackage(ctx, vnf.Status.PorchPackageRevision); err != nil {
			log.Error(err, "Failed to delete Porch package")
			// Continue cleanup
		}
	}

	vnf.Status.Phase = "Deleting"
	return nil
}

func (r *VNFReconciler) validateVNF(vnf *manov1alpha1.VNF) error {
	// Validate QoS parameters
	if vnf.Spec.QoS.Bandwidth < 1 || vnf.Spec.QoS.Bandwidth > 5 {
		return fmt.Errorf("bandwidth must be between 1 and 5 Mbps")
	}
	if vnf.Spec.QoS.Latency < 1 || vnf.Spec.QoS.Latency > 10 {
		return fmt.Errorf("latency must be between 1 and 10 ms")
	}

	// Validate placement
	validCloudTypes := map[string]bool{"edge": true, "regional": true, "central": true}
	if !validCloudTypes[vnf.Spec.Placement.CloudType] {
		return fmt.Errorf("invalid cloud type: %s", vnf.Spec.Placement.CloudType)
	}

	return nil
}

func (r *VNFReconciler) updateStatusWithError(ctx context.Context, vnf *manov1alpha1.VNF,
	reason string, err error) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Error(err, "VNF reconciliation failed", "reason", reason)

	vnf.Status.Phase = "Failed"
	r.setCondition(vnf, reason, metav1.ConditionFalse, "Error", err.Error())

	if statusErr := r.Status().Update(ctx, vnf); statusErr != nil {
		return ctrl.Result{}, statusErr
	}

	return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
}

func (r *VNFReconciler) setCondition(vnf *manov1alpha1.VNF, conditionType string,
	status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&vnf.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: vnf.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *VNFReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&manov1alpha1.VNF{}).
		Complete(r)
}