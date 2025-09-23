package unit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/nephio-generator/pkg/generator"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/nephio-generator/pkg/renderer"
)

// MockGeneratorFunctionRegistry provides a mock function registry for generator
type MockGeneratorFunctionRegistry struct{}

func NewMockGeneratorFunctionRegistry() *MockGeneratorFunctionRegistry {
	return &MockGeneratorFunctionRegistry{}
}

func (r *MockGeneratorFunctionRegistry) GetFunction(name string) (*generator.KptFunction, error) {
	return &generator.KptFunction{
		Name:        name,
		Image:       fmt.Sprintf("gcr.io/kpt-fn/%s:v0.2.0", name),
		Version:     "v0.2.0",
		Description: fmt.Sprintf("Mock function: %s", name),
		ExecTimeout: 30 * time.Second,
		Required:    false,
	}, nil
}

func (r *MockGeneratorFunctionRegistry) ListFunctions() ([]*generator.KptFunction, error) {
	return []*generator.KptFunction{
		{
			Name:        "set-labels",
			Image:       "gcr.io/kpt-fn/set-labels:v0.2.0",
			Version:     "v0.2.0",
			Description: "Set labels on resources",
			ExecTimeout: 30 * time.Second,
			Required:    false,
		},
	}, nil
}

func (r *MockGeneratorFunctionRegistry) ValidateFunction(fn *generator.KptFunction) error {
	if fn == nil {
		return fmt.Errorf("function is nil")
	}
	if fn.Name == "" {
		return fmt.Errorf("function name is required")
	}
	return nil
}

// MockTemplateRegistry provides a mock template registry
type MockTemplateRegistry struct{}

func NewMockTemplateRegistry() *MockTemplateRegistry {
	return &MockTemplateRegistry{}
}

func (r *MockTemplateRegistry) GetTemplate(vnfType, templateType string) (*generator.PackageTemplate, error) {
	return &generator.PackageTemplate{
		Name:    fmt.Sprintf("%s-%s-template", vnfType, templateType),
		VNFType: vnfType,
		Version: "v1.0.0",
		Type:    generator.TemplateType(templateType),
		Files: []generator.TemplateFile{
			{
				Path:       "deployment.yaml",
				Content:    "# Deployment template",
				IsTemplate: true,
			},
		},
		Variables: map[string]generator.Variable{
			"vnf-name": {
				Name:        "vnf-name",
				Type:        "string",
				Description: "VNF name",
				Required:    true,
			},
		},
	}, nil
}

func (r *MockTemplateRegistry) ListTemplates() ([]generator.TemplateInfo, error) {
	return []generator.TemplateInfo{
		{
			Name:        "ran-kpt-template",
			VNFType:     "RAN",
			Type:        generator.TemplateTypeKpt,
			Version:     "v1.0.0",
			Description: "RAN Kpt template",
		},
	}, nil
}

// MockPackageValidator provides a mock package validator
type MockPackageValidator struct{}

func NewMockPackageValidator() *MockPackageValidator {
	return &MockPackageValidator{}
}

func (v *MockPackageValidator) ValidatePackage(pkg *generator.EnhancedPackage) error {
	return nil
}

func (v *MockPackageValidator) ValidateKptfile(kptfile *generator.EnhancedKptfile) error {
	return nil
}

func (v *MockPackageValidator) ValidateResources(resources []generator.EnhancedResource) error {
	return nil
}

func TestMockRegistryImplementsGeneratorInterface(t *testing.T) {
	// Test that MockGeneratorFunctionRegistry implements generator.KptFunctionRegistry
	registry := NewMockGeneratorFunctionRegistry()
	templateReg := NewMockTemplateRegistry()
	validator := NewMockPackageValidator()

	// This should compile without type errors
	gen := generator.NewEnhancedPackageGenerator(
		templateReg,
		"/tmp/test",
		"test-repo",
		registry,
		validator,
	)

	require.NotNil(t, gen)

	// Test GetFunction
	fn, err := registry.GetFunction("test-function")
	require.NoError(t, err)
	assert.Equal(t, "test-function", fn.Name)
	assert.Equal(t, "gcr.io/kpt-fn/test-function:v0.2.0", fn.Image)

	// Test ListFunctions
	funcs, err := registry.ListFunctions()
	require.NoError(t, err)
	assert.Len(t, funcs, 1)
	assert.Equal(t, "set-labels", funcs[0].Name)

	// Test ValidateFunction
	err = registry.ValidateFunction(&generator.KptFunction{
		Name: "valid-function",
	})
	require.NoError(t, err)

	err = registry.ValidateFunction(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "function is nil")

	// Test template registry
	tmpl, err := templateReg.GetTemplate("RAN", "kpt")
	require.NoError(t, err)
	assert.Equal(t, "RAN-kpt-template", tmpl.Name)
	assert.Equal(t, "RAN", tmpl.VNFType)

	templates, err := templateReg.ListTemplates()
	require.NoError(t, err)
	assert.Len(t, templates, 1)
}

// MockRendererFunctionRegistry for renderer package
type MockRendererFunctionRegistry struct{}

func NewMockRendererFunctionRegistry() *MockRendererFunctionRegistry {
	return &MockRendererFunctionRegistry{}
}

func (r *MockRendererFunctionRegistry) GetFunction(name string) (*renderer.KptFunction, error) {
	return &renderer.KptFunction{
		Name:        name,
		Image:       fmt.Sprintf("gcr.io/kpt-fn/%s:v0.2.0", name),
		Version:     "v0.2.0",
		Type:        renderer.FunctionTypeMutator,
		Description: fmt.Sprintf("Mock function: %s", name),
		ExecTimeout: 30 * time.Second,
	}, nil
}

func (r *MockRendererFunctionRegistry) ListFunctions() ([]*renderer.KptFunction, error) {
	return []*renderer.KptFunction{
		{
			Name:        "set-labels",
			Image:       "gcr.io/kpt-fn/set-labels:v0.2.0",
			Version:     "v0.2.0",
			Type:        renderer.FunctionTypeMutator,
			Description: "Set labels on resources",
			ExecTimeout: 30 * time.Second,
		},
	}, nil
}

func (r *MockRendererFunctionRegistry) ValidateFunction(fn *renderer.KptFunction) error {
	if fn == nil {
		return fmt.Errorf("function is nil")
	}
	if fn.Name == "" {
		return fmt.Errorf("function name is required")
	}
	return nil
}

func (r *MockRendererFunctionRegistry) ExecuteFunction(ctx context.Context, fn *renderer.KptFunction, packagePath string) error {
	return nil
}

// MockRenderValidator provides a mock render validator
type MockRenderValidator struct{}

func NewMockRenderValidator() *MockRenderValidator {
	return &MockRenderValidator{}
}

func (v *MockRenderValidator) ValidateRenderedPackage(packagePath string) (*renderer.ValidationResult, error) {
	return &renderer.ValidationResult{
		Valid:       true,
		Errors:      []renderer.ValidationError{},
		Warnings:    []renderer.ValidationError{},
		Suggestions: []renderer.ValidationError{},
		Summary: renderer.ValidationSummary{
			TotalResources:   10,
			ValidResources:   10,
			InvalidResources: 0,
			ErrorCount:       0,
			WarningCount:     0,
		},
	}, nil
}

func (v *MockRenderValidator) ValidateResources(resources []renderer.RenderedResource) (*renderer.ValidationResult, error) {
	return &renderer.ValidationResult{
		Valid:       true,
		Errors:      []renderer.ValidationError{},
		Warnings:    []renderer.ValidationError{},
		Suggestions: []renderer.ValidationError{},
		Summary: renderer.ValidationSummary{
			TotalResources:   len(resources),
			ValidResources:   len(resources),
			InvalidResources: 0,
			ErrorCount:       0,
			WarningCount:     0,
		},
	}, nil
}

func TestMockRegistryImplementsRendererInterface(t *testing.T) {
	// Test that MockRendererFunctionRegistry implements renderer.FunctionRegistry
	registry := NewMockRendererFunctionRegistry()
	validator := NewMockRenderValidator()

	// This should compile without type errors
	pkgRenderer := renderer.NewPackageRenderer(
		"/tmp/test",
		"/usr/local/bin/kpt",
		registry,
		validator,
	)

	require.NotNil(t, pkgRenderer)

	// Test GetFunction
	fn, err := registry.GetFunction("test-function")
	require.NoError(t, err)
	assert.Equal(t, "test-function", fn.Name)
	assert.Equal(t, renderer.FunctionTypeMutator, fn.Type)

	// Test ExecuteFunction
	err = registry.ExecuteFunction(context.Background(), fn, "/tmp/test")
	require.NoError(t, err)
}