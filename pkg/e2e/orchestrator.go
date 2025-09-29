package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/claude"
	"gopkg.in/yaml.v3"
)

// E2EOrchestrator handles complete flow from NLP to ArgoCD deployment
type E2EOrchestrator struct {
	claudeClient  *claude.Client
	gitRepo       string
	argocdNS      string
	prometheusURL string
	namespace     string
	repository    string
}

// NewE2EOrchestrator creates a new E2E orchestrator
func NewE2EOrchestrator(config *E2EConfig) (*E2EOrchestrator, error) {
	claudeClient, err := claude.NewClient(context.Background(), &claude.ClientConfig{
		SessionName: "e2e-orchestrator",
		Timeout:     30 * time.Second,
		UseFallback: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Claude client: %w", err)
	}

	return &E2EOrchestrator{
		claudeClient:  claudeClient,
		gitRepo:       config.GitRepo,
		argocdNS:      config.ArgoCDNamespace,
		prometheusURL: config.PrometheusURL,
		namespace:     config.Namespace,
		repository:    config.Repository,
	}, nil
}

// ProcessIntent handles the complete E2E flow
func (o *E2EOrchestrator) ProcessIntent(ctx context.Context, intent string) (*E2EResult, error) {
	result := &E2EResult{
		Intent:    intent,
		Timestamp: time.Now(),
		Steps:     []StepResult{},
	}

	// Step 1: Parse intent with Claude
	claudeResp, err := o.processWithClaude(ctx, intent)
	if err != nil {
		return result, fmt.Errorf("Claude processing failed: %w", err)
	}
	result.Steps = append(result.Steps, StepResult{
		Name:      "Claude NLP Processing",
		Status:    "completed",
		Details:   claudeResp,
		Timestamp: time.Now(),
	})

	// Step 2: Generate Nephio package
	packagePath, err := o.generateNephioPackage(ctx, claudeResp)
	if err != nil {
		return result, fmt.Errorf("package generation failed: %w", err)
	}
	result.Steps = append(result.Steps, StepResult{
		Name:      "Nephio Package Generation",
		Status:    "completed",
		Details:   map[string]string{"path": packagePath},
		Timestamp: time.Now(),
	})

	// Step 3: Commit to Git
	commitHash, err := o.commitToGit(ctx, packagePath, claudeResp)
	if err != nil {
		return result, fmt.Errorf("git commit failed: %w", err)
	}
	result.Steps = append(result.Steps, StepResult{
		Name:      "Git Commit",
		Status:    "completed",
		Details:   map[string]string{"commit": commitHash},
		Timestamp: time.Now(),
	})

	// Step 4: Create/Update ArgoCD Application
	appName, err := o.createArgoCDApp(ctx, claudeResp, packagePath)
	if err != nil {
		return result, fmt.Errorf("ArgoCD app creation failed: %w", err)
	}
	result.Steps = append(result.Steps, StepResult{
		Name:      "ArgoCD Application",
		Status:    "completed",
		Details:   map[string]string{"app": appName},
		Timestamp: time.Now(),
	})

	// Step 5: Trigger ArgoCD Sync
	syncStatus, err := o.syncArgoCD(ctx, appName)
	if err != nil {
		return result, fmt.Errorf("ArgoCD sync failed: %w", err)
	}
	result.Steps = append(result.Steps, StepResult{
		Name:      "ArgoCD Sync",
		Status:    syncStatus,
		Details:   map[string]string{"app": appName},
		Timestamp: time.Now(),
	})

	// Step 6: Wait for deployment
	deployStatus, err := o.waitForDeployment(ctx, appName, 5*time.Minute)
	if err != nil {
		return result, fmt.Errorf("deployment failed: %w", err)
	}
	result.Steps = append(result.Steps, StepResult{
		Name:      "Kubernetes Deployment",
		Status:    deployStatus.Status,
		Details:   deployStatus,
		Timestamp: time.Now(),
	})

	// Step 7: Collect metrics
	metrics, err := o.collectMetrics(ctx, claudeResp.ParsedIntent.SliceType)
	if err != nil {
		// Metrics are optional, don't fail the flow
		result.Steps = append(result.Steps, StepResult{
			Name:      "Metrics Collection",
			Status:    "warning",
			Details:   map[string]string{"error": err.Error()},
			Timestamp: time.Now(),
		})
	} else {
		result.Steps = append(result.Steps, StepResult{
			Name:      "Metrics Collection",
			Status:    "completed",
			Details:   metrics,
			Timestamp: time.Now(),
		})
	}

	result.Success = true
	result.SliceID = generateSliceID(claudeResp.ParsedIntent.SliceType)
	result.DeploymentStatus = deployStatus

	return result, nil
}

// processWithClaude processes intent with Claude CLI
func (o *E2EOrchestrator) processWithClaude(ctx context.Context, intent string) (*claude.IntentResponse, error) {
	req := &claude.IntentRequest{
		Text: intent,
	}
	return o.claudeClient.ProcessIntent(ctx, req)
}

// generateNephioPackage creates Nephio R4 package from Claude response
func (o *E2EOrchestrator) generateNephioPackage(ctx context.Context, claudeResp *claude.IntentResponse) (string, error) {
	// Create package directory
	packageName := fmt.Sprintf("%s-slice-%d",
		claudeResp.ParsedIntent.SliceType,
		time.Now().Unix())
	packagePath := filepath.Join("/tmp", "nephio-packages", packageName)

	if err := os.MkdirAll(packagePath, 0755); err != nil {
		return "", err
	}

	// Generate Kptfile
	kptfile := &KptFile{
		APIVersion: "kpt.dev/v1",
		Kind:       "Kptfile",
		Metadata: KptMetadata{
			Name:      packageName,
			Namespace: "network-slices",
			Annotations: map[string]string{
				"config.kubernetes.io/local-config": "true",
				"nephio.org/cluster-name":           "edge-cluster",
			},
		},
		Info: KptInfo{
			Description: fmt.Sprintf("%s network slice generated from: %s",
				claudeResp.ParsedIntent.SliceType, claudeResp.Response),
			Keywords: []string{
				claudeResp.ParsedIntent.SliceType,
				"5g",
				"network-slice",
			},
		},
		Pipeline: KptPipeline{
			Mutators: []KptFunction{
				{
					Image: "gcr.io/nephio/slice-config-fn:v1.0.0",
					ConfigMap: map[string]interface{}{
						"sliceType": claudeResp.ParsedIntent.SliceType,
						"qos": map[string]interface{}{
							"throughput":  claudeResp.ParsedIntent.Requirements.Throughput,
							"latency":     claudeResp.ParsedIntent.Requirements.Latency,
							"reliability": claudeResp.ParsedIntent.Requirements.Reliability,
						},
					},
				},
			},
		},
	}

	kptfileYAML, err := yaml.Marshal(kptfile)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(filepath.Join(packagePath, "Kptfile"), kptfileYAML, 0644); err != nil {
		return "", err
	}

	// Generate NetworkSlice CR
	networkSlice := &NetworkSliceCR{
		APIVersion: "workload.nephio.org/v1alpha1",
		Kind:       "NetworkSlice",
		Metadata: Metadata{
			Name:      packageName,
			Namespace: "network-slices",
			Labels: map[string]string{
				"slice-type":    claudeResp.ParsedIntent.SliceType,
				"generated-by":  "claude-e2e",
				"intent-driven": "true",
			},
		},
		Spec: NetworkSliceSpec{
			SliceType: claudeResp.ParsedIntent.SliceType,
			QoS: QoSSpec{
				Throughput:  claudeResp.ParsedIntent.Requirements.Throughput,
				Latency:     claudeResp.ParsedIntent.Requirements.Latency,
				Reliability: claudeResp.ParsedIntent.Requirements.Reliability,
			},
			NetworkFunctions: generateNetworkFunctions(claudeResp.ParsedIntent.SliceType),
			Capacity: CapacitySpec{
				MaxSessions: getMaxSessions(claudeResp.ParsedIntent.SliceType),
			},
		},
	}

	sliceYAML, err := yaml.Marshal(networkSlice)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(filepath.Join(packagePath, "network-slice.yaml"), sliceYAML, 0644); err != nil {
		return "", err
	}

	// Create README
	readme := fmt.Sprintf(`# %s Network Slice

Generated from natural language intent: "%s"

## Configuration
- Slice Type: %s
- Throughput: %d Mbps
- Latency: %.1f ms
- Reliability: %.3f%%

## Deployment
This package will be deployed via ArgoCD to the edge cluster.
`, packageName, claudeResp.Response, claudeResp.ParsedIntent.SliceType,
		claudeResp.ParsedIntent.Requirements.Throughput,
		claudeResp.ParsedIntent.Requirements.Latency,
		claudeResp.ParsedIntent.Requirements.Reliability)

	if err := os.WriteFile(filepath.Join(packagePath, "README.md"), []byte(readme), 0644); err != nil {
		return "", err
	}

	return packagePath, nil
}

// commitToGit commits package to Git repository
func (o *E2EOrchestrator) commitToGit(ctx context.Context, packagePath string, claudeResp *claude.IntentResponse) (string, error) {
	// Create branch
	branchName := fmt.Sprintf("slice-%s-%d", claudeResp.ParsedIntent.SliceType, time.Now().Unix())

	// Initialize git in package directory
	cmd := exec.CommandContext(ctx, "git", "init")
	cmd.Dir = packagePath
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git init failed: %w", err)
	}

	// Add remote
	cmd = exec.CommandContext(ctx, "git", "remote", "add", "origin", o.gitRepo)
	cmd.Dir = packagePath
	if err := cmd.Run(); err != nil {
		// Remote might already exist
	}

	// Create and checkout branch
	cmd = exec.CommandContext(ctx, "git", "checkout", "-b", branchName)
	cmd.Dir = packagePath
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git checkout failed: %w", err)
	}

	// Add files
	cmd = exec.CommandContext(ctx, "git", "add", ".")
	cmd.Dir = packagePath
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git add failed: %w", err)
	}

	// Commit
	commitMsg := fmt.Sprintf("Add %s network slice from intent: %s",
		claudeResp.ParsedIntent.SliceType,
		claudeResp.Response)
	cmd = exec.CommandContext(ctx, "git", "commit", "-m", commitMsg)
	cmd.Dir = packagePath
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git commit failed: %w", err)
	}

	// Get commit hash
	cmd = exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = packagePath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse failed: %w", err)
	}

	commitHash := string(output)[:8]

	// Push to remote (this might fail in test environment)
	cmd = exec.CommandContext(ctx, "git", "push", "origin", branchName)
	cmd.Dir = packagePath
	cmd.Run() // Ignore error as this might fail in test

	return commitHash, nil
}

// createArgoCDApp creates or updates ArgoCD application
func (o *E2EOrchestrator) createArgoCDApp(ctx context.Context, claudeResp *claude.IntentResponse, packagePath string) (string, error) {
	appName := fmt.Sprintf("%s-slice-%d",
		claudeResp.ParsedIntent.SliceType,
		time.Now().Unix())

	// Create ArgoCD Application manifest
	argoApp := &ArgoCDApplication{
		APIVersion: "argoproj.io/v1alpha1",
		Kind:       "Application",
		Metadata: Metadata{
			Name:      appName,
			Namespace: o.argocdNS,
			Labels: map[string]string{
				"slice-type": claudeResp.ParsedIntent.SliceType,
				"e2e-test":   "true",
			},
		},
		Spec: ArgoCDAppSpec{
			Project: "default",
			Source: ArgoCDSource{
				RepoURL:        o.gitRepo,
				Path:           filepath.Base(packagePath),
				TargetRevision: "HEAD",
			},
			Destination: ArgoCDDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: "network-slices",
			},
			SyncPolicy: &ArgoCDSyncPolicy{
				Automated: &ArgoCDAutomated{
					Prune:      true,
					SelfHeal:   true,
					AllowEmpty: false,
				},
				SyncOptions: []string{
					"CreateNamespace=true",
					"PrunePropagationPolicy=foreground",
				},
				Retry: &ArgoCDRetry{
					Limit: 5,
					Backoff: &ArgoCDBackoff{
						Duration:    "5s",
						Factor:      2,
						MaxDuration: "3m",
					},
				},
			},
		},
	}

	// Convert to YAML
	appYAML, err := yaml.Marshal(argoApp)
	if err != nil {
		return "", err
	}

	// Save to file
	appPath := filepath.Join("/tmp", "argocd-apps", appName+".yaml")
	os.MkdirAll(filepath.Dir(appPath), 0755)
	if err := os.WriteFile(appPath, appYAML, 0644); err != nil {
		return "", err
	}

	// Apply to cluster (kubectl apply)
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", appPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Log but don't fail - might not have cluster access
		fmt.Printf("ArgoCD app apply warning: %s\n", output)
	}

	return appName, nil
}

// syncArgoCD triggers ArgoCD sync
func (o *E2EOrchestrator) syncArgoCD(ctx context.Context, appName string) (string, error) {
	// Trigger sync via ArgoCD CLI
	cmd := exec.CommandContext(ctx, "argocd", "app", "sync", appName,
		"--namespace", o.argocdNS)

	if output, err := cmd.CombinedOutput(); err != nil {
		// Try kubectl as fallback
		patchCmd := exec.CommandContext(ctx, "kubectl", "patch", "application", appName,
			"-n", o.argocdNS,
			"--type", "merge",
			"-p", `{"metadata":{"annotations":{"argocd.argoproj.io/sync":"true"}}}`)

		if _, patchErr := patchCmd.CombinedOutput(); patchErr != nil {
			return "pending", fmt.Errorf("sync trigger failed: %s", output)
		}
	}

	return "syncing", nil
}

// waitForDeployment waits for deployment to complete
func (o *E2EOrchestrator) waitForDeployment(ctx context.Context, appName string, timeout time.Duration) (*DeploymentStatus, error) {
	deadline := time.Now().Add(timeout)

	status := &DeploymentStatus{
		AppName:   appName,
		Namespace: "network-slices",
		Status:    "pending",
	}

	for time.Now().Before(deadline) {
		// Check ArgoCD app status
		cmd := exec.CommandContext(ctx, "argocd", "app", "get", appName,
			"--namespace", o.argocdNS,
			"--output", "json")

		output, err := cmd.Output()
		if err != nil {
			// Try kubectl
			kubectlCmd := exec.CommandContext(ctx, "kubectl", "get", "application", appName,
				"-n", o.argocdNS,
				"-o", "json")

			if kubectlOutput, kubectlErr := kubectlCmd.Output(); kubectlErr == nil {
				output = kubectlOutput
			} else {
				time.Sleep(10 * time.Second)
				continue
			}
		}

		// Parse status
		var appStatus map[string]interface{}
		if err := json.Unmarshal(output, &appStatus); err == nil {
			if statusField, ok := appStatus["status"].(map[string]interface{}); ok {
				if health, ok := statusField["health"].(map[string]interface{}); ok {
					status.Health = health["status"].(string)
				}
				if sync, ok := statusField["sync"].(map[string]interface{}); ok {
					status.Sync = sync["status"].(string)
				}

				// Check if healthy and synced
				if status.Health == "Healthy" && status.Sync == "Synced" {
					status.Status = "deployed"
					status.Ready = true
					return status, nil
				}
			}
		}

		time.Sleep(10 * time.Second)
	}

	status.Status = "timeout"
	return status, fmt.Errorf("deployment timeout after %v", timeout)
}

// collectMetrics collects metrics from Prometheus
func (o *E2EOrchestrator) collectMetrics(ctx context.Context, sliceType string) (*Metrics, error) {
	metrics := &Metrics{
		SliceType: sliceType,
		Timestamp: time.Now(),
	}

	// Query Prometheus for metrics
	queries := map[string]string{
		"throughput": fmt.Sprintf(`rate(slice_throughput_bytes{slice_type="%s"}[5m])`, sliceType),
		"latency":    fmt.Sprintf(`slice_latency_ms{slice_type="%s"}`, sliceType),
		"sessions":   fmt.Sprintf(`slice_active_sessions{slice_type="%s"}`, sliceType),
	}

	for metric, query := range queries {
		cmd := exec.CommandContext(ctx, "curl", "-s", "-G",
			fmt.Sprintf("%s/api/v1/query", o.prometheusURL),
			"--data-urlencode", fmt.Sprintf("query=%s", query))

		output, err := cmd.Output()
		if err != nil {
			continue // Skip if metric not available
		}

		var result map[string]interface{}
		if err := json.Unmarshal(output, &result); err == nil {
			// Parse metric value
			if data, ok := result["data"].(map[string]interface{}); ok {
				if results, ok := data["result"].([]interface{}); ok && len(results) > 0 {
					if res, ok := results[0].(map[string]interface{}); ok {
						if value, ok := res["value"].([]interface{}); ok && len(value) > 1 {
							switch metric {
							case "throughput":
								metrics.Throughput = fmt.Sprintf("%v", value[1])
							case "latency":
								metrics.Latency = fmt.Sprintf("%v", value[1])
							case "sessions":
								metrics.ActiveSessions = fmt.Sprintf("%v", value[1])
							}
						}
					}
				}
			}
		}
	}

	return metrics, nil
}

// Helper functions

func generateSliceID(sliceType string) string {
	return fmt.Sprintf("%s-%s", sliceType, uuid.New().String()[:8])
}

func generateNetworkFunctions(sliceType string) []NetworkFunction {
	switch sliceType {
	case "eMBB":
		return []NetworkFunction{
			{Name: "amf", Replicas: 2},
			{Name: "smf", Replicas: 2},
			{Name: "upf", Replicas: 3},
			{Name: "pcf", Replicas: 1},
		}
	case "URLLC":
		return []NetworkFunction{
			{Name: "amf", Replicas: 3},
			{Name: "smf", Replicas: 3},
			{Name: "upf", Replicas: 4},
			{Name: "pcf", Replicas: 2},
		}
	case "mIoT":
		return []NetworkFunction{
			{Name: "amf", Replicas: 4},
			{Name: "smf", Replicas: 2},
			{Name: "upf", Replicas: 2},
			{Name: "nef", Replicas: 2},
		}
	default:
		return []NetworkFunction{
			{Name: "amf", Replicas: 1},
			{Name: "smf", Replicas: 1},
			{Name: "upf", Replicas: 1},
		}
	}
}

func getMaxSessions(sliceType string) int {
	switch sliceType {
	case "eMBB":
		return 10000
	case "URLLC":
		return 5000
	case "mIoT":
		return 1000000
	default:
		return 1000
	}
}