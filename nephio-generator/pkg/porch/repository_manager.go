package porch

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// RepositoryManager manages Porch repositories for Nephio package deployment
type RepositoryManager struct {
	client        dynamic.Interface
	restConfig    *rest.Config
	namespace     string
	defaultBranch string
}

// Repository represents a Porch repository configuration
type Repository struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Type        RepositoryType    `json:"type"`
	GitConfig   *GitConfig        `json:"gitConfig,omitempty"`
	OciConfig   *OciConfig        `json:"ociConfig,omitempty"`
	Deployment  bool              `json:"deployment"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Status      RepositoryStatus  `json:"status"`
}

// RepositoryType represents the type of repository
type RepositoryType string

const (
	RepositoryTypeGit RepositoryType = "git"
	RepositoryTypeOCI RepositoryType = "oci"
)

// GitConfig represents Git repository configuration
type GitConfig struct {
	Repo         string            `json:"repo"`
	Branch       string            `json:"branch"`
	Directory    string            `json:"directory,omitempty"`
	SecretRef    *SecretRef        `json:"secretRef,omitempty"`
	CreateBranch bool              `json:"createBranch,omitempty"`
	Auth         GitAuthType       `json:"auth"`
	Credentials  map[string]string `json:"credentials,omitempty"`
}

// OciConfig represents OCI repository configuration
type OciConfig struct {
	Registry  string     `json:"registry"`
	SecretRef *SecretRef `json:"secretRef,omitempty"`
}

// SecretRef represents a reference to a Kubernetes secret
type SecretRef struct {
	Name string `json:"name"`
	Key  string `json:"key,omitempty"`
}

// GitAuthType represents Git authentication type
type GitAuthType string

const (
	GitAuthTypeNone  GitAuthType = "none"
	GitAuthTypeToken GitAuthType = "token"
	GitAuthTypeSSH   GitAuthType = "ssh"
)

// RepositoryStatus represents repository status
type RepositoryStatus struct {
	Conditions    []RepositoryCondition `json:"conditions,omitempty"`
	LastSyncTime  *time.Time            `json:"lastSyncTime,omitempty"`
	PackageCount  int                   `json:"packageCount"`
	Ready         bool                  `json:"ready"`
	ErrorMessage  string                `json:"errorMessage,omitempty"`
}

// RepositoryCondition represents a repository condition
type RepositoryCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	LastTransitionTime time.Time `json:"lastTransitionTime"`
	Reason             string    `json:"reason"`
	Message            string    `json:"message"`
}

// PackageRevision represents a package revision in Porch
type PackageRevision struct {
	Name        string                 `json:"name"`
	Namespace   string                 `json:"namespace"`
	Repository  string                 `json:"repository"`
	Package     string                 `json:"package"`
	Revision    string                 `json:"revision"`
	WorkspaceName string               `json:"workspaceName"`
	Lifecycle   PackageLifecycle       `json:"lifecycle"`
	ReadinessGates []ReadinessGate      `json:"readinessGates,omitempty"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Annotations map[string]string      `json:"annotations,omitempty"`
	Spec        PackageRevisionSpec    `json:"spec"`
	Status      PackageRevisionStatus  `json:"status"`
}

// PackageLifecycle represents package lifecycle
type PackageLifecycle string

const (
	PackageLifecycleDraft         PackageLifecycle = "Draft"
	PackageLifecycleProposed      PackageLifecycle = "Proposed"
	PackageLifecyclePublished     PackageLifecycle = "Published"
	PackageLifecycleDeletionStart PackageLifecycle = "DeletionStart"
)

// PackageRevisionSpec represents package revision specification
type PackageRevisionSpec struct {
	PackageName    string                 `json:"packageName"`
	WorkspaceName  string                 `json:"workspaceName"`
	Revision       string                 `json:"revision"`
	Repository     string                 `json:"repository"`
	Tasks          []Task                 `json:"tasks,omitempty"`
	ReadinessGates []ReadinessGate        `json:"readinessGates,omitempty"`
	Lifecycle      PackageLifecycle       `json:"lifecycle"`
}

// PackageRevisionStatus represents package revision status
type PackageRevisionStatus struct {
	Conditions          []PackageRevisionCondition `json:"conditions,omitempty"`
	PublishedBy         string                     `json:"publishedBy,omitempty"`
	PublishedAt         *time.Time                 `json:"publishedAt,omitempty"`
	UpstreamLock        *UpstreamLock              `json:"upstreamLock,omitempty"`
	DeploymentStatus    DeploymentStatus           `json:"deploymentStatus"`
}

// Task represents a Kpt function task
type Task struct {
	Type   TaskType `json:"type"`
	Init   *Init    `json:"init,omitempty"`
	Clone  *Clone   `json:"clone,omitempty"`
	Edit   *Edit    `json:"edit,omitempty"`
	Eval   *Eval    `json:"eval,omitempty"`
}

// TaskType represents task type
type TaskType string

const (
	TaskTypeInit  TaskType = "init"
	TaskTypeClone TaskType = "clone"
	TaskTypeEdit  TaskType = "edit"
	TaskTypeEval  TaskType = "eval"
)

// Init represents init task
type Init struct {
	Subpackage string            `json:"subpackage,omitempty"`
	Keywords   []string          `json:"keywords,omitempty"`
	Site       string            `json:"site,omitempty"`
	License    string            `json:"license,omitempty"`
	Data       map[string]string `json:"data,omitempty"`
}

// Clone represents clone task
type Clone struct {
	Upstream UpstreamRef `json:"upstream"`
	Strategy string      `json:"strategy,omitempty"`
}

// Edit represents edit task
type Edit struct {
	Source *Source `json:"source,omitempty"`
}

// Eval represents eval task
type Eval struct {
	Image      string                 `json:"image"`
	ConfigPath string                 `json:"configPath,omitempty"`
	Config     map[string]interface{} `json:"config,omitempty"`
}

// UpstreamRef represents upstream reference
type UpstreamRef struct {
	Type       string `json:"type"`
	Git        *Git   `json:"git,omitempty"`
	Oci        *Oci   `json:"oci,omitempty"`
}

// Git represents git upstream
type Git struct {
	Repo      string `json:"repo"`
	Ref       string `json:"ref"`
	Directory string `json:"directory,omitempty"`
}

// Oci represents OCI upstream
type Oci struct {
	Image string `json:"image"`
}

// Source represents package source
type Source struct {
	Repo      string `json:"repo,omitempty"`
	Directory string `json:"directory,omitempty"`
	Ref       string `json:"ref,omitempty"`
}

// ReadinessGate represents a readiness gate
type ReadinessGate struct {
	ConditionType string `json:"conditionType"`
}

// PackageRevisionCondition represents a package revision condition
type PackageRevisionCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	LastTransitionTime time.Time `json:"lastTransitionTime"`
	Reason             string    `json:"reason"`
	Message            string    `json:"message"`
}

// UpstreamLock represents upstream lock information
type UpstreamLock struct {
	Type string        `json:"type"`
	Git  *GitLock      `json:"git,omitempty"`
	Oci  *OciLock      `json:"oci,omitempty"`
}

// GitLock represents git lock
type GitLock struct {
	Repo      string `json:"repo"`
	Directory string `json:"directory"`
	Ref       string `json:"ref"`
	Commit    string `json:"commit"`
}

// OciLock represents OCI lock
type OciLock struct {
	Image  string `json:"image"`
	Digest string `json:"digest"`
}

// DeploymentStatus represents deployment status
type DeploymentStatus struct {
	Deployed         bool      `json:"deployed"`
	DeploymentTime   *time.Time `json:"deploymentTime,omitempty"`
	DeploymentTarget string    `json:"deploymentTarget,omitempty"`
	Conditions       []DeploymentCondition `json:"conditions,omitempty"`
}

// DeploymentCondition represents deployment condition
type DeploymentCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	LastTransitionTime time.Time `json:"lastTransitionTime"`
	Reason             string    `json:"reason"`
	Message            string    `json:"message"`
}

// Porch GVRs
var (
	RepositoryGVR = schema.GroupVersionResource{
		Group:    "config.porch.kpt.dev",
		Version:  "v1alpha1",
		Resource: "repositories",
	}

	PackageRevisionGVR = schema.GroupVersionResource{
		Group:    "porch.kpt.dev",
		Version:  "v1alpha1",
		Resource: "packagerevisions",
	}

	PackageRevisionResourceGVR = schema.GroupVersionResource{
		Group:    "porch.kpt.dev",
		Version:  "v1alpha1",
		Resource: "packagerevisionresources",
	}
)

// NewRepositoryManager creates a new repository manager
func NewRepositoryManager(config *rest.Config, namespace, defaultBranch string) (*RepositoryManager, error) {
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &RepositoryManager{
		client:        dynamicClient,
		restConfig:    config,
		namespace:     namespace,
		defaultBranch: defaultBranch,
	}, nil
}

// CreateRepository creates a new Porch repository
func (rm *RepositoryManager) CreateRepository(ctx context.Context, repo *Repository) error {
	// Create repository manifest
	repoManifest := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "config.porch.kpt.dev/v1alpha1",
			"kind":       "Repository",
			"metadata": map[string]interface{}{
				"name":      repo.Name,
				"namespace": repo.Namespace,
				"labels":    repo.Labels,
				"annotations": repo.Annotations,
			},
			"spec": rm.buildRepositorySpec(repo),
		},
	}

	// Create repository
	_, err := rm.client.Resource(RepositoryGVR).Namespace(repo.Namespace).
		Create(ctx, repoManifest, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create repository %s: %w", repo.Name, err)
	}

	return nil
}

// GetRepository retrieves a repository
func (rm *RepositoryManager) GetRepository(ctx context.Context, name, namespace string) (*Repository, error) {
	repoManifest, err := rm.client.Resource(RepositoryGVR).Namespace(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get repository %s: %w", name, err)
	}

	return rm.parseRepository(repoManifest)
}

// ListRepositories lists all repositories
func (rm *RepositoryManager) ListRepositories(ctx context.Context, namespace string) ([]*Repository, error) {
	repoList, err := rm.client.Resource(RepositoryGVR).Namespace(namespace).
		List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	var repositories []*Repository
	for _, item := range repoList.Items {
		repo, err := rm.parseRepository(&item)
		if err != nil {
			return nil, fmt.Errorf("failed to parse repository: %w", err)
		}
		repositories = append(repositories, repo)
	}

	return repositories, nil
}

// UpdateRepository updates a repository
func (rm *RepositoryManager) UpdateRepository(ctx context.Context, repo *Repository) error {
	// Get existing repository
	existingRepo, err := rm.client.Resource(RepositoryGVR).Namespace(repo.Namespace).
		Get(ctx, repo.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get existing repository %s: %w", repo.Name, err)
	}

	// Update spec
	spec := rm.buildRepositorySpec(repo)
	if err := unstructured.SetNestedMap(existingRepo.Object, spec, "spec"); err != nil {
		return fmt.Errorf("failed to set repository spec: %w", err)
	}

	// Update labels and annotations
	if repo.Labels != nil {
		if err := unstructured.SetNestedStringMap(existingRepo.Object, repo.Labels, "metadata", "labels"); err != nil {
			return fmt.Errorf("failed to set repository labels: %w", err)
		}
	}

	if repo.Annotations != nil {
		if err := unstructured.SetNestedStringMap(existingRepo.Object, repo.Annotations, "metadata", "annotations"); err != nil {
			return fmt.Errorf("failed to set repository annotations: %w", err)
		}
	}

	// Update repository
	_, err = rm.client.Resource(RepositoryGVR).Namespace(repo.Namespace).
		Update(ctx, existingRepo, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update repository %s: %w", repo.Name, err)
	}

	return nil
}

// DeleteRepository deletes a repository
func (rm *RepositoryManager) DeleteRepository(ctx context.Context, name, namespace string) error {
	err := rm.client.Resource(RepositoryGVR).Namespace(namespace).
		Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete repository %s: %w", name, err)
	}

	return nil
}

// CreatePackageRevision creates a new package revision
func (rm *RepositoryManager) CreatePackageRevision(ctx context.Context, pr *PackageRevision) error {
	// Create package revision manifest
	prManifest := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "porch.kpt.dev/v1alpha1",
			"kind":       "PackageRevision",
			"metadata": map[string]interface{}{
				"name":      pr.Name,
				"namespace": pr.Namespace,
				"labels":    pr.Labels,
				"annotations": pr.Annotations,
			},
			"spec": rm.buildPackageRevisionSpec(pr),
		},
	}

	// Create package revision
	_, err := rm.client.Resource(PackageRevisionGVR).Namespace(pr.Namespace).
		Create(ctx, prManifest, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create package revision %s: %w", pr.Name, err)
	}

	return nil
}

// GetPackageRevision retrieves a package revision
func (rm *RepositoryManager) GetPackageRevision(ctx context.Context, name, namespace string) (*PackageRevision, error) {
	prManifest, err := rm.client.Resource(PackageRevisionGVR).Namespace(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get package revision %s: %w", name, err)
	}

	return rm.parsePackageRevision(prManifest)
}

// ListPackageRevisions lists package revisions
func (rm *RepositoryManager) ListPackageRevisions(ctx context.Context, namespace string, repository string) ([]*PackageRevision, error) {
	listOptions := metav1.ListOptions{}
	if repository != "" {
		listOptions.LabelSelector = fmt.Sprintf("porch.kpt.dev/repository=%s", repository)
	}

	prList, err := rm.client.Resource(PackageRevisionGVR).Namespace(namespace).
		List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list package revisions: %w", err)
	}

	var packageRevisions []*PackageRevision
	for _, item := range prList.Items {
		pr, err := rm.parsePackageRevision(&item)
		if err != nil {
			return nil, fmt.Errorf("failed to parse package revision: %w", err)
		}
		packageRevisions = append(packageRevisions, pr)
	}

	return packageRevisions, nil
}

// UpdatePackageRevision updates a package revision
func (rm *RepositoryManager) UpdatePackageRevision(ctx context.Context, pr *PackageRevision) error {
	// Get existing package revision
	existingPR, err := rm.client.Resource(PackageRevisionGVR).Namespace(pr.Namespace).
		Get(ctx, pr.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get existing package revision %s: %w", pr.Name, err)
	}

	// Update spec
	spec := rm.buildPackageRevisionSpec(pr)
	if err := unstructured.SetNestedMap(existingPR.Object, spec, "spec"); err != nil {
		return fmt.Errorf("failed to set package revision spec: %w", err)
	}

	// Update labels and annotations
	if pr.Labels != nil {
		if err := unstructured.SetNestedStringMap(existingPR.Object, pr.Labels, "metadata", "labels"); err != nil {
			return fmt.Errorf("failed to set package revision labels: %w", err)
		}
	}

	if pr.Annotations != nil {
		if err := unstructured.SetNestedStringMap(existingPR.Object, pr.Annotations, "metadata", "annotations"); err != nil {
			return fmt.Errorf("failed to set package revision annotations: %w", err)
		}
	}

	// Update package revision
	_, err = rm.client.Resource(PackageRevisionGVR).Namespace(pr.Namespace).
		Update(ctx, existingPR, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update package revision %s: %w", pr.Name, err)
	}

	return nil
}

// DeletePackageRevision deletes a package revision
func (rm *RepositoryManager) DeletePackageRevision(ctx context.Context, name, namespace string) error {
	err := rm.client.Resource(PackageRevisionGVR).Namespace(namespace).
		Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete package revision %s: %w", name, err)
	}

	return nil
}

// ProposePackageRevision proposes a package revision for publishing
func (rm *RepositoryManager) ProposePackageRevision(ctx context.Context, name, namespace string) error {
	return rm.updatePackageLifecycle(ctx, name, namespace, PackageLifecycleProposed)
}

// PublishPackageRevision publishes a package revision
func (rm *RepositoryManager) PublishPackageRevision(ctx context.Context, name, namespace string) error {
	return rm.updatePackageLifecycle(ctx, name, namespace, PackageLifecyclePublished)
}

// CreateBranch creates a new branch for package development
func (rm *RepositoryManager) CreateBranch(ctx context.Context, repositoryName, branchName, baseBranch string) error {
	// This would typically involve creating a PackageRevision with a Clone task
	// pointing to the base branch and specifying the new branch name

	pr := &PackageRevision{
		Name:      fmt.Sprintf("%s-%s-%d", repositoryName, branchName, time.Now().Unix()),
		Namespace: rm.namespace,
		Repository: repositoryName,
		WorkspaceName: branchName,
		Lifecycle: PackageLifecycleDraft,
		Spec: PackageRevisionSpec{
			PackageName:   branchName,
			WorkspaceName: branchName,
			Repository:    repositoryName,
			Lifecycle:     PackageLifecycleDraft,
			Tasks: []Task{
				{
					Type: TaskTypeClone,
					Clone: &Clone{
						Upstream: UpstreamRef{
							Type: "git",
							Git: &Git{
								Repo: "", // Will be filled by Porch
								Ref:  baseBranch,
							},
						},
						Strategy: "resource-merge",
					},
				},
			},
		},
	}

	return rm.CreatePackageRevision(ctx, pr)
}

// GetBranches lists all branches in a repository
func (rm *RepositoryManager) GetBranches(ctx context.Context, repositoryName string) ([]string, error) {
	// List all package revisions for the repository
	packageRevisions, err := rm.ListPackageRevisions(ctx, rm.namespace, repositoryName)
	if err != nil {
		return nil, fmt.Errorf("failed to list package revisions: %w", err)
	}

	// Extract unique workspace names (branches)
	branchSet := make(map[string]bool)
	for _, pr := range packageRevisions {
		if pr.WorkspaceName != "" {
			branchSet[pr.WorkspaceName] = true
		}
	}

	branches := make([]string, 0, len(branchSet))
	for branch := range branchSet {
		branches = append(branches, branch)
	}

	return branches, nil
}

// buildRepositorySpec builds repository specification
func (rm *RepositoryManager) buildRepositorySpec(repo *Repository) map[string]interface{} {
	spec := map[string]interface{}{
		"type":       string(repo.Type),
		"deployment": repo.Deployment,
	}

	switch repo.Type {
	case RepositoryTypeGit:
		if repo.GitConfig != nil {
			gitSpec := map[string]interface{}{
				"repo":   repo.GitConfig.Repo,
				"branch": repo.GitConfig.Branch,
			}

			if repo.GitConfig.Directory != "" {
				gitSpec["directory"] = repo.GitConfig.Directory
			}

			if repo.GitConfig.SecretRef != nil {
				gitSpec["secretRef"] = map[string]interface{}{
					"name": repo.GitConfig.SecretRef.Name,
				}
				if repo.GitConfig.SecretRef.Key != "" {
					gitSpec["secretRef"].(map[string]interface{})["key"] = repo.GitConfig.SecretRef.Key
				}
			}

			if repo.GitConfig.CreateBranch {
				gitSpec["createBranch"] = true
			}

			spec["git"] = gitSpec
		}
	case RepositoryTypeOCI:
		if repo.OciConfig != nil {
			ociSpec := map[string]interface{}{
				"registry": repo.OciConfig.Registry,
			}

			if repo.OciConfig.SecretRef != nil {
				ociSpec["secretRef"] = map[string]interface{}{
					"name": repo.OciConfig.SecretRef.Name,
				}
				if repo.OciConfig.SecretRef.Key != "" {
					ociSpec["secretRef"].(map[string]interface{})["key"] = repo.OciConfig.SecretRef.Key
				}
			}

			spec["oci"] = ociSpec
		}
	}

	return spec
}

// buildPackageRevisionSpec builds package revision specification
func (rm *RepositoryManager) buildPackageRevisionSpec(pr *PackageRevision) map[string]interface{} {
	spec := map[string]interface{}{
		"packageName":   pr.Spec.PackageName,
		"workspaceName": pr.Spec.WorkspaceName,
		"revision":      pr.Spec.Revision,
		"repository":    pr.Spec.Repository,
		"lifecycle":     string(pr.Spec.Lifecycle),
	}

	if len(pr.Spec.Tasks) > 0 {
		tasks := make([]map[string]interface{}, len(pr.Spec.Tasks))
		for i, task := range pr.Spec.Tasks {
			tasks[i] = rm.buildTaskSpec(task)
		}
		spec["tasks"] = tasks
	}

	if len(pr.Spec.ReadinessGates) > 0 {
		gates := make([]map[string]interface{}, len(pr.Spec.ReadinessGates))
		for i, gate := range pr.Spec.ReadinessGates {
			gates[i] = map[string]interface{}{
				"conditionType": gate.ConditionType,
			}
		}
		spec["readinessGates"] = gates
	}

	return spec
}

// buildTaskSpec builds task specification
func (rm *RepositoryManager) buildTaskSpec(task Task) map[string]interface{} {
	taskSpec := map[string]interface{}{
		"type": string(task.Type),
	}

	switch task.Type {
	case TaskTypeInit:
		if task.Init != nil {
			initSpec := map[string]interface{}{}
			if task.Init.Subpackage != "" {
				initSpec["subpackage"] = task.Init.Subpackage
			}
			if len(task.Init.Keywords) > 0 {
				initSpec["keywords"] = task.Init.Keywords
			}
			if task.Init.Site != "" {
				initSpec["site"] = task.Init.Site
			}
			if task.Init.License != "" {
				initSpec["license"] = task.Init.License
			}
			if len(task.Init.Data) > 0 {
				initSpec["data"] = task.Init.Data
			}
			taskSpec["init"] = initSpec
		}
	case TaskTypeClone:
		if task.Clone != nil {
			cloneSpec := map[string]interface{}{
				"upstream": rm.buildUpstreamRef(task.Clone.Upstream),
			}
			if task.Clone.Strategy != "" {
				cloneSpec["strategy"] = task.Clone.Strategy
			}
			taskSpec["clone"] = cloneSpec
		}
	case TaskTypeEdit:
		if task.Edit != nil {
			editSpec := map[string]interface{}{}
			if task.Edit.Source != nil {
				sourceSpec := map[string]interface{}{}
				if task.Edit.Source.Repo != "" {
					sourceSpec["repo"] = task.Edit.Source.Repo
				}
				if task.Edit.Source.Directory != "" {
					sourceSpec["directory"] = task.Edit.Source.Directory
				}
				if task.Edit.Source.Ref != "" {
					sourceSpec["ref"] = task.Edit.Source.Ref
				}
				editSpec["source"] = sourceSpec
			}
			taskSpec["edit"] = editSpec
		}
	case TaskTypeEval:
		if task.Eval != nil {
			evalSpec := map[string]interface{}{
				"image": task.Eval.Image,
			}
			if task.Eval.ConfigPath != "" {
				evalSpec["configPath"] = task.Eval.ConfigPath
			}
			if len(task.Eval.Config) > 0 {
				evalSpec["config"] = task.Eval.Config
			}
			taskSpec["eval"] = evalSpec
		}
	}

	return taskSpec
}

// buildUpstreamRef builds upstream reference
func (rm *RepositoryManager) buildUpstreamRef(upstream UpstreamRef) map[string]interface{} {
	upstreamSpec := map[string]interface{}{
		"type": upstream.Type,
	}

	switch upstream.Type {
	case "git":
		if upstream.Git != nil {
			gitSpec := map[string]interface{}{
				"repo": upstream.Git.Repo,
				"ref":  upstream.Git.Ref,
			}
			if upstream.Git.Directory != "" {
				gitSpec["directory"] = upstream.Git.Directory
			}
			upstreamSpec["git"] = gitSpec
		}
	case "oci":
		if upstream.Oci != nil {
			upstreamSpec["oci"] = map[string]interface{}{
				"image": upstream.Oci.Image,
			}
		}
	}

	return upstreamSpec
}

// parseRepository parses repository from unstructured
func (rm *RepositoryManager) parseRepository(obj *unstructured.Unstructured) (*Repository, error) {
	repo := &Repository{}

	if err := rm.parseRepositoryMetadata(obj, repo); err != nil {
		return nil, err
	}

	if err := rm.parseRepositorySpec(obj, repo); err != nil {
		return nil, err
	}

	if err := rm.parseRepositoryStatusFromObject(obj, repo); err != nil {
		return nil, err
	}

	return repo, nil
}

// parseRepositoryMetadata parses repository metadata
func (rm *RepositoryManager) parseRepositoryMetadata(obj *unstructured.Unstructured, repo *Repository) error {
	metadata, found, err := unstructured.NestedMap(obj.Object, "metadata")
	if err != nil || !found {
		return fmt.Errorf("failed to get repository metadata: %w", err)
	}

	if name, found, err := unstructured.NestedString(metadata, "name"); err == nil && found {
		repo.Name = name
	}

	if namespace, found, err := unstructured.NestedString(metadata, "namespace"); err == nil && found {
		repo.Namespace = namespace
	}

	if labels, found, err := unstructured.NestedStringMap(metadata, "labels"); err == nil && found {
		repo.Labels = labels
	}

	if annotations, found, err := unstructured.NestedStringMap(metadata, "annotations"); err == nil && found {
		repo.Annotations = annotations
	}

	return nil
}

// parseRepositorySpec parses repository specification
func (rm *RepositoryManager) parseRepositorySpec(obj *unstructured.Unstructured, repo *Repository) error {
	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil || !found {
		return fmt.Errorf("failed to get repository spec: %w", err)
	}

	if repoType, found, err := unstructured.NestedString(spec, "type"); err == nil && found {
		repo.Type = RepositoryType(repoType)
	}

	if deployment, found, err := unstructured.NestedBool(spec, "deployment"); err == nil && found {
		repo.Deployment = deployment
	}

	if err := rm.parseGitConfig(spec, repo); err != nil {
		return err
	}

	if err := rm.parseOciConfig(spec, repo); err != nil {
		return err
	}

	return nil
}

// parseGitConfig parses Git configuration from spec
func (rm *RepositoryManager) parseGitConfig(spec map[string]interface{}, repo *Repository) error {
	gitConfig, found, err := unstructured.NestedMap(spec, "git")
	if err != nil || !found {
		return nil // Git config is optional
	}

	repo.GitConfig = &GitConfig{}

	if gitRepo, found, err := unstructured.NestedString(gitConfig, "repo"); err == nil && found {
		repo.GitConfig.Repo = gitRepo
	}

	if branch, found, err := unstructured.NestedString(gitConfig, "branch"); err == nil && found {
		repo.GitConfig.Branch = branch
	}

	if directory, found, err := unstructured.NestedString(gitConfig, "directory"); err == nil && found {
		repo.GitConfig.Directory = directory
	}

	if createBranch, found, err := unstructured.NestedBool(gitConfig, "createBranch"); err == nil && found {
		repo.GitConfig.CreateBranch = createBranch
	}

	return rm.parseSecretRef(gitConfig, &repo.GitConfig.SecretRef)
}

// parseOciConfig parses OCI configuration from spec
func (rm *RepositoryManager) parseOciConfig(spec map[string]interface{}, repo *Repository) error {
	ociConfig, found, err := unstructured.NestedMap(spec, "oci")
	if err != nil || !found {
		return nil // OCI config is optional
	}

	repo.OciConfig = &OciConfig{}

	if registry, found, err := unstructured.NestedString(ociConfig, "registry"); err == nil && found {
		repo.OciConfig.Registry = registry
	}

	return rm.parseSecretRef(ociConfig, &repo.OciConfig.SecretRef)
}

// parseSecretRef parses secret reference from config
func (rm *RepositoryManager) parseSecretRef(config map[string]interface{}, secretRef **SecretRef) error {
	secretRefMap, found, err := unstructured.NestedMap(config, "secretRef")
	if err != nil || !found {
		return nil // Secret ref is optional
	}

	*secretRef = &SecretRef{}

	if name, found, err := unstructured.NestedString(secretRefMap, "name"); err == nil && found {
		(*secretRef).Name = name
	}

	if key, found, err := unstructured.NestedString(secretRefMap, "key"); err == nil && found {
		(*secretRef).Key = key
	}

	return nil
}

// parseRepositoryStatusFromObject parses repository status from object
func (rm *RepositoryManager) parseRepositoryStatusFromObject(obj *unstructured.Unstructured, repo *Repository) error {
	status, found, err := unstructured.NestedMap(obj.Object, "status")
	if err != nil || !found {
		return nil // Status is optional
	}

	repo.Status = rm.parseRepositoryStatus(status)
	return nil
}

// parsePackageRevision parses package revision from unstructured
func (rm *RepositoryManager) parsePackageRevision(obj *unstructured.Unstructured) (*PackageRevision, error) {
	pr := &PackageRevision{}

	if err := rm.parsePackageRevisionMetadata(obj, pr); err != nil {
		return nil, err
	}

	if err := rm.parsePackageRevisionSpec(obj, pr); err != nil {
		return nil, err
	}

	if err := rm.parsePackageRevisionStatusFromObject(obj, pr); err != nil {
		return nil, err
	}

	return pr, nil
}

// parsePackageRevisionMetadata parses package revision metadata
func (rm *RepositoryManager) parsePackageRevisionMetadata(obj *unstructured.Unstructured, pr *PackageRevision) error {
	metadata, found, err := unstructured.NestedMap(obj.Object, "metadata")
	if err != nil || !found {
		return fmt.Errorf("failed to get package revision metadata: %w", err)
	}

	if name, found, err := unstructured.NestedString(metadata, "name"); err == nil && found {
		pr.Name = name
	}

	if namespace, found, err := unstructured.NestedString(metadata, "namespace"); err == nil && found {
		pr.Namespace = namespace
	}

	if annotations, found, err := unstructured.NestedStringMap(metadata, "annotations"); err == nil && found {
		pr.Annotations = annotations
	}

	return rm.parsePackageRevisionLabels(metadata, pr)
}

// parsePackageRevisionLabels parses package revision labels and extracts repository/package info
func (rm *RepositoryManager) parsePackageRevisionLabels(metadata map[string]interface{}, pr *PackageRevision) error {
	labels, found, err := unstructured.NestedStringMap(metadata, "labels")
	if err != nil || !found {
		return nil // Labels are optional
	}

	pr.Labels = labels

	// Extract repository from labels
	if repo, exists := labels["porch.kpt.dev/repository"]; exists {
		pr.Repository = repo
	}

	// Extract package from labels
	if pkg, exists := labels["porch.kpt.dev/package"]; exists {
		pr.Package = pkg
	}

	return nil
}

// parsePackageRevisionSpec parses package revision specification
func (rm *RepositoryManager) parsePackageRevisionSpec(obj *unstructured.Unstructured, pr *PackageRevision) error {
	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil || !found {
		return fmt.Errorf("failed to get package revision spec: %w", err)
	}

	if packageName, found, err := unstructured.NestedString(spec, "packageName"); err == nil && found {
		pr.Spec.PackageName = packageName
	}

	if workspaceName, found, err := unstructured.NestedString(spec, "workspaceName"); err == nil && found {
		pr.Spec.WorkspaceName = workspaceName
		pr.WorkspaceName = workspaceName
	}

	if revision, found, err := unstructured.NestedString(spec, "revision"); err == nil && found {
		pr.Spec.Revision = revision
		pr.Revision = revision
	}

	if repository, found, err := unstructured.NestedString(spec, "repository"); err == nil && found {
		pr.Spec.Repository = repository
		if pr.Repository == "" {
			pr.Repository = repository
		}
	}

	if lifecycle, found, err := unstructured.NestedString(spec, "lifecycle"); err == nil && found {
		pr.Spec.Lifecycle = PackageLifecycle(lifecycle)
		pr.Lifecycle = PackageLifecycle(lifecycle)
	}

	return nil
}

// parsePackageRevisionStatusFromObject parses package revision status from object
func (rm *RepositoryManager) parsePackageRevisionStatusFromObject(obj *unstructured.Unstructured, pr *PackageRevision) error {
	status, found, err := unstructured.NestedMap(obj.Object, "status")
	if err != nil || !found {
		return nil // Status is optional
	}

	pr.Status = rm.parsePackageRevisionStatus(status)
	return nil
}

// parseRepositoryStatus parses repository status
func (rm *RepositoryManager) parseRepositoryStatus(status map[string]interface{}) RepositoryStatus {
	repoStatus := RepositoryStatus{}

	if conditions, found, err := unstructured.NestedSlice(status, "conditions"); err == nil && found {
		for _, conditionInterface := range conditions {
			if condition, ok := conditionInterface.(map[string]interface{}); ok {
				repoCondition := RepositoryCondition{}

				if condType, found, err := unstructured.NestedString(condition, "type"); err == nil && found {
					repoCondition.Type = condType
				}

				if condStatus, found, err := unstructured.NestedString(condition, "status"); err == nil && found {
					repoCondition.Status = condStatus
				}

				if reason, found, err := unstructured.NestedString(condition, "reason"); err == nil && found {
					repoCondition.Reason = reason
				}

				if message, found, err := unstructured.NestedString(condition, "message"); err == nil && found {
					repoCondition.Message = message
				}

				if lastTransitionTime, found, err := unstructured.NestedString(condition, "lastTransitionTime"); err == nil && found {
					if parsedTime, err := time.Parse(time.RFC3339, lastTransitionTime); err == nil {
						repoCondition.LastTransitionTime = parsedTime
					}
				}

				repoStatus.Conditions = append(repoStatus.Conditions, repoCondition)
			}
		}
	}

	if lastSyncTime, found, err := unstructured.NestedString(status, "lastSyncTime"); err == nil && found {
		if parsedTime, err := time.Parse(time.RFC3339, lastSyncTime); err == nil {
			repoStatus.LastSyncTime = &parsedTime
		}
	}

	if packageCount, found, err := unstructured.NestedInt64(status, "packageCount"); err == nil && found {
		repoStatus.PackageCount = int(packageCount)
	}

	if ready, found, err := unstructured.NestedBool(status, "ready"); err == nil && found {
		repoStatus.Ready = ready
	}

	if errorMessage, found, err := unstructured.NestedString(status, "errorMessage"); err == nil && found {
		repoStatus.ErrorMessage = errorMessage
	}

	return repoStatus
}

// parsePackageRevisionStatus parses package revision status
func (rm *RepositoryManager) parsePackageRevisionStatus(status map[string]interface{}) PackageRevisionStatus {
	prStatus := PackageRevisionStatus{}

	if conditions, found, err := unstructured.NestedSlice(status, "conditions"); err == nil && found {
		for _, conditionInterface := range conditions {
			if condition, ok := conditionInterface.(map[string]interface{}); ok {
				prCondition := PackageRevisionCondition{}

				if condType, found, err := unstructured.NestedString(condition, "type"); err == nil && found {
					prCondition.Type = condType
				}

				if condStatus, found, err := unstructured.NestedString(condition, "status"); err == nil && found {
					prCondition.Status = condStatus
				}

				if reason, found, err := unstructured.NestedString(condition, "reason"); err == nil && found {
					prCondition.Reason = reason
				}

				if message, found, err := unstructured.NestedString(condition, "message"); err == nil && found {
					prCondition.Message = message
				}

				if lastTransitionTime, found, err := unstructured.NestedString(condition, "lastTransitionTime"); err == nil && found {
					if parsedTime, err := time.Parse(time.RFC3339, lastTransitionTime); err == nil {
						prCondition.LastTransitionTime = parsedTime
					}
				}

				prStatus.Conditions = append(prStatus.Conditions, prCondition)
			}
		}
	}

	if publishedBy, found, err := unstructured.NestedString(status, "publishedBy"); err == nil && found {
		prStatus.PublishedBy = publishedBy
	}

	if publishedAt, found, err := unstructured.NestedString(status, "publishedAt"); err == nil && found {
		if parsedTime, err := time.Parse(time.RFC3339, publishedAt); err == nil {
			prStatus.PublishedAt = &parsedTime
		}
	}

	return prStatus
}

// updatePackageLifecycle updates the lifecycle of a package revision
func (rm *RepositoryManager) updatePackageLifecycle(ctx context.Context, name, namespace string, lifecycle PackageLifecycle) error {
	// Get existing package revision
	existingPR, err := rm.client.Resource(PackageRevisionGVR).Namespace(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get existing package revision %s: %w", name, err)
	}

	// Update lifecycle in spec
	if err := unstructured.SetNestedField(existingPR.Object, string(lifecycle), "spec", "lifecycle"); err != nil {
		return fmt.Errorf("failed to set package revision lifecycle: %w", err)
	}

	// Update package revision
	_, err = rm.client.Resource(PackageRevisionGVR).Namespace(namespace).
		Update(ctx, existingPR, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update package revision %s: %w", name, err)
	}

	return nil
}