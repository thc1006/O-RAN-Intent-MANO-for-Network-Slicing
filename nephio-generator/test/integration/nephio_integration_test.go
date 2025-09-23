package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/nephio-generator/pkg/configsync"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/nephio-generator/pkg/generator"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/nephio-generator/pkg/porch"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/nephio-generator/pkg/renderer"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/nephio-generator/pkg/validation"
)

// NephioIntegrationTestSuite provides comprehensive integration tests for Nephio components
type NephioIntegrationTestSuite struct {
	suite.Suite

	// Test environment
	testEnv    *envtest.Environment
	config     *rest.Config
	client     client.Client
	clientset  kubernetes.Interface

	// Components under test
	packageGenerator     *generator.EnhancedPackageGenerator
	repositoryManager    *porch.RepositoryManager
	packageRenderer      *renderer.PackageRenderer
	configSyncManager    *configsync.ConfigSyncManager
	deploymentValidator  *validation.DeploymentValidator

	// Test configuration
	testNamespace  string
	testDir        string
	cleanup        []func()
}

// SetupSuite initializes the test environment
func (suite *NephioIntegrationTestSuite) SetupSuite() {
	// Create test directory
	suite.testDir = filepath.Join(os.TempDir(), "nephio-integration-test")
	err := os.MkdirAll(suite.testDir, 0755)
	require.NoError(suite.T(), err)

	// Setup test environment
	suite.testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "api", "workload", "v1alpha1"),
			filepath.Join("..", "..", "deploy", "crds"),
		},
		ErrorIfCRDPathMissing: false,
	}

	// Start test environment
	suite.config, err = suite.testEnv.Start()
	require.NoError(suite.T(), err)

	// Create kubernetes clientset
	suite.clientset, err = kubernetes.NewForConfig(suite.config)
	require.NoError(suite.T(), err)

	// Create controller-runtime client
	scheme := runtime.NewScheme()
	err = corev1.AddToScheme(scheme)
	require.NoError(suite.T(), err)
	err = appsv1.AddToScheme(scheme)
	require.NoError(suite.T(), err)

	suite.client, err = client.New(suite.config, client.Options{Scheme: scheme})
	require.NoError(suite.T(), err)

	// Create test namespace
	suite.testNamespace = fmt.Sprintf("nephio-test-%d", time.Now().Unix())
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: suite.testNamespace,
		},
	}
	err = suite.client.Create(context.Background(), namespace)
	require.NoError(suite.T(), err)

	// Initialize components
	suite.initializeComponents()
}

// TearDownSuite cleans up the test environment
func (suite *NephioIntegrationTestSuite) TearDownSuite() {
	// Run cleanup functions
	for _, cleanup := range suite.cleanup {
		cleanup()
	}

	// Stop test environment
	if suite.testEnv != nil {
		err := suite.testEnv.Stop()
		assert.NoError(suite.T(), err)
	}

	// Clean up test directory
	if suite.testDir != "" {
		os.RemoveAll(suite.testDir)
	}
}

// initializeComponents initializes all Nephio components for testing
func (suite *NephioIntegrationTestSuite) initializeComponents() {
	var err error

	// Initialize package generator
	templateRegistry := NewMockTemplateRegistry()
	functionRegistry := NewMockFunctionRegistry()
	packageValidator := NewMockPackageValidator()

	suite.packageGenerator = generator.NewEnhancedPackageGenerator(
		templateRegistry,
		suite.testDir,
		"nephio-test",
		functionRegistry,
		packageValidator,
	)

	// Initialize repository manager
	suite.repositoryManager, err = porch.NewRepositoryManager(
		suite.config,
		suite.testNamespace,
		"main",
	)
	require.NoError(suite.T(), err)

	// Initialize package renderer
	suite.packageRenderer = renderer.NewPackageRenderer(
		suite.testDir,
		"/usr/local/bin/kpt", // Assume kpt is installed
		functionRegistry,
		NewMockRenderValidator(),
	)

	// Initialize ConfigSync manager
	suite.configSyncManager, err = configsync.NewConfigSyncManager(
		suite.config,
		"config-management-system",
	)
	require.NoError(suite.T(), err)

	// Initialize deployment validator
	validationConfig := validation.DefaultValidationConfig()
	suite.deploymentValidator, err = validation.NewDeploymentValidator(
		suite.config,
		validationConfig,
	)
	require.NoError(suite.T(), err)
}

// Test Cases

// TestPackageGenerationWorkflow tests the complete package generation workflow
func (suite *NephioIntegrationTestSuite) TestPackageGenerationWorkflow() {
	ctx := context.Background()

	// Test data
	vnfSpec := &generator.VNFSpec{
		Name:    "test-ran",
		Type:    "RAN",
		Version: "v1.0.0",
		QoS: generator.QoSRequirements{
			Bandwidth:   100.0,
			Latency:     10.0,
			Jitter:      1.0,
			PacketLoss:  0.01,
			Reliability: 99.9,
			SliceType:   "URLLC",
		},
		Placement: generator.PlacementSpec{
			CloudType: "edge",
			Region:    "us-west-1",
			Zone:      "us-west-1a",
			Site:      "edge01",
		},
		Resources: generator.ResourceSpec{
			CPUCores:  4,
			MemoryGB:  8,
			StorageGB: 100,
		},
		Image: generator.ImageSpec{
			Repository: "oran/ran",
			Tag:        "v1.0.0",
			PullPolicy: "IfNotPresent",
		},
	}

	// Generate package
	pkg, err := suite.packageGenerator.GenerateEnhancedPackage(
		ctx,
		vnfSpec,
		generator.TemplateTypeKpt,
	)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), pkg)
	assert.Equal(suite.T(), "nephio-test-test-ran-ran-edge", pkg.Name)
	assert.Equal(suite.T(), generator.TemplateTypeKpt, pkg.Type)
	assert.NotNil(suite.T(), pkg.Kptfile)
	assert.Greater(suite.T(), len(pkg.Resources), 0)

	// Validate package structure
	assert.Equal(suite.T(), "kpt.dev/v1", pkg.Kptfile.APIVersion)
	assert.Equal(suite.T(), "Kptfile", pkg.Kptfile.Kind)
	assert.NotEmpty(suite.T(), pkg.Kptfile.Metadata.Name)
	assert.NotEmpty(suite.T(), pkg.Kptfile.Info.Description)
	assert.Greater(suite.T(), len(pkg.Kptfile.Pipeline.Mutators), 0)
	assert.Greater(suite.T(), len(pkg.Kptfile.Pipeline.Validators), 0)

	// Validate resources
	foundNamespace := false
	foundDeployment := false
	foundService := false

	for _, resource := range pkg.Resources {
		switch resource.Kind {
		case "Namespace":
			foundNamespace = true
			assert.Equal(suite.T(), "v1", resource.APIVersion)
		case "Deployment":
			foundDeployment = true
			assert.Equal(suite.T(), "apps/v1", resource.APIVersion)
		case "Service":
			foundService = true
			assert.Equal(suite.T(), "v1", resource.APIVersion)
		}
	}

	assert.True(suite.T(), foundNamespace, "Package should contain a Namespace resource")
	assert.True(suite.T(), foundDeployment, "Package should contain a Deployment resource")
	assert.True(suite.T(), foundService, "Package should contain a Service resource")

	suite.T().Logf("Package generation test completed successfully")
}

// TestRepositoryManagement tests Porch repository management
func (suite *NephioIntegrationTestSuite) TestRepositoryManagement() {
	ctx := context.Background()

	// Create test repository
	repo := &porch.Repository{
		Name:      "test-repo",
		Namespace: suite.testNamespace,
		Type:      porch.RepositoryTypeGit,
		GitConfig: &porch.GitConfig{
			Repo:   "https://github.com/nephio-project/test-packages.git",
			Branch: "main",
			Auth:   porch.GitAuthTypeNone,
		},
		Deployment: false,
		Labels: map[string]string{
			"nephio.io/component": "test",
		},
		Annotations: map[string]string{
			"nephio.io/test": "true",
		},
	}

	// Create repository (this may fail in test environment without Porch)
	err := suite.repositoryManager.CreateRepository(ctx, repo)
	if err != nil {
		suite.T().Skipf("Skipping repository creation test - Porch not available: %v", err)
		return
	}

	// Add cleanup
	suite.cleanup = append(suite.cleanup, func() {
		suite.repositoryManager.DeleteRepository(ctx, repo.Name, repo.Namespace)
	})

	// Get repository
	retrievedRepo, err := suite.repositoryManager.GetRepository(ctx, repo.Name, repo.Namespace)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), repo.Name, retrievedRepo.Name)
	assert.Equal(suite.T(), repo.Type, retrievedRepo.Type)
	assert.Equal(suite.T(), repo.GitConfig.Repo, retrievedRepo.GitConfig.Repo)

	// List repositories
	repos, err := suite.repositoryManager.ListRepositories(ctx, suite.testNamespace)
	require.NoError(suite.T(), err)
	assert.Greater(suite.T(), len(repos), 0)

	suite.T().Logf("Repository management test completed successfully")
}

// TestPackageRendering tests package rendering with kpt functions
func (suite *NephioIntegrationTestSuite) TestPackageRendering() {
	ctx := context.Background()

	// Create test package directory
	packageDir := filepath.Join(suite.testDir, "test-render-package")
	err := os.MkdirAll(packageDir, 0755)
	require.NoError(suite.T(), err)

	// Create basic Kptfile
	kptfileContent := `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: test-package
  annotations:
    config.kubernetes.io/local-config: "true"
info:
  description: Test package for rendering
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-labels:v0.2.0
      configMap:
        app: test-app
  validators:
    - image: gcr.io/kpt-fn/kubeval:v0.3
`

	err = os.WriteFile(filepath.Join(packageDir, "Kptfile"), []byte(kptfileContent), 0644)
	require.NoError(suite.T(), err)

	// Create test resource
	resourceContent := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-app
  template:
    metadata:
      labels:
        app: test-app
    spec:
      containers:
      - name: test-container
        image: nginx:latest
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
`

	err = os.WriteFile(filepath.Join(packageDir, "deployment.yaml"), []byte(resourceContent), 0644)
	require.NoError(suite.T(), err)

	// Render package
	renderOptions := &renderer.RenderOptions{
		DryRun:   true,
		FailFast: false,
	}

	result, err := suite.packageRenderer.RenderPackage(ctx, packageDir, renderOptions)
	if err != nil {
		// Skip if kpt is not available in test environment
		suite.T().Skipf("Skipping package rendering test - kpt not available: %v", err)
		return
	}

	require.NoError(suite.T(), err)
	assert.True(suite.T(), result.Success)
	assert.Greater(suite.T(), len(result.Resources), 0)
	assert.Greater(suite.T(), len(result.FunctionResults), 0)

	suite.T().Logf("Package rendering test completed successfully")
}

// TestConfigSyncIntegration tests ConfigSync integration
func (suite *NephioIntegrationTestSuite) TestConfigSyncIntegration() {
	ctx := context.Background()

	// Create test RootSync
	rootSync := &configsync.RootSync{
		Name:      "test-rootsync",
		Namespace: "config-management-system",
		Labels: map[string]string{
			"nephio.io/test": "true",
		},
		Spec: configsync.RootSyncSpec{
			SourceFormat: "unstructured",
			Git: &configsync.GitSyncSpec{
				Repo:   "https://github.com/nephio-project/test-configs.git",
				Branch: "main",
				Dir:    "configs",
				Auth:   "none",
			},
			Override: &configsync.OverrideSpec{
				StatusMode:       "enabled",
				ReconcileTimeout: stringPtr("5m"),
				APIServerTimeout: stringPtr("15s"),
			},
		},
	}

	// Create RootSync (this may fail in test environment without ConfigSync)
	err := suite.configSyncManager.CreateRootSync(ctx, rootSync)
	if err != nil {
		suite.T().Skipf("Skipping ConfigSync test - ConfigSync not available: %v", err)
		return
	}

	// Add cleanup
	suite.cleanup = append(suite.cleanup, func() {
		suite.configSyncManager.DeleteRootSync(ctx, rootSync.Name, rootSync.Namespace)
	})

	// Get RootSync
	retrievedRootSync, err := suite.configSyncManager.GetRootSync(ctx, rootSync.Name, rootSync.Namespace)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), rootSync.Name, retrievedRootSync.Name)
	assert.Equal(suite.T(), rootSync.Spec.SourceFormat, retrievedRootSync.Spec.SourceFormat)

	suite.T().Logf("ConfigSync integration test completed successfully")
}

// TestDeploymentValidation tests deployment validation
func (suite *NephioIntegrationTestSuite) TestDeploymentValidation() {
	ctx := context.Background()

	// Create test deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: suite.testNamespace,
			Labels: map[string]string{
				"app":                     "test-app",
				"app.kubernetes.io/name":  "test-app",
				"oran.io/vnf-type":        "RAN",
				"oran.io/cloud-type":      "edge",
			},
			Annotations: map[string]string{
				"oran.io/qos-bandwidth": "100.0",
				"oran.io/qos-latency":   "10.0",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test-app",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test-app",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "nginx:latest",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    parseQuantity("100m"),
									"memory": parseQuantity("128Mi"),
								},
								Limits: corev1.ResourceList{
									"cpu":    parseQuantity("500m"),
									"memory": parseQuantity("512Mi"),
								},
							},
						},
					},
				},
			},
		},
	}

	// Create deployment
	err := suite.client.Create(ctx, deployment)
	require.NoError(suite.T(), err)

	// Add cleanup
	suite.cleanup = append(suite.cleanup, func() {
		suite.client.Delete(ctx, deployment)
	})

	// Wait for deployment to be ready
	time.Sleep(2 * time.Second)

	// Create test service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: suite.testNamespace,
			Labels: map[string]string{
				"app":              "test-app",
				"oran.io/vnf-type": "RAN",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "test-app",
			},
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Port:     80,
					Protocol: corev1.ProtocolTCP,
				},
				{
					Name:     "sctp",
					Port:     38412,
					Protocol: corev1.ProtocolSCTP,
				},
			},
		},
	}

	// Create service
	err = suite.client.Create(ctx, service)
	require.NoError(suite.T(), err)

	// Add cleanup
	suite.cleanup = append(suite.cleanup, func() {
		suite.client.Delete(ctx, service)
	})

	// Validate deployment
	validationSpec := &validation.DeploymentValidationSpec{
		Namespace: suite.testNamespace,
		LabelSelector: map[string]string{
			"app": "test-app",
		},
		VNFType:   "RAN",
		CloudType: "edge",
		Timeout:   30 * time.Second,
	}

	result, err := suite.deploymentValidator.ValidateDeployment(ctx, validationSpec)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Greater(suite.T(), result.Summary.TotalValidations, 0)
	assert.Greater(suite.T(), result.Summary.ValidationScore, 0.0)

	// Check specific validation results
	foundDeploymentValidation := false
	foundServiceValidation := false

	for _, validationResult := range result.Results {
		switch validationResult.Type {
		case validation.ValidationTypeDeployment:
			foundDeploymentValidation = true
		case validation.ValidationTypeService:
			foundServiceValidation = true
		}
	}

	assert.True(suite.T(), foundDeploymentValidation, "Should have deployment validation results")
	assert.True(suite.T(), foundServiceValidation, "Should have service validation results")

	suite.T().Logf("Deployment validation test completed successfully")
}

// TestEndToEndWorkflow tests the complete end-to-end workflow
func (suite *NephioIntegrationTestSuite) TestEndToEndWorkflow() {
	ctx := context.Background()

	suite.T().Log("Starting end-to-end workflow test")

	// Step 1: Generate package
	vnfSpec := &generator.VNFSpec{
		Name:    "e2e-test-vnf",
		Type:    "CN",
		Version: "v1.0.0",
		QoS: generator.QoSRequirements{
			Bandwidth:   50.0,
			Latency:     20.0,
			SliceType:   "eMBB",
		},
		Placement: generator.PlacementSpec{
			CloudType: "regional",
			Region:    "us-east-1",
		},
		Resources: generator.ResourceSpec{
			CPUCores:  2,
			MemoryGB:  4,
		},
		Image: generator.ImageSpec{
			Repository: "oran/upf",
			Tag:        "v1.0.0",
		},
	}

	pkg, err := suite.packageGenerator.GenerateEnhancedPackage(
		ctx,
		vnfSpec,
		generator.TemplateTypeKpt,
	)
	require.NoError(suite.T(), err)
	suite.T().Log("✓ Package generation completed")

	// Step 2: Validate package structure
	assert.NotNil(suite.T(), pkg)
	assert.NotEmpty(suite.T(), pkg.Name)
	assert.NotNil(suite.T(), pkg.Kptfile)
	assert.Greater(suite.T(), len(pkg.Resources), 0)
	suite.T().Log("✓ Package structure validation completed")

	// Step 3: Create mock deployment for validation
	// (In a real scenario, this would be deployed via Porch and ConfigSync)
	deployment := suite.createMockDeploymentFromPackage(ctx, pkg)
	require.NotNil(suite.T(), deployment)

	// Add cleanup
	suite.cleanup = append(suite.cleanup, func() {
		suite.client.Delete(ctx, deployment)
	})

	suite.T().Log("✓ Mock deployment created")

	// Step 4: Validate deployment
	validationSpec := &validation.DeploymentValidationSpec{
		Namespace: suite.testNamespace,
		LabelSelector: map[string]string{
			"app": pkg.VNFSpec.Name,
		},
		VNFType:   pkg.VNFSpec.Type,
		CloudType: pkg.VNFSpec.Placement.CloudType,
		Timeout:   30 * time.Second,
	}

	validationResult, err := suite.deploymentValidator.ValidateDeployment(ctx, validationSpec)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), validationResult.Valid || len(validationResult.Errors) == 0)
	suite.T().Log("✓ Deployment validation completed")

	suite.T().Log("End-to-end workflow test completed successfully")
}

// Helper methods

func (suite *NephioIntegrationTestSuite) createMockDeploymentFromPackage(ctx context.Context, pkg *generator.EnhancedPackage) *appsv1.Deployment {
	// Create a mock deployment based on the generated package
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pkg.VNFSpec.Name,
			Namespace: suite.testNamespace,
			Labels: map[string]string{
				"app":                     pkg.VNFSpec.Name,
				"app.kubernetes.io/name":  pkg.VNFSpec.Name,
				"oran.io/vnf-type":        pkg.VNFSpec.Type,
				"oran.io/cloud-type":      pkg.VNFSpec.Placement.CloudType,
			},
			Annotations: map[string]string{
				"oran.io/qos-bandwidth": fmt.Sprintf("%.2f", pkg.VNFSpec.QoS.Bandwidth),
				"oran.io/qos-latency":   fmt.Sprintf("%.2f", pkg.VNFSpec.QoS.Latency),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": pkg.VNFSpec.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": pkg.VNFSpec.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  pkg.VNFSpec.Type,
							Image: fmt.Sprintf("%s:%s", pkg.VNFSpec.Image.Repository, pkg.VNFSpec.Image.Tag),
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    parseQuantity(fmt.Sprintf("%d", pkg.VNFSpec.Resources.CPUCores)),
									"memory": parseQuantity(fmt.Sprintf("%dGi", pkg.VNFSpec.Resources.MemoryGB)),
								},
								Limits: corev1.ResourceList{
									"cpu":    parseQuantity(fmt.Sprintf("%d", pkg.VNFSpec.Resources.CPUCores)),
									"memory": parseQuantity(fmt.Sprintf("%dGi", pkg.VNFSpec.Resources.MemoryGB)),
								},
							},
						},
					},
				},
			},
		},
	}

	err := suite.client.Create(ctx, deployment)
	if err != nil {
		suite.T().Errorf("Failed to create mock deployment: %v", err)
		return nil
	}

	return deployment
}

// Test suite execution
func TestNephioIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(NephioIntegrationTestSuite))
}

// Helper functions and mocks

func stringPtr(s string) *string {
	return &s
}

func int32Ptr(i int32) *int32 {
	return &i
}

func parseQuantity(s string) resource.Quantity {
	q, _ := resource.ParseQuantity(s)
	return q
}

// Mock implementations for testing

// MockTemplateRegistry provides a mock template registry
type MockTemplateRegistry struct{}

func NewMockTemplateRegistry() *MockTemplateRegistry {
	return &MockTemplateRegistry{}
}

func (r *MockTemplateRegistry) GetTemplate(vnfType, templateType string) (*generator.PackageTemplate, error) {
	return &generator.PackageTemplate{
		Name:    fmt.Sprintf("%s-%s-template", vnfType, templateType),
		Version: "v1.0.0",
		VNFType: vnfType,
		Type:    generator.TemplateType(templateType),
		Files: []generator.TemplateFile{
			{
				Path:       "deployment.yaml",
				Content:    "# Deployment template",
				IsTemplate: true,
			},
			{
				Path:       "service.yaml",
				Content:    "# Service template",
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
		Dependencies: []string{},
		Metadata:     map[string]interface{}{},
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
		{
			Name:        "cn-kpt-template",
			VNFType:     "CN",
			Type:        generator.TemplateTypeKpt,
			Version:     "v1.0.0",
			Description: "CN Kpt template",
		},
	}, nil
}

// MockFunctionRegistry provides a mock function registry
type MockFunctionRegistry struct{}

func NewMockFunctionRegistry() *MockFunctionRegistry {
	return &MockFunctionRegistry{}
}

func (r *MockFunctionRegistry) GetFunction(name string) (*generator.KptFunction, error) {
	return &generator.KptFunction{
		Name:        name,
		Image:       fmt.Sprintf("gcr.io/kpt-fn/%s:v0.2.0", name),
		Version:     "v0.2.0",
		Description: fmt.Sprintf("Mock function: %s", name),
		ExecTimeout: 30 * time.Second,
	}, nil
}

func (r *MockFunctionRegistry) ListFunctions() ([]*renderer.KptFunction, error) {
	return []*renderer.KptFunction{
		{
			Name:        "set-labels",
			Image:       "gcr.io/kpt-fn/set-labels:v0.2.0",
			Type:        renderer.FunctionTypeMutator,
			Description: "Set labels on resources",
		},
		{
			Name:        "kubeval",
			Image:       "gcr.io/kpt-fn/kubeval:v0.3",
			Type:        renderer.FunctionTypeValidator,
			Description: "Validate Kubernetes resources",
		},
	}, nil
}

func (r *MockFunctionRegistry) ValidateFunction(fn *renderer.KptFunction) error {
	return nil
}

func (r *MockFunctionRegistry) ExecuteFunction(ctx context.Context, fn *renderer.KptFunction, packagePath string) error {
	// Mock successful execution
	return nil
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

// MockRenderValidator provides a mock render validator
type MockRenderValidator struct{}

func NewMockRenderValidator() *MockRenderValidator {
	return &MockRenderValidator{}
}

func (v *MockRenderValidator) ValidateRenderedPackage(packagePath string) (*renderer.ValidationResult, error) {
	return &renderer.ValidationResult{
		Valid: true,
		Summary: renderer.ValidationSummary{
			TotalResources:   1,
			ValidResources:   1,
			InvalidResources: 0,
			ErrorCount:       0,
			WarningCount:     0,
		},
	}, nil
}

func (v *MockRenderValidator) ValidateResources(resources []renderer.RenderedResource) (*renderer.ValidationResult, error) {
	return &renderer.ValidationResult{
		Valid: true,
		Summary: renderer.ValidationSummary{
			TotalResources:   len(resources),
			ValidResources:   len(resources),
			InvalidResources: 0,
			ErrorCount:       0,
			WarningCount:     0,
		},
	}, nil
}

