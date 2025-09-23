package tc

import (
	"fmt"
	"os/exec"
	"strings"
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
func (s *Shaper) ApplyRules(iface string, rules []Rule) error {
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
	cmd := exec.Command("tc", "qdisc", "del", "dev", iface, "root")
	if output, err := cmd.CombinedOutput(); err != nil {
		// Ignore errors if no qdisc exists
		if !strings.Contains(string(output), "RTNETLINK answers: No such file or directory") {
			return fmt.Errorf("failed to clear root qdisc: %v, output: %s", err, output)
		}
	}

	// Clear ingress qdisc
	cmd = exec.Command("tc", "qdisc", "del", "dev", iface, "ingress")
	if output, err := cmd.CombinedOutput(); err != nil {
		// Ignore errors if no qdisc exists
		if !strings.Contains(string(output), "RTNETLINK answers: Invalid argument") &&
			!strings.Contains(string(output), "RTNETLINK answers: No such file or directory") {
			return fmt.Errorf("failed to clear ingress qdisc: %v, output: %s", err, output)
		}
	}

	return nil
}

func (s *Shaper) applyRule(iface string, rule Rule) error {
	// This is a simplified implementation
	// In production, you would use netlink library for more control

	// Add root HTB qdisc if not exists
	cmd := exec.Command("tc", "qdisc", "add", "dev", iface, "root", "handle", "1:", "htb", "default", "30")
	if output, err := cmd.CombinedOutput(); err != nil {
		if !strings.Contains(string(output), "File exists") {
			return fmt.Errorf("failed to add root qdisc: %v, output: %s", err, output)
		}
	}

	// Add class with rate limit
	classID := fmt.Sprintf("1:%d", rule.Priority)
	cmd = exec.Command("tc", "class", "add", "dev", iface, "parent", "1:",
		"classid", classID, "htb",
		"rate", fmt.Sprintf("%dkbit", rule.Rate),
		"burst", fmt.Sprintf("%dk", rule.Burst))

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add class: %v, output: %s", err, output)
	}

	// Add netem for latency if specified
	if rule.Latency > 0 {
		cmd = exec.Command("tc", "qdisc", "add", "dev", iface,
			"parent", classID,
			"handle", fmt.Sprintf("%d:", rule.Priority*10),
			"netem", "delay", fmt.Sprintf("%.1fms", rule.Latency))

		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to add netem: %v, output: %s", err, output)
		}
	}

	return nil
}

// GetStatistics retrieves TC statistics for an interface
func (s *Shaper) GetStatistics(iface string) (string, error) {
	cmd := exec.Command("tc", "-s", "qdisc", "show", "dev", iface)
	output, err := cmd.CombinedOutput()
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