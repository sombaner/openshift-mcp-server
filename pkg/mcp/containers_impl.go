package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

// performContainerBuildWithValidation executes container build with UBI and security validation
func (s *Server) performContainerBuildWithValidation(ctx context.Context, config ContainerBuildConfig, gitBranch, gitCommit string, noCache, pull, validateUBI, generateUBIDockerfile, securityScan bool) (map[string]interface{}, error) {
	startTime := time.Now()
	
	// Detect container runtime (podman or docker)
	containerRuntime, err := detectContainerRuntime()
	if err != nil {
		return nil, fmt.Errorf("no container runtime found: %v", err)
	}
	
	klog.V(1).Infof("Using container runtime: %s", containerRuntime)

	// Prepare build directory
	buildDir, err := s.prepareBuildSource(ctx, config, gitBranch, gitCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare build source: %v", err)
	}
	defer func() {
		if config.SourceType != "local" {
			os.RemoveAll(buildDir)
		}
	}()

	// Perform validations if requested
	var validation map[string]interface{}
	if validateUBI || securityScan {
		validation, err = s.enhancedContainerBuildValidation(ctx, config, buildDir)
		if err != nil {
			klog.V(1).Infof("Validation failed: %v", err)
		}
	}

	// Handle UBI Dockerfile generation if requested
	actualDockerfile := config.Dockerfile
	if generateUBIDockerfile && validation != nil {
		if ubiValidation, ok := validation["ubi_compliance"].(*UBIValidation); ok && !ubiValidation.IsUBI {
			ubiDockerfilePath, err := s.generateUBIDockerfile(ctx, 
				filepath.Join(buildDir, config.BuildContext, config.Dockerfile), 
				ubiValidation.SuggestedUBIImage)
			if err != nil {
				klog.V(1).Infof("Failed to generate UBI Dockerfile: %v", err)
			} else {
				actualDockerfile = filepath.Base(ubiDockerfilePath)
				validation["ubi_dockerfile_generated"] = ubiDockerfilePath
				klog.V(1).Infof("Generated UBI Dockerfile: %s", ubiDockerfilePath)
			}
		}
	}

	// Update config with actual dockerfile
	config.Dockerfile = actualDockerfile

	// Construct build command
	buildCmd := s.constructBuildCommand(containerRuntime, config, buildDir, noCache, pull)
	
	klog.V(2).Infof("Executing build command: %s", strings.Join(buildCmd.Args, " "))

	// Execute build with output capture
	buildOutput, err := s.executeBuildCommand(ctx, buildCmd)
	if err != nil {
		return nil, fmt.Errorf("build failed: %v\nOutput: %s", err, buildOutput)
	}

	buildDuration := time.Since(startTime)

	// Get image information
	imageInfo, err := s.getImageInfo(ctx, containerRuntime, config.ImageName)
	if err != nil {
		klog.V(1).Infof("Warning: failed to get image info: %v", err)
		imageInfo = &ContainerImageInfo{
			ImageName:     config.ImageName,
			Tags:          append([]string{extractTagFromImage(config.ImageName)}, config.Tags...),
			BuildDuration: buildDuration.String(),
			CreatedAt:     time.Now(),
		}
	} else {
		imageInfo.BuildDuration = buildDuration.String()
	}

	result := map[string]interface{}{
		"status":          "success",
		"message":         fmt.Sprintf("Container image '%s' built successfully", config.ImageName),
		"image_info":      imageInfo,
		"build_duration":  buildDuration.String(),
		"container_runtime": containerRuntime,
		"build_output":    strings.Split(buildOutput, "\n"),
		"source_info": map[string]interface{}{
			"type":         config.SourceType,
			"source":       config.Source,
			"dockerfile":   config.Dockerfile,
			"build_context": config.BuildContext,
		},
		"next_steps": []string{
			fmt.Sprintf("Image is ready for local use: %s run %s", containerRuntime, config.ImageName),
			fmt.Sprintf("To push to registry: %s push %s", containerRuntime, config.ImageName),
		},
	}

	// Include validation results if performed
	if validation != nil {
		result["validation"] = validation
		
		// Add UBI-specific recommendations
		if ubiValidation, ok := validation["ubi_compliance"].(*UBIValidation); ok {
			if !ubiValidation.IsUBI {
				result["next_steps"] = append(result["next_steps"].([]string), 
					fmt.Sprintf("⚠️ Consider using Red Hat UBI base image: %s", ubiValidation.SuggestedUBIImage))
				result["next_steps"] = append(result["next_steps"].([]string), 
					"UBI provides enterprise security, compliance, and support")
			} else {
				result["next_steps"] = append(result["next_steps"].([]string), 
					"✅ Image uses Red Hat UBI - enterprise ready!")
			}
		}
	}

	return result, nil
}

// performContainerPush executes the actual container push process
func (s *Server) performContainerPush(ctx context.Context, imageName, registry, username, password string, additionalTags []string, allTags, skipTLSVerify bool) (map[string]interface{}, error) {
	containerRuntime, err := detectContainerRuntime()
	if err != nil {
		return nil, fmt.Errorf("no container runtime found: %v", err)
	}

	// Authenticate if credentials provided
	if username != "" && password != "" {
		if err := s.authenticateRegistry(ctx, containerRuntime, registry, username, password); err != nil {
			return nil, fmt.Errorf("registry authentication failed: %v", err)
		}
	}

	pushedImages := []string{}
	pushResults := []map[string]interface{}{}

	// Push main image
	pushResult, err := s.pushSingleImage(ctx, containerRuntime, imageName, skipTLSVerify)
	if err != nil {
		return nil, fmt.Errorf("failed to push image %s: %v", imageName, err)
	}
	pushedImages = append(pushedImages, imageName)
	pushResults = append(pushResults, pushResult)

	// Push additional tags
	for _, tag := range additionalTags {
		taggedImage := addTagToImage(imageName, tag)
		
		// Tag the image first
		if err := s.tagImage(ctx, containerRuntime, imageName, taggedImage); err != nil {
			klog.V(1).Infof("Warning: failed to tag image %s as %s: %v", imageName, taggedImage, err)
			continue
		}

		// Push tagged image
		pushResult, err := s.pushSingleImage(ctx, containerRuntime, taggedImage, skipTLSVerify)
		if err != nil {
			klog.V(1).Infof("Warning: failed to push tagged image %s: %v", taggedImage, err)
			continue
		}
		pushedImages = append(pushedImages, taggedImage)
		pushResults = append(pushResults, pushResult)
	}

	result := map[string]interface{}{
		"status":             "success",
		"message":            fmt.Sprintf("Successfully pushed %d image(s)", len(pushedImages)),
		"pushed_images":      pushedImages,
		"push_results":       pushResults,
		"registry":           registry,
		"container_runtime":  containerRuntime,
		"authentication":     username != "",
		"total_pushed":       len(pushedImages),
	}

	return result, nil
}

// performContainerList executes container image listing
func (s *Server) performContainerList(ctx context.Context, filter, registry string, showAll bool, format string) (interface{}, error) {
	containerRuntime, err := detectContainerRuntime()
	if err != nil {
		return nil, fmt.Errorf("no container runtime found: %v", err)
	}

	cmd := exec.CommandContext(ctx, containerRuntime, "images")
	
	if showAll {
		cmd.Args = append(cmd.Args, "--all")
	}
	
	if filter != "" {
		cmd.Args = append(cmd.Args, "--filter", fmt.Sprintf("reference=%s", filter))
	}

	if format == "json" {
		cmd.Args = append(cmd.Args, "--format", "json")
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %v", err)
	}

	if format == "json" {
		// Parse and return structured data
		images := []map[string]interface{}{}
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			if line != "" {
				imageInfo := parseImageLine(line)
				if registry == "" || strings.Contains(imageInfo["repository"].(string), registry) {
					images = append(images, imageInfo)
				}
			}
		}
		return map[string]interface{}{
			"images": images,
			"total":  len(images),
			"filter": filter,
			"registry": registry,
		}, nil
	}

	// Return formatted text output
	result := fmt.Sprintf("Container Images (using %s):\n", containerRuntime)
	result += "REPOSITORY                TAG       IMAGE ID       CREATED        SIZE\n"
	result += strings.Repeat("-", 80) + "\n"
	result += string(output)

	return result, nil
}

// performContainerRemove executes container image removal
func (s *Server) performContainerRemove(ctx context.Context, imageName string, force, prune bool) (map[string]interface{}, error) {
	containerRuntime, err := detectContainerRuntime()
	if err != nil {
		return nil, fmt.Errorf("no container runtime found: %v", err)
	}

	cmd := exec.CommandContext(ctx, containerRuntime, "rmi")
	
	if force {
		cmd.Args = append(cmd.Args, "--force")
	}
	
	cmd.Args = append(cmd.Args, imageName)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to remove image: %v\nOutput: %s", err, string(output))
	}

	result := map[string]interface{}{
		"status":             "success",
		"message":            fmt.Sprintf("Successfully removed image: %s", imageName),
		"removed_image":      imageName,
		"container_runtime":  containerRuntime,
		"output":             string(output),
	}

	// Optionally prune unused images
	if prune {
		pruneCmd := exec.CommandContext(ctx, containerRuntime, "image", "prune", "-f")
		pruneOutput, pruneErr := pruneCmd.CombinedOutput()
		if pruneErr != nil {
			klog.V(1).Infof("Warning: failed to prune images: %v", pruneErr)
		} else {
			result["prune_output"] = string(pruneOutput)
		}
	}

	return result, nil
}

// performContainerInspect executes container image inspection
func (s *Server) performContainerInspect(ctx context.Context, imageName, format string) (interface{}, error) {
	containerRuntime, err := detectContainerRuntime()
	if err != nil {
		return nil, fmt.Errorf("no container runtime found: %v", err)
	}

	cmd := exec.CommandContext(ctx, containerRuntime, "inspect", imageName)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect image: %v", err)
	}

	// Parse JSON output (both podman and docker return JSON)
	var inspectData interface{}
	if err := parseJSON(string(output), &inspectData); err != nil {
		return nil, fmt.Errorf("failed to parse inspect output: %v", err)
	}

	result := map[string]interface{}{
		"image_name":        imageName,
		"container_runtime": containerRuntime,
		"inspect_data":      inspectData,
		"format":            format,
	}

	return result, nil
}

// Helper implementation functions

func detectContainerRuntime() (string, error) {
	// Check if runtime is explicitly set via environment variable
	if runtime := os.Getenv("CONTAINER_RUNTIME"); runtime != "" {
		if _, err := exec.LookPath(runtime); err == nil {
			// Initialize Podman if it's the chosen runtime
			if runtime == "podman" {
				initializePodman()
			}
			return runtime, nil
		}
		klog.V(1).Infof("Requested container runtime %s not found, falling back to auto-detection", runtime)
	}
	
	// Check for podman first (preferred for OpenShift environments)
	if _, err := exec.LookPath("podman"); err == nil {
		initializePodman()
		return "podman", nil
	}
	
	// Fall back to docker
	if _, err := exec.LookPath("docker"); err == nil {
		return "docker", nil
	}
	
	return "", fmt.Errorf("neither podman nor docker found in PATH")
}

// initializePodman ensures Podman storage is properly initialized
func initializePodman() {
	// Run podman system migrate to initialize storage (ignore errors as it may already be initialized)
	cmd := exec.Command("podman", "system", "migrate")
	if err := cmd.Run(); err != nil {
		klog.V(2).Infof("Podman system migrate failed (may already be initialized): %v", err)
	}
	
	// Verify Podman is working by running podman info
	cmd = exec.Command("podman", "info", "--format", "json")
	if err := cmd.Run(); err != nil {
		klog.V(1).Infof("Warning: Podman info check failed: %v", err)
	} else {
		klog.V(1).Info("Podman runtime initialized successfully")
	}
}

func (s *Server) prepareBuildSource(ctx context.Context, config ContainerBuildConfig, gitBranch, gitCommit string) (string, error) {
	switch config.SourceType {
	case "local":
		return config.Source, nil
		
	case "git":
		return s.cloneGitRepository(ctx, config.Source, gitBranch, gitCommit)
		
	case "url":
		return s.downloadAndExtractArchive(ctx, config.Source)
		
	default:
		return "", fmt.Errorf("unsupported source type: %s", config.SourceType)
	}
}

func (s *Server) cloneGitRepository(ctx context.Context, repoURL, branch, commit string) (string, error) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "mcp-build-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}

	// Clone repository
	cloneCmd := exec.CommandContext(ctx, "git", "clone")
	if branch != "" && commit == "" {
		cloneCmd.Args = append(cloneCmd.Args, "--branch", branch)
	}
	cloneCmd.Args = append(cloneCmd.Args, repoURL, tempDir)

	if err := cloneCmd.Run(); err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to clone repository: %v", err)
	}

	// Checkout specific commit if provided
	if commit != "" {
		checkoutCmd := exec.CommandContext(ctx, "git", "-C", tempDir, "checkout", commit)
		if err := checkoutCmd.Run(); err != nil {
			os.RemoveAll(tempDir)
			return "", fmt.Errorf("failed to checkout commit %s: %v", commit, err)
		}
	}

	return tempDir, nil
}

func (s *Server) downloadAndExtractArchive(ctx context.Context, archiveURL string) (string, error) {
	tempDir, err := os.MkdirTemp("", "mcp-build-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}

	downloadCmd := exec.CommandContext(ctx, "curl", "-L", "-o", filepath.Join(tempDir, "archive"), archiveURL)
	if err := downloadCmd.Run(); err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to download archive: %v", err)
	}

	return tempDir, nil
}

func (s *Server) constructBuildCommand(runtime string, config ContainerBuildConfig, buildDir string, noCache, pull bool) *exec.Cmd {
	args := []string{"build"}
	
	if noCache {
		args = append(args, "--no-cache")
	}
	
	if pull {
		args = append(args, "--pull")
	}
	
	if config.Platform != "" {
		args = append(args, "--platform", config.Platform)
	}
	
	if config.Dockerfile != "" {
		dockerfilePath := filepath.Join(buildDir, config.BuildContext, config.Dockerfile)
		args = append(args, "-f", dockerfilePath)
	}
	
	// Add build args
	for key, value := range config.BuildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
	}
	
	// Tag the image
	args = append(args, "-t", config.ImageName)
	
	// Add additional tags
	for _, tag := range config.Tags {
		taggedName := addTagToImage(config.ImageName, tag)
		args = append(args, "-t", taggedName)
	}
	
	// Build context
	contextPath := filepath.Join(buildDir, config.BuildContext)
	args = append(args, contextPath)
	
	return exec.Command(runtime, args...)
}

func (s *Server) executeBuildCommand(ctx context.Context, cmd *exec.Cmd) (string, error) {
	// Set up pipes for real-time output capture
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", err
	}

	// Capture output
	var output strings.Builder
	done := make(chan error, 1)
	
	go func() {
		defer close(done)
		
		// Combine stdout and stderr
		combined := io.MultiReader(stdout, stderr)
		scanner := bufio.NewScanner(combined)
		
		for scanner.Scan() {
			line := scanner.Text()
			output.WriteString(line + "\n")
			klog.V(3).Info("Build: " + line)
		}
		
		done <- scanner.Err()
	}()

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		return output.String(), err
	}

	// Wait for output capture to complete
	if err := <-done; err != nil {
		return output.String(), err
	}

	return output.String(), nil
}

func (s *Server) getImageInfo(ctx context.Context, runtime, imageName string) (*ContainerImageInfo, error) {
	cmd := exec.CommandContext(ctx, runtime, "inspect", imageName)
	_, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return &ContainerImageInfo{
		ImageName: imageName,
		Tags:      []string{extractTagFromImage(imageName)},
		CreatedAt: time.Now(),
		Size:      "unknown",
	}, nil
}

func (s *Server) authenticateRegistry(ctx context.Context, runtime, registry, username, password string) error {
	cmd := exec.CommandContext(ctx, runtime, "login", "--username", username, "--password-stdin", registry)
	cmd.Stdin = strings.NewReader(password)
	
	return cmd.Run()
}

func (s *Server) pushSingleImage(ctx context.Context, runtime, imageName string, skipTLSVerify bool) (map[string]interface{}, error) {
	args := []string{"push"}
	if skipTLSVerify {
		args = append(args, "--tls-verify=false")
	}
	args = append(args, imageName)
	
	cmd := exec.CommandContext(ctx, runtime, args...)
	output, err := cmd.CombinedOutput()
	
	result := map[string]interface{}{
		"image":  imageName,
		"output": string(output),
	}
	
	if err != nil {
		result["error"] = err.Error()
		return result, err
	}
	
	result["status"] = "success"
	return result, nil
}

func (s *Server) tagImage(ctx context.Context, runtime, sourceImage, targetImage string) error {
	cmd := exec.CommandContext(ctx, runtime, "tag", sourceImage, targetImage)
	return cmd.Run()
}

// Utility functions

func extractTagFromImage(imageName string) string {
	parts := strings.Split(imageName, ":")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return "latest"
}

func addTagToImage(imageName, tag string) string {
	parts := strings.Split(imageName, ":")
	if len(parts) > 1 {
		// Replace existing tag
		return strings.Join(parts[:len(parts)-1], ":") + ":" + tag
	}
	// Add tag
	return imageName + ":" + tag
}

func parseImageLine(line string) map[string]interface{} {
	fields := strings.Fields(line)
	if len(fields) >= 5 {
		return map[string]interface{}{
			"repository": fields[0],
			"tag":        fields[1],
			"image_id":   fields[2],
			"created":    fields[3],
			"size":       fields[4],
		}
	}
	return map[string]interface{}{"raw": line}
}

func parseJSON(jsonStr string, v interface{}) error {
	return json.Unmarshal([]byte(jsonStr), v)
}


