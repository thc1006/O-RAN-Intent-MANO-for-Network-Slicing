package configsync

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// ConfigSyncManager manages Config Sync configurations for Nephio GitOps
type ConfigSyncManager struct {
	client     dynamic.Interface
	restConfig *rest.Config
	namespace  string
}

// RootSync represents a Config Sync RootSync configuration
type RootSync struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Spec        RootSyncSpec      `json:"spec"`
	Status      RootSyncStatus    `json:"status,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// RootSyncSpec defines the specification for a RootSync
type RootSyncSpec struct {
	// SourceFormat specifies the source format (hierarchy, unstructured)
	// +kubebuilder:validation:Enum=hierarchy;unstructured
	SourceFormat string `json:"sourceFormat"`

	// Git specifies Git repository configuration
	Git *GitSyncSpec `json:"git,omitempty"`

	// OCI specifies OCI repository configuration
	OCI *OCISyncSpec `json:"oci,omitempty"`

	// Override specifies override configuration
	Override *OverrideSpec `json:"override,omitempty"`

	// SafeOverride specifies safe override configuration
	SafeOverride *SafeOverrideSpec `json:"safeOverride,omitempty"`
}

// GitSyncSpec defines Git synchronization configuration
type GitSyncSpec struct {
	// Repo specifies the Git repository URL
	Repo string `json:"repo"`

	// Branch specifies the Git branch
	Branch string `json:"branch,omitempty"`

	// Revision specifies the Git revision
	Revision string `json:"revision,omitempty"`

	// Dir specifies the directory in the repository
	Dir string `json:"dir,omitempty"`

	// Auth specifies authentication method
	// +kubebuilder:validation:Enum=none;ssh;cookiefile;token;gcenode;gcpserviceaccount
	Auth string `json:"auth,omitempty"`

	// SecretRef specifies the secret reference for authentication
	SecretRef *SecretReference `json:"secretRef,omitempty"`

	// GCPServiceAccountEmail specifies GCP service account email
	GCPServiceAccountEmail string `json:"gcpServiceAccountEmail,omitempty"`

	// Proxy specifies proxy configuration
	Proxy *ProxySpec `json:"proxy,omitempty"`

	// NoSSLVerify disables SSL verification
	NoSSLVerify bool `json:"noSSLVerify,omitempty"`

	// CACertSecretRef specifies CA certificate secret reference
	CACertSecretRef *SecretReference `json:"caCertSecretRef,omitempty"`
}

// OCISyncSpec defines OCI synchronization configuration
type OCISyncSpec struct {
	// Image specifies the OCI image
	Image string `json:"image"`

	// Dir specifies the directory in the image
	Dir string `json:"dir,omitempty"`

	// Auth specifies authentication method
	// +kubebuilder:validation:Enum=none;gcenode;gcpserviceaccount;k8sserviceaccount
	Auth string `json:"auth,omitempty"`

	// GCPServiceAccountEmail specifies GCP service account email
	GCPServiceAccountEmail string `json:"gcpServiceAccountEmail,omitempty"`
}

// OverrideSpec defines override configuration
type OverrideSpec struct {
	// StatusMode specifies status mode
	// +kubebuilder:validation:Enum=enabled;disabled
	StatusMode string `json:"statusMode,omitempty"`

	// ReconcileTimeout specifies reconcile timeout
	ReconcileTimeout *string `json:"reconcileTimeout,omitempty"`

	// APIServerTimeout specifies API server timeout
	APIServerTimeout *string `json:"apiServerTimeout,omitempty"`

	// Resources specifies resource overrides
	Resources []ResourceOverride `json:"resources,omitempty"`

	// GitSyncDepth specifies Git sync depth
	GitSyncDepth *int64 `json:"gitSyncDepth,omitempty"`

	// EnableShellInRendering enables shell in rendering
	EnableShellInRendering bool `json:"enableShellInRendering,omitempty"`
}

// SafeOverrideSpec defines safe override configuration
type SafeOverrideSpec struct {
	// ClusterRole specifies cluster role name
	ClusterRole string `json:"clusterRole,omitempty"`

	// ClusterRoleBinding specifies cluster role binding name
	ClusterRoleBinding string `json:"clusterRoleBinding,omitempty"`
}

// ResourceOverride defines resource override
type ResourceOverride struct {
	// Group specifies the API group
	Group string `json:"group,omitempty"`

	// Version specifies the API version
	Version string `json:"version,omitempty"`

	// Kind specifies the resource kind
	Kind string `json:"kind"`

	// ContainerResources specifies container resource limits
	ContainerResources *ContainerResourcesSpec `json:"containerResources,omitempty"`
}

// ContainerResourcesSpec defines container resource specifications
type ContainerResourcesSpec struct {
	// CPURequest specifies CPU request
	CPURequest string `json:"cpuRequest,omitempty"`

	// CPULimit specifies CPU limit
	CPULimit string `json:"cpuLimit,omitempty"`

	// MemoryRequest specifies memory request
	MemoryRequest string `json:"memoryRequest,omitempty"`

	// MemoryLimit specifies memory limit
	MemoryLimit string `json:"memoryLimit,omitempty"`
}

// SecretReference defines a secret reference
type SecretReference struct {
	// Name specifies the secret name
	Name string `json:"name"`

	// Key specifies the secret key
	Key string `json:"key,omitempty"`
}

// ProxySpec defines proxy configuration
type ProxySpec struct {
	// HTTPProxy specifies HTTP proxy
	HTTPProxy string `json:"httpProxy,omitempty"`

	// HTTPSProxy specifies HTTPS proxy
	HTTPSProxy string `json:"httpsProxy,omitempty"`

	// NoProxy specifies no proxy
	NoProxy string `json:"noProxy,omitempty"`
}

// RootSyncStatus defines the status of a RootSync
type RootSyncStatus struct {
	// Conditions represent the latest available observations
	Conditions []RootSyncCondition `json:"conditions,omitempty"`

	// Sync represents sync status
	Sync *SyncStatus `json:"sync,omitempty"`

	// Rendering represents rendering status
	Rendering *RenderingStatus `json:"rendering,omitempty"`

	// Source represents source status
	Source *SourceStatus `json:"source,omitempty"`

	// ObservedGeneration represents observed generation
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// LastSyncedCommit represents the last synced commit
	LastSyncedCommit string `json:"lastSyncedCommit,omitempty"`
}

// RootSyncCondition defines a condition for RootSync
type RootSyncCondition struct {
	// Type specifies the condition type
	Type string `json:"type"`

	// Status specifies the condition status
	// +kubebuilder:validation:Enum=True;False;Unknown
	Status string `json:"status"`

	// LastUpdateTime specifies the last update time
	LastUpdateTime *time.Time `json:"lastUpdateTime,omitempty"`

	// LastTransitionTime specifies the last transition time
	LastTransitionTime *time.Time `json:"lastTransitionTime,omitempty"`

	// Reason specifies the condition reason
	Reason string `json:"reason,omitempty"`

	// Message specifies the condition message
	Message string `json:"message,omitempty"`

	// Commit specifies the Git commit
	Commit string `json:"commit,omitempty"`

	// ErrorSourceRefs specifies error source references
	ErrorSourceRefs []ErrorSource `json:"errorSourceRefs,omitempty"`

	// ErrorSummary specifies error summary
	ErrorSummary *ErrorSummary `json:"errorSummary,omitempty"`
}

// SyncStatus represents synchronization status
type SyncStatus struct {
	// SyncToken specifies the sync token
	SyncToken string `json:"syncToken,omitempty"`

	// LastUpdate specifies the last update time
	LastUpdate *time.Time `json:"lastUpdate,omitempty"`

	// Errors specifies sync errors
	Errors []ConfigSyncError `json:"errors,omitempty"`
}

// RenderingStatus represents rendering status
type RenderingStatus struct {
	// LastUpdate specifies the last update time
	LastUpdate *time.Time `json:"lastUpdate,omitempty"`

	// Errors specifies rendering errors
	Errors []ConfigSyncError `json:"errors,omitempty"`

	// Message specifies rendering message
	Message string `json:"message,omitempty"`
}

// SourceStatus represents source status
type SourceStatus struct {
	// LastUpdate specifies the last update time
	LastUpdate *time.Time `json:"lastUpdate,omitempty"`

	// Errors specifies source errors
	Errors []ConfigSyncError `json:"errors,omitempty"`

	// Commit specifies the Git commit
	Commit string `json:"commit,omitempty"`

	// Identifiers specifies resource identifiers
	Identifiers []ResourceIdentifier `json:"identifiers,omitempty"`
}

// ConfigSyncError represents a Config Sync error
type ConfigSyncError struct {
	// Code specifies the error code
	Code string `json:"code"`

	// Message specifies the error message
	Message string `json:"message"`

	// Resources specifies affected resources
	Resources []ResourceReference `json:"resources,omitempty"`
}

// ErrorSource represents an error source
type ErrorSource struct {
	// Source specifies the source
	Source string `json:"source"`

	// Errors specifies the errors
	Errors []ConfigSyncError `json:"errors"`
}

// ErrorSummary represents an error summary
type ErrorSummary struct {
	// TotalCount specifies the total error count
	TotalCount int32 `json:"totalCount"`

	// Truncated specifies if errors are truncated
	Truncated bool `json:"truncated,omitempty"`

	// ErrorCountAfterTruncation specifies error count after truncation
	ErrorCountAfterTruncation int32 `json:"errorCountAfterTruncation,omitempty"`
}

// ResourceReference represents a resource reference
type ResourceReference struct {
	// SourcePath specifies the source path
	SourcePath string `json:"sourcePath,omitempty"`

	// Group specifies the API group
	Group string `json:"group,omitempty"`

	// Version specifies the API version
	Version string `json:"version,omitempty"`

	// Kind specifies the resource kind
	Kind string `json:"kind,omitempty"`

	// Name specifies the resource name
	Name string `json:"name,omitempty"`

	// Namespace specifies the resource namespace
	Namespace string `json:"namespace,omitempty"`
}

// ResourceIdentifier represents a resource identifier
type ResourceIdentifier struct {
	// Group specifies the API group
	Group string `json:"group,omitempty"`

	// Version specifies the API version
	Version string `json:"version,omitempty"`

	// Kind specifies the resource kind
	Kind string `json:"kind,omitempty"`

	// Name specifies the resource name
	Name string `json:"name,omitempty"`

	// Namespace specifies the resource namespace
	Namespace string `json:"namespace,omitempty"`
}

// RepoSync represents a Config Sync RepoSync configuration
type RepoSync struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Spec        RepoSyncSpec      `json:"spec"`
	Status      RepoSyncStatus    `json:"status,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// RepoSyncSpec defines the specification for a RepoSync
type RepoSyncSpec struct {
	// SourceFormat specifies the source format
	// +kubebuilder:validation:Enum=hierarchy;unstructured
	SourceFormat string `json:"sourceFormat"`

	// Git specifies Git repository configuration
	Git *GitSyncSpec `json:"git,omitempty"`

	// OCI specifies OCI repository configuration
	OCI *OCISyncSpec `json:"oci,omitempty"`

	// Override specifies override configuration
	Override *OverrideSpec `json:"override,omitempty"`
}

// RepoSyncStatus defines the status of a RepoSync
type RepoSyncStatus struct {
	// Conditions represent the latest available observations
	Conditions []RootSyncCondition `json:"conditions,omitempty"`

	// Sync represents sync status
	Sync *SyncStatus `json:"sync,omitempty"`

	// Rendering represents rendering status
	Rendering *RenderingStatus `json:"rendering,omitempty"`

	// Source represents source status
	Source *SourceStatus `json:"source,omitempty"`

	// ObservedGeneration represents observed generation
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// LastSyncedCommit represents the last synced commit
	LastSyncedCommit string `json:"lastSyncedCommit,omitempty"`
}

// Config Sync GVRs
var (
	RootSyncGVR = schema.GroupVersionResource{
		Group:    "configsync.gke.io",
		Version:  "v1beta1",
		Resource: "rootsyncs",
	}

	RepoSyncGVR = schema.GroupVersionResource{
		Group:    "configsync.gke.io",
		Version:  "v1beta1",
		Resource: "reposyncs",
	}

	ConfigManagementGVR = schema.GroupVersionResource{
		Group:    "configmanagement.gke.io",
		Version:  "v1",
		Resource: "configmanagements",
	}
)

// NewConfigSyncManager creates a new Config Sync manager
func NewConfigSyncManager(config *rest.Config, namespace string) (*ConfigSyncManager, error) {
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &ConfigSyncManager{
		client:     dynamicClient,
		restConfig: config,
		namespace:  namespace,
	}, nil
}

// CreateRootSync creates a new RootSync configuration
func (csm *ConfigSyncManager) CreateRootSync(ctx context.Context, rootSync *RootSync) error {
	// Create RootSync manifest
	rootSyncManifest := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "configsync.gke.io/v1beta1",
			"kind":       "RootSync",
			"metadata": map[string]interface{}{
				"name":        rootSync.Name,
				"namespace":   rootSync.Namespace,
				"labels":      rootSync.Labels,
				"annotations": rootSync.Annotations,
			},
			"spec": csm.buildRootSyncSpec(rootSync.Spec),
		},
	}

	// Create RootSync
	_, err := csm.client.Resource(RootSyncGVR).Namespace(rootSync.Namespace).
		Create(ctx, rootSyncManifest, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create RootSync %s: %w", rootSync.Name, err)
	}

	return nil
}

// GetRootSync retrieves a RootSync configuration
func (csm *ConfigSyncManager) GetRootSync(ctx context.Context, name, namespace string) (*RootSync, error) {
	rootSyncManifest, err := csm.client.Resource(RootSyncGVR).Namespace(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get RootSync %s: %w", name, err)
	}

	return csm.parseRootSync(rootSyncManifest)
}

// ListRootSyncs lists all RootSync configurations
func (csm *ConfigSyncManager) ListRootSyncs(ctx context.Context, namespace string) ([]*RootSync, error) {
	rootSyncList, err := csm.client.Resource(RootSyncGVR).Namespace(namespace).
		List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list RootSyncs: %w", err)
	}

	var rootSyncs []*RootSync
	for _, item := range rootSyncList.Items {
		rootSync, err := csm.parseRootSync(&item)
		if err != nil {
			return nil, fmt.Errorf("failed to parse RootSync: %w", err)
		}
		rootSyncs = append(rootSyncs, rootSync)
	}

	return rootSyncs, nil
}

// UpdateRootSync updates a RootSync configuration
func (csm *ConfigSyncManager) UpdateRootSync(ctx context.Context, rootSync *RootSync) error {
	// Get existing RootSync
	existingRootSync, err := csm.client.Resource(RootSyncGVR).Namespace(rootSync.Namespace).
		Get(ctx, rootSync.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get existing RootSync %s: %w", rootSync.Name, err)
	}

	// Update spec
	spec := csm.buildRootSyncSpec(rootSync.Spec)
	if err := unstructured.SetNestedMap(existingRootSync.Object, spec, "spec"); err != nil {
		return fmt.Errorf("failed to set RootSync spec: %w", err)
	}

	// Update labels and annotations
	if rootSync.Labels != nil {
		if err := unstructured.SetNestedStringMap(existingRootSync.Object, rootSync.Labels, "metadata", "labels"); err != nil {
			return fmt.Errorf("failed to set RootSync labels: %w", err)
		}
	}

	if rootSync.Annotations != nil {
		if err := unstructured.SetNestedStringMap(existingRootSync.Object, rootSync.Annotations, "metadata", "annotations"); err != nil {
			return fmt.Errorf("failed to set RootSync annotations: %w", err)
		}
	}

	// Update RootSync
	_, err = csm.client.Resource(RootSyncGVR).Namespace(rootSync.Namespace).
		Update(ctx, existingRootSync, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update RootSync %s: %w", rootSync.Name, err)
	}

	return nil
}

// DeleteRootSync deletes a RootSync configuration
func (csm *ConfigSyncManager) DeleteRootSync(ctx context.Context, name, namespace string) error {
	err := csm.client.Resource(RootSyncGVR).Namespace(namespace).
		Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete RootSync %s: %w", name, err)
	}

	return nil
}

// CreateRepoSync creates a new RepoSync configuration
func (csm *ConfigSyncManager) CreateRepoSync(ctx context.Context, repoSync *RepoSync) error {
	// Create RepoSync manifest
	repoSyncManifest := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "configsync.gke.io/v1beta1",
			"kind":       "RepoSync",
			"metadata": map[string]interface{}{
				"name":        repoSync.Name,
				"namespace":   repoSync.Namespace,
				"labels":      repoSync.Labels,
				"annotations": repoSync.Annotations,
			},
			"spec": csm.buildRepoSyncSpec(repoSync.Spec),
		},
	}

	// Create RepoSync
	_, err := csm.client.Resource(RepoSyncGVR).Namespace(repoSync.Namespace).
		Create(ctx, repoSyncManifest, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create RepoSync %s: %w", repoSync.Name, err)
	}

	return nil
}

// CreateNephioRootSync creates a Nephio-specific RootSync for O-RAN workloads
func (csm *ConfigSyncManager) CreateNephioRootSync(ctx context.Context, clusterName, repoURL, branch, directory string) error {
	rootSync := &RootSync{
		Name:      fmt.Sprintf("nephio-%s", clusterName),
		Namespace: "config-management-system",
		Labels: map[string]string{
			"nephio.io/component":    "configsync",
			"nephio.io/cluster":      clusterName,
			"oran.io/managed-by":     "nephio-generator",
		},
		Annotations: map[string]string{
			"config.kubernetes.io/local-config": "true",
			"nephio.io/cluster":                  clusterName,
			"nephio.io/sync-type":                "oran-workload",
		},
		Spec: RootSyncSpec{
			SourceFormat: "unstructured",
			Git: &GitSyncSpec{
				Repo:   repoURL,
				Branch: branch,
				Dir:    filepath.Join(directory, clusterName, "config"),
				Auth:   "none", // For production, use proper authentication
			},
			Override: &OverrideSpec{
				StatusMode:       "enabled",
				ReconcileTimeout: stringPtr("5m"),
				APIServerTimeout: stringPtr("15s"),
				Resources: []ResourceOverride{
					{
						Group:   "apps",
						Version: "v1",
						Kind:    "Deployment",
						ContainerResources: &ContainerResourcesSpec{
							CPURequest:    "100m",
							CPULimit:      "1",
							MemoryRequest: "128Mi",
							MemoryLimit:   "1Gi",
						},
					},
				},
			},
		},
	}

	return csm.CreateRootSync(ctx, rootSync)
}

// CreateNephioRepoSync creates a Nephio-specific RepoSync for namespace-scoped workloads
func (csm *ConfigSyncManager) CreateNephioRepoSync(ctx context.Context, namespace, repoURL, branch, directory string) error {
	repoSync := &RepoSync{
		Name:      fmt.Sprintf("nephio-%s", namespace),
		Namespace: namespace,
		Labels: map[string]string{
			"nephio.io/component":    "configsync",
			"nephio.io/namespace":    namespace,
			"oran.io/managed-by":     "nephio-generator",
		},
		Annotations: map[string]string{
			"config.kubernetes.io/local-config": "true",
			"nephio.io/namespace":                namespace,
			"nephio.io/sync-type":                "oran-namespace-workload",
		},
		Spec: RepoSyncSpec{
			SourceFormat: "unstructured",
			Git: &GitSyncSpec{
				Repo:   repoURL,
				Branch: branch,
				Dir:    filepath.Join(directory, "namespaces", namespace),
				Auth:   "none", // For production, use proper authentication
			},
			Override: &OverrideSpec{
				StatusMode:       "enabled",
				ReconcileTimeout: stringPtr("3m"),
				APIServerTimeout: stringPtr("10s"),
			},
		},
	}

	return csm.CreateRepoSync(ctx, repoSync)
}

// GetSyncStatus gets the synchronization status for a RootSync or RepoSync
func (csm *ConfigSyncManager) GetSyncStatus(ctx context.Context, name, namespace string, syncType string) (*SyncStatusReport, error) {
	var obj *unstructured.Unstructured
	var err error

	switch syncType {
	case "RootSync":
		obj, err = csm.client.Resource(RootSyncGVR).Namespace(namespace).
			Get(ctx, name, metav1.GetOptions{})
	case "RepoSync":
		obj, err = csm.client.Resource(RepoSyncGVR).Namespace(namespace).
			Get(ctx, name, metav1.GetOptions{})
	default:
		return nil, fmt.Errorf("unsupported sync type: %s", syncType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get %s %s: %w", syncType, name, err)
	}

	return csm.parseSyncStatus(obj)
}

// SyncStatusReport represents a comprehensive sync status report
type SyncStatusReport struct {
	Name               string                `json:"name"`
	Namespace          string                `json:"namespace"`
	Type               string                `json:"type"`
	Overall            string                `json:"overall"`
	LastSyncedCommit   string                `json:"lastSyncedCommit,omitempty"`
	LastUpdateTime     *time.Time            `json:"lastUpdateTime,omitempty"`
	Conditions         []RootSyncCondition   `json:"conditions,omitempty"`
	SyncErrors         []ConfigSyncError     `json:"syncErrors,omitempty"`
	RenderingErrors    []ConfigSyncError     `json:"renderingErrors,omitempty"`
	SourceErrors       []ConfigSyncError     `json:"sourceErrors,omitempty"`
	ResourceCount      int32                 `json:"resourceCount"`
	SyncedResources    []ResourceIdentifier  `json:"syncedResources,omitempty"`
	ErrorSummary       *ErrorSummary         `json:"errorSummary,omitempty"`
}

// WaitForSync waits for a sync to complete
func (csm *ConfigSyncManager) WaitForSync(ctx context.Context, name, namespace, syncType string, timeout time.Duration) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("timeout waiting for %s %s to sync", syncType, name)
		case <-ticker.C:
			status, err := csm.GetSyncStatus(timeoutCtx, name, namespace, syncType)
			if err != nil {
				return fmt.Errorf("failed to get sync status: %w", err)
			}

			if status.Overall == "Synced" {
				return nil
			}

			if status.Overall == "Error" && len(status.SyncErrors) > 0 {
				return fmt.Errorf("sync failed with errors: %v", status.SyncErrors)
			}
		}
	}
}

// CreateMultiClusterSync creates sync configurations for multiple clusters
func (csm *ConfigSyncManager) CreateMultiClusterSync(ctx context.Context, clusters []string, repoURL, branch, baseDirectory string) error {
	for _, cluster := range clusters {
		clusterDir := filepath.Join(baseDirectory, cluster)

		if err := csm.CreateNephioRootSync(ctx, cluster, repoURL, branch, clusterDir); err != nil {
			return fmt.Errorf("failed to create RootSync for cluster %s: %w", cluster, err)
		}
	}

	return nil
}

// Helper functions

// buildRootSyncSpec builds RootSync specification
func (csm *ConfigSyncManager) buildRootSyncSpec(spec RootSyncSpec) map[string]interface{} {
	result := map[string]interface{}{
		"sourceFormat": spec.SourceFormat,
	}

	if spec.Git != nil {
		gitSpec := map[string]interface{}{
			"repo": spec.Git.Repo,
		}

		if spec.Git.Branch != "" {
			gitSpec["branch"] = spec.Git.Branch
		}

		if spec.Git.Revision != "" {
			gitSpec["revision"] = spec.Git.Revision
		}

		if spec.Git.Dir != "" {
			gitSpec["dir"] = spec.Git.Dir
		}

		if spec.Git.Auth != "" {
			gitSpec["auth"] = spec.Git.Auth
		}

		if spec.Git.SecretRef != nil {
			secretRef := map[string]interface{}{
				"name": spec.Git.SecretRef.Name,
			}
			if spec.Git.SecretRef.Key != "" {
				secretRef["key"] = spec.Git.SecretRef.Key
			}
			gitSpec["secretRef"] = secretRef
		}

		if spec.Git.GCPServiceAccountEmail != "" {
			gitSpec["gcpServiceAccountEmail"] = spec.Git.GCPServiceAccountEmail
		}

		if spec.Git.NoSSLVerify {
			gitSpec["noSSLVerify"] = true
		}

		if spec.Git.CACertSecretRef != nil {
			caCertRef := map[string]interface{}{
				"name": spec.Git.CACertSecretRef.Name,
			}
			if spec.Git.CACertSecretRef.Key != "" {
				caCertRef["key"] = spec.Git.CACertSecretRef.Key
			}
			gitSpec["caCertSecretRef"] = caCertRef
		}

		result["git"] = gitSpec
	}

	if spec.OCI != nil {
		ociSpec := map[string]interface{}{
			"image": spec.OCI.Image,
		}

		if spec.OCI.Dir != "" {
			ociSpec["dir"] = spec.OCI.Dir
		}

		if spec.OCI.Auth != "" {
			ociSpec["auth"] = spec.OCI.Auth
		}

		if spec.OCI.GCPServiceAccountEmail != "" {
			ociSpec["gcpServiceAccountEmail"] = spec.OCI.GCPServiceAccountEmail
		}

		result["oci"] = ociSpec
	}

	if spec.Override != nil {
		overrideSpec := map[string]interface{}{}

		if spec.Override.StatusMode != "" {
			overrideSpec["statusMode"] = spec.Override.StatusMode
		}

		if spec.Override.ReconcileTimeout != nil {
			overrideSpec["reconcileTimeout"] = *spec.Override.ReconcileTimeout
		}

		if spec.Override.APIServerTimeout != nil {
			overrideSpec["apiServerTimeout"] = *spec.Override.APIServerTimeout
		}

		if len(spec.Override.Resources) > 0 {
			resources := make([]map[string]interface{}, len(spec.Override.Resources))
			for i, resource := range spec.Override.Resources {
				resourceSpec := map[string]interface{}{
					"kind": resource.Kind,
				}

				if resource.Group != "" {
					resourceSpec["group"] = resource.Group
				}

				if resource.Version != "" {
					resourceSpec["version"] = resource.Version
				}

				if resource.ContainerResources != nil {
					containerRes := map[string]interface{}{}

					if resource.ContainerResources.CPURequest != "" {
						containerRes["cpuRequest"] = resource.ContainerResources.CPURequest
					}

					if resource.ContainerResources.CPULimit != "" {
						containerRes["cpuLimit"] = resource.ContainerResources.CPULimit
					}

					if resource.ContainerResources.MemoryRequest != "" {
						containerRes["memoryRequest"] = resource.ContainerResources.MemoryRequest
					}

					if resource.ContainerResources.MemoryLimit != "" {
						containerRes["memoryLimit"] = resource.ContainerResources.MemoryLimit
					}

					resourceSpec["containerResources"] = containerRes
				}

				resources[i] = resourceSpec
			}
			overrideSpec["resources"] = resources
		}

		if spec.Override.GitSyncDepth != nil {
			overrideSpec["gitSyncDepth"] = *spec.Override.GitSyncDepth
		}

		if spec.Override.EnableShellInRendering {
			overrideSpec["enableShellInRendering"] = true
		}

		result["override"] = overrideSpec
	}

	return result
}

// buildRepoSyncSpec builds RepoSync specification
func (csm *ConfigSyncManager) buildRepoSyncSpec(spec RepoSyncSpec) map[string]interface{} {
	// RepoSync spec is similar to RootSync spec but with some differences
	result := map[string]interface{}{
		"sourceFormat": spec.SourceFormat,
	}

	if spec.Git != nil {
		gitSpec := map[string]interface{}{
			"repo": spec.Git.Repo,
		}

		if spec.Git.Branch != "" {
			gitSpec["branch"] = spec.Git.Branch
		}

		if spec.Git.Revision != "" {
			gitSpec["revision"] = spec.Git.Revision
		}

		if spec.Git.Dir != "" {
			gitSpec["dir"] = spec.Git.Dir
		}

		if spec.Git.Auth != "" {
			gitSpec["auth"] = spec.Git.Auth
		}

		if spec.Git.SecretRef != nil {
			secretRef := map[string]interface{}{
				"name": spec.Git.SecretRef.Name,
			}
			if spec.Git.SecretRef.Key != "" {
				secretRef["key"] = spec.Git.SecretRef.Key
			}
			gitSpec["secretRef"] = secretRef
		}

		result["git"] = gitSpec
	}

	if spec.OCI != nil {
		ociSpec := map[string]interface{}{
			"image": spec.OCI.Image,
		}

		if spec.OCI.Dir != "" {
			ociSpec["dir"] = spec.OCI.Dir
		}

		if spec.OCI.Auth != "" {
			ociSpec["auth"] = spec.OCI.Auth
		}

		result["oci"] = ociSpec
	}

	if spec.Override != nil {
		overrideSpec := map[string]interface{}{}

		if spec.Override.StatusMode != "" {
			overrideSpec["statusMode"] = spec.Override.StatusMode
		}

		if spec.Override.ReconcileTimeout != nil {
			overrideSpec["reconcileTimeout"] = *spec.Override.ReconcileTimeout
		}

		if spec.Override.APIServerTimeout != nil {
			overrideSpec["apiServerTimeout"] = *spec.Override.APIServerTimeout
		}

		result["override"] = overrideSpec
	}

	return result
}

// parseRootSync parses RootSync from unstructured
func (csm *ConfigSyncManager) parseRootSync(obj *unstructured.Unstructured) (*RootSync, error) {
	rootSync := &RootSync{}

	// Parse metadata
	metadata, found, err := unstructured.NestedMap(obj.Object, "metadata")
	if err != nil || !found {
		return nil, fmt.Errorf("failed to get RootSync metadata: %w", err)
	}

	if name, found, err := unstructured.NestedString(metadata, "name"); err == nil && found {
		rootSync.Name = name
	}

	if namespace, found, err := unstructured.NestedString(metadata, "namespace"); err == nil && found {
		rootSync.Namespace = namespace
	}

	if labels, found, err := unstructured.NestedStringMap(metadata, "labels"); err == nil && found {
		rootSync.Labels = labels
	}

	if annotations, found, err := unstructured.NestedStringMap(metadata, "annotations"); err == nil && found {
		rootSync.Annotations = annotations
	}

	// Parse spec
	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil || !found {
		return nil, fmt.Errorf("failed to get RootSync spec: %w", err)
	}

	rootSync.Spec = csm.parseRootSyncSpec(spec)

	// Parse status if available
	if status, found, err := unstructured.NestedMap(obj.Object, "status"); err == nil && found {
		rootSync.Status = csm.parseRootSyncStatus(status)
	}

	return rootSync, nil
}

// parseRootSyncSpec parses RootSync specification
func (csm *ConfigSyncManager) parseRootSyncSpec(spec map[string]interface{}) RootSyncSpec {
	result := RootSyncSpec{}

	if sourceFormat, found, err := unstructured.NestedString(spec, "sourceFormat"); err == nil && found {
		result.SourceFormat = sourceFormat
	}

	// Parse Git configuration
	if gitConfig, found, err := unstructured.NestedMap(spec, "git"); err == nil && found {
		result.Git = &GitSyncSpec{}

		if repo, found, err := unstructured.NestedString(gitConfig, "repo"); err == nil && found {
			result.Git.Repo = repo
		}

		if branch, found, err := unstructured.NestedString(gitConfig, "branch"); err == nil && found {
			result.Git.Branch = branch
		}

		if revision, found, err := unstructured.NestedString(gitConfig, "revision"); err == nil && found {
			result.Git.Revision = revision
		}

		if dir, found, err := unstructured.NestedString(gitConfig, "dir"); err == nil && found {
			result.Git.Dir = dir
		}

		if auth, found, err := unstructured.NestedString(gitConfig, "auth"); err == nil && found {
			result.Git.Auth = auth
		}
	}

	// Parse OCI configuration
	if ociConfig, found, err := unstructured.NestedMap(spec, "oci"); err == nil && found {
		result.OCI = &OCISyncSpec{}

		if image, found, err := unstructured.NestedString(ociConfig, "image"); err == nil && found {
			result.OCI.Image = image
		}

		if dir, found, err := unstructured.NestedString(ociConfig, "dir"); err == nil && found {
			result.OCI.Dir = dir
		}

		if auth, found, err := unstructured.NestedString(ociConfig, "auth"); err == nil && found {
			result.OCI.Auth = auth
		}
	}

	return result
}

// parseRootSyncStatus parses RootSync status
func (csm *ConfigSyncManager) parseRootSyncStatus(status map[string]interface{}) RootSyncStatus {
	result := RootSyncStatus{}

	if observedGeneration, found, err := unstructured.NestedInt64(status, "observedGeneration"); err == nil && found {
		result.ObservedGeneration = observedGeneration
	}

	if lastSyncedCommit, found, err := unstructured.NestedString(status, "lastSyncedCommit"); err == nil && found {
		result.LastSyncedCommit = lastSyncedCommit
	}

	// Parse conditions
	if conditions, found, err := unstructured.NestedSlice(status, "conditions"); err == nil && found {
		for _, conditionInterface := range conditions {
			if condition, ok := conditionInterface.(map[string]interface{}); ok {
				syncCondition := RootSyncCondition{}

				if condType, found, err := unstructured.NestedString(condition, "type"); err == nil && found {
					syncCondition.Type = condType
				}

				if condStatus, found, err := unstructured.NestedString(condition, "status"); err == nil && found {
					syncCondition.Status = condStatus
				}

				if reason, found, err := unstructured.NestedString(condition, "reason"); err == nil && found {
					syncCondition.Reason = reason
				}

				if message, found, err := unstructured.NestedString(condition, "message"); err == nil && found {
					syncCondition.Message = message
				}

				result.Conditions = append(result.Conditions, syncCondition)
			}
		}
	}

	return result
}

// parseSyncStatus parses sync status from an object
func (csm *ConfigSyncManager) parseSyncStatus(obj *unstructured.Unstructured) (*SyncStatusReport, error) {
	report := &SyncStatusReport{}

	// Parse metadata
	if name, found, err := unstructured.NestedString(obj.Object, "metadata", "name"); err == nil && found {
		report.Name = name
	}

	if namespace, found, err := unstructured.NestedString(obj.Object, "metadata", "namespace"); err == nil && found {
		report.Namespace = namespace
	}

	if kind, found, err := unstructured.NestedString(obj.Object, "kind"); err == nil && found {
		report.Type = kind
	}

	// Parse status
	status, found, err := unstructured.NestedMap(obj.Object, "status")
	if err != nil || !found {
		report.Overall = "Unknown"
		return report, nil
	}

	// Determine overall status
	report.Overall = csm.determineOverallStatus(status)

	if lastSyncedCommit, found, err := unstructured.NestedString(status, "lastSyncedCommit"); err == nil && found {
		report.LastSyncedCommit = lastSyncedCommit
	}

	// Parse conditions
	if conditions, found, err := unstructured.NestedSlice(status, "conditions"); err == nil && found {
		for _, conditionInterface := range conditions {
			if condition, ok := conditionInterface.(map[string]interface{}); ok {
				syncCondition := RootSyncCondition{}

				if condType, found, err := unstructured.NestedString(condition, "type"); err == nil && found {
					syncCondition.Type = condType
				}

				if condStatus, found, err := unstructured.NestedString(condition, "status"); err == nil && found {
					syncCondition.Status = condStatus
				}

				if reason, found, err := unstructured.NestedString(condition, "reason"); err == nil && found {
					syncCondition.Reason = reason
				}

				if message, found, err := unstructured.NestedString(condition, "message"); err == nil && found {
					syncCondition.Message = message
				}

				report.Conditions = append(report.Conditions, syncCondition)
			}
		}
	}

	return report, nil
}

// determineOverallStatus determines overall sync status
func (csm *ConfigSyncManager) determineOverallStatus(status map[string]interface{}) string {
	if conditions, found, err := unstructured.NestedSlice(status, "conditions"); err == nil && found {
		syncedCondition := false
		errorCondition := false

		for _, conditionInterface := range conditions {
			if condition, ok := conditionInterface.(map[string]interface{}); ok {
				if condType, found, err := unstructured.NestedString(condition, "type"); err == nil && found {
					if condStatus, found, err := unstructured.NestedString(condition, "status"); err == nil && found {
						switch condType {
						case "Synced":
							if condStatus == "True" {
								syncedCondition = true
							}
						case "Stalled":
							if condStatus == "True" {
								errorCondition = true
							}
						}
					}
				}
			}
		}

		if errorCondition {
			return "Error"
		}

		if syncedCondition {
			return "Synced"
		}

		return "Syncing"
	}

	return "Unknown"
}

// stringPtr returns a pointer to the given string
func stringPtr(s string) *string {
	return &s
}