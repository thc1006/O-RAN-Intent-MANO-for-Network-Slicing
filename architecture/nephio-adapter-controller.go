// Package nephio provides the Nephio Adapter Controller for O-RAN Intent-Based MANO
// This controller integrates the existing orchestrator with Nephio R5+ for package management
package nephio

import (
	"context"
	"fmt"
	"time"

	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/orchestrator/pkg/placement"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/o2client"
)

// NephioAdapterReconciler reconciles NetworkSliceIntent objects
type NephioAdapterReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	PorchClient       PorchClient
	PackageGenerator  PackageGenerator
	PlacementEngine   placement.Policy
	O2Client         o2client.Client
	Repository        string
	Namespace         string
}

// NetworkSliceIntentSpec defines the desired state of NetworkSliceIntent
type NetworkSliceIntentSpec struct {
	// Original natural language intent
	Intent string `json:"intent"`

	// QoS requirements extracted from intent
	QoSProfile QoSProfile `json:"qosProfile"`

	// Network functions to be deployed
	NetworkFunctions []NetworkFunctionSpec `json:"networkFunctions"`

	// Deployment configuration
	DeploymentConfig DeploymentConfig `json:"deploymentConfig"`

	// Target clusters for multi-cluster deployment
	TargetClusters []string `json:"targetClusters,omitempty"`
}

// QoSProfile defines quality of service requirements
type QoSProfile struct {
	// Bandwidth requirement (e.g., "4.5Mbps")
	Bandwidth string `json:"bandwidth"`

	// Latency requirement (e.g., "10ms")
	Latency string `json:"latency"`

	// Reliability requirement (e.g., "99.9%")
	Reliability string `json:"reliability,omitempty"`

	// Slice type
	SliceType string `json:"sliceType"`
}

// NetworkFunctionSpec defines a network function to be deployed
type NetworkFunctionSpec struct {
	// Type of network function (gNB, AMF, SMF, UPF, etc.)
	Type string `json:"type"`

	// Placement constraints
	Placement PlacementSpec `json:"placement"`

	// Resource requirements
	Resources ResourceRequirements `json:"resources,omitempty"`

	// Configuration parameters
	Config map[string]string `json:"config,omitempty"`
}

// PlacementSpec defines placement constraints
type PlacementSpec struct {
	// Site ID for placement
	SiteID string `json:"siteId,omitempty"`

	// Cloud type preference
	CloudType string `json:"cloudType"`

	// Geographic constraints
	Region string `json:"region,omitempty"`
	Zone   string `json:"zone,omitempty"`

	// Affinity rules
	AffinityRules []AffinityRule `json:"affinityRules,omitempty"`
}

// AffinityRule defines placement affinity constraints
type AffinityRule struct {
	Type   string `json:"type"`   // "affinity" or "anti-affinity"
	Scope  string `json:"scope"`  // "host", "rack", "zone", "region"
	Target string `json:"target"` // target VNF or service
}

// ResourceRequirements defines compute resource requirements
type ResourceRequirements struct {
	CPUCores  int `json:"cpuCores,omitempty"`
	MemoryGB  int `json:"memoryGB,omitempty"`
	StorageGB int `json:"storageGB,omitempty"`
}

// DeploymentConfig defines deployment strategy
type DeploymentConfig struct {
	// Deployment strategy (rolling, blue-green, canary)
	Strategy string `json:"strategy"`

	// Timeout for deployment operations
	Timeout metav1.Duration `json:"timeout"`

	// Health check configuration
	HealthChecks []HealthCheck `json:"healthChecks,omitempty"`
}

// HealthCheck defines health checking parameters
type HealthCheck struct {
	Type     string            `json:"type"`     // "http", "tcp", "exec"
	Path     string            `json:"path,omitempty"`
	Port     int32             `json:"port,omitempty"`
	Command  []string          `json:"command,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	Interval metav1.Duration   `json:"interval"`
	Timeout  metav1.Duration   `json:"timeout"`
}

// NetworkSliceIntentStatus defines the observed state
type NetworkSliceIntentStatus struct {
	// Current phase of the slice intent
	Phase string `json:"phase,omitempty"`

	// Human-readable message indicating details about last transition
	Message string `json:"message,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Deployed network functions
	DeployedFunctions []DeployedFunction `json:"deployedFunctions,omitempty"`

	// Package revisions created
	PackageRevisions []PackageRevision `json:"packageRevisions,omitempty"`

	// Deployment metrics
	Metrics DeploymentMetrics `json:"metrics,omitempty"`
}

// DeployedFunction represents a deployed network function
type DeployedFunction struct {
	Name             string    `json:"name"`
	Type             string    `json:"type"`
	Cluster          string    `json:"cluster"`
	Namespace        string    `json:"namespace"`
	Status           string    `json:"status"`
	PackageRevision  string    `json:"packageRevision"`
	DeploymentTime   time.Time `json:"deploymentTime"`
	HealthStatus     string    `json:"healthStatus"`
}

// PackageRevision represents a Nephio package revision
type PackageRevision struct {
	Name        string    `json:"name"`
	Revision    string    `json:"revision"`
	Lifecycle   string    `json:"lifecycle"`
	Repository  string    `json:"repository"`
	CreatedTime time.Time `json:"createdTime"`
}

// DeploymentMetrics captures deployment performance metrics
type DeploymentMetrics struct {
	TotalDeploymentTime   time.Duration `json:"totalDeploymentTime"`
	PackageGenerationTime time.Duration `json:"packageGenerationTime"`
	PlacementDecisionTime time.Duration `json:"placementDecisionTime"`
	ActualDeploymentTime  time.Duration `json:"actualDeploymentTime"`
	SuccessRate           float64       `json:"successRate"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Slice Type",type=string,JSONPath=`.spec.qosProfile.sliceType`
// +kubebuilder:printcolumn:name="Functions",type=integer,JSONPath=`.status.deployedFunctions.length`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// NetworkSliceIntent represents a network slice deployment intent
type NetworkSliceIntent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkSliceIntentSpec   `json:"spec,omitempty"`
	Status NetworkSliceIntentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NetworkSliceIntentList contains a list of NetworkSliceIntent
type NetworkSliceIntentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkSliceIntent `json:"items"`
}

// Reconcile handles NetworkSliceIntent reconciliation
func (r *NephioAdapterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	startTime := time.Now()

	logger.Info("Reconciling NetworkSliceIntent", "namespace", req.Namespace, "name", req.Name)

	// Fetch the NetworkSliceIntent instance
	intent := &NetworkSliceIntent{}
	if err := r.Get(ctx, req.NamespacedName, intent); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Add finalizer for cleanup
	if intent.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(intent, "nephio-adapter.mano.oran.io/finalizer") {
			controllerutil.AddFinalizer(intent, "nephio-adapter.mano.oran.io/finalizer")
			return ctrl.Result{}, r.Update(ctx, intent)
		}
	} else {
		// Handle deletion
		return r.handleDeletion(ctx, intent)
	}

	// Update status to show processing has started
	if intent.Status.Phase == "" {
		intent.Status.Phase = "Pending"
		r.updateStatus(ctx, intent, "Starting network slice intent processing")
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	switch intent.Status.Phase {
	case "Pending":
		return r.handlePendingPhase(ctx, intent, startTime)
	case "Planning":
		return r.handlePlanningPhase(ctx, intent, startTime)
	case "Packaging":
		return r.handlePackagingPhase(ctx, intent, startTime)
	case "Deploying":
		return r.handleDeployingPhase(ctx, intent, startTime)
	case "Ready":
		return r.handleReadyPhase(ctx, intent)
	case "Failed":
		return r.handleFailedPhase(ctx, intent)
	}

	return ctrl.Result{}, nil
}

// handlePendingPhase validates the intent and starts planning
func (r *NephioAdapterReconciler) handlePendingPhase(ctx context.Context, intent *NetworkSliceIntent, startTime time.Time) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Validate QoS requirements
	if err := r.validateQoSRequirements(intent.Spec.QoSProfile); err != nil {
		intent.Status.Phase = "Failed"
		r.updateStatus(ctx, intent, fmt.Sprintf("QoS validation failed: %v", err))
		return ctrl.Result{}, nil
	}

	// Validate network function specifications
	if err := r.validateNetworkFunctions(intent.Spec.NetworkFunctions); err != nil {
		intent.Status.Phase = "Failed"
		r.updateStatus(ctx, intent, fmt.Sprintf("Network function validation failed: %v", err))
		return ctrl.Result{}, nil
	}

	// Move to planning phase
	intent.Status.Phase = "Planning"
	r.updateStatus(ctx, intent, "Validation completed, starting placement planning")

	logger.Info("NetworkSliceIntent validation completed", "intent", intent.Name, "duration", time.Since(startTime))
	return ctrl.Result{RequeueAfter: time.Second * 2}, nil
}

// handlePlanningPhase generates placement decisions
func (r *NephioAdapterReconciler) handlePlanningPhase(ctx context.Context, intent *NetworkSliceIntent, startTime time.Time) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	planningStartTime := time.Now()

	// Get available sites from O2ims
	siteNames, err := r.O2Client.GetAvailableSites(ctx)
	if err != nil {
		intent.Status.Phase = "Failed"
		r.updateStatus(ctx, intent, fmt.Sprintf("Failed to get available sites: %v", err))
		return ctrl.Result{}, nil
	}

	// Convert site names to Site objects
	sites := make([]*placement.Site, len(siteNames))
	for i, name := range siteNames {
		sites[i] = &placement.Site{
			ID:   name,
			Name: name,
			Type: "edge", // Default to edge, would be determined from actual site info
		}
	}

	// Generate placement decisions for each network function
	placements := make([]*placement.Decision, 0, len(intent.Spec.NetworkFunctions))
	for _, nfSpec := range intent.Spec.NetworkFunctions {
		nf := r.convertToNetworkFunction(nfSpec, intent.Spec.QoSProfile)
		decision, err := r.PlacementEngine.Place(nf, sites)
		if err != nil {
			intent.Status.Phase = "Failed"
			r.updateStatus(ctx, intent, fmt.Sprintf("Placement failed for %s: %v", nfSpec.Type, err))
			return ctrl.Result{}, nil
		}
		placements = append(placements, decision)
	}

	// Store placement decisions in status
	r.updatePlacementDecisions(intent, placements)

	// Record planning time
	intent.Status.Metrics.PlacementDecisionTime = time.Since(planningStartTime)

	// Move to packaging phase
	intent.Status.Phase = "Packaging"
	r.updateStatus(ctx, intent, "Placement planning completed, generating packages")

	logger.Info("Placement planning completed", "intent", intent.Name, "placements", len(placements), "duration", time.Since(planningStartTime))
	return ctrl.Result{RequeueAfter: time.Second * 2}, nil
}

// handlePackagingPhase generates Nephio packages
func (r *NephioAdapterReconciler) handlePackagingPhase(ctx context.Context, intent *NetworkSliceIntent, startTime time.Time) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	packagingStartTime := time.Now()

	// Generate Nephio packages from the intent
	packages, err := r.PackageGenerator.GeneratePackages(ctx, intent)
	if err != nil {
		intent.Status.Phase = "Failed"
		r.updateStatus(ctx, intent, fmt.Sprintf("Package generation failed: %v", err))
		return ctrl.Result{}, nil
	}

	// Create package revisions in Porch
	revisions := make([]PackageRevision, 0, len(packages))
	for _, pkg := range packages {
		revision, err := r.PorchClient.CreatePackageRevision(ctx, pkg)
		if err != nil {
			intent.Status.Phase = "Failed"
			r.updateStatus(ctx, intent, fmt.Sprintf("Failed to create package revision for %s: %v", pkg.Metadata.Name, err))
			return ctrl.Result{}, nil
		}

		revisions = append(revisions, PackageRevision{
			Name:        revision.Name,
			Revision:    revision.Spec.Revision,
			Lifecycle:   string(revision.Spec.Lifecycle),
			Repository:  revision.Spec.RepositoryName, // Changed from Repository to RepositoryName
			CreatedTime: revision.CreationTimestamp.Time,
		})

		// Propose the package
		if err := r.PorchClient.ProposePackageRevision(ctx, revision.Name); err != nil {
			logger.Error(err, "Failed to propose package revision", "package", revision.Name)
		}
	}

	// Update status with package revisions
	intent.Status.PackageRevisions = revisions
	intent.Status.Metrics.PackageGenerationTime = time.Since(packagingStartTime)

	// Move to deploying phase
	intent.Status.Phase = "Deploying"
	r.updateStatus(ctx, intent, fmt.Sprintf("Generated %d packages, starting deployment", len(packages)))

	logger.Info("Package generation completed", "intent", intent.Name, "packages", len(packages), "duration", time.Since(packagingStartTime))
	return ctrl.Result{RequeueAfter: time.Second * 5}, nil
}

// handleDeployingPhase monitors deployment progress
func (r *NephioAdapterReconciler) handleDeployingPhase(ctx context.Context, intent *NetworkSliceIntent, startTime time.Time) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	deploymentStartTime := time.Now()

	// Check deployment status via O2dms
	deploymentStatus, err := r.O2Client.GetDeploymentStatus(ctx, intent.Name)
	if err != nil {
		logger.Error(err, "Failed to get deployment status", "intent", intent.Name)
		return ctrl.Result{RequeueAfter: time.Second * 30}, nil
	}

	// Update deployed functions status
	deployedFunctions := make([]DeployedFunction, 0)
	allReady := true
	for _, status := range deploymentStatus {
		deployedFunctions = append(deployedFunctions, DeployedFunction{
			Name:            status.Name,
			Type:            status.Type,
			Cluster:         status.Cluster,
			Namespace:       status.Namespace,
			Status:          status.Status,
			PackageRevision: "", // To be filled from package revision tracking
			DeploymentTime:  time.Now(),
			HealthStatus:    status.Status, // Use Status as HealthStatus for now
		})

		if status.Status != "Ready" {
			allReady = false
		}
	}

	intent.Status.DeployedFunctions = deployedFunctions

	if allReady {
		// All functions are ready, move to ready phase
		intent.Status.Phase = "Ready"
		intent.Status.Metrics.ActualDeploymentTime = time.Since(deploymentStartTime)
		intent.Status.Metrics.TotalDeploymentTime = time.Since(startTime)
		intent.Status.Metrics.SuccessRate = 1.0

		r.updateStatus(ctx, intent, fmt.Sprintf("Network slice deployed successfully with %d functions", len(deployedFunctions)))

		logger.Info("Network slice deployment completed successfully",
			"intent", intent.Name,
			"functions", len(deployedFunctions),
			"totalDuration", intent.Status.Metrics.TotalDeploymentTime)

		return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
	}

	// Still deploying, check again later
	r.updateStatus(ctx, intent, fmt.Sprintf("Deployment in progress: %d/%d functions ready", r.countReadyFunctions(deployedFunctions), len(deployedFunctions)))
	return ctrl.Result{RequeueAfter: time.Second * 30}, nil
}

// handleReadyPhase monitors the deployed network slice
func (r *NephioAdapterReconciler) handleReadyPhase(ctx context.Context, intent *NetworkSliceIntent) (ctrl.Result, error) {
	// Monitor health of deployed functions
	// Check for any configuration updates needed
	// Handle scaling events if required

	// Periodic health check
	return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
}

// handleFailedPhase handles cleanup and retry logic
func (r *NephioAdapterReconciler) handleFailedPhase(ctx context.Context, intent *NetworkSliceIntent) (ctrl.Result, error) {
	// Implement retry logic based on failure reason
	// Clean up partially deployed resources
	// Alert operators

	return ctrl.Result{}, nil
}

// handleDeletion handles NetworkSliceIntent deletion
func (r *NephioAdapterReconciler) handleDeletion(ctx context.Context, intent *NetworkSliceIntent) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Clean up deployed resources via O2dms
	if err := r.O2Client.DeleteDeployment(ctx, intent.Name); err != nil {
		logger.Error(err, "Failed to delete deployment via O2dms", "intent", intent.Name)
		return ctrl.Result{RequeueAfter: time.Second * 30}, nil
	}

	// Clean up package revisions
	for _, revision := range intent.Status.PackageRevisions {
		if err := r.PorchClient.DeletePackageRevision(ctx, revision.Name); err != nil {
			logger.Error(err, "Failed to delete package revision", "revision", revision.Name)
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(intent, "nephio-adapter.mano.oran.io/finalizer")
	return ctrl.Result{}, r.Update(ctx, intent)
}

// Helper methods

func (r *NephioAdapterReconciler) validateQoSRequirements(qos QoSProfile) error {
	// Implement QoS validation logic
	if qos.Bandwidth == "" || qos.Latency == "" {
		return fmt.Errorf("bandwidth and latency are required")
	}
	return nil
}

func (r *NephioAdapterReconciler) validateNetworkFunctions(nfs []NetworkFunctionSpec) error {
	// Implement network function validation logic
	if len(nfs) == 0 {
		return fmt.Errorf("at least one network function must be specified")
	}
	return nil
}

func (r *NephioAdapterReconciler) convertToNetworkFunction(nfSpec NetworkFunctionSpec, qos QoSProfile) *placement.NetworkFunction {
	// Convert NetworkFunctionSpec to placement.NetworkFunction
	return &placement.NetworkFunction{
		ID:   fmt.Sprintf("%s-%s", nfSpec.Type, nfSpec.Placement.SiteID),
		Type: nfSpec.Type,
		Requirements: placement.ResourceRequirements{
			MinCPUCores:      nfSpec.Resources.CPUCores,
			MinMemoryGB:      nfSpec.Resources.MemoryGB,
			MinStorageGB:     nfSpec.Resources.StorageGB,
			MinBandwidthMbps: 100, // Default bandwidth requirement
		},
		QoSRequirements: placement.QoSRequirements{
			MaxLatencyMs:      10, // Parse from qos.Latency
			MinThroughputMbps: 5,  // Parse from qos.Bandwidth
		},
	}
}

func (r *NephioAdapterReconciler) updatePlacementDecisions(intent *NetworkSliceIntent, placements []*placement.Decision) {
	// Update intent with placement decisions
	for i, placement := range placements {
		if i < len(intent.Spec.NetworkFunctions) {
			intent.Spec.NetworkFunctions[i].Placement.SiteID = placement.Site.ID
		}
	}
}

func (r *NephioAdapterReconciler) countReadyFunctions(functions []DeployedFunction) int {
	ready := 0
	for _, fn := range functions {
		if fn.Status == "Ready" {
			ready++
		}
	}
	return ready
}

func (r *NephioAdapterReconciler) updateStatus(ctx context.Context, intent *NetworkSliceIntent, message string) {
	intent.Status.Message = message
	r.Status().Update(ctx, intent)
}

// SetupWithManager sets up the controller with the Manager
func (r *NephioAdapterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&NetworkSliceIntent{}).
		Complete(r)
}

// PorchClient interface for Porch API operations
type PorchClient interface {
	CreatePackageRevision(ctx context.Context, pkg *Package) (*porchapi.PackageRevision, error)
	ProposePackageRevision(ctx context.Context, name string) error
	PublishPackageRevision(ctx context.Context, name string) error
	DeletePackageRevision(ctx context.Context, name string) error
}

// PackageGenerator interface for generating Nephio packages
type PackageGenerator interface {
	GeneratePackages(ctx context.Context, intent *NetworkSliceIntent) ([]*Package, error)
}

// Package type is already defined in package-generator.go

// SchemeBuilder is used to add go types to the GroupVersionKind scheme
var (
	SchemeGroupVersion = schema.GroupVersion{Group: "mano.o-ran.org", Version: "v1alpha1"}
	SchemeBuilder      = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme        = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&NetworkSliceIntent{},
		&NetworkSliceIntentList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

func init() {
	// Types are already registered via addKnownTypes
}