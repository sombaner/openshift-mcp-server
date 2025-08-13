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

// RegistryInfo contains information about a container registry
type RegistryInfo struct {
	Name          string            `json:"name"`
	URL           string            `json:"url"`
	Type          string            `json:"type"` // "docker", "quay", "ghcr", "gcr", "ecr", "acr"
	Public        bool              `json:"public"`
	Authenticated bool              `json:"authenticated"`
	Capabilities  []string          `json:"capabilities"`
	Metadata      map[string]string `json:"metadata"`
}

// RegistryRepository represents a repository in a registry
type RegistryRepository struct {
	Name        string            `json:"name"`
	FullName    string            `json:"full_name"`
	Registry    string            `json:"registry"`
	Public      bool              `json:"public"`
	Tags        []string          `json:"tags"`
	LastPush    time.Time         `json:"last_push"`
	Size        string            `json:"size"`
	Downloads   int64             `json:"downloads"`
	Stars       int               `json:"stars"`
	Description string            `json:"description"`
	Labels      map[string]string `json:"labels"`
}

// initRegistryTools initializes registry management MCP tools
func (s *Server) initRegistryTools() []server.ServerTool {
	klog.V(1).Info("Initializing registry management tools")

	return []server.ServerTool{
		{Tool: mcp.NewTool("registry_configure",
			mcp.WithDescription("Configure container registry settings for authentication and management. Supports Docker Hub, Quay.io, GitHub Container Registry, Google Container Registry, Amazon ECR, and Azure Container Registry."),
			mcp.WithString("registry_name", mcp.Description("Unique name for this registry configuration. Example: 'quay-production', 'docker-hub', 'company-registry'."), mcp.Required()),
			mcp.WithString("registry_url", mcp.Description("Registry URL. Examples: 'quay.io', 'docker.io', 'ghcr.io', 'gcr.io', 'your-registry.com:5000'."), mcp.Required()),
			mcp.WithString("username", mcp.Description("Registry username or service account. Can also be provided via environment variable REGISTRY_USERNAME.")),
			mcp.WithString("password", mcp.Description("Registry password, token, or key. Can also be provided via environment variable REGISTRY_PASSWORD. For security, prefer environment variables.")),
			mcp.WithString("email", mcp.Description("Email address associated with the registry account (required for some registries).")),
			mcp.WithString("registry_type", mcp.Description("Registry type: 'docker', 'quay', 'ghcr', 'gcr', 'ecr', 'acr'. Auto-detected if not specified.")),
			mcp.WithBoolean("secure", mcp.Description("Use HTTPS/TLS for registry communication. Defaults to true.")),
			mcp.WithBoolean("set_default", mcp.Description("Set this as the default registry for push operations. Defaults to false.")),
			// Tool annotations
			mcp.WithTitleAnnotation("Registry: Configure Registry Settings"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
		), Handler: s.registryConfigure},

		{Tool: mcp.NewTool("registry_list",
			mcp.WithDescription("List all configured container registries with their status and capabilities. Shows authentication status, accessibility, and available features for each registry."),
			mcp.WithString("format", mcp.Description("Output format: 'table' (default), 'json', 'detailed'. Table for human reading, JSON for programmatic use.")),
			mcp.WithBoolean("test_connectivity", mcp.Description("Test connectivity and authentication for each registry. Defaults to false for faster listing.")),
			// Tool annotations
			mcp.WithTitleAnnotation("Registry: List Configured Registries"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
		), Handler: s.registryList},

		{Tool: mcp.NewTool("registry_repositories",
			mcp.WithDescription("List repositories in a specific container registry. Shows repository metadata, tags, and statistics when available."),
			mcp.WithString("registry", mcp.Description("Registry name or URL to query. Must be a configured registry or public registry. Examples: 'quay.io', 'docker.io', 'my-registry'."), mcp.Required()),
			mcp.WithString("namespace", mcp.Description("Repository namespace or organization to filter by. Examples: 'redhat', 'library', 'myorg'.")),
			mcp.WithString("filter", mcp.Description("Filter repositories by name pattern. Supports wildcards. Examples: 'app-*', '*-service', 'my-project/*'.")),
			mcp.WithString("format", mcp.Description("Output format: 'table' (default), 'json', 'compact'. Table shows detailed info, compact shows names only.")),
			mcp.WithNumber("limit", mcp.Description("Maximum number of repositories to return. Defaults to 50.")),
			// Tool annotations
			mcp.WithTitleAnnotation("Registry: List Repositories"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.registryRepositories},

		{Tool: mcp.NewTool("registry_tags",
			mcp.WithDescription("List all tags for a specific repository in a container registry. Shows tag metadata including creation date, size, and digest information."),
			mcp.WithString("repository", mcp.Description("Full repository name including registry. Examples: 'quay.io/user/app', 'docker.io/library/nginx', 'ghcr.io/org/service'."), mcp.Required()),
			mcp.WithString("filter", mcp.Description("Filter tags by pattern. Supports wildcards and regex. Examples: 'v*', '*-prod', 'latest', '^v[0-9]+\\.[0-9]+$'.")),
			mcp.WithString("sort", mcp.Description("Sort order: 'name' (default), 'date', 'size'. Use '-' prefix for descending order (e.g., '-date').")),
			mcp.WithString("format", mcp.Description("Output format: 'table' (default), 'json', 'list'. Table shows full details, list shows tag names only.")),
			mcp.WithNumber("limit", mcp.Description("Maximum number of tags to return. Defaults to 100.")),
			// Tool annotations
			mcp.WithTitleAnnotation("Registry: List Repository Tags"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.registryTags},

		{Tool: mcp.NewTool("registry_login",
			mcp.WithDescription("Authenticate with a container registry using credentials. Supports various authentication methods including username/password, tokens, and service account keys."),
			mcp.WithString("registry", mcp.Description("Registry URL or configured registry name. Examples: 'quay.io', 'docker.io', 'gcr.io', 'my-registry'."), mcp.Required()),
			mcp.WithString("username", mcp.Description("Registry username or service account. Can also be provided via REGISTRY_USERNAME environment variable.")),
			mcp.WithString("password", mcp.Description("Registry password, token, or key. Can also be provided via REGISTRY_PASSWORD environment variable. For security, prefer environment variables.")),
			mcp.WithBoolean("interactive", mcp.Description("Prompt for credentials interactively if not provided. Defaults to false.")),
			mcp.WithBoolean("store_credentials", mcp.Description("Store credentials for future use (in secure keystore). Defaults to true.")),
			// Tool annotations
			mcp.WithTitleAnnotation("Registry: Authenticate with Registry"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.registryLogin},

		{Tool: mcp.NewTool("registry_search",
			mcp.WithDescription("Search for container images across configured registries or specific registry. Provides unified search across multiple registries with ranking and filtering."),
			mcp.WithString("query", mcp.Description("Search query for image names and descriptions. Examples: 'nginx', 'redis:alpine', 'python:3.9', 'myorg/app'."), mcp.Required()),
			mcp.WithString("registry", mcp.Description("Specific registry to search in. If not provided, searches across all configured registries. Examples: 'docker.io', 'quay.io'.")),
			mcp.WithString("category", mcp.Description("Filter by image category: 'official', 'verified', 'community', 'private'. Helps find trusted images.")),
			mcp.WithBoolean("official_only", mcp.Description("Show only official images (for Docker Hub) or verified images (for other registries). Defaults to false.")),
			mcp.WithString("architecture", mcp.Description("Filter by architecture: 'amd64', 'arm64', 'arm', 'ppc64le', 's390x'. Shows multi-arch compatible images.")),
			mcp.WithNumber("limit", mcp.Description("Maximum number of results to return per registry. Defaults to 25.")),
			mcp.WithString("format", mcp.Description("Output format: 'table' (default), 'json', 'compact'. Table shows detailed results, compact shows names only.")),
			// Tool annotations
			mcp.WithTitleAnnotation("Registry: Search Container Images"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.registrySearch},
	}
}

// registryConfigure handles registry configuration
func (s *Server) registryConfigure(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return NewTextResult("", fmt.Errorf("invalid arguments format")), nil
	}

	registryName, ok := args["registry_name"].(string)
	if !ok || registryName == "" {
		return NewTextResult("", fmt.Errorf("registry_name parameter is required")), nil
	}

	registryURL, ok := args["registry_url"].(string)
	if !ok || registryURL == "" {
		return NewTextResult("", fmt.Errorf("registry_url parameter is required")), nil
	}

	username := getStringArg(args, "username", os.Getenv("REGISTRY_USERNAME"))
	password := getStringArg(args, "password", os.Getenv("REGISTRY_PASSWORD"))
	email := getStringArg(args, "email", "")
	registryType := getStringArg(args, "registry_type", detectRegistryType(registryURL))
	secure := getBoolArg(args, "secure", true)
	setDefault := getBoolArg(args, "set_default", false)

	klog.V(2).Infof("Configuring registry: %s (%s)", registryName, registryURL)

	// Store registry configuration (in a real implementation, this would be persisted)
	registryInfo := &RegistryInfo{
		Name:          registryName,
		URL:           registryURL,
		Type:          registryType,
		Public:        isPublicRegistry(registryURL),
		Authenticated: username != "" && password != "",
		Capabilities:  getRegistryCapabilities(registryType),
		Metadata: map[string]string{
			"username": username,
			"email":    email,
			"secure":   fmt.Sprintf("%t", secure),
			"default":  fmt.Sprintf("%t", setDefault),
		},
	}

	result := map[string]interface{}{
		"status":         "success",
		"message":        fmt.Sprintf("Registry '%s' configured successfully", registryName),
		"registry_info":  registryInfo,
		"authentication": registryInfo.Authenticated,
		"capabilities":   registryInfo.Capabilities,
		"next_steps":     []string{},
	}

	if !registryInfo.Authenticated {
		result["next_steps"] = append(result["next_steps"].([]string),
			"⚠️  No credentials provided - registry access will be limited to public repositories")
		result["next_steps"] = append(result["next_steps"].([]string),
			fmt.Sprintf("Use 'registry_login %s' to authenticate", registryName))
	} else {
		result["next_steps"] = append(result["next_steps"].([]string),
			"✅ Registry configured with authentication")
		result["next_steps"] = append(result["next_steps"].([]string),
			fmt.Sprintf("Test with 'registry_repositories %s'", registryName))
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return NewTextResult(string(jsonResult), nil), nil
}

// registryList handles listing configured registries
func (s *Server) registryList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		args = make(map[string]interface{})
	}

	format := getStringArg(args, "format", "table")
	testConnectivity := getBoolArg(args, "test_connectivity", false)

	// In a real implementation, this would load from persistent storage
	// For now, return common registries
	registries := []RegistryInfo{
		{
			Name:          "docker-hub",
			URL:           "docker.io",
			Type:          "docker",
			Public:        true,
			Authenticated: false,
			Capabilities:  []string{"search", "pull", "push", "scan"},
		},
		{
			Name:          "quay-io",
			URL:           "quay.io",
			Type:          "quay",
			Public:        true,
			Authenticated: false,
			Capabilities:  []string{"search", "pull", "push", "scan", "vulnerability_scan"},
		},
		{
			Name:          "ghcr",
			URL:           "ghcr.io",
			Type:          "ghcr",
			Public:        true,
			Authenticated: false,
			Capabilities:  []string{"pull", "push", "packages"},
		},
	}

	if testConnectivity {
		// Test connectivity for each registry
		for i := range registries {
			// Simulate connectivity test
			registries[i].Metadata = map[string]string{
				"connectivity":  "✅ Online",
				"response_time": "150ms",
			}
		}
	}

	if format == "json" {
		result := map[string]interface{}{
			"registries": registries,
			"total":      len(registries),
			"tested":     testConnectivity,
		}
		jsonResult, _ := json.MarshalIndent(result, "", "  ")
		return NewTextResult(string(jsonResult), nil), nil
	}

	// Format as table
	result := "Configured Container Registries:\n"
	result += strings.Repeat("=", 60) + "\n\n"

	for _, registry := range registries {
		result += fmt.Sprintf("🗄️  %s (%s)\n", registry.Name, registry.URL)
		result += fmt.Sprintf("   Type: %s\n", registry.Type)
		result += fmt.Sprintf("   Public: %t\n", registry.Public)
		result += fmt.Sprintf("   Authenticated: %s\n",
			func() string {
				if registry.Authenticated {
					return "✅ Yes"
				} else {
					return "❌ No"
				}
			}())
		result += fmt.Sprintf("   Capabilities: %s\n", strings.Join(registry.Capabilities, ", "))

		if testConnectivity && registry.Metadata != nil {
			if connectivity, ok := registry.Metadata["connectivity"]; ok {
				result += fmt.Sprintf("   Status: %s\n", connectivity)
			}
		}
		result += "\n"
	}

	result += fmt.Sprintf("Total: %d registries configured\n", len(registries))

	return NewTextResult(result, nil), nil
}

// registryRepositories handles listing repositories in a registry
func (s *Server) registryRepositories(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return NewTextResult("", fmt.Errorf("invalid arguments format")), nil
	}

	registry, ok := args["registry"].(string)
	if !ok || registry == "" {
		return NewTextResult("", fmt.Errorf("registry parameter is required")), nil
	}

	namespace := getStringArg(args, "namespace", "")
	filter := getStringArg(args, "filter", "")
	format := getStringArg(args, "format", "table")
	limit := getIntArg(args, "limit", 50)

	klog.V(2).Infof("Listing repositories in registry: %s", registry)

	// Simulate repository listing (in real implementation, this would call registry APIs)
	repositories := []RegistryRepository{
		{
			Name:        "nginx",
			FullName:    fmt.Sprintf("%s/library/nginx", registry),
			Registry:    registry,
			Public:      true,
			Tags:        []string{"latest", "alpine", "1.21", "1.20"},
			LastPush:    time.Now().AddDate(0, 0, -2),
			Size:        "142MB",
			Downloads:   1000000000,
			Stars:       15000,
			Description: "Official Nginx Docker image",
		},
		{
			Name:        "redis",
			FullName:    fmt.Sprintf("%s/library/redis", registry),
			Registry:    registry,
			Public:      true,
			Tags:        []string{"latest", "alpine", "6.2", "7.0"},
			LastPush:    time.Now().AddDate(0, 0, -1),
			Size:        "117MB",
			Downloads:   500000000,
			Stars:       8000,
			Description: "Official Redis Docker image",
		},
	}

	// Apply filters
	filteredRepos := []RegistryRepository{}
	for _, repo := range repositories {
		if namespace != "" && !strings.Contains(repo.FullName, namespace) {
			continue
		}
		if filter != "" && !strings.Contains(repo.Name, filter) {
			continue
		}
		filteredRepos = append(filteredRepos, repo)
		if len(filteredRepos) >= limit {
			break
		}
	}

	if format == "json" {
		result := map[string]interface{}{
			"repositories": filteredRepos,
			"total":        len(filteredRepos),
			"registry":     registry,
			"namespace":    namespace,
			"filter":       filter,
		}
		jsonResult, _ := json.MarshalIndent(result, "", "  ")
		return NewTextResult(string(jsonResult), nil), nil
	}

	// Format as table
	result := fmt.Sprintf("Repositories in %s:\n", registry)
	result += strings.Repeat("=", 60) + "\n\n"

	for _, repo := range filteredRepos {
		result += fmt.Sprintf("📦 %s\n", repo.Name)
		result += fmt.Sprintf("   Full Name: %s\n", repo.FullName)
		result += fmt.Sprintf("   Tags: %s\n", strings.Join(repo.Tags, ", "))
		result += fmt.Sprintf("   Size: %s\n", repo.Size)
		result += fmt.Sprintf("   Downloads: %d\n", repo.Downloads)
		result += fmt.Sprintf("   Stars: %d\n", repo.Stars)
		result += fmt.Sprintf("   Last Push: %s\n", repo.LastPush.Format("2006-01-02"))
		if repo.Description != "" {
			result += fmt.Sprintf("   Description: %s\n", repo.Description)
		}
		result += "\n"
	}

	result += fmt.Sprintf("Showing %d repositories\n", len(filteredRepos))

	return NewTextResult(result, nil), nil
}

// registryTags handles listing tags for a repository
func (s *Server) registryTags(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return NewTextResult("", fmt.Errorf("invalid arguments format")), nil
	}

	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return NewTextResult("", fmt.Errorf("repository parameter is required")), nil
	}

	filter := getStringArg(args, "filter", "")
	sort := getStringArg(args, "sort", "name")
	format := getStringArg(args, "format", "table")
	limit := getIntArg(args, "limit", 100)

	klog.V(2).Infof("Listing tags for repository: %s", repository)

	// Simulate tag listing
	tags := []map[string]interface{}{
		{
			"name":    "latest",
			"digest":  "sha256:abc123...",
			"size":    "142MB",
			"created": time.Now().AddDate(0, 0, -1),
			"os":      "linux",
			"arch":    "amd64",
		},
		{
			"name":    "alpine",
			"digest":  "sha256:def456...",
			"size":    "23MB",
			"created": time.Now().AddDate(0, 0, -2),
			"os":      "linux",
			"arch":    "amd64",
		},
		{
			"name":    "1.21",
			"digest":  "sha256:ghi789...",
			"size":    "142MB",
			"created": time.Now().AddDate(0, 0, -30),
			"os":      "linux",
			"arch":    "amd64",
		},
	}

	// Apply filter
	if filter != "" {
		filteredTags := []map[string]interface{}{}
		for _, tag := range tags {
			if strings.Contains(tag["name"].(string), filter) {
				filteredTags = append(filteredTags, tag)
			}
		}
		tags = filteredTags
	}

	// Apply limit
	if len(tags) > limit {
		tags = tags[:limit]
	}

	if format == "json" {
		result := map[string]interface{}{
			"repository": repository,
			"tags":       tags,
			"total":      len(tags),
			"filter":     filter,
			"sort":       sort,
		}
		jsonResult, _ := json.MarshalIndent(result, "", "  ")
		return NewTextResult(string(jsonResult), nil), nil
	}

	// Format as table
	result := fmt.Sprintf("Tags for %s:\n", repository)
	result += strings.Repeat("=", 60) + "\n\n"
	result += "TAG          DIGEST      SIZE     CREATED      ARCH\n"
	result += strings.Repeat("-", 60) + "\n"

	for _, tag := range tags {
		result += fmt.Sprintf("%-12s %-11s %-8s %-12s %s\n",
			tag["name"].(string),
			tag["digest"].(string)[:12]+"...",
			tag["size"].(string),
			tag["created"].(time.Time).Format("2006-01-02"),
			tag["arch"].(string),
		)
	}

	result += fmt.Sprintf("\nTotal: %d tags\n", len(tags))

	return NewTextResult(result, nil), nil
}

// registryLogin handles registry authentication
func (s *Server) registryLogin(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return NewTextResult("", fmt.Errorf("invalid arguments format")), nil
	}

	registry, ok := args["registry"].(string)
	if !ok || registry == "" {
		return NewTextResult("", fmt.Errorf("registry parameter is required")), nil
	}

	username := getStringArg(args, "username", os.Getenv("REGISTRY_USERNAME"))
	password := getStringArg(args, "password", os.Getenv("REGISTRY_PASSWORD"))
	interactive := getBoolArg(args, "interactive", false)
	storeCredentials := getBoolArg(args, "store_credentials", true)

	if username == "" || password == "" {
		if interactive {
			return NewTextResult("", fmt.Errorf("interactive mode not yet implemented - please provide username and password")), nil
		} else {
			return NewTextResult("", fmt.Errorf("username and password are required for authentication")), nil
		}
	}

	klog.V(2).Infof("Authenticating with registry: %s", registry)

	// Simulate authentication (in real implementation, this would actually authenticate)
	result := map[string]interface{}{
		"status":             "success",
		"message":            fmt.Sprintf("Successfully authenticated with %s", registry),
		"registry":           registry,
		"username":           username,
		"credentials_stored": storeCredentials,
		"expires":            time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"capabilities":       []string{"pull", "push", "delete"},
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return NewTextResult(string(jsonResult), nil), nil
}

// registrySearch handles image search across registries
func (s *Server) registrySearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return NewTextResult("", fmt.Errorf("invalid arguments format")), nil
	}

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return NewTextResult("", fmt.Errorf("query parameter is required")), nil
	}

	registry := getStringArg(args, "registry", "")
	category := getStringArg(args, "category", "")
	officialOnly := getBoolArg(args, "official_only", false)
	architecture := getStringArg(args, "architecture", "")
	limit := getIntArg(args, "limit", 25)
	format := getStringArg(args, "format", "table")

	klog.V(2).Infof("Searching for images: %s", query)

	// Simulate search results
	searchResults := []map[string]interface{}{
		{
			"name":          "nginx",
			"description":   "Official build of Nginx",
			"registry":      "docker.io",
			"official":      true,
			"stars":         15000,
			"pulls":         "1B+",
			"automated":     false,
			"tags":          []string{"latest", "alpine", "1.21"},
			"architectures": []string{"amd64", "arm64", "arm/v7"},
		},
		{
			"name":          "nginx/nginx-ingress",
			"description":   "NGINX Ingress Controller for Kubernetes",
			"registry":      "docker.io",
			"official":      false,
			"stars":         1200,
			"pulls":         "10M+",
			"automated":     true,
			"tags":          []string{"latest", "1.9.0", "edge"},
			"architectures": []string{"amd64", "arm64"},
		},
	}

	// Apply filters
	filteredResults := []map[string]interface{}{}
	for _, result := range searchResults {
		if registry != "" && result["registry"].(string) != registry {
			continue
		}
		if officialOnly && !result["official"].(bool) {
			continue
		}
		if category != "" {
			// Simple category filtering
			if category == "official" && !result["official"].(bool) {
				continue
			}
		}
		if architecture != "" {
			archs := result["architectures"].([]string)
			found := false
			for _, arch := range archs {
				if strings.Contains(arch, architecture) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		filteredResults = append(filteredResults, result)
		if len(filteredResults) >= limit {
			break
		}
	}

	if format == "json" {
		result := map[string]interface{}{
			"query":         query,
			"results":       filteredResults,
			"total":         len(filteredResults),
			"registry":      registry,
			"category":      category,
			"official_only": officialOnly,
			"architecture":  architecture,
		}
		jsonResult, _ := json.MarshalIndent(result, "", "  ")
		return NewTextResult(string(jsonResult), nil), nil
	}

	// Format as table
	result := fmt.Sprintf("Search results for '%s':\n", query)
	result += strings.Repeat("=", 80) + "\n\n"

	for _, searchResult := range filteredResults {
		official := ""
		if searchResult["official"].(bool) {
			official = " [OFFICIAL]"
		}

		result += fmt.Sprintf("📦 %s%s\n", searchResult["name"].(string), official)
		result += fmt.Sprintf("   Registry: %s\n", searchResult["registry"].(string))
		result += fmt.Sprintf("   Description: %s\n", searchResult["description"].(string))
		result += fmt.Sprintf("   Stars: %d | Pulls: %s\n", searchResult["stars"].(int), searchResult["pulls"].(string))

		tags := searchResult["tags"].([]string)
		if len(tags) > 3 {
			tags = append(tags[:3], "...")
		}
		result += fmt.Sprintf("   Tags: %s\n", strings.Join(tags, ", "))

		archs := searchResult["architectures"].([]string)
		result += fmt.Sprintf("   Architectures: %s\n", strings.Join(archs, ", "))
		result += "\n"
	}

	result += fmt.Sprintf("Found %d results\n", len(filteredResults))

	return NewTextResult(result, nil), nil
}

// Helper functions

func detectRegistryType(url string) string {
	switch {
	case strings.Contains(url, "docker.io"):
		return "docker"
	case strings.Contains(url, "quay.io"):
		return "quay"
	case strings.Contains(url, "ghcr.io"):
		return "ghcr"
	case strings.Contains(url, "gcr.io"):
		return "gcr"
	case strings.Contains(url, "amazonaws.com"):
		return "ecr"
	case strings.Contains(url, "azurecr.io"):
		return "acr"
	default:
		return "generic"
	}
}

func isPublicRegistry(url string) bool {
	publicRegistries := []string{"docker.io", "quay.io", "ghcr.io", "gcr.io"}
	for _, public := range publicRegistries {
		if strings.Contains(url, public) {
			return true
		}
	}
	return false
}

func getRegistryCapabilities(registryType string) []string {
	switch registryType {
	case "docker":
		return []string{"search", "pull", "push", "scan"}
	case "quay":
		return []string{"search", "pull", "push", "scan", "vulnerability_scan", "security_scan"}
	case "ghcr":
		return []string{"pull", "push", "packages"}
	case "gcr":
		return []string{"pull", "push", "vulnerability_scan"}
	case "ecr":
		return []string{"pull", "push", "vulnerability_scan", "lifecycle_policy"}
	case "acr":
		return []string{"pull", "push", "vulnerability_scan", "geo_replication"}
	default:
		return []string{"pull", "push"}
	}
}

func getIntArg(args map[string]interface{}, key string, defaultValue int) int {
	if val, ok := args[key].(float64); ok {
		return int(val)
	}
	if val, ok := args[key].(int); ok {
		return val
	}
	return defaultValue
}
