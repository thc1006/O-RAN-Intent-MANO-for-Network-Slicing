package kpt

import (
	"context"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// FunctionInput represents the input to a KPT function
type FunctionInput struct {
	ResourceList *ResourceList
}

// FunctionOutput represents the output from a KPT function
type FunctionOutput struct {
	ResourceList *ResourceList
}

// ResourceList represents a list of Kubernetes resources
type ResourceList struct {
	Items          []*yaml.RNode
	FunctionConfig *yaml.RNode
}

// PipelineInput represents input to a function pipeline
type PipelineInput struct {
	Resources []*yaml.RNode
	Configs   map[string]*yaml.RNode
}

// PipelineOutput represents output from a function pipeline
type PipelineOutput struct {
	Resources []*yaml.RNode
	Results   []Result
}

// Result represents a function execution result
type Result struct {
	Message  string
	Severity string
	File     string
	Field    string
}

// Function represents a KPT function interface
type Function interface {
	Run(ctx context.Context, input *FunctionInput) (*FunctionOutput, error)
}
