// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

package security

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// AllowedCommand represents a command that is allowed to be executed
type AllowedCommand struct {
	Command     string            // Base command (e.g., "iperf3", "tc", "ip")
	AllowedArgs map[string]bool   // Map of allowed arguments
	ArgPatterns []string          // Regex patterns for dynamic arguments
	MaxArgs     int               // Maximum number of arguments
	Timeout     time.Duration     // Maximum execution time
	Description string            // Description of the command purpose
}

// SecureSubprocessExecutor provides secure subprocess execution with validation
type SecureSubprocessExecutor struct {
	allowedCommands map[string]*AllowedCommand
	defaultTimeout  time.Duration
	maxOutputSize   int64
}

// NewSecureSubprocessExecutor creates a new secure subprocess executor
func NewSecureSubprocessExecutor() *SecureSubprocessExecutor {
	executor := &SecureSubprocessExecutor{
		allowedCommands: make(map[string]*AllowedCommand),
		defaultTimeout:  30 * time.Second,
		maxOutputSize:   10 * 1024 * 1024, // 10MB max output
	}

	// Register common safe commands with their allowed arguments
	executor.registerDefaultCommands()
	return executor
}

// registerDefaultCommands registers commonly used safe commands
func (se *SecureSubprocessExecutor) registerDefaultCommands() {
	// iperf3 command allowlist
	se.RegisterCommand(&AllowedCommand{
		Command: "iperf3",
		AllowedArgs: map[string]bool{
			"-s": true, "-c": true, "-p": true, "-t": true, "-u": true,
			"-b": true, "-P": true, "-w": true, "-i": true, "-R": true,
			"--bidir": true, "-J": true, "-D": true, "-V": true, "-h": true,
		},
		ArgPatterns: []string{
			`^\d{1,5}$`,                    // Port numbers (1-65535)
			`^\d+(\.\d+)?[KMG]?$`,         // Bandwidth values (e.g., 10M, 1G)
			`^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$`, // IPv4 addresses
			`^[0-9a-fA-F:]+$`,             // IPv6 addresses
		},
		MaxArgs:     20,
		Timeout:     300 * time.Second, // 5 minutes for network tests
		Description: "iperf3 network performance testing tool",
	})

	// tc (traffic control) command allowlist
	se.RegisterCommand(&AllowedCommand{
		Command: "tc",
		AllowedArgs: map[string]bool{
			"qdisc": true, "add": true, "del": true, "show": true, "replace": true,
			"dev": true, "root": true, "handle": true, "htb": true, "default": true,
			"class": true, "rate": true, "ceil": true, "burst": true, "cburst": true,
			"prio": true, "-s": true, "-d": true,
		},
		ArgPatterns: []string{
			`^[a-zA-Z0-9\-_\.]+$`,         // Interface names
			`^\d+:?$`,                     // Handle IDs (1:, 10:0)
			`^\d+(\.\d+)?[KMG]?bit$`,      // Rate values (10Mbit, 1Gbit)
			`^\d+[KMG]?b$`,                // Burst values (1Kb, 10Mb)
			`^\d+$`,                       // Numeric values
		},
		MaxArgs:     15,
		Timeout:     10 * time.Second,
		Description: "Traffic control utility for bandwidth shaping",
	})

	// ip command allowlist
	se.RegisterCommand(&AllowedCommand{
		Command: "ip",
		AllowedArgs: map[string]bool{
			"link": true, "add": true, "del": true, "set": true, "show": true,
			"type": true, "vxlan": true, "id": true, "dstport": true, "local": true,
			"learning": true, "nolearning": true, "mtu": true, "up": true, "down": true,
			"dev": true, "delete": true, "-s": true, "-d": true,
		},
		ArgPatterns: []string{
			`^[a-zA-Z0-9\-_\.]+$`,         // Interface names and device names
			`^\d{1,5}$`,                   // Port numbers and VNI values
			`^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$`, // IPv4 addresses
			`^\d{1,4}$`,                   // MTU values
		},
		MaxArgs:     15,
		Timeout:     10 * time.Second,
		Description: "IP utility for network interface management",
	})

	// bridge command allowlist (for VXLAN FDB management)
	se.RegisterCommand(&AllowedCommand{
		Command: "bridge",
		AllowedArgs: map[string]bool{
			"fdb": true, "add": true, "del": true, "append": true, "show": true,
			"dev": true, "dst": true, "permanent": true, "temp": true,
		},
		ArgPatterns: []string{
			`^[a-fA-F0-9]{2}:[a-fA-F0-9]{2}:[a-fA-F0-9]{2}:[a-fA-F0-9]{2}:[a-fA-F0-9]{2}:[a-fA-F0-9]{2}$`, // MAC addresses
			`^[a-zA-Z0-9\-_\.]+$`,         // Interface names
			`^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$`, // IPv4 addresses
		},
		MaxArgs:     10,
		Timeout:     5 * time.Second,
		Description: "Bridge utility for FDB management",
	})

	// ping command allowlist
	se.RegisterCommand(&AllowedCommand{
		Command: "ping",
		AllowedArgs: map[string]bool{
			"-c": true, "-i": true, "-W": true, "-w": true, "-s": true,
			"-I": true, "-t": true, "-q": true, "-n": true,
		},
		ArgPatterns: []string{
			`^\d{1,3}$`,                   // Count, interval, timeout values
			`^\d{1,4}$`,                   // Packet size
			`^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$`, // IPv4 addresses
			`^[0-9a-fA-F:]+$`,             // IPv6 addresses
			`^[a-zA-Z0-9\-_\.]+$`,         // Interface names and hostnames
		},
		MaxArgs:     15,
		Timeout:     30 * time.Second,
		Description: "Ping utility for connectivity testing",
	})

	// pkill command allowlist (limited to specific patterns)
	se.RegisterCommand(&AllowedCommand{
		Command: "pkill",
		AllowedArgs: map[string]bool{
			"-f": true, "-x": true, "-u": true, "-g": true,
		},
		ArgPatterns: []string{
			`^iperf3.*-p \d{1,5}$`,        // iperf3 processes with specific port
			`^[a-zA-Z0-9\-_\.\s]+$`,       // Simple process names/patterns
		},
		MaxArgs:     5,
		Timeout:     5 * time.Second,
		Description: "Process killing utility (restricted patterns)",
	})

	// cat command allowlist (for safe file reading)
	se.RegisterCommand(&AllowedCommand{
		Command: "cat",
		AllowedArgs: map[string]bool{},
		ArgPatterns: []string{
			`^/proc/net/dev$`,             // Network device statistics
			`^/proc/[0-9]+/stat$`,         // Process statistics
			`^/sys/class/net/[a-zA-Z0-9\-_\.]+/statistics/.*$`, // Network interface statistics
		},
		MaxArgs:     3,
		Timeout:     5 * time.Second,
		Description: "File reading utility (restricted paths)",
	})

	// pgrep command allowlist (for finding processes)
	se.RegisterCommand(&AllowedCommand{
		Command: "pgrep",
		AllowedArgs: map[string]bool{
			"-f": true, "-x": true, "-l": true, "-u": true, "-g": true,
		},
		ArgPatterns: []string{
			`^iperf3.*-p.*\d{1,5}$`,       // iperf3 processes with port
			`^[a-zA-Z0-9\-_\.\s]+$`,       // Simple process names/patterns
		},
		MaxArgs:     5,
		Timeout:     5 * time.Second,
		Description: "Process finding utility (restricted patterns)",
	})

	// git command allowlist
	se.RegisterCommand(&AllowedCommand{
		Command: "git",
		AllowedArgs: map[string]bool{
			"config": true, "rev-parse": true, "log": true, "status": true, "diff": true,
			"fetch": true, "pull": true, "push": true, "checkout": true, "branch": true,
			"reset": true, "show": true, "remote": true, "tag": true, "rev-list": true,
			"diff-tree": true, "--git-dir": true, "--abbrev-ref": true, "--porcelain": true,
			"--pretty=format:%H|%an|%ae|%ct|%s": true, "--no-commit-id": true, "--name-only": true,
			"-r": true, "-1": true, "--left-right": true, "--count": true, "get-url": true,
			"--sort=-version:refname": true, "-v": true, "-b": true, "--hard": true,
			"--local": true, "credential.helper": true, "store": true, "origin": true,
			"HEAD": true, "HEAD...@{upstream}": true,
		},
		ArgPatterns: []string{
			`^[a-fA-F0-9]{7,40}$`,                    // Git commit hashes
			`^[a-zA-Z0-9\-_\./]+$`,                   // Branch names, tag names, file paths
			`^origin$`,                               // Remote name
			`^HEAD(\.\.\.\@\{upstream\})?$`,          // HEAD references
			`^--pretty=format:%[HanectmsSd\|]+$`,     // Pretty format strings
			`^\d{1,3}$`,                              // Numeric limits
			`^backup-\d+$`,                           // Backup branch names
		},
		MaxArgs:     15,
		Timeout:     60 * time.Second,
		Description: "Git version control system (restricted operations)",
	})

	// kpt command allowlist (for Nephio package management)
	se.RegisterCommand(&AllowedCommand{
		Command: "kpt",
		AllowedArgs: map[string]bool{
			"fn": true, "render": true, "pkg": true, "get": true, "live": true,
			"init": true, "apply": true, "status": true, "destroy": true,
		},
		ArgPatterns: []string{
			`^[a-zA-Z0-9\-_\./]+$`,       // Package paths and names
			`^https?://[a-zA-Z0-9\-\._~/]+$`, // Git URLs
		},
		MaxArgs:     10,
		Timeout:     120 * time.Second,
		Description: "Kubernetes package tool (restricted operations)",
	})

	// cp command allowlist (for file copying in validators)
	se.RegisterCommand(&AllowedCommand{
		Command: "cp",
		AllowedArgs: map[string]bool{
			"-r": true, "-R": true, "-p": true, "-a": true,
		},
		ArgPatterns: []string{
			`^[a-zA-Z0-9\-_\./]+$`,       // File and directory paths
			`^.*\/\.$`,                   // Copy to directory patterns
		},
		MaxArgs:     5,
		Timeout:     30 * time.Second,
		Description: "File copy utility (restricted paths)",
	})
}

// RegisterCommand registers a new allowed command
func (se *SecureSubprocessExecutor) RegisterCommand(cmd *AllowedCommand) error {
	if cmd.Command == "" {
		return fmt.Errorf("command name cannot be empty")
	}

	// Validate command name
	if err := ValidateCommandArgument(cmd.Command); err != nil {
		return fmt.Errorf("invalid command name: %w", err)
	}

	// Set default timeout if not specified
	if cmd.Timeout == 0 {
		cmd.Timeout = se.defaultTimeout
	}

	// Set default max args if not specified
	if cmd.MaxArgs == 0 {
		cmd.MaxArgs = 10
	}

	se.allowedCommands[cmd.Command] = cmd
	return nil
}

// SecureExecute executes a command with security validation
func (se *SecureSubprocessExecutor) SecureExecute(ctx context.Context, command string, args ...string) ([]byte, error) {
	// Validate command is allowed
	allowedCmd, exists := se.allowedCommands[command]
	if !exists {
		return nil, fmt.Errorf("command not in allowlist: %s", command)
	}

	// Validate number of arguments
	if len(args) > allowedCmd.MaxArgs {
		return nil, fmt.Errorf("too many arguments: %d (max: %d)", len(args), allowedCmd.MaxArgs)
	}

	// Validate each argument
	if err := se.validateArguments(allowedCmd, args); err != nil {
		return nil, fmt.Errorf("argument validation failed: %w", err)
	}

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, allowedCmd.Timeout)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(timeoutCtx, command, args...)

	// Set security-oriented environment
	cmd.Env = []string{
		"PATH=/usr/bin:/bin:/usr/sbin:/sbin",
		"LANG=C",
		"LC_ALL=C",
	}

	// Execute command and capture output
	output, err := cmd.CombinedOutput()

	// Check output size limit
	if int64(len(output)) > se.maxOutputSize {
		return nil, fmt.Errorf("command output too large: %d bytes (max: %d)", len(output), se.maxOutputSize)
	}

	// Return output and error
	if err != nil {
		return output, fmt.Errorf("command execution failed: %w", err)
	}

	return output, nil
}

// validateArguments validates command arguments against allowlists and patterns
func (se *SecureSubprocessExecutor) validateArguments(allowedCmd *AllowedCommand, args []string) error {
	for i, arg := range args {
		// Skip empty arguments
		if arg == "" {
			continue
		}

		// Basic security validation
		if err := ValidateCommandArgument(arg); err != nil {
			return fmt.Errorf("argument %d failed basic validation: %w", i, err)
		}

		// Check if argument is in allowed list
		if len(allowedCmd.AllowedArgs) > 0 {
			if _, allowed := allowedCmd.AllowedArgs[arg]; allowed {
				continue // This argument is explicitly allowed
			}
		}

		// Check against patterns for dynamic arguments
		matched := false
		for _, pattern := range allowedCmd.ArgPatterns {
			matched, _ = regexp.MatchString(pattern, arg)
			if matched {
				break
			}
		}

		// If we have allowlists or patterns, argument must match one of them
		if (len(allowedCmd.AllowedArgs) > 0 || len(allowedCmd.ArgPatterns) > 0) && !matched {
			return fmt.Errorf("argument %d not allowed: %s", i, arg)
		}
	}

	return nil
}

// SecureExecuteWithValidation executes a command with additional custom validation
func (se *SecureSubprocessExecutor) SecureExecuteWithValidation(
	ctx context.Context,
	command string,
	customValidator func([]string) error,
	args ...string,
) ([]byte, error) {
	// Run custom validation first
	if customValidator != nil {
		if err := customValidator(args); err != nil {
			return nil, fmt.Errorf("custom validation failed: %w", err)
		}
	}

	// Then run standard secure execution
	return se.SecureExecute(ctx, command, args...)
}

// QuickSecureExecute provides a simplified interface for common commands
func (se *SecureSubprocessExecutor) QuickSecureExecute(command string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return se.SecureExecute(ctx, command, args...)
}

// ValidateIPerfArgs provides specialized validation for iperf3 arguments
func ValidateIPerfArgs(args []string) error {
	var hasServer, hasClient bool
	var serverIP string

	for i, arg := range args {
		switch arg {
		case "-s":
			hasServer = true
		case "-c":
			hasClient = true
			if i+1 < len(args) {
				serverIP = args[i+1]
			}
		case "-p":
			if i+1 < len(args) {
				if err := ValidatePort(parseIntSafe(args[i+1])); err != nil {
					return fmt.Errorf("invalid port: %w", err)
				}
			}
		case "-t":
			if i+1 < len(args) {
				duration := parseIntSafe(args[i+1])
				if duration < 1 || duration > 3600 {
					return fmt.Errorf("invalid duration: %d (must be 1-3600 seconds)", duration)
				}
			}
		case "-b":
			if i+1 < len(args) {
				if err := ValidateBandwidth(args[i+1]); err != nil {
					return fmt.Errorf("invalid bandwidth: %w", err)
				}
			}
		}
	}

	// Must be either server or client, not both
	if hasServer && hasClient {
		return fmt.Errorf("cannot be both server and client")
	}
	if !hasServer && !hasClient {
		return fmt.Errorf("must specify either server (-s) or client (-c)")
	}

	// Validate server IP if client mode
	if hasClient && serverIP != "" {
		if err := ValidateIPAddress(serverIP); err != nil {
			return fmt.Errorf("invalid server IP: %w", err)
		}
	}

	return nil
}

// ValidateTCArgs provides specialized validation for tc arguments
func ValidateTCArgs(args []string) error {
	var hasQdisc, hasDev bool
	var interfaceName string

	for i, arg := range args {
		switch arg {
		case "qdisc":
			hasQdisc = true
		case "dev":
			hasDev = true
			if i+1 < len(args) {
				interfaceName = args[i+1]
			}
		case "rate", "ceil":
			if i+1 < len(args) {
				rate := args[i+1]
				// Simple rate validation (more specific than general bandwidth)
				if !strings.HasSuffix(rate, "bit") {
					return fmt.Errorf("rate must end with 'bit': %s", rate)
				}
			}
		}
	}

	// Basic tc command structure validation
	if hasQdisc && !hasDev {
		return fmt.Errorf("qdisc operations require device specification")
	}

	// Validate interface name if specified
	if hasDev && interfaceName != "" {
		if err := ValidateNetworkInterface(interfaceName); err != nil {
			return fmt.Errorf("invalid interface: %w", err)
		}
	}

	return nil
}

// ValidateIPArgs provides specialized validation for ip command arguments
func ValidateIPArgs(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("ip command requires at least 2 arguments")
	}

	subcommand := args[0]
	switch subcommand {
	case "link":
		return validateIPLinkArgs(args[1:])
	default:
		// Allow other ip subcommands but with basic validation
		return nil
	}
}

// validateIPLinkArgs validates ip link specific arguments
func validateIPLinkArgs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("ip link requires action")
	}

	action := args[0]
	switch action {
	case "add":
		// Validate VXLAN creation arguments
		for i, arg := range args {
			if arg == "type" && i+1 < len(args) {
				if args[i+1] != "vxlan" {
					return fmt.Errorf("only VXLAN type allowed for link add")
				}
			}
			if arg == "id" && i+1 < len(args) {
				vni := parseUint32Safe(args[i+1])
				if err := ValidateVNI(vni); err != nil {
					return fmt.Errorf("invalid VNI: %w", err)
				}
			}
		}
	case "delete", "del", "set", "show":
		// These are generally safe operations
		return nil
	default:
		return fmt.Errorf("ip link action not allowed: %s", action)
	}

	return nil
}

// parseIntSafe safely parses an integer, returning 0 on error
func parseIntSafe(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

// parseUint32Safe safely parses a uint32, returning 0 on error
func parseUint32Safe(s string) uint32 {
	var result uint32
	fmt.Sscanf(s, "%d", &result)
	return result
}

// Global secure executor instance
var DefaultSecureExecutor = NewSecureSubprocessExecutor()

// Convenience functions using the default executor
func SecureExecute(ctx context.Context, command string, args ...string) ([]byte, error) {
	return DefaultSecureExecutor.SecureExecute(ctx, command, args...)
}

func QuickSecureExecute(command string, args ...string) ([]byte, error) {
	return DefaultSecureExecutor.QuickSecureExecute(command, args...)
}

func RegisterSecureCommand(cmd *AllowedCommand) error {
	return DefaultSecureExecutor.RegisterCommand(cmd)
}

func SecureExecuteWithValidation(ctx context.Context, command string, customValidator func([]string) error, args ...string) ([]byte, error) {
	return DefaultSecureExecutor.SecureExecuteWithValidation(ctx, command, customValidator, args...)
}

// ValidateGitArgs provides specialized validation for git arguments
func ValidateGitArgs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("git command requires arguments")
	}

	subcommand := args[0]

	// Validate based on subcommand
	switch subcommand {
	case "config":
		return validateGitConfigArgs(args[1:])
	case "show":
		return validateGitShowArgs(args[1:])
	case "checkout":
		return validateGitCheckoutArgs(args[1:])
	case "reset":
		return validateGitResetArgs(args[1:])
	case "pull", "push":
		return validateGitRemoteArgs(args[1:])
	case "diff", "diff-tree":
		return validateGitDiffArgs(args[1:])
	default:
		// For other commands, use basic validation
		for _, arg := range args {
			if err := ValidateCommandArgument(arg); err != nil {
				return fmt.Errorf("invalid git argument: %w", err)
			}
		}
	}

	return nil
}

// validateGitConfigArgs validates git config arguments
func validateGitConfigArgs(args []string) error {
	allowedConfigs := []string{
		"--local", "--global", "--system",
		"credential.helper", "user.name", "user.email",
		"core.autocrlf", "core.filemode",
	}

	for _, arg := range args {
		if arg == "" {
			continue
		}

		// Check if it's an allowed config option
		allowed := false
		for _, config := range allowedConfigs {
			if arg == config || strings.HasPrefix(arg, config+"=") {
				allowed = true
				break
			}
		}

		if !allowed {
			// Check if it's a safe value
			if err := ValidateCommandArgument(arg); err != nil {
				return fmt.Errorf("unsafe git config argument: %s", arg)
			}
		}
	}

	return nil
}

// validateGitShowArgs validates git show arguments
func validateGitShowArgs(args []string) error {
	for _, arg := range args {
		if strings.Contains(arg, ":") {
			// Format: commit:filename
			parts := strings.SplitN(arg, ":", 2)
			if len(parts) == 2 {
				if err := ValidateGitRef(parts[0]); err != nil {
					return fmt.Errorf("invalid commit in git show: %w", err)
				}
				if err := ValidateFilePath(parts[1]); err != nil {
					return fmt.Errorf("invalid filename in git show: %w", err)
				}
			}
		} else {
			if err := ValidateGitRef(arg); err != nil {
				return fmt.Errorf("invalid git ref in show: %w", err)
			}
		}
	}
	return nil
}

// validateGitCheckoutArgs validates git checkout arguments
func validateGitCheckoutArgs(args []string) error {
	for _, arg := range args {
		if arg == "-b" {
			continue // Flag is allowed
		}
		if err := ValidateGitRef(arg); err != nil {
			return fmt.Errorf("invalid branch/ref in checkout: %w", err)
		}
	}
	return nil
}

// validateGitResetArgs validates git reset arguments
func validateGitResetArgs(args []string) error {
	for _, arg := range args {
		if arg == "--hard" || arg == "--soft" || arg == "--mixed" {
			continue // Flags are allowed
		}
		if err := ValidateGitRef(arg); err != nil {
			return fmt.Errorf("invalid commit in reset: %w", err)
		}
	}
	return nil
}

// validateGitRemoteArgs validates git remote operation arguments
func validateGitRemoteArgs(args []string) error {
	for _, arg := range args {
		if arg == "origin" || arg == "upstream" {
			continue // Common remote names
		}
		if err := ValidateGitRef(arg); err != nil {
			return fmt.Errorf("invalid ref in remote operation: %w", err)
		}
	}
	return nil
}

// validateGitDiffArgs validates git diff arguments
func validateGitDiffArgs(args []string) error {
	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			continue // Flags are generally safe
		}
		if err := ValidateGitRef(arg); err != nil {
			return fmt.Errorf("invalid ref in diff: %w", err)
		}
	}
	return nil
}

// ValidateKptArgs provides specialized validation for kpt arguments
func ValidateKptArgs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("kpt command requires arguments")
	}

	subcommand := args[0]
	switch subcommand {
	case "fn":
		if len(args) > 1 && args[1] == "render" {
			// kpt fn render - validate package path
			for i := 2; i < len(args); i++ {
				if err := ValidateFilePath(args[i]); err != nil {
					return fmt.Errorf("invalid package path: %w", err)
				}
			}
		}
	default:
		// Basic validation for other kpt commands
		for _, arg := range args[1:] {
			if err := ValidateCommandArgument(arg); err != nil {
				return fmt.Errorf("invalid kpt argument: %w", err)
			}
		}
	}

	return nil
}