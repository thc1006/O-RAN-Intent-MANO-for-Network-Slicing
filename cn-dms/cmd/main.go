package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
)

// Version information
var (
	version = "v1.0.0"
	build   = "unknown"
)

// Config holds the application configuration
type Config struct {
	Server struct {
		Port       int    `mapstructure:"port"`
		TLSPort    int    `mapstructure:"tls_port"`
		TLSEnabled bool   `mapstructure:"tls_enabled"`
		CertFile   string `mapstructure:"cert_file"`
		KeyFile    string `mapstructure:"key_file"`
	} `mapstructure:"server"`

	Logging struct {
		Level  string `mapstructure:"level"`
		Format string `mapstructure:"format"`
	} `mapstructure:"logging"`

	Metrics struct {
		Enabled bool `mapstructure:"enabled"`
		Port    int  `mapstructure:"port"`
	} `mapstructure:"metrics"`

	CN struct {
		UpdateInterval time.Duration `mapstructure:"update_interval"`
		MaxSlices      int          `mapstructure:"max_slices"`
	} `mapstructure:"cn"`
}

// Metrics
var (
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cn_dms_requests_total",
			Help: "Total number of requests to CN DMS",
		},
		[]string{"method", "endpoint", "status"},
	)

	slicesActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "cn_dms_slices_active",
			Help: "Number of active CN slices",
		},
	)

	operationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cn_dms_operation_duration_seconds",
			Help:    "Duration of CN DMS operations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)
)

func init() {
	// Register metrics
	prometheus.MustRegister(requestsTotal)
	prometheus.MustRegister(slicesActive)
	prometheus.MustRegister(operationDuration)
}

func main() {
	// Parse flags
	configFile := flag.String("config", "/config/cn-dms.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("CN DMS %s (build: %s)\n", version, build)
		os.Exit(0)
	}

	// Load configuration
	config := loadConfig(*configFile)

	// Setup logging
	setupLogging(config.Logging.Level, config.Logging.Format)

	log.WithFields(log.Fields{
		"version": version,
		"build":   build,
	}).Info("Starting CN DMS")

	// Create router
	router := setupRouter(config)

	// Start metrics server if enabled
	if config.Metrics.Enabled {
		go startMetricsServer(config.Metrics.Port)
	}

	// Create server
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", config.Server.Port),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,  // Prevent Slowloris attacks
		ReadTimeout:       15 * time.Second,  // Total time to read request
		WriteTimeout:      15 * time.Second,  // Time to write response
		IdleTimeout:       60 * time.Second,  // Keep-alive timeout
		MaxHeaderBytes:    1 << 20,           // 1MB max header size
	}

	// Start server in goroutine
	go func() {
		log.Infof("Starting HTTP server on port %d", config.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Start TLS server if enabled
	if config.Server.TLSEnabled {
		tlsSrv := &http.Server{
			Addr:              fmt.Sprintf(":%d", config.Server.TLSPort),
			Handler:           router,
			ReadHeaderTimeout: 10 * time.Second,  // Prevent Slowloris attacks
			ReadTimeout:       15 * time.Second,  // Total time to read request
			WriteTimeout:      15 * time.Second,  // Time to write response
			IdleTimeout:       60 * time.Second,  // Keep-alive timeout
			MaxHeaderBytes:    1 << 20,           // 1MB max header size
		}

		go func() {
			log.Infof("Starting HTTPS server on port %d", config.Server.TLSPort)
			if err := tlsSrv.ListenAndServeTLS(config.Server.CertFile, config.Server.KeyFile); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Failed to start TLS server: %v", err)
			}
		}()
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down CN DMS...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Errorf("Server forced to shutdown: %v", err)
	}

	log.Info("CN DMS stopped")
}

func loadConfig(configFile string) *Config {
	config := &Config{}

	// Set defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.tls_port", 8443)
	viper.SetDefault("server.tls_enabled", false)
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("metrics.enabled", true)
	viper.SetDefault("metrics.port", 9090)
	viper.SetDefault("cn.update_interval", "30s")
	viper.SetDefault("cn.max_slices", 100)

	// Read from environment
	viper.SetEnvPrefix("CN_DMS")
	viper.AutomaticEnv()

	// Read config file if exists
	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Fatalf("Error reading config file: %v", err)
		}
		log.Warnf("Config file not found, using defaults and environment variables")
	}

	// Unmarshal config
	if err := viper.Unmarshal(config); err != nil {
		log.Fatalf("Unable to decode config: %v", err)
	}

	return config
}

func setupLogging(level, format string) {
	// Set log level
	logLevel, err := log.ParseLevel(level)
	if err != nil {
		logLevel = log.InfoLevel
		log.Warnf("Invalid log level %s, using info", level)
	}
	log.SetLevel(logLevel)

	// Set log format
	if format == "json" {
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp: true,
		})
	}

	// Output to stdout
	log.SetOutput(os.Stdout)
}

func setupRouter(config *Config) *gin.Engine {
	// Set Gin mode based on log level
	if config.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add middleware
	router.Use(customRecoveryMiddleware())
	router.Use(securityHeadersMiddleware())
	router.Use(rateLimitingMiddleware())
	router.Use(loggingMiddleware())
	router.Use(metricsMiddleware())

	// API routes
	api := router.Group("/api/v1")
	{
		// Slice management
		api.GET("/slices", getSlices)
		api.GET("/slices/:id", getSlice)
		api.POST("/slices", createSlice)
		api.PUT("/slices/:id", updateSlice)
		api.DELETE("/slices/:id", deleteSlice)

		// Network functions
		api.GET("/nfs", getNetworkFunctions)
		api.GET("/nfs/:id", getNetworkFunction)
		api.POST("/nfs", deployNetworkFunction)
		api.DELETE("/nfs/:id", undeployNetworkFunction)

		// Core network status
		api.GET("/status", getCNStatus)
		api.GET("/capabilities", getCNCapabilities)
	}

	// Health check
	router.GET("/health", healthCheck)
	router.GET("/ready", readinessCheck)

	// Version
	router.GET("/version", getVersion)

	return router
}

func startMetricsServer(port int) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,  // Prevent Slowloris attacks
		ReadTimeout:       30 * time.Second,  // Total time to read request
		WriteTimeout:      30 * time.Second,  // Time to write response
		IdleTimeout:       120 * time.Second, // Keep-alive timeout
	}

	log.Infof("Starting metrics server on port %d", port)
	if err := server.ListenAndServe(); err != nil {
		log.Errorf("Failed to start metrics server: %v", err)
	}
}

// Middleware

// customRecoveryMiddleware provides enhanced error handling and logging
func customRecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(os.Stderr, func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			log.WithFields(log.Fields{
				"error":      err,
				"path":       c.Request.URL.Path,
				"method":     c.Request.Method,
				"client_ip":  c.ClientIP(),
				"user_agent": c.Request.UserAgent(),
			}).Error("Panic recovered in CN-DMS")
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
			"code":  "INTERNAL_ERROR",
		})
	})
}

// securityHeadersMiddleware adds security headers to all responses
func securityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Remove server information
		c.Header("Server", "")

		c.Next()
	}
}

// rateLimitingMiddleware implements rate limiting
func rateLimitingMiddleware() gin.HandlerFunc {
	limiter := rate.NewLimiter(rate.Limit(100), 200) // 100 req/sec with burst of 200

	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
				"retry_after": "1s",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func loggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/health", "/ready"},
	})
}

func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)
		status := fmt.Sprintf("%d", c.Writer.Status())

		requestsTotal.WithLabelValues(c.Request.Method, c.Request.URL.Path, status).Inc()
		operationDuration.WithLabelValues(c.Request.URL.Path).Observe(duration.Seconds())
	}
}

// Handlers
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"time":   time.Now().Unix(),
	})
}

func readinessCheck(c *gin.Context) {
	// TODO: Add actual readiness checks
	c.JSON(http.StatusOK, gin.H{
		"ready": true,
		"time":  time.Now().Unix(),
	})
}

func getVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": version,
		"build":   build,
	})
}

func getSlices(c *gin.Context) {
	// TODO: Implement slice retrieval
	c.JSON(http.StatusOK, gin.H{
		"slices": []gin.H{},
		"count":  0,
	})
}

func getSlice(c *gin.Context) {
	id := c.Param("id")
	// TODO: Implement single slice retrieval
	c.JSON(http.StatusNotFound, gin.H{
		"error": fmt.Sprintf("Slice %s not found", id),
	})
}

func createSlice(c *gin.Context) {
	// TODO: Implement slice creation
	c.JSON(http.StatusCreated, gin.H{
		"message": "Slice creation initiated",
		"id":      "cn-slice-001",
	})
}

func updateSlice(c *gin.Context) {
	id := c.Param("id")
	// TODO: Implement slice update
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Slice %s updated", id),
	})
}

func deleteSlice(c *gin.Context) {
	id := c.Param("id")
	// TODO: Implement slice deletion
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Slice %s deleted", id),
	})
}

func getNetworkFunctions(c *gin.Context) {
	// TODO: Implement NF retrieval
	c.JSON(http.StatusOK, gin.H{
		"nfs":   []gin.H{},
		"count": 0,
	})
}

func getNetworkFunction(c *gin.Context) {
	id := c.Param("id")
	// TODO: Implement single NF retrieval
	c.JSON(http.StatusNotFound, gin.H{
		"error": fmt.Sprintf("Network function %s not found", id),
	})
}

func deployNetworkFunction(c *gin.Context) {
	// TODO: Implement NF deployment
	c.JSON(http.StatusCreated, gin.H{
		"message": "Network function deployment initiated",
		"id":      "nf-001",
	})
}

func undeployNetworkFunction(c *gin.Context) {
	id := c.Param("id")
	// TODO: Implement NF undeployment
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Network function %s undeployed", id),
	})
}

func getCNStatus(c *gin.Context) {
	// TODO: Implement status retrieval
	c.JSON(http.StatusOK, gin.H{
		"status": "operational",
		"uptime": time.Now().Unix(),
		"slices": 0,
		"nfs":    0,
	})
}

func getCNCapabilities(c *gin.Context) {
	// TODO: Implement capabilities retrieval
	c.JSON(http.StatusOK, gin.H{
		"capabilities": gin.H{
			"max_slices":          100,
			"supported_nf_types":  []string{"AMF", "SMF", "UPF", "PCF", "UDM"},
			"slice_types":         []string{"eMBB", "URLLC", "mMTC"},
			"api_version":         "v1",
		},
	})
}