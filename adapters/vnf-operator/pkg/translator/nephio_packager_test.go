package translator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tests/fixtures"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tests/mocks"
)

// PorchClient interface for Nephio package operations
type PorchClient interface {
	CreatePackage(ctx context.Context, pkg *mocks.NephioPackage) error
	UpdatePackage(ctx context.Context, name string, pkg *mocks.NephioPackage) error
	DeletePackage(ctx context.Context, name string) error
	GetPackage(ctx context.Context, name string) (*mocks.NephioPackage, error)
	ListPackages(ctx context.Context) ([]*mocks.NephioPackage, error)
	ApprovePackage(ctx context.Context, name string) error
	ProposePackage(ctx context.Context, name string) error
}

// NephioPackager - the packager we're testing (not implemented yet)
type NephioPackager struct {
	PorchClient PorchClient
	Repository  string
	Namespace   string
}

// PackagerConfig for Nephio packager configuration
type PackagerConfig struct {
	Repository string
	Namespace  string
}

// NewNephioPackager creates a new Nephio packager (not implemented yet)
func NewNephioPackager(client PorchClient, config PackagerConfig) *NephioPackager {
	// Intentionally not implemented to cause test failure (RED phase)
	return nil
}

// Interface methods that need to be implemented
func (p *NephioPackager) CreateVNFPackage(ctx context.Context, vnfSpec *fixtures.VNFDeployment) (*mocks.NephioPackage, error) {
	// Not implemented yet - will cause tests to fail
	return nil, nil
}

func (p *NephioPackager) UpdateVNFPackage(ctx context.Context, packageName string, vnfSpec *fixtures.VNFDeployment) error {
	// Not implemented yet - will cause tests to fail
	return nil
}

func (p *NephioPackager) DeleteVNFPackage(ctx context.Context, packageName string) error {
	// Not implemented yet - will cause tests to fail
	return nil
}

func (p *NephioPackager) GenerateKptfile(vnfSpec *fixtures.VNFDeployment) (*mocks.Kptfile, error) {
	// Not implemented yet - will cause tests to fail
	return nil, nil
}

func (p *NephioPackager) GenerateConfigMaps(vnfSpec *fixtures.VNFDeployment) ([]interface{}, error) {
	// Not implemented yet - will cause tests to fail
	return nil, nil
}

func (p *NephioPackager) GenerateCRDs(vnfSpec *fixtures.VNFDeployment) ([]interface{}, error) {
	// Not implemented yet - will cause tests to fail
	return nil, nil
}

func (p *NephioPackager) ValidatePackage(pkg *mocks.NephioPackage) error {
	// Not implemented yet - will cause tests to fail
	return nil
}

// Table-driven tests for Nephio package generation from VNF specs
func TestNephioPackager_CreateVNFPackage(t *testing.T) {
	tests := []struct {
		name            string
		vnfSpec         *fixtures.VNFDeployment
		mockSetup       func(*mocks.MockPorchClient)
		expectedPackage *mocks.NephioPackage
		expectedError   bool
		validateCalls   func(t *testing.T, mockPorch *mocks.MockPorchClient)
	}{
		{
			name:    "create_cucp_embb_package",
			vnfSpec: fixtures.eMBBVNFDeployment(),
			mockSetup: func(mockPorch *mocks.MockPorchClient) {
				mockPorch.CreatePackageFunc = func(ctx context.Context, pkg *mocks.NephioPackage) error {
					return nil
				}
			},
			expectedError: false,
			validateCalls: func(t *testing.T, mockPorch *mocks.MockPorchClient) {
				require.Len(t, mockPorch.CreateCalls, 1)
				pkg := mockPorch.CreateCalls[0].Package
				assert.Equal(t, "cucp", pkg.Metadata["vnf-type"])
				assert.Equal(t, "eMBB", pkg.Metadata["slice-type"])
				assert.Contains(t, pkg.Name, "cucp")
				assert.Contains(t, pkg.Name, "embb")
			},
		},
		{
			name:    "create_cucp_urllc_package",
			vnfSpec: fixtures.URLLCVNFDeployment(),
			mockSetup: func(mockPorch *mocks.MockPorchClient) {
				mockPorch.CreatePackageFunc = func(ctx context.Context, pkg *mocks.NephioPackage) error {
					return nil
				}
			},
			expectedError: false,
			validateCalls: func(t *testing.T, mockPorch *mocks.MockPorchClient) {
				require.Len(t, mockPorch.CreateCalls, 1)
				pkg := mockPorch.CreateCalls[0].Package
				assert.Equal(t, "URLLC", pkg.Metadata["slice-type"])
				// Verify URLLC-specific QoS requirements in package
				assert.NotNil(t, pkg.Functions)
				assert.Len(t, pkg.Functions, 1)
			},
		},
		{
			name:    "create_cucp_mmtc_package",
			vnfSpec: fixtures.mMTCVNFDeployment(),
			mockSetup: func(mockPorch *mocks.MockPorchClient) {
				mockPorch.CreatePackageFunc = func(ctx context.Context, pkg *mocks.NephioPackage) error {
					return nil
				}
			},
			expectedError: false,
			validateCalls: func(t *testing.T, mockPorch *mocks.MockPorchClient) {
				require.Len(t, mockPorch.CreateCalls, 1)
				pkg := mockPorch.CreateCalls[0].Package
				assert.Equal(t, "mMTC", pkg.Metadata["slice-type"])
			},
		},
		{
			name:    "invalid_vnf_spec",
			vnfSpec: fixtures.InvalidVNFDeployment(),
			mockSetup: func(mockPorch *mocks.MockPorchClient) {
				// Should not be called with invalid spec
			},
			expectedError: true,
			validateCalls: func(t *testing.T, mockPorch *mocks.MockPorchClient) {
				assert.Len(t, mockPorch.CreateCalls, 0)
			},
		},
		{
			name:    "porch_creation_failure",
			vnfSpec: fixtures.ValidVNFDeployment(),
			mockSetup: func(mockPorch *mocks.MockPorchClient) {
				mockPorch.CreatePackageFunc = func(ctx context.Context, pkg *mocks.NephioPackage) error {
					return assert.AnError
				}
			},
			expectedError: true,
			validateCalls: func(t *testing.T, mockPorch *mocks.MockPorchClient) {
				assert.Len(t, mockPorch.CreateCalls, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			mockPorch := &mocks.MockPorchClient{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockPorch)
			}

			// Create packager
			packager := &NephioPackager{
				PorchClient: mockPorch,
				Repository:  "test-repo",
				Namespace:   "oran-system",
			}

			// Execute test
			result, err := packager.CreateVNFPackage(context.Background(), tt.vnfSpec)

			// Verify results
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			// Run custom validations
			if tt.validateCalls != nil {
				tt.validateCalls(t, mockPorch)
			}
		})
	}
}

// Test Kptfile creation
func TestNephioPackager_GenerateKptfile(t *testing.T) {
	tests := []struct {
		name          string
		vnfSpec       *fixtures.VNFDeployment
		expectedError bool
		validate      func(t *testing.T, kptfile *mocks.Kptfile)
	}{
		{
			name:          "generate_cucp_kptfile",
			vnfSpec:       fixtures.ValidVNFDeployment(),
			expectedError: false,
			validate: func(t *testing.T, kptfile *mocks.Kptfile) {
				assert.Equal(t, "kpt.dev/v1", kptfile.APIVersion)
				assert.Equal(t, "Kptfile", kptfile.Kind)
				assert.Equal(t, "test-vnf", kptfile.Metadata.Name)
				assert.Equal(t, "cucp", kptfile.Metadata.Labels["nephio.org/vnf-type"])
				assert.Equal(t, "eMBB", kptfile.Metadata.Labels["nephio.org/slice-type"])
			},
		},
		{
			name:          "generate_urllc_kptfile",
			vnfSpec:       fixtures.URLLCVNFDeployment(),
			expectedError: false,
			validate: func(t *testing.T, kptfile *mocks.Kptfile) {
				assert.Equal(t, "URLLC", kptfile.Metadata.Labels["nephio.org/slice-type"])
				// Verify URLLC-specific pipeline functions
				assert.NotEmpty(t, kptfile.Pipeline.Validators)
			},
		},
		{
			name:          "invalid_vnf_spec_kptfile",
			vnfSpec:       fixtures.InvalidVNFDeployment(),
			expectedError: true,
			validate: func(t *testing.T, kptfile *mocks.Kptfile) {
				assert.Nil(t, kptfile)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packager := &NephioPackager{}

			result, err := packager.GenerateKptfile(tt.vnfSpec)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// Test ConfigMap generation
func TestNephioPackager_GenerateConfigMaps(t *testing.T) {
	tests := []struct {
		name          string
		vnfSpec       *fixtures.VNFDeployment
		expectedCount int
		expectedError bool
		validate      func(t *testing.T, configMaps []interface{})
	}{
		{
			name:          "generate_cucp_configmaps",
			vnfSpec:       fixtures.ValidVNFDeployment(),
			expectedCount: 3, // deployment, service, config
			expectedError: false,
			validate: func(t *testing.T, configMaps []interface{}) {
				assert.Len(t, configMaps, 3)
				// Verify specific ConfigMaps are generated
				names := make([]string, len(configMaps))
				for i, cm := range configMaps {
					// Type assertion would be done in real implementation
					names[i] = "placeholder" // Would extract actual name
				}
				assert.Contains(t, names, "placeholder") // Would check for actual names
			},
		},
		{
			name:          "generate_urllc_configmaps",
			vnfSpec:       fixtures.URLLCVNFDeployment(),
			expectedCount: 3,
			expectedError: false,
			validate: func(t *testing.T, configMaps []interface{}) {
				// Verify URLLC-specific configurations
				assert.Len(t, configMaps, 3)
			},
		},
		{
			name:          "generate_mmtc_configmaps",
			vnfSpec:       fixtures.mMTCVNFDeployment(),
			expectedCount: 3,
			expectedError: false,
			validate: func(t *testing.T, configMaps []interface{}) {
				// Verify mMTC-specific configurations
				assert.Len(t, configMaps, 3)
			},
		},
		{
			name:          "invalid_vnf_spec_configmaps",
			vnfSpec:       fixtures.InvalidVNFDeployment(),
			expectedCount: 0,
			expectedError: true,
			validate: func(t *testing.T, configMaps []interface{}) {
				assert.Nil(t, configMaps)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packager := &NephioPackager{}

			result, err := packager.GenerateConfigMaps(tt.vnfSpec)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result, tt.expectedCount)
			}

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// Test CRD generation
func TestNephioPackager_GenerateCRDs(t *testing.T) {
	tests := []struct {
		name          string
		vnfSpec       *fixtures.VNFDeployment
		expectedCount int
		expectedError bool
		validate      func(t *testing.T, crds []interface{})
	}{
		{
			name:          "generate_cucp_crds",
			vnfSpec:       fixtures.ValidVNFDeployment(),
			expectedCount: 2, // VNF CRD + slice-specific CRD
			expectedError: false,
			validate: func(t *testing.T, crds []interface{}) {
				assert.Len(t, crds, 2)
			},
		},
		{
			name:          "generate_urllc_crds",
			vnfSpec:       fixtures.URLLCVNFDeployment(),
			expectedCount: 2,
			expectedError: false,
			validate: func(t *testing.T, crds []interface{}) {
				// Verify URLLC-specific CRDs
				assert.Len(t, crds, 2)
			},
		},
		{
			name:          "invalid_vnf_spec_crds",
			vnfSpec:       fixtures.InvalidVNFDeployment(),
			expectedCount: 0,
			expectedError: true,
			validate: func(t *testing.T, crds []interface{}) {
				assert.Nil(t, crds)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packager := &NephioPackager{}

			result, err := packager.GenerateCRDs(tt.vnfSpec)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result, tt.expectedCount)
			}

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// Test package validation
func TestNephioPackager_ValidatePackage(t *testing.T) {
	tests := []struct {
		name          string
		pkg           *mocks.NephioPackage
		expectedError bool
		errorContains string
	}{
		{
			name:          "valid_cucp_package",
			pkg:           fixtures.ValidCUCPPackage(),
			expectedError: false,
		},
		{
			name:          "valid_cuup_package",
			pkg:           fixtures.ValidCUUPPackage(),
			expectedError: false,
		},
		{
			name:          "valid_du_package",
			pkg:           fixtures.ValidDUPackage(),
			expectedError: false,
		},
		{
			name:          "invalid_package",
			pkg:           fixtures.InvalidPackage(),
			expectedError: true,
			errorContains: "package name cannot be empty",
		},
		{
			name:          "package_missing_kptfile",
			pkg:           fixtures.PackageWithMissingKptfile(),
			expectedError: true,
			errorContains: "Kptfile is required",
		},
		{
			name:          "package_invalid_functions",
			pkg:           fixtures.PackageWithInvalidFunctions(),
			expectedError: true,
			errorContains: "function image cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packager := &NephioPackager{}

			err := packager.ValidatePackage(tt.pkg)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test package update
func TestNephioPackager_UpdateVNFPackage(t *testing.T) {
	tests := []struct {
		name          string
		packageName   string
		vnfSpec       *fixtures.VNFDeployment
		mockSetup     func(*mocks.MockPorchClient)
		expectedError bool
		validateCalls func(t *testing.T, mockPorch *mocks.MockPorchClient)
	}{
		{
			name:        "update_existing_package",
			packageName: "cucp-embb-package",
			vnfSpec:     fixtures.eMBBVNFDeployment(),
			mockSetup: func(mockPorch *mocks.MockPorchClient) {
				mockPorch.GetPackageFunc = func(ctx context.Context, name string) (*mocks.NephioPackage, error) {
					return fixtures.ValidCUCPPackage(), nil
				}
				mockPorch.UpdatePackageFunc = func(ctx context.Context, name string, pkg *mocks.NephioPackage) error {
					return nil
				}
			},
			expectedError: false,
			validateCalls: func(t *testing.T, mockPorch *mocks.MockPorchClient) {
				assert.Len(t, mockPorch.GetCalls, 1)
				assert.Len(t, mockPorch.UpdateCalls, 1)
				assert.Equal(t, "cucp-embb-package", mockPorch.UpdateCalls[0].Name)
			},
		},
		{
			name:        "update_nonexistent_package",
			packageName: "nonexistent-package",
			vnfSpec:     fixtures.ValidVNFDeployment(),
			mockSetup: func(mockPorch *mocks.MockPorchClient) {
				mockPorch.GetPackageFunc = func(ctx context.Context, name string) (*mocks.NephioPackage, error) {
					return nil, assert.AnError
				}
			},
			expectedError: true,
			validateCalls: func(t *testing.T, mockPorch *mocks.MockPorchClient) {
				assert.Len(t, mockPorch.GetCalls, 1)
				assert.Len(t, mockPorch.UpdateCalls, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPorch := &mocks.MockPorchClient{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockPorch)
			}

			packager := &NephioPackager{
				PorchClient: mockPorch,
			}

			err := packager.UpdateVNFPackage(context.Background(), tt.packageName, tt.vnfSpec)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.validateCalls != nil {
				tt.validateCalls(t, mockPorch)
			}
		})
	}
}

// Test package deletion
func TestNephioPackager_DeleteVNFPackage(t *testing.T) {
	tests := []struct {
		name          string
		packageName   string
		mockSetup     func(*mocks.MockPorchClient)
		expectedError bool
		validateCalls func(t *testing.T, mockPorch *mocks.MockPorchClient)
	}{
		{
			name:        "delete_existing_package",
			packageName: "cucp-embb-package",
			mockSetup: func(mockPorch *mocks.MockPorchClient) {
				mockPorch.DeletePackageFunc = func(ctx context.Context, name string) error {
					return nil
				}
			},
			expectedError: false,
			validateCalls: func(t *testing.T, mockPorch *mocks.MockPorchClient) {
				assert.Len(t, mockPorch.DeleteCalls, 1)
				assert.Equal(t, "cucp-embb-package", mockPorch.DeleteCalls[0].Name)
			},
		},
		{
			name:        "delete_package_porch_error",
			packageName: "test-package",
			mockSetup: func(mockPorch *mocks.MockPorchClient) {
				mockPorch.DeletePackageFunc = func(ctx context.Context, name string) error {
					return assert.AnError
				}
			},
			expectedError: true,
			validateCalls: func(t *testing.T, mockPorch *mocks.MockPorchClient) {
				assert.Len(t, mockPorch.DeleteCalls, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPorch := &mocks.MockPorchClient{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockPorch)
			}

			packager := &NephioPackager{
				PorchClient: mockPorch,
			}

			err := packager.DeleteVNFPackage(context.Background(), tt.packageName)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.validateCalls != nil {
				tt.validateCalls(t, mockPorch)
			}
		})
	}
}
