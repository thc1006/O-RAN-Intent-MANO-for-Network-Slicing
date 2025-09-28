package kpt

import (
	"context"
	"fmt"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// NetworkPolicyFunction generates NetworkPolicies for services
type NetworkPolicyFunction struct{}

// NewNetworkPolicyFunction creates a new NetworkPolicyFunction
func NewNetworkPolicyFunction() *NetworkPolicyFunction {
	return &NetworkPolicyFunction{}
}

// Run executes the network policy function
func (f *NetworkPolicyFunction) Run(ctx context.Context, input *FunctionInput) (*FunctionOutput, error) {
	if input == nil || input.ResourceList == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	// Extract configuration
	var mode string
	var allowedNamespaces []string

	if input.ResourceList.FunctionConfig != nil {
		modeNode, _ := input.ResourceList.FunctionConfig.Pipe(yaml.Lookup("mode"))
		if modeNode != nil {
			mode = modeNode.YNode().Value
		}

		allowedNsNode, _ := input.ResourceList.FunctionConfig.Pipe(yaml.Lookup("allowedNamespaces"))
		if allowedNsNode != nil && allowedNsNode.YNode() != nil {
			for _, ns := range allowedNsNode.YNode().Content {
				allowedNamespaces = append(allowedNamespaces, ns.Value)
			}
		}
	}

	// Generate network policies for services
	newResources := []*yaml.RNode{}
	for _, resource := range input.ResourceList.Items {
		newResources = append(newResources, resource)

		kind, _ := resource.Pipe(yaml.Lookup("kind"))
		if kind != nil && kind.YNode().Value == "Service" {
			networkPolicy := f.generateNetworkPolicy(resource, mode, allowedNamespaces)
			if networkPolicy != nil {
				newResources = append(newResources, networkPolicy)
			}
		}
	}

	return &FunctionOutput{
		ResourceList: &ResourceList{
			Items:          newResources,
			FunctionConfig: input.ResourceList.FunctionConfig,
		},
	}, nil
}

func (f *NetworkPolicyFunction) generateNetworkPolicy(service *yaml.RNode, mode string, allowedNamespaces []string) *yaml.RNode {
	name, _ := service.Pipe(yaml.Lookup("metadata", "name"))
	if name == nil {
		return nil
	}

	namespace, _ := service.Pipe(yaml.Lookup("metadata", "namespace"))
	nsValue := "default"
	if namespace != nil {
		nsValue = namespace.YNode().Value
	}

	selector, _ := service.Pipe(yaml.Lookup("spec", "selector"))

	// Build ingress rules
	ingressRules := ""
	if mode == "strict" {
		for _, ns := range allowedNamespaces {
			ingressRules += fmt.Sprintf(`
    - from:
      - namespaceSelector:
          matchLabels:
            name: %s`, ns)
		}
	}

	// Generate NetworkPolicy YAML
	policyYAML := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: %s-network-policy
  namespace: %s
spec:
  podSelector:
    matchLabels:`, name.YNode().Value, nsValue)

	// Add selector labels
	if selector != nil && selector.YNode() != nil {
		content := selector.YNode().Content
		for i := 0; i < len(content)-1; i += 2 {
			key := content[i].Value
			value := content[i+1].Value
			policyYAML += fmt.Sprintf("\n      %s: %s", key, value)
		}
	}

	policyYAML += `
  policyTypes:
  - Ingress
  - Egress`

	if ingressRules != "" {
		policyYAML += "\n  ingress:" + ingressRules
	}

	policyYAML += `
  egress:
  - to:
    - podSelector: {}
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
    ports:
    - protocol: TCP
      port: 53
    - protocol: UDP
      port: 53`

	return yaml.MustParse(policyYAML)
}
