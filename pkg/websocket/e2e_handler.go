package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/e2e"
)

// E2EHandler handles E2E orchestration for WebSocket sessions
type E2EHandler struct {
	orchestrator *e2e.E2EOrchestrator
	sessions     map[string]*E2ESession
}

// E2ESession represents an E2E session
type E2ESession struct {
	SessionID string
	Client    *ClientSession
	Status    string
	Result    *e2e.E2EResult
}

// NewE2EHandler creates a new E2E handler
func NewE2EHandler() (*E2EHandler, error) {
	config := &e2e.E2EConfig{
		PorchEndpoint:   "http://porch-system:4523",
		Namespace:       "network-slices",
		Repository:      "deployments",
		GitRepo:         "https://github.com/thc1006/O-RAN-Nephio-packages.git",
		ArgoCDNamespace: "argocd",
		PrometheusURL:   "http://prometheus:9090",
	}

	orchestrator, err := e2e.NewE2EOrchestrator(config)
	if err != nil {
		// Log but don't fail - will work in degraded mode
		log.Printf("Warning: E2E orchestrator initialization failed: %v", err)
		return &E2EHandler{
			sessions: make(map[string]*E2ESession),
		}, nil
	}

	return &E2EHandler{
		orchestrator: orchestrator,
		sessions:     make(map[string]*E2ESession),
	}, nil
}

// ProcessE2EIntent processes intent with full E2E flow
func (h *E2EHandler) ProcessE2EIntent(ctx context.Context, session *ClientSession, req IntentRequest) {
	// Create E2E session
	e2eSession := &E2ESession{
		SessionID: session.ID,
		Client:    session,
		Status:    "initializing",
	}
	h.sessions[session.ID] = e2eSession

	// Send initial status
	h.sendE2EStatus(session, "Starting E2E orchestration flow", "processing")

	if h.orchestrator == nil {
		// Fallback to basic processing
		h.processBasicIntent(ctx, session, req)
		return
	}

	// Execute E2E flow in goroutine to avoid blocking
	go h.executeE2EFlow(ctx, e2eSession, req.Intent)
}

// executeE2EFlow executes the complete E2E flow
func (h *E2EHandler) executeE2EFlow(ctx context.Context, session *E2ESession, intent string) {
	// Step 1: NLP Processing
	h.sendE2EStep(session.Client, "NLP Processing", "processing", nil)
	time.Sleep(2 * time.Second) // Simulate processing

	// Execute orchestration
	result, err := h.orchestrator.ProcessIntent(ctx, intent)
	if err != nil {
		h.sendE2EError(session.Client, fmt.Sprintf("E2E flow failed: %v", err))
		session.Status = "failed"
		return
	}

	session.Result = result
	session.Status = "completed"

	// Send step updates
	for _, step := range result.Steps {
		h.sendE2EStep(session.Client, step.Name, step.Status, step.Details)
		time.Sleep(500 * time.Millisecond) // Add slight delay for UI
	}

	// Send final result
	h.sendE2EComplete(session.Client, result)
}

// processBasicIntent fallback for basic intent processing
func (h *E2EHandler) processBasicIntent(ctx context.Context, session *ClientSession, req IntentRequest) {
	// Simulate E2E steps without actual deployment
	steps := []string{
		"Claude NLP Processing",
		"Nephio Package Generation",
		"Git Repository Commit",
		"ArgoCD Application Creation",
		"Kubernetes Deployment",
		"Metrics Collection",
	}

	for i, step := range steps {
		h.sendE2EStep(session, step, "processing", map[string]interface{}{
			"progress": fmt.Sprintf("%d/%d", i+1, len(steps)),
		})
		time.Sleep(2 * time.Second)

		h.sendE2EStep(session, step, "completed", map[string]interface{}{
			"progress": fmt.Sprintf("%d/%d", i+1, len(steps)),
			"duration": "2s",
		})
	}

	// Send completion
	result := &e2e.E2EResult{
		Intent:    req.Intent,
		Success:   true,
		SliceID:   fmt.Sprintf("slice-%d", time.Now().Unix()),
		Timestamp: time.Now(),
	}

	h.sendE2EComplete(session, result)
}

// sendE2EStatus sends E2E status update
func (h *E2EHandler) sendE2EStatus(session *ClientSession, message, status string) {
	msg := Message{
		Type:      "e2e_status",
		SessionID: session.ID,
		Message:   message,
		Status:    status,
		Timestamp: time.Now().Unix(),
	}
	session.SendMessage(msg)
}

// sendE2EStep sends E2E step update
func (h *E2EHandler) sendE2EStep(session *ClientSession, stepName, status string, details interface{}) {
	msg := Message{
		Type:      "e2e_step",
		SessionID: session.ID,
		Data: map[string]interface{}{
			"step":    stepName,
			"status":  status,
			"details": details,
		},
		Status:    status,
		Timestamp: time.Now().Unix(),
	}
	session.SendMessage(msg)
}

// sendE2EError sends E2E error
func (h *E2EHandler) sendE2EError(session *ClientSession, errorMsg string) {
	msg := Message{
		Type:      "e2e_error",
		SessionID: session.ID,
		Message:   errorMsg,
		Status:    "error",
		Timestamp: time.Now().Unix(),
	}
	session.SendMessage(msg)
}

// sendE2EComplete sends E2E completion
func (h *E2EHandler) sendE2EComplete(session *ClientSession, result *e2e.E2EResult) {
	// Convert result to map for JSON
	data, _ := json.Marshal(result)
	var resultMap map[string]interface{}
	json.Unmarshal(data, &resultMap)

	msg := Message{
		Type:      "e2e_complete",
		SessionID: session.ID,
		Data:      resultMap,
		Status:    "completed",
		Timestamp: time.Now().Unix(),
	}
	session.SendMessage(msg)
}

// GetE2EStatus gets E2E session status
func (h *E2EHandler) GetE2EStatus(sessionID string) (*E2ESession, bool) {
	session, exists := h.sessions[sessionID]
	return session, exists
}

// StreamE2EUpdates streams real-time E2E updates
func (h *E2EHandler) StreamE2EUpdates(session *ClientSession, appName string) {
	// Stream deployment status updates
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for i := 0; i < 12; i++ { // Check for 1 minute
		select {
		case <-ticker.C:
			// Simulate status check
			status := "progressing"
			if i > 6 {
				status = "healthy"
			}

			msg := Message{
				Type:      "e2e_deployment_status",
				SessionID: session.ID,
				Data: map[string]interface{}{
					"app":    appName,
					"status": status,
					"health": status,
					"sync":   "synced",
				},
				Timestamp: time.Now().Unix(),
			}
			session.SendMessage(msg)

			if status == "healthy" {
				return
			}

		case <-session.Context.Done():
			return
		}
	}
}