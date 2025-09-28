package kpt

import (
	"context"
	"fmt"
	"strconv"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// ApplyQoSPolicyFunction applies QoS policies to network slices
type ApplyQoSPolicyFunction struct{}

// NewApplyQoSPolicyFunction creates a new ApplyQoSPolicyFunction
func NewApplyQoSPolicyFunction() *ApplyQoSPolicyFunction {
	return &ApplyQoSPolicyFunction{}
}

// Run executes the QoS policy function
func (f *ApplyQoSPolicyFunction) Run(ctx context.Context, input *FunctionInput) (*FunctionOutput, error) {
	if input == nil || input.ResourceList == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	// Extract QoS policies from config
	policies := make(map[string]map[string]interface{})
	if input.ResourceList.FunctionConfig != nil {
		policiesNode, _ := input.ResourceList.FunctionConfig.Pipe(yaml.Lookup("policies"))
		if policiesNode != nil && policiesNode.YNode() != nil {
			f.extractPolicies(policiesNode.YNode(), policies)
		}
	}

	// Apply QoS to network slices
	for _, resource := range input.ResourceList.Items {
		if err := f.applyQoSToSlice(resource, policies); err != nil {
			return nil, err
		}
	}

	return &FunctionOutput{
		ResourceList: input.ResourceList,
	}, nil
}

func (f *ApplyQoSPolicyFunction) applyQoSToSlice(resource *yaml.RNode, policies map[string]map[string]interface{}) error {
	kind, _ := resource.Pipe(yaml.Lookup("kind"))
	if kind == nil || kind.YNode().Value != "NetworkSlice" {
		return nil
	}

	// Get slice type
	sliceType, err := resource.Pipe(yaml.Lookup("spec", "sliceType"))
	if err != nil || sliceType == nil {
		return nil
	}

	// Get corresponding QoS policy
	policy, exists := policies[sliceType.YNode().Value]
	if !exists {
		return nil
	}

	// Create QoS node
	qosNode := yaml.NewMapRNode(nil)

	for key, value := range policy {
		switch v := value.(type) {
		case string:
			qosNode.PipeE(yaml.SetField(key, yaml.NewScalarRNode(v)))
		case int:
			qosNode.PipeE(yaml.SetField(key, yaml.NewScalarRNode(strconv.Itoa(v))))
		case float64:
			qosNode.PipeE(yaml.SetField(key, yaml.NewScalarRNode(strconv.FormatFloat(v, 'f', -1, 64))))
		}
	}

	// Set QoS on the slice
	spec, err := resource.Pipe(yaml.LookupCreate(yaml.MappingNode, "spec"))
	if err != nil {
		return err
	}
	return spec.PipeE(yaml.SetField("qos", qosNode))
}

func (f *ApplyQoSPolicyFunction) extractPolicies(node *yaml.Node, policies map[string]map[string]interface{}) {
	if node == nil || node.Content == nil {
		return
	}

	for i := 0; i < len(node.Content)-1; i += 2 {
		sliceType := node.Content[i].Value
		policyNode := node.Content[i+1]

		policy := make(map[string]interface{})
		if policyNode.Content != nil {
			for j := 0; j < len(policyNode.Content)-1; j += 2 {
				key := policyNode.Content[j].Value
				valueNode := policyNode.Content[j+1]

				// Try to parse as number
				if intVal, err := strconv.Atoi(valueNode.Value); err == nil {
					policy[key] = intVal
				} else if floatVal, err := strconv.ParseFloat(valueNode.Value, 64); err == nil {
					policy[key] = floatVal
				} else {
					policy[key] = valueNode.Value
				}
			}
		}

		policies[sliceType] = policy
	}
}
