package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/go-git/go-git/v5/plumbing/transport/http"

	"github.com/manusa/kubernetes-mcp-server/pkg/cicd"
	"github.com/manusa/kubernetes-mcp-server/pkg/output"
)

// Initialize CI/CD components for the server
func (s *Server) initCicd() []server.ServerTool {
	return []server.ServerTool{
		// Git Repository Management
		{Tool: mcp.NewTool("git_add_repository",
			mcp.WithDescription("Add a Git repository for monitoring commits. When new commits are detected, it can trigger automated CI/CD pipelines."),
			mcp.WithString("url", mcp.Description("Git repository URL (e.g., https://github.com/user/repo.git)"), mcp.Required()),
			mcp.WithString("branch", mcp.Description("Branch to monitor (default: main)"), mcp.Default("main")),
			mcp.WithString("username", mcp.Description("Git username for authentication (optional)")),
			mcp.WithString("token", mcp.Description("Git personal access token for authentication (optional)")),
			// Tool annotations
			mcp.WithTitleAnnotation("Git: Add Repository"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.gitAddRepository},

		{Tool: mcp.NewTool("git_list_repositories",
			mcp.WithDescription("List all Git repositories currently being monitored for commits"),
			// Tool annotations
			mcp.WithTitleAnnotation("Git: List Repositories"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.gitListRepositories},

		{Tool: mcp.NewTool("git_remove_repository",
			mcp.WithDescription("Remove a Git repository from monitoring"),
			mcp.WithString("url", mcp.Description("Git repository URL"), mcp.Required()),
			mcp.WithString("branch", mcp.Description("Branch being monitored (default: main)"), mcp.Default("main")),
			// Tool annotations
			mcp.WithTitleAnnotation("Git: Remove Repository"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.gitRemoveRepository},

		// Container Image Building
		{Tool: mcp.NewTool("build_image",
			mcp.WithDescription("Build a container image from source code using Docker or OpenShift build strategies"),
			mcp.WithString("name", mcp.Description("Build/image name"), mcp.Required()),
			mcp.WithString("source_repo", mcp.Description("Source Git repository URL"), mcp.Required()),
			mcp.WithString("source_branch", mcp.Description("Source branch (default: main)"), mcp.Default("main")),
			mcp.WithString("image_name", mcp.Description("Target image name (e.g., myapp)"), mcp.Required()),
			mcp.WithString("image_tag", mcp.Description("Target image tag (default: latest)"), mcp.Default("latest")),
			mcp.WithString("dockerfile", mcp.Description("Dockerfile path (default: Dockerfile)"), mcp.Default("Dockerfile")),
			mcp.WithString("context_path", mcp.Description("Build context path (default: .)"), mcp.Default(".")),
			mcp.WithString("namespace", mcp.Description("Kubernetes namespace for OpenShift builds (optional)")),
			mcp.WithString("strategy", mcp.Description("Build strategy: 'docker' or 'openshift' (default: docker)"), mcp.Default("docker")),
			mcp.WithObject("build_args", mcp.Description("Build arguments as key-value pairs (optional)")),
			mcp.WithObject("labels", mcp.Description("Image labels as key-value pairs (optional)")),
			// Tool annotations
			mcp.WithTitleAnnotation("CI/CD: Build Image"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.buildImage},

		{Tool: mcp.NewTool("list_images",
			mcp.WithDescription("List built container images in the specified namespace or Docker daemon"),
			mcp.WithString("namespace", mcp.Description("Kubernetes namespace (for OpenShift images, optional)")),
			// Tool annotations
			mcp.WithTitleAnnotation("CI/CD: List Images"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.listImages},

		// Container Registry Management
		{Tool: mcp.NewTool("registry_add",
			mcp.WithDescription("Add container registry configuration for pushing images"),
			mcp.WithString("name", mcp.Description("Registry configuration name"), mcp.Required()),
			mcp.WithString("url", mcp.Description("Registry URL (e.g., quay.io, docker.io)"), mcp.Required()),
			mcp.WithString("username", mcp.Description("Registry username"), mcp.Required()),
			mcp.WithString("password", mcp.Description("Registry password or token"), mcp.Required()),
			mcp.WithString("email", mcp.Description("Registry email (optional)")),
			mcp.WithBoolean("secure", mcp.Description("Use HTTPS (default: true)"), mcp.Default(true)),
			// Tool annotations
			mcp.WithTitleAnnotation("Registry: Add Configuration"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.registryAdd},

		{Tool: mcp.NewTool("registry_push",
			mcp.WithDescription("Push a container image to a registry"),
			mcp.WithString("source_image", mcp.Description("Source image name and tag"), mcp.Required()),
			mcp.WithString("target_image", mcp.Description("Target image name in registry"), mcp.Required()),
			mcp.WithString("target_tag", mcp.Description("Target image tag (default: latest)"), mcp.Default("latest")),
			mcp.WithString("registry", mcp.Description("Registry configuration name"), mcp.Required()),
			mcp.WithString("namespace", mcp.Description("Kubernetes namespace (for OpenShift, optional)")),
			mcp.WithObject("labels", mcp.Description("Additional image labels (optional)")),
			mcp.WithObject("annotations", mcp.Description("Additional annotations (optional)")),
			// Tool annotations
			mcp.WithTitleAnnotation("Registry: Push Image"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.registryPush},

		{Tool: mcp.NewTool("registry_list",
			mcp.WithDescription("List configured container registries"),
			// Tool annotations
			mcp.WithTitleAnnotation("Registry: List Configurations"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.registryList},

		// Application Deployment
		{Tool: mcp.NewTool("deploy_application",
			mcp.WithDescription("Deploy an application to OpenShift/Kubernetes from a container image"),
			mcp.WithString("name", mcp.Description("Application deployment name"), mcp.Required()),
			mcp.WithString("image", mcp.Description("Container image name"), mcp.Required()),
			mcp.WithString("tag", mcp.Description("Container image tag (default: latest)"), mcp.Default("latest")),
			mcp.WithString("namespace", mcp.Description("Kubernetes namespace"), mcp.Required()),
			mcp.WithInteger("replicas", mcp.Description("Number of replicas (default: 1)"), mcp.Default(1)),
			mcp.WithInteger("port", mcp.Description("Application port (default: 8080)"), mcp.Default(8080)),
			mcp.WithString("service_type", mcp.Description("Service type: ClusterIP, NodePort, LoadBalancer (default: ClusterIP)"), mcp.Default("ClusterIP")),
			mcp.WithBoolean("expose_route", mcp.Description("Create OpenShift route (default: true)"), mcp.Default(true)),
			mcp.WithString("route_domain", mcp.Description("Custom route domain (optional)")),
			mcp.WithObject("env_vars", mcp.Description("Environment variables as key-value pairs (optional)")),
			mcp.WithObject("labels", mcp.Description("Labels as key-value pairs (optional)")),
			mcp.WithObject("resources", mcp.Description("Resource requests and limits (optional)")),
			// Tool annotations
			mcp.WithTitleAnnotation("Deploy: Application"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.deployApplication},

		{Tool: mcp.NewTool("list_applications",
			mcp.WithDescription("List deployed applications in a namespace"),
			mcp.WithString("namespace", mcp.Description("Kubernetes namespace"), mcp.Required()),
			// Tool annotations
			mcp.WithTitleAnnotation("Deploy: List Applications"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.listApplications},

		{Tool: mcp.NewTool("delete_application",
			mcp.WithDescription("Delete a deployed application including deployment, service, and route"),
			mcp.WithString("name", mcp.Description("Application name"), mcp.Required()),
			mcp.WithString("namespace", mcp.Description("Kubernetes namespace"), mcp.Required()),
			// Tool annotations
			mcp.WithTitleAnnotation("Deploy: Delete Application"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.deleteApplication},

		// CI/CD Pipeline Management
		{Tool: mcp.NewTool("cicd_create_pipeline",
			mcp.WithDescription("Create a complete CI/CD pipeline that watches a Git repo, builds images, and deploys applications automatically"),
			mcp.WithString("name", mcp.Description("Pipeline name"), mcp.Required()),
			mcp.WithString("git_url", mcp.Description("Git repository URL"), mcp.Required()),
			mcp.WithString("git_branch", mcp.Description("Git branch to monitor (default: main)"), mcp.Default("main")),
			mcp.WithString("image_name", mcp.Description("Target image name"), mcp.Required()),
			mcp.WithString("registry", mcp.Description("Target registry configuration name"), mcp.Required()),
			mcp.WithString("deploy_namespace", mcp.Description("Deployment namespace"), mcp.Required()),
			mcp.WithString("dockerfile", mcp.Description("Dockerfile path (default: Dockerfile)"), mcp.Default("Dockerfile")),
			mcp.WithObject("build_args", mcp.Description("Build arguments (optional)")),
			mcp.WithObject("env_vars", mcp.Description("Deployment environment variables (optional)")),
			mcp.WithString("git_username", mcp.Description("Git username (optional)")),
			mcp.WithString("git_token", mcp.Description("Git token (optional)")),
			// Tool annotations
			mcp.WithTitleAnnotation("CI/CD: Create Pipeline"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.cicdCreatePipeline},

		{Tool: mcp.NewTool("cicd_list_pipelines",
			mcp.WithDescription("List all configured CI/CD pipelines"),
			// Tool annotations
			mcp.WithTitleAnnotation("CI/CD: List Pipelines"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.cicdListPipelines},

		{Tool: mcp.NewTool("cicd_trigger_pipeline",
			mcp.WithDescription("Manually trigger a CI/CD pipeline execution"),
			mcp.WithString("name", mcp.Description("Pipeline name"), mcp.Required()),
			// Tool annotations
			mcp.WithTitleAnnotation("CI/CD: Trigger Pipeline"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.cicdTriggerPipeline},
	}
}

// CI/CD manager to hold all components
type CicdManager struct {
	gitWatcher           *cicd.GitWatcher
	imageBuilder         *cicd.ImageBuilder
	registryPusher       *cicd.RegistryPusher
	deploymentAutomation *cicd.DeploymentAutomation
	pipelines            map[string]*Pipeline
}

type Pipeline struct {
	Name            string                     `json:"name"`
	GitURL          string                     `json:"git_url"`
	GitBranch       string                     `json:"git_branch"`
	ImageName       string                     `json:"image_name"`
	Registry        string                     `json:"registry"`
	DeployNamespace string                     `json:"deploy_namespace"`
	Dockerfile      string                     `json:"dockerfile"`
	BuildArgs       map[string]string          `json:"build_args,omitempty"`
	EnvVars         map[string]string          `json:"env_vars,omitempty"`
	GitCredentials  *http.BasicAuth            `json:"-"`
	Active          bool                       `json:"active"`
	LastExecution   *time.Time                 `json:"last_execution,omitempty"`
	LastStatus      string                     `json:"last_status,omitempty"`
}

// Initialize CI/CD manager
func (s *Server) initCicdManager() error {
	if s.cicdManager != nil {
		return nil // Already initialized
	}

	// Initialize components
	gitWatcher := cicd.NewGitWatcher(30 * time.Second)
	
	imageBuilder, err := cicd.NewImageBuilder(s.k.GetConfig(), "default")
	if err != nil {
		return fmt.Errorf("failed to initialize image builder: %w", err)
	}

	registryPusher, err := cicd.NewRegistryPusher(s.k.GetConfig())
	if err != nil {
		return fmt.Errorf("failed to initialize registry pusher: %w", err)
	}

	deploymentAutomation, err := cicd.NewDeploymentAutomation(s.k.GetConfig())
	if err != nil {
		return fmt.Errorf("failed to initialize deployment automation: %w", err)
	}

	s.cicdManager = &CicdManager{
		gitWatcher:           gitWatcher,
		imageBuilder:         imageBuilder,
		registryPusher:       registryPusher,
		deploymentAutomation: deploymentAutomation,
		pipelines:            make(map[string]*Pipeline),
	}

	// Set up CI/CD pipeline callback
	s.cicdManager.gitWatcher.AddCommitCallback(s.handleCommitEvent)

	// Start Git watcher
	go s.cicdManager.gitWatcher.StartPolling(context.Background())

	return nil
}

// Handle commit events for CI/CD pipelines
func (s *Server) handleCommitEvent(event cicd.CommitEvent) error {
	fmt.Printf("Commit detected: %s in %s:%s\n", event.CommitHash, event.RepoURL, event.Branch)
	
	// Find matching pipelines
	for _, pipeline := range s.cicdManager.pipelines {
		if pipeline.GitURL == event.RepoURL && pipeline.GitBranch == event.Branch && pipeline.Active {
			fmt.Printf("Triggering pipeline %s for commit %s\n", pipeline.Name, event.CommitHash)
			go s.executePipeline(pipeline, event.CommitHash)
		}
	}
	
	return nil
}

// Execute a CI/CD pipeline
func (s *Server) executePipeline(pipeline *Pipeline, commitHash string) error {
	now := time.Now()
	pipeline.LastExecution = &now
	pipeline.LastStatus = "running"

	defer func() {
		if r := recover(); r != nil {
			pipeline.LastStatus = "failed"
			fmt.Printf("Pipeline %s failed with panic: %v\n", pipeline.Name, r)
		}
	}()

	ctx := context.Background()

	// Step 1: Build image
	buildConfig := cicd.BuildConfig{
		Name:         pipeline.Name,
		SourceRepo:   pipeline.GitURL,
		SourceBranch: pipeline.GitBranch,
		ImageName:    pipeline.ImageName,
		ImageTag:     commitHash[:8], // Use short commit hash as tag
		Dockerfile:   pipeline.Dockerfile,
		BuildArgs:    pipeline.BuildArgs,
		BuildStrategy: "docker",
	}

	fmt.Printf("Building image for pipeline %s...\n", pipeline.Name)
	buildResult, err := s.cicdManager.imageBuilder.BuildImage(ctx, buildConfig)
	if err != nil || !buildResult.Success {
		pipeline.LastStatus = "build_failed"
		return fmt.Errorf("build failed: %w", err)
	}

	// Step 2: Push image
	pushConfig := cicd.PushConfig{
		SourceImage: buildResult.FullImageName,
		TargetImage: pipeline.ImageName,
		TargetTag:   commitHash[:8],
		Registry:    pipeline.Registry,
	}

	fmt.Printf("Pushing image for pipeline %s...\n", pipeline.Name)
	pushResult, err := s.cicdManager.registryPusher.PushImage(ctx, pushConfig)
	if err != nil || !pushResult.Success {
		pipeline.LastStatus = "push_failed"
		return fmt.Errorf("push failed: %w", err)
	}

	// Step 3: Deploy application
	deployConfig := cicd.DeploymentConfig{
		Name:      pipeline.Name,
		Namespace: pipeline.DeployNamespace,
		Image:     pipeline.ImageName,
		Tag:       commitHash[:8],
		EnvVars:   pipeline.EnvVars,
		ExposeRoute: true,
	}

	fmt.Printf("Deploying application for pipeline %s...\n", pipeline.Name)
	deployResult, err := s.cicdManager.deploymentAutomation.DeployApplication(ctx, deployConfig)
	if err != nil || !deployResult.Success {
		pipeline.LastStatus = "deploy_failed"
		return fmt.Errorf("deploy failed: %w", err)
	}

	pipeline.LastStatus = "success"
	fmt.Printf("Pipeline %s completed successfully\n", pipeline.Name)
	return nil
}

// MCP Tool Handlers

func (s *Server) gitAddRepository(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.initCicdManager(); err != nil {
		return mcp.NewToolResultError("failed to initialize CI/CD manager", err), nil
	}

	var args struct {
		URL      string `json:"url"`
		Branch   string `json:"branch"`
		Username string `json:"username"`
		Token    string `json:"token"`
	}

	if err := mcp.UnmarshalArguments(request.Params.Arguments, &args); err != nil {
		return mcp.NewToolResultError("invalid arguments", err), nil
	}

	if args.Branch == "" {
		args.Branch = "main"
	}

	var credentials *http.BasicAuth
	if args.Username != "" && args.Token != "" {
		credentials = &http.BasicAuth{
			Username: args.Username,
			Password: args.Token,
		}
	}

	err := s.cicdManager.gitWatcher.AddRepository(args.URL, args.Branch, credentials)
	if err != nil {
		return mcp.NewToolResultError("failed to add repository", err), nil
	}

	result := map[string]interface{}{
		"status":     "success",
		"message":    fmt.Sprintf("Repository %s (branch: %s) added for monitoring", args.URL, args.Branch),
		"repository": args.URL,
		"branch":     args.Branch,
	}

	return mcp.NewToolResultText(output.FormatOutput(result, s.configuration.ListOutput)), nil
}

func (s *Server) gitListRepositories(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.initCicdManager(); err != nil {
		return mcp.NewToolResultError("failed to initialize CI/CD manager", err), nil
	}

	repositories := s.cicdManager.gitWatcher.GetRepositories()
	
	var repoList []map[string]interface{}
	for key, repo := range repositories {
		repoList = append(repoList, map[string]interface{}{
			"key":         key,
			"url":         repo.URL,
			"branch":      repo.Branch,
			"last_commit": repo.LastCommit,
		})
	}

	result := map[string]interface{}{
		"repositories": repoList,
		"count":        len(repoList),
	}

	return mcp.NewToolResultText(output.FormatOutput(result, s.configuration.ListOutput)), nil
}

func (s *Server) gitRemoveRepository(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.initCicdManager(); err != nil {
		return mcp.NewToolResultError("failed to initialize CI/CD manager", err), nil
	}

	var args struct {
		URL    string `json:"url"`
		Branch string `json:"branch"`
	}

	if err := mcp.UnmarshalArguments(request.Params.Arguments, &args); err != nil {
		return mcp.NewToolResultError("invalid arguments", err), nil
	}

	if args.Branch == "" {
		args.Branch = "main"
	}

	s.cicdManager.gitWatcher.RemoveRepository(args.URL, args.Branch)

	result := map[string]interface{}{
		"status":     "success",
		"message":    fmt.Sprintf("Repository %s (branch: %s) removed from monitoring", args.URL, args.Branch),
		"repository": args.URL,
		"branch":     args.Branch,
	}

	return mcp.NewToolResultText(output.FormatOutput(result, s.configuration.ListOutput)), nil
}

func (s *Server) buildImage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.initCicdManager(); err != nil {
		return mcp.NewToolResultError("failed to initialize CI/CD manager", err), nil
	}

	var args struct {
		Name         string            `json:"name"`
		SourceRepo   string            `json:"source_repo"`
		SourceBranch string            `json:"source_branch"`
		ImageName    string            `json:"image_name"`
		ImageTag     string            `json:"image_tag"`
		Dockerfile   string            `json:"dockerfile"`
		ContextPath  string            `json:"context_path"`
		Namespace    string            `json:"namespace"`
		Strategy     string            `json:"strategy"`
		BuildArgs    map[string]string `json:"build_args"`
		Labels       map[string]string `json:"labels"`
	}

	if err := mcp.UnmarshalArguments(request.Params.Arguments, &args); err != nil {
		return mcp.NewToolResultError("invalid arguments", err), nil
	}

	// Apply defaults
	if args.SourceBranch == "" {
		args.SourceBranch = "main"
	}
	if args.ImageTag == "" {
		args.ImageTag = "latest"
	}
	if args.Dockerfile == "" {
		args.Dockerfile = "Dockerfile"
	}
	if args.ContextPath == "" {
		args.ContextPath = "."
	}
	if args.Strategy == "" {
		args.Strategy = "docker"
	}

	buildConfig := cicd.BuildConfig{
		Name:          args.Name,
		Namespace:     args.Namespace,
		SourceRepo:    args.SourceRepo,
		SourceBranch:  args.SourceBranch,
		Dockerfile:    args.Dockerfile,
		ContextPath:   args.ContextPath,
		ImageName:     args.ImageName,
		ImageTag:      args.ImageTag,
		BuildArgs:     args.BuildArgs,
		Labels:        args.Labels,
		BuildStrategy: args.Strategy,
	}

	buildResult, err := s.cicdManager.imageBuilder.BuildImage(ctx, buildConfig)
	if err != nil {
		return mcp.NewToolResultError("build failed", err), nil
	}

	if !buildResult.Success {
		return mcp.NewToolResultError("build failed", buildResult.Error), nil
	}

	result := map[string]interface{}{
		"status":         "success",
		"image_name":     buildResult.ImageName,
		"image_tag":      buildResult.ImageTag,
		"full_image":     buildResult.FullImageName,
		"build_time_ms":  buildResult.BuildTime.Milliseconds(),
		"build_logs":     buildResult.BuildLogs,
	}

	return mcp.NewToolResultText(output.FormatOutput(result, s.configuration.ListOutput)), nil
}

func (s *Server) listImages(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.initCicdManager(); err != nil {
		return mcp.NewToolResultError("failed to initialize CI/CD manager", err), nil
	}

	var args struct {
		Namespace string `json:"namespace"`
	}

	if err := mcp.UnmarshalArguments(request.Params.Arguments, &args); err != nil {
		return mcp.NewToolResultError("invalid arguments", err), nil
	}

	images, err := s.cicdManager.imageBuilder.ListImages(ctx, args.Namespace)
	if err != nil {
		return mcp.NewToolResultError("failed to list images", err), nil
	}

	result := map[string]interface{}{
		"images": images,
		"count":  len(images),
	}

	return mcp.NewToolResultText(output.FormatOutput(result, s.configuration.ListOutput)), nil
}

func (s *Server) registryAdd(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.initCicdManager(); err != nil {
		return mcp.NewToolResultError("failed to initialize CI/CD manager", err), nil
	}

	var args struct {
		Name     string `json:"name"`
		URL      string `json:"url"`
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
		Secure   bool   `json:"secure"`
	}

	if err := mcp.UnmarshalArguments(request.Params.Arguments, &args); err != nil {
		return mcp.NewToolResultError("invalid arguments", err), nil
	}

	registryConfig := &cicd.RegistryConfig{
		URL:      args.URL,
		Username: args.Username,
		Password: args.Password,
		Email:    args.Email,
		Secure:   args.Secure,
	}

	s.cicdManager.registryPusher.AddRegistry(args.Name, registryConfig)

	result := map[string]interface{}{
		"status":   "success",
		"message":  fmt.Sprintf("Registry configuration '%s' added for %s", args.Name, args.URL),
		"name":     args.Name,
		"url":      args.URL,
		"username": args.Username,
	}

	return mcp.NewToolResultText(output.FormatOutput(result, s.configuration.ListOutput)), nil
}

func (s *Server) registryPush(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.initCicdManager(); err != nil {
		return mcp.NewToolResultError("failed to initialize CI/CD manager", err), nil
	}

	var args struct {
		SourceImage string            `json:"source_image"`
		TargetImage string            `json:"target_image"`
		TargetTag   string            `json:"target_tag"`
		Registry    string            `json:"registry"`
		Namespace   string            `json:"namespace"`
		Labels      map[string]string `json:"labels"`
		Annotations map[string]string `json:"annotations"`
	}

	if err := mcp.UnmarshalArguments(request.Params.Arguments, &args); err != nil {
		return mcp.NewToolResultError("invalid arguments", err), nil
	}

	if args.TargetTag == "" {
		args.TargetTag = "latest"
	}

	pushConfig := cicd.PushConfig{
		SourceImage: args.SourceImage,
		TargetImage: args.TargetImage,
		TargetTag:   args.TargetTag,
		Registry:    args.Registry,
		Namespace:   args.Namespace,
		Labels:      args.Labels,
		Annotations: args.Annotations,
	}

	pushResult, err := s.cicdManager.registryPusher.PushImage(ctx, pushConfig)
	if err != nil {
		return mcp.NewToolResultError("push failed", err), nil
	}

	if !pushResult.Success {
		return mcp.NewToolResultError("push failed", pushResult.Error), nil
	}

	result := map[string]interface{}{
		"status":         "success",
		"source_image":   pushResult.SourceImage,
		"target_image":   pushResult.TargetImage,
		"full_image":     pushResult.FullImageName,
		"push_time_ms":   pushResult.PushTime.Milliseconds(),
		"digest":         pushResult.Digest,
	}

	return mcp.NewToolResultText(output.FormatOutput(result, s.configuration.ListOutput)), nil
}

func (s *Server) registryList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.initCicdManager(); err != nil {
		return mcp.NewToolResultError("failed to initialize CI/CD manager", err), nil
	}

	registries := s.cicdManager.registryPusher.GetRegistries()

	result := map[string]interface{}{
		"registries": registries,
		"count":      len(registries),
	}

	return mcp.NewToolResultText(output.FormatOutput(result, s.configuration.ListOutput)), nil
}

func (s *Server) deployApplication(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.initCicdManager(); err != nil {
		return mcp.NewToolResultError("failed to initialize CI/CD manager", err), nil
	}

	var args struct {
		Name        string            `json:"name"`
		Image       string            `json:"image"`
		Tag         string            `json:"tag"`
		Namespace   string            `json:"namespace"`
		Replicas    int32             `json:"replicas"`
		Port        int32             `json:"port"`
		ServiceType string            `json:"service_type"`
		ExposeRoute bool              `json:"expose_route"`
		RouteDomain string            `json:"route_domain"`
		EnvVars     map[string]string `json:"env_vars"`
		Labels      map[string]string `json:"labels"`
		Resources   struct {
			Requests map[string]string `json:"requests"`
			Limits   map[string]string `json:"limits"`
		} `json:"resources"`
	}

	if err := mcp.UnmarshalArguments(request.Params.Arguments, &args); err != nil {
		return mcp.NewToolResultError("invalid arguments", err), nil
	}

	// Apply defaults
	if args.Tag == "" {
		args.Tag = "latest"
	}
	if args.Replicas == 0 {
		args.Replicas = 1
	}
	if args.Port == 0 {
		args.Port = 8080
	}
	if args.ServiceType == "" {
		args.ServiceType = "ClusterIP"
	}

	var resources *cicd.ResourceRequirements
	if len(args.Resources.Requests) > 0 || len(args.Resources.Limits) > 0 {
		resources = &cicd.ResourceRequirements{
			Requests: args.Resources.Requests,
			Limits:   args.Resources.Limits,
		}
	}

	deployConfig := cicd.DeploymentConfig{
		Name:        args.Name,
		Namespace:   args.Namespace,
		Image:       args.Image,
		Tag:         args.Tag,
		Replicas:    args.Replicas,
		Port:        args.Port,
		ServiceType: args.ServiceType,
		Labels:      args.Labels,
		EnvVars:     args.EnvVars,
		Resources:   resources,
		ExposeRoute: args.ExposeRoute,
		RouteDomain: args.RouteDomain,
	}

	deployResult, err := s.cicdManager.deploymentAutomation.DeployApplication(ctx, deployConfig)
	if err != nil {
		return mcp.NewToolResultError("deployment failed", err), nil
	}

	if !deployResult.Success {
		return mcp.NewToolResultError("deployment failed", deployResult.Error), nil
	}

	result := map[string]interface{}{
		"status":         "success",
		"name":           deployResult.Name,
		"namespace":      deployResult.Namespace,
		"image":          deployResult.Image,
		"replicas":       deployResult.Replicas,
		"service_name":   deployResult.ServiceName,
		"route_url":      deployResult.RouteURL,
		"deploy_time_ms": deployResult.DeployTime.Milliseconds(),
		"logs":           deployResult.Logs,
	}

	return mcp.NewToolResultText(output.FormatOutput(result, s.configuration.ListOutput)), nil
}

func (s *Server) listApplications(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.initCicdManager(); err != nil {
		return mcp.NewToolResultError("failed to initialize CI/CD manager", err), nil
	}

	var args struct {
		Namespace string `json:"namespace"`
	}

	if err := mcp.UnmarshalArguments(request.Params.Arguments, &args); err != nil {
		return mcp.NewToolResultError("invalid arguments", err), nil
	}

	applications, err := s.cicdManager.deploymentAutomation.ListApplications(ctx, args.Namespace)
	if err != nil {
		return mcp.NewToolResultError("failed to list applications", err), nil
	}

	result := map[string]interface{}{
		"applications": applications,
		"count":        len(applications),
		"namespace":    args.Namespace,
	}

	return mcp.NewToolResultText(output.FormatOutput(result, s.configuration.ListOutput)), nil
}

func (s *Server) deleteApplication(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.initCicdManager(); err != nil {
		return mcp.NewToolResultError("failed to initialize CI/CD manager", err), nil
	}

	var args struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	}

	if err := mcp.UnmarshalArguments(request.Params.Arguments, &args); err != nil {
		return mcp.NewToolResultError("invalid arguments", err), nil
	}

	err := s.cicdManager.deploymentAutomation.DeleteApplication(ctx, args.Namespace, args.Name)
	if err != nil {
		return mcp.NewToolResultError("deletion failed", err), nil
	}

	result := map[string]interface{}{
		"status":    "success",
		"message":   fmt.Sprintf("Application %s deleted from namespace %s", args.Name, args.Namespace),
		"name":      args.Name,
		"namespace": args.Namespace,
	}

	return mcp.NewToolResultText(output.FormatOutput(result, s.configuration.ListOutput)), nil
}

func (s *Server) cicdCreatePipeline(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.initCicdManager(); err != nil {
		return mcp.NewToolResultError("failed to initialize CI/CD manager", err), nil
	}

	var args struct {
		Name            string            `json:"name"`
		GitURL          string            `json:"git_url"`
		GitBranch       string            `json:"git_branch"`
		ImageName       string            `json:"image_name"`
		Registry        string            `json:"registry"`
		DeployNamespace string            `json:"deploy_namespace"`
		Dockerfile      string            `json:"dockerfile"`
		BuildArgs       map[string]string `json:"build_args"`
		EnvVars         map[string]string `json:"env_vars"`
		GitUsername     string            `json:"git_username"`
		GitToken        string            `json:"git_token"`
	}

	if err := mcp.UnmarshalArguments(request.Params.Arguments, &args); err != nil {
		return mcp.NewToolResultError("invalid arguments", err), nil
	}

	// Apply defaults
	if args.GitBranch == "" {
		args.GitBranch = "main"
	}
	if args.Dockerfile == "" {
		args.Dockerfile = "Dockerfile"
	}

	var gitCredentials *http.BasicAuth
	if args.GitUsername != "" && args.GitToken != "" {
		gitCredentials = &http.BasicAuth{
			Username: args.GitUsername,
			Password: args.GitToken,
		}
	}

	pipeline := &Pipeline{
		Name:            args.Name,
		GitURL:          args.GitURL,
		GitBranch:       args.GitBranch,
		ImageName:       args.ImageName,
		Registry:        args.Registry,
		DeployNamespace: args.DeployNamespace,
		Dockerfile:      args.Dockerfile,
		BuildArgs:       args.BuildArgs,
		EnvVars:         args.EnvVars,
		GitCredentials:  gitCredentials,
		Active:          true,
		LastStatus:      "created",
	}

	// Add pipeline to manager
	s.cicdManager.pipelines[args.Name] = pipeline

	// Add Git repository for monitoring
	err := s.cicdManager.gitWatcher.AddRepository(args.GitURL, args.GitBranch, gitCredentials)
	if err != nil {
		return mcp.NewToolResultError("failed to add git repository", err), nil
	}

	result := map[string]interface{}{
		"status":   "success",
		"message":  fmt.Sprintf("CI/CD pipeline '%s' created successfully", args.Name),
		"pipeline": pipeline,
	}

	return mcp.NewToolResultText(output.FormatOutput(result, s.configuration.ListOutput)), nil
}

func (s *Server) cicdListPipelines(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.initCicdManager(); err != nil {
		return mcp.NewToolResultError("failed to initialize CI/CD manager", err), nil
	}

	var pipelines []interface{}
	for _, pipeline := range s.cicdManager.pipelines {
		pipelines = append(pipelines, pipeline)
	}

	result := map[string]interface{}{
		"pipelines": pipelines,
		"count":     len(pipelines),
	}

	return mcp.NewToolResultText(output.FormatOutput(result, s.configuration.ListOutput)), nil
}

func (s *Server) cicdTriggerPipeline(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.initCicdManager(); err != nil {
		return mcp.NewToolResultError("failed to initialize CI/CD manager", err), nil
	}

	var args struct {
		Name string `json:"name"`
	}

	if err := mcp.UnmarshalArguments(request.Params.Arguments, &args); err != nil {
		return mcp.NewToolResultError("invalid arguments", err), nil
	}

	pipeline, exists := s.cicdManager.pipelines[args.Name]
	if !exists {
		return mcp.NewToolResultError("pipeline not found", nil), nil
	}

	// Trigger pipeline execution with latest commit
	go s.executePipeline(pipeline, "manual-trigger")

	result := map[string]interface{}{
		"status":   "success",
		"message":  fmt.Sprintf("Pipeline '%s' triggered successfully", args.Name),
		"pipeline": pipeline.Name,
	}

	return mcp.NewToolResultText(output.FormatOutput(result, s.configuration.ListOutput)), nil
}
