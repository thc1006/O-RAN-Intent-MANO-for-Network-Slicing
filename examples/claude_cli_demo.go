package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/claude"
)

// This example demonstrates how to use the Claude CLI integration
// for natural language processing of network slicing intents.
//
// Prerequisites:
// 1. tmux must be installed: sudo apt-get install tmux
// 2. claude CLI must be installed and configured
// 3. Run: claude --dangerously-skip-permissions (to verify it works)

func main() {
	ctx := context.Background()

	fmt.Println("🚀 O-RAN Network Slicing with Claude CLI Integration")
	fmt.Println("====================================================\n")

	// Create Claude client with tmux integration
	config := &claude.ClientConfig{
		SessionName: "oran-demo",
		Timeout:     30 * time.Second,
		UseFallback: false, // Force real Claude CLI usage
	}

	fmt.Println("📝 Initializing Claude CLI client...")
	client, err := claude.NewClient(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create Claude client: %v", err)
	}
	defer client.Cleanup(ctx)

	if client.IsFallbackMode() {
		fmt.Println("⚠️  Running in fallback mode (Claude CLI not available)")
	} else {
		fmt.Println("✅ Claude CLI connected via tmux session")
	}

	// Example 1: Create an eMBB slice
	fmt.Println("\n📡 Example 1: Creating eMBB Slice")
	fmt.Println("----------------------------------")
	processIntent(ctx, client, "Deploy an enhanced mobile broadband network slice for 4K video streaming with 1 Gbps throughput and 20ms latency")

	// Example 2: Create a URLLC slice
	fmt.Println("\n⚡ Example 2: Creating URLLC Slice")
	fmt.Println("----------------------------------")
	processIntent(ctx, client, "Create an ultra-reliable low latency slice for autonomous vehicle control with 1ms latency and 99.999% reliability")

	// Example 3: Create an mIoT slice
	fmt.Println("\n🌐 Example 2: Creating mIoT Slice")
	fmt.Println("----------------------------------")
	processIntent(ctx, client, "Setup a massive IoT network slice for smart city sensors supporting 1 million devices per km²")

	// Example 4: Modify existing slice
	fmt.Println("\n🔧 Example 4: Modifying Existing Slice")
	fmt.Println("--------------------------------------")
	client.SetContext(ctx, "embb-slice-001", "eMBB")
	processIntent(ctx, client, "Increase the throughput to 2 Gbps and reduce latency to 10ms for better performance")

	// Example 5: Batch processing
	fmt.Println("\n📦 Example 5: Batch Processing")
	fmt.Println("-----------------------------")
	intents := []string{
		"Create a network slice for emergency services",
		"Deploy IoT slice for industrial automation",
		"Setup broadband slice for residential users",
	}

	results, err := client.ProcessBatch(ctx, intents)
	if err != nil {
		log.Printf("Batch processing error: %v", err)
	} else {
		for i, result := range results {
			if result.Success {
				fmt.Printf("  Intent %d: ✅ Processed successfully\n", i+1)
				if result.Response != nil && result.Response.ParsedIntent != nil {
					fmt.Printf("    - Slice Type: %s\n", result.Response.ParsedIntent.SliceType)
				}
			} else {
				fmt.Printf("  Intent %d: ❌ Failed: %v\n", i+1, result.Error)
			}
		}
	}

	// Example 6: Export configuration
	fmt.Println("\n💾 Example 6: Exporting Configuration")
	fmt.Println("-------------------------------------")

	// Create a sample response for export
	sampleResponse := &claude.IntentResponse{
		ActionType: "create",
		ParsedIntent: &claude.ParsedIntent{
			SliceType: "eMBB",
			Requirements: &claude.Requirements{
				Throughput:  1000,
				Latency:     20,
				Reliability: 99.9,
			},
		},
	}

	yamlConfig, err := client.ExportToYAML(ctx, sampleResponse)
	if err == nil {
		fmt.Println("YAML Configuration:")
		fmt.Println(yamlConfig)
	}

	fmt.Println("\n✨ Demo completed successfully!")
}

func processIntent(ctx context.Context, client *claude.Client, intentText string) {
	fmt.Printf("\n📝 Intent: \"%s\"\n", intentText)

	intent := &claude.IntentRequest{
		Text: intentText,
	}

	response, err := client.ProcessIntent(ctx, intent)
	if err != nil {
		log.Printf("❌ Error processing intent: %v", err)
		return
	}

	if response.ParsedIntent != nil {
		fmt.Printf("✅ Parsed Results:\n")
		fmt.Printf("   - Action Type: %s\n", response.ActionType)
		fmt.Printf("   - Slice Type: %s\n", response.ParsedIntent.SliceType)
		if response.ParsedIntent.Requirements != nil {
			fmt.Printf("   - Requirements:\n")
			if response.ParsedIntent.Requirements.Throughput > 0 {
				fmt.Printf("     • Throughput: %d Mbps\n", response.ParsedIntent.Requirements.Throughput)
			}
			if response.ParsedIntent.Requirements.Latency > 0 {
				fmt.Printf("     • Latency: %.1f ms\n", response.ParsedIntent.Requirements.Latency)
			}
			if response.ParsedIntent.Requirements.Reliability > 0 {
				fmt.Printf("     • Reliability: %.3f%%\n", response.ParsedIntent.Requirements.Reliability)
			}
		}
	}

	if response.Response != "" {
		fmt.Printf("   - Response: %s\n", response.Response)
	}
}