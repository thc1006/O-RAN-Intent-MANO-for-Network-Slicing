package mocks

import (
	"context"
	"k8s.io/apimachinery/pkg/runtime"
)

// MockPorchClient provides a mock implementation of Porch client for testing
type MockPorchClient struct {
	CreatePackageFunc    func(ctx context.Context, pkg *NephioPackage) error
	UpdatePackageFunc    func(ctx context.Context, name string, pkg *NephioPackage) error
	DeletePackageFunc    func(ctx context.Context, name string) error
	GetPackageFunc       func(ctx context.Context, name string) (*NephioPackage, error)
	ListPackagesFunc     func(ctx context.Context) ([]*NephioPackage, error)
	ApprovePackageFunc   func(ctx context.Context, name string) error
	ProposePackageFunc   func(ctx context.Context, name string) error

	// Call tracking
	CreateCalls   []CreatePackageCall
	UpdateCalls   []UpdatePackageCall
	DeleteCalls   []DeletePackageCall
	GetCalls      []GetPackageCall
	ListCalls     []ListPackageCall
	ApproveCalls  []ApprovePackageCall
	ProposeCalls  []ProposePackageCall
}

type CreatePackageCall struct {
	Package *NephioPackage
}

type UpdatePackageCall struct {
	Name    string
	Package *NephioPackage
}

type DeletePackageCall struct {
	Name string
}

type GetPackageCall struct {
	Name string
}

type ListPackageCall struct {
	// No parameters for list
}

type ApprovePackageCall struct {
	Name string
}

type ProposePackageCall struct {
	Name string
}

// NephioPackage represents a Nephio package structure
type NephioPackage struct {
	Name        string                 `json:"name"`
	Namespace   string                 `json:"namespace"`
	Repository  string                 `json:"repository"`
	Revision    string                 `json:"revision"`
	Kptfile     *Kptfile               `json:"kptfile,omitempty"`
	Resources   []runtime.Object       `json:"resources,omitempty"`
	Functions   []Function             `json:"functions,omitempty"`
	Conditions  []PackageCondition     `json:"conditions,omitempty"`
	Lifecycle   PackageLifecycle       `json:"lifecycle"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
}

type Kptfile struct {
	APIVersion string         `json:"apiVersion"`
	Kind       string         `json:"kind"`
	Metadata   KptMetadata    `json:"metadata"`
	Info       KptInfo        `json:"info,omitempty"`
	Pipeline   KptPipeline    `json:"pipeline,omitempty"`
	Inventory  KptInventory   `json:"inventory,omitempty"`
	Upstream   KptUpstream    `json:"upstream,omitempty"`
}

type KptMetadata struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type KptInfo struct {
	Description string `json:"description,omitempty"`
	Site        string `json:"site,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
}

type KptPipeline struct {
	Mutators   []Function `json:"mutators,omitempty"`
	Validators []Function `json:"validators,omitempty"`
}

type KptInventory struct {
	Namespace   string `json:"namespace,omitempty"`
	Name        string `json:"name,omitempty"`
	InventoryID string `json:"inventoryID,omitempty"`
}

type KptUpstream struct {
	Type string `json:"type"`
	Git  GitRef `json:"git,omitempty"`
}

type GitRef struct {
	Repo      string `json:"repo"`
	Directory string `json:"directory,omitempty"`
	Ref       string `json:"ref,omitempty"`
}

type Function struct {
	Image       string                 `json:"image"`
	ConfigPath  string                 `json:"configPath,omitempty"`
	ConfigMap   map[string]interface{} `json:"configMap,omitempty"`
	Selectors   []Selector             `json:"selectors,omitempty"`
}

type Selector struct {
	APIVersion string            `json:"apiVersion,omitempty"`
	Kind       string            `json:"kind,omitempty"`
	Name       string            `json:"name,omitempty"`
	Namespace  string            `json:"namespace,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

type PackageCondition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

type PackageLifecycle string

const (
	PackageLifecycleDraft    PackageLifecycle = "Draft"
	PackageLifecycleProposed PackageLifecycle = "Proposed"
	PackageLifecyclePublished PackageLifecycle = "Published"
)

// Mock implementations
func (m *MockPorchClient) CreatePackage(ctx context.Context, pkg *NephioPackage) error {
	m.CreateCalls = append(m.CreateCalls, CreatePackageCall{Package: pkg})
	if m.CreatePackageFunc != nil {
		return m.CreatePackageFunc(ctx, pkg)
	}
	return nil
}

func (m *MockPorchClient) UpdatePackage(ctx context.Context, name string, pkg *NephioPackage) error {
	m.UpdateCalls = append(m.UpdateCalls, UpdatePackageCall{Name: name, Package: pkg})
	if m.UpdatePackageFunc != nil {
		return m.UpdatePackageFunc(ctx, name, pkg)
	}
	return nil
}

func (m *MockPorchClient) DeletePackage(ctx context.Context, name string) error {
	m.DeleteCalls = append(m.DeleteCalls, DeletePackageCall{Name: name})
	if m.DeletePackageFunc != nil {
		return m.DeletePackageFunc(ctx, name)
	}
	return nil
}

func (m *MockPorchClient) GetPackage(ctx context.Context, name string) (*NephioPackage, error) {
	m.GetCalls = append(m.GetCalls, GetPackageCall{Name: name})
	if m.GetPackageFunc != nil {
		return m.GetPackageFunc(ctx, name)
	}
	return &NephioPackage{Name: name}, nil
}

func (m *MockPorchClient) ListPackages(ctx context.Context) ([]*NephioPackage, error) {
	m.ListCalls = append(m.ListCalls, ListPackageCall{})
	if m.ListPackagesFunc != nil {
		return m.ListPackagesFunc(ctx)
	}
	return []*NephioPackage{}, nil
}

func (m *MockPorchClient) ApprovePackage(ctx context.Context, name string) error {
	m.ApproveCalls = append(m.ApproveCalls, ApprovePackageCall{Name: name})
	if m.ApprovePackageFunc != nil {
		return m.ApprovePackageFunc(ctx, name)
	}
	return nil
}

func (m *MockPorchClient) ProposePackage(ctx context.Context, name string) error {
	m.ProposeCalls = append(m.ProposeCalls, ProposePackageCall{Name: name})
	if m.ProposePackageFunc != nil {
		return m.ProposePackageFunc(ctx, name)
	}
	return nil
}

// Helper functions for creating test packages
func CreateTestNephioPackage(name string) *NephioPackage {
	return &NephioPackage{
		Name:       name,
		Namespace:  "default",
		Repository: "test-repo",
		Revision:   "v1.0.0",
		Lifecycle:  PackageLifecycleDraft,
		Kptfile: &Kptfile{
			APIVersion: "kpt.dev/v1",
			Kind:       "Kptfile",
			Metadata: KptMetadata{
				Name: name,
			},
		},
		Metadata: map[string]string{
			"vnf-type":   "cucp",
			"slice-type": "eMBB",
		},
	}
}

func CreateVNFPackage(vnfType, sliceType string) *NephioPackage {
	pkg := CreateTestNephioPackage(vnfType + "-" + sliceType)
	pkg.Metadata["vnf-type"] = vnfType
	pkg.Metadata["slice-type"] = sliceType
	return pkg
}

func CreateInvalidPackage() *NephioPackage {
	return &NephioPackage{
		Name: "", // Invalid: empty name
		Kptfile: &Kptfile{
			APIVersion: "invalid/v1", // Invalid API version
		},
	}
}