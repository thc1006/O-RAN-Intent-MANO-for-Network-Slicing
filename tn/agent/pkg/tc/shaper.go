// Package tc provides secure traffic control (TC) management functionality
// All subprocess operations use SecureExecuteWithValidation to prevent command injection
package tc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
)

// Shaper manages TC (Traffic Control) configurations
type Shaper struct {
	// Map of interface to current configuration
	currentConfigs map[string]*Config
}

// Config represents the current TC configuration for an interface
type Config struct {
	Interface string
	Rules     []Rule
}

// Rule represents a single TC rule
type Rule struct {
	Priority int32
	Rate     int    // in kbit
	Burst    int    // in KB
	Latency  float32 // in ms
}

// NewShaper creates a new TC shaper
func NewShaper() *Shaper {
	return &Shaper{
		currentConfigs: make(map[string]*Config),
	}
}

// ApplyRules applies TC rules to an interface
// Security: All subprocess calls use SecureExecuteWithValidation with proper argument validation
// to prevent command injection attacks. Interface names and rule values are validated.
func (s *Shaper) ApplyRules(iface string, rules []Rule) error {
	// Security: Validate interface name to prevent command injection
	if err := security.ValidateNetworkInterface(iface); err != nil {
		return fmt.Errorf("invalid interface name: %w", err)
	}

	// Security: Validate rule parameters
	for i, rule := range rules {
		if rule.Priority < 1 || rule.Priority > 65535 {
			return fmt.Errorf("invalid priority in rule %d: %d (must be 1-65535)", i, rule.Priority)
		}
		if rule.Rate < 1 {
			return fmt.Errorf("invalid rate in rule %d: %d (must be positive)", i, rule.Rate)
		}
		if rule.Burst < 1 {
			return fmt.Errorf("invalid burst in rule %d: %d (must be positive)", i, rule.Burst)
		}
		if rule.Latency < 0 {
			return fmt.Errorf("invalid latency in rule %d: %f (must be non-negative)", i, rule.Latency)
		}
	}

	// Clear existing configuration
	if err := s.clearInterface(iface); err != nil {
		return fmt.Errorf("failed to clear interface %s: %w", iface, err)
	}

	// Apply new rules
	for _, rule := range rules {
		if err := s.applyRule(iface, rule); err != nil {
			return fmt.Errorf("failed to apply rule on %s: %w", iface, err)
		}
	}

	// Store configuration
	s.currentConfigs[iface] = &Config{
		Interface: iface,
		Rules:     rules,
	}

	return nil
}

func (s *Shaper) clearInterface(iface string) error {
	// Clear root qdisc (this removes all child qdiscs)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Security: Using SecureExecuteWithValidation with ValidateTCArgs to prevent command injection
	// All arguments are validated against allowlists and patterns before execution
	rootArgs := []string{"qdisc", "del", "dev", iface, "root"}
	// #nosec G204 - Using security.SecureExecuteWithValidation with argument validation to prevent command injection
	if _, err := security.SecureExecuteWithValidation(ctx, "tc", security.ValidateTCArgs, rootArgs...); err != nil {
		// Ignore errors if no qdisc exists
		if !strings.Contains(err.Error(), "No such file or directory") {
			return fmt.Errorf("failed to clear root qdisc: %v", err)
		}
	}

	// Clear ingress qdisc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Security: Using SecureExecuteWithValidation with ValidateTCArgs to prevent command injection
	ingressArgs := []string{"qdisc", "del", "dev", iface, "ingress"}
	// #nosec G204 - Using security.SecureExecuteWithValidation with argument validation to prevent command injection
	if _, err := security.SecureExecuteWithValidation(ctx, "tc", security.ValidateTCArgs, ingressArgs...); err != nil {
		// Ignore errors if no qdisc exists
		if !strings.Contains(err.Error(), "Invalid argument") &&
			!strings.Contains(err.Error(), "No such file or directory") {
			return fmt.Errorf("failed to clear ingress qdisc: %v", err)
		}
	}

	return nil
}

func (s *Shaper) applyRule(iface string, rule Rule) error {
	// This is a simplified implementation
	// In production, you would use netlink library for more control

	// Add root HTB qdisc if not exists
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Security: Using SecureExecuteWithValidation with ValidateTCArgs to prevent command injection
	// All tc commands are validated against allowlists defined in security package
	qdiscArgs := []string{"qdisc", "add", "dev", iface, "root", "handle", "1:", "htb", "default", "30"}
	// #nosec G204 - Using security.SecureExecuteWithValidation with argument validation to prevent command injection
	if _, err := security.SecureExecuteWithValidation(ctx, "tc", security.ValidateTCArgs, qdiscArgs...); err != nil {
		if !strings.Contains(err.Error(), "exists") {
			return fmt.Errorf("failed to add root qdisc: %v", err)
		}
	}

	// Add class with rate limit
	// Security: classID is formatted using validated rule.Priority (int32), safe from injection
	classID := fmt.Sprintf("1:%d", rule.Priority)
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Security: Using SecureExecuteWithValidation with ValidateTCArgs to prevent command injection
	// All arguments including formatted values are validated before execution
	classArgs := []string{"class", "add", "dev", iface, "parent", "1:",
		"classid", classID, "htb",
		"rate", fmt.Sprintf("%dkbit", rule.Rate),
		"burst", fmt.Sprintf("%dk", rule.Burst)}

	// #nosec G204 - Using security.SecureExecuteWithValidation with argument validation to prevent command injection
	if _, err := security.SecureExecuteWithValidation(ctx, "tc", security.ValidateTCArgs, classArgs...); err != nil {
		return fmt.Errorf("failed to add class: %v", err)
	}

	// Add netem for latency if specified
	if rule.Latency > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Security: Using SecureExecuteWithValidation with ValidateTCArgs to prevent command injection
		// All formatted values (Priority*10, Latency) are from validated struct fields
		netemArgs := []string{"qdisc", "add", "dev", iface,
			"parent", classID,
			"handle", fmt.Sprintf("%d:", rule.Priority*10),
			"netem", "delay", fmt.Sprintf("%.1fms", rule.Latency)}

		// #nosec G204 - Using security.SecureExecuteWithValidation with argument validation to prevent command injection
		if _, err := security.SecureExecuteWithValidation(ctx, "tc", security.ValidateTCArgs, netemArgs...); err != nil {
			return fmt.Errorf("failed to add netem: %v", err)
		}
	}

	return nil
}

// GetStatistics retrieves TC statistics for an interface
func (s *Shaper) GetStatistics(iface string) (string, error) {
	// Security: Validate interface name to prevent command injection
	if err := security.ValidateNetworkInterface(iface); err != nil {
		return "", fmt.Errorf("invalid interface name: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Security: Using SecureExecuteWithValidation with ValidateTCArgs to prevent command injection
	// Interface name is validated against allowlist patterns
	statsArgs := []string{"-s", "qdisc", "show", "dev", iface}
	// #nosec G204 - Using security.SecureExecuteWithValidation with argument validation to prevent command injection
	output, err := security.SecureExecuteWithValidation(ctx, "tc", security.ValidateTCArgs, statsArgs...)
	if err != nil {
		return "", fmt.Errorf("failed to get statistics: %v", err)
	}
	return string(output), nil
}

// Cleanup removes all TC configurations
func (s *Shaper) Cleanup() error {
	for iface := range s.currentConfigs {
		if err := s.clearInterface(iface); err != nil {
			return err
		}
	}
	s.currentConfigs = make(map[string]*Config)
	return nil
}