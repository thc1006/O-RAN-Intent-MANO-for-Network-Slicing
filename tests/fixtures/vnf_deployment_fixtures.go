package fixtures

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// VNFDeployment represents a VNF deployment custom resource
type VNFDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              VNFDeploymentSpec   `json:"spec,omitempty"`
	Status            VNFDeploymentStatus `json:"status,omitempty"`
}

type VNFDeploymentSpec struct {
	VNFType     string            `json:"vnfType"`
	SliceType   string            `json:"sliceType"`
	Resources   ResourceRequests  `json:"resources"`
	QoSProfile  QoSProfile        `json:"qosProfile"`
	Placement   PlacementPolicy   `json:"placement"`
	DMSConfig   DMSConfiguration  `json:"dmsConfig"`
}

type VNFDeploymentStatus struct {
	Phase       string             `json:"phase"`
	Conditions  []metav1.Condition `json:"conditions,omitempty"`
	Resources   AllocatedResources `json:"resources,omitempty"`
	DMSStatus   DMSStatus          `json:"dmsStatus,omitempty"`
}

type ResourceRequests struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	GPU    string `json:"gpu,omitempty"`
}

type QoSProfile struct {
	Latency    string `json:"latency"`
	Throughput string `json:"throughput"`
	Reliability string `json:"reliability"`
}

type PlacementPolicy struct {
	Affinity     map[string]string `json:"affinity,omitempty"`
	AntiAffinity map[string]string `json:"antiAffinity,omitempty"`
	Zones        []string          `json:"zones,omitempty"`
}

type DMSConfiguration struct {
	Endpoint string            `json:"endpoint"`
	Auth     AuthConfig        `json:"auth"`
	Options  map[string]string `json:"options,omitempty"`
}

type AuthConfig struct {
	Type   string `json:"type"`
	Token  string `json:"token,omitempty"`
	KeyRef string `json:"keyRef,omitempty"`
}

type AllocatedResources struct {
	Nodes     []string `json:"nodes"`
	Pods      []string `json:"pods"`
	Services  []string `json:"services"`
}

type DMSStatus struct {
	Connected    bool   `json:"connected"`
	LastSync     string `json:"lastSync,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// Test fixtures
func ValidVNFDeployment() *VNFDeployment {
	return &VNFDeployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "oran.io/v1",
			Kind:       "VNFDeployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vnf",
			Namespace: "oran-system",
			Labels: map[string]string{
				"slice-type": "eMBB",
				"vnf-type":   "cucp",
			},
		},
		Spec: VNFDeploymentSpec{
			VNFType:   "cucp",
			SliceType: "eMBB",
			Resources: ResourceRequests{
				CPU:    "2000m",
				Memory: "4Gi",
				GPU:    "1",
			},
			QoSProfile: QoSProfile{
				Latency:     "10ms",
				Throughput:  "1Gbps",
				Reliability: "99.99%",
			},
			Placement: PlacementPolicy{
				Affinity: map[string]string{
					"node-type": "edge",
				},
				Zones: []string{"zone-a", "zone-b"},
			},
			DMSConfig: DMSConfiguration{
				Endpoint: "http://o2dms:8080",
				Auth: AuthConfig{
					Type:  "bearer",
					Token: "test-token",
				},
			},
		},
	}
}

func InvalidVNFDeployment() *VNFDeployment {
	vnf := ValidVNFDeployment()
	vnf.Spec.VNFType = "" // Invalid: empty VNF type
	vnf.Spec.Resources.CPU = "invalid-cpu" // Invalid: malformed CPU request
	return vnf
}

func eMBBVNFDeployment() *VNFDeployment {
	vnf := ValidVNFDeployment()
	vnf.Name = "embb-vnf"
	vnf.Spec.SliceType = "eMBB"
	vnf.Spec.QoSProfile = QoSProfile{
		Latency:     "20ms",
		Throughput:  "10Gbps",
		Reliability: "99.9%",
	}
	return vnf
}

func URLLCVNFDeployment() *VNFDeployment {
	vnf := ValidVNFDeployment()
	vnf.Name = "urllc-vnf"
	vnf.Spec.SliceType = "URLLC"
	vnf.Spec.QoSProfile = QoSProfile{
		Latency:     "1ms",
		Throughput:  "100Mbps",
		Reliability: "99.999%",
	}
	return vnf
}

func mMTCVNFDeployment() *VNFDeployment {
	vnf := ValidVNFDeployment()
	vnf.Name = "mmtc-vnf"
	vnf.Spec.SliceType = "mMTC"
	vnf.Spec.QoSProfile = QoSProfile{
		Latency:     "100ms",
		Throughput:  "10Mbps",
		Reliability: "99.9%",
	}
	return vnf
}

// DeepCopyObject implements runtime.Object
func (in *VNFDeployment) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopy creates a deep copy of VNFDeployment
func (in *VNFDeployment) DeepCopy() *VNFDeployment {
	if in == nil {
		return nil
	}
	out := new(VNFDeployment)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all properties of this object into another object of the same type
func (in *VNFDeployment) DeepCopyInto(out *VNFDeployment) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}