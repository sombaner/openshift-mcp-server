package mcp

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/klog/v2"
)

// UBIValidation represents UBI validation results
type UBIValidation struct {
	IsUBI              bool     `json:"is_ubi"`
	CurrentBaseImage   string   `json:"current_base_image"`
	SuggestedUBIImage  string   `json:"suggested_ubi_image"`
	SecurityBenefits   []string `json:"security_benefits"`
	ComplianceBenefits []string `json:"compliance_benefits"`
	ValidationMessage  string   `json:"validation_message"`
}

// Red Hat UBI image mappings
var ubiImageMappings = map[string]string{
	// Standard OS bases
	"alpine":              "registry.access.redhat.com/ubi8/ubi-minimal:latest",
	"ubuntu":              "registry.access.redhat.com/ubi8/ubi:latest",
	"centos":              "registry.access.redhat.com/ubi8/ubi:latest",
	"debian":              "registry.access.redhat.com/ubi8/ubi:latest",
	"fedora":              "registry.access.redhat.com/ubi8/ubi:latest",
	"rhel":                "registry.access.redhat.com/ubi8/ubi:latest",
	
	// Language-specific bases
	"node":                "registry.access.redhat.com/ubi8/nodejs-18:latest",
	"python":              "registry.access.redhat.com/ubi8/python-39:latest",
	"golang":              "registry.access.redhat.com/ubi8/go-toolset:latest",
	"openjdk":             "registry.access.redhat.com/ubi8/openjdk-11:latest",
	"java":                "registry.access.redhat.com/ubi8/openjdk-11:latest",
	"nginx":               "registry.access.redhat.com/ubi8/nginx-120:latest",
	"httpd":               "registry.access.redhat.com/ubi8/httpd-24:latest",
	"php":                 "registry.access.redhat.com/ubi8/php-74:latest",
	"ruby":                "registry.access.redhat.com/ubi8/ruby-27:latest",
	"postgres":            "registry.access.redhat.com/rhel8/postgresql-13:latest",
	"mysql":               "registry.access.redhat.com/rhel8/mysql-80:latest",
	"redis":               "registry.access.redhat.com/rhel8/redis-6:latest",
	
	// Default fallback
	"default":             "registry.access.redhat.com/ubi8/ubi:latest",
}

// Red Hat UBI registry patterns
var ubiRegistryPatterns = []string{
	"registry.access.redhat.com/ubi",
	"registry.redhat.io/ubi",
	"quay.io/redhat/ubi",
}

// validateUBICompliance checks if the Dockerfile uses Red Hat UBI base images
func (s *Server) validateUBICompliance(ctx context.Context, dockerfilePath string) (*UBIValidation, error) {
	klog.V(2).Infof("Validating UBI compliance for Dockerfile: %s", dockerfilePath)
	
	// Read Dockerfile
	file, err := os.Open(dockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Dockerfile: %v", err)
	}
	defer file.Close()

	var baseImage string
	scanner := bufio.NewScanner(file)
	
	// Find the first FROM instruction
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(strings.ToUpper(line), "FROM ") {
			// Extract base image (handle multi-stage builds)
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				baseImage = parts[1]
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading Dockerfile: %v", err)
	}

	if baseImage == "" {
		return nil, fmt.Errorf("no FROM instruction found in Dockerfile")
	}

	// Check if current image is UBI-based
	isUBI := s.isUBIImage(baseImage)
	suggestedUBI := s.suggestUBIAlternative(baseImage)
	
	validation := &UBIValidation{
		IsUBI:             isUBI,
		CurrentBaseImage:  baseImage,
		SuggestedUBIImage: suggestedUBI,
		SecurityBenefits: []string{
			"Enhanced security with Red Hat's security patches",
			"Regular vulnerability scanning and updates",
			"FIPS 140-2 compliance support",
			"Reduced attack surface with minimal base",
		},
		ComplianceBenefits: []string{
			"Enterprise-grade support and SLAs",
			"GPL-free licensing for commercial use",
			"SOC 2 and ISO 27001 compliance",
			"OpenShift and Kubernetes optimized",
		},
	}

	if isUBI {
		validation.ValidationMessage = fmt.Sprintf("✅ Base image '%s' is Red Hat UBI compliant", baseImage)
	} else {
		validation.ValidationMessage = fmt.Sprintf("⚠️ Base image '%s' is not Red Hat UBI. Suggested alternative: '%s'", baseImage, suggestedUBI)
	}

	return validation, nil
}

// isUBIImage checks if the given image is a Red Hat UBI image
func (s *Server) isUBIImage(image string) bool {
	imageLower := strings.ToLower(image)
	
	for _, pattern := range ubiRegistryPatterns {
		if strings.Contains(imageLower, pattern) {
			return true
		}
	}
	
	return false
}

// suggestUBIAlternative suggests a Red Hat UBI alternative for the given base image
func (s *Server) suggestUBIAlternative(image string) string {
	imageLower := strings.ToLower(image)
	
	// Extract image name without tag and registry
	imageName := s.extractImageName(imageLower)
	
	// Check for exact matches first
	if ubiImage, exists := ubiImageMappings[imageName]; exists {
		return ubiImage
	}
	
	// Check for partial matches
	for pattern, ubiImage := range ubiImageMappings {
		if strings.Contains(imageName, pattern) {
			return ubiImage
		}
	}
	
	// Return default UBI image
	return ubiImageMappings["default"]
}

// extractImageName extracts the image name from a full image reference
func (s *Server) extractImageName(image string) string {
	// Remove registry part if present
	parts := strings.Split(image, "/")
	imagePart := parts[len(parts)-1]
	
	// Remove tag if present
	if colonIndex := strings.Index(imagePart, ":"); colonIndex != -1 {
		imagePart = imagePart[:colonIndex]
	}
	
	// Remove digest if present
	if atIndex := strings.Index(imagePart, "@"); atIndex != -1 {
		imagePart = imagePart[:atIndex]
	}
	
	return imagePart
}

// generateUBIDockerfile generates a new Dockerfile with UBI base image
func (s *Server) generateUBIDockerfile(ctx context.Context, originalDockerfile, suggestedUBI string) (string, error) {
	// Read original Dockerfile
	content, err := os.ReadFile(originalDockerfile)
	if err != nil {
		return "", fmt.Errorf("failed to read original Dockerfile: %v", err)
	}

	// Replace the FROM instruction
	lines := strings.Split(string(content), "\n")
	var newLines []string
	
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(trimmedLine), "FROM ") {
			// Replace with UBI image
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				// Preserve any alias (AS name)
				if len(parts) >= 4 && strings.ToUpper(parts[2]) == "AS" {
					newLines = append(newLines, fmt.Sprintf("FROM %s AS %s", suggestedUBI, parts[3]))
				} else {
					newLines = append(newLines, fmt.Sprintf("FROM %s", suggestedUBI))
				}
			} else {
				newLines = append(newLines, line)
			}
		} else {
			newLines = append(newLines, line)
		}
	}
	
	// Write to new file
	newDockerfilePath := filepath.Join(filepath.Dir(originalDockerfile), "Dockerfile.ubi")
	newContent := strings.Join(newLines, "\n")
	
	err = os.WriteFile(newDockerfilePath, []byte(newContent), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write UBI Dockerfile: %v", err)
	}
	
	return newDockerfilePath, nil
}

// validateDockerfileForSecurity performs additional security validations
func (s *Server) validateDockerfileForSecurity(ctx context.Context, dockerfilePath string) ([]string, []string, error) {
	var warnings []string
	var recommendations []string
	
	content, err := os.ReadFile(dockerfilePath)
	if err != nil {
		return warnings, recommendations, fmt.Errorf("failed to read Dockerfile: %v", err)
	}
	
	lines := strings.Split(string(content), "\n")
	
	// Security validation patterns
	for i, line := range lines {
		lineNum := i + 1
		trimmedLine := strings.TrimSpace(line)
		upperLine := strings.ToUpper(trimmedLine)
		
		// Check for root user
		if strings.Contains(upperLine, "USER ROOT") || strings.Contains(upperLine, "USER 0") {
			warnings = append(warnings, fmt.Sprintf("Line %d: Running as root user detected", lineNum))
			recommendations = append(recommendations, "Consider creating and using a non-root user")
		}
		
		// Check for missing USER instruction
		if upperLine == "USER" {
			recommendations = append(recommendations, "Add a USER instruction to run container as non-root")
		}
		
		// Check for package manager without cleanup
		if (strings.Contains(upperLine, "APT-GET") || strings.Contains(upperLine, "YUM") || strings.Contains(upperLine, "DNF")) &&
		   !strings.Contains(upperLine, "CLEAN") && !strings.Contains(upperLine, "REMOVE") {
			warnings = append(warnings, fmt.Sprintf("Line %d: Package installation without cleanup", lineNum))
			recommendations = append(recommendations, "Clean package cache after installation to reduce image size")
		}
		
		// Check for secrets in environment variables
		secretPatterns := []string{"PASSWORD", "SECRET", "KEY", "TOKEN", "CREDENTIAL"}
		for _, pattern := range secretPatterns {
			if strings.Contains(upperLine, "ENV") && strings.Contains(upperLine, pattern) {
				warnings = append(warnings, fmt.Sprintf("Line %d: Potential secret in ENV instruction", lineNum))
				recommendations = append(recommendations, "Use secrets management instead of ENV for sensitive data")
			}
		}
		
		// Check for COPY/ADD with broad patterns
		if (strings.Contains(upperLine, "COPY .") || strings.Contains(upperLine, "ADD .")) &&
		   !strings.Contains(trimmedLine, ".dockerignore") {
			warnings = append(warnings, fmt.Sprintf("Line %d: Copying entire context", lineNum))
			recommendations = append(recommendations, "Use specific COPY instructions and .dockerignore to reduce context")
		}
	}
	
	return warnings, recommendations, nil
}

// Enhanced container build validation with UBI compliance
func (s *Server) enhancedContainerBuildValidation(ctx context.Context, config ContainerBuildConfig, buildDir string) (map[string]interface{}, error) {
	dockerfilePath := filepath.Join(buildDir, config.BuildContext, config.Dockerfile)
	
	// UBI compliance validation
	ubiValidation, err := s.validateUBICompliance(ctx, dockerfilePath)
	if err != nil {
		klog.V(1).Infof("UBI validation failed: %v", err)
		ubiValidation = &UBIValidation{
			IsUBI:             false,
			ValidationMessage: fmt.Sprintf("Could not validate UBI compliance: %v", err),
		}
	}
	
	// Security validation
	warnings, recommendations, err := s.validateDockerfileForSecurity(ctx, dockerfilePath)
	if err != nil {
		klog.V(1).Infof("Security validation failed: %v", err)
	}
	
	validation := map[string]interface{}{
		"ubi_compliance":        ubiValidation,
		"security_warnings":     warnings,
		"security_recommendations": recommendations,
		"dockerfile_path":       dockerfilePath,
	}
	
	// If not UBI compliant, offer to generate UBI Dockerfile
	if !ubiValidation.IsUBI && ubiValidation.SuggestedUBIImage != "" {
		validation["ubi_dockerfile_available"] = true
		validation["suggested_actions"] = []string{
			fmt.Sprintf("Consider switching to Red Hat UBI: %s", ubiValidation.SuggestedUBIImage),
			"UBI images provide enterprise-grade security and compliance",
			"Use the container_build tool with 'generate_ubi_dockerfile=true' to create UBI version",
		}
	}
	
	return validation, nil
}
