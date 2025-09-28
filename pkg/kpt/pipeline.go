package kpt

import (
	"context"
	"fmt"
)

// FunctionPipeline chains multiple KPT functions
type FunctionPipeline struct {
	functions []Function
}

// NewFunctionPipeline creates a new function pipeline
func NewFunctionPipeline() *FunctionPipeline {
	return &FunctionPipeline{
		functions: []Function{},
	}
}

// AddFunction adds a function to the pipeline
func (p *FunctionPipeline) AddFunction(fn Function) {
	p.functions = append(p.functions, fn)
}

// Run executes all functions in the pipeline
func (p *FunctionPipeline) Run(ctx context.Context, input *PipelineInput) (*PipelineOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	currentResources := input.Resources
	results := []Result{}

	// Execute each function in sequence
	for i, fn := range p.functions {
		// Prepare function input
		fnInput := &FunctionInput{
			ResourceList: &ResourceList{
				Items: currentResources,
			},
		}

		// Set appropriate config for each function
		switch i {
		case 0: // SetNamespace
			if config, exists := input.Configs["namespace"]; exists {
				fnInput.ResourceList.FunctionConfig = config
			}
		case 1: // ResourceQuota
			if config, exists := input.Configs["quotas"]; exists {
				fnInput.ResourceList.FunctionConfig = config
			}
		case 2: // NetworkPolicy
			if config, exists := input.Configs["network"]; exists {
				fnInput.ResourceList.FunctionConfig = config
			}
		}

		// Execute function
		output, err := fn.Run(ctx, fnInput)
		if err != nil {
			return nil, fmt.Errorf("function %d failed: %w", i, err)
		}

		// Update resources for next function
		currentResources = output.ResourceList.Items

		// Record result
		results = append(results, Result{
			Message:  fmt.Sprintf("Function %d completed successfully", i),
			Severity: "info",
		})
	}

	return &PipelineOutput{
		Resources: currentResources,
		Results:   results,
	}, nil
}
