package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn/agent/pkg/watcher"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

// TestConfigWatcherStart tests the ConfigWatcher start mechanism
func TestConfigWatcherStart(t *testing.T) {
	tests := []struct {
		name      string
		nodeName  string
		timeout   time.Duration
		wantErr   bool
		errMsg    string
	}{
		{
			name:     "successful start",
			nodeName: "test-node",
			timeout:  5 * time.Second,
			wantErr:  false,
		},
		{
			name:     "empty node name",
			nodeName: "",
			timeout:  5 * time.Second,
			wantErr:  true,
			errMsg:   "node name cannot be empty",
		},
		{
			name:     "context cancellation",
			nodeName: "test-node",
			timeout:  100 * time.Millisecond,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			client := fake.NewSimpleClientset()
			w := watcher.NewConfigWatcher(client, "default", tt.nodeName)
			ch := make(chan *corev1.ConfigMap, 10)

			err := w.Start(ctx, ch)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				// For successful start, we expect graceful shutdown
				// The error may be nil or context.Canceled
				if err != nil {
					assert.ErrorIs(t, err, context.Canceled)
				}
			}
		})
	}
}

// TestConfigWatcherProcessUpdates tests ConfigMap event processing
func TestConfigWatcherProcessUpdates(t *testing.T) {
	tests := []struct {
		name       string
		events     []watch.Event
		wantCount  int
		wantValues []string
	}{
		{
			name: "single ConfigMap added",
			events: []watch.Event{
				{
					Type: watch.Added,
					Object: &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "config1",
							Namespace: "default",
						},
						Data: map[string]string{
							"key1": "value1",
						},
					},
				},
			},
			wantCount:  1,
			wantValues: []string{"value1"},
		},
		{
			name: "ConfigMap modified",
			events: []watch.Event{
				{
					Type: watch.Added,
					Object: &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name: "config1",
						},
						Data: map[string]string{"key": "original"},
					},
				},
				{
					Type: watch.Modified,
					Object: &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name: "config1",
						},
						Data: map[string]string{"key": "modified"},
					},
				},
			},
			wantCount:  2,
			wantValues: []string{"original", "modified"},
		},
		{
			name: "ConfigMap deleted",
			events: []watch.Event{
				{
					Type: watch.Added,
					Object: &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{Name: "config1"},
					},
				},
				{
					Type: watch.Deleted,
					Object: &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{Name: "config1"},
					},
				},
			},
			wantCount:  2,
			wantValues: []string{},
		},
		{
			name: "multiple ConfigMaps",
			events: []watch.Event{
				{
					Type:   watch.Added,
					Object: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "cm1"}},
				},
				{
					Type:   watch.Added,
					Object: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "cm2"}},
				},
				{
					Type:   watch.Added,
					Object: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "cm3"}},
				},
			},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			client := fake.NewSimpleClientset()
			w := watcher.NewConfigWatcher(client, "default", "test-node")
			ch := make(chan *corev1.ConfigMap, 10)

			// Start watcher in background
			go func() {
				_ = w.Start(ctx, ch)
			}()

			// Simulate events
			go func() {
				time.Sleep(100 * time.Millisecond)
				for _, event := range tt.events {
					w.ProcessEvent(event)
				}
			}()

			// Collect results
			var received []*corev1.ConfigMap
			timeout := time.After(1 * time.Second)

		collectLoop:
			for {
				select {
				case cm := <-ch:
					received = append(received, cm)
					if len(received) >= tt.wantCount {
						break collectLoop
					}
				case <-timeout:
					break collectLoop
				}
			}

			assert.Equal(t, tt.wantCount, len(received))

			// Verify values if specified
			for i, wantVal := range tt.wantValues {
				if i < len(received) && len(received[i].Data) > 0 {
					found := false
					for _, v := range received[i].Data {
						if v == wantVal {
							found = true
							break
						}
					}
					assert.True(t, found, "expected value %s not found", wantVal)
				}
			}
		})
	}
}

// TestConfigWatcherContextCancellation tests graceful shutdown
func TestConfigWatcherContextCancellation(t *testing.T) {
	tests := []struct {
		name         string
		cancelAfter  time.Duration
		expectClean  bool
	}{
		{
			name:        "immediate cancellation",
			cancelAfter: 10 * time.Millisecond,
			expectClean: true,
		},
		{
			name:        "delayed cancellation",
			cancelAfter: 500 * time.Millisecond,
			expectClean: true,
		},
		{
			name:        "long running cancellation",
			cancelAfter: 2 * time.Second,
			expectClean: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())

			client := fake.NewSimpleClientset()
			w := watcher.NewConfigWatcher(client, "default", "test-node")
			ch := make(chan *corev1.ConfigMap, 10)

			done := make(chan error, 1)
			go func() {
				done <- w.Start(ctx, ch)
			}()

			// Cancel after specified duration
			time.Sleep(tt.cancelAfter)
			cancel()

			// Wait for cleanup
			select {
			case err := <-done:
				if tt.expectClean {
					// Should either be nil, context.Canceled, or failed to sync cache
					if err != nil {
						// Both context.Canceled and "failed to sync cache" are acceptable
						// when context is cancelled during initialization
						isContextCanceled := errors.Is(err, context.Canceled)
						isCacheSyncFailure := err.Error() == "failed to sync cache"
						assert.True(t, isContextCanceled || isCacheSyncFailure,
							"expected context.Canceled or 'failed to sync cache', got: %v", err)
					}
				}
			case <-time.After(3 * time.Second):
				t.Fatal("watcher did not shutdown cleanly")
			}

			// Verify channel is properly closed or drained
			select {
			case _, ok := <-ch:
				if ok {
					t.Log("channel still has data, but that's acceptable")
				}
			default:
				// Channel is empty or closed
			}
		})
	}
}

// TestConfigWatcherFilterByNode tests node-specific filtering
func TestConfigWatcherFilterByNode(t *testing.T) {
	tests := []struct {
		name       string
		nodeName   string
		configMaps []*corev1.ConfigMap
		wantCount  int
	}{
		{
			name:     "filter by node label",
			nodeName: "node-1",
			configMaps: []*corev1.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cm1",
						Labels: map[string]string{
							"node": "node-1",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cm2",
						Labels: map[string]string{
							"node": "node-2",
						},
					},
				},
			},
			wantCount: 1,
		},
		{
			name:     "no matching nodes",
			nodeName: "node-3",
			configMaps: []*corev1.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cm1",
						Labels: map[string]string{"node": "node-1"},
					},
				},
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewSimpleClientset()
			w := watcher.NewConfigWatcher(client, "default", tt.nodeName)

			filtered := w.FilterByNode(tt.configMaps)
			assert.Equal(t, tt.wantCount, len(filtered))
		})
	}
}

// TestConfigWatcherErrorHandling tests error scenarios
func TestConfigWatcherErrorHandling(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(kubernetes.Interface) kubernetes.Interface
		wantErr bool
		errMsg  string
	}{
		{
			name: "client error",
			setup: func(k kubernetes.Interface) kubernetes.Interface {
				return nil // nil client should cause error
			},
			wantErr: true,
			errMsg:  "kubernetes client is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client kubernetes.Interface
			client = fake.NewSimpleClientset()
			if tt.setup != nil {
				client = tt.setup(client)
			}

			w := watcher.NewConfigWatcher(client, "default", "test-node")
			ch := make(chan *corev1.ConfigMap, 10)

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			err := w.Start(ctx, ch)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// Test helper functions removed - using actual implementation from watcher package