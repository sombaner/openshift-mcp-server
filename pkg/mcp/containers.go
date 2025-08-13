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
			mcp.WithDescription("Build a container image from source code. Supports Git repositories, local directories, and remote archives. Uses podman/docker for building and can automatically detect Dockerfile location. Provides comprehensive build logging and error handling."),
			mcp.WithString("source", mcp.Description("Source location for the container build. Can be a Git repository URL (https://github.com/user/repo.git), local directory path (/path/to/source), or remote archive URL. For Git repos, supports specific branches and commits."), mcp.Required()),
			mcp.WithString("source_type", mcp.Description("Type of source: 'git' for Git repositories, 'local' for local directories, 'url' for remote archives. Defaults to 'git' if source looks like a Git URL, 'local' otherwise.")),
			mcp.WithString("image_name", mcp.Description("Target container image name. Should include registry if pushing later. Examples: 'my-app:latest', 'quay.io/user/app:v1.0', 'localhost/test:dev'. If not provided, generates name from source."), mcp.Required()),
			mcp.WithString("dockerfile", mcp.Description("Path to Dockerfile relative to build context. Defaults to 'Dockerfile' in the root. Examples: './Dockerfile', 'docker/Dockerfile', 'build/Dockerfile.prod'.")),
			mcp.WithString("build_context", mcp.Description("Build context directory relative to source root. Defaults to '.' (source root). Use subdirectory like 'backend' or 'src' if needed.")),
			mcp.WithString("registry", mcp.Description("Target container registry for the image. Examples: 'quay.io', 'docker.io', 'ghcr.io', 'localhost:5000'. Used for proper image naming and future push operations.")),
			mcp.WithString("tags", mcp.Description("Comma-separated list of additional tags for the image. Example: 'latest,v1.0,staging'. The main tag is derived from image_name.")),
			mcp.WithString("platform", mcp.Description("Target platform for the build. Examples: 'linux/amd64', 'linux/arm64', 'linux/amd64,linux/arm64' for multi-platform. Defaults to current platform.")),
			mcp.WithString("build_args", mcp.Description("Build arguments as JSON string. Example: '{\"ENV\":\"production\",\"VERSION\":\"1.0\"}'. These are passed as --build-arg to the container build.")),
			mcp.WithString("git_branch", mcp.Description("Git branch to checkout (only for Git sources). Defaults to 'main' or default branch.")),
			mcp.WithString("git_commit", mcp.Description("Specific Git commit hash to checkout (only for Git sources). Takes precedence over branch.")),
			mcp.WithBoolean("no_cache", mcp.Description("Disable build cache. Useful for ensuring fresh builds. Defaults to false (cache enabled).")),
			mcp.WithBoolean("pull", mcp.Description("Always pull latest base images during build. Defaults to true for fresh builds.")),
			// Tool annotations
			mcp.WithTitleAnnotation("Container: Build Image from Source"),
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

	// Build the container
	buildResult, err := s.performContainerBuild(ctx, ContainerBuildConfig{
		SourceType:   sourceType,
		Source:       source,
		Dockerfile:   dockerfile,
		BuildContext: buildContext,
		ImageName:    imageName,
		Registry:     registry,
		Tags:         additionalTags,
		BuildArgs:    buildArgs,
		Platform:     platform,
	}, gitBranch, gitCommit, noCache, pull)

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
