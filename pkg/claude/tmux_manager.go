package claude

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// TmuxManager manages tmux sessions for Claude CLI interaction
type TmuxManager struct {
	sessionName string
	isActive    bool
	mu          sync.RWMutex
	outputChan  chan string
}

// NewTmuxManager creates a new tmux manager
func NewTmuxManager(sessionName string) *TmuxManager {
	return &TmuxManager{
		sessionName: sessionName,
		outputChan:  make(chan string, 100),
	}
}

// CreateSession creates a new tmux session
func (tm *TmuxManager) CreateSession(ctx context.Context) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Check if tmux is installed
	if err := tm.checkTmuxInstalled(ctx); err != nil {
		return fmt.Errorf("tmux not installed: %w", err)
	}

	// Kill existing session if exists
	_ = tm.killSession(ctx)

	// Create new tmux session
	cmd := exec.CommandContext(ctx, "tmux", "new-session", "-d", "-s", tm.sessionName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	tm.isActive = true
	return nil
}

// SendCommand sends a command to the tmux session
func (tm *TmuxManager) SendCommand(ctx context.Context, command string) error {
	tm.mu.RLock()
	if !tm.isActive {
		tm.mu.RUnlock()
		return fmt.Errorf("tmux session not active")
	}
	tm.mu.RUnlock()

	// Send keys to tmux session
	cmd := exec.CommandContext(ctx, "tmux", "send-keys", "-t", tm.sessionName, command, "Enter")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}

	return nil
}

// CaptureOutput captures output from the tmux session
func (tm *TmuxManager) CaptureOutput(ctx context.Context) (string, error) {
	tm.mu.RLock()
	if !tm.isActive {
		tm.mu.RUnlock()
		return "", fmt.Errorf("tmux session not active")
	}
	tm.mu.RUnlock()

	// Wait a bit for command to execute
	time.Sleep(2 * time.Second)

	// Capture pane content
	cmd := exec.CommandContext(ctx, "tmux", "capture-pane", "-t", tm.sessionName, "-p")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to capture output: %w", err)
	}

	return string(output), nil
}

// ExecuteClaudeCommand executes claude CLI command and returns response
func (tm *TmuxManager) ExecuteClaudeCommand(ctx context.Context, prompt string) (string, error) {
	tm.mu.RLock()
	if !tm.isActive {
		tm.mu.RUnlock()
		return "", fmt.Errorf("tmux session not active")
	}
	tm.mu.RUnlock()

	// Build claude command with proper escaping
	escapedPrompt := strings.ReplaceAll(prompt, `"`, `\"`)
	escapedPrompt = strings.ReplaceAll(escapedPrompt, `'`, `'\''`)
	command := fmt.Sprintf(`claude --dangerously-skip-permissions '%s'`, escapedPrompt)

	// Clear the pane first
	clearCmd := exec.CommandContext(ctx, "tmux", "send-keys", "-t", tm.sessionName, "clear", "Enter")
	if err := clearCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to clear pane: %w", err)
	}
	time.Sleep(500 * time.Millisecond)

	// Send the claude command
	if err := tm.SendCommand(ctx, command); err != nil {
		return "", err
	}

	// Wait for claude to process (adjust timeout as needed)
	time.Sleep(5 * time.Second)

	// Capture the output
	output, err := tm.CaptureOutput(ctx)
	if err != nil {
		return "", err
	}

	// Parse and clean the output
	cleanOutput := tm.parseClaudeOutput(output)
	return cleanOutput, nil
}

// StreamClaudeInteraction starts an interactive Claude session
func (tm *TmuxManager) StreamClaudeInteraction(ctx context.Context) error {
	tm.mu.RLock()
	if !tm.isActive {
		tm.mu.RUnlock()
		return fmt.Errorf("tmux session not active")
	}
	tm.mu.RUnlock()

	// Start claude in interactive mode
	command := "claude --dangerously-skip-permissions"
	if err := tm.SendCommand(ctx, command); err != nil {
		return err
	}

	// Start monitoring output in background
	go tm.monitorOutput(ctx)

	return nil
}

// SendPrompt sends a prompt to interactive Claude session
func (tm *TmuxManager) SendPrompt(ctx context.Context, prompt string) error {
	return tm.SendCommand(ctx, prompt)
}

// GetStreamedOutput returns the output channel for streaming responses
func (tm *TmuxManager) GetStreamedOutput() <-chan string {
	return tm.outputChan
}

// KillSession terminates the tmux session
func (tm *TmuxManager) KillSession(ctx context.Context) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	err := tm.killSession(ctx)
	tm.isActive = false
	close(tm.outputChan)
	return err
}

// Helper methods

func (tm *TmuxManager) checkTmuxInstalled(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "which", "tmux")
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (tm *TmuxManager) killSession(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "tmux", "kill-session", "-t", tm.sessionName)
	_ = cmd.Run() // Ignore error if session doesn't exist
	return nil
}

func (tm *TmuxManager) parseClaudeOutput(output string) string {
	lines := strings.Split(output, "\n")
	var result []string
	var inResponse bool

	for _, line := range lines {
		// Skip command echo and prompts
		if strings.Contains(line, "claude --dangerously-skip-permissions") {
			inResponse = true
			continue
		}
		if strings.HasPrefix(line, "$") || strings.HasPrefix(line, ">") {
			continue
		}
		if inResponse && line != "" {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

func (tm *TmuxManager) monitorOutput(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	lastOutput := ""
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			output, err := tm.CaptureOutput(ctx)
			if err != nil {
				continue
			}

			if output != lastOutput {
				newContent := tm.extractNewContent(lastOutput, output)
				if newContent != "" {
					select {
					case tm.outputChan <- newContent:
					default:
						// Channel full, skip
					}
				}
				lastOutput = output
			}
		}
	}
}

func (tm *TmuxManager) extractNewContent(oldOutput, newOutput string) string {
	if len(newOutput) <= len(oldOutput) {
		return ""
	}
	return newOutput[len(oldOutput):]
}

// AttachToSession attaches to the tmux session for debugging
func (tm *TmuxManager) AttachToSession(ctx context.Context) error {
	tm.mu.RLock()
	if !tm.isActive {
		tm.mu.RUnlock()
		return fmt.Errorf("tmux session not active")
	}
	tm.mu.RUnlock()

	cmd := exec.CommandContext(ctx, "tmux", "attach-session", "-t", tm.sessionName)
	return cmd.Run()
}

// ExecuteWithPipe executes claude with input/output pipes
func (tm *TmuxManager) ExecuteWithPipe(ctx context.Context, prompt string) (string, error) {
	// Alternative approach using pipes instead of tmux
	cmd := exec.CommandContext(ctx, "claude", "--dangerously-skip-permissions")

	// Create pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return "", err
	}

	// Send prompt
	if _, err := fmt.Fprintln(stdin, prompt); err != nil {
		return "", err
	}
	stdin.Close()

	// Read response
	scanner := bufio.NewScanner(stdout)
	var response strings.Builder
	for scanner.Scan() {
		response.WriteString(scanner.Text())
		response.WriteString("\n")
	}

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		return "", err
	}

	return response.String(), nil
}