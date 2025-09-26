package controller

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	manov1alpha1 "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/api/v1alpha1"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/pkg/dms"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/pkg/gitops"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/pkg/translator"
)

const (
	deploymentControllerFinalizer = "deployment.mano.oran.io/finalizer"
)

// DeploymentController handles VNF deployment lifecycle with advanced reconciliation
type DeploymentController struct {
	client.Client
	Scheme            *runtime.Scheme
	logger            *slog.Logger
	DMSClient         dms.Client
	GitOpsClient      gitops.Client
	PorchTranslator   *translator.PorchTranslator
	NephioPackager    *translator.NephioPackager
	deploymentStates  map[string]*DeploymentState
	statesMutex       sync.RWMutex
	retryConfig       *RetryConfig
	healthChecker     *HealthChecker
	resourceAllocator *ResourceAllocator
	failureAnalyzer   *FailureAnalyzer
}

// DeploymentState tracks the state of a VNF deployment
type DeploymentState struct {
	VNF          *manov1alpha1.VNF
	Phase        DeploymentPhase
	LastUpdate   time.Time
	RetryCount   int
	NextRetry    time.Time
	HealthStatus HealthStatus
	Resources    *AllocatedResources
	Metrics      *DeploymentMetrics
	ErrorHistory []DeploymentError
	Dependencies []string
	Events       []DeploymentEvent
}

// DeploymentPhase represents the current phase of deployment
type DeploymentPhase string

const (
	PhaseInitializing DeploymentPhase = "Initializing"
	PhaseValidating   DeploymentPhase = "Validating"
	PhaseTranslating  DeploymentPhase = "Translating"
	PhaseAllocating   DeploymentPhase = "Allocating"
	PhaseDeploying    DeploymentPhase = "Deploying"
	PhaseRunning      DeploymentPhase = "Running"
	PhaseUpdating     DeploymentPhase = "Updating"
	PhaseFailed       DeploymentPhase = "Failed"
	PhaseTerminating  DeploymentPhase = "Terminating"
	PhaseTerminated   DeploymentPhase = "Terminated"
)

// HealthStatus represents health check status
type HealthStatus struct {
	Overall    HealthState            `json:"overall"`
	Components map[string]HealthState `json:"components"`
	LastCheck  time.Time              `json:"last_check"`
	Errors     []string               `json:"errors,omitempty"`
}

// HealthState represents individual health states
type HealthState string

const (
	HealthHealthy   HealthState = "healthy"
	HealthDegraded  HealthState = "degraded"
	HealthUnhealthy HealthState = "unhealthy"
	HealthUnknown   HealthState = "unknown"
)

// AllocatedResources tracks resource allocation
type AllocatedResources struct {
	CPUCores     int                    `json:"cpu_cores"`
	MemoryGB     int                    `json:"memory_gb"`
	StorageGB    int                    `json:"storage_gb"`
	NetworkBW    float64                `json:"network_bandwidth_mbps"`
	Clusters     []string               `json:"clusters"`
	Reservations map[string]interface{} `json:"reservations"`
}

// DeploymentMetrics tracks deployment performance metrics
type DeploymentMetrics struct {
	DeploymentTime      time.Duration      `json:"deployment_time"`
	SuccessRate         float64            `json:"success_rate"`
	AvailabilityPct     float64            `json:"availability_percent"`
	ErrorRate           float64            `json:"error_rate"`
	ResourceUtilization map[string]float64 `json:"resource_utilization"`
	QoSMetrics          map[string]float64 `json:"qos_metrics"`
}

// DeploymentError represents a deployment error with context
type DeploymentError struct {
	Timestamp   time.Time `json:"timestamp"`
	Phase       string    `json:"phase"`
	Error       string    `json:"error"`
	Code        string    `json:"code"`
	Severity    string    `json:"severity"`
	Recoverable bool      `json:"recoverable"`
	Actions     []string  `json:"suggested_actions"`
}

// DeploymentEvent represents significant deployment events
type DeploymentEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RetryConfig configures retry behavior
type RetryConfig struct {
	MaxRetries        int           `json:"max_retries"`
	InitialBackoff    time.Duration `json:"initial_backoff"`
	MaxBackoff        time.Duration `json:"max_backoff"`
	BackoffMultiplier float64       `json:"backoff_multiplier"`
	RetryableErrors   []string      `json:"retryable_errors"`
}

// HealthChecker performs health checks on deployments
type HealthChecker struct {
	client        client.Client
	dmsClient     dms.Client
	logger        *slog.Logger
	checkInterval time.Duration
}

// ResourceAllocator manages resource allocation and validation
type ResourceAllocator struct {
	client client.Client
	logger *slog.Logger
	quotas map[string]*ResourceQuota
}

// ResourceQuota represents resource limits per cluster/zone
type ResourceQuota struct {
	CPUCores      int     `json:"cpu_cores"`
	MemoryGB      int     `json:"memory_gb"`
	StorageGB     int     `json:"storage_gb"`
	NetworkBWMbps float64 `json:"network_bandwidth_mbps"`
}

// FailureAnalyzer analyzes deployment failures and suggests remediation
type FailureAnalyzer struct {
	logger      *slog.Logger
	patterns    map[string]*FailurePattern
	remediation map[string][]RemediationAction
}

// FailurePattern represents common failure patterns
type FailurePattern struct {
	Name        string   `json:"name"`
	Indicators  []string `json:"indicators"`
	Probability float64  `json:"probability"`
	Category    string   `json:"category"`
}

// RemediationAction represents automated remediation actions
type RemediationAction struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	AutoExecute bool                   `json:"auto_execute"`
}

// NewDeploymentController creates a new deployment controller
func NewDeploymentController(
	client client.Client,
	scheme *runtime.Scheme,
	dmsClient dms.Client,
	gitopsClient gitops.Client,
) *DeploymentController {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	retryConfig := &RetryConfig{
		MaxRetries:        5,
		InitialBackoff:    30 * time.Second,
		MaxBackoff:        10 * time.Minute,
		BackoffMultiplier: 2.0,
		RetryableErrors:   []string{"timeout", "connection", "temporary"},
	}

	controller := &DeploymentController{
		Client:           client,
		Scheme:           scheme,
		logger:           logger,
		DMSClient:        dmsClient,
		GitOpsClient:     gitopsClient,
		PorchTranslator:  translator.NewPorchTranslator(),
		NephioPackager:   translator.NewNephioPackager("", ""),
		deploymentStates: make(map[string]*DeploymentState),
		retryConfig:      retryConfig,
	}

	// Initialize components
	controller.healthChecker = &HealthChecker{
		client:        client,
		dmsClient:     dmsClient,
		logger:        logger,
		checkInterval: 30 * time.Second,
	}

	controller.resourceAllocator = &ResourceAllocator{
		client: client,
		logger: logger,
		quotas: make(map[string]*ResourceQuota),
	}

	controller.failureAnalyzer = &FailureAnalyzer{
		logger:      logger,
		patterns:    make(map[string]*FailurePattern),
		remediation: make(map[string][]RemediationAction),
	}

	controller.initializeFailurePatterns()
	return controller
}

//+kubebuilder:rbac:groups=mano.oran.io,resources=vnfs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mano.oran.io,resources=vnfs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=mano.oran.io,resources=vnfs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups="",resources=configmaps;secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile performs advanced VNF deployment reconciliation
func (dc *DeploymentController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	startTime := time.Now()
	dc.logger.Info("Starting VNF deployment reconciliation", "vnf", req.NamespacedName)

	// Fetch VNF instance
	vnf := &manov1alpha1.VNF{}
	if err := dc.Get(ctx, req.NamespacedName, vnf); err != nil {
		if errors.IsNotFound(err) {
			dc.logger.Info("VNF not found, cleaning up state", "vnf", req.NamespacedName)
			dc.cleanupDeploymentState(req.NamespacedName.String())
			return ctrl.Result{}, nil
		}
		dc.logger.Error("Failed to fetch VNF", "error", err)
		return ctrl.Result{}, err
	}

	// Handle deletion
	if vnf.DeletionTimestamp != nil {
		return dc.handleDeletion(ctx, vnf)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(vnf, deploymentControllerFinalizer) {
		controllerutil.AddFinalizer(vnf, deploymentControllerFinalizer)
		if err := dc.Update(ctx, vnf); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Get or create deployment state
	state := dc.getOrCreateDeploymentState(vnf)

	// Check if retry is needed
	if dc.shouldSkipRetry(state) {
		nextRetry := time.Until(state.NextRetry)
		dc.logger.Info("Skipping reconciliation due to backoff", "vnf", vnf.Name, "retry_in", nextRetry)
		return ctrl.Result{RequeueAfter: nextRetry}, nil
	}

	// Perform reconciliation based on current phase
	result, err := dc.reconcileByPhase(ctx, vnf, state)

	// Update metrics
	dc.updateDeploymentMetrics(state, time.Since(startTime), err)

	// Handle errors and retry logic
	if err != nil {
		dc.handleError(ctx, vnf, state, err)
		return dc.calculateRetryResult(state), err
	}

	// Update state and status
	state.LastUpdate = time.Now()
	dc.updateDeploymentState(vnf.Name, state)

	// Update VNF status
	if err := dc.updateVNFStatus(ctx, vnf, state); err != nil {
		dc.logger.Error("Failed to update VNF status", "error", err)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	dc.logger.Info("VNF reconciliation completed",
		"vnf", vnf.Name,
		"phase", state.Phase,
		"duration", time.Since(startTime))

	return result, nil
}

// reconcileByPhase handles reconciliation based on current deployment phase
func (dc *DeploymentController) reconcileByPhase(ctx context.Context, vnf *manov1alpha1.VNF, state *DeploymentState) (ctrl.Result, error) {
	switch state.Phase {
	case PhaseInitializing:
		return dc.handleInitializing(ctx, vnf, state)
	case PhaseValidating:
		return dc.handleValidating(ctx, vnf, state)
	case PhaseTranslating:
		return dc.handleTranslating(ctx, vnf, state)
	case PhaseAllocating:
		return dc.handleAllocating(ctx, vnf, state)
	case PhaseDeploying:
		return dc.handleDeploying(ctx, vnf, state)
	case PhaseRunning:
		return dc.handleRunning(ctx, vnf, state)
	case PhaseUpdating:
		return dc.handleUpdating(ctx, vnf, state)
	case PhaseFailed:
		return dc.handleFailed(ctx, vnf, state)
	default:
		state.Phase = PhaseInitializing
		return ctrl.Result{Requeue: true}, nil
	}
}

// handleInitializing handles the initialization phase
func (dc *DeploymentController) handleInitializing(ctx context.Context, vnf *manov1alpha1.VNF, state *DeploymentState) (ctrl.Result, error) {
	dc.logger.Info("Initializing VNF deployment", "vnf", vnf.Name)

	// Record initialization event
	dc.recordEvent(state, "Initialization", "Starting VNF deployment initialization")

	// Initialize deployment state
	state.VNF = vnf.DeepCopy()
	state.Resources = &AllocatedResources{
		Clusters:     vnf.Spec.TargetClusters,
		Reservations: make(map[string]interface{}),
	}
	state.Metrics = &DeploymentMetrics{
		ResourceUtilization: make(map[string]float64),
		QoSMetrics:          make(map[string]float64),
	}

	// Initialize VNF status if needed
	if vnf.Status.Phase == "" {
		vnf.Status.Phase = "Initializing"
		vnf.Status.ObservedGeneration = vnf.Generation
		if err := dc.Status().Update(ctx, vnf); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Move to validation phase
	state.Phase = PhaseValidating
	return ctrl.Result{Requeue: true}, nil
}

// handleValidating handles the validation phase
func (dc *DeploymentController) handleValidating(ctx context.Context, vnf *manov1alpha1.VNF, state *DeploymentState) (ctrl.Result, error) {
	dc.logger.Info("Validating VNF specification", "vnf", vnf.Name)

	// Validate VNF specification
	if err := dc.validateVNFSpec(vnf); err != nil {
		return ctrl.Result{}, fmt.Errorf("VNF validation failed: %w", err)
	}

	// Validate resource requirements
	if err := dc.validateResourceRequirements(vnf); err != nil {
		return ctrl.Result{}, fmt.Errorf("resource validation failed: %w", err)
	}

	// Validate target clusters
	if err := dc.validateTargetClusters(ctx, vnf); err != nil {
		return ctrl.Result{}, fmt.Errorf("cluster validation failed: %w", err)
	}

	dc.recordEvent(state, "Validation", "VNF specification validated successfully")

	// Move to translation phase
	state.Phase = PhaseTranslating
	return ctrl.Result{Requeue: true}, nil
}

// handleTranslating handles the translation phase
func (dc *DeploymentController) handleTranslating(ctx context.Context, vnf *manov1alpha1.VNF, state *DeploymentState) (ctrl.Result, error) {
	dc.logger.Info("Translating VNF to deployment packages", "vnf", vnf.Name)

	// Generate Porch package
	porchPkg, err := dc.PorchTranslator.TranslateVNF(vnf)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("Porch translation failed: %w", err)
	}

	// Generate Nephio package
	nephioPkg, err := dc.NephioPackager.GeneratePackage(ctx, vnf)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("Nephio package generation failed: %w", err)
	}

	// Validate packages
	if err := dc.NephioPackager.ValidatePackage(ctx, nephioPkg); err != nil {
		return ctrl.Result{}, fmt.Errorf("package validation failed: %w", err)
	}

	// Store package references in state
	state.Events = append(state.Events, DeploymentEvent{
		Timestamp:   time.Now(),
		Type:        "Translation",
		Description: "VNF translated to deployment packages",
		Metadata: map[string]interface{}{
			"porch_package":  porchPkg.Name,
			"nephio_package": nephioPkg.Metadata.Name,
		},
	})

	// Move to resource allocation phase
	state.Phase = PhaseAllocating
	return ctrl.Result{Requeue: true}, nil
}

// handleAllocating handles the resource allocation phase
func (dc *DeploymentController) handleAllocating(ctx context.Context, vnf *manov1alpha1.VNF, state *DeploymentState) (ctrl.Result, error) {
	dc.logger.Info("Allocating resources for VNF", "vnf", vnf.Name)

	// Allocate resources on target clusters
	allocation, err := dc.resourceAllocator.AllocateResources(ctx, vnf)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("resource allocation failed: %w", err)
	}

	// Update state with allocation
	state.Resources = allocation

	dc.recordEvent(state, "Allocation", "Resources allocated successfully")

	// Move to deployment phase
	state.Phase = PhaseDeploying
	return ctrl.Result{Requeue: true}, nil
}

// handleDeploying handles the deployment phase
func (dc *DeploymentController) handleDeploying(ctx context.Context, vnf *manov1alpha1.VNF, state *DeploymentState) (ctrl.Result, error) {
	dc.logger.Info("Deploying VNF via DMS", "vnf", vnf.Name)

	// Create DMS deployment
	deploymentID, err := dc.DMSClient.CreateDeployment(ctx, vnf)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("DMS deployment failed: %w", err)
	}

	// Update VNF status
	vnf.Status.DMSDeploymentID = deploymentID
	vnf.Status.Phase = "Deploying"

	dc.recordEvent(state, "Deployment", fmt.Sprintf("DMS deployment created: %s", deploymentID))

	// Start health monitoring
	go dc.startHealthMonitoring(ctx, vnf, state)

	// Move to running phase
	state.Phase = PhaseRunning
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// handleRunning handles the running phase
func (dc *DeploymentController) handleRunning(ctx context.Context, vnf *manov1alpha1.VNF, state *DeploymentState) (ctrl.Result, error) {
	dc.logger.Debug("Monitoring running VNF", "vnf", vnf.Name)

	// Check DMS deployment status
	status, err := dc.DMSClient.GetDeploymentStatus(ctx, vnf.Status.DMSDeploymentID)
	if err != nil {
		dc.logger.Error("Failed to get deployment status", "error", err)
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	// Update status based on DMS response
	switch status {
	case "Running":
		vnf.Status.Phase = "Running"
		state.HealthStatus.Overall = HealthHealthy
	case "Failed":
		state.Phase = PhaseFailed
		state.HealthStatus.Overall = HealthUnhealthy
		return ctrl.Result{Requeue: true}, nil
	case "Updating":
		state.Phase = PhaseUpdating
		return ctrl.Result{Requeue: true}, nil
	}

	// Check for spec changes (generation update)
	if vnf.Generation != vnf.Status.ObservedGeneration {
		dc.logger.Info("VNF spec changed, initiating update", "vnf", vnf.Name)
		state.Phase = PhaseUpdating
		return ctrl.Result{Requeue: true}, nil
	}

	// Perform health checks
	dc.performHealthChecks(ctx, vnf, state)

	// Update metrics
	dc.updateRuntimeMetrics(ctx, vnf, state)

	// Periodic reconciliation
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// handleUpdating handles the updating phase
func (dc *DeploymentController) handleUpdating(ctx context.Context, vnf *manov1alpha1.VNF, state *DeploymentState) (ctrl.Result, error) {
	dc.logger.Info("Updating VNF deployment", "vnf", vnf.Name)

	// Update DMS deployment
	err := dc.DMSClient.UpdateDeployment(ctx, vnf.Status.DMSDeploymentID, vnf)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("DMS update failed: %w", err)
	}

	// Update observed generation
	vnf.Status.ObservedGeneration = vnf.Generation
	vnf.Status.Phase = "Updating"

	dc.recordEvent(state, "Update", "VNF deployment update initiated")

	// Return to running phase
	state.Phase = PhaseRunning
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// handleFailed handles the failed phase
func (dc *DeploymentController) handleFailed(ctx context.Context, vnf *manov1alpha1.VNF, state *DeploymentState) (ctrl.Result, error) {
	dc.logger.Error("VNF deployment in failed state", "vnf", vnf.Name)

	// Analyze failure
	analysis := dc.failureAnalyzer.AnalyzeFailure(state)

	// Attempt automatic remediation if configured
	if len(analysis.AutoRemediationActions) > 0 {
		dc.logger.Info("Attempting automatic remediation", "vnf", vnf.Name, "actions", len(analysis.AutoRemediationActions))

		if dc.executeRemediationActions(ctx, vnf, state, analysis.AutoRemediationActions) {
			// Retry deployment
			state.Phase = PhaseValidating
			state.RetryCount++
			return ctrl.Result{Requeue: true}, nil
		}
	}

	// Check if manual intervention is needed
	if state.RetryCount >= dc.retryConfig.MaxRetries {
		vnf.Status.Phase = "Failed"
		dc.recordEvent(state, "Failure", "Maximum retry attempts exceeded, manual intervention required")
		return ctrl.Result{}, nil
	}

	// Calculate next retry time with exponential backoff
	state.NextRetry = time.Now().Add(dc.calculateBackoff(state.RetryCount))
	state.RetryCount++

	return ctrl.Result{RequeueAfter: time.Until(state.NextRetry)}, nil
}

// handleDeletion handles VNF deletion
func (dc *DeploymentController) handleDeletion(ctx context.Context, vnf *manov1alpha1.VNF) (ctrl.Result, error) {
	dc.logger.Info("Handling VNF deletion", "vnf", vnf.Name)

	if controllerutil.ContainsFinalizer(vnf, deploymentControllerFinalizer) {
		// Clean up DMS deployment
		if vnf.Status.DMSDeploymentID != "" {
			if err := dc.DMSClient.DeleteDeployment(ctx, vnf.Status.DMSDeploymentID); err != nil {
				dc.logger.Error("Failed to delete DMS deployment", "error", err)
				// Continue cleanup even if DMS deletion fails
			}
		}

		// Clean up allocated resources
		state := dc.getDeploymentState(vnf.Name)
		if state != nil && state.Resources != nil {
			dc.resourceAllocator.ReleaseResources(ctx, state.Resources)
		}

		// Clean up state
		dc.cleanupDeploymentState(vnf.Name)

		// Remove finalizer
		controllerutil.RemoveFinalizer(vnf, deploymentControllerFinalizer)
		if err := dc.Update(ctx, vnf); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// Helper methods for state management
func (dc *DeploymentController) getOrCreateDeploymentState(vnf *manov1alpha1.VNF) *DeploymentState {
	dc.statesMutex.Lock()
	defer dc.statesMutex.Unlock()

	state, exists := dc.deploymentStates[vnf.Name]
	if !exists {
		state = &DeploymentState{
			VNF:        vnf.DeepCopy(),
			Phase:      PhaseInitializing,
			LastUpdate: time.Now(),
			HealthStatus: HealthStatus{
				Overall:    HealthUnknown,
				Components: make(map[string]HealthState),
			},
			ErrorHistory: []DeploymentError{},
			Events:       []DeploymentEvent{},
		}
		dc.deploymentStates[vnf.Name] = state
	}

	return state
}

func (dc *DeploymentController) getDeploymentState(vnfName string) *DeploymentState {
	dc.statesMutex.RLock()
	defer dc.statesMutex.RUnlock()
	return dc.deploymentStates[vnfName]
}

func (dc *DeploymentController) updateDeploymentState(vnfName string, state *DeploymentState) {
	dc.statesMutex.Lock()
	defer dc.statesMutex.Unlock()
	dc.deploymentStates[vnfName] = state
}

func (dc *DeploymentController) cleanupDeploymentState(vnfName string) {
	dc.statesMutex.Lock()
	defer dc.statesMutex.Unlock()
	delete(dc.deploymentStates, vnfName)
}

// Validation methods
func (dc *DeploymentController) validateVNFSpec(vnf *manov1alpha1.VNF) error {
	if vnf.Spec.Type == "" {
		return fmt.Errorf("VNF type is required")
	}
	if vnf.Spec.Version == "" {
		return fmt.Errorf("VNF version is required")
	}
	if vnf.Spec.Image.Repository == "" {
		return fmt.Errorf("VNF image repository is required")
	}
	if vnf.Spec.Image.Tag == "" {
		return fmt.Errorf("VNF image tag is required")
	}
	return nil
}

func (dc *DeploymentController) validateResourceRequirements(vnf *manov1alpha1.VNF) error {
	if vnf.Spec.Resources.CPUCores <= 0 {
		return fmt.Errorf("CPU cores must be positive")
	}
	if vnf.Spec.Resources.MemoryGB <= 0 {
		return fmt.Errorf("memory must be positive")
	}
	return nil
}

func (dc *DeploymentController) validateTargetClusters(ctx context.Context, vnf *manov1alpha1.VNF) error {
	if len(vnf.Spec.TargetClusters) == 0 {
		return fmt.Errorf("at least one target cluster is required")
	}
	// Additional cluster validation logic would go here
	return nil
}

// Utility methods
func (dc *DeploymentController) shouldSkipRetry(state *DeploymentState) bool {
	return !state.NextRetry.IsZero() && time.Now().Before(state.NextRetry)
}

func (dc *DeploymentController) calculateBackoff(retryCount int) time.Duration {
	backoff := time.Duration(float64(dc.retryConfig.InitialBackoff) * math.Pow(dc.retryConfig.BackoffMultiplier, float64(retryCount)))
	if backoff > dc.retryConfig.MaxBackoff {
		backoff = dc.retryConfig.MaxBackoff
	}
	return backoff
}

func (dc *DeploymentController) calculateRetryResult(state *DeploymentState) ctrl.Result {
	if state.NextRetry.IsZero() {
		return ctrl.Result{RequeueAfter: 30 * time.Second}
	}
	return ctrl.Result{RequeueAfter: time.Until(state.NextRetry)}
}

func (dc *DeploymentController) recordEvent(state *DeploymentState, eventType, description string) {
	event := DeploymentEvent{
		Timestamp:   time.Now(),
		Type:        eventType,
		Description: description,
	}
	state.Events = append(state.Events, event)

	// Keep only last 50 events
	if len(state.Events) > 50 {
		state.Events = state.Events[len(state.Events)-50:]
	}
}

func (dc *DeploymentController) handleError(ctx context.Context, vnf *manov1alpha1.VNF, state *DeploymentState, err error) {
	deploymentErr := DeploymentError{
		Timestamp:   time.Now(),
		Phase:       string(state.Phase),
		Error:       err.Error(),
		Severity:    "error",
		Recoverable: dc.isRecoverableError(err),
	}

	state.ErrorHistory = append(state.ErrorHistory, deploymentErr)
	state.Phase = PhaseFailed

	dc.logger.Error("Deployment error occurred",
		"vnf", vnf.Name,
		"phase", state.Phase,
		"error", err,
		"recoverable", deploymentErr.Recoverable)
}

func (dc *DeploymentController) isRecoverableError(err error) bool {
	errorStr := err.Error()
	for _, retryableError := range dc.retryConfig.RetryableErrors {
		if contains(errorStr, retryableError) {
			return true
		}
	}
	return false
}

func (dc *DeploymentController) updateVNFStatus(ctx context.Context, vnf *manov1alpha1.VNF, state *DeploymentState) error {
	vnf.Status.Phase = string(state.Phase)
	vnf.Status.LastReconcileTime = &metav1.Time{Time: state.LastUpdate}

	if state.Resources != nil {
		vnf.Status.DeployedClusters = state.Resources.Clusters
	}

	return dc.Status().Update(ctx, vnf)
}

func (dc *DeploymentController) updateDeploymentMetrics(state *DeploymentState, duration time.Duration, err error) {
	if state.Metrics == nil {
		state.Metrics = &DeploymentMetrics{
			ResourceUtilization: make(map[string]float64),
			QoSMetrics:          make(map[string]float64),
		}
	}

	state.Metrics.DeploymentTime = duration

	if err != nil {
		state.Metrics.ErrorRate = math.Min(state.Metrics.ErrorRate+0.1, 1.0)
	} else {
		state.Metrics.ErrorRate = math.Max(state.Metrics.ErrorRate-0.05, 0.0)
	}
}

// Placeholder methods (implement these for full functionality)
func (dc *DeploymentController) startHealthMonitoring(ctx context.Context, vnf *manov1alpha1.VNF, state *DeploymentState) {
	// Implementation would start background health monitoring
}

func (dc *DeploymentController) performHealthChecks(ctx context.Context, vnf *manov1alpha1.VNF, state *DeploymentState) {
	// Implementation would perform comprehensive health checks
}

func (dc *DeploymentController) updateRuntimeMetrics(ctx context.Context, vnf *manov1alpha1.VNF, state *DeploymentState) {
	// Implementation would update runtime performance metrics
}

func (dc *DeploymentController) initializeFailurePatterns() {
	// Implementation would initialize common failure patterns
}

func (dc *DeploymentController) executeRemediationActions(ctx context.Context, vnf *manov1alpha1.VNF, state *DeploymentState, actions []RemediationAction) bool {
	// Implementation would execute automatic remediation actions
	return false
}

// ResourceAllocator methods
func (ra *ResourceAllocator) AllocateResources(ctx context.Context, vnf *manov1alpha1.VNF) (*AllocatedResources, error) {
	return &AllocatedResources{
		CPUCores:     vnf.Spec.Resources.CPUCores,
		MemoryGB:     vnf.Spec.Resources.MemoryGB,
		StorageGB:    vnf.Spec.Resources.MemoryGB * 2, // Default storage allocation
		NetworkBW:    vnf.Spec.QoS.Bandwidth,
		Clusters:     vnf.Spec.TargetClusters,
		Reservations: make(map[string]interface{}),
	}, nil
}

func (ra *ResourceAllocator) ReleaseResources(ctx context.Context, resources *AllocatedResources) error {
	// Implementation would release allocated resources
	return nil
}

// FailureAnalyzer methods
type FailureAnalysis struct {
	Patterns               []string
	RootCause              string
	Confidence             float64
	AutoRemediationActions []RemediationAction
	ManualActions          []string
}

func (fa *FailureAnalyzer) AnalyzeFailure(state *DeploymentState) *FailureAnalysis {
	return &FailureAnalysis{
		Patterns:               []string{},
		RootCause:              "Unknown failure",
		Confidence:             0.5,
		AutoRemediationActions: []RemediationAction{},
		ManualActions:          []string{"Check logs", "Verify resources"},
	}
}

// SetupWithManager sets up the controller with the Manager
func (dc *DeploymentController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&manov1alpha1.VNF{}).
		Complete(dc)
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			(len(s) > len(substr) && s[1:len(substr)+1] == substr))))
}
