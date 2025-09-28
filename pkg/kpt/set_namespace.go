package kpt

import (
	"context"
	"fmt"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// SetNamespaceFunction sets namespace on all applicable resources
type SetNamespaceFunction struct{}

// NewSetNamespaceFunction creates a new SetNamespaceFunction
func NewSetNamespaceFunction() *SetNamespaceFunction {
	return &SetNamespaceFunction{}
}

// Run executes the set-namespace function
func (f *SetNamespaceFunction) Run(ctx context.Context, input *FunctionInput) (*FunctionOutput, error) {
	if input == nil || input.ResourceList == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	// Get the target namespace from function config
	var targetNamespace string
	if input.ResourceList.FunctionConfig != nil {
		ns, err := input.ResourceList.FunctionConfig.Pipe(yaml.Lookup("namespace"))
		if err == nil && ns != nil {
			targetNamespace = ns.YNode().Value
		}
	}

	if targetNamespace == "" {
		return nil, fmt.Errorf("namespace not specified in function config")
	}

	// Process each resource
	for _, resource := range input.ResourceList.Items {
		if err := f.setNamespaceOnResource(resource, targetNamespace); err != nil {
			return nil, err
		}
	}

	return &FunctionOutput{
		ResourceList: input.ResourceList,
	}, nil
}

func (f *SetNamespaceFunction) setNamespaceOnResource(resource *yaml.RNode, namespace string) error {
	// Get the kind of the resource
	kind, err := resource.Pipe(yaml.Lookup("kind"))
	if err != nil {
		return err
	}

	if kind == nil {
		return nil
	}

	// Skip cluster-scoped resources
	clusterScopedKinds := map[string]bool{
		"Namespace":                true,
		"Node":                      true,
		"PersistentVolume":          true,
		"ClusterRole":               true,
		"ClusterRoleBinding":        true,
		"StorageClass":              true,
		"CustomResourceDefinition":  true,
	}

	if clusterScopedKinds[kind.YNode().Value] {
		return nil
	}

	// Set the namespace
	metadata, err := resource.Pipe(yaml.LookupCreate(yaml.MappingNode, "metadata"))
	if err != nil {
		return err
	}
	return metadata.PipeE(yaml.SetField("namespace", yaml.NewScalarRNode(namespace)))
}
