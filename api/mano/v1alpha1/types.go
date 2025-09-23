// Package v1alpha1 contains API Schema definitions for the mano v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=mano.o-ran.org
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NetworkSliceIntentSpec defines the desired state of NetworkSliceIntent
type NetworkSliceIntentSpec struct {
	// SliceName is the name of the network slice
	SliceName string `json:"sliceName,omitempty"`

	// Description is a human-readable description of the intent
	Description string `json:"description,omitempty"`

	// QoS defines the quality of service requirements
	QoS QoSParameters `json:"qos,omitempty"`

	// Placement defines the placement requirements
	Placement PlacementRequirements `json:"placement,omitempty"`
}

// QoSParameters defines QoS requirements for a network slice
type QoSParameters struct {
	// Throughput in Mbps
	Throughput float64 `json:"throughput,omitempty"`

	// Latency in ms
	Latency float64 `json:"latency,omitempty"`

	// Reliability percentage
	Reliability float64 `json:"reliability,omitempty"`
}

// PlacementRequirements defines placement constraints
type PlacementRequirements struct {
	// CloudType (e.g., edge, regional, central)
	CloudType string `json:"cloudType,omitempty"`

	// Location constraints
	Location string `json:"location,omitempty"`
}

// NetworkSliceIntentStatus defines the observed state of NetworkSliceIntent
type NetworkSliceIntentStatus struct {
	// Phase represents the current phase of the slice
	Phase string `json:"phase,omitempty"`

	// Conditions represent the current conditions of the slice
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastUpdated is the last time the status was updated
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// NetworkSliceIntent is the Schema for the network slice intents API
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