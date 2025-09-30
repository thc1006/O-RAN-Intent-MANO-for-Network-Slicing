// Package e2e provides helper functions for end-to-end integration tests
package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Helper methods for E2ETestSuite

// processNLPIntent sends intent to NLP module and returns parsed result
func (s *E2ETestSuite) processNLPIntent(intent string) (*ParsedIntent, error) {
	url := fmt.Sprintf("%s/api/v1/nlp/parse", s.orchestratorURL)

	payload := map[string]string{"intent": intent}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal intent: %w", err)
	}

	req, err := http.NewRequestWithContext(s.ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("NLP processing failed: %s (status %d)", string(body), resp.StatusCode)
	}

	var parsedIntent ParsedIntent
	if err := json.NewDecoder(resp.Body).Decode(&parsedIntent); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &parsedIntent, nil
}

// analyzeWithClaude sends intent to Claude AI for analysis
func (s *E2ETestSuite) analyzeWithClaude(ctx context.Context, intent string) (*Analysis, error) {
	url := fmt.Sprintf("%s/api/v1/claude/analyze", s.orchestratorURL)

	payload := map[string]string{"intent": intent}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal intent: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Claude analysis failed: %s (status %d)", string(body), resp.StatusCode)
	}

	var analysis Analysis
	if err := json.NewDecoder(resp.Body).Decode(&analysis); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &analysis, nil
}

// calculatePlacement requests orchestrator to calculate optimal placement
func (s *E2ETestSuite) calculatePlacement(ctx context.Context) (*Placement, error) {
	url := fmt.Sprintf("%s/api/v1/orchestrator/placement", s.orchestratorURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("placement calculation failed: %s (status %d)", string(body), resp.StatusCode)
	}

	var placement Placement
	if err := json.NewDecoder(resp.Body).Decode(&placement); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &placement, nil
}

// calculatePlacementWithConstraints calculates placement with specific resource constraints
func (s *E2ETestSuite) calculatePlacementWithConstraints(ctx context.Context, constraints ResourceConstraints) (*Placement, error) {
	url := fmt.Sprintf("%s/api/v1/orchestrator/placement", s.orchestratorURL)

	jsonData, err := json.Marshal(constraints)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal constraints: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("placement calculation failed: %s (status %d)", string(body), resp.StatusCode)
	}

	var placement Placement
	if err := json.NewDecoder(resp.Body).Decode(&placement); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &placement, nil
}

// generateNephioPackages generates Nephio packages using the renderer
func (s *E2ETestSuite) generateNephioPackages(ctx context.Context) ([]*Package, error) {
	url := fmt.Sprintf("%s/api/v1/nephio/generate", s.nephioRendererURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("package generation failed: %s (status %d)", string(body), resp.StatusCode)
	}

	var packages []*Package
	if err := json.NewDecoder(resp.Body).Decode(&packages); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return packages, nil
}

// generateNephioPackagesWithO2Mock generates packages with mocked O2 inventory
func (s *E2ETestSuite) generateNephioPackagesWithO2Mock(ctx context.Context) ([]*Package, error) {
	url := fmt.Sprintf("%s/api/v1/nephio/generate?mock=o2", s.nephioRendererURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("package generation failed: %s (status %d)", string(body), resp.StatusCode)
	}

	var packages []*Package
	if err := json.NewDecoder(resp.Body).Decode(&packages); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return packages, nil
}

// deployVNFs initiates VNF deployment
func (s *E2ETestSuite) deployVNFs(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/api/v1/vnf/deploy", s.vnfOperatorURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("VNF deployment failed: %s (status %d)", string(body), resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	deploymentID, ok := result["deployment_id"]
	if !ok {
		return "", fmt.Errorf("deployment_id not found in response")
	}

	return deploymentID, nil
}

// deployVNFsWithConfig deploys VNFs with custom configuration
func (s *E2ETestSuite) deployVNFsWithConfig(ctx context.Context, config interface{}) (string, error) {
	url := fmt.Sprintf("%s/api/v1/vnf/deploy", s.vnfOperatorURL)

	jsonData, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("VNF deployment failed: %s (status %d)", string(body), resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result["deployment_id"], nil
}

// waitForDeploymentReady polls deployment status until ready or timeout
func (s *E2ETestSuite) waitForDeploymentReady(ctx context.Context, deploymentID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for deployment %s to become ready", deploymentID)
			}

			status, err := s.getDeploymentStatus(ctx, deploymentID)
			if err != nil {
				continue // Retry on error
			}

			if status.State == "Ready" {
				return nil
			}

			if status.State == "Failed" || status.State == "RolledBack" {
				return fmt.Errorf("deployment %s failed with state: %s", deploymentID, status.State)
			}
		}
	}
}

// getDeploymentStatus retrieves current deployment status
func (s *E2ETestSuite) getDeploymentStatus(ctx context.Context, deploymentID string) (*DeploymentStatus, error) {
	url := fmt.Sprintf("%s/api/v1/vnf/status/%s", s.vnfOperatorURL, deploymentID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get status: %s (status %d)", string(body), resp.StatusCode)
	}

	var status DeploymentStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &status, nil
}

// configureTN configures transport network
func (s *E2ETestSuite) configureTN(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/v1/tn/configure", s.tnAgentURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("TN configuration failed: %s (status %d)", string(body), resp.StatusCode)
	}

	return nil
}

// listVXLANInterfaces retrieves list of VXLAN interfaces
func (s *E2ETestSuite) listVXLANInterfaces(ctx context.Context) ([]*VXLANInterface, error) {
	url := fmt.Sprintf("%s/api/v1/tn/vxlan/interfaces", s.tnAgentURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list interfaces: %s (status %d)", string(body), resp.StatusCode)
	}

	var interfaces []*VXLANInterface
	if err := json.NewDecoder(resp.Body).Decode(&interfaces); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return interfaces, nil
}

// getSliceStatus retrieves operational status of network slice
func (s *E2ETestSuite) getSliceStatus(ctx context.Context, sliceID string) (*SliceStatus, error) {
	url := fmt.Sprintf("%s/api/v1/slice/status/%s", s.orchestratorURL, sliceID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get slice status: %s (status %d)", string(body), resp.StatusCode)
	}

	var status SliceStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &status, nil
}

// testOrchestratorVNFIntegration tests orchestrator-VNF integration
func (s *E2ETestSuite) testOrchestratorVNFIntegration(ctx context.Context) (*Placement, error) {
	// This would test the integration between orchestrator and VNF operator
	// For now, use existing calculatePlacement
	return s.calculatePlacement(ctx)
}

// testTNAgentVXLANIntegration tests TN agent-VXLAN integration
func (s *E2ETestSuite) testTNAgentVXLANIntegration(ctx context.Context) error {
	// Configure TN and verify VXLAN interfaces
	if err := s.configureTN(ctx); err != nil {
		return err
	}

	interfaces, err := s.listVXLANInterfaces(ctx)
	if err != nil {
		return err
	}

	if len(interfaces) == 0 {
		return fmt.Errorf("no VXLAN interfaces found after configuration")
	}

	return nil
}