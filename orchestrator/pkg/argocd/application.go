package argocd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/orchestrator/pkg/argocd/apis"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// DefaultArgoCDNamespace is the default namespace for ArgoCD
	DefaultArgoCDNamespace = "argocd"

	// ConfigMapKeyManifests is the key in ConfigMap for manifests
	ConfigMapKeyManifests = "manifests.yaml"
)

// Client provides interface to interact with ArgoCD
type Client struct {
	Namespace       string
	K8sClientset    *kubernetes.Clientset
	DynamicClient   dynamic.Interface
	ApplicationGVR  schema.GroupVersionResource
}

// ApplicationConfig holds configuration for creating an ArgoCD Application
type ApplicationConfig struct {
	Name          string // Application name (same as slice ID)
	Namespace     string // Target namespace for slice deployment
	SliceType     string // eMBB, URLLC, mMTC, etc.
	ManifestsYAML string // YAML content of Kubernetes manifests
}

// Validate checks if the application config is valid
func (ac *ApplicationConfig) Validate() error {
	if ac.Name == "" {
		return fmt.Errorf("application name cannot be empty")
	}
	if ac.Namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}
	if ac.ManifestsYAML == "" {
		return fmt.Errorf("manifests cannot be empty")
	}
	return nil
}

// NewClient creates a new ArgoCD client
func NewClient(namespace string) (*Client, error) {
	if namespace == "" {
		namespace = DefaultArgoCDNamespace
	}

	// Get Kubernetes config
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fallback to kubeconfig
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
		config, err = kubeConfig.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
		}
	}

	// Create standard clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	// Create dynamic client for CRDs
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Define Argo CD Application GVR
	appGVR := schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "applications",
	}

	return &Client{
		Namespace:      namespace,
		K8sClientset:   clientset,
		DynamicClient:  dynamicClient,
		ApplicationGVR: appGVR,
	}, nil
}

// StoreManifestsInConfigMap stores manifests in a ConfigMap
func (c *Client) StoreManifestsInConfigMap(ctx context.Context, sliceID string, manifests string) (string, error) {
	cmName := fmt.Sprintf("%s-manifests", sliceID)

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: c.Namespace,
			Labels: map[string]string{
				"slice-id":   sliceID,
				"managed-by": "oran-orchestrator",
				"type":       "slice-manifests",
			},
		},
		Data: map[string]string{
			ConfigMapKeyManifests: manifests,
		},
	}

	// Try to create ConfigMap using standard clientset
	_, err := c.K8sClientset.CoreV1().ConfigMaps(c.Namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			// Update if already exists
			existingCM, getErr := c.K8sClientset.CoreV1().ConfigMaps(c.Namespace).Get(ctx, cmName, metav1.GetOptions{})
			if getErr != nil {
				return "", fmt.Errorf("failed to get existing configmap: %w", getErr)
			}

			existingCM.Data = configMap.Data
			_, updateErr := c.K8sClientset.CoreV1().ConfigMaps(c.Namespace).Update(ctx, existingCM, metav1.UpdateOptions{})
			if updateErr != nil {
				return "", fmt.Errorf("failed to update configmap: %w", updateErr)
			}
		} else {
			return "", fmt.Errorf("failed to create configmap: %w", err)
		}
	}

	return cmName, nil
}

// CreateApplication creates an ArgoCD Application
func (c *Client) CreateApplication(ctx context.Context, config ApplicationConfig) (*apis.Application, error) {
	// Validate config
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Store manifests in ConfigMap
	cmName, err := c.StoreManifestsInConfigMap(ctx, config.Name, config.ManifestsYAML)
	if err != nil {
		return nil, fmt.Errorf("failed to store manifests in configmap: %w", err)
	}

	// Create ArgoCD Application
	prune := true
	selfHeal := true

	app := &apis.Application{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "argoproj.io/v1alpha1",
			Kind:       "Application",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: c.Namespace,
			Labels: map[string]string{
				"slice-id":   config.Name,
				"slice-type": strings.ToLower(config.SliceType),
				"managed-by": "oran-orchestrator",
			},
			Finalizers: []string{
				"resources-finalizer.argocd.argoproj.io",
			},
		},
		Spec: apis.ApplicationSpec{
			Project: "default",
			Source: &apis.ApplicationSource{
				// Dummy repo URL - in production this would point to actual Git repo
				// or use Argo CD's ApplicationSet for dynamic generation
				RepoURL: "https://github.com/placeholder/manifests",
				Path:    ".", // Required by Argo CD
				// Use ConfigMap plugin to read manifests
				Plugin: &apis.ApplicationSourcePlugin{
					Name: "configmap",
					Env: []apis.EnvEntry{
						{
							Name:  "CONFIGMAP_NAME",
							Value: cmName,
						},
						{
							Name:  "CONFIGMAP_NAMESPACE",
							Value: c.Namespace,
						},
						{
							Name:  "CONFIGMAP_KEY",
							Value: ConfigMapKeyManifests,
						},
					},
				},
			},
			Destination: apis.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: config.Namespace,
			},
			SyncPolicy: &apis.SyncPolicy{
				Automated: &apis.SyncPolicyAutomated{
					Prune:    &prune,
					SelfHeal: &selfHeal,
				},
				SyncOptions: []string{
					"CreateNamespace=true",
				},
			},
		},
	}

	// Convert typed Application to unstructured for dynamic client
	unstructuredApp, err := toUnstructured(app)
	if err != nil {
		return nil, fmt.Errorf("failed to convert application to unstructured: %w", err)
	}

	// Create the application using dynamic client
	createdUnstructured, err := c.DynamicClient.Resource(c.ApplicationGVR).Namespace(c.Namespace).Create(ctx, unstructuredApp, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			// Get existing application
			existingUnstructured, getErr := c.DynamicClient.Resource(c.ApplicationGVR).Namespace(c.Namespace).Get(ctx, config.Name, metav1.GetOptions{})
			if getErr != nil {
				return nil, fmt.Errorf("failed to get existing application: %w", getErr)
			}

			// Convert back to typed Application
			existingApp := &apis.Application{}
			if convErr := fromUnstructured(existingUnstructured, existingApp); convErr != nil {
				return nil, fmt.Errorf("failed to convert existing application: %w", convErr)
			}
			return existingApp, nil
		}
		return nil, fmt.Errorf("failed to create application: %w", err)
	}

	// Convert result back to typed Application
	createdApp := &apis.Application{}
	if err := fromUnstructured(createdUnstructured, createdApp); err != nil {
		return nil, fmt.Errorf("failed to convert created application: %w", err)
	}

	return createdApp, nil
}

// toUnstructured converts a typed object to unstructured
func toUnstructured(obj interface{}) (*unstructured.Unstructured, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	var unstructuredMap map[string]interface{}
	if err := json.Unmarshal(data, &unstructuredMap); err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{Object: unstructuredMap}, nil
}

// fromUnstructured converts unstructured to a typed object
func fromUnstructured(u *unstructured.Unstructured, obj interface{}) error {
	data, err := u.MarshalJSON()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, obj)
}

// GetApplication retrieves an ArgoCD Application
func (c *Client) GetApplication(ctx context.Context, name string) (*apis.Application, error) {
	unstructuredApp, err := c.DynamicClient.Resource(c.ApplicationGVR).Namespace(c.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	app := &apis.Application{}
	if err := fromUnstructured(unstructuredApp, app); err != nil {
		return nil, fmt.Errorf("failed to convert application: %w", err)
	}
	return app, nil
}

// DeleteApplication deletes an ArgoCD Application
func (c *Client) DeleteApplication(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("application name cannot be empty")
	}

	err := c.DynamicClient.Resource(c.ApplicationGVR).Namespace(c.Namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("application not found: %s", name)
		}
		return fmt.Errorf("failed to delete application: %w", err)
	}

	// Also delete the ConfigMap
	cmName := fmt.Sprintf("%s-manifests", name)
	_ = c.K8sClientset.CoreV1().ConfigMaps(c.Namespace).Delete(ctx, cmName, metav1.DeleteOptions{}) // Ignore error if CM doesn't exist

	return nil
}

// GetSyncStatus retrieves the sync status of an ArgoCD Application
func (c *Client) GetSyncStatus(ctx context.Context, name string) (string, error) {
	app, err := c.GetApplication(ctx, name)
	if err != nil {
		return "", err
	}

	if app.Status.Sync.Status == "" {
		return "Unknown", nil
	}

	return string(app.Status.Sync.Status), nil
}

// GetHealthStatus retrieves the health status of an ArgoCD Application
func (c *Client) GetHealthStatus(ctx context.Context, name string) (string, error) {
	app, err := c.GetApplication(ctx, name)
	if err != nil {
		return "", err
	}

	if app.Status.Health.Status == "" {
		return "Unknown", nil
	}

	return string(app.Status.Health.Status), nil
}

// WaitForSync waits for an application to sync (for testing)
func (c *Client) WaitForSync(ctx context.Context, name string) error {
	// This is a simplified version - in production would use watch
	app, err := c.GetApplication(ctx, name)
	if err != nil {
		return err
	}

	if app.Status.Sync.Status == apis.SyncStatusCodeSynced {
		return nil
	}

	return fmt.Errorf("application not yet synced: %s", app.Status.Sync.Status)
}