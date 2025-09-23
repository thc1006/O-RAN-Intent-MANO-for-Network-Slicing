package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TNSliceSpec defines the desired state of TNSlice
type TNSliceSpec struct {
	// SliceID is the unique identifier for this transport network slice
	SliceID string `json:"sliceId"`

	// Bandwidth in Mbps (1-5 for standard profiles)
	// +kubebuilder:validation:Minimum=0.1
	// +kubebuilder:validation:Maximum=10
	Bandwidth float32 `json:"bandwidth"`

	// Latency target in milliseconds
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	Latency float32 `json:"latency"`

	// Jitter in milliseconds (optional)
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=50
	Jitter float32 `json:"jitter,omitempty"`

	// PacketLoss as percentage (0-5)
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=5
	PacketLoss float32 `json:"packetLoss,omitempty"`

	// VxlanID for tunnel identification
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=16777215
	VxlanID int32 `json:"vxlanId"`

	// Endpoints define the source and destination for this slice
	Endpoints []Endpoint `json:"endpoints"`

	// Priority for QoS scheduling (1-10, higher is more important)
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	// +kubebuilder:default=5
	Priority int32 `json:"priority,omitempty"`

	// NodeSelector for agent deployment
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Profile is a predefined QoS profile
	// +kubebuilder:validation:Enum=eMBB;uRLLC;mIoT;custom
	// +optional
	Profile string `json:"profile,omitempty"`
}

// Endpoint represents a network endpoint for the slice
type Endpoint struct {
	// NodeName where the endpoint exists
	NodeName string `json:"nodeName"`

	// IP address of the endpoint
	IP string `json:"ip"`

	// Interface name on the node
	// +kubebuilder:default="eth0"
	Interface string `json:"interface,omitempty"`

	// Role of this endpoint
	// +kubebuilder:validation:Enum=source;destination;transit
	Role string `json:"role"`
}

// TNSliceStatus defines the observed state of TNSlice
type TNSliceStatus struct {
	// Phase of the slice lifecycle
	// +kubebuilder:validation:Enum=Pending;Configuring;Active;Failed;Deleting
	Phase string `json:"phase,omitempty"`

	// ActiveTunnels lists currently established VXLAN tunnels
	ActiveTunnels []TunnelStatus `json:"activeTunnels,omitempty"`

	// MeasuredMetrics contains real-time performance measurements
	MeasuredMetrics *Metrics `json:"measuredMetrics,omitempty"`

	// Conditions represent the latest observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastConfigTime when TC rules were last applied
	LastConfigTime *metav1.Time `json:"lastConfigTime,omitempty"`

	// ConfiguredNodes lists nodes where the slice is configured
	ConfiguredNodes []string `json:"configuredNodes,omitempty"`

	// ObservedGeneration reflects the generation most recently observed
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// TunnelStatus represents the status of a VXLAN tunnel
type TunnelStatus struct {
	// TunnelID is the VXLAN interface name
	TunnelID string `json:"tunnelId"`

	// SourceIP of the tunnel
	SourceIP string `json:"sourceIp"`

	// DestinationIP of the tunnel
	DestinationIP string `json:"destinationIp"`

	// State of the tunnel
	// +kubebuilder:validation:Enum=up;down;configuring
	State string `json:"state"`

	// BytesTransmitted through this tunnel
	BytesTransmitted int64 `json:"bytesTransmitted,omitempty"`

	// BytesReceived through this tunnel
	BytesReceived int64 `json:"bytesReceived,omitempty"`
}

// Metrics contains measured network performance metrics
type Metrics struct {
	// Throughput in Mbps (averaged over last minute)
	Throughput float32 `json:"throughput,omitempty"`

	// Latency in milliseconds (p50)
	LatencyP50 float32 `json:"latencyP50,omitempty"`

	// Latency in milliseconds (p95)
	LatencyP95 float32 `json:"latencyP95,omitempty"`

	// Latency in milliseconds (p99)
	LatencyP99 float32 `json:"latencyP99,omitempty"`

	// JitterMeasured in milliseconds
	JitterMeasured float32 `json:"jitterMeasured,omitempty"`

	// PacketLossMeasured as percentage
	PacketLossMeasured float32 `json:"packetLossMeasured,omitempty"`

	// LastMeasurement timestamp
	LastMeasurement *metav1.Time `json:"lastMeasurement,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories={tn,network,slicing}
// +kubebuilder:printcolumn:name="SliceID",type="string",JSONPath=".spec.sliceId"
// +kubebuilder:printcolumn:name="Bandwidth",type="number",JSONPath=".spec.bandwidth"
// +kubebuilder:printcolumn:name="Latency",type="number",JSONPath=".spec.latency"
// +kubebuilder:printcolumn:name="VxlanID",type="integer",JSONPath=".spec.vxlanId"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TNSlice is the Schema for the tnslices API
type TNSlice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TNSliceSpec   `json:"spec,omitempty"`
	Status TNSliceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TNSliceList contains a list of TNSlice
type TNSliceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TNSlice `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TNSlice{}, &TNSliceList{})
}