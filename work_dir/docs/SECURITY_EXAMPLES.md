# Security Scanner Usage Examples

## Example 1: Basic Package Scanning

```go
package main

import (
    "fmt"
    "log"

    "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/work_dir/security"
)

func main() {
    // Create a new security scanner
    scanner := security.NewSecurityScanner()

    // Scan a Nephio package
    packagePath := "deployments/5g-core-network"
    fmt.Printf("Scanning package: %s\n", packagePath)

    err := scanner.ScanPackage(packagePath)
    if err != nil {
        log.Fatalf("Scan failed: %v", err)
    }

    // Get all violations
    violations := scanner.GetViolations()

    fmt.Printf("\nFound %d security violations\n", len(violations))

    // Print violations by severity
    for _, v := range violations {
        fmt.Printf("[%s] %s: %s\n", v.Severity, v.Type, v.Message)
    }

    // Generate detailed report
    reportPath := "reports/security-scan-results.md"
    err = scanner.GenerateReport(reportPath)
    if err != nil {
        log.Fatalf("Report generation failed: %v", err)
    }

    fmt.Printf("\nDetailed report saved to: %s\n", reportPath)
}
```

## Example 2: Strict Mode Enforcement

```go
package main

import (
    "fmt"
    "log"

    "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/work_dir/security"
)

func main() {
    // Scan package
    scanner := security.NewSecurityScanner()
    err := scanner.ScanPackage("deployments/ran-controller")
    if err != nil {
        log.Fatalf("Scan failed: %v", err)
    }

    violations := scanner.GetViolations()

    // Create strict enforcer
    enforcer := security.NewPolicyEnforcer(true)

    // Enforce security policies
    err = enforcer.Enforce(violations)
    if err != nil {
        fmt.Printf("❌ Security policy enforcement FAILED\n")
        fmt.Printf("Error: %v\n\n", err)

        // Get recommended actions
        actions := enforcer.GetRecommendedActions(violations)
        fmt.Println("Recommended Actions:")
        for i, action := range actions {
            fmt.Printf("%d. %s\n", i+1, action)
        }

        log.Fatal("Deployment blocked due to security violations")
    }

    fmt.Println("✅ Security policy enforcement PASSED")
    fmt.Println("Package is safe to deploy")
}
```

## Example 3: Custom Enforcement Rules

```go
package main

import (
    "fmt"
    "log"

    "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/work_dir/security"
)

func main() {
    // Configure custom enforcement rules
    config := security.EnforcerConfig{
        StrictMode: true,
        MaxCritical: 0,
        MaxHigh: 1,  // Allow 1 high severity violation
        BlockOnCritical: true,
        BlockOnHigh: false,
        CustomRules: []security.EnforcementRule{
            {
                ViolationType: "HostPathVolume",
                MaxAllowed: 0,
                BlockOnExceed: true,
                Message: "Organization policy: No hostPath volumes allowed",
            },
            {
                ViolationType: "PrivilegedContainer",
                MaxAllowed: 0,
                BlockOnExceed: true,
                Message: "Organization policy: No privileged containers",
            },
        },
        AllowedViolations: map[string]int{
            "MissingReadOnlyRootFS": 3,  // Allow up to 3 missing readOnlyRootFS
        },
    }

    enforcer := security.NewPolicyEnforcerWithConfig(config)

    // Scan and enforce
    scanner := security.NewSecurityScanner()
    scanner.ScanPackage("deployments/upf")
    violations := scanner.GetViolations()

    err := enforcer.Enforce(violations)
    if err != nil {
        log.Fatalf("Custom policy violation: %v", err)
    }

    fmt.Println("✅ Custom security policies satisfied")
}
```

## Example 4: Pod Security Standards Validation

```go
package main

import (
    "fmt"
    "log"

    "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/work_dir/security"
)

func main() {
    scanner := security.NewSecurityScanner()
    scanner.ScanPackage("deployments/amf")
    violations := scanner.GetViolations()

    enforcer := security.NewPolicyEnforcer(true)

    // Validate against different PSS levels
    levels := []string{"Privileged", "Baseline", "Restricted"}

    for _, level := range levels {
        passed, failures := enforcer.ValidateAgainstPodSecurityStandards(violations, level)

        fmt.Printf("\nPod Security Standard: %s\n", level)
        if passed {
            fmt.Println("Status: ✅ PASSED")
        } else {
            fmt.Printf("Status: ❌ FAILED (%d failures)\n", len(failures))
            fmt.Println("\nFailures:")
            for _, failure := range failures {
                fmt.Printf("  - %s\n", failure)
            }
        }
    }
}
```

## Example 5: Integrated Package Validation

```go
package main

import (
    "fmt"
    "log"

    "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/work_dir/security"
)

func main() {
    // Create integrated security configuration
    config := security.SecurityIntegrationConfig{
        EnableScanning:   true,
        StrictMode:       true,
        GenerateReports:  true,
        ReportOutputPath: "reports/deployment-security-scan.md",
        FailOnViolations: true,
        PodSecurityLevel: "Baseline",
        BlockCritical:    true,
        BlockHigh:        true,
    }

    renderer := security.NewSecurePackageRenderer(config)

    // Validate package before deployment
    packagePath := "nephio-packages/5g-smf"
    fmt.Printf("Validating package: %s\n", packagePath)

    result, err := renderer.ValidatePackage(packagePath)

    // Print validation result
    security.PrintValidationResult(result)

    if err != nil {
        log.Fatalf("Validation failed: %v", err)
    }

    if !result.Valid {
        fmt.Println("\n⚠️  Package has security issues but deployment allowed (non-strict mode)")
    } else {
        fmt.Println("\n✅ Package is secure and ready for deployment")
    }

    // Print summary
    summary := result.Summary.(map[string]interface{})
    fmt.Printf("\nSummary:\n")
    fmt.Printf("  Total Violations: %d\n", summary["total_violations"])
    fmt.Printf("  Has Critical: %v\n", summary["has_critical"])
    fmt.Printf("  Has High: %v\n", summary["has_high"])
}
```

## Example 6: Batch Package Scanning

```go
package main

import (
    "fmt"
    "log"

    "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/work_dir/security"
)

func main() {
    config := security.CreateDefaultSecurityConfig()
    config.FailOnViolations = false  // Don't stop on first failure

    // Scan all packages in directory
    packagesDir := "nephio-packages"
    fmt.Printf("Scanning all packages in: %s\n\n", packagesDir)

    results, err := security.ScanPackageDirectory(packagesDir, config)
    if err != nil {
        log.Fatalf("Batch scan failed: %v", err)
    }

    // Print summary for each package
    fmt.Println("Scan Results:")
    fmt.Println("=" + repeat("=", 79))

    totalPackages := len(results)
    passedPackages := 0

    for pkgName, result := range results {
        status := "✅ PASS"
        if !result.Valid {
            status = "❌ FAIL"
        } else {
            passedPackages++
        }

        fmt.Printf("%-30s %s (violations: %d)\n",
            pkgName,
            status,
            len(result.Violations))
    }

    fmt.Println("=" + repeat("=", 79))
    fmt.Printf("\nSummary: %d/%d packages passed security validation\n",
        passedPackages, totalPackages)
}

func repeat(s string, count int) string {
    result := ""
    for i := 0; i < count; i++ {
        result += s
    }
    return result
}
```

## Example 7: CI/CD Integration

```go
// ci-security-check.go
package main

import (
    "fmt"
    "os"

    "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/work_dir/security"
)

func main() {
    // Get package path from environment
    packagePath := os.Getenv("NEPHIO_PACKAGE_PATH")
    if packagePath == "" {
        fmt.Println("Error: NEPHIO_PACKAGE_PATH environment variable not set")
        os.Exit(1)
    }

    // Use strict configuration for CI/CD
    config := security.SecurityIntegrationConfig{
        EnableScanning:   true,
        StrictMode:       true,
        GenerateReports:  true,
        ReportOutputPath: os.Getenv("CI_PROJECT_DIR") + "/security-report.md",
        FailOnViolations: true,
        PodSecurityLevel: "Baseline",
        BlockCritical:    true,
        BlockHigh:        true,
    }

    renderer := security.NewSecurePackageRenderer(config)
    result, err := renderer.ValidatePackage(packagePath)

    if result != nil {
        security.PrintValidationResult(result)
    }

    if err != nil {
        fmt.Printf("\n❌ CI Security Check FAILED\n")
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }

    fmt.Println("\n✅ CI Security Check PASSED")
    os.Exit(0)
}
```

### GitLab CI Configuration

```yaml
# .gitlab-ci.yml
security-validation:
  stage: validate
  image: golang:1.21
  before_script:
    - cd work_dir/security
    - go build -o ../../ci-security-check ./examples/ci-security-check.go
  script:
    - ../../ci-security-check
  artifacts:
    when: always
    paths:
      - security-report.md
    reports:
      junit: security-report.xml
  variables:
    NEPHIO_PACKAGE_PATH: "nephio-packages/production-deployment"
```

## Example 8: Real-time Monitoring

```go
package main

import (
    "fmt"
    "path/filepath"
    "time"

    "github.com/fsnotify/fsnotify"
    "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/work_dir/security"
)

func main() {
    watchDir := "nephio-packages"
    scanner := security.NewSecurityScanner()

    // Create file watcher
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        panic(err)
    }
    defer watcher.Close()

    fmt.Printf("Watching for changes in: %s\n", watchDir)

    go func() {
        for {
            select {
            case event := <-watcher.Events:
                if filepath.Ext(event.Name) == ".yaml" || filepath.Ext(event.Name) == ".yml" {
                    fmt.Printf("\n📄 File changed: %s\n", event.Name)

                    // Re-scan package
                    pkgPath := filepath.Dir(event.Name)
                    err := scanner.ScanPackage(pkgPath)
                    if err != nil {
                        fmt.Printf("❌ Scan failed: %v\n", err)
                        continue
                    }

                    violations := scanner.GetViolations()
                    if len(violations) > 0 {
                        fmt.Printf("⚠️  Found %d security violations\n", len(violations))
                    } else {
                        fmt.Println("✅ No security violations")
                    }
                }
            case err := <-watcher.Errors:
                fmt.Printf("Error: %v\n", err)
            }
        }
    }()

    err = watcher.Add(watchDir)
    if err != nil {
        panic(err)
    }

    // Keep running
    for {
        time.Sleep(time.Second)
    }
}
```

## Example 9: Custom Remediation Workflow

```go
package main

import (
    "fmt"
    "io/ioutil"
    "strings"

    "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/work_dir/security"
)

func main() {
    scanner := security.NewSecurityScanner()
    scanner.ScanPackage("deployments/test-app")

    violations := scanner.GetViolations()

    // Group violations by file
    fileViolations := make(map[string][]security.SecurityViolation)
    for _, v := range violations {
        fileViolations[v.Resource] = append(fileViolations[v.Resource], v)
    }

    // Generate remediation guide for each file
    fmt.Println("Remediation Guide")
    fmt.Println("=" + strings.Repeat("=", 79))

    for file, viols := range fileViolations {
        fmt.Printf("\nFile: %s\n", file)
        fmt.Printf("Violations: %d\n\n", len(viols))

        for i, v := range viols {
            fmt.Printf("%d. [%s] %s\n", i+1, v.Severity, v.Type)
            fmt.Printf("   Problem: %s\n", v.Message)
            fmt.Printf("   Fix: %s\n\n", v.Remediation)
        }

        // Offer to create patch file
        fmt.Printf("Would you like to generate a patch file? (y/n): ")
        // In real implementation, wait for user input
        // generatePatchFile(file, viols)
    }
}

func generatePatchFile(file string, violations []security.SecurityViolation) {
    patchContent := "# Security Patch for " + file + "\n\n"

    for _, v := range violations {
        patchContent += fmt.Sprintf("## Fix: %s\n", v.Type)
        patchContent += fmt.Sprintf("```\n%s\n```\n\n", v.Remediation)
    }

    patchFile := file + ".patch"
    ioutil.WriteFile(patchFile, []byte(patchContent), 0644)

    fmt.Printf("Patch file created: %s\n", patchFile)
}
```

## Example 10: Kubernetes Admission Webhook

```go
package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"

    "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/work_dir/security"
    admissionv1 "k8s.io/api/admission/v1"
)

func main() {
    http.HandleFunc("/validate", validatePod)

    fmt.Println("Starting security admission webhook on :8443")
    http.ListenAndServeTLS(":8443", "cert.pem", "key.pem", nil)
}

func validatePod(w http.ResponseWriter, r *http.Request) {
    body, _ := ioutil.ReadAll(r.Body)

    var admissionReview admissionv1.AdmissionReview
    json.Unmarshal(body, &admissionReview)

    // Create temp directory with the pod manifest
    // Scan it with security scanner
    scanner := security.NewSecurityScanner()

    // For demo purposes, assume scanning succeeds
    // In real implementation, write pod manifest to temp file and scan
    violations := scanner.GetViolations()

    enforcer := security.NewPolicyEnforcer(true)
    err := enforcer.Enforce(violations)

    // Create admission response
    response := &admissionv1.AdmissionResponse{
        UID: admissionReview.Request.UID,
    }

    if err != nil {
        response.Allowed = false
        response.Result = &metav1.Status{
            Message: fmt.Sprintf("Security policy violation: %v", err),
        }
    } else {
        response.Allowed = true
    }

    admissionReview.Response = response
    responseBytes, _ := json.Marshal(admissionReview)

    w.Header().Set("Content-Type", "application/json")
    w.Write(responseBytes)
}
```

These examples demonstrate the full range of security scanner capabilities, from basic usage to advanced integration patterns.