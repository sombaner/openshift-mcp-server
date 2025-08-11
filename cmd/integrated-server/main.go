package main

import (
	"context"
	"log"

	"github.com/manusa/kubernetes-mcp-server/pkg/integrated"
)

func main() {
	log.Println("Starting OpenShift AI MCP Server with Inference...")

	// Load configuration from environment
	config := integrated.LoadConfigFromEnv()

	// Create integrated server
	server, err := integrated.NewIntegratedServer(config)
	if err != nil {
		log.Fatalf("Failed to create integrated server: %v", err)
	}

	// Start the server
	ctx := context.Background()
	if err := server.Start(ctx); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
