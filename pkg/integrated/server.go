package integrated

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/manusa/kubernetes-mcp-server/pkg/config"
	"github.com/manusa/kubernetes-mcp-server/pkg/mcp"
	"github.com/manusa/kubernetes-mcp-server/pkg/output"

	// Import the Python inference server components
	_ "github.com/manusa/kubernetes-mcp-server/python/kubernetes_mcp_server"
)

type IntegratedServer struct {
	mcpServer       *mcp.Server
	httpServer      *http.Server
	inferenceServer *http.Server
	config          *IntegratedConfig
}

type IntegratedConfig struct {
	// MCP Configuration
	MCPProfile      string
	MCPPort         int
	MCPReadOnly     bool
	
	// Inference Configuration
	InferencePort   int
	ModelsPath      string
	
	// CI/CD Configuration
	DefaultRegistry string
	DefaultNamespace string
	
	// General Configuration
	LogLevel        int
	KubeConfig      string
}

func NewIntegratedServer(config *IntegratedConfig) (*IntegratedServer, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Create MCP server configuration
	mcpConfig := mcp.Configuration{
		Profile:    mcp.ProfileFromString(config.MCPProfile),
		ListOutput: output.Table,
		StaticConfig: &config.StaticConfig{
			ReadOnly: config.MCPReadOnly,
			LogLevel: config.LogLevel,
		},
	}

	// Initialize MCP server
	mcpServer, err := mcp.NewServer(mcpConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}

	// Create HTTP server for MCP
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/mcp", mcpServer.ServeHTTP)
	httpMux.HandleFunc("/health/mcp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"mcp-server"}`))
	})
	
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.MCPPort),
		Handler: httpMux,
	}

	// Create inference server
	inferenceMux := http.NewServeMux()
	
	// Add inference endpoints
	inferenceMux.HandleFunc("/infer", handleInference)
	inferenceMux.HandleFunc("/models", handleListModels)
	inferenceMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"inference-server"}`))
	})
	inferenceMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := fmt.Sprintf(`{
			"service": "OpenShift AI MCP Server",
			"version": "1.0.0",
			"components": {
				"mcp": "http://localhost:%d/mcp",
				"inference": "http://localhost:%d/infer",
				"health": {
					"mcp": "http://localhost:%d/health/mcp",
					"inference": "http://localhost:%d/health"
				}
			}
		}`, config.MCPPort, config.InferencePort, config.MCPPort, config.InferencePort)
		w.Write([]byte(response))
	})

	inferenceServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.InferencePort),
		Handler: inferenceMux,
	}

	return &IntegratedServer{
		mcpServer:       mcpServer,
		httpServer:      httpServer,
		inferenceServer: inferenceServer,
		config:          config,
	}, nil
}

func DefaultConfig() *IntegratedConfig {
	return &IntegratedConfig{
		MCPProfile:       "cicd",
		MCPPort:         8081,
		MCPReadOnly:     false,
		InferencePort:   8080,
		ModelsPath:      "/app/models",
		DefaultRegistry: "quay.io",
		DefaultNamespace: "openshift-ai-mcp",
		LogLevel:        2,
		KubeConfig:      "",
	}
}

func (s *IntegratedServer) Start(ctx context.Context) error {
	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start MCP server
	go func() {
		log.Printf("Starting MCP server on port %d", s.config.MCPPort)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("MCP server error: %v", err)
		}
	}()

	// Start inference server
	go func() {
		log.Printf("Starting inference server on port %d", s.config.InferencePort)
		if err := s.inferenceServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Inference server error: %v", err)
		}
	}()

	// Start MCP server background processes
	go func() {
		if err := s.mcpServer.Start(ctx); err != nil {
			log.Printf("MCP server start error: %v", err)
		}
	}()

	log.Printf("Integrated server started successfully")
	log.Printf("MCP endpoint: http://localhost:%d/mcp", s.config.MCPPort)
	log.Printf("Inference endpoint: http://localhost:%d/infer", s.config.InferencePort)
	log.Printf("Health check: http://localhost:%d/health", s.config.InferencePort)

	// Wait for shutdown signal
	select {
	case <-ctx.Done():
		log.Println("Context cancelled, shutting down...")
	case sig := <-sigChan:
		log.Printf("Received signal %s, shutting down...", sig)
	}

	return s.Shutdown()
}

func (s *IntegratedServer) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Println("Shutting down servers...")

	// Shutdown HTTP servers
	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down MCP server: %v", err)
	}

	if err := s.inferenceServer.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down inference server: %v", err)
	}

	// Shutdown MCP server
	if err := s.mcpServer.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down MCP server: %v", err)
	}

	log.Println("Shutdown complete")
	return nil
}

// Inference handler functions (simplified versions)
func handleInference(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// This would typically call the Python inference server
	// For now, we'll return a simple response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := `{
		"outputs": "Mock inference response",
		"model_name": "default",
		"processing_time_ms": 10.5,
		"metadata": {"status": "success"}
	}`
	w.Write([]byte(response))
}

func handleListModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := `{
		"models": ["default", "simple_classifier", "text_embeddings"],
		"count": 3
	}`
	w.Write([]byte(response))
}

// Configuration loading from environment
func LoadConfigFromEnv() *IntegratedConfig {
	config := DefaultConfig()

	if port := os.Getenv("MCP_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.MCPPort = p
		}
	}

	if port := os.Getenv("INFERENCE_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.InferencePort = p
		}
	}

	if port := os.Getenv("PORT"); port != "" {
		// If generic PORT is set, use it for inference
		if p, err := strconv.Atoi(port); err == nil {
			config.InferencePort = p
		}
	}

	if profile := os.Getenv("MCP_PROFILE"); profile != "" {
		config.MCPProfile = profile
	}

	if readOnly := os.Getenv("MCP_READ_ONLY"); readOnly == "true" {
		config.MCPReadOnly = true
	}

	if modelsPath := os.Getenv("MODELS_PATH"); modelsPath != "" {
		config.ModelsPath = modelsPath
	}

	if registry := os.Getenv("DEFAULT_REGISTRY"); registry != "" {
		config.DefaultRegistry = registry
	}

	if namespace := os.Getenv("DEFAULT_NAMESPACE"); namespace != "" {
		config.DefaultNamespace = namespace
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		if level, err := strconv.Atoi(logLevel); err == nil {
			config.LogLevel = level
		}
	}

	if kubeConfig := os.Getenv("KUBECONFIG"); kubeConfig != "" {
		config.KubeConfig = kubeConfig
	}

	return config
}
