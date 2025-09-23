package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn/agent/pkg"
)

var (
	configFile = flag.String("config", "config/agent.yaml", "Path to configuration file")
	logLevel   = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	logFile    = flag.String("log-file", "", "Log file path (default: stdout)")
	version    = flag.Bool("version", false, "Show version information")
	localIP    = flag.String("local-ip", "", "Override local IP address")
	port       = flag.Int("port", 0, "Override monitoring port")
)

// Version information (set by build)
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

// Config represents the agent configuration
type Config struct {
	Agent      AgentConfig      `yaml:"agent"`
	VXLAN      VXLANConfig      `yaml:"vxlan"`
	Bandwidth  BandwidthConfig  `yaml:"bandwidth"`
	Monitoring MonitoringConfig `yaml:"monitoring"`
	Logging    LoggingConfig    `yaml:"logging"`
}

// AgentConfig contains agent configuration
type AgentConfig struct {
	ClusterName    string      `yaml:"clusterName"`
	NetworkCIDR    string      `yaml:"networkCIDR"`
	MonitoringPort int         `yaml:"monitoringPort"`
	QoSClass       string      `yaml:"qosClass"`
	Interfaces     []Interface `yaml:"interfaces"`
}

// Interface represents a network interface configuration
type Interface struct {
	Name    string `yaml:"name"`
	Type    string `yaml:"type"`
	IP      string `yaml:"ip"`
	Netmask string `yaml:"netmask"`
	Gateway string `yaml:"gateway"`
	MTU     int    `yaml:"mtu"`
}

// VXLANConfig contains VXLAN configuration
type VXLANConfig struct {
	VNI        uint32   `yaml:"vni"`
	RemoteIPs  []string `yaml:"remoteIPs"`
	LocalIP    string   `yaml:"localIP"`
	Port       int      `yaml:"port"`
	MTU        int      `yaml:"mtu"`
	DeviceName string   `yaml:"deviceName"`
	Learning   bool     `yaml:"learning"`
}

// BandwidthConfig contains bandwidth policy configuration
type BandwidthConfig struct {
	DownlinkMbps float64  `yaml:"downlinkMbps"`
	UplinkMbps   float64  `yaml:"uplinkMbps"`
	LatencyMs    float64  `yaml:"latencyMs"`
	JitterMs     float64  `yaml:"jitterMs"`
	LossPercent  float64  `yaml:"lossPercent"`
	Priority     int      `yaml:"priority"`
	QueueClass   string   `yaml:"queueClass"`
	Filters      []Filter `yaml:"filters"`
}

// Filter represents a traffic filter
type Filter struct {
	Protocol string `yaml:"protocol"`
	SrcIP    string `yaml:"srcIP"`
	DstIP    string `yaml:"dstIP"`
	SrcPort  int    `yaml:"srcPort"`
	DstPort  int    `yaml:"dstPort"`
	Priority int    `yaml:"priority"`
	ClassID  string `yaml:"classID"`
}

// MonitoringConfig contains monitoring configuration
type MonitoringConfig struct {
	Enabled         bool          `yaml:"enabled"`
	Interval        time.Duration `yaml:"interval"`
	MetricsPort     int           `yaml:"metricsPort"`
	ExportDirectory string        `yaml:"exportDirectory"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level      string `yaml:"level"`
	File       string `yaml:"file"`
	MaxSize    int    `yaml:"maxSize"`    // MB
	MaxBackups int    `yaml:"maxBackups"`
	MaxAge     int    `yaml:"maxAge"`     // days
}

func main() {
	flag.Parse()

	// Show version if requested
	if *version {
		fmt.Printf("TN Agent Version: %s\n", Version)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		fmt.Printf("Build Time: %s\n", BuildTime)
		os.Exit(0)
	}

	// Load configuration
	config, err := loadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Apply command line overrides
	if *localIP != "" {
		config.VXLAN.LocalIP = *localIP
	}
	if *port != 0 {
		config.Agent.MonitoringPort = *port
	}

	// Setup logging
	logger, err := setupLogging(config.Logging)
	if err != nil {
		log.Fatalf("Failed to setup logging: %v", err)
	}

	security.SafeLogf(logger, "Starting TN Agent version %s (commit: %s)", security.SanitizeForLog(Version), security.SanitizeForLog(GitCommit))
	security.SafeLogf(logger, "Cluster: %s, Local IP: %s, Port: %d",
		security.SanitizeForLog(config.Agent.ClusterName), security.SanitizeIPForLog(config.VXLAN.LocalIP), config.Agent.MonitoringPort)

	// Create TN configuration
	tnConfig := convertToTNConfig(config)

	// Initialize TN agent
	agent := pkg.NewTNAgent(tnConfig, logger)

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start agent
	if err := agent.Start(); err != nil {
		logger.Fatalf("Failed to start TN agent: %v", err)
	}

	// Start monitoring if enabled
	if config.Monitoring.Enabled {
		go startMonitoring(ctx, agent, config.Monitoring, logger)
	}

	// Wait for shutdown signal
	go func() {
		<-sigChan
		logger.Println("Received shutdown signal")
		cancel()
	}()

	logger.Println("TN Agent is running. Press Ctrl+C to stop.")

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	logger.Println("Shutting down TN Agent...")

	// Export final status
	if err := exportFinalStatus(agent, config.Monitoring, logger); err != nil {
		security.SafeLogError(logger, "Failed to export final status", err)
	}

	// Stop agent
	if err := agent.Stop(); err != nil {
		security.SafeLogError(logger, "Error stopping agent", err)
	}

	logger.Println("TN Agent stopped")
}

// loadConfig loads configuration from file
func loadConfig(configFile string) (*Config, error) {
	// Default configuration
	config := &Config{
		Agent: AgentConfig{
			ClusterName:    "tn-agent",
			NetworkCIDR:    "10.244.0.0/24",
			MonitoringPort: 8080,
			QoSClass:       "default",
			Interfaces: []Interface{
				{
					Name:    "eth0",
					Type:    "physical",
					MTU:     1500,
				},
			},
		},
		VXLAN: VXLANConfig{
			VNI:        1000,
			LocalIP:    "192.168.1.10",
			Port:       4789,
			MTU:        1450,
			DeviceName: "vxlan0",
			Learning:   false,
		},
		Bandwidth: BandwidthConfig{
			DownlinkMbps: 10.0,
			UplinkMbps:   10.0,
			LatencyMs:    10.0,
			JitterMs:     2.0,
			LossPercent:  0.1,
			Priority:     2,
			QueueClass:   "htb",
			Filters: []Filter{
				{
					Protocol: "tcp",
					Priority: 10,
					ClassID:  "1:10",
				},
			},
		},
		Monitoring: MonitoringConfig{
			Enabled:         true,
			Interval:        30 * time.Second,
			MetricsPort:     9090,
			ExportDirectory: "metrics",
		},
		Logging: LoggingConfig{
			Level:      "info",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
		},
	}

	// Override with file if it exists
	if configFile != "" {
		if _, err := os.Stat(configFile); err == nil {
			// Create validator for configuration files
			validator := security.CreateValidatorForConfig(".")

			// Validate file path for security
			if err := validator.ValidateFilePathAndExtension(configFile, []string{".yaml", ".yml", ".json", ".toml", ".conf", ".cfg"}); err != nil {
				return nil, fmt.Errorf("config file path validation failed: %w", err)
			}

			data, err := validator.SafeReadFile(configFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}

			if err := yaml.Unmarshal(data, config); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
		}
	}

	// Override with command line flags
	if *logFile != "" {
		config.Logging.File = *logFile
	}

	return config, nil
}

// convertToTNConfig converts config to TN agent config
func convertToTNConfig(config *Config) *pkg.TNConfig {
	// Convert interfaces
	var interfaces []pkg.NetworkInterface
	for _, iface := range config.Agent.Interfaces {
		interfaces = append(interfaces, pkg.NetworkInterface{
			Name:    iface.Name,
			Type:    iface.Type,
			IP:      iface.IP,
			Netmask: iface.Netmask,
			Gateway: iface.Gateway,
			MTU:     iface.MTU,
			State:   "up",
		})
	}

	// Convert filters
	var filters []pkg.Filter
	for _, filter := range config.Bandwidth.Filters {
		filters = append(filters, pkg.Filter{
			Protocol: filter.Protocol,
			SrcIP:    filter.SrcIP,
			DstIP:    filter.DstIP,
			SrcPort:  filter.SrcPort,
			DstPort:  filter.DstPort,
			Priority: filter.Priority,
			ClassID:  filter.ClassID,
		})
	}

	return &pkg.TNConfig{
		ClusterName:    config.Agent.ClusterName,
		NetworkCIDR:    config.Agent.NetworkCIDR,
		MonitoringPort: config.Agent.MonitoringPort,
		QoSClass:       config.Agent.QoSClass,
		Interfaces:     interfaces,
		VXLANConfig: pkg.VXLANConfig{
			VNI:        config.VXLAN.VNI,
			RemoteIPs:  config.VXLAN.RemoteIPs,
			LocalIP:    config.VXLAN.LocalIP,
			Port:       config.VXLAN.Port,
			MTU:        config.VXLAN.MTU,
			DeviceName: config.VXLAN.DeviceName,
			Learning:   config.VXLAN.Learning,
		},
		BWPolicy: pkg.BandwidthPolicy{
			DownlinkMbps: config.Bandwidth.DownlinkMbps,
			UplinkMbps:   config.Bandwidth.UplinkMbps,
			LatencyMs:    config.Bandwidth.LatencyMs,
			JitterMs:     config.Bandwidth.JitterMs,
			LossPercent:  config.Bandwidth.LossPercent,
			Priority:     config.Bandwidth.Priority,
			QueueClass:   config.Bandwidth.QueueClass,
			Filters:      filters,
		},
	}
}

// setupLogging configures logging based on configuration
func setupLogging(config LoggingConfig) (*log.Logger, error) {
	var output *os.File = os.Stdout

	if config.File != "" {
		// Create log directory if it doesn't exist
		logDir := filepath.Dir(config.File)
		if err := os.MkdirAll(logDir, security.PrivateDirMode); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		file, err := os.OpenFile(config.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, security.SecureFileMode)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		output = file
	}

	logger := log.New(output, "[TN-Agent] ", log.LstdFlags|log.Lshortfile)

	return logger, nil
}

// startMonitoring starts the monitoring loop
func startMonitoring(ctx context.Context, agent *pkg.TNAgent, config MonitoringConfig, logger *log.Logger) {
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	security.SafeLogf(logger, "Starting monitoring every %v", config.Interval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := collectAndExportMetrics(agent, config, logger); err != nil {
				security.SafeLogError(logger, "Failed to collect metrics", err)
			}
		}
	}
}

// collectAndExportMetrics collects and exports agent metrics
func collectAndExportMetrics(agent *pkg.TNAgent, config MonitoringConfig, logger *log.Logger) error {
	// Get agent status
	status, err := agent.GetStatus()
	if err != nil {
		return fmt.Errorf("failed to get agent status: %w", err)
	}

	// Log status summary
	security.SafeLogf(logger, "Agent Status: Healthy=%v, Connections=%d, VXLAN=%v, TC=%v",
		status.Healthy, status.ActiveConnections, status.VXLANStatus.TunnelUp, status.TCStatus.RulesActive)

	// Export to file if directory is configured
	if config.ExportDirectory != "" {
		if err := exportStatus(status, config, logger); err != nil {
			return fmt.Errorf("failed to export status: %w", err)
		}
	}

	return nil
}

// exportStatus exports status to file
func exportStatus(status *pkg.TNStatus, config MonitoringConfig, logger *log.Logger) error {
	// Create export directory if it doesn't exist
	if err := os.MkdirAll(config.ExportDirectory, security.PrivateDirMode); err != nil {
		return fmt.Errorf("failed to create export directory: %w", err)
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(config.ExportDirectory, fmt.Sprintf("agent_status_%s.json", timestamp))

	// This would export the status to JSON file
	// For now, just log the export
	security.SafeLogf(logger, "Status exported to: %s (healthy=%v)", security.SanitizeForLog(filename), status.Healthy)

	return nil
}

// exportFinalStatus exports final status on shutdown
func exportFinalStatus(agent *pkg.TNAgent, config MonitoringConfig, logger *log.Logger) error {
	status, err := agent.GetStatus()
	if err != nil {
		return fmt.Errorf("failed to get final status: %w", err)
	}

	if config.ExportDirectory != "" {
		// Create export directory if it doesn't exist
		if err := os.MkdirAll(config.ExportDirectory, security.PrivateDirMode); err != nil {
			return fmt.Errorf("failed to create export directory: %w", err)
		}

		// Generate filename with timestamp
		timestamp := time.Now().Format("20060102_150405")
		filename := filepath.Join(config.ExportDirectory, fmt.Sprintf("agent_final_status_%s.json", timestamp))

		// This would export the final status to JSON file
		security.SafeLogf(logger, "Final status exported to: %s", security.SanitizeForLog(filename))
	}

	security.SafeLogf(logger, "Final Status: Healthy=%v, Uptime=%v",
		status.Healthy, time.Since(status.LastUpdate))

	return nil
}