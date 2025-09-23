package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn/agent/pkg/tc"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn/agent/pkg/vxlan"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn/agent/pkg/watcher"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type Agent struct {
	nodeName     string
	namespace    string
	kubeClient   kubernetes.Interface
	tcShaper     *tc.Shaper
	vxlanManager *vxlan.Manager
	configWatcher *watcher.ConfigWatcher
}

func main() {
	var (
		nodeName  string
		namespace string
		interval  time.Duration
	)

	flag.StringVar(&nodeName, "node-name", os.Getenv("NODE_NAME"), "Node name where agent is running")
	flag.StringVar(&namespace, "namespace", "default", "Namespace to watch for configurations")
	flag.DurationVar(&interval, "interval", 10*time.Second, "Configuration check interval")
	flag.Parse()

	if nodeName == "" {
		klog.Fatal("Node name is required")
	}

	klog.Infof("Starting TN Agent on node %s", nodeName)

	// Create Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatalf("Failed to get in-cluster config: %v", err)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Create agent
	agent := &Agent{
		nodeName:     nodeName,
		namespace:    namespace,
		kubeClient:   kubeClient,
		tcShaper:     tc.NewShaper(),
		vxlanManager: vxlan.NewManager(),
		configWatcher: watcher.NewConfigWatcher(kubeClient, namespace, nodeName),
	}

	// Start agent
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigChan
		klog.Info("Received shutdown signal")
		cancel()
	}()

	// Run agent
	if err := agent.Run(ctx, interval); err != nil {
		klog.Fatalf("Agent failed: %v", err)
	}
}

func (a *Agent) Run(ctx context.Context, interval time.Duration) error {
	klog.Info("Agent starting main loop")

	// Initial cleanup of any stale configurations
	if err := a.cleanup(); err != nil {
		klog.Errorf("Initial cleanup failed: %v", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			klog.Info("Agent shutting down")
			return a.cleanup()

		case <-ticker.C:
			if err := a.reconcile(ctx); err != nil {
				klog.Errorf("Reconciliation failed: %v", err)
			}
		}
	}
}

func (a *Agent) reconcile(ctx context.Context) error {
	klog.V(2).Info("Starting reconciliation")

	// Watch for ConfigMaps with TN slice configurations
	configs, err := a.configWatcher.GetConfigurations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get configurations: %w", err)
	}

	// Apply each configuration
	for _, configData := range configs {
		var agentConfig AgentConfig
		if err := json.Unmarshal([]byte(configData), &agentConfig); err != nil {
			klog.Errorf("Failed to unmarshal config: %v", err)
			continue
		}

		klog.Infof("Applying configuration for slice %s", agentConfig.SliceID)

		// Apply VXLAN tunnels
		for _, tunnel := range agentConfig.Tunnels {
			if err := a.applyVXLANConfig(tunnel); err != nil {
				klog.Errorf("Failed to apply VXLAN config: %v", err)
				continue
			}
		}

		// Apply TC rules
		for _, rule := range agentConfig.TCRules {
			// Replace placeholder interface with actual VXLAN interface
			rule.Interface = fmt.Sprintf("vxlan%d", agentConfig.VxlanID)
			if err := a.applyTCRule(rule); err != nil {
				klog.Errorf("Failed to apply TC rule: %v", err)
				continue
			}
		}
	}

	return nil
}

func (a *Agent) applyVXLANConfig(config VXLANTunnelConfig) error {
	klog.V(1).Infof("Applying VXLAN configuration for interface %s", config.InterfaceName)

	// Execute VXLAN setup commands
	for _, cmd := range config.Commands {
		if err := a.executeCommand(cmd); err != nil {
			return fmt.Errorf("failed to execute VXLAN command '%s': %w", cmd, err)
		}
	}

	return nil
}

func (a *Agent) applyTCRule(rule TCRule) error {
	klog.V(1).Infof("Applying TC rule for interface %s", rule.Interface)

	// Check if interface exists
	if !a.interfaceExists(rule.Interface) {
		return fmt.Errorf("interface %s does not exist", rule.Interface)
	}

	// Clear existing TC configuration
	cleanupCmds := []string{
		fmt.Sprintf("tc qdisc del dev %s root 2>/dev/null || true", rule.Interface),
		fmt.Sprintf("tc qdisc del dev %s ingress 2>/dev/null || true", rule.Interface),
	}

	for _, cmd := range cleanupCmds {
		_ = a.executeCommand(cmd) // Ignore errors for cleanup
	}

	// Apply new TC configuration
	for _, cmd := range rule.TCCommands {
		if err := a.executeCommand(cmd); err != nil {
			return fmt.Errorf("failed to execute TC command '%s': %w", cmd, err)
		}
	}

	// Verify configuration was applied
	output, err := a.executeCommandOutput(fmt.Sprintf("tc qdisc show dev %s", rule.Interface))
	if err != nil {
		return fmt.Errorf("failed to verify TC configuration: %w", err)
	}

	klog.V(2).Infof("TC configuration applied successfully:\n%s", output)
	return nil
}

// executeCommand executes system commands with enhanced security measures to prevent command injection.
// All commands are validated through multiple layers:
// 1. Command string validation to prevent shell metacharacters
// 2. Command allowlisting to only permit safe commands
// 3. Individual argument validation
// 4. Command-specific validation (tc, ip, iperf3)
// 5. Secure subprocess execution with timeouts
func (a *Agent) executeCommand(cmdStr string) error {
	// Enhanced security: First validate the entire command string
	if err := security.ValidateCommandArgument(cmdStr); err != nil {
		return fmt.Errorf("unsafe command string: %w", err)
	}

	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	// Validate command and arguments for security
	allowedCommands := []string{"ip", "tc", "bridge", "ping", "iperf3"}
	commandAllowed := false
	for _, allowed := range allowedCommands {
		if parts[0] == allowed {
			commandAllowed = true
			break
		}
	}
	if !commandAllowed {
		return fmt.Errorf("command not allowed: %s", security.SanitizeForLog(parts[0]))
	}

	// Validate each argument individually
	for i, arg := range parts[1:] {
		if err := security.ValidateCommandArgument(arg); err != nil {
			return fmt.Errorf("invalid argument %d: %w", i+1, err)
		}
	}

	// Use secure execution framework with enhanced validation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Apply command-specific validation
	var customValidator func([]string) error
	switch parts[0] {
	case "tc":
		customValidator = security.ValidateTCArgs
	case "ip":
		customValidator = security.ValidateIPArgs
	case "iperf3":
		customValidator = security.ValidateIPerfArgs
	default:
		customValidator = nil
	}

	var err error
	if customValidator != nil {
		_, err = security.SecureExecuteWithValidation(ctx, parts[0], customValidator, parts[1:]...)
	} else {
		_, err = security.SecureExecute(ctx, parts[0], parts[1:]...)
	}

	if err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	return nil
}

// executeCommandOutput executes system commands and returns output with enhanced security measures.
// This function follows the same security validation patterns as executeCommand but returns command output.
// All subprocess execution is protected against command injection attacks.
func (a *Agent) executeCommandOutput(cmdStr string) (string, error) {
	// Enhanced security: First validate the entire command string
	if err := security.ValidateCommandArgument(cmdStr); err != nil {
		return "", fmt.Errorf("unsafe command string: %w", err)
	}

	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	// Validate command and arguments for security
	allowedCommands := []string{"ip", "tc", "bridge", "ping", "iperf3"}
	commandAllowed := false
	for _, allowed := range allowedCommands {
		if parts[0] == allowed {
			commandAllowed = true
			break
		}
	}
	if !commandAllowed {
		return "", fmt.Errorf("command not allowed: %s", security.SanitizeForLog(parts[0]))
	}

	// Validate each argument individually
	for i, arg := range parts[1:] {
		if err := security.ValidateCommandArgument(arg); err != nil {
			return "", fmt.Errorf("invalid argument %d: %w", i+1, err)
		}
	}

	// Use secure execution framework with enhanced validation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Apply command-specific validation
	var customValidator func([]string) error
	switch parts[0] {
	case "tc":
		customValidator = security.ValidateTCArgs
	case "ip":
		customValidator = security.ValidateIPArgs
	case "iperf3":
		customValidator = security.ValidateIPerfArgs
	default:
		customValidator = nil
	}

	var output []byte
	var err error
	if customValidator != nil {
		output, err = security.SecureExecuteWithValidation(ctx, parts[0], customValidator, parts[1:]...)
	} else {
		output, err = security.SecureExecute(ctx, parts[0], parts[1:]...)
	}

	if err != nil {
		return "", fmt.Errorf("command execution failed: %w", err)
	}

	return string(output), nil
}

func (a *Agent) interfaceExists(name string) bool {
	// Validate interface name for security
	if err := security.ValidateNetworkInterface(name); err != nil {
		return false
	}
	_, err := a.executeCommandOutput(fmt.Sprintf("ip link show %s", name))
	return err == nil
}

func (a *Agent) cleanup() error {
	klog.Info("Cleaning up agent configurations")

	// List all VXLAN interfaces and remove them
	output, err := a.executeCommandOutput("ip -o link show type vxlan")
	if err == nil && output != "" {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				// Extract interface name (format: "2: vxlan0@eth0:")
				ifaceName := strings.TrimSuffix(parts[1], ":")
				ifaceName = strings.Split(ifaceName, "@")[0]

				klog.Infof("Removing VXLAN interface %s", ifaceName)
				_ = a.executeCommand(fmt.Sprintf("ip link del %s", ifaceName))
			}
		}
	}

	return nil
}

// AgentConfig represents the configuration received from the manager
type AgentConfig struct {
	SliceID  string               `json:"sliceId"`
	VxlanID  int32                `json:"vxlanId"`
	TCRules  []TCRule             `json:"tcRules"`
	Tunnels  []VXLANTunnelConfig  `json:"tunnels"`
	Priority int32                `json:"priority"`
}

// TCRule represents a traffic control rule
type TCRule struct {
	Interface   string   `json:"interface"`
	Direction   string   `json:"direction"`
	RateKbit    int      `json:"rateKbit"`
	BurstKB     int      `json:"burstKB"`
	LatencyMs   float32  `json:"latencyMs"`
	JitterMs    float32  `json:"jitterMs,omitempty"`
	LossPercent float32  `json:"lossPercent,omitempty"`
	Priority    int32    `json:"priority"`
	Handle      string   `json:"handle"`
	Parent      string   `json:"parent"`
	Classid     string   `json:"classid"`
	TCCommands  []string `json:"tcCommands"`
}

// VXLANTunnelConfig represents VXLAN tunnel configuration
type VXLANTunnelConfig struct {
	InterfaceName string   `json:"interfaceName"`
	VxlanID      int32    `json:"vxlanId"`
	LocalIP      string   `json:"localIp"`
	RemoteIPs    []string `json:"remoteIps"`
	MTU          int      `json:"mtu"`
	Port         int      `json:"port"`
	Commands     []string `json:"commands"`
}