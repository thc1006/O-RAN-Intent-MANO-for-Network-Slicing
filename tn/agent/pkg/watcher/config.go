package watcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// ConfigWatcher watches for ConfigMap changes containing TN slice configurations
type ConfigWatcher struct {
	kubeClient kubernetes.Interface
	namespace  string
	nodeName   string
	informer   cache.SharedIndexInformer
	channel    chan<- *corev1.ConfigMap
	ctx        context.Context
	mu         sync.RWMutex
}

// NewConfigWatcher creates a new configuration watcher
func NewConfigWatcher(kubeClient kubernetes.Interface, namespace, nodeName string) *ConfigWatcher {
	return &ConfigWatcher{
		kubeClient: kubeClient,
		namespace:  namespace,
		nodeName:   nodeName,
	}
}

// Start starts watching ConfigMap changes with Kubernetes informers
func (w *ConfigWatcher) Start(ctx context.Context, ch chan<- *corev1.ConfigMap) error {
	// Validate inputs
	if w.kubeClient == nil {
		return fmt.Errorf("kubernetes client is nil")
	}
	if w.nodeName == "" {
		return fmt.Errorf("node name cannot be empty")
	}

	w.ctx = ctx
	w.channel = ch

	// Create informer factory with label selector
	labelSelector := fmt.Sprintf("node=%s", w.nodeName)

	// Use namespace if provided, otherwise default to all namespaces
	ns := w.namespace
	if ns == "" {
		ns = corev1.NamespaceAll
	}

	factory := informers.NewSharedInformerFactoryWithOptions(
		w.kubeClient,
		30*time.Second, // Resync period
		informers.WithNamespace(ns),
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.LabelSelector = labelSelector
		}),
	)

	// Get ConfigMap informer
	w.informer = factory.Core().V1().ConfigMaps().Informer()

	// Add event handlers
	_, err := w.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if cm, ok := obj.(*corev1.ConfigMap); ok {
				w.sendConfigMap(cm)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if cm, ok := newObj.(*corev1.ConfigMap); ok {
				w.sendConfigMap(cm)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if cm, ok := obj.(*corev1.ConfigMap); ok {
				// Mark as deleted for downstream processing
				now := metav1.Now()
				cm.DeletionTimestamp = &now
				w.sendConfigMap(cm)
			}
		},
	})
	if err != nil {
		return fmt.Errorf("failed to add event handler: %w", err)
	}

	// Start informer in goroutine
	go w.informer.Run(ctx.Done())

	// Wait for cache sync
	if !cache.WaitForCacheSync(ctx.Done(), w.informer.HasSynced) {
		return fmt.Errorf("failed to sync cache")
	}

	// Wait for context cancellation
	<-ctx.Done()
	return nil
}

// sendConfigMap sends a ConfigMap to the channel without blocking
func (w *ConfigWatcher) sendConfigMap(cm *corev1.ConfigMap) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.channel == nil || w.ctx == nil {
		return
	}

	select {
	case w.channel <- cm:
		// Successfully sent
	case <-w.ctx.Done():
		// Context cancelled, stop sending
		return
	default:
		// Channel full, drop event to avoid blocking
		// In production, consider logging this
	}
}

// ProcessEvent processes watch events manually (for testing)
func (w *ConfigWatcher) ProcessEvent(event watch.Event) {
	if w.channel == nil {
		return
	}

	cm, ok := event.Object.(*corev1.ConfigMap)
	if !ok {
		return
	}

	switch event.Type {
	case watch.Added, watch.Modified:
		w.sendConfigMap(cm)
	case watch.Deleted:
		// Mark as deleted
		now := metav1.Now()
		cm.DeletionTimestamp = &now
		w.sendConfigMap(cm)
	}
}

// FilterByNode filters ConfigMaps by node label
func (w *ConfigWatcher) FilterByNode(configMaps []*corev1.ConfigMap) []*corev1.ConfigMap {
	var filtered []*corev1.ConfigMap

	for _, cm := range configMaps {
		if cm.Labels != nil {
			if nodeLabel, exists := cm.Labels["node"]; exists && nodeLabel == w.nodeName {
				filtered = append(filtered, cm)
			}
		}
	}

	return filtered
}

// Stop stops the watcher (informer stops when context is cancelled)
func (w *ConfigWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	// Informer stops when context is cancelled
	w.channel = nil
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

// WatchConfigurations watches for configuration changes
func (w *ConfigWatcher) WatchConfigurations(ctx context.Context) (<-chan *corev1.ConfigMap, error) {
	ch := make(chan *corev1.ConfigMap, 10)

	go func() {
		_ = w.Start(ctx, ch)
	}()

	return ch, nil
}