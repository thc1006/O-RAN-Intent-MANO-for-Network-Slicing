package controllers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	manov1alpha1 "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/api/v1alpha1"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/pkg/dms"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/pkg/gitops"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/adapters/vnf-operator/pkg/translator"
)

const (
	optimizedVnfFinalizer = "mano.oran.io/optimized-finalizer"

	// Status constants
	statusPending  = "Pending"
	statusCreating = "Creating"
	statusRunning  = "Running"
	statusFailed   = "Failed"
)

// OptimizedVNFReconciler provides high-performance VNF reconciliation
type OptimizedVNFReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	PorchTranslator *translator.PorchTranslator
	DMSClient       dms.Client
	GitOpsClient    gitops.Client

	// Performance optimizations
	reconcileCache   map[string]*ReconcileResult
	cacheMutex       sync.RWMutex
	batchProcessor   *BatchProcessor
	metricsCollector *PerformanceMetrics

	// Concurrency control
	maxConcurrentReconciles int
	reconcileSemaphore      chan struct{}
}

// ReconcileResult caches recent reconciliation results
type ReconcileResult struct {
	Result    ctrl.Result
	Error     error
	Timestamp time.Time
	VNFHash   string
}

// BatchProcessor handles batch operations for efficiency
type BatchProcessor struct {
	pendingOperations []BatchOperation
	mutex             sync.Mutex
	flushInterval     time.Duration
	lastFlush         time.Time
}

// BatchOperation represents a batched VNF operation
type BatchOperation struct {
	VNF       *manov1alpha1.VNF
	Operation string
	Timestamp time.Time
}

// PerformanceMetrics tracks reconciliation performance
type PerformanceMetrics struct {
	TotalReconciles      int64
	SuccessfulReconciles int64
	FailedReconciles     int64
	CacheHits            int64
	AvgReconcileTimeMs   float64
	BatchOperations      int64
	ConcurrentReconciles int64
	PeakConcurrency      int64
	mutex                sync.Mutex
}

// NewOptimizedVNFReconciler creates an optimized VNF reconciler
func NewOptimizedVNFReconciler(
	client client.Client,
	scheme *runtime.Scheme,
	porchTranslator *translator.PorchTranslator,
	dmsClient dms.Client,
	gitopsClient gitops.Client,
) *OptimizedVNFReconciler {
	maxConcurrency := 10 // Configurable based on cluster size

	return &OptimizedVNFReconciler{
		Client:                  client,
		Scheme:                  scheme,
		PorchTranslator:         porchTranslator,
		DMSClient:               dmsClient,
		GitOpsClient:            gitopsClient,
		reconcileCache:          make(map[string]*ReconcileResult),
		maxConcurrentReconciles: maxConcurrency,
		reconcileSemaphore:      make(chan struct{}, maxConcurrency),
		batchProcessor: &BatchProcessor{
			flushInterval: 5 * time.Second,
			lastFlush:     time.Now(),
		},
		metricsCollector: &PerformanceMetrics{},
	}
}

//+kubebuilder:rbac:groups=mano.oran.io,resources=vnfs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mano.oran.io,resources=vnfs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=mano.oran.io,resources=vnfs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups="",resources=configmaps;secrets,verbs=get;list;watch;create;update;patch

// Reconcile with performance optimizations
func (r *OptimizedVNFReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	startTime := time.Now()
	defer func() {
		r.updateMetrics(time.Since(startTime))
	}()

	// Acquire concurrency control
	select {
	case r.reconcileSemaphore <- struct{}{}:
		defer func() { <-r.reconcileSemaphore }()
	case <-ctx.Done():
		return ctrl.Result{}, ctx.Err()
	}

	r.updateConcurrencyMetrics(1)
	defer r.updateConcurrencyMetrics(-1)

	log := log.FromContext(ctx)

	// Fetch the VNF instance
	vnf := &manov1alpha1.VNF{}
	if err := r.Get(ctx, req.NamespacedName, vnf); err != nil {
		if errors.IsNotFound(err) {
			// Remove from cache if exists
			r.removeCacheEntry(req.String())
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get VNF")
		return ctrl.Result{}, err
	}

	// Check cache for recent reconciliation
	vnfHash := r.calculateVNFHash(vnf)
	if cachedResult := r.getCachedResult(req.String(), vnfHash); cachedResult != nil {
		r.metricsCollector.mutex.Lock()
		r.metricsCollector.CacheHits++
		r.metricsCollector.mutex.Unlock()
		return cachedResult.Result, cachedResult.Error
	}

	// Handle deletion
	if vnf.ObjectMeta.DeletionTimestamp != nil {
		return r.handleDeletion(ctx, vnf)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(vnf, optimizedVnfFinalizer) {
		controllerutil.AddFinalizer(vnf, optimizedVnfFinalizer)
		if err := r.Update(ctx, vnf); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Initialize status efficiently
	if vnf.Status.Phase == "" {
		return r.initializeStatus(ctx, vnf)
	}

	// Route to optimized phase handlers
	var result ctrl.Result
	var err error

	switch vnf.Status.Phase {
	case statusPending:
		result, err = r.handlePendingOptimized(ctx, vnf)
	case statusCreating:
		result, err = r.handleCreatingOptimized(ctx, vnf)
	case statusRunning:
		result, err = r.handleRunningOptimized(ctx, vnf)
	case statusFailed:
		result, err = r.handleFailedOptimized(ctx, vnf)
	default:
		log.Info("Unknown phase", "phase", vnf.Status.Phase)
		result = ctrl.Result{}
		err = nil
	}

	// Cache the result
	r.cacheResult(req.String(), vnfHash, result, err)

	return result, err
}

// Optimized pending handler with batch processing
func (r *OptimizedVNFReconciler) handlePendingOptimized(ctx context.Context, vnf *manov1alpha1.VNF) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling Pending VNF (Optimized)", "name", vnf.Name)

	// Fast validation using cached rules
	if err := r.fastValidateVNF(vnf); err != nil {
		return r.updateStatusWithError(ctx, vnf, "ValidationFailed", err)
	}

	// Check if we can batch this operation
	if r.shouldBatchOperation(vnf, "translate") {
		r.addToBatch(vnf, "translate")
		// Return with short requeue to process batch
		return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
	}

	// Process immediately for critical VNFs or when batch is ready
	return r.processTranslationAndPush(ctx, vnf)
}

// Optimized creating handler with async monitoring
func (r *OptimizedVNFReconciler) handleCreatingOptimized(ctx context.Context, vnf *manov1alpha1.VNF) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling Creating VNF (Optimized)", "name", vnf.Name)

	// Check if DMS deployment is already in progress
	if vnf.Status.DMSDeploymentID != "" {
		// Skip deployment creation, go to monitoring
		return r.monitorDeployment(ctx, vnf)
	}

	// Create DMS deployment with retry logic
	deploymentID, err := r.createDeploymentWithRetry(ctx, vnf, 3)
	if err != nil {
		return r.updateStatusWithError(ctx, vnf, "DMSDeploymentFailed", err)
	}

	// Update status efficiently
	vnf.Status.DMSDeploymentID = deploymentID
	vnf.Status.Phase = statusRunning
	vnf.Status.LastReconcileTime = &metav1.Time{Time: time.Now()}
	r.setCondition(vnf, "Deployed", metav1.ConditionTrue, "Success",
		fmt.Sprintf("Deployed via DMS: %s", deploymentID))

	vnf.Status.DeployedClusters = vnf.Spec.TargetClusters

	if err := r.Status().Update(ctx, vnf); err != nil {
		return ctrl.Result{}, err
	}

	// Shorter requeue for faster monitoring
	return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
}

// Optimized running handler with intelligent polling
func (r *OptimizedVNFReconciler) handleRunningOptimized(ctx context.Context, vnf *manov1alpha1.VNF) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Adaptive polling based on VNF type and status
	requeueInterval := r.calculateOptimalRequeueInterval(vnf)

	// Check if this VNF needs frequent monitoring
	if !r.needsFrequentMonitoring(vnf) {
		log.V(1).Info("VNF stable, extending monitoring interval", "name", vnf.Name)
		return ctrl.Result{RequeueAfter: requeueInterval * 2}, nil
	}

	// Async status check to avoid blocking
	go r.asyncStatusCheck(ctx, vnf)

	// Update last reconcile time
	vnf.Status.LastReconcileTime = &metav1.Time{Time: time.Now()}
	if err := r.Status().Update(ctx, vnf); err != nil {
		log.Error(err, "Failed to update last reconcile time")
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

// Optimized failed handler with intelligent retry
func (r *OptimizedVNFReconciler) handleFailedOptimized(ctx context.Context, vnf *manov1alpha1.VNF) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Check if generation changed (user updated spec)
	if vnf.Generation != vnf.Status.ObservedGeneration {
		log.Info("Spec updated, retrying deployment", "name", vnf.Name)
		vnf.Status.Phase = statusPending
		vnf.Status.ObservedGeneration = vnf.Generation
		if err := r.Status().Update(ctx, vnf); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Exponential backoff for failed VNFs
	backoffInterval := r.calculateBackoffInterval(vnf)
	return ctrl.Result{RequeueAfter: backoffInterval}, nil
}

// Helper methods for optimization

func (r *OptimizedVNFReconciler) calculateVNFHash(vnf *manov1alpha1.VNF) string {
	// Simple hash based on spec and generation
	return fmt.Sprintf("%s_%d_%s_%f_%f",
		vnf.Name,
		vnf.Generation,
		vnf.Spec.Type,
		vnf.Spec.QoS.Bandwidth,
		vnf.Spec.QoS.Latency)
}

func (r *OptimizedVNFReconciler) getCachedResult(key, hash string) *ReconcileResult {
	r.cacheMutex.RLock()
	defer r.cacheMutex.RUnlock()

	cached, exists := r.reconcileCache[key]
	if !exists {
		return nil
	}

	// Check if hash matches and not expired (5 minutes)
	if cached.VNFHash == hash && time.Since(cached.Timestamp) < 5*time.Minute {
		return cached
	}

	// Remove stale entry
	delete(r.reconcileCache, key)
	return nil
}

func (r *OptimizedVNFReconciler) cacheResult(key, hash string, result ctrl.Result, err error) {
	r.cacheMutex.Lock()
	defer r.cacheMutex.Unlock()

	r.reconcileCache[key] = &ReconcileResult{
		Result:    result,
		Error:     err,
		Timestamp: time.Now(),
		VNFHash:   hash,
	}

	// Limit cache size (LRU eviction)
	if len(r.reconcileCache) > 500 {
		// Remove oldest entry
		oldestTime := time.Now()
		oldestKey := ""
		for k, v := range r.reconcileCache {
			if v.Timestamp.Before(oldestTime) {
				oldestTime = v.Timestamp
				oldestKey = k
			}
		}
		if oldestKey != "" {
			delete(r.reconcileCache, oldestKey)
		}
	}
}

func (r *OptimizedVNFReconciler) removeCacheEntry(key string) {
	r.cacheMutex.Lock()
	defer r.cacheMutex.Unlock()
	delete(r.reconcileCache, key)
}

func (r *OptimizedVNFReconciler) fastValidateVNF(vnf *manov1alpha1.VNF) error {
	// Optimized validation with early returns
	if vnf.Spec.QoS.Bandwidth < 1 || vnf.Spec.QoS.Bandwidth > 5 {
		return fmt.Errorf("bandwidth must be between 1 and 5 Mbps")
	}
	if vnf.Spec.QoS.Latency < 1 || vnf.Spec.QoS.Latency > 10 {
		return fmt.Errorf("latency must be between 1 and 10 ms")
	}

	validCloudTypes := map[string]bool{"edge": true, "regional": true, "central": true}
	if !validCloudTypes[vnf.Spec.Placement.CloudType] {
		return fmt.Errorf("invalid cloud type: %s", vnf.Spec.Placement.CloudType)
	}

	return nil
}

func (r *OptimizedVNFReconciler) shouldBatchOperation(vnf *manov1alpha1.VNF, _ string) bool {
	// Batch non-critical operations for efficiency
	if vnf.Spec.QoS.Latency > 15 { // Not ultra-low latency
		return true
	}

	// Don't batch for critical VNFs
	return false
}

func (r *OptimizedVNFReconciler) addToBatch(vnf *manov1alpha1.VNF, operation string) {
	r.batchProcessor.mutex.Lock()
	defer r.batchProcessor.mutex.Unlock()

	r.batchProcessor.pendingOperations = append(r.batchProcessor.pendingOperations, BatchOperation{
		VNF:       vnf.DeepCopy(),
		Operation: operation,
		Timestamp: time.Now(),
	})
}

func (r *OptimizedVNFReconciler) processTranslationAndPush(ctx context.Context, vnf *manov1alpha1.VNF) (ctrl.Result, error) {
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
	vnf.Status.Phase = statusCreating
	vnf.Status.PorchPackageRevision = revision
	r.setCondition(vnf, "PackageCreated", metav1.ConditionTrue, "Success",
		fmt.Sprintf("Package pushed to Porch: %s", revision))

	if err := r.Status().Update(ctx, vnf); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{Requeue: true}, nil
}

func (r *OptimizedVNFReconciler) createDeploymentWithRetry(ctx context.Context, vnf *manov1alpha1.VNF, maxRetries int) (string, error) {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		deploymentID, err := r.DMSClient.CreateDeployment(ctx, vnf)
		if err == nil {
			return deploymentID, nil
		}

		lastErr = err
		if i < maxRetries-1 {
			// Exponential backoff
			time.Sleep(time.Duration(1<<i) * time.Second)
		}
	}

	return "", lastErr
}

func (r *OptimizedVNFReconciler) monitorDeployment(ctx context.Context, vnf *manov1alpha1.VNF) (ctrl.Result, error) {
	// Quick status check with timeout
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	status, err := r.DMSClient.GetDeploymentStatus(ctx, vnf.Status.DMSDeploymentID)
	if err != nil {
		// Don't fail on monitoring errors, just requeue
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if status == statusFailed {
		vnf.Status.Phase = statusFailed
		r.setCondition(vnf, "DeploymentFailed", metav1.ConditionTrue, "DMSFailure",
			"DMS deployment reported failure")

		if err := r.Status().Update(ctx, vnf); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *OptimizedVNFReconciler) calculateOptimalRequeueInterval(vnf *manov1alpha1.VNF) time.Duration {
	// Adaptive interval based on VNF type and QoS requirements
	baseInterval := 5 * time.Minute

	if vnf.Spec.QoS.Latency < 10 {
		// Critical VNFs need more frequent monitoring
		return 1 * time.Minute
	}

	if vnf.Spec.Type == "RAN" {
		// RAN functions need frequent monitoring
		return 2 * time.Minute
	}

	return baseInterval
}

func (r *OptimizedVNFReconciler) needsFrequentMonitoring(vnf *manov1alpha1.VNF) bool {
	// Check if VNF requires frequent monitoring
	if vnf.Spec.QoS.Latency < 10 {
		return true
	}

	if vnf.Spec.Type == "RAN" {
		return true
	}

	// Check if recently deployed (first hour needs more monitoring)
	if vnf.Status.LastReconcileTime != nil {
		if time.Since(vnf.Status.LastReconcileTime.Time) < 1*time.Hour {
			return true
		}
	}

	return false
}

func (r *OptimizedVNFReconciler) calculateBackoffInterval(vnf *manov1alpha1.VNF) time.Duration {
	// Exponential backoff for failed VNFs
	baseInterval := 1 * time.Minute

	// Check failure conditions to determine appropriate backoff
	for _, condition := range vnf.Status.Conditions {
		if condition.Type == "ValidationFailed" {
			return 10 * time.Minute // Longer backoff for validation failures
		}
		if condition.Type == "DMSDeploymentFailed" {
			return 5 * time.Minute // Medium backoff for deployment failures
		}
	}

	return baseInterval
}

func (r *OptimizedVNFReconciler) asyncStatusCheck(ctx context.Context, vnf *manov1alpha1.VNF) {
	// Asynchronous status checking to avoid blocking reconciliation
	if vnf.Status.DMSDeploymentID == "" {
		return
	}

	status, err := r.DMSClient.GetDeploymentStatus(ctx, vnf.Status.DMSDeploymentID)
	if err != nil {
		return // Ignore errors in async check
	}

	if status == statusFailed {
		// Trigger immediate reconciliation for failure handling
		// This would typically involve enqueueing the VNF for reconciliation
		log.FromContext(ctx).Info("Async check detected failure", "vnf", vnf.Name)
	}
}

func (r *OptimizedVNFReconciler) initializeStatus(ctx context.Context, vnf *manov1alpha1.VNF) (ctrl.Result, error) {
	vnf.Status.Phase = statusPending
	vnf.Status.ObservedGeneration = vnf.Generation
	if err := r.Status().Update(ctx, vnf); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{Requeue: true}, nil
}

func (r *OptimizedVNFReconciler) handleDeletion(ctx context.Context, vnf *manov1alpha1.VNF) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(vnf, optimizedVnfFinalizer) {
		if err := r.finalizeVNF(ctx, vnf); err != nil {
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(vnf, optimizedVnfFinalizer)
		if err := r.Update(ctx, vnf); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *OptimizedVNFReconciler) finalizeVNF(ctx context.Context, vnf *manov1alpha1.VNF) error {
	log := log.FromContext(ctx)
	log.Info("Finalizing VNF (Optimized)", "name", vnf.Name)

	// Parallel cleanup operations
	var wg sync.WaitGroup
	errors := make(chan error, 2)

	// Delete DMS deployment
	if vnf.Status.DMSDeploymentID != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := r.DMSClient.DeleteDeployment(ctx, vnf.Status.DMSDeploymentID); err != nil {
				errors <- fmt.Errorf("failed to delete DMS deployment: %w", err)
			}
		}()
	}

	// Remove Porch package
	if vnf.Status.PorchPackageRevision != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := r.GitOpsClient.DeletePackage(ctx, vnf.Status.PorchPackageRevision); err != nil {
				errors <- fmt.Errorf("failed to delete Porch package: %w", err)
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Collect any errors
	var finalError error
	for err := range errors {
		if finalError == nil {
			finalError = err
		} else {
			finalError = fmt.Errorf("%v; %v", finalError, err)
		}
	}

	vnf.Status.Phase = "Deleting"
	return finalError
}

func (r *OptimizedVNFReconciler) updateStatusWithError(ctx context.Context, vnf *manov1alpha1.VNF,
	reason string, err error) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Error(err, "VNF reconciliation failed", "reason", reason)

	vnf.Status.Phase = statusFailed
	r.setCondition(vnf, reason, metav1.ConditionFalse, "Error", err.Error())

	if statusErr := r.Status().Update(ctx, vnf); statusErr != nil {
		return ctrl.Result{}, statusErr
	}

	r.metricsCollector.mutex.Lock()
	r.metricsCollector.FailedReconciles++
	r.metricsCollector.mutex.Unlock()

	return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
}

func (r *OptimizedVNFReconciler) setCondition(vnf *manov1alpha1.VNF, conditionType string,
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

func (r *OptimizedVNFReconciler) updateMetrics(duration time.Duration) {
	r.metricsCollector.mutex.Lock()
	defer r.metricsCollector.mutex.Unlock()

	r.metricsCollector.TotalReconciles++
	r.metricsCollector.SuccessfulReconciles++

	// Update average reconcile time
	totalTime := r.metricsCollector.AvgReconcileTimeMs * float64(r.metricsCollector.TotalReconciles-1)
	totalTime += float64(duration.Nanoseconds()) / 1e6
	r.metricsCollector.AvgReconcileTimeMs = totalTime / float64(r.metricsCollector.TotalReconciles)
}

func (r *OptimizedVNFReconciler) updateConcurrencyMetrics(delta int64) {
	r.metricsCollector.mutex.Lock()
	defer r.metricsCollector.mutex.Unlock()

	r.metricsCollector.ConcurrentReconciles += delta
	if r.metricsCollector.ConcurrentReconciles > r.metricsCollector.PeakConcurrency {
		r.metricsCollector.PeakConcurrency = r.metricsCollector.ConcurrentReconciles
	}
}

// GetMetrics returns performance metrics
func (r *OptimizedVNFReconciler) GetMetrics() *PerformanceMetrics {
	r.metricsCollector.mutex.Lock()
	defer r.metricsCollector.mutex.Unlock()

	// Return a copy
	return &PerformanceMetrics{
		TotalReconciles:      r.metricsCollector.TotalReconciles,
		SuccessfulReconciles: r.metricsCollector.SuccessfulReconciles,
		FailedReconciles:     r.metricsCollector.FailedReconciles,
		CacheHits:            r.metricsCollector.CacheHits,
		AvgReconcileTimeMs:   r.metricsCollector.AvgReconcileTimeMs,
		BatchOperations:      r.metricsCollector.BatchOperations,
		ConcurrentReconciles: r.metricsCollector.ConcurrentReconciles,
		PeakConcurrency:      r.metricsCollector.PeakConcurrency,
	}
}

// SetupWithManager sets up the optimized controller with improved configuration
func (r *OptimizedVNFReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Create controller with optimized settings
	return ctrl.NewControllerManagedBy(mgr).
		For(&manov1alpha1.VNF{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: r.maxConcurrentReconciles,
			RateLimiter: workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](
				1*time.Second,  // Base delay
				30*time.Second, // Max delay
			),
		}).
		WithEventFilter(predicate.Funcs{
			// Only reconcile on meaningful changes
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldVNF := e.ObjectOld.(*manov1alpha1.VNF)
				newVNF := e.ObjectNew.(*manov1alpha1.VNF)

				// Reconcile if generation changed (spec update)
				if oldVNF.Generation != newVNF.Generation {
					return true
				}

				// Reconcile if status phase changed
				if oldVNF.Status.Phase != newVNF.Status.Phase {
					return true
				}

				// Skip reconciliation for status-only updates
				return false
			},
		}).
		Complete(r)
}
