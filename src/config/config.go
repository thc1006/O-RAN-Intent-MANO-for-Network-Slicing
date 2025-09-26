package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration values for the O-RAN MANO system
type Config struct {
	// General configuration
	Environment     string `json:"environment"`
	LogLevel        string `json:"log_level"`
	ServiceName     string `json:"service_name"`
	ServiceVersion  string `json:"service_version"`

	// Server configuration
	HTTPPort        int           `json:"http_port"`
	GRPCPort        int           `json:"grpc_port"`
	ReadTimeout     time.Duration `json:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout"`
	IdleTimeout     time.Duration `json:"idle_timeout"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`

	// Database configuration
	DatabaseURL      string        `json:"database_url"`
	DatabaseDriver   string        `json:"database_driver"`
	DatabaseName     string        `json:"database_name"`
	DatabaseMaxConns int           `json:"database_max_conns"`
	DatabaseMaxIdle  int           `json:"database_max_idle"`
	DatabaseTimeout  time.Duration `json:"database_timeout"`

	// O2 DMS configuration
	DMSEndpoint     string        `json:"dms_endpoint"`
	DMSToken        string        `json:"dms_token"`
	DMSTimeout      time.Duration `json:"dms_timeout"`
	DMSMaxRetries   int           `json:"dms_max_retries"`
	DMSRetryDelay   time.Duration `json:"dms_retry_delay"`

	// GitOps configuration
	GitOpsRepoURL   string `json:"gitops_repo_url"`
	GitOpsUsername  string `json:"gitops_username"`
	GitOpsToken     string `json:"gitops_token"`
	GitOpsBranch    string `json:"gitops_branch"`
	GitOpsPath      string `json:"gitops_path"`

	// Nephio configuration
	NephioRegistry     string `json:"nephio_registry"`
	NephioWorkingDir   string `json:"nephio_working_dir"`
	NephioPackageTTL   time.Duration `json:"nephio_package_ttl"`

	// Placement optimizer configuration
	PlacementAlgorithm      string        `json:"placement_algorithm"`
	PlacementCacheEnabled   bool          `json:"placement_cache_enabled"`
	PlacementCacheTTL       time.Duration `json:"placement_cache_ttl"`
	PlacementMaxIterations  int           `json:"placement_max_iterations"`
	PlacementTimeout        time.Duration `json:"placement_timeout"`
	PlacementParallelWorkers int          `json:"placement_parallel_workers"`

	// VNF Controller configuration
	VNFControllerMaxRetries      int           `json:"vnf_controller_max_retries"`
	VNFControllerInitialBackoff  time.Duration `json:"vnf_controller_initial_backoff"`
	VNFControllerMaxBackoff      time.Duration `json:"vnf_controller_max_backoff"`
	VNFControllerBackoffMultiplier float64     `json:"vnf_controller_backoff_multiplier"`

	// Health check configuration
	HealthCheckInterval     time.Duration `json:"health_check_interval"`
	HealthCheckTimeout      time.Duration `json:"health_check_timeout"`
	HealthCheckRetries      int           `json:"health_check_retries"`

	// Metrics configuration
	MetricsEnabled          bool          `json:"metrics_enabled"`
	MetricsPort            int           `json:"metrics_port"`
	MetricsPath            string        `json:"metrics_path"`
	MetricsCollectionInterval time.Duration `json:"metrics_collection_interval"`

	// Security configuration
	TLSEnabled          bool   `json:"tls_enabled"`
	TLSCertFile         string `json:"tls_cert_file"`
	TLSKeyFile          string `json:"tls_key_file"`
	TLSCAFile           string `json:"tls_ca_file"`
	AuthEnabled         bool   `json:"auth_enabled"`
	AuthTokenSecret     string `json:"auth_token_secret"`
	AuthTokenExpiry     time.Duration `json:"auth_token_expiry"`

	// Kubernetes configuration
	KubeConfig          string `json:"kube_config"`
	KubeNamespace       string `json:"kube_namespace"`
	KubeTimeout         time.Duration `json:"kube_timeout"`
	KubeRetries         int    `json:"kube_retries"`

	// Intent parser configuration
	IntentParserConfidenceThreshold float64 `json:"intent_parser_confidence_threshold"`
	IntentParserMaxPatterns         int     `json:"intent_parser_max_patterns"`
	IntentParserCacheEnabled        bool    `json:"intent_parser_cache_enabled"`

	// Resource limits
	MaxConcurrentDeployments    int `json:"max_concurrent_deployments"`
	MaxResourceAllocationMB     int `json:"max_resource_allocation_mb"`
	MaxNetworkBandwidthMbps     int `json:"max_network_bandwidth_mbps"`

	// Feature flags
	FeatureAdvancedPlacement    bool `json:"feature_advanced_placement"`
	FeatureAutoRemediation      bool `json:"feature_auto_remediation"`
	FeatureMultiClusterSupport  bool `json:"feature_multi_cluster_support"`
	FeatureNeuralOptimization   bool `json:"feature_neural_optimization"`
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() (*Config, error) {
	config := &Config{}

	// General configuration
	config.Environment = getEnvString("ENVIRONMENT", "development")
	config.LogLevel = getEnvString("LOG_LEVEL", "info")
	config.ServiceName = getEnvString("SERVICE_NAME", "oran-mano")
	config.ServiceVersion = getEnvString("SERVICE_VERSION", "1.0.0")

	// Server configuration
	config.HTTPPort = getEnvInt("HTTP_PORT", 8080)
	config.GRPCPort = getEnvInt("GRPC_PORT", 9090)
	config.ReadTimeout = getEnvDuration("READ_TIMEOUT", 30*time.Second)
	config.WriteTimeout = getEnvDuration("WRITE_TIMEOUT", 30*time.Second)
	config.IdleTimeout = getEnvDuration("IDLE_TIMEOUT", 120*time.Second)
	config.ShutdownTimeout = getEnvDuration("SHUTDOWN_TIMEOUT", 30*time.Second)

	// Database configuration
	config.DatabaseURL = getEnvString("DATABASE_URL", "postgresql://localhost:5432/oran_mano")
	config.DatabaseDriver = getEnvString("DATABASE_DRIVER", "postgres")
	config.DatabaseName = getEnvString("DATABASE_NAME", "oran_mano")
	config.DatabaseMaxConns = getEnvInt("DATABASE_MAX_CONNS", 25)
	config.DatabaseMaxIdle = getEnvInt("DATABASE_MAX_IDLE", 5)
	config.DatabaseTimeout = getEnvDuration("DATABASE_TIMEOUT", 30*time.Second)

	// O2 DMS configuration
	config.DMSEndpoint = getEnvString("DMS_ENDPOINT", "http://localhost:8081")
	config.DMSToken = getEnvString("DMS_TOKEN", "")
	config.DMSTimeout = getEnvDuration("DMS_TIMEOUT", 30*time.Second)
	config.DMSMaxRetries = getEnvInt("DMS_MAX_RETRIES", 3)
	config.DMSRetryDelay = getEnvDuration("DMS_RETRY_DELAY", 1*time.Second)

	// GitOps configuration
	config.GitOpsRepoURL = getEnvString("GITOPS_REPO_URL", "")
	config.GitOpsUsername = getEnvString("GITOPS_USERNAME", "")
	config.GitOpsToken = getEnvString("GITOPS_TOKEN", "")
	config.GitOpsBranch = getEnvString("GITOPS_BRANCH", "main")
	config.GitOpsPath = getEnvString("GITOPS_PATH", "manifests")

	// Nephio configuration
	config.NephioRegistry = getEnvString("NEPHIO_REGISTRY", "nephio.io")
	config.NephioWorkingDir = getEnvString("NEPHIO_WORKING_DIR", "/tmp/nephio-packages")
	config.NephioPackageTTL = getEnvDuration("NEPHIO_PACKAGE_TTL", 24*time.Hour)

	// Placement optimizer configuration
	config.PlacementAlgorithm = getEnvString("PLACEMENT_ALGORITHM", "weighted_score")
	config.PlacementCacheEnabled = getEnvBool("PLACEMENT_CACHE_ENABLED", true)
	config.PlacementCacheTTL = getEnvDuration("PLACEMENT_CACHE_TTL", 5*time.Minute)
	config.PlacementMaxIterations = getEnvInt("PLACEMENT_MAX_ITERATIONS", 1000)
	config.PlacementTimeout = getEnvDuration("PLACEMENT_TIMEOUT", 30*time.Second)
	config.PlacementParallelWorkers = getEnvInt("PLACEMENT_PARALLEL_WORKERS", 4)

	// VNF Controller configuration
	config.VNFControllerMaxRetries = getEnvInt("VNF_CONTROLLER_MAX_RETRIES", 5)
	config.VNFControllerInitialBackoff = getEnvDuration("VNF_CONTROLLER_INITIAL_BACKOFF", 30*time.Second)
	config.VNFControllerMaxBackoff = getEnvDuration("VNF_CONTROLLER_MAX_BACKOFF", 10*time.Minute)
	config.VNFControllerBackoffMultiplier = getEnvFloat("VNF_CONTROLLER_BACKOFF_MULTIPLIER", 2.0)

	// Health check configuration
	config.HealthCheckInterval = getEnvDuration("HEALTH_CHECK_INTERVAL", 30*time.Second)
	config.HealthCheckTimeout = getEnvDuration("HEALTH_CHECK_TIMEOUT", 5*time.Second)
	config.HealthCheckRetries = getEnvInt("HEALTH_CHECK_RETRIES", 3)

	// Metrics configuration
	config.MetricsEnabled = getEnvBool("METRICS_ENABLED", true)
	config.MetricsPort = getEnvInt("METRICS_PORT", 8081)
	config.MetricsPath = getEnvString("METRICS_PATH", "/metrics")
	config.MetricsCollectionInterval = getEnvDuration("METRICS_COLLECTION_INTERVAL", 15*time.Second)

	// Security configuration
	config.TLSEnabled = getEnvBool("TLS_ENABLED", false)
	config.TLSCertFile = getEnvString("TLS_CERT_FILE", "")
	config.TLSKeyFile = getEnvString("TLS_KEY_FILE", "")
	config.TLSCAFile = getEnvString("TLS_CA_FILE", "")
	config.AuthEnabled = getEnvBool("AUTH_ENABLED", false)
	config.AuthTokenSecret = getEnvString("AUTH_TOKEN_SECRET", "")
	config.AuthTokenExpiry = getEnvDuration("AUTH_TOKEN_EXPIRY", 24*time.Hour)

	// Kubernetes configuration
	config.KubeConfig = getEnvString("KUBE_CONFIG", "")
	config.KubeNamespace = getEnvString("KUBE_NAMESPACE", "default")
	config.KubeTimeout = getEnvDuration("KUBE_TIMEOUT", 30*time.Second)
	config.KubeRetries = getEnvInt("KUBE_RETRIES", 3)

	// Intent parser configuration
	config.IntentParserConfidenceThreshold = getEnvFloat("INTENT_PARSER_CONFIDENCE_THRESHOLD", 0.7)
	config.IntentParserMaxPatterns = getEnvInt("INTENT_PARSER_MAX_PATTERNS", 100)
	config.IntentParserCacheEnabled = getEnvBool("INTENT_PARSER_CACHE_ENABLED", true)

	// Resource limits
	config.MaxConcurrentDeployments = getEnvInt("MAX_CONCURRENT_DEPLOYMENTS", 50)
	config.MaxResourceAllocationMB = getEnvInt("MAX_RESOURCE_ALLOCATION_MB", 10240)
	config.MaxNetworkBandwidthMbps = getEnvInt("MAX_NETWORK_BANDWIDTH_MBPS", 10000)

	// Feature flags
	config.FeatureAdvancedPlacement = getEnvBool("FEATURE_ADVANCED_PLACEMENT", true)
	config.FeatureAutoRemediation = getEnvBool("FEATURE_AUTO_REMEDIATION", true)
	config.FeatureMultiClusterSupport = getEnvBool("FEATURE_MULTI_CLUSTER_SUPPORT", true)
	config.FeatureNeuralOptimization = getEnvBool("FEATURE_NEURAL_OPTIMIZATION", false)

	// Validate required configuration
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// validate checks that required configuration values are set
func (c *Config) validate() error {
	var errors []string

	// Validate required fields
	if c.ServiceName == "" {
		errors = append(errors, "SERVICE_NAME is required")
	}

	if c.HTTPPort <= 0 || c.HTTPPort > 65535 {
		errors = append(errors, "HTTP_PORT must be between 1 and 65535")
	}

	if c.GRPCPort <= 0 || c.GRPCPort > 65535 {
		errors = append(errors, "GRPC_PORT must be between 1 and 65535")
	}

	if c.DatabaseMaxConns <= 0 {
		errors = append(errors, "DATABASE_MAX_CONNS must be positive")
	}

	if c.DMSMaxRetries < 0 {
		errors = append(errors, "DMS_MAX_RETRIES must be non-negative")
	}

	if c.PlacementMaxIterations <= 0 {
		errors = append(errors, "PLACEMENT_MAX_ITERATIONS must be positive")
	}

	if c.VNFControllerMaxRetries < 0 {
		errors = append(errors, "VNF_CONTROLLER_MAX_RETRIES must be non-negative")
	}

	if c.VNFControllerBackoffMultiplier <= 1.0 {
		errors = append(errors, "VNF_CONTROLLER_BACKOFF_MULTIPLIER must be greater than 1.0")
	}

	if c.IntentParserConfidenceThreshold < 0 || c.IntentParserConfidenceThreshold > 1 {
		errors = append(errors, "INTENT_PARSER_CONFIDENCE_THRESHOLD must be between 0 and 1")
	}

	if c.MaxConcurrentDeployments <= 0 {
		errors = append(errors, "MAX_CONCURRENT_DEPLOYMENTS must be positive")
	}

	// Validate TLS configuration if enabled
	if c.TLSEnabled {
		if c.TLSCertFile == "" {
			errors = append(errors, "TLS_CERT_FILE is required when TLS is enabled")
		}
		if c.TLSKeyFile == "" {
			errors = append(errors, "TLS_KEY_FILE is required when TLS is enabled")
		}
	}

	// Validate auth configuration if enabled
	if c.AuthEnabled {
		if c.AuthTokenSecret == "" {
			errors = append(errors, "AUTH_TOKEN_SECRET is required when auth is enabled")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation errors: %s", strings.Join(errors, ", "))
	}

	return nil
}

// GetLogLevel returns the slog.Level for the configured log level
func (c *Config) GetLogLevel() slog.Level {
	switch strings.ToLower(c.LogLevel) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// IsDevelopment returns true if the environment is development
func (c *Config) IsDevelopment() bool {
	return strings.ToLower(c.Environment) == "development"
}

// IsProduction returns true if the environment is production
func (c *Config) IsProduction() bool {
	return strings.ToLower(c.Environment) == "production"
}

// PrintConfig logs the configuration (excluding sensitive fields)
func (c *Config) PrintConfig(logger *slog.Logger) {
	logger.Info("Configuration loaded",
		"environment", c.Environment,
		"service_name", c.ServiceName,
		"service_version", c.ServiceVersion,
		"http_port", c.HTTPPort,
		"grpc_port", c.GRPCPort,
		"log_level", c.LogLevel,
		"database_driver", c.DatabaseDriver,
		"database_name", c.DatabaseName,
		"dms_endpoint", c.DMSEndpoint,
		"placement_algorithm", c.PlacementAlgorithm,
		"placement_cache_enabled", c.PlacementCacheEnabled,
		"metrics_enabled", c.MetricsEnabled,
		"tls_enabled", c.TLSEnabled,
		"auth_enabled", c.AuthEnabled,
		"feature_advanced_placement", c.FeatureAdvancedPlacement,
		"feature_auto_remediation", c.FeatureAutoRemediation,
		"feature_multi_cluster_support", c.FeatureMultiClusterSupport,
		"feature_neural_optimization", c.FeatureNeuralOptimization,
	)
}

// Environment variable helper functions
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}