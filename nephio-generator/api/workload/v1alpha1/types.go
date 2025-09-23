package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// WorkloadIdentity represents a Nephio workload identity
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={nephio,workload}
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status",description="Ready status"
// +kubebuilder:printcolumn:name="VNF Type",type="string",JSONPath=".spec.vnfType",description="VNF Type"
// +kubebuilder:printcolumn:name="Cloud Type",type="string",JSONPath=".spec.placement.cloudType",description="Cloud Type"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type WorkloadIdentity struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkloadIdentitySpec   `json:"spec,omitempty"`
	Status WorkloadIdentityStatus `json:"status,omitempty"`
}

// WorkloadIdentitySpec defines the desired state of WorkloadIdentity
type WorkloadIdentitySpec struct {
	// VNFType specifies the type of VNF (RAN, CN, TN)
	// +kubebuilder:validation:Enum=RAN;CN;TN
	VNFType string `json:"vnfType"`

	// Version specifies the VNF version
	Version string `json:"version"`

	// Placement defines where the workload should be placed
	Placement PlacementSpec `json:"placement"`

	// QoS defines quality of service requirements
	QoS QoSSpec `json:"qos"`

	// Resources defines resource requirements
	Resources ResourceSpec `json:"resources"`

	// Image defines container image configuration
	Image ImageSpec `json:"image"`

	// Configuration defines VNF-specific configuration
	Configuration map[string]string `json:"configuration,omitempty"`

	// Dependencies defines workload dependencies
	Dependencies []WorkloadDependency `json:"dependencies,omitempty"`

	// NetworkInterfaces defines network interface requirements
	NetworkInterfaces []NetworkInterface `json:"networkInterfaces,omitempty"`

	// SecurityPolicy defines security policy requirements
	SecurityPolicy *SecurityPolicy `json:"securityPolicy,omitempty"`

	// Scaling defines scaling parameters
	Scaling *ScalingSpec `json:"scaling,omitempty"`

	// Lifecycle defines lifecycle management
	Lifecycle *LifecycleSpec `json:"lifecycle,omitempty"`
}

// PlacementSpec defines placement requirements
type PlacementSpec struct {
	// CloudType specifies the cloud deployment type (edge, regional, central)
	// +kubebuilder:validation:Enum=edge;regional;central;multi-cloud
	CloudType string `json:"cloudType"`

	// Region specifies the geographic region
	Region string `json:"region,omitempty"`

	// Zone specifies the availability zone
	Zone string `json:"zone,omitempty"`

	// Site specifies the specific site
	Site string `json:"site,omitempty"`

	// ClusterSelector defines cluster selection criteria
	ClusterSelector *ClusterSelector `json:"clusterSelector,omitempty"`

	// NodeSelector defines node selection criteria
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Affinity defines pod affinity rules
	Affinity *AffinitySpec `json:"affinity,omitempty"`

	// Tolerations defines pod tolerations
	Tolerations []TolerationSpec `json:"tolerations,omitempty"`

	// TopologySpreadConstraints defines topology spread constraints
	TopologySpreadConstraints []TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
}

// QoSSpec defines quality of service requirements
type QoSSpec struct {
	// Bandwidth specifies required bandwidth in Mbps
	// +kubebuilder:validation:Minimum=0
	Bandwidth float64 `json:"bandwidth"`

	// Latency specifies maximum allowed latency in milliseconds
	// +kubebuilder:validation:Minimum=0
	Latency float64 `json:"latency"`

	// Jitter specifies maximum allowed jitter in milliseconds
	// +kubebuilder:validation:Minimum=0
	Jitter *float64 `json:"jitter,omitempty"`

	// PacketLoss specifies maximum allowed packet loss as a percentage
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	PacketLoss *float64 `json:"packetLoss,omitempty"`

	// Reliability specifies required reliability as a percentage
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Reliability *float64 `json:"reliability,omitempty"`

	// SliceType specifies the network slice type
	// +kubebuilder:validation:Enum=eMBB;URLLC;mMTC
	SliceType string `json:"sliceType,omitempty"`

	// Priority specifies QoS priority (1-9, where 1 is highest)
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=9
	Priority *int32 `json:"priority,omitempty"`

	// TrafficClass specifies the traffic class
	TrafficClass string `json:"trafficClass,omitempty"`
}

// ResourceSpec defines resource requirements
type ResourceSpec struct {
	// CPUCores specifies required CPU cores
	// +kubebuilder:validation:Minimum=0
	CPUCores int32 `json:"cpuCores"`

	// MemoryGB specifies required memory in GB
	// +kubebuilder:validation:Minimum=0
	MemoryGB int32 `json:"memoryGB"`

	// StorageGB specifies required storage in GB
	// +kubebuilder:validation:Minimum=0
	StorageGB int32 `json:"storageGB,omitempty"`

	// GPUType specifies GPU type if required
	GPUType string `json:"gpuType,omitempty"`

	// GPUCount specifies number of GPUs required
	// +kubebuilder:validation:Minimum=0
	GPUCount int32 `json:"gpuCount,omitempty"`

	// HugePagesSize specifies huge pages size
	HugePagesSize string `json:"hugePagesSize,omitempty"`

	// HugePagesCount specifies number of huge pages
	// +kubebuilder:validation:Minimum=0
	HugePagesCount int32 `json:"hugePagesCount,omitempty"`

	// SRIOVNetworkDevices specifies SR-IOV network device requirements
	SRIOVNetworkDevices []SRIOVNetworkDevice `json:"sriovNetworkDevices,omitempty"`
}

// ImageSpec defines container image configuration
type ImageSpec struct {
	// Repository specifies the image repository
	Repository string `json:"repository"`

	// Tag specifies the image tag
	Tag string `json:"tag"`

	// PullPolicy specifies the image pull policy
	// +kubebuilder:validation:Enum=Always;Never;IfNotPresent
	PullPolicy string `json:"pullPolicy,omitempty"`

	// PullSecrets specifies image pull secrets
	PullSecrets []string `json:"pullSecrets,omitempty"`

	// Digest specifies the image digest for immutable references
	Digest string `json:"digest,omitempty"`
}

// WorkloadDependency defines a dependency on another workload
type WorkloadDependency struct {
	// Name specifies the dependency name
	Name string `json:"name"`

	// Type specifies the dependency type
	// +kubebuilder:validation:Enum=workload;service;database;message-queue;storage
	Type string `json:"type"`

	// Version specifies the required version
	Version string `json:"version,omitempty"`

	// Namespace specifies the dependency namespace
	Namespace string `json:"namespace,omitempty"`

	// Optional specifies if the dependency is optional
	Optional bool `json:"optional,omitempty"`

	// HealthCheck defines health check for the dependency
	HealthCheck *HealthCheckSpec `json:"healthCheck,omitempty"`
}

// NetworkInterface defines network interface requirements
type NetworkInterface struct {
	// Name specifies the interface name
	Name string `json:"name"`

	// Type specifies the interface type
	// +kubebuilder:validation:Enum=eth;sriov;dpdk;macvlan;ipvlan
	Type string `json:"type"`

	// NetworkName specifies the network name
	NetworkName string `json:"networkName"`

	// IPAddress specifies a static IP address
	IPAddress string `json:"ipAddress,omitempty"`

	// VLAN specifies VLAN ID
	VLAN *int32 `json:"vlan,omitempty"`

	// Bandwidth specifies interface bandwidth in Mbps
	Bandwidth *float64 `json:"bandwidth,omitempty"`

	// MTU specifies maximum transmission unit
	MTU *int32 `json:"mtu,omitempty"`

	// SecurityGroups specifies security groups
	SecurityGroups []string `json:"securityGroups,omitempty"`
}

// SecurityPolicy defines security policy requirements
type SecurityPolicy struct {
	// RunAsNonRoot specifies if containers should run as non-root
	RunAsNonRoot *bool `json:"runAsNonRoot,omitempty"`

	// RunAsUser specifies the user ID to run containers
	RunAsUser *int64 `json:"runAsUser,omitempty"`

	// RunAsGroup specifies the group ID to run containers
	RunAsGroup *int64 `json:"runAsGroup,omitempty"`

	// FSGroup specifies the filesystem group ID
	FSGroup *int64 `json:"fsGroup,omitempty"`

	// AllowPrivilegeEscalation specifies if privilege escalation is allowed
	AllowPrivilegeEscalation *bool `json:"allowPrivilegeEscalation,omitempty"`

	// Privileged specifies if containers should run as privileged
	Privileged *bool `json:"privileged,omitempty"`

	// ReadOnlyRootFilesystem specifies if root filesystem should be read-only
	ReadOnlyRootFilesystem *bool `json:"readOnlyRootFilesystem,omitempty"`

	// Capabilities defines Linux capabilities
	Capabilities *CapabilitiesSpec `json:"capabilities,omitempty"`

	// SELinuxOptions defines SELinux options
	SELinuxOptions *SELinuxOptionsSpec `json:"selinuxOptions,omitempty"`

	// AppArmorProfile defines AppArmor profile
	AppArmorProfile string `json:"apparmorProfile,omitempty"`

	// SeccompProfile defines Seccomp profile
	SeccompProfile string `json:"seccompProfile,omitempty"`
}

// ScalingSpec defines scaling parameters
type ScalingSpec struct {
	// MinReplicas specifies minimum number of replicas
	// +kubebuilder:validation:Minimum=0
	MinReplicas *int32 `json:"minReplicas,omitempty"`

	// MaxReplicas specifies maximum number of replicas
	// +kubebuilder:validation:Minimum=1
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`

	// AutoScaling enables auto-scaling
	AutoScaling *AutoScalingSpec `json:"autoScaling,omitempty"`

	// ScalingPolicy defines scaling policies
	ScalingPolicy *ScalingPolicy `json:"scalingPolicy,omitempty"`
}

// LifecycleSpec defines lifecycle management
type LifecycleSpec struct {
	// PreStart defines pre-start hooks
	PreStart *LifecycleHook `json:"preStart,omitempty"`

	// PostStart defines post-start hooks
	PostStart *LifecycleHook `json:"postStart,omitempty"`

	// PreStop defines pre-stop hooks
	PreStop *LifecycleHook `json:"preStop,omitempty"`

	// TerminationGracePeriodSeconds specifies termination grace period
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`

	// RestartPolicy specifies restart policy
	// +kubebuilder:validation:Enum=Always;OnFailure;Never
	RestartPolicy string `json:"restartPolicy,omitempty"`
}

// Supporting types

// ClusterSelector defines cluster selection criteria
type ClusterSelector struct {
	// MatchLabels specifies labels to match
	MatchLabels map[string]string `json:"matchLabels,omitempty"`

	// MatchExpressions specifies label expressions to match
	MatchExpressions []ClusterSelectorRequirement `json:"matchExpressions,omitempty"`
}

// ClusterSelectorRequirement defines a cluster selector requirement
type ClusterSelectorRequirement struct {
	// Key specifies the label key
	Key string `json:"key"`

	// Operator specifies the operator
	// +kubebuilder:validation:Enum=In;NotIn;Exists;DoesNotExist
	Operator string `json:"operator"`

	// Values specifies the values
	Values []string `json:"values,omitempty"`
}

// AffinitySpec defines affinity specifications
type AffinitySpec struct {
	// NodeAffinity defines node affinity
	NodeAffinity *NodeAffinitySpec `json:"nodeAffinity,omitempty"`

	// PodAffinity defines pod affinity
	PodAffinity *PodAffinitySpec `json:"podAffinity,omitempty"`

	// PodAntiAffinity defines pod anti-affinity
	PodAntiAffinity *PodAffinitySpec `json:"podAntiAffinity,omitempty"`
}

// NodeAffinitySpec defines node affinity
type NodeAffinitySpec struct {
	// RequiredDuringSchedulingIgnoredDuringExecution specifies required node affinity
	RequiredDuringSchedulingIgnoredDuringExecution *NodeSelector `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`

	// PreferredDuringSchedulingIgnoredDuringExecution specifies preferred node affinity
	PreferredDuringSchedulingIgnoredDuringExecution []PreferredSchedulingTerm `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

// PodAffinitySpec defines pod affinity
type PodAffinitySpec struct {
	// RequiredDuringSchedulingIgnoredDuringExecution specifies required pod affinity
	RequiredDuringSchedulingIgnoredDuringExecution []PodAffinityTerm `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`

	// PreferredDuringSchedulingIgnoredDuringExecution specifies preferred pod affinity
	PreferredDuringSchedulingIgnoredDuringExecution []WeightedPodAffinityTerm `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

// TolerationSpec defines pod tolerations
type TolerationSpec struct {
	// Key specifies the toleration key
	Key string `json:"key,omitempty"`

	// Operator specifies the toleration operator
	// +kubebuilder:validation:Enum=Equal;Exists
	Operator string `json:"operator,omitempty"`

	// Value specifies the toleration value
	Value string `json:"value,omitempty"`

	// Effect specifies the toleration effect
	// +kubebuilder:validation:Enum=NoSchedule;PreferNoSchedule;NoExecute
	Effect string `json:"effect,omitempty"`

	// TolerationSeconds specifies toleration seconds
	TolerationSeconds *int64 `json:"tolerationSeconds,omitempty"`
}

// TopologySpreadConstraint defines topology spread constraints
type TopologySpreadConstraint struct {
	// MaxSkew specifies maximum skew
	MaxSkew int32 `json:"maxSkew"`

	// TopologyKey specifies topology key
	TopologyKey string `json:"topologyKey"`

	// WhenUnsatisfiable specifies behavior when unsatisfiable
	// +kubebuilder:validation:Enum=DoNotSchedule;ScheduleAnyway
	WhenUnsatisfiable string `json:"whenUnsatisfiable"`

	// LabelSelector specifies label selector
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
}

// SRIOVNetworkDevice defines SR-IOV network device requirements
type SRIOVNetworkDevice struct {
	// ResourceName specifies the SR-IOV resource name
	ResourceName string `json:"resourceName"`

	// Count specifies the number of devices required
	// +kubebuilder:validation:Minimum=1
	Count int32 `json:"count"`

	// DeviceType specifies the device type
	DeviceType string `json:"deviceType,omitempty"`

	// Vendor specifies the vendor ID
	Vendor string `json:"vendor,omitempty"`

	// Device specifies the device ID
	Device string `json:"device,omitempty"`
}

// HealthCheckSpec defines health check configuration
type HealthCheckSpec struct {
	// Type specifies the health check type
	// +kubebuilder:validation:Enum=http;tcp;grpc;exec;custom
	Type string `json:"type"`

	// HTTPGet defines HTTP health check
	HTTPGet *HTTPGetAction `json:"httpGet,omitempty"`

	// TCPSocket defines TCP health check
	TCPSocket *TCPSocketAction `json:"tcpSocket,omitempty"`

	// Exec defines exec health check
	Exec *ExecAction `json:"exec,omitempty"`

	// InitialDelaySeconds specifies initial delay
	InitialDelaySeconds *int32 `json:"initialDelaySeconds,omitempty"`

	// PeriodSeconds specifies check period
	PeriodSeconds *int32 `json:"periodSeconds,omitempty"`

	// TimeoutSeconds specifies check timeout
	TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty"`

	// SuccessThreshold specifies success threshold
	SuccessThreshold *int32 `json:"successThreshold,omitempty"`

	// FailureThreshold specifies failure threshold
	FailureThreshold *int32 `json:"failureThreshold,omitempty"`
}

// AutoScalingSpec defines auto-scaling configuration
type AutoScalingSpec struct {
	// Enabled specifies if auto-scaling is enabled
	Enabled bool `json:"enabled"`

	// Metrics specifies scaling metrics
	Metrics []ScalingMetric `json:"metrics,omitempty"`

	// Behavior specifies scaling behavior
	Behavior *ScalingBehavior `json:"behavior,omitempty"`
}

// ScalingPolicy defines scaling policies
type ScalingPolicy struct {
	// ScaleUp defines scale-up policy
	ScaleUp *ScalingRules `json:"scaleUp,omitempty"`

	// ScaleDown defines scale-down policy
	ScaleDown *ScalingRules `json:"scaleDown,omitempty"`
}

// ScalingMetric defines a scaling metric
type ScalingMetric struct {
	// Type specifies the metric type
	// +kubebuilder:validation:Enum=Resource;Pods;Object;External
	Type string `json:"type"`

	// Resource defines resource metric
	Resource *ResourceMetricSource `json:"resource,omitempty"`

	// Pods defines pods metric
	Pods *PodsMetricSource `json:"pods,omitempty"`

	// Object defines object metric
	Object *ObjectMetricSource `json:"object,omitempty"`

	// External defines external metric
	External *ExternalMetricSource `json:"external,omitempty"`
}

// LifecycleHook defines lifecycle hook
type LifecycleHook struct {
	// Exec defines exec hook
	Exec *ExecAction `json:"exec,omitempty"`

	// HTTPGet defines HTTP hook
	HTTPGet *HTTPGetAction `json:"httpGet,omitempty"`
}

// CapabilitiesSpec defines Linux capabilities
type CapabilitiesSpec struct {
	// Add specifies capabilities to add
	Add []string `json:"add,omitempty"`

	// Drop specifies capabilities to drop
	Drop []string `json:"drop,omitempty"`
}

// SELinuxOptionsSpec defines SELinux options
type SELinuxOptionsSpec struct {
	// User specifies SELinux user
	User string `json:"user,omitempty"`

	// Type specifies SELinux type
	Type string `json:"type,omitempty"`

	// Level specifies SELinux level
	Level string `json:"level,omitempty"`

	// Role specifies SELinux role
	Role string `json:"role,omitempty"`
}

// Action types for health checks and lifecycle hooks
type HTTPGetAction struct {
	// Path specifies the HTTP path
	Path string `json:"path,omitempty"`

	// Port specifies the port
	Port int32 `json:"port"`

	// Host specifies the host
	Host string `json:"host,omitempty"`

	// Scheme specifies the scheme (HTTP or HTTPS)
	// +kubebuilder:validation:Enum=HTTP;HTTPS
	Scheme string `json:"scheme,omitempty"`

	// HTTPHeaders specifies HTTP headers
	HTTPHeaders []HTTPHeader `json:"httpHeaders,omitempty"`
}

// TCPSocketAction defines TCP socket action
type TCPSocketAction struct {
	// Port specifies the port
	Port int32 `json:"port"`

	// Host specifies the host
	Host string `json:"host,omitempty"`
}

// ExecAction defines exec action
type ExecAction struct {
	// Command specifies the command to execute
	Command []string `json:"command,omitempty"`
}

// HTTPHeader defines HTTP header
type HTTPHeader struct {
	// Name specifies the header name
	Name string `json:"name"`

	// Value specifies the header value
	Value string `json:"value"`
}

// Metric source types for auto-scaling
type ResourceMetricSource struct {
	// Name specifies the resource name
	Name string `json:"name"`

	// Target specifies the target value
	Target MetricTarget `json:"target"`
}

type PodsMetricSource struct {
	// Metric specifies the metric
	Metric MetricIdentifier `json:"metric"`

	// Target specifies the target value
	Target MetricTarget `json:"target"`
}

type ObjectMetricSource struct {
	// DescribedObject specifies the object
	DescribedObject CrossVersionObjectReference `json:"describedObject"`

	// Metric specifies the metric
	Metric MetricIdentifier `json:"metric"`

	// Target specifies the target value
	Target MetricTarget `json:"target"`
}

type ExternalMetricSource struct {
	// Metric specifies the metric
	Metric MetricIdentifier `json:"metric"`

	// Target specifies the target value
	Target MetricTarget `json:"target"`
}

// Supporting types for metrics
type MetricTarget struct {
	// Type specifies the target type
	// +kubebuilder:validation:Enum=Utilization;Value;AverageValue
	Type string `json:"type"`

	// Value specifies the target value
	Value *string `json:"value,omitempty"`

	// AverageValue specifies the average target value
	AverageValue *string `json:"averageValue,omitempty"`

	// AverageUtilization specifies the average utilization percentage
	AverageUtilization *int32 `json:"averageUtilization,omitempty"`
}

type MetricIdentifier struct {
	// Name specifies the metric name
	Name string `json:"name"`

	// Selector specifies the metric selector
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

type CrossVersionObjectReference struct {
	// Kind specifies the object kind
	Kind string `json:"kind"`

	// Name specifies the object name
	Name string `json:"name"`

	// APIVersion specifies the API version
	APIVersion string `json:"apiVersion,omitempty"`
}

// Scaling behavior and rules
type ScalingBehavior struct {
	// ScaleUp defines scale-up behavior
	ScaleUp *HPAScalingRules `json:"scaleUp,omitempty"`

	// ScaleDown defines scale-down behavior
	ScaleDown *HPAScalingRules `json:"scaleDown,omitempty"`
}

type HPAScalingRules struct {
	// StabilizationWindowSeconds specifies stabilization window
	StabilizationWindowSeconds *int32 `json:"stabilizationWindowSeconds,omitempty"`

	// SelectPolicy specifies the selection policy
	// +kubebuilder:validation:Enum=Min;Max;Disabled
	SelectPolicy *string `json:"selectPolicy,omitempty"`

	// Policies specifies scaling policies
	Policies []HPAScalingPolicy `json:"policies,omitempty"`
}

type HPAScalingPolicy struct {
	// Type specifies the policy type
	// +kubebuilder:validation:Enum=Pods;Percent
	Type string `json:"type"`

	// Value specifies the policy value
	Value int32 `json:"value"`

	// PeriodSeconds specifies the period
	PeriodSeconds int32 `json:"periodSeconds"`
}

type ScalingRules struct {
	// Policies specifies scaling policies
	Policies []ScalingPolicyRule `json:"policies,omitempty"`

	// StabilizationWindowSeconds specifies stabilization window
	StabilizationWindowSeconds *int32 `json:"stabilizationWindowSeconds,omitempty"`
}

type ScalingPolicyRule struct {
	// Type specifies the policy type
	// +kubebuilder:validation:Enum=Pods;Percent
	Type string `json:"type"`

	// Value specifies the policy value
	Value int32 `json:"value"`

	// PeriodSeconds specifies the period
	PeriodSeconds int32 `json:"periodSeconds"`
}

// Affinity supporting types
type NodeSelector struct {
	// NodeSelectorTerms specifies node selector terms
	NodeSelectorTerms []NodeSelectorTerm `json:"nodeSelectorTerms"`
}

type NodeSelectorTerm struct {
	// MatchExpressions specifies match expressions
	MatchExpressions []NodeSelectorRequirement `json:"matchExpressions,omitempty"`

	// MatchFields specifies match fields
	MatchFields []NodeSelectorRequirement `json:"matchFields,omitempty"`
}

type NodeSelectorRequirement struct {
	// Key specifies the selector key
	Key string `json:"key"`

	// Operator specifies the operator
	// +kubebuilder:validation:Enum=In;NotIn;Exists;DoesNotExist;Gt;Lt
	Operator string `json:"operator"`

	// Values specifies the values
	Values []string `json:"values,omitempty"`
}

type PreferredSchedulingTerm struct {
	// Weight specifies the weight
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	Weight int32 `json:"weight"`

	// Preference specifies the node selector term
	Preference NodeSelectorTerm `json:"preference"`
}

type PodAffinityTerm struct {
	// LabelSelector specifies the label selector
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`

	// Namespaces specifies the namespaces
	Namespaces []string `json:"namespaces,omitempty"`

	// TopologyKey specifies the topology key
	TopologyKey string `json:"topologyKey"`
}

type WeightedPodAffinityTerm struct {
	// Weight specifies the weight
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	Weight int32 `json:"weight"`

	// PodAffinityTerm specifies the pod affinity term
	PodAffinityTerm PodAffinityTerm `json:"podAffinityTerm"`
}

// WorkloadIdentityStatus defines the observed state of WorkloadIdentity
type WorkloadIdentityStatus struct {
	// Conditions represent the latest available observations of the workload's current state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Phase represents the current phase of the workload
	// +kubebuilder:validation:Enum=Pending;Scheduling;Running;Succeeded;Failed;Unknown
	Phase string `json:"phase,omitempty"`

	// Replicas represents the current number of replicas
	Replicas int32 `json:"replicas,omitempty"`

	// ReadyReplicas represents the number of ready replicas
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// AvailableReplicas represents the number of available replicas
	AvailableReplicas int32 `json:"availableReplicas,omitempty"`

	// PlacementStatus represents placement status
	PlacementStatus *PlacementStatus `json:"placementStatus,omitempty"`

	// QoSStatus represents QoS status
	QoSStatus *QoSStatus `json:"qosStatus,omitempty"`

	// ResourceStatus represents resource status
	ResourceStatus *ResourceStatus `json:"resourceStatus,omitempty"`

	// NetworkStatus represents network status
	NetworkStatus *NetworkStatus `json:"networkStatus,omitempty"`

	// LastUpdated represents the last update timestamp
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`

	// ObservedGeneration represents the observed generation
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// PlacementStatus represents placement status
type PlacementStatus struct {
	// PlacedCluster represents the cluster where workload is placed
	PlacedCluster string `json:"placedCluster,omitempty"`

	// PlacedNodes represents the nodes where workload is placed
	PlacedNodes []string `json:"placedNodes,omitempty"`

	// SchedulingAttempts represents the number of scheduling attempts
	SchedulingAttempts int32 `json:"schedulingAttempts,omitempty"`

	// LastScheduleTime represents the last schedule time
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`

	// SchedulingErrors represents scheduling errors
	SchedulingErrors []string `json:"schedulingErrors,omitempty"`
}

// QoSStatus represents QoS status
type QoSStatus struct {
	// AllocatedBandwidth represents allocated bandwidth
	AllocatedBandwidth *float64 `json:"allocatedBandwidth,omitempty"`

	// MeasuredLatency represents measured latency
	MeasuredLatency *float64 `json:"measuredLatency,omitempty"`

	// MeasuredJitter represents measured jitter
	MeasuredJitter *float64 `json:"measuredJitter,omitempty"`

	// MeasuredPacketLoss represents measured packet loss
	MeasuredPacketLoss *float64 `json:"measuredPacketLoss,omitempty"`

	// QoSClass represents the assigned QoS class
	QoSClass string `json:"qosClass,omitempty"`

	// QoSViolations represents QoS violations
	QoSViolations []QoSViolation `json:"qosViolations,omitempty"`
}

// QoSViolation represents a QoS violation
type QoSViolation struct {
	// Type represents the violation type
	Type string `json:"type"`

	// Threshold represents the threshold that was violated
	Threshold float64 `json:"threshold"`

	// ActualValue represents the actual measured value
	ActualValue float64 `json:"actualValue"`

	// Timestamp represents when the violation occurred
	Timestamp metav1.Time `json:"timestamp"`

	// Duration represents how long the violation lasted
	Duration *metav1.Duration `json:"duration,omitempty"`
}

// ResourceStatus represents resource status
type ResourceStatus struct {
	// AllocatedCPU represents allocated CPU
	AllocatedCPU *string `json:"allocatedCPU,omitempty"`

	// AllocatedMemory represents allocated memory
	AllocatedMemory *string `json:"allocatedMemory,omitempty"`

	// AllocatedStorage represents allocated storage
	AllocatedStorage *string `json:"allocatedStorage,omitempty"`

	// AllocatedGPU represents allocated GPU
	AllocatedGPU *int32 `json:"allocatedGPU,omitempty"`

	// UsedCPU represents used CPU
	UsedCPU *string `json:"usedCPU,omitempty"`

	// UsedMemory represents used memory
	UsedMemory *string `json:"usedMemory,omitempty"`

	// UsedStorage represents used storage
	UsedStorage *string `json:"usedStorage,omitempty"`

	// ResourceConstraints represents resource constraints
	ResourceConstraints []ResourceConstraint `json:"resourceConstraints,omitempty"`
}

// ResourceConstraint represents a resource constraint
type ResourceConstraint struct {
	// Type represents the constraint type
	Type string `json:"type"`

	// Resource represents the constrained resource
	Resource string `json:"resource"`

	// Message represents the constraint message
	Message string `json:"message"`

	// Timestamp represents when the constraint was detected
	Timestamp metav1.Time `json:"timestamp"`
}

// NetworkStatus represents network status
type NetworkStatus struct {
	// AssignedIPs represents assigned IP addresses
	AssignedIPs []AssignedIP `json:"assignedIPs,omitempty"`

	// NetworkInterfaces represents network interface status
	NetworkInterfaces []NetworkInterfaceStatus `json:"networkInterfaces,omitempty"`

	// NetworkPolicies represents network policy status
	NetworkPolicies []NetworkPolicyStatus `json:"networkPolicies,omitempty"`
}

// AssignedIP represents an assigned IP address
type AssignedIP struct {
	// Interface represents the network interface
	Interface string `json:"interface"`

	// IPAddress represents the assigned IP address
	IPAddress string `json:"ipAddress"`

	// NetworkName represents the network name
	NetworkName string `json:"networkName"`
}

// NetworkInterfaceStatus represents network interface status
type NetworkInterfaceStatus struct {
	// Name represents the interface name
	Name string `json:"name"`

	// Type represents the interface type
	Type string `json:"type"`

	// Status represents the interface status
	// +kubebuilder:validation:Enum=Up;Down;Unknown
	Status string `json:"status"`

	// IPAddresses represents assigned IP addresses
	IPAddresses []string `json:"ipAddresses,omitempty"`

	// MacAddress represents the MAC address
	MacAddress string `json:"macAddress,omitempty"`

	// MTU represents the MTU
	MTU *int32 `json:"mtu,omitempty"`
}

// NetworkPolicyStatus represents network policy status
type NetworkPolicyStatus struct {
	// Name represents the policy name
	Name string `json:"name"`

	// Applied represents if the policy is applied
	Applied bool `json:"applied"`

	// Message represents status message
	Message string `json:"message,omitempty"`
}

// WorkloadIdentityList contains a list of WorkloadIdentity
// +kubebuilder:object:root=true
type WorkloadIdentityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkloadIdentity `json:"items"`
}

// WorkloadCluster represents a cluster in the Nephio workload cluster
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={nephio,cluster}
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status",description="Ready status"
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".spec.clusterType",description="Cluster Type"
// +kubebuilder:printcolumn:name="Region",type="string",JSONPath=".spec.region",description="Region"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type WorkloadCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkloadClusterSpec   `json:"spec,omitempty"`
	Status WorkloadClusterStatus `json:"status,omitempty"`
}

// WorkloadClusterSpec defines the desired state of WorkloadCluster
type WorkloadClusterSpec struct {
	// ClusterType specifies the cluster type
	// +kubebuilder:validation:Enum=edge;regional;central
	ClusterType string `json:"clusterType"`

	// Region specifies the geographic region
	Region string `json:"region"`

	// Zone specifies the availability zone
	Zone string `json:"zone,omitempty"`

	// Site specifies the site identifier
	Site string `json:"site,omitempty"`

	// Capabilities specifies cluster capabilities
	Capabilities ClusterCapabilities `json:"capabilities"`

	// Network specifies network configuration
	Network ClusterNetwork `json:"network"`

	// Resources specifies cluster resources
	Resources ClusterResources `json:"resources"`

	// Configuration specifies cluster-specific configuration
	Configuration map[string]string `json:"configuration,omitempty"`
}

// ClusterCapabilities defines cluster capabilities
type ClusterCapabilities struct {
	// VNFTypes specifies supported VNF types
	VNFTypes []string `json:"vnfTypes"`

	// QoSClasses specifies supported QoS classes
	QoSClasses []string `json:"qosClasses"`

	// NetworkFeatures specifies supported network features
	NetworkFeatures []string `json:"networkFeatures"`

	// StorageClasses specifies available storage classes
	StorageClasses []string `json:"storageClasses"`

	// HardwareFeatures specifies available hardware features
	HardwareFeatures []string `json:"hardwareFeatures,omitempty"`
}

// ClusterNetwork defines cluster network configuration
type ClusterNetwork struct {
	// CNI specifies the CNI plugin
	CNI string `json:"cni"`

	// ServiceCIDR specifies the service CIDR
	ServiceCIDR string `json:"serviceCIDR"`

	// PodCIDR specifies the pod CIDR
	PodCIDR string `json:"podCIDR"`

	// ClusterDNS specifies cluster DNS configuration
	ClusterDNS string `json:"clusterDNS"`

	// ExternalNetworks specifies external networks
	ExternalNetworks []ExternalNetwork `json:"externalNetworks,omitempty"`

	// NetworkPolicies specifies if network policies are supported
	NetworkPolicies bool `json:"networkPolicies"`
}

// ClusterResources defines cluster resource configuration
type ClusterResources struct {
	// Nodes specifies node information
	Nodes []NodeInfo `json:"nodes"`

	// TotalCPU specifies total CPU capacity
	TotalCPU string `json:"totalCPU"`

	// TotalMemory specifies total memory capacity
	TotalMemory string `json:"totalMemory"`

	// TotalStorage specifies total storage capacity
	TotalStorage string `json:"totalStorage"`

	// AvailableCPU specifies available CPU
	AvailableCPU string `json:"availableCPU"`

	// AvailableMemory specifies available memory
	AvailableMemory string `json:"availableMemory"`

	// AvailableStorage specifies available storage
	AvailableStorage string `json:"availableStorage"`
}

// NodeInfo defines node information
type NodeInfo struct {
	// Name specifies the node name
	Name string `json:"name"`

	// Role specifies the node role
	// +kubebuilder:validation:Enum=master;worker;edge
	Role string `json:"role"`

	// CPU specifies node CPU capacity
	CPU string `json:"cpu"`

	// Memory specifies node memory capacity
	Memory string `json:"memory"`

	// Storage specifies node storage capacity
	Storage string `json:"storage"`

	// Labels specifies node labels
	Labels map[string]string `json:"labels,omitempty"`

	// Taints specifies node taints
	Taints []NodeTaint `json:"taints,omitempty"`
}

// NodeTaint defines a node taint
type NodeTaint struct {
	// Key specifies the taint key
	Key string `json:"key"`

	// Value specifies the taint value
	Value string `json:"value,omitempty"`

	// Effect specifies the taint effect
	// +kubebuilder:validation:Enum=NoSchedule;PreferNoSchedule;NoExecute
	Effect string `json:"effect"`
}

// ExternalNetwork defines external network configuration
type ExternalNetwork struct {
	// Name specifies the network name
	Name string `json:"name"`

	// Type specifies the network type
	// +kubebuilder:validation:Enum=provider;tenant;management;storage
	Type string `json:"type"`

	// CIDR specifies the network CIDR
	CIDR string `json:"cidr"`

	// Gateway specifies the gateway address
	Gateway string `json:"gateway,omitempty"`

	// VLAN specifies the VLAN ID
	VLAN *int32 `json:"vlan,omitempty"`
}

// WorkloadClusterStatus defines the observed state of WorkloadCluster
type WorkloadClusterStatus struct {
	// Conditions represent the latest available observations of the cluster's current state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Phase represents the current phase of the cluster
	// +kubebuilder:validation:Enum=Pending;Ready;NotReady;Unknown
	Phase string `json:"phase,omitempty"`

	// ConnectedWorkloads represents the number of connected workloads
	ConnectedWorkloads int32 `json:"connectedWorkloads,omitempty"`

	// ResourceUtilization represents current resource utilization
	ResourceUtilization *ResourceUtilization `json:"resourceUtilization,omitempty"`

	// NetworkStatus represents network status
	NetworkStatus *ClusterNetworkStatus `json:"networkStatus,omitempty"`

	// LastHeartbeat represents the last heartbeat time
	LastHeartbeat *metav1.Time `json:"lastHeartbeat,omitempty"`

	// ObservedGeneration represents the observed generation
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// ResourceUtilization represents resource utilization
type ResourceUtilization struct {
	// CPUUtilization represents CPU utilization percentage
	CPUUtilization *float64 `json:"cpuUtilization,omitempty"`

	// MemoryUtilization represents memory utilization percentage
	MemoryUtilization *float64 `json:"memoryUtilization,omitempty"`

	// StorageUtilization represents storage utilization percentage
	StorageUtilization *float64 `json:"storageUtilization,omitempty"`

	// NetworkUtilization represents network utilization
	NetworkUtilization *float64 `json:"networkUtilization,omitempty"`
}

// ClusterNetworkStatus represents cluster network status
type ClusterNetworkStatus struct {
	// ConnectedNetworks represents connected networks
	ConnectedNetworks []string `json:"connectedNetworks,omitempty"`

	// ActiveConnections represents active connections
	ActiveConnections int32 `json:"activeConnections,omitempty"`

	// NetworkErrors represents network errors
	NetworkErrors []string `json:"networkErrors,omitempty"`
}

// WorkloadClusterList contains a list of WorkloadCluster
// +kubebuilder:object:root=true
type WorkloadClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkloadCluster `json:"items"`
}

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "workload.nephio.org", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// DeepCopyObject implementations for runtime.Object interface

// DeepCopyObject returns a generically typed copy of an object
func (wi *WorkloadIdentity) DeepCopyObject() runtime.Object {
	if c := wi.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopy returns a deep copy of WorkloadIdentity
func (wi *WorkloadIdentity) DeepCopy() *WorkloadIdentity {
	if wi == nil {
		return nil
	}
	out := new(WorkloadIdentity)
	wi.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all properties from this object into another
func (wi *WorkloadIdentity) DeepCopyInto(out *WorkloadIdentity) {
	*out = *wi
	out.TypeMeta = wi.TypeMeta
	wi.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	// Note: This is a basic implementation. In a real scenario, you'd need
	// to implement deep copying for all nested fields
}

// DeepCopyObject returns a generically typed copy of an object
func (wil *WorkloadIdentityList) DeepCopyObject() runtime.Object {
	if c := wil.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopy returns a deep copy of WorkloadIdentityList
func (wil *WorkloadIdentityList) DeepCopy() *WorkloadIdentityList {
	if wil == nil {
		return nil
	}
	out := new(WorkloadIdentityList)
	wil.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all properties from this object into another
func (wil *WorkloadIdentityList) DeepCopyInto(out *WorkloadIdentityList) {
	*out = *wil
	out.TypeMeta = wil.TypeMeta
	wil.ListMeta.DeepCopyInto(&out.ListMeta)
	if wil.Items != nil {
		in, out := &wil.Items, &out.Items
		*out = make([]WorkloadIdentity, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopyObject returns a generically typed copy of an object
func (wc *WorkloadCluster) DeepCopyObject() runtime.Object {
	if c := wc.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopy returns a deep copy of WorkloadCluster
func (wc *WorkloadCluster) DeepCopy() *WorkloadCluster {
	if wc == nil {
		return nil
	}
	out := new(WorkloadCluster)
	wc.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all properties from this object into another
func (wc *WorkloadCluster) DeepCopyInto(out *WorkloadCluster) {
	*out = *wc
	out.TypeMeta = wc.TypeMeta
	wc.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	// Note: This is a basic implementation
}

// DeepCopyObject returns a generically typed copy of an object
func (wcl *WorkloadClusterList) DeepCopyObject() runtime.Object {
	if c := wcl.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopy returns a deep copy of WorkloadClusterList
func (wcl *WorkloadClusterList) DeepCopy() *WorkloadClusterList {
	if wcl == nil {
		return nil
	}
	out := new(WorkloadClusterList)
	wcl.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all properties from this object into another
func (wcl *WorkloadClusterList) DeepCopyInto(out *WorkloadClusterList) {
	*out = *wcl
	out.TypeMeta = wcl.TypeMeta
	wcl.ListMeta.DeepCopyInto(&out.ListMeta)
	if wcl.Items != nil {
		in, out := &wcl.Items, &out.Items
		*out = make([]WorkloadCluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func init() {
	SchemeBuilder.Register(&WorkloadIdentity{}, &WorkloadIdentityList{})
	SchemeBuilder.Register(&WorkloadCluster{}, &WorkloadClusterList{})
}