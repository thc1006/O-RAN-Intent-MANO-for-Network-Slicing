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

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn/manager/pkg"
)

var (
	configFile = flag.String("config", "config/manager.yaml", "Path to configuration file")
	logLevel   = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	logFile    = flag.String("log-file", "", "Log file path (default: stdout)")
	version    = flag.Bool("version", false, "Show version information")
)

// Version information (set by build)
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

// Config represents the manager configuration
type Config struct {
	Manager    ManagerConfig    `yaml:"manager"`
	Agents     []AgentConfig    `yaml:"agents"`
	Monitoring MonitoringConfig `yaml:"monitoring"`
	Logging    LoggingConfig    `yaml:"logging"`
}

// ManagerConfig contains TN manager configuration
type ManagerConfig struct {
	ClusterName    string `yaml:"clusterName"`
	NetworkCIDR    string `yaml:"networkCIDR"`
	MonitoringPort int    `yaml:"monitoringPort"`
}

// AgentConfig contains agent registration configuration
type AgentConfig struct {
	Name     string `yaml:"name"`
	Endpoint string `yaml:"endpoint"`
	Enabled  bool   `yaml:"enabled"`
}

// MonitoringConfig contains monitoring configuration
type MonitoringConfig struct {
	MetricsInterval  time.Duration `yaml:"metricsInterval"`
	RetentionPeriod  time.Duration `yaml:"retentionPeriod"`
	MaxSamples       int           `yaml:"maxSamples"`
	ExportDirectory  string        `yaml:"exportDirectory"`
	EnableContinuous bool          `yaml:"enableContinuous"`
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
		fmt.Printf("TN Manager Version: %s\n", Version)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		fmt.Printf("Build Time: %s\n", BuildTime)
		os.Exit(0)
	}

	// Load configuration
	config, err := loadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logging
	logger, err := setupLogging(config.Logging)
	if err != nil {
		log.Fatalf("Failed to setup logging: %v", err)
	}

	logger.Printf("Starting TN Manager version %s (commit: %s)", Version, GitCommit)

	// Create TN configuration
	tnConfig := &pkg.TNConfig{
		ClusterName:    config.Manager.ClusterName,
		NetworkCIDR:    config.Manager.NetworkCIDR,
		MonitoringPort: config.Manager.MonitoringPort,
	}

	// Initialize TN manager
	manager := pkg.NewTNManager(tnConfig, logger)

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start manager
	if err := manager.Start(); err != nil {
		logger.Fatalf("Failed to start TN manager: %v", err)
	}

	// Register agents
	for _, agentConfig := range config.Agents {
		if !agentConfig.Enabled {
			logger.Printf("Skipping disabled agent: %s", agentConfig.Name)
			continue
		}

		logger.Printf("Registering agent: %s at %s", agentConfig.Name, agentConfig.Endpoint)
		if err := manager.RegisterAgent(agentConfig.Name, agentConfig.Endpoint); err != nil {
			logger.Printf("Warning: Failed to register agent %s: %v", agentConfig.Name, err)
		}
	}

	// Start metrics export if configured
	if config.Monitoring.EnableContinuous {
		go startContinuousMetricsExport(ctx, manager, config.Monitoring, logger)
	}

	// Wait for shutdown signal
	go func() {
		<-sigChan
		logger.Println("Received shutdown signal")
		cancel()
	}()

	logger.Println("TN Manager is running. Press Ctrl+C to stop.")

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	logger.Println("Shutting down TN Manager...")

	// Export final metrics
	if err := exportFinalMetrics(manager, config.Monitoring, logger); err != nil {
		logger.Printf("Failed to export final metrics: %v", err)
	}

	// Stop manager
	if err := manager.Stop(); err != nil {
		logger.Printf("Error stopping manager: %v", err)
	}

	logger.Println("TN Manager stopped")
}

// loadConfig loads configuration from file
func loadConfig(configFile string) (*Config, error) {
	// Default configuration
	config := &Config{
		Manager: ManagerConfig{
			ClusterName:    "tn-manager",
			NetworkCIDR:    "10.244.0.0/16",
			MonitoringPort: 9090,
		},
		Monitoring: MonitoringConfig{
			MetricsInterval:  30 * time.Second,
			RetentionPeriod:  24 * time.Hour,
			MaxSamples:       1000,
			ExportDirectory:  "reports",
			EnableContinuous: false,
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
			data, err := os.ReadFile(configFile)
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

// setupLogging configures logging based on configuration
func setupLogging(config LoggingConfig) (*log.Logger, error) {
	var output *os.File = os.Stdout

	if config.File != "" {
		// Create log directory if it doesn't exist
		logDir := filepath.Dir(config.File)
		if err := os.MkdirAll(logDir, 0750); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		file, err := os.OpenFile(config.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		output = file
	}

	logger := log.New(output, "[TN-Manager] ", log.LstdFlags|log.Lshortfile)

	return logger, nil
}

// startContinuousMetricsExport starts continuous metrics export
func startContinuousMetricsExport(ctx context.Context, manager *pkg.TNManager, config MonitoringConfig, logger *log.Logger) {
	ticker := time.NewTicker(config.MetricsInterval)
	defer ticker.Stop()

	logger.Printf("Starting continuous metrics export every %v", config.MetricsInterval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := exportMetrics(manager, config, logger); err != nil {
				logger.Printf("Failed to export metrics: %v", err)
			}
		}
	}
}

// exportMetrics exports metrics to file
func exportMetrics(manager *pkg.TNManager, config MonitoringConfig, logger *log.Logger) error {
	// Create export directory if it doesn't exist
	if err := os.MkdirAll(config.ExportDirectory, 0750); err != nil {
		return fmt.Errorf("failed to create export directory: %w", err)
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(config.ExportDirectory, fmt.Sprintf("tn_metrics_%s.json", timestamp))

	// Export metrics
	if err := manager.ExportMetrics(filename); err != nil {
		return fmt.Errorf("failed to export metrics: %w", err)
	}

	logger.Printf("Metrics exported to: %s", filename)
	return nil
}

// exportFinalMetrics exports final metrics on shutdown
func exportFinalMetrics(manager *pkg.TNManager, config MonitoringConfig, logger *log.Logger) error {
	// Create export directory if it doesn't exist
	if err := os.MkdirAll(config.ExportDirectory, 0750); err != nil {
		return fmt.Errorf("failed to create export directory: %w", err)
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(config.ExportDirectory, fmt.Sprintf("tn_final_metrics_%s.json", timestamp))

	// Export final metrics
	if err := manager.ExportMetrics(filename); err != nil {
		return fmt.Errorf("failed to export final metrics: %w", err)
	}

	logger.Printf("Final metrics exported to: %s", filename)
	return nil
}