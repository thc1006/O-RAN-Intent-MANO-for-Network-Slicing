package porch_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/porch"
)

// TestPackageCreation verifies that we can create a new package through Porch
func TestPackageCreation(t *testing.T) {
	t.Run("Create new package with valid spec", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		manager := porch.NewPackageLifecycleManager()

		packageSpec := &porch.PackageSpec{
			Name:        "test-network-slice",
			Namespace:   "default",
			Repository:  "nephio-packages",
			Description: "Test network slice package",
			Version:     "v1.0.0",
			Resources: []porch.Resource{
				{
					Name: "embb-slice.yaml",
					Content: `apiVersion: ran.o-ran.org/v1alpha1
kind: NetworkSlice
metadata:
  name: embb-slice
spec:
  sliceType: eMBB
  qos:
    bandwidth: 100
    latency: 20`,
				},
			},
		}

		// Act
		createdPackage, err := manager.CreatePackage(ctx, packageSpec)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, createdPackage)
		assert.Equal(t, "test-network-slice", createdPackage.Name)
		assert.Equal(t, "v1.0.0", createdPackage.Version)
		assert.Equal(t, porch.PackageStatusDraft, createdPackage.Status)
		assert.NotEmpty(t, createdPackage.UID)
		assert.NotZero(t, createdPackage.CreatedAt)
	})

	t.Run("Fail to create package with invalid spec", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		manager := porch.NewPackageLifecycleManager()

		invalidSpec := &porch.PackageSpec{
			Name: "", // Invalid: empty name
		}

		// Act
		createdPackage, err := manager.CreatePackage(ctx, invalidSpec)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, createdPackage)
		assert.Contains(t, err.Error(), "package name is required")
	})
}

// TestPackageValidation verifies package validation logic
func TestPackageValidation(t *testing.T) {
	t.Run("Validate package with correct schema", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		manager := porch.NewPackageLifecycleManager()

		pkg := &porch.Package{
			Name:      "valid-package",
			Namespace: "default",
			Resources: []porch.Resource{
				{
					Name: "deployment.yaml",
					Content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: test
        image: nginx:latest`,
				},
			},
		}

		// Act
		validationResult, err := manager.ValidatePackage(ctx, pkg)

		// Assert
		require.NoError(t, err)
		assert.True(t, validationResult.IsValid)
		assert.Empty(t, validationResult.Errors)
		assert.Contains(t, validationResult.ValidatedResources, "deployment.yaml")
	})

	t.Run("Fail validation with invalid Kubernetes resources", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		manager := porch.NewPackageLifecycleManager()

		pkg := &porch.Package{
			Name:      "invalid-package",
			Namespace: "default",
			Resources: []porch.Resource{
				{
					Name:    "invalid.yaml",
					Content: `invalid: yaml content: [`,
				},
			},
		}

		// Act
		validationResult, err := manager.ValidatePackage(ctx, pkg)

		// Assert
		require.NoError(t, err) // Validation should not error, just return invalid result
		assert.False(t, validationResult.IsValid)
		assert.NotEmpty(t, validationResult.Errors)
		assert.Contains(t, validationResult.Errors[0], "yaml")
	})
}

// TestPackageDependencyResolution verifies dependency resolution
func TestPackageDependencyResolution(t *testing.T) {
	t.Run("Resolve simple package dependencies", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		manager := porch.NewPackageLifecycleManager()

		pkg := &porch.Package{
			Name:      "app-package",
			Namespace: "default",
			Dependencies: []porch.Dependency{
				{
					Name:    "base-config",
					Version: ">=1.0.0",
				},
				{
					Name:    "network-policy",
					Version: "~2.1.0",
				},
			},
		}

		// Act
		resolvedDeps, err := manager.ResolveDependencies(ctx, pkg)

		// Assert
		require.NoError(t, err)
		assert.Len(t, resolvedDeps, 2)

		baseConfig := findDependency(resolvedDeps, "base-config")
		assert.NotNil(t, baseConfig)
		assert.True(t, baseConfig.Version >= "1.0.0")

		networkPolicy := findDependency(resolvedDeps, "network-policy")
		assert.NotNil(t, networkPolicy)
		assert.True(t, networkPolicy.Version >= "2.1.0" && networkPolicy.Version < "2.2.0")
	})

	t.Run("Handle circular dependencies", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		manager := porch.NewPackageLifecycleManager()

		// First, create package-b that depends on package-a (circular)
		packageBSpec := &porch.PackageSpec{
			Name:      "package-b",
			Namespace: "default",
			Version:   "1.0.0",
			Dependencies: []porch.Dependency{
				{
					Name:    "package-a",
					Version: "1.0.0",
				},
			},
		}
		_, err := manager.CreatePackage(ctx, packageBSpec)
		require.NoError(t, err)

		// Now create package-a that depends on package-b
		pkg := &porch.Package{
			Name:      "package-a",
			Namespace: "default",
			Dependencies: []porch.Dependency{
				{
					Name:    "package-b",
					Version: "1.0.0",
				},
			},
		}

		// Act
		resolvedDeps, err := manager.ResolveDependencies(ctx, pkg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, resolvedDeps)
		assert.Contains(t, err.Error(), "circular dependency")
	})
}

// TestPackagePromotion verifies package promotion through lifecycle stages
func TestPackagePromotion(t *testing.T) {
	t.Run("Promote package from draft to published", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		manager := porch.NewPackageLifecycleManager()

		// Create a draft package first
		packageSpec := &porch.PackageSpec{
			Name:        "promotion-test",
			Namespace:   "default",
			Repository:  "nephio-packages",
			Description: "Package for promotion testing",
		}

		draftPkg, err := manager.CreatePackage(ctx, packageSpec)
		require.NoError(t, err)
		require.Equal(t, porch.PackageStatusDraft, draftPkg.Status)

		// Act - Validate the package first
		validationResult, err := manager.ValidatePackage(ctx, draftPkg)
		require.NoError(t, err)
		require.True(t, validationResult.IsValid)

		// Act - Promote to proposed
		proposedPkg, err := manager.PromotePackage(ctx, draftPkg.UID, porch.PackageStatusProposed)
		require.NoError(t, err)
		assert.Equal(t, porch.PackageStatusProposed, proposedPkg.Status)

		// Act - Promote to published
		publishedPkg, err := manager.PromotePackage(ctx, proposedPkg.UID, porch.PackageStatusPublished)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, porch.PackageStatusPublished, publishedPkg.Status)
		assert.NotEmpty(t, publishedPkg.PublishedAt)
		assert.Equal(t, draftPkg.Name, publishedPkg.Name)
	})

	t.Run("Prevent invalid promotion transitions", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		manager := porch.NewPackageLifecycleManager()

		// Create a draft package
		packageSpec := &porch.PackageSpec{
			Name:       "invalid-promotion",
			Namespace:  "default",
			Repository: "nephio-packages",
		}

		draftPkg, err := manager.CreatePackage(ctx, packageSpec)
		require.NoError(t, err)

		// Act - Try to promote directly from draft to published (should fail)
		publishedPkg, err := manager.PromotePackage(ctx, draftPkg.UID, porch.PackageStatusPublished)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, publishedPkg)
		assert.Contains(t, err.Error(), "cannot promote from draft to published")
	})
}

// TestPackageRollback verifies package rollback functionality
func TestPackageRollback(t *testing.T) {
	t.Run("Rollback to previous package version", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		manager := porch.NewPackageLifecycleManager()

		// Create and publish v1.0.0
		v1Spec := &porch.PackageSpec{
			Name:       "rollback-test",
			Namespace:  "default",
			Repository: "nephio-packages",
			Version:    "v1.0.0",
		}
		v1Pkg, err := manager.CreatePackage(ctx, v1Spec)
		require.NoError(t, err)

		// Promote v1 to published
		_, err = manager.ValidatePackage(ctx, v1Pkg)
		require.NoError(t, err)
		v1Pkg, _ = manager.PromotePackage(ctx, v1Pkg.UID, porch.PackageStatusProposed)
		v1Published, _ := manager.PromotePackage(ctx, v1Pkg.UID, porch.PackageStatusPublished)

		// Create and publish v2.0.0
		v2Spec := &porch.PackageSpec{
			Name:       "rollback-test",
			Namespace:  "default",
			Repository: "nephio-packages",
			Version:    "v2.0.0",
		}
		_, err = manager.CreatePackage(ctx, v2Spec)
		require.NoError(t, err)

		// Act - Rollback to v1.0.0
		rollbackResult, err := manager.RollbackPackage(ctx, "rollback-test", "v1.0.0")

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, rollbackResult)
		assert.Equal(t, "v1.0.0", rollbackResult.CurrentVersion)
		assert.Equal(t, "v2.0.0", rollbackResult.PreviousVersion)
		assert.Equal(t, v1Published.UID, rollbackResult.CurrentPackageUID)
	})

	t.Run("Fail rollback to non-existent version", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		manager := porch.NewPackageLifecycleManager()

		// Act
		rollbackResult, err := manager.RollbackPackage(ctx, "non-existent", "v0.0.0")

		// Assert
		assert.Error(t, err)
		assert.Nil(t, rollbackResult)
		assert.Contains(t, err.Error(), "package version not found")
	})
}

// Helper function to find dependency by name
func findDependency(deps []*porch.ResolvedDependency, name string) *porch.ResolvedDependency {
	if deps == nil {
		return nil
	}
	for _, dep := range deps {
		if dep != nil && dep.Name == name {
			return dep
		}
	}
	return nil
}