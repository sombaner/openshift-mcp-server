package integrated

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	mcpconfig "github.com/sur309/openshift-mcp-server/pkg/config"
	"github.com/sur309/openshift-mcp-server/pkg/mcp"
	"github.com/sur309/openshift-mcp-server/pkg/output"
	// Note: Python inference server runs separately in the same container
)

type IntegratedServer struct {
	mcpServer       *mcp.Server
	httpServer      *http.Server
	inferenceServer *http.Server
	config          *IntegratedConfig
}

type IntegratedConfig struct {
	// MCP Configuration
	MCPProfile  string
	MCPPort     int
	MCPReadOnly bool

	// Inference Configuration
	InferencePort int
	ModelsPath    string

	// CI/CD Configuration
	DefaultRegistry  string
	DefaultNamespace string

	// General Configuration
	LogLevel   int
	KubeConfig string
}

func NewIntegratedServer(config *IntegratedConfig) (*IntegratedServer, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Create MCP server configuration
	mcpConfig := mcp.Configuration{
		Profile:    mcp.ProfileFromString(config.MCPProfile),
		ListOutput: output.FromString("table"),
		StaticConfig: &mcpconfig.StaticConfig{
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
	httpServer := &http.Server{
		Addr: fmt.Sprintf(":%d", config.MCPPort),
	}

	// Add MCP health endpoint
	httpMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"mcp-server"}`))
	})

	// Mount the MCP server HTTP handler at root for JSON-RPC (initialize, tools/list, tools/call, ...)
	streamHandler := mcpServer.ServeHTTP(httpServer)
	httpMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := fmt.Sprintf(`{
                "service": "OpenShift AI MCP Server",
                "version": "1.0.0",
                "protocol": "model-context-protocol",
                "capabilities": [
                    "git_repository_management",
                    "container_image_building",
                    "deployment_automation",
                    "kubernetes_operations"
                ],
                "endpoints": {
                    "health": "http://localhost:%d/health",
                    "jsonrpc": "http://localhost:%d/"
                }
            }`, config.MCPPort, config.MCPPort)
			_, _ = w.Write([]byte(response))
			return
		}
		// Delegate non-GET methods (e.g., POST) to the MCP JSON-RPC handler
		streamHandler.ServeHTTP(w, r)
	})

	// Attach mux to server
	httpServer.Handler = httpMux

	// Create inference server
	inferenceMux := http.NewServeMux()

	// Add inference endpoints (minimal versions)
	inferenceMux.HandleFunc("/infer", handleMinimalInference)
	inferenceMux.HandleFunc("/models", handleMinimalListModels)
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
		MCPPort:          8081,
		MCPReadOnly:      false,
		InferencePort:    8080,
		ModelsPath:       "/app/models",
		DefaultRegistry:  "quay.io",
		DefaultNamespace: "ai-mcp-openshift",
		LogLevel:         2,
		KubeConfig:       "",
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

	// MCP server is integrated into HTTP handlers above
	log.Printf("MCP server initialized successfully")

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

	// Close MCP server
	s.mcpServer.Close()

	log.Println("Shutdown complete")
	return nil
}

// Minimal inference handler functions (optimized for size)
func handleMinimalInference(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Lightweight mock inference response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := `{
		"outputs": {
			"message": "Lightweight inference completed",
			"input_processed": true,
			"response_id": "minimal_resp_001"
		},
		"model_name": "lightweight",
		"processing_time_ms": 2.5,
		"metadata": {
			"model_type": "mock",
			"model_size_mb": 0.1,
			"mode": "minimal",
			"optimized": true
		}
	}`
	w.Write([]byte(response))
}

func handleMinimalListModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := `{
		"models": [
			{
				"name": "lightweight",
				"type": "mock", 
				"description": "Minimal model for CI/CD testing",
				"size_mb": 0.1,
				"ready": true
			}
		],
		"total_models": 1,
		"total_size_mb": 0.1
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

// JSON-RPC request/response structures for MCP
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      interface{} `json:"id"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// Handle MCP JSON-RPC requests
func handleMCPRequest(w http.ResponseWriter, r *http.Request, mcpServer *mcp.Server) {
	w.Header().Set("Content-Type", "application/json")

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Parse JSON-RPC request
	var req JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		response := JSONRPCResponse{
			JSONRPC: "2.0",
			Error:   map[string]interface{}{"code": -32700, "message": "Parse error"},
			ID:      nil,
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Handle different MCP methods
	var result interface{}
	var responseError interface{}

	switch req.Method {
	case "initialize":
		result = map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{
					"listChanged": false,
				},
			},
			"serverInfo": map[string]interface{}{
				"name":    "OpenShift AI MCP Server",
				"version": "1.0.0",
			},
		}
	case "tools/list":
		// Return available tools from the MCP server
		result = map[string]interface{}{
			"tools": []map[string]interface{}{
				{
					"name":        "repo_add",
					"description": "Add a Git repository for CI/CD monitoring and automated deployment",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"url": map[string]interface{}{
								"type":        "string",
								"description": "Git repository URL (e.g., https://github.com/user/repo.git)",
							},
							"name": map[string]interface{}{
								"type":        "string",
								"description": "Friendly name for the repository (Optional)",
							},
							"branch": map[string]interface{}{
								"type":        "string",
								"description": "Git branch to monitor (Optional, defaults to 'main')",
							},
							"namespace": map[string]interface{}{
								"type":        "string",
								"description": "OpenShift/Kubernetes namespace for deployment (Required)",
							},
							"dockerfile": map[string]interface{}{
								"type":        "string",
								"description": "Path to Dockerfile (Optional, defaults to './Dockerfile')",
							},
							"registry": map[string]interface{}{
								"type":        "string",
								"description": "Container registry URL (Optional, defaults to 'quay.io')",
							},
							"image_name": map[string]interface{}{
								"type":        "string",
								"description": "Container image name (Optional, auto-generated from repo name)",
							},
						},
						"required": []string{"url", "namespace"},
					},
				},
				{
					"name":        "repo_list",
					"description": "List all monitored Git repositories with their CI/CD configurations",
					"inputSchema": map[string]interface{}{
						"type":       "object",
						"properties": map[string]interface{}{},
					},
				},
				{
					"name":        "repo_status",
					"description": "Get detailed status of a specific repository's CI/CD pipeline",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name": map[string]interface{}{
								"type":        "string",
								"description": "Repository name or URL",
							},
						},
						"required": []string{"name"},
					},
				},
				{
					"name":        "repo_build",
					"description": "Trigger a manual build for a specific repository",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name": map[string]interface{}{
								"type":        "string",
								"description": "Repository name or URL",
							},
							"commit": map[string]interface{}{
								"type":        "string",
								"description": "Specific commit hash to build (Optional, defaults to latest)",
							},
							"push": map[string]interface{}{
								"type":        "boolean",
								"description": "Push built image to registry (Optional, defaults to true)",
							},
						},
						"required": []string{"name"},
					},
				},
				{
					"name":        "repo_deploy",
					"description": "Deploy a repository to its configured OpenShift namespace",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name": map[string]interface{}{
								"type":        "string",
								"description": "Repository name or URL",
							},
							"image_tag": map[string]interface{}{
								"type":        "string",
								"description": "Specific image tag to deploy (Optional, defaults to latest)",
							},
							"namespace": map[string]interface{}{
								"type":        "string",
								"description": "Override target namespace (Optional, uses repo config)",
							},
						},
						"required": []string{"name"},
					},
				},
				{
					"name":        "namespace_create",
					"description": "Create a new OpenShift namespace/project for deployments",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name": map[string]interface{}{
								"type":        "string",
								"description": "Namespace name",
							},
							"display_name": map[string]interface{}{
								"type":        "string",
								"description": "Display name for OpenShift project (Optional)",
							},
							"description": map[string]interface{}{
								"type":        "string",
								"description": "Description for the namespace/project (Optional)",
							},
						},
						"required": []string{"name"},
					},
				},
				{
					"name":        "cicd_status",
					"description": "Get overall CI/CD system status and capabilities",
					"inputSchema": map[string]interface{}{
						"type":       "object",
						"properties": map[string]interface{}{},
					},
				},
			},
		}
	case "tools/call":
		// Handle tool calls - this would integrate with the actual MCP server tools
		responseError = map[string]interface{}{
			"code":    -32601,
			"message": "Tool execution not implemented in this simplified handler",
		}
	default:
		responseError = map[string]interface{}{
			"code":    -32601,
			"message": "Method not found: " + req.Method,
		}
	}

	// Send response
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		Error:   responseError,
		ID:      req.ID,
	}

	json.NewEncoder(w).Encode(response)
}
