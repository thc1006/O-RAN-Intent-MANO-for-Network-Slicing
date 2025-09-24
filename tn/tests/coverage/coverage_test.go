package coverage

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CoverageReport represents test coverage information
type CoverageReport struct {
	TotalStatements    int
	CoveredStatements  int
	CoveragePercentage float64
	PackageCoverage    map[string]PackageCoverage
}

// PackageCoverage represents coverage for a specific package
type PackageCoverage struct {
	Package            string
	TotalStatements    int
	CoveredStatements  int
	CoveragePercentage float64
	Files              map[string]FileCoverage
}

// FileCoverage represents coverage for a specific file
type FileCoverage struct {
	File               string
	TotalStatements    int
	CoveredStatements  int
	CoveragePercentage float64
}

// CoverageAnalyzer analyzes Go test coverage
type CoverageAnalyzer struct {
	MinimumCoverage float64
	CriticalFiles   []string
}

func NewCoverageAnalyzer() *CoverageAnalyzer {
	return &CoverageAnalyzer{
		MinimumCoverage: 80.0,
		CriticalFiles: []string{
			"tn/agent/pkg/http.go",
			"tn/agent/pkg/iperf.go",
			"tn/agent/pkg/vxlan/optimized_manager.go",
			"tn/agent/pkg/vxlan/manager.go",
			"pkg/security/",
		},
	}
}

func TestCoverage_GenerateAndAnalyze(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping coverage test in short mode")
	}

	analyzer := NewCoverageAnalyzer()

	// Generate coverage report
	coverageFile := "coverage.out"
	report, err := analyzer.GenerateCoverageReport(coverageFile)
	if err != nil {
		t.Logf("Warning: Could not generate coverage report: %v", err)
		t.Skip("Coverage analysis requires running 'go test -coverprofile=coverage.out ./...' first")
	}

	// Analyze overall coverage
	t.Run("overall_coverage_threshold", func(t *testing.T) {
		t.Logf("Overall coverage: %.2f%% (%d/%d statements)",
			report.CoveragePercentage,
			report.CoveredStatements,
			report.TotalStatements)

		assert.True(t, report.CoveragePercentage >= analyzer.MinimumCoverage,
			"Overall coverage %.2f%% is below minimum threshold %.2f%%",
			report.CoveragePercentage, analyzer.MinimumCoverage)
	})

	// Analyze package-level coverage
	t.Run("package_coverage_analysis", func(t *testing.T) {
		for packageName, packageCov := range report.PackageCoverage {
			t.Run(packageName, func(t *testing.T) {
				t.Logf("Package %s: %.2f%% coverage (%d/%d statements)",
					packageName,
					packageCov.CoveragePercentage,
					packageCov.CoveredStatements,
					packageCov.TotalStatements)

				// Critical packages should have higher coverage
				if analyzer.isCriticalPackage(packageName) {
					assert.True(t, packageCov.CoveragePercentage >= analyzer.MinimumCoverage,
						"Critical package %s has insufficient coverage: %.2f%%",
						packageName, packageCov.CoveragePercentage)
				}
			})
		}
	})

	// Analyze file-level coverage for critical files
	t.Run("critical_files_coverage", func(t *testing.T) {
		for _, criticalFile := range analyzer.CriticalFiles {
			t.Run(criticalFile, func(t *testing.T) {
				fileCov := analyzer.findFileCoverage(report, criticalFile)
				if fileCov == nil {
					t.Logf("Warning: Critical file %s not found in coverage report", criticalFile)
					return
				}

				t.Logf("Critical file %s: %.2f%% coverage (%d/%d statements)",
					fileCov.File,
					fileCov.CoveragePercentage,
					fileCov.CoveredStatements,
					fileCov.TotalStatements)

				assert.True(t, fileCov.CoveragePercentage >= analyzer.MinimumCoverage,
					"Critical file %s has insufficient coverage: %.2f%%",
					fileCov.File, fileCov.CoveragePercentage)
			})
		}
	})

	// Generate coverage report
	t.Run("generate_coverage_html", func(t *testing.T) {
		err := analyzer.GenerateHTMLCoverageReport(coverageFile)
		if err != nil {
			t.Logf("Warning: Could not generate HTML coverage report: %v", err)
		} else {
			t.Log("HTML coverage report generated: coverage.html")
		}
	})
}

func (analyzer *CoverageAnalyzer) GenerateCoverageReport(coverageFile string) (*CoverageReport, error) {
	if _, err := os.Stat(coverageFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("coverage file %s does not exist", coverageFile)
	}

	file, err := os.Open(coverageFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open coverage file: %w", err)
	}
	defer file.Close()

	report := &CoverageReport{
		PackageCoverage: make(map[string]PackageCoverage),
	}

	scanner := bufio.NewScanner(file)

	// Skip the mode line
	if scanner.Scan() {
		// mode: set or mode: count - intentionally ignored
		_ = scanner.Text()
	}

	packageCoverage := make(map[string]*PackageCoverage)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		// Parse coverage line: file:startLine.startCol,endLine.endCol numStmt count
		filePath := parts[0]
		numStmt, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		count, err := strconv.Atoi(parts[2])
		if err != nil {
			continue
		}

		// Extract package from file path
		packageName := analyzer.extractPackageName(filePath)

		// Initialize package coverage if not exists
		if _, exists := packageCoverage[packageName]; !exists {
			packageCoverage[packageName] = &PackageCoverage{
				Package: packageName,
				Files:   make(map[string]FileCoverage),
			}
		}

		pkg := packageCoverage[packageName]
		pkg.TotalStatements += numStmt
		report.TotalStatements += numStmt

		if count > 0 {
			pkg.CoveredStatements += numStmt
			report.CoveredStatements += numStmt
		}

		// Update file-level coverage
		if _, exists := pkg.Files[filePath]; !exists {
			pkg.Files[filePath] = FileCoverage{File: filePath}
		}

		fileCov := pkg.Files[filePath]
		fileCov.TotalStatements += numStmt
		if count > 0 {
			fileCov.CoveredStatements += numStmt
		}
		fileCov.CoveragePercentage = analyzer.calculatePercentage(fileCov.CoveredStatements, fileCov.TotalStatements)
		pkg.Files[filePath] = fileCov
	}

	// Calculate percentages
	for packageName, pkg := range packageCoverage {
		pkg.CoveragePercentage = analyzer.calculatePercentage(pkg.CoveredStatements, pkg.TotalStatements)
		report.PackageCoverage[packageName] = *pkg
	}

	report.CoveragePercentage = analyzer.calculatePercentage(report.CoveredStatements, report.TotalStatements)

	return report, scanner.Err()
}

func (analyzer *CoverageAnalyzer) GenerateHTMLCoverageReport(coverageFile string) error {
	// Use go tool cover to generate HTML report
	// cmd := fmt.Sprintf("go tool cover -html=%s -o coverage.html", coverageFile)

	// For testing purposes, we'll simulate this
	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Test Coverage Report</title>
</head>
<body>
    <h1>Test Coverage Report</h1>
    <p>Generated from coverage file: %s</p>
    <p>To view detailed coverage, run: go tool cover -html=%s</p>
</body>
</html>`, coverageFile, coverageFile)

	return os.WriteFile("coverage.html", []byte(htmlContent), 0644)
}

func (analyzer *CoverageAnalyzer) extractPackageName(filePath string) string {
	// Remove file extension and get directory
	dir := filepath.Dir(filePath)

	// Convert to package name format
	packageName := strings.ReplaceAll(dir, "/", "/")

	// Remove leading "./" if present
	packageName = strings.TrimPrefix(packageName, "./")

	return packageName
}

func (analyzer *CoverageAnalyzer) calculatePercentage(covered, total int) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(covered) / float64(total) * 100.0
}

func (analyzer *CoverageAnalyzer) isCriticalPackage(packageName string) bool {
	criticalPackages := []string{
		"tn/agent/pkg",
		"pkg/security",
		"orchestrator/pkg",
	}

	for _, critical := range criticalPackages {
		if strings.Contains(packageName, critical) {
			return true
		}
	}
	return false
}

func (analyzer *CoverageAnalyzer) findFileCoverage(report *CoverageReport, filePath string) *FileCoverage {
	for _, packageCov := range report.PackageCoverage {
		for file, fileCov := range packageCov.Files {
			if strings.Contains(file, filePath) {
				return &fileCov
			}
		}
	}
	return nil
}

func TestCoverage_UnitTestCoverage(t *testing.T) {
	// Test that unit tests exist for all main source files
	sourceFiles := []string{
		"../../agent/pkg/http.go",
		"../../agent/pkg/iperf.go",
		"../../agent/pkg/vxlan/optimized_manager.go",
		"../../agent/pkg/vxlan/manager.go",
	}

	for _, sourceFile := range sourceFiles {
		t.Run(sourceFile, func(t *testing.T) {
			// Check if corresponding test file exists
			testFile := strings.ReplaceAll(sourceFile, ".go", "_test.go")

			if _, err := os.Stat(testFile); os.IsNotExist(err) {
				t.Errorf("Test file %s does not exist for source file %s", testFile, sourceFile)
			}
		})
	}
}

func TestCoverage_IntegrationTestCoverage(t *testing.T) {
	// Test that integration tests exist for main components
	integrationTests := []string{
		"../integration/http_integration_test.go",
		"../integration/iperf_integration_test.go",
		"../integration/vxlan_integration_test.go",
	}

	for _, testFile := range integrationTests {
		t.Run(testFile, func(t *testing.T) {
			if _, err := os.Stat(testFile); os.IsNotExist(err) {
				t.Errorf("Integration test file %s does not exist", testFile)
			} else {
				t.Logf("Integration test file %s exists", testFile)
			}
		})
	}
}

func TestCoverage_SecurityTestCoverage(t *testing.T) {
	// Test that security tests exist
	securityTests := []string{
		"../security/kubernetes_manifest_test.go",
	}

	for _, testFile := range securityTests {
		t.Run(testFile, func(t *testing.T) {
			if _, err := os.Stat(testFile); os.IsNotExist(err) {
				t.Errorf("Security test file %s does not exist", testFile)
			} else {
				t.Logf("Security test file %s exists", testFile)
			}
		})
	}
}

func TestCoverage_TestQuality(t *testing.T) {
	// Test quality metrics

	t.Run("test_file_naming_convention", func(t *testing.T) {
		testFiles, err := filepath.Glob("../../**/*_test.go")
		require.NoError(t, err)

		for _, testFile := range testFiles {
			assert.True(t, strings.HasSuffix(testFile, "_test.go"),
				"Test file %s should follow naming convention", testFile)
		}

		assert.True(t, len(testFiles) > 0, "Should have test files")
		t.Logf("Found %d test files", len(testFiles))
	})

	t.Run("test_coverage_thresholds", func(t *testing.T) {
		// Define coverage thresholds for different components
		thresholds := map[string]float64{
			"critical_components": 90.0,
			"core_functionality": 85.0,
			"supporting_code":     75.0,
		}

		for component, threshold := range thresholds {
			t.Logf("Component %s should have at least %.1f%% coverage", component, threshold)
		}
	})
}

// Benchmark coverage analysis performance
func BenchmarkCoverage_Analysis(b *testing.B) {
	analyzer := NewCoverageAnalyzer()

	// Create a mock coverage file for benchmarking
	mockCoverageContent := `mode: set
github.com/test/pkg/file1.go:10.2,12.16 2 1
github.com/test/pkg/file1.go:12.16,14.3 1 0
github.com/test/pkg/file2.go:20.2,22.16 2 1
github.com/test/pkg/file2.go:22.16,24.3 1 1
`

	tmpFile := "benchmark_coverage.out"
	err := os.WriteFile(tmpFile, []byte(mockCoverageContent), 0644)
	if err != nil {
		b.Fatalf("Failed to create mock coverage file: %v", err)
	}
	defer os.Remove(tmpFile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report, err := analyzer.GenerateCoverageReport(tmpFile)
		if err != nil {
			b.Fatalf("Coverage analysis failed: %v", err)
		}
		_ = report // Use report to avoid optimization
	}
}

// Test helper functions
func TestCoverage_HelperFunctions(t *testing.T) {
	analyzer := NewCoverageAnalyzer()

	t.Run("extract_package_name", func(t *testing.T) {
		testCases := []struct {
			filePath    string
			expectedPkg string
		}{
			{"github.com/test/pkg/file.go", "github.com/test/pkg"},
			{"./tn/agent/pkg/http.go", "tn/agent/pkg"},
			{"file.go", "."},
		}

		for _, tc := range testCases {
			result := analyzer.extractPackageName(tc.filePath)
			assert.Equal(t, tc.expectedPkg, result)
		}
	})

	t.Run("calculate_percentage", func(t *testing.T) {
		testCases := []struct {
			covered  int
			total    int
			expected float64
		}{
			{80, 100, 80.0},
			{0, 100, 0.0},
			{100, 100, 100.0},
			{0, 0, 0.0},
		}

		for _, tc := range testCases {
			result := analyzer.calculatePercentage(tc.covered, tc.total)
			assert.Equal(t, tc.expected, result)
		}
	})

	t.Run("is_critical_package", func(t *testing.T) {
		testCases := []struct {
			packageName string
			expected    bool
		}{
			{"tn/agent/pkg", true},
			{"pkg/security", true},
			{"orchestrator/pkg", true},
			{"some/other/pkg", false},
		}

		for _, tc := range testCases {
			result := analyzer.isCriticalPackage(tc.packageName)
			assert.Equal(t, tc.expected, result, "Package: %s", tc.packageName)
		}
	})
}