package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/o2client"
)

const (
	defaultPort = "8080"
	defaultO2URL = "http://localhost:8081"
)

func main() {
	// Setup router
	router := gin.Default()

	// Initialize O2 client
	o2URL := os.Getenv("O2_BASE_URL")
	if o2URL == "" {
		o2URL = defaultO2URL
	}

	client := o2client.NewClient(o2URL)

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"service": "o2-client",
			"timestamp": time.Now().UTC(),
		})
	})

	// O2 DMS endpoints
	router.GET("/deployment-managers", func(c *gin.Context) {
		dms, err := client.GetDeploymentManagers(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, dms)
	})

	router.GET("/sites", func(c *gin.Context) {
		sites, err := client.GetAvailableSites(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"sites": sites})
	})

	router.POST("/deploy/:dmsId", func(c *gin.Context) {
		dmsID := c.Param("dmsId")
		var nfSpec interface{}
		if err := c.ShouldBindJSON(&nfSpec); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := client.DeployNetworkFunction(c.Request.Context(), dmsID, nfSpec)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "deployment initiated"})
	})

	router.GET("/deployment/:id/status", func(c *gin.Context) {
		deploymentID := c.Param("id")
		status, err := client.GetDeploymentStatus(c.Request.Context(), deploymentID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, status)
	})

	router.DELETE("/deployment/:id", func(c *gin.Context) {
		deploymentID := c.Param("id")
		err := client.DeleteDeployment(c.Request.Context(), deploymentID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "deployment deleted"})
	})

	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	// Setup server with timeout configurations to prevent Slowloris attacks
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("O2 Client server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Give the server 30 seconds to finish handling requests
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}