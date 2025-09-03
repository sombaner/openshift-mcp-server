package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"k8s.io/klog/v2"
)

// ContainerBuildConfig represents configuration for building a container
type ContainerBuildConfig struct {
	SourceType    string `json:"source_type"`    // "git", "local", "url"
	Source        string `json:"source"`         // Source location
	Dockerfile    string `json:"dockerfile"`     // Path to Dockerfile
	BuildContext  string `json:"build_context"`  // Build context directory
	ImageName     string `json:"image_name"`     // Target image name
	Registry      string `json:"registry"`       // Container registry
	Tags          []string `json:"tags"`         // Image tags
	BuildArgs     map[string]string `json:"build_args"` // Build arguments
	Platform      string `json:"platform"`      // Target platform
}

// ContainerImageInfo represents information about a built container image
type ContainerImageInfo struct {
	ImageName     string            `json:"image_name"`
	Tags          []string          `json:"tags"`
	Size          string            `json:"size"`
	CreatedAt     time.Time         `json:"created_at"`
	Registry      string            `json:"registry"`
	Digest        string            `json:"digest"`
	Labels        map[string]string `json:"labels"`
	BuildDuration string            `json:"build_duration"`
}

// initContainers initializes container-related MCP tools
func (s *Server) initContainers() []server.ServerTool {
	klog.V(1).Info("Initializing container build and registry tools")

	return []server.ServerTool{
		{Tool: mcp.NewTool("container_build",
			mcp.WithDescription("Build a container image from source code with Red Hat UBI compliance validation. Supports Git repositories, local directories, and remote archives. Automatically validates Dockerfile for Red Hat UBI base images and security best practices."),
			mcp.WithString("source", mcp.Description("Source location for the container build. Can be a Git repository URL (https://github.com/user/repo.git), local directory path (/path/to/source), or remote archive URL."), mcp.Required()),
			mcp.WithString("source_type", mcp.Description("Type of source: 'git' for Git repositories, 'local' for local directories, 'url' for remote archives. Auto-detected if not specified.")),
			mcp.WithString("image_name", mcp.Description("Target container image name. Should include registry if pushing later. Examples: 'my-app:latest', 'quay.io/user/app:v1.0'."), mcp.Required()),
			mcp.WithString("dockerfile", mcp.Description("Path to Dockerfile relative to build context. Defaults to 'Dockerfile'.")),
			mcp.WithString("build_context", mcp.Description("Build context directory relative to source root. Defaults to '.' (source root).")),
			mcp.WithString("registry", mcp.Description("Target container registry. Examples: 'quay.io', 'docker.io', 'ghcr.io'.")),
			mcp.WithString("tags", mcp.Description("Comma-separated list of additional tags. Example: 'latest,v1.0,staging'.")),
			mcp.WithString("platform", mcp.Description("Target platform. Examples: 'linux/amd64', 'linux/arm64'. Defaults to current platform.")),
			mcp.WithString("build_args", mcp.Description("Build arguments as JSON string. Example: '{\"ENV\":\"production\",\"VERSION\":\"1.0\"}'.")),
			mcp.WithString("git_branch", mcp.Description("Git branch to checkout (only for Git sources). Defaults to 'main'.")),
			mcp.WithString("git_commit", mcp.Description("Specific Git commit hash to checkout (only for Git sources).")),
			mcp.WithBoolean("no_cache", mcp.Description("Disable build cache. Defaults to false.")),
			mcp.WithBoolean("pull", mcp.Description("Always pull latest base images during build. Defaults to true.")),
			mcp.WithBoolean("validate_ubi", mcp.Description("Validate Red Hat UBI compliance and suggest alternatives. Defaults to true.")),
			mcp.WithBoolean("generate_ubi_dockerfile", mcp.Description("Generate UBI-compliant Dockerfile if current base image is not UBI. Defaults to false.")),
			mcp.WithBoolean("security_scan", mcp.Description("Perform security validation on Dockerfile. Defaults to true.")),
			// Tool annotations
			mcp.WithTitleAnnotation("Container: Build Image with UBI Validation"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.containerBuild},

		{Tool: mcp.NewTool("container_push",
			mcp.WithDescription("Push a container image to a registry. Supports authentication via environment variables or registry login. Can push single or multiple tags simultaneously. Provides detailed push progress and error handling."),
			mcp.WithString("image_name", mcp.Description("Container image name to push. Should include registry and tag. Examples: 'quay.io/user/app:latest', 'docker.io/company/product:v1.0', 'ghcr.io/org/service:dev'."), mcp.Required()),
			mcp.WithString("registry", mcp.Description("Target container registry. Will be extracted from image_name if not provided. Examples: 'quay.io', 'docker.io', 'ghcr.io', 'localhost:5000'.")),
			mcp.WithString("username", mcp.Description("Registry username for authentication. Can also be provided via REGISTRY_USERNAME environment variable.")),
			mcp.WithString("password", mcp.Description("Registry password/token for authentication. Can also be provided via REGISTRY_PASSWORD environment variable. For security, prefer environment variables.")),
			mcp.WithString("additional_tags", mcp.Description("Comma-separated list of additional tags to push. Example: 'latest,v1.0,stable'. Each tag will be pushed separately.")),
			mcp.WithBoolean("all_tags", mcp.Description("Push all tags of the image. Defaults to false (push only specified tag).")),
			mcp.WithBoolean("skip_tls_verify", mcp.Description("Skip TLS certificate verification. Only use for private registries with self-signed certificates. Defaults to false.")),
			// Tool annotations
			mcp.WithTitleAnnotation("Container: Push Image to Registry"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.containerPush},

		{Tool: mcp.NewTool("container_list",
			mcp.WithDescription("List local container images with detailed information including size, creation date, and tags. Useful for managing local container storage and finding images for deployment."),
			mcp.WithString("filter", mcp.Description("Filter images by name pattern. Examples: 'my-app*', '*:latest', 'quay.io/user/*'. Supports wildcards.")),
			mcp.WithString("registry", mcp.Description("Filter images by registry. Examples: 'quay.io', 'docker.io', 'localhost'. Shows only images from specified registry.")),
			mcp.WithBoolean("show_all", mcp.Description("Show all images including intermediate layers. Defaults to false (show only tagged images).")),
			mcp.WithString("format", mcp.Description("Output format: 'table' (default), 'json', 'compact'. Table format is human-readable, JSON for programmatic use.")),
			// Tool annotations
			mcp.WithTitleAnnotation("Container: List Local Images"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.containerList},

		{Tool: mcp.NewTool("container_remove",
			mcp.WithDescription("Remove local container images to free up disk space. Can remove by name, tag, or image ID. Supports bulk removal with patterns."),
			mcp.WithString("image_name", mcp.Description("Container image name or ID to remove. Examples: 'my-app:latest', 'quay.io/user/app:v1.0', 'sha256:abc123...'. Can use partial IDs."), mcp.Required()),
			mcp.WithBoolean("force", mcp.Description("Force removal of image even if containers are using it. Use with caution. Defaults to false.")),
			mcp.WithBoolean("prune", mcp.Description("Also remove unused parent images. Defaults to false.")),
			// Tool annotations
			mcp.WithTitleAnnotation("Container: Remove Local Image"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.containerRemove},

		{Tool: mcp.NewTool("container_inspect",
			mcp.WithDescription("Inspect a container image to view detailed metadata, configuration, layers, and security information. Useful for debugging and understanding image composition."),
			mcp.WithString("image_name", mcp.Description("Container image name or ID to inspect. Examples: 'my-app:latest', 'quay.io/user/app:v1.0', 'sha256:abc123...'."), mcp.Required()),
			mcp.WithString("format", mcp.Description("Output format: 'full' (default), 'config', 'layers', 'security'. Full shows all info, others show specific sections.")),
			// Tool annotations
			mcp.WithTitleAnnotation("Container: Inspect Image Details"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.containerInspect},

		{Tool: mcp.NewTool("container_pull",
			mcp.WithDescription("Pull a container image from a registry to local storage. Supports authentication via environment variables or registry login. Can pull from Docker Hub, Quay.io, or private registries."),
			mcp.WithString("image_name", mcp.Description("Container image name to pull. Examples: 'nginx:latest', 'quay.io/user/app:v1.0', 'docker.io/library/redis:alpine'. Registry will be auto-detected or default to docker.io."), mcp.Required()),
			mcp.WithString("registry", mcp.Description("Source container registry. Will be extracted from image_name if not provided. Examples: 'quay.io', 'docker.io', 'ghcr.io', 'localhost:5000'.")),
			mcp.WithString("username", mcp.Description("Registry username for authentication. Can also be provided via REGISTRY_USERNAME environment variable.")),
			mcp.WithString("password", mcp.Description("Registry password/token for authentication. Can also be provided via REGISTRY_PASSWORD environment variable. For security, prefer environment variables.")),
			mcp.WithString("platform", mcp.Description("Target platform for multi-arch images. Examples: 'linux/amd64', 'linux/arm64'. Defaults to current platform.")),
			mcp.WithBoolean("skip_tls_verify", mcp.Description("Skip TLS certificate verification. Only use for private registries with self-signed certificates. Defaults to false.")),
			mcp.WithBoolean("all_tags", mcp.Description("Pull all tags of the image. Defaults to false (pull only specified tag).")),
			// Tool annotations
			mcp.WithTitleAnnotation("Container: Pull Image from Registry"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.containerPull},

		{Tool: mcp.NewTool("container_run",
			mcp.WithDescription("Run a container from an image with configurable options including port mappings, environment variables, and volumes. Supports both interactive and detached modes."),
			mcp.WithString("image_name", mcp.Description("Container image name to run. Examples: 'nginx:latest', 'quay.io/user/app:v1.0', 'redis:alpine'."), mcp.Required()),
			mcp.WithString("container_name", mcp.Description("Name to assign to the container. If not provided, a random name will be generated.")),
			mcp.WithArray("ports", mcp.Description("Port mappings to expose on the host. Format: <hostPort>:<containerPort>. Examples: ['8080:80', '8443:443']. Use 'publish_all' to expose all ports."),
				func(schema map[string]interface{}) {
					schema["type"] = "array"
					schema["items"] = map[string]interface{}{
						"type": "string",
					}
				},
			),
			mcp.WithArray("environment", mcp.Description("Environment variables to set in the container. Format: <key>=<value>. Examples: ['ENV=production', 'PORT=8080']."),
				func(schema map[string]interface{}) {
					schema["type"] = "array"
					schema["items"] = map[string]interface{}{
						"type": "string",
					}
				},
			),
			mcp.WithArray("volumes", mcp.Description("Volume mounts for persistent storage. Format: <hostPath>:<containerPath>[:ro]. Examples: ['/data:/app/data', '/config:/etc/config:ro']."),
				func(schema map[string]interface{}) {
					schema["type"] = "array"
					schema["items"] = map[string]interface{}{
						"type": "string",
					}
				},
			),
			mcp.WithString("command", mcp.Description("Override the default command/entrypoint of the container. Example: '/bin/bash -c \"echo hello\"'.")),
			mcp.WithString("working_dir", mcp.Description("Set the working directory inside the container. Example: '/app'.")),
			mcp.WithString("user", mcp.Description("Run container as specific user. Format: 'uid:gid' or 'username'. Example: '1000:1000' or 'app'.")),
			mcp.WithBoolean("detached", mcp.Description("Run container in detached mode (background). Defaults to true.")),
			mcp.WithBoolean("interactive", mcp.Description("Keep STDIN open and allocate pseudo-TTY. Defaults to false.")),
			mcp.WithBoolean("remove", mcp.Description("Automatically remove container when it exits. Defaults to false.")),
			mcp.WithBoolean("publish_all", mcp.Description("Publish all exposed ports to random host ports. Defaults to false.")),
			mcp.WithString("restart", mcp.Description("Restart policy. Options: 'no', 'always', 'unless-stopped', 'on-failure'. Defaults to 'no'.")),
			// Tool annotations
			mcp.WithTitleAnnotation("Container: Run Container from Image"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.containerRun},

		{Tool: mcp.NewTool("container_stop",
			mcp.WithDescription("Stop a running container gracefully. Sends SIGTERM signal first, then SIGKILL after timeout. Can stop by container name or ID."),
			mcp.WithString("container_name", mcp.Description("Container name or ID to stop. Examples: 'my-app', 'wonderful_turing', 'sha256:abc123...'. Can use partial IDs."), mcp.Required()),
			mcp.WithString("timeout", mcp.Description("Time to wait for graceful shutdown before sending SIGKILL. Examples: '30s', '1m', '5'. Defaults to '10s'.")),
			mcp.WithBoolean("force", mcp.Description("Force stop the container immediately with SIGKILL. Defaults to false.")),
			// Tool annotations
			mcp.WithTitleAnnotation("Container: Stop Running Container"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.containerStop},
	}
}

// containerBuild handles building container images from source code
func (s *Server) containerBuild(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return NewTextResult("", fmt.Errorf("invalid arguments format")), nil
	}

	// Extract required parameters
	source, ok := args["source"].(string)
	if !ok || source == "" {
		return NewTextResult("", fmt.Errorf("source parameter is required")), nil
	}

	imageName, ok := args["image_name"].(string)
	if !ok || imageName == "" {
		return NewTextResult("", fmt.Errorf("image_name parameter is required")), nil
	}

	// Extract optional parameters with defaults
	sourceType := getStringArg(args, "source_type", detectSourceType(source))
	dockerfile := getStringArg(args, "dockerfile", "Dockerfile")
	buildContext := getStringArg(args, "build_context", ".")
	registry := getStringArg(args, "registry", "")
	platform := getStringArg(args, "platform", "")
	gitBranch := getStringArg(args, "git_branch", "main")
	gitCommit := getStringArg(args, "git_commit", "")
	tagsStr := getStringArg(args, "tags", "")
	buildArgsStr := getStringArg(args, "build_args", "{}")
	noCache := getBoolArg(args, "no_cache", false)
	pull := getBoolArg(args, "pull", true)
	validateUBI := getBoolArg(args, "validate_ubi", true)
	generateUBIDockerfile := getBoolArg(args, "generate_ubi_dockerfile", false)
	securityScan := getBoolArg(args, "security_scan", true)

	// Parse additional tags
	var additionalTags []string
	if tagsStr != "" {
		additionalTags = strings.Split(tagsStr, ",")
		for i, tag := range additionalTags {
			additionalTags[i] = strings.TrimSpace(tag)
		}
	}

	// Parse build args
	buildArgs := make(map[string]string)
	if buildArgsStr != "{}" {
		if err := json.Unmarshal([]byte(buildArgsStr), &buildArgs); err != nil {
			return NewTextResult("", fmt.Errorf("invalid build_args JSON: %v", err)), nil
		}
	}

	klog.V(2).Infof("Building container image: source=%s, type=%s, image=%s", source, sourceType, imageName)

	// Build the container with UBI validation
	buildResult, err := s.performContainerBuildWithValidation(ctx, ContainerBuildConfig{
		SourceType:   sourceType,
		Source:       source,
		Dockerfile:   dockerfile,
		BuildContext: buildContext,
		ImageName:    imageName,
		Registry:     registry,
		Tags:         additionalTags,
		BuildArgs:    buildArgs,
		Platform:     platform,
	}, gitBranch, gitCommit, noCache, pull, validateUBI, generateUBIDockerfile, securityScan)

	if err != nil {
		return NewTextResult("", fmt.Errorf("container build failed: %v", err)), nil
	}

	jsonResult, _ := json.MarshalIndent(buildResult, "", "  ")
	return NewTextResult(string(jsonResult), nil), nil
}

// containerPush handles pushing container images to registries
func (s *Server) containerPush(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return NewTextResult("", fmt.Errorf("invalid arguments format")), nil
	}

	imageName, ok := args["image_name"].(string)
	if !ok || imageName == "" {
		return NewTextResult("", fmt.Errorf("image_name parameter is required")), nil
	}

	registry := getStringArg(args, "registry", extractRegistryFromImage(imageName))
	username := getStringArg(args, "username", os.Getenv("REGISTRY_USERNAME"))
	password := getStringArg(args, "password", os.Getenv("REGISTRY_PASSWORD"))
	additionalTagsStr := getStringArg(args, "additional_tags", "")
	allTags := getBoolArg(args, "all_tags", false)
	skipTLSVerify := getBoolArg(args, "skip_tls_verify", false)

	var additionalTags []string
	if additionalTagsStr != "" {
		additionalTags = strings.Split(additionalTagsStr, ",")
		for i, tag := range additionalTags {
			additionalTags[i] = strings.TrimSpace(tag)
		}
	}

	klog.V(2).Infof("Pushing container image: %s to registry: %s", imageName, registry)

	pushResult, err := s.performContainerPush(ctx, imageName, registry, username, password, additionalTags, allTags, skipTLSVerify)
	if err != nil {
		return NewTextResult("", fmt.Errorf("container push failed: %v", err)), nil
	}

	jsonResult, _ := json.MarshalIndent(pushResult, "", "  ")
	return NewTextResult(string(jsonResult), nil), nil
}

// containerList handles listing local container images
func (s *Server) containerList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		args = make(map[string]interface{})
	}

	filter := getStringArg(args, "filter", "")
	registry := getStringArg(args, "registry", "")
	showAll := getBoolArg(args, "show_all", false)
	format := getStringArg(args, "format", "table")

	klog.V(2).Infof("Listing container images: filter=%s, registry=%s", filter, registry)

	listResult, err := s.performContainerList(ctx, filter, registry, showAll, format)
	if err != nil {
		return NewTextResult("", fmt.Errorf("container list failed: %v", err)), nil
	}

	if format == "json" {
		jsonResult, _ := json.MarshalIndent(listResult, "", "  ")
		return NewTextResult(string(jsonResult), nil), nil
	}

	return NewTextResult(listResult.(string), nil), nil
}

// containerRemove handles removing local container images
func (s *Server) containerRemove(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return NewTextResult("", fmt.Errorf("invalid arguments format")), nil
	}

	imageName, ok := args["image_name"].(string)
	if !ok || imageName == "" {
		return NewTextResult("", fmt.Errorf("image_name parameter is required")), nil
	}

	force := getBoolArg(args, "force", false)
	prune := getBoolArg(args, "prune", false)

	klog.V(2).Infof("Removing container image: %s", imageName)

	removeResult, err := s.performContainerRemove(ctx, imageName, force, prune)
	if err != nil {
		return NewTextResult("", fmt.Errorf("container remove failed: %v", err)), nil
	}

	jsonResult, _ := json.MarshalIndent(removeResult, "", "  ")
	return NewTextResult(string(jsonResult), nil), nil
}

// containerInspect handles inspecting container images
func (s *Server) containerInspect(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return NewTextResult("", fmt.Errorf("invalid arguments format")), nil
	}

	imageName, ok := args["image_name"].(string)
	if !ok || imageName == "" {
		return NewTextResult("", fmt.Errorf("image_name parameter is required")), nil
	}

	format := getStringArg(args, "format", "full")

	klog.V(2).Infof("Inspecting container image: %s", imageName)

	inspectResult, err := s.performContainerInspect(ctx, imageName, format)
	if err != nil {
		return NewTextResult("", fmt.Errorf("container inspect failed: %v", err)), nil
	}

	jsonResult, _ := json.MarshalIndent(inspectResult, "", "  ")
	return NewTextResult(string(jsonResult), nil), nil
}

// containerPull handles pulling container images from registries
func (s *Server) containerPull(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return NewTextResult("", fmt.Errorf("invalid arguments format")), nil
	}

	imageName, ok := args["image_name"].(string)
	if !ok || imageName == "" {
		return NewTextResult("", fmt.Errorf("image_name parameter is required")), nil
	}

	registry := getStringArg(args, "registry", extractRegistryFromImage(imageName))
	username := getStringArg(args, "username", os.Getenv("REGISTRY_USERNAME"))
	password := getStringArg(args, "password", os.Getenv("REGISTRY_PASSWORD"))
	platform := getStringArg(args, "platform", "")
	skipTLSVerify := getBoolArg(args, "skip_tls_verify", false)
	allTags := getBoolArg(args, "all_tags", false)

	klog.V(2).Infof("Pulling container image: %s from registry: %s", imageName, registry)

	pullResult, err := s.performContainerPull(ctx, imageName, registry, username, password, platform, skipTLSVerify, allTags)
	if err != nil {
		return NewTextResult("", fmt.Errorf("container pull failed: %v", err)), nil
	}

	jsonResult, _ := json.MarshalIndent(pullResult, "", "  ")
	return NewTextResult(string(jsonResult), nil), nil
}

// containerRun handles running containers from images
func (s *Server) containerRun(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return NewTextResult("", fmt.Errorf("invalid arguments format")), nil
	}

	imageName, ok := args["image_name"].(string)
	if !ok || imageName == "" {
		return NewTextResult("", fmt.Errorf("image_name parameter is required")), nil
	}

	containerName := getStringArg(args, "container_name", "")
	command := getStringArg(args, "command", "")
	workingDir := getStringArg(args, "working_dir", "")
	user := getStringArg(args, "user", "")
	restart := getStringArg(args, "restart", "no")
	detached := getBoolArg(args, "detached", true)
	interactive := getBoolArg(args, "interactive", false)
	remove := getBoolArg(args, "remove", false)
	publishAll := getBoolArg(args, "publish_all", false)

	// Parse port mappings
	var ports []string
	if portsArg, ok := args["ports"].([]interface{}); ok {
		for _, port := range portsArg {
			if portStr, ok := port.(string); ok {
				ports = append(ports, portStr)
			}
		}
	}

	// Parse environment variables
	var environment []string
	if envArg, ok := args["environment"].([]interface{}); ok {
		for _, env := range envArg {
			if envStr, ok := env.(string); ok {
				environment = append(environment, envStr)
			}
		}
	}

	// Parse volume mounts
	var volumes []string
	if volArg, ok := args["volumes"].([]interface{}); ok {
		for _, vol := range volArg {
			if volStr, ok := vol.(string); ok {
				volumes = append(volumes, volStr)
			}
		}
	}

	klog.V(2).Infof("Running container from image: %s", imageName)

	runResult, err := s.performContainerRun(ctx, imageName, containerName, command, workingDir, user, restart, ports, environment, volumes, detached, interactive, remove, publishAll)
	if err != nil {
		return NewTextResult("", fmt.Errorf("container run failed: %v", err)), nil
	}

	jsonResult, _ := json.MarshalIndent(runResult, "", "  ")
	return NewTextResult(string(jsonResult), nil), nil
}

// containerStop handles stopping running containers
func (s *Server) containerStop(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return NewTextResult("", fmt.Errorf("invalid arguments format")), nil
	}

	containerName, ok := args["container_name"].(string)
	if !ok || containerName == "" {
		return NewTextResult("", fmt.Errorf("container_name parameter is required")), nil
	}

	timeout := getStringArg(args, "timeout", "10s")
	force := getBoolArg(args, "force", false)

	klog.V(2).Infof("Stopping container: %s", containerName)

	stopResult, err := s.performContainerStop(ctx, containerName, timeout, force)
	if err != nil {
		return NewTextResult("", fmt.Errorf("container stop failed: %v", err)), nil
	}

	jsonResult, _ := json.MarshalIndent(stopResult, "", "  ")
	return NewTextResult(string(jsonResult), nil), nil
}

// Helper functions

func detectSourceType(source string) string {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		if strings.Contains(source, ".git") {
			return "git"
		}
		return "url"
	}
	if strings.HasPrefix(source, "git@") {
		return "git"
	}
	return "local"
}

func extractRegistryFromImage(imageName string) string {
	parts := strings.Split(imageName, "/")
	if len(parts) > 1 && strings.Contains(parts[0], ".") {
		return parts[0]
	}
	return "docker.io"
}

func getStringArg(args map[string]interface{}, key, defaultValue string) string {
	if val, ok := args[key].(string); ok {
		return val
	}
	return defaultValue
}

func getBoolArg(args map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := args[key].(bool); ok {
		return val
	}
	return defaultValue
}
