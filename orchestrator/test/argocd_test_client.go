package main

import (
	"context"
	"fmt"
	"log"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/orchestrator/pkg/argocd"
	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/orchestrator/pkg/slices"
)

func main() {
	log.Println("Testing Argo CD client...")

	// Initialize client
	client, err := argocd.NewClient("argocd")
	if err != nil {
		log.Fatalf("Failed to create Argo CD client: %v", err)
	}
	log.Println("✓ Argo CD client created successfully")

	// Create a test slice configuration
	sliceConfig := slices.SliceConfiguration{
		SliceID:   "test-e2e-slice",
		SliceType: "eMBB",
		Namespace: "oran-slice-test",
		QoS: slices.QoSProfile{
			Throughput:  1000,
			Latency:     20,
			Reliability: 99.9,
		},
	}

	// Generate manifests
	manifests, err := slices.GenerateSliceManifests(sliceConfig)
	if err != nil {
		log.Fatalf("Failed to generate manifests: %v", err)
	}
	log.Println("✓ Manifests generated successfully")

	// Serialize to YAML
	yamlData, err := manifests.ToYAML()
	if err != nil {
		log.Fatalf("Failed to serialize manifests: %v", err)
	}
	log.Printf("✓ Manifests serialized (%d bytes)", len(yamlData))

	// Create Argo CD Application
	appConfig := argocd.ApplicationConfig{
		Name:          sliceConfig.SliceID,
		Namespace:     sliceConfig.Namespace,
		SliceType:     sliceConfig.SliceType,
		ManifestsYAML: string(yamlData),
	}

	ctx := context.Background()
	app, err := client.CreateApplication(ctx, appConfig)
	if err != nil {
		log.Fatalf("Failed to create Argo CD application: %v", err)
	}
	log.Printf("✓ Argo CD Application created: %s", app.Name)

	// Get sync status
	syncStatus, err := client.GetSyncStatus(ctx, sliceConfig.SliceID)
	if err != nil {
		log.Printf("Warning: Failed to get sync status: %v", err)
	} else {
		log.Printf("✓ Sync Status: %s", syncStatus)
	}

	// Get health status
	healthStatus, err := client.GetHealthStatus(ctx, sliceConfig.SliceID)
	if err != nil {
		log.Printf("Warning: Failed to get health status: %v", err)
	} else {
		log.Printf("✓ Health Status: %s", healthStatus)
	}

	fmt.Println("\n=== Test completed successfully! ===")
}