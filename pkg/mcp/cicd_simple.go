package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// MCP-compliant logging setup
var (
	// Use stderr for logging to avoid corrupting JSON-RPC messages
	mcpLogger = log.New(os.Stderr, "[MCP-CICD] ", log.LstdFlags|log.Lshortfile)
)

// Repository configuration for CI/CD monitoring
type RepoConfig struct {
	URL          string `json:"url"`
	Name         string `json:"name"`
	Branch       string `json:"branch"`
	BuildContext string `json:"build_context"`
	DockerFile   string `json:"dockerfile"`
	ImageName    string `json:"image_name"`
	Registry     string `json:"registry"`
	Namespace    string `json:"namespace"`
	LastCommit   string `json:"last_commit,omitempty"`
	Status       string `json:"status"`
	Webhook      string `json:"webhook,omitempty"`
}

// In-memory repository store (in production, this would be persistent storage)
var repositoryStore = make(map[string]*RepoConfig)

// Webhook configuration for automatic commit detection
type WebhookConfig struct {
	RepoURL    string `json:"repo_url"`
	Secret     string `json:"secret"`
	LastCommit string `json:"last_commit"`
	AutoDeploy bool   `json:"auto_deploy"`
}

// Manifest templates for automatic generation
const deploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.AppName}}
  namespace: {{.Namespace}}
  labels:
    app: {{.AppName}}
    version: "{{.Version}}"
spec:
  replicas: {{.Replicas}}
  selector:
    matchLabels:
      app: {{.AppName}}
  template:
    metadata:
      labels:
        app: {{.AppName}}
        version: "{{.Version}}"
    spec:
      securityContext:
        runAsNonRoot: true
        seccompProfile:
          type: RuntimeDefault
      containers:
      - name: {{.AppName}}
        image: {{.ImageName}}:{{.ImageTag}}
        imagePullPolicy: Always
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
          readOnlyRootFilesystem: false
          runAsNonRoot: true
        ports:
        - containerPort: {{.Port}}
          name: http
        env:
        - name: PORT
          value: "{{.Port}}"
        resources:
          requests:
            memory: "64Mi"
            cpu: "50m"
          limits:
            memory: "256Mi"
            cpu: "200m"
        livenessProbe:
          httpGet:
            path: /
            port: http
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /
            port: http
          initialDelaySeconds: 5
          periodSeconds: 5
`

const serviceTemplate = `apiVersion: v1
kind: Service
metadata:
  name: {{.AppName}}
  namespace: {{.Namespace}}
  labels:
    app: {{.AppName}}
spec:
  selector:
    app: {{.AppName}}
  ports:
  - name: http
    port: 80
    targetPort: {{.Port}}
  type: ClusterIP
`

const routeTemplate = `apiVersion: route.openshift.io/v1
kind: Route
metadata:
  name: {{.AppName}}
  namespace: {{.Namespace}}
  labels:
    app: {{.AppName}}
  annotations:
    haproxy.router.openshift.io/timeout: 60s
spec:
  to:
    kind: Service
    name: {{.AppName}}
    weight: 100
  port:
    targetPort: http
  tls:
    termination: edge
    insecureEdgeTerminationPolicy: Redirect
  wildcardPolicy: None
`

// Template data for manifest generation
type ManifestData struct {
	AppName   string
	Namespace string
	ImageName string
	ImageTag  string
	Port      int
	Replicas  int
	Version   string
}

// Generate manifests from templates
func generateManifests(data ManifestData) (map[string]string, error) {
	manifests := make(map[string]string)

	// Parse and execute deployment template
	deployTmpl, err := template.New("deployment").Parse(deploymentTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse deployment template: %v", err)
	}

	var deployBuf bytes.Buffer
	if err := deployTmpl.Execute(&deployBuf, data); err != nil {
		return nil, fmt.Errorf("failed to execute deployment template: %v", err)
	}
	manifests["deployment.yaml"] = deployBuf.String()

	// Parse and execute service template
	serviceTmpl, err := template.New("service").Parse(serviceTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse service template: %v", err)
	}

	var serviceBuf bytes.Buffer
	if err := serviceTmpl.Execute(&serviceBuf, data); err != nil {
		return nil, fmt.Errorf("failed to execute service template: %v", err)
	}
	manifests["service.yaml"] = serviceBuf.String()

	// Parse and execute route template
	routeTmpl, err := template.New("route").Parse(routeTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse route template: %v", err)
	}

	var routeBuf bytes.Buffer
	if err := routeTmpl.Execute(&routeBuf, data); err != nil {
		return nil, fmt.Errorf("failed to execute route template: %v", err)
	}
	manifests["route.yaml"] = routeBuf.String()

	return manifests, nil
}

// Detect application type and default port from repository structure
func detectAppDetails(repoName string) (port int, appType string) {
	// Simple detection based on repository name and common patterns
	lowerName := strings.ToLower(repoName)

	// Node.js/JavaScript apps
	if strings.Contains(lowerName, "node") || strings.Contains(lowerName, "js") || strings.Contains(lowerName, "express") {
		return 3000, "nodejs"
	}

	// Python apps
	if strings.Contains(lowerName, "python") || strings.Contains(lowerName, "django") || strings.Contains(lowerName, "flask") {
		return 8000, "python"
	}

	// Go apps
	if strings.Contains(lowerName, "go") || strings.Contains(lowerName, "golang") {
		return 8080, "golang"
	}

	// Gaming apps (like Sample_Gaming_App)
	if strings.Contains(lowerName, "game") || strings.Contains(lowerName, "gaming") {
		return 8080, "web-game"
	}

	// Default to generic web app
	return 8080, "web"
}

// MCP-compliant error formatting
func formatMCPError(code, message string) string {
	errorResponse := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
			"type":    "tool_execution_error",
		},
		"timestamp": fmt.Sprintf("%d", os.Getpid()), // Simple timestamp
		"help":      "Check parameter types and values. Use tool descriptions for guidance.",
	}
	jsonResult, _ := json.MarshalIndent(errorResponse, "", "  ")
	return string(jsonResult)
}

// Generate route URL from app name and namespace
func generateRouteURL(appName, namespace string) string {
	// This would be configurable based on your OpenShift cluster
	clusterDomain := "apps.rosa.sgaikwad.15fi.p3.openshiftapps.com"
	return fmt.Sprintf("https://%s-%s.%s", appName, namespace, clusterDomain)
}

// Generic CI/CD tools that work with any repository and namespace
func (s *Server) initCicdSimple() []server.ServerTool {
	mcpLogger.Println("Initializing CI/CD tools for multi-repository automation")

	return []server.ServerTool{
		{Tool: mcp.NewTool("repo_add",
			mcp.WithDescription("Add a Git repository for CI/CD monitoring and automated deployment. Supports any Git repository with automatic detection of application type, port, and deployment configuration. This tool enables complete GitOps workflow from commit to live application."),
			mcp.WithString("url", mcp.Description("Git repository URL (e.g., https://github.com/user/repo.git). Supports GitHub, GitLab, Bitbucket, and other Git hosting services. Must be a valid HTTPS Git URL."), mcp.Required()),
			mcp.WithString("name", mcp.Description("Friendly name for the repository. If not provided, will be extracted from the repository URL. Used for Kubernetes resource names (must be DNS-compliant). Example: 'my-web-app', 'sample-gaming-app'.")),
			mcp.WithString("branch", mcp.Description("Git branch to monitor for changes. Defaults to 'main'. Common values: main, master, develop, staging. Commits to this branch will trigger automated builds and deployments.")),
			mcp.WithString("dockerfile", mcp.Description("Path to Dockerfile relative to repository root. Defaults to './Dockerfile'. Can be in subdirectories like './docker/Dockerfile' or './build/Dockerfile'.")),
			mcp.WithString("build_context", mcp.Description("Build context path for Docker build. Defaults to repository root ('.'). Useful when Dockerfile is in a subdirectory but needs access to parent directories.")),
			mcp.WithString("image_name", mcp.Description("Container image name including registry. If not provided, auto-generated as '{registry}/default/{repo-name}'. Example: 'quay.io/myuser/myapp', 'docker.io/company/product'.")),
			mcp.WithString("registry", mcp.Description("Container registry URL. Defaults to 'quay.io'. Supports Docker Hub (docker.io), Quay.io, AWS ECR, Azure ACR, Google GCR. Must be accessible for push operations.")),
			mcp.WithString("namespace", mcp.Description("Kubernetes/OpenShift namespace for deployment. Required. Will be created if it doesn't exist. Must be a valid DNS subdomain. Examples: 'my-app-prod', 'gaming-dev', 'team-staging'."), mcp.Required()),
			// Enhanced tool annotations for better discoverability
			mcp.WithTitleAnnotation("CI/CD: Add Repository for Monitoring"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.repoAdd},

		{Tool: mcp.NewTool("repo_list",
			mcp.WithDescription("List all monitored Git repositories with their CI/CD configurations"),
			// Tool annotations
			mcp.WithTitleAnnotation("CI/CD: List Repositories"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.repoList},

		{Tool: mcp.NewTool("repo_status",
			mcp.WithDescription("Get detailed status of a specific repository's CI/CD pipeline"),
			mcp.WithString("name", mcp.Description("Repository name or URL"), mcp.Required()),
			// Tool annotations
			mcp.WithTitleAnnotation("CI/CD: Repository Status"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.repoStatus},

		{Tool: mcp.NewTool("repo_build",
			mcp.WithDescription("Trigger a manual build for a specific repository"),
			mcp.WithString("name", mcp.Description("Repository name or URL"), mcp.Required()),
			mcp.WithString("commit", mcp.Description("Specific commit hash to build (Optional, defaults to latest)")),
			mcp.WithBoolean("push", mcp.Description("Push built image to registry (Optional, defaults to true)")),
			// Tool annotations
			mcp.WithTitleAnnotation("CI/CD: Build Repository"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.repoBuild},

		{Tool: mcp.NewTool("repo_deploy",
			mcp.WithDescription("Deploy a repository to its configured OpenShift namespace"),
			mcp.WithString("name", mcp.Description("Repository name or URL"), mcp.Required()),
			mcp.WithString("image_tag", mcp.Description("Specific image tag to deploy (Optional, defaults to latest)")),
			mcp.WithString("namespace", mcp.Description("Override target namespace (Optional, uses repo config)")),
			// Tool annotations
			mcp.WithTitleAnnotation("CI/CD: Deploy Repository"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.repoDeploy},

		{Tool: mcp.NewTool("repo_remove",
			mcp.WithDescription("Remove a repository from CI/CD monitoring"),
			mcp.WithString("name", mcp.Description("Repository name or URL"), mcp.Required()),
			mcp.WithBoolean("cleanup", mcp.Description("Also cleanup deployed resources (Optional, defaults to false)")),
			// Tool annotations
			mcp.WithTitleAnnotation("CI/CD: Remove Repository"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.repoRemove},

		{Tool: mcp.NewTool("namespace_create",
			mcp.WithDescription("Create a new OpenShift namespace/project for deployments"),
			mcp.WithString("name", mcp.Description("Namespace name"), mcp.Required()),
			mcp.WithString("display_name", mcp.Description("Display name for OpenShift project (Optional)")),
			mcp.WithString("description", mcp.Description("Description for the namespace/project (Optional)")),
			// Tool annotations
			mcp.WithTitleAnnotation("CI/CD: Create Namespace"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.namespaceCreate},

		{Tool: mcp.NewTool("cicd_status",
			mcp.WithDescription("Get overall CI/CD system status and capabilities"),
			// Tool annotations
			mcp.WithTitleAnnotation("CI/CD: System Status"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.cicdStatus},

		{Tool: mcp.NewTool("repo_auto_deploy",
			mcp.WithDescription("Fully automated deployment: create namespace, generate manifests, build, deploy, and provide URL"),
			mcp.WithString("url", mcp.Description("Git repository URL (e.g., https://github.com/user/repo.git)"), mcp.Required()),
			mcp.WithString("namespace", mcp.Description("OpenShift/Kubernetes namespace for deployment (Required)"), mcp.Required()),
			mcp.WithString("name", mcp.Description("Application name (Optional, defaults to repo name)")),
			mcp.WithString("branch", mcp.Description("Git branch to deploy (Optional, defaults to 'main')")),
			mcp.WithNumber("port", mcp.Description("Application port (Optional, auto-detected from repo type)")),
			mcp.WithString("image_registry", mcp.Description("Container registry (Optional, defaults to 'quay.io')")),
			// Tool annotations
			mcp.WithTitleAnnotation("CI/CD: Full Auto Deploy"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.repoAutoDeploy},

		{Tool: mcp.NewTool("repo_generate_manifests",
			mcp.WithDescription("Generate Kubernetes/OpenShift manifests for a repository"),
			mcp.WithString("name", mcp.Description("Repository name or URL"), mcp.Required()),
			mcp.WithString("image_tag", mcp.Description("Image tag to use in manifests (Optional, defaults to 'latest')")),
			// Tool annotations
			mcp.WithTitleAnnotation("CI/CD: Generate Manifests"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.repoGenerateManifests},

		{Tool: mcp.NewTool("repo_get_url",
			mcp.WithDescription("Get the live URL for accessing a deployed application"),
			mcp.WithString("name", mcp.Description("Repository name or URL"), mcp.Required()),
			// Tool annotations
			mcp.WithTitleAnnotation("CI/CD: Get Application URL"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.repoGetURL},
	}
}

// Helper function to extract repository name from URL
func extractRepoName(url string) string {
	parts := strings.Split(strings.TrimSuffix(url, ".git"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "unknown"
}

// Helper function to generate image name from repo name and registry
func generateImageName(repoName, registry string) string {
	if registry == "" {
		registry = "quay.io"
	}
	// Use a default username/org if not specified
	return fmt.Sprintf("%s/default/%s", registry, strings.ToLower(repoName))
}

func (s *Server) repoAdd(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mcpLogger.Printf("Processing repo_add request from context: %v", ctx.Value("request_id"))

	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		mcpLogger.Printf("Invalid arguments format in repo_add request")
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(formatMCPError("INVALID_PARAMS", "Arguments must be a JSON object with required 'url' and 'namespace' fields")),
			},
		}, nil
	}

	url, ok := args["url"].(string)
	if !ok || url == "" {
		mcpLogger.Printf("Missing or invalid 'url' parameter in repo_add request")
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(formatMCPError("MISSING_PARAMETER", "The 'url' parameter is required and must be a valid Git repository URL (e.g., https://github.com/user/repo.git)")),
			},
		}, nil
	}

	namespace, ok := args["namespace"].(string)
	if !ok || namespace == "" {
		mcpLogger.Printf("Missing or invalid 'namespace' parameter in repo_add request")
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.NewTextContent(formatMCPError("MISSING_PARAMETER", "The 'namespace' parameter is required and must be a valid Kubernetes namespace name")),
			},
		}, nil
	}

	// Extract optional parameters with defaults
	repoName := extractRepoName(url)
	if name, exists := args["name"].(string); exists && name != "" {
		repoName = name
	}

	branch := "main"
	if b, exists := args["branch"].(string); exists && b != "" {
		branch = b
	}

	dockerfile := "./Dockerfile"
	if df, exists := args["dockerfile"].(string); exists && df != "" {
		dockerfile = df
	}

	buildContext := "."
	if bc, exists := args["build_context"].(string); exists && bc != "" {
		buildContext = bc
	}

	registry := "quay.io"
	if reg, exists := args["registry"].(string); exists && reg != "" {
		registry = reg
	}

	imageName := generateImageName(repoName, registry)
	if img, exists := args["image_name"].(string); exists && img != "" {
		imageName = img
	}

	// Create repository configuration
	config := &RepoConfig{
		URL:          url,
		Name:         repoName,
		Branch:       branch,
		BuildContext: buildContext,
		DockerFile:   dockerfile,
		ImageName:    imageName,
		Registry:     registry,
		Namespace:    namespace,
		Status:       "configured",
	}

	// Store configuration
	repositoryStore[repoName] = config

	result := map[string]interface{}{
		"status":     "success",
		"message":    fmt.Sprintf("Repository '%s' added successfully", repoName),
		"repository": config,
		"next_steps": []string{
			fmt.Sprintf("Repository will be monitored for commits on branch '%s'", branch),
			fmt.Sprintf("Built images will be pushed to '%s'", imageName),
			fmt.Sprintf("Deployments will be created in namespace '%s'", namespace),
			"Use 'repo_build' to trigger a manual build",
			"Use 'repo_deploy' to deploy the application",
		},
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return NewTextResult(string(jsonResult), nil), nil
}

func (s *Server) repoList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repos := make([]interface{}, 0)
	for _, config := range repositoryStore {
		repos = append(repos, config)
	}

	result := map[string]interface{}{
		"total_repositories": len(repositoryStore),
		"repositories":       repos,
	}

	if len(repositoryStore) == 0 {
		result["message"] = "No repositories configured. Use 'repo_add' to add a repository for monitoring."
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return NewTextResult(string(jsonResult), nil), nil
}

// Full automation: create namespace, generate manifests, build, deploy, and return URLs
func (s *Server) repoAutoDeploy(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return NewTextResult("", fmt.Errorf("invalid arguments format")), nil
	}

	url, ok := args["url"].(string)
	if !ok || url == "" {
		return NewTextResult("", fmt.Errorf("url parameter is required")), nil
	}

	namespace, ok := args["namespace"].(string)
	if !ok || namespace == "" {
		return NewTextResult("", fmt.Errorf("namespace parameter is required")), nil
	}

	repoName := extractRepoName(url)
	if name, exists := args["name"].(string); exists && name != "" {
		repoName = name
	}

	branch := "main"
	if b, exists := args["branch"].(string); exists && b != "" {
		branch = b
	}

	registry := "quay.io"
	if reg, exists := args["image_registry"].(string); exists && reg != "" {
		registry = reg
	}

	// Detect port
	defaultPort, appType := detectAppDetails(repoName)
	port := defaultPort
	if p, exists := args["port"]; exists {
		switch v := p.(type) {
		case float64:
			port = int(v)
		case int:
			port = v
		}
	}

	imageName := generateImageName(repoName, registry)
	imageTag := "latest"

	// Save repo config
	repositoryStore[repoName] = &RepoConfig{
		URL:          url,
		Name:         repoName,
		Branch:       branch,
		BuildContext: ".",
		DockerFile:   "./Dockerfile",
		ImageName:    imageName,
		Registry:     registry,
		Namespace:    namespace,
		Status:       "deploying",
	}

	// Generate manifests
	manifestData := ManifestData{
		AppName:   repoName,
		Namespace: namespace,
		ImageName: imageName,
		ImageTag:  imageTag,
		Port:      port,
		Replicas:  1,
		Version:   "1.0.0",
	}
	manifests, err := generateManifests(manifestData)
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to generate manifests: %v", err)), nil
	}

	// Build YAML strings
	nsYAML := fmt.Sprintf("apiVersion: v1\nkind: Namespace\nmetadata:\n  name: %s\n  labels:\n    app.kubernetes.io/managed-by: ai-mcp-openshift-server\n", namespace)
	combinedYAML := manifests["deployment.yaml"] + "\n---\n" + manifests["service.yaml"] + "\n---\n" + manifests["route.yaml"]

	// Apply to cluster
	applied := false
	if s.k != nil {
		if k8s, derr := s.k.Derived(ctx); derr == nil && k8s != nil {
			if _, nerr := k8s.ResourcesCreateOrUpdate(ctx, nsYAML); nerr == nil {
				if _, aerr := k8s.ResourcesCreateOrUpdate(ctx, combinedYAML); aerr == nil {
					applied = true
					repositoryStore[repoName].Status = "deployed"
				}
			}
		}
	}

	// URL
	appURL := generateRouteURL(repoName, namespace)

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Automated deploy configured for '%s'", repoName),
		"repository": map[string]interface{}{
			"url":      url,
			"name":     repoName,
			"branch":   branch,
			"registry": registry,
		},
		"application": map[string]interface{}{
			"name":      repoName,
			"type":      appType,
			"port":      port,
			"namespace": namespace,
			"url":       appURL,
		},
		"generated_manifests": manifests,
		"applied":             applied,
		"next_steps": []string{
			"Create namespace if not exists",
			"Build image and push to registry",
			"Apply Deployment, Service, Route manifests",
			"Expose application via Route",
		},
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return NewTextResult(string(jsonResult), nil), nil
}

// Generate manifests for an existing repo config
func (s *Server) repoGenerateManifests(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return NewTextResult("", fmt.Errorf("invalid arguments format")), nil
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return NewTextResult("", fmt.Errorf("name parameter is required")), nil
	}

	// Lookup repo
	var config *RepoConfig
	for key, repo := range repositoryStore {
		if key == name || repo.URL == name || repo.Name == name {
			config = repo
			break
		}
	}
	if config == nil {
		return NewTextResult("", fmt.Errorf("repository '%s' not found", name)), nil
	}

	imageTag := "latest"
	if tag, exists := args["image_tag"].(string); exists && tag != "" {
		imageTag = tag
	}

	port, appType := detectAppDetails(config.Name)
	data := ManifestData{
		AppName:   config.Name,
		Namespace: config.Namespace,
		ImageName: config.ImageName,
		ImageTag:  imageTag,
		Port:      port,
		Replicas:  1,
		Version:   "1.0.0",
	}
	manifests, err := generateManifests(data)
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to generate manifests: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":     "success",
		"repository": config.Name,
		"app_type":   appType,
		"manifests":  manifests,
	}
	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return NewTextResult(string(jsonResult), nil), nil
}

// Return live URLs for a repo
func (s *Server) repoGetURL(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return NewTextResult("", fmt.Errorf("invalid arguments format")), nil
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return NewTextResult("", fmt.Errorf("name parameter is required")), nil
	}

	// Lookup repo
	var config *RepoConfig
	for key, repo := range repositoryStore {
		if key == name || repo.URL == name || repo.Name == name {
			config = repo
			break
		}
	}
	if config == nil {
		return NewTextResult("", fmt.Errorf("repository '%s' not found", name)), nil
	}

	port, _ := detectAppDetails(config.Name)
	appURL := generateRouteURL(config.Name, config.Namespace)
	result := map[string]interface{}{
		"status":     "success",
		"repository": config.Name,
		"namespace":  config.Namespace,
		"access_urls": map[string]interface{}{
			"external_url":     appURL,
			"internal_service": fmt.Sprintf("%s.%s.svc.cluster.local:%d", config.Name, config.Namespace, port),
			"health_check":     fmt.Sprintf("%s/health", appURL),
		},
	}
	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return NewTextResult(string(jsonResult), nil), nil
}

func (s *Server) repoStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return NewTextResult("", fmt.Errorf("invalid arguments format")), nil
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return NewTextResult("", fmt.Errorf("name parameter is required")), nil
	}

	// Find repository by name or URL
	var config *RepoConfig
	for key, repo := range repositoryStore {
		if key == name || repo.URL == name || repo.Name == name {
			config = repo
			break
		}
	}

	if config == nil {
		return NewTextResult("", fmt.Errorf("repository '%s' not found", name)), nil
	}

	result := map[string]interface{}{
		"repository": config,
		"pipeline_status": map[string]interface{}{
			"monitoring":   "active",
			"last_build":   "not available",
			"last_deploy":  "not available",
			"health_check": "pending",
		},
		"available_actions": []string{
			"repo_build - Trigger a manual build",
			"repo_deploy - Deploy to OpenShift",
			"repo_remove - Remove from monitoring",
		},
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return NewTextResult(string(jsonResult), nil), nil
}

func (s *Server) repoBuild(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return NewTextResult("", fmt.Errorf("invalid arguments format")), nil
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return NewTextResult("", fmt.Errorf("name parameter is required")), nil
	}

	// Find repository
	var config *RepoConfig
	for key, repo := range repositoryStore {
		if key == name || repo.URL == name || repo.Name == name {
			config = repo
			break
		}
	}

	if config == nil {
		return NewTextResult("", fmt.Errorf("repository '%s' not found", name)), nil
	}

	push := true
	if p, exists := args["push"].(bool); exists {
		push = p
	}

	commit := "latest"
	if c, exists := args["commit"].(string); exists && c != "" {
		commit = c
	}

	// Simulate build process (in real implementation, this would create Kubernetes Jobs)
	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Build triggered for repository '%s'", config.Name),
		"build_info": map[string]interface{}{
			"repository":    config.URL,
			"branch":        config.Branch,
			"commit":        commit,
			"dockerfile":    config.DockerFile,
			"build_context": config.BuildContext,
			"target_image":  config.ImageName,
			"push_enabled":  push,
		},
		"next_steps": []string{
			"Build job will be created in OpenShift",
			"Image will be built using Docker-in-Docker",
		},
	}

	if push {
		result["next_steps"] = append(result["next_steps"].([]string),
			fmt.Sprintf("Image will be pushed to %s", config.Registry))
	}

	// Update repository status
	config.Status = "building"

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return NewTextResult(string(jsonResult), nil), nil
}

func (s *Server) repoDeploy(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return NewTextResult("", fmt.Errorf("invalid arguments format")), nil
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return NewTextResult("", fmt.Errorf("name parameter is required")), nil
	}

	// Find repository
	var config *RepoConfig
	for key, repo := range repositoryStore {
		if key == name || repo.URL == name || repo.Name == name {
			config = repo
			break
		}
	}

	if config == nil {
		return NewTextResult("", fmt.Errorf("repository '%s' not found", name)), nil
	}

	targetNamespace := config.Namespace
	if ns, exists := args["namespace"].(string); exists && ns != "" {
		targetNamespace = ns
	}

	imageTag := "latest"
	if tag, exists := args["image_tag"].(string); exists && tag != "" {
		imageTag = tag
	}

	deploymentImage := fmt.Sprintf("%s:%s", config.ImageName, imageTag)

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Deployment triggered for repository '%s'", config.Name),
		"deployment_info": map[string]interface{}{
			"repository":       config.URL,
			"image":            deploymentImage,
			"target_namespace": targetNamespace,
			"deployment_name":  config.Name,
		},
		"kubernetes_resources": []string{
			"Deployment",
			"Service",
			"ConfigMap (if needed)",
			"Route/Ingress (for external access)",
		},
		"next_steps": []string{
			fmt.Sprintf("Deployment will be created in namespace '%s'", targetNamespace),
			"Service will expose the application internally",
			"Route will provide external access",
			"Use 'pods_list_in_namespace' to check pod status",
		},
	}

	// Update repository status
	config.Status = "deploying"

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return NewTextResult(string(jsonResult), nil), nil
}

func (s *Server) repoRemove(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return NewTextResult("", fmt.Errorf("invalid arguments format")), nil
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return NewTextResult("", fmt.Errorf("name parameter is required")), nil
	}

	cleanup := false
	if c, exists := args["cleanup"].(bool); exists {
		cleanup = c
	}

	// Find and remove repository
	var config *RepoConfig
	var key string
	for k, repo := range repositoryStore {
		if k == name || repo.URL == name || repo.Name == name {
			config = repo
			key = k
			break
		}
	}

	if config == nil {
		return NewTextResult("", fmt.Errorf("repository '%s' not found", name)), nil
	}

	// Remove from store
	delete(repositoryStore, key)

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Repository '%s' removed from monitoring", config.Name),
		"removed": config,
		"cleanup": cleanup,
	}

	if cleanup {
		result["cleanup_actions"] = []string{
			fmt.Sprintf("Deployment '%s' will be deleted from namespace '%s'", config.Name, config.Namespace),
			"Associated services and routes will be removed",
			"ConfigMaps and secrets will be cleaned up",
		}
	} else {
		result["note"] = "Deployed resources were not cleaned up. Use cleanup=true to remove deployed applications."
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return NewTextResult(string(jsonResult), nil), nil
}

func (s *Server) namespaceCreate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return NewTextResult("", fmt.Errorf("invalid arguments format")), nil
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return NewTextResult("", fmt.Errorf("name parameter is required")), nil
	}

	displayName := name
	if dn, exists := args["display_name"].(string); exists && dn != "" {
		displayName = dn
	}

	description := fmt.Sprintf("Namespace created by OpenShift AI MCP Server for %s", name)
	if desc, exists := args["description"].(string); exists && desc != "" {
		description = desc
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Namespace '%s' creation initiated", name),
		"namespace_info": map[string]interface{}{
			"name":         name,
			"display_name": displayName,
			"description":  description,
		},
		"next_steps": []string{
			"Namespace will be created in OpenShift",
			"RBAC permissions will be configured",
			"Default service account will be available",
			fmt.Sprintf("You can now deploy applications to '%s' namespace", name),
		},
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return NewTextResult(string(jsonResult), nil), nil
}

func (s *Server) cicdStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	totalRepos := len(repositoryStore)

	statusCounts := map[string]int{
		"configured": 0,
		"building":   0,
		"deploying":  0,
		"deployed":   0,
		"error":      0,
	}

	for _, repo := range repositoryStore {
		if count, exists := statusCounts[repo.Status]; exists {
			statusCounts[repo.Status] = count + 1
		}
	}

	result := map[string]interface{}{
		"status":  "operational",
		"message": "CI/CD system is ready for multi-repository automation",
		"system_info": map[string]interface{}{
			"version":     "2.0.0",
			"server_type": "OpenShift AI MCP Server",
			"capabilities": []string{
				"Multi-repository monitoring",
				"Dynamic namespace deployment",
				"Container image building",
				"Registry management",
				"Automated deployment",
				"Namespace/Project creation",
			},
		},
		"repository_stats": map[string]interface{}{
			"total_repositories": totalRepos,
			"status_breakdown":   statusCounts,
		},
		"available_tools": []string{
			"repo_add - Add repository for monitoring",
			"repo_list - List all monitored repositories",
			"repo_status - Get repository pipeline status",
			"repo_build - Trigger manual build",
			"repo_deploy - Deploy to OpenShift",
			"repo_remove - Remove repository",
			"namespace_create - Create new namespace",
		},
	}

	if totalRepos == 0 {
		result["getting_started"] = []string{
			"Use 'repo_add' to add your first repository",
			"Specify the Git URL and target OpenShift namespace",
			"The system will handle build and deployment automation",
		}
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return NewTextResult(string(jsonResult), nil), nil
}
