package watcher

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ConfigWatcher watches for ConfigMap changes containing TN slice configurations
type ConfigWatcher struct {
	kubeClient kubernetes.Interface
	namespace  string
	nodeName   string
}

// NewConfigWatcher creates a new configuration watcher
func NewConfigWatcher(kubeClient kubernetes.Interface, namespace, nodeName string) *ConfigWatcher {
	return &ConfigWatcher{
		kubeClient: kubeClient,
		namespace:  namespace,
		nodeName:   nodeName,
	}
}

// GetConfigurations retrieves all configurations for this node
func (w *ConfigWatcher) GetConfigurations(ctx context.Context) (map[string]string, error) {
	configs := make(map[string]string)

	// List ConfigMaps with appropriate labels
	labelSelector := fmt.Sprintf("node=%s", w.nodeName)
	configMaps, err := w.kubeClient.CoreV1().ConfigMaps(w.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list configmaps: %w", err)
	}

	// Extract configuration data
	for _, cm := range configMaps.Items {
		if data, ok := cm.Data["config.json"]; ok {
			configs[cm.Name] = data
		}
	}

	return configs, nil
}

// WatchConfigurations watches for configuration changes (not implemented for simplicity)
func (w *ConfigWatcher) WatchConfigurations(ctx context.Context) (<-chan *corev1.ConfigMap, error) {
	// TODO: Implement watch functionality using informers
	return nil, fmt.Errorf("not implemented")
}