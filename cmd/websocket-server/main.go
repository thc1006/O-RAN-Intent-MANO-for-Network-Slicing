package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/websocket"
)

func main() {
	var (
		addr = flag.String("addr", ":8080", "HTTP service address")
		help = flag.Bool("help", false, "Show help message")
	)
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	fmt.Println("🚀 O-RAN Network Slicing WebSocket Server")
	fmt.Println("=========================================")
	fmt.Printf("📡 Server starting on %s\n", *addr)
	fmt.Println("🤖 Claude CLI + tmux integration enabled")
	fmt.Println()

	// Create WebSocket server
	server := websocket.NewServer(*addr)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("🛑 Shutdown signal received, closing server...")
		cancel()
	}()

	// Start server in goroutine
	go func() {
		log.Printf("🌐 WebSocket server running at http://localhost%s", *addr)
		log.Printf("💻 Frontend available at http://localhost%s/", *addr)
		log.Printf("🔌 WebSocket endpoint: ws://localhost%s/ws", *addr)
		log.Printf("❤️  Health check: http://localhost%s/health", *addr)

		if err := server.Start(); err != nil {
			log.Fatalf("❌ Server failed to start: %v", err)
		}
	}()

	// Wait for shutdown
	<-ctx.Done()
	log.Println("✅ Server shutdown complete")
}

func showHelp() {
	fmt.Println("O-RAN Network Slicing WebSocket Server")
	fmt.Println("=====================================")
	fmt.Println()
	fmt.Println("This server provides a WebSocket interface for natural language")
	fmt.Println("processing of network slicing intents using Claude CLI + tmux.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  websocket-server [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -addr string")
	fmt.Println("        HTTP service address (default \":8080\")")
	fmt.Println("  -help")
	fmt.Println("        Show this help message")
	fmt.Println()
	fmt.Println("Endpoints:")
	fmt.Println("  /          - Frontend web interface")
	fmt.Println("  /ws        - WebSocket endpoint for client connections")
	fmt.Println("  /health    - Health check endpoint")
	fmt.Println()
	fmt.Println("Prerequisites:")
	fmt.Println("  - tmux must be installed")
	fmt.Println("  - claude CLI must be installed and configured")
	fmt.Println("  - Run 'claude --dangerously-skip-permissions' to verify setup")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Start server on default port 8080")
	fmt.Println("  websocket-server")
	fmt.Println()
	fmt.Println("  # Start server on custom port")
	fmt.Println("  websocket-server -addr :9090")
	fmt.Println()
	fmt.Println("  # Access the web interface")
	fmt.Println("  open http://localhost:8080")
	fmt.Println()
	fmt.Println("Demo Scenarios:")
	fmt.Println("  Try these natural language intents:")
	fmt.Println("  - \"Deploy an eMBB slice for 4K video streaming with 1 Gbps throughput\"")
	fmt.Println("  - \"Create a URLLC slice for autonomous vehicle control with 1ms latency\"")
	fmt.Println("  - \"Setup mIoT slice for smart city sensors supporting 1M devices\"")
}