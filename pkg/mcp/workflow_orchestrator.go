package mcp

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"k8s.io/klog/v2"
)

// WorkflowOrchestrator manages intelligent tool invocation based on user prompts
type WorkflowOrchestrator struct {
	server    *Server
	workflows map[string]*Workflow
}

// Workflow represents a sequence of tool invocations
type Workflow struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Keywords    []string            `json:"keywords"`
	Steps       []WorkflowStep      `json:"steps"`
	Conditions  []WorkflowCondition `json:"conditions"`
}

// WorkflowStep represents a single step in a workflow
type WorkflowStep struct {
	Tool        string                 `json:"tool"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Conditional bool                   `json:"conditional"`
	OnSuccess   []WorkflowStep         `json:"on_success"`
	OnFailure   []WorkflowStep         `json:"on_failure"`
}

// WorkflowCondition defines when a workflow should be triggered
type WorkflowCondition struct {
	Type       string `json:"type"` // "keyword", "regex", "context"
	Pattern    string `json:"pattern"`
	Required   bool   `json:"required"`
	Confidence int    `json:"confidence"` // 0-100
}

// WorkflowResult contains the results of workflow execution
type WorkflowResult struct {
	WorkflowName    string               `json:"workflow_name"`
	ExecutedSteps   []WorkflowStepResult `json:"executed_steps"`
	Success         bool                 `json:"success"`
	Error           string               `json:"error,omitempty"`
	Duration        time.Duration        `json:"duration"`
	Recommendations []string             `json:"recommendations"`
}

// WorkflowStepResult contains the result of a single workflow step
type WorkflowStepResult struct {
	Tool       string                 `json:"tool"`
	Parameters map[string]interface{} `json:"parameters"`
	Result     *mcp.CallToolResult    `json:"result"`
	Success    bool                   `json:"success"`
	Error      string                 `json:"error,omitempty"`
	Duration   time.Duration          `json:"duration"`
}

// NewWorkflowOrchestrator creates a new workflow orchestrator
func NewWorkflowOrchestrator(server *Server) *WorkflowOrchestrator {
	wo := &WorkflowOrchestrator{
		server:    server,
		workflows: make(map[string]*Workflow),
	}

	// Initialize built-in workflows
	wo.initializeBuiltInWorkflows()

	return wo
}

// initializeBuiltInWorkflows sets up common container workflows
func (wo *WorkflowOrchestrator) initializeBuiltInWorkflows() {
	// Build and Push Workflow
	wo.workflows["build_and_push"] = &Workflow{
		Name:        "Build and Push Container",
		Description: "Build a container image from source and push to registry",
		Keywords:    []string{"build", "push", "deploy", "container", "image", "registry"},
		Conditions: []WorkflowCondition{
			{Type: "keyword", Pattern: "build.*push|deploy.*container|containerize.*deploy", Required: true, Confidence: 90},
			{Type: "context", Pattern: "source.*registry|git.*registry|dockerfile.*registry", Required: false, Confidence: 80},
		},
		Steps: []WorkflowStep{
			{
				Tool:        "container_build",
				Description: "Build container image from source",
				Parameters: map[string]interface{}{
					"validate_ubi":  true,
					"security_scan": true,
				},
				OnSuccess: []WorkflowStep{
					{
						Tool:        "container_push",
						Description: "Push built image to registry",
						Parameters:  map[string]interface{}{},
					},
				},
			},
		},
	}

	// Complete CI/CD Workflow
	wo.workflows["complete_cicd"] = &Workflow{
		Name:        "Complete CI/CD Pipeline",
		Description: "Full CI/CD pipeline: build, test, push, and deploy",
		Keywords:    []string{"cicd", "pipeline", "full", "complete", "end-to-end"},
		Conditions: []WorkflowCondition{
			{Type: "keyword", Pattern: "full.*pipeline|complete.*cicd|end.*end", Required: true, Confidence: 95},
		},
		Steps: []WorkflowStep{
			{
				Tool:        "container_build",
				Description: "Build container image",
				Parameters: map[string]interface{}{
					"validate_ubi":  true,
					"security_scan": true,
				},
				OnSuccess: []WorkflowStep{
					{
						Tool:        "container_push",
						Description: "Push to registry",
						OnSuccess: []WorkflowStep{
							{
								Tool:        "repo_auto_deploy",
								Description: "Deploy to OpenShift",
							},
						},
					},
				},
			},
		},
	}

	// Security Scan Workflow
	wo.workflows["security_scan"] = &Workflow{
		Name:        "Container Security Scan",
		Description: "Comprehensive security scanning of container images",
		Keywords:    []string{"security", "scan", "vulnerability", "audit", "compliance"},
		Conditions: []WorkflowCondition{
			{Type: "keyword", Pattern: "security.*scan|vulnerability.*check|audit.*container", Required: true, Confidence: 90},
		},
		Steps: []WorkflowStep{
			{
				Tool:        "container_inspect",
				Description: "Inspect container image",
				Parameters: map[string]interface{}{
					"format": "security",
				},
			},
		},
	}

	// Registry Management Workflow
	wo.workflows["registry_management"] = &Workflow{
		Name:        "Registry Management",
		Description: "Manage container registries and images",
		Keywords:    []string{"registry", "manage", "list", "clean", "prune"},
		Conditions: []WorkflowCondition{
			{Type: "keyword", Pattern: "manage.*registry|list.*images|clean.*registry", Required: true, Confidence: 85},
		},
		Steps: []WorkflowStep{
			{
				Tool:        "container_list",
				Description: "List container images",
				Parameters: map[string]interface{}{
					"format": "json",
				},
			},
		},
	}
}

// AnalyzePrompt analyzes a user prompt to determine the appropriate workflow
func (wo *WorkflowOrchestrator) AnalyzePrompt(prompt string) (*Workflow, map[string]interface{}, error) {
	prompt = strings.ToLower(prompt)
	bestMatch := ""
	bestScore := 0
	extractedParams := make(map[string]interface{})

	// Extract common parameters from prompt
	extractedParams = wo.extractParametersFromPrompt(prompt)

	// Score each workflow
	for name, workflow := range wo.workflows {
		score := wo.scoreWorkflow(prompt, workflow)
		klog.V(2).Infof("Workflow %s scored %d for prompt: %s", name, score, prompt)

		if score > bestScore {
			bestScore = score
			bestMatch = name
		}
	}

	if bestScore < 50 { // Minimum confidence threshold
		return nil, extractedParams, fmt.Errorf("no suitable workflow found for prompt (best score: %d)", bestScore)
	}

	selectedWorkflow := wo.workflows[bestMatch]
	klog.V(1).Infof("Selected workflow: %s (score: %d)", selectedWorkflow.Name, bestScore)

	return selectedWorkflow, extractedParams, nil
}

// scoreWorkflow calculates a confidence score for a workflow based on the prompt
func (wo *WorkflowOrchestrator) scoreWorkflow(prompt string, workflow *Workflow) int {
	score := 0

	// Check keywords
	for _, keyword := range workflow.Keywords {
		if strings.Contains(prompt, keyword) {
			score += 15
		}
	}

	// Check conditions
	for _, condition := range workflow.Conditions {
		matched := false
		switch condition.Type {
		case "keyword":
			if strings.Contains(prompt, condition.Pattern) {
				matched = true
			}
		case "regex":
			if matched, _ := regexp.MatchString(condition.Pattern, prompt); matched {
				matched = true
			}
		case "context":
			// Simple context matching
			parts := strings.Split(condition.Pattern, ".*")
			allFound := true
			for _, part := range parts {
				if !strings.Contains(prompt, part) {
					allFound = false
					break
				}
			}
			matched = allFound
		}

		if matched {
			if condition.Required {
				score += condition.Confidence
			} else {
				score += condition.Confidence / 2
			}
		} else if condition.Required {
			score -= 30 // Penalty for missing required conditions
		}
	}

	return score
}

// extractParametersFromPrompt extracts common parameters from the user prompt
func (wo *WorkflowOrchestrator) extractParametersFromPrompt(prompt string) map[string]interface{} {
	params := make(map[string]interface{})

	// Extract Git repository URLs
	gitRegex := regexp.MustCompile(`https?://[^\s]+\.git|git@[^\s]+\.git`)
	if matches := gitRegex.FindStringSubmatch(prompt); len(matches) > 0 {
		params["source"] = matches[0]
		params["source_type"] = "git"
	}

	// Extract image names
	imageRegex := regexp.MustCompile(`(?:image[:\s]+|docker[:\s]+|container[:\s]+)([a-zA-Z0-9._/-]+:[a-zA-Z0-9._-]+)`)
	if matches := imageRegex.FindStringSubmatch(prompt); len(matches) > 1 {
		params["image_name"] = matches[1]
	}

	// Extract registry URLs
	registryRegex := regexp.MustCompile(`(?:registry[:\s]+|push to[:\s]+)([a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`)
	if matches := registryRegex.FindStringSubmatch(prompt); len(matches) > 1 {
		params["registry"] = matches[1]
	}

	// Extract namespace
	namespaceRegex := regexp.MustCompile(`(?:namespace[:\s]+|deploy to[:\s]+)([a-zA-Z0-9-]+)`)
	if matches := namespaceRegex.FindStringSubmatch(prompt); len(matches) > 1 {
		params["namespace"] = matches[1]
	}

	// Extract dockerfile path
	dockerfileRegex := regexp.MustCompile(`(?:dockerfile[:\s]+|docker file[:\s]+)([a-zA-Z0-9./]+)`)
	if matches := dockerfileRegex.FindStringSubmatch(prompt); len(matches) > 1 {
		params["dockerfile"] = matches[1]
	}

	return params
}

// ExecuteWorkflow executes a workflow with the given parameters
func (wo *WorkflowOrchestrator) ExecuteWorkflow(ctx context.Context, workflow *Workflow, userParams map[string]interface{}) (*WorkflowResult, error) {
	startTime := time.Now()
	result := &WorkflowResult{
		WorkflowName:    workflow.Name,
		ExecutedSteps:   []WorkflowStepResult{},
		Success:         true,
		Recommendations: []string{},
	}

	klog.V(1).Infof("Starting workflow execution: %s", workflow.Name)

	// Execute each step
	for _, step := range workflow.Steps {
		stepResult, err := wo.executeWorkflowStep(ctx, step, userParams)
		result.ExecutedSteps = append(result.ExecutedSteps, *stepResult)

		if err != nil || !stepResult.Success {
			result.Success = false
			result.Error = fmt.Sprintf("Step %s failed: %v", step.Tool, err)
			klog.V(1).Infof("Workflow step failed: %s - %v", step.Tool, err)
			break
		}

		// Execute conditional next steps
		if stepResult.Success && len(step.OnSuccess) > 0 {
			for _, nextStep := range step.OnSuccess {
				nextStepResult, err := wo.executeWorkflowStep(ctx, nextStep, userParams)
				result.ExecutedSteps = append(result.ExecutedSteps, *nextStepResult)

				if err != nil || !nextStepResult.Success {
					result.Success = false
					result.Error = fmt.Sprintf("Conditional step %s failed: %v", nextStep.Tool, err)
					break
				}
			}
		}
	}

	result.Duration = time.Since(startTime)
	klog.V(1).Infof("Workflow execution completed: %s (success: %t, duration: %v)",
		workflow.Name, result.Success, result.Duration)

	// Generate recommendations
	result.Recommendations = wo.generateRecommendations(result)

	return result, nil
}

// executeWorkflowStep executes a single workflow step
func (wo *WorkflowOrchestrator) executeWorkflowStep(ctx context.Context, step WorkflowStep, userParams map[string]interface{}) (*WorkflowStepResult, error) {
	startTime := time.Now()
	stepResult := &WorkflowStepResult{
		Tool:       step.Tool,
		Parameters: make(map[string]interface{}),
	}

	// Merge step parameters with user parameters
	for k, v := range step.Parameters {
		stepResult.Parameters[k] = v
	}
	for k, v := range userParams {
		stepResult.Parameters[k] = v
	}

	klog.V(2).Infof("Executing workflow step: %s with parameters: %v", step.Tool, stepResult.Parameters)

	// Create tool call request
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      step.Tool,
			Arguments: stepResult.Parameters,
		},
	}

	// Find and execute the tool
	toolResult, err := wo.executeTool(ctx, request)
	stepResult.Result = toolResult
	stepResult.Duration = time.Since(startTime)

	if err != nil {
		stepResult.Success = false
		stepResult.Error = err.Error()
		return stepResult, err
	}

	stepResult.Success = !toolResult.IsError
	if toolResult.IsError && len(toolResult.Content) > 0 {
		if textContent, ok := toolResult.Content[0].(mcp.TextContent); ok {
			stepResult.Error = textContent.Text
		}
	}

	return stepResult, nil
}

// executeTool executes a specific MCP tool
func (wo *WorkflowOrchestrator) executeTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// This would normally use the server's tool registry
	// For now, we'll call the tools directly based on the tool name
	switch request.Params.Name {
	case "container_build":
		return wo.server.containerBuild(ctx, request)
	case "container_push":
		return wo.server.containerPush(ctx, request)
	case "container_list":
		return wo.server.containerList(ctx, request)
	case "container_remove":
		return wo.server.containerRemove(ctx, request)
	case "container_inspect":
		return wo.server.containerInspect(ctx, request)
	case "repo_auto_deploy":
		// This would call the repo deployment tool
		return &mcp.CallToolResult{Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Deployment initiated"}}}, nil
	default:
		return nil, fmt.Errorf("unknown tool: %s", request.Params.Name)
	}
}

// generateRecommendations generates recommendations based on workflow results
func (wo *WorkflowOrchestrator) generateRecommendations(result *WorkflowResult) []string {
	recommendations := []string{}

	if result.Success {
		recommendations = append(recommendations, "✅ Workflow completed successfully!")

		// Analyze results for specific recommendations
		for _, step := range result.ExecutedSteps {
			if step.Tool == "container_build" && step.Success {
				recommendations = append(recommendations, "📦 Container image built successfully")
				recommendations = append(recommendations, "🚀 Consider setting up automated CI/CD pipeline")
			}
			if step.Tool == "container_push" && step.Success {
				recommendations = append(recommendations, "📤 Image pushed to registry")
				recommendations = append(recommendations, "🔄 Set up image scanning for security compliance")
			}
		}
	} else {
		recommendations = append(recommendations, "❌ Workflow failed - check the error details above")
		recommendations = append(recommendations, "🔍 Try running individual tools to debug the issue")
	}

	return recommendations
}

// AddCustomWorkflow allows adding custom workflows
func (wo *WorkflowOrchestrator) AddCustomWorkflow(workflow *Workflow) {
	wo.workflows[strings.ToLower(strings.ReplaceAll(workflow.Name, " ", "_"))] = workflow
	klog.V(1).Infof("Added custom workflow: %s", workflow.Name)
}

// ListWorkflows returns all available workflows
func (wo *WorkflowOrchestrator) ListWorkflows() map[string]*Workflow {
	return wo.workflows
}

// GetWorkflow returns a specific workflow by name
func (wo *WorkflowOrchestrator) GetWorkflow(name string) (*Workflow, bool) {
	workflow, exists := wo.workflows[name]
	return workflow, exists
}
