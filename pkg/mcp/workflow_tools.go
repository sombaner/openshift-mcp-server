package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"k8s.io/klog/v2"
)

// initWorkflowTools initializes workflow-related MCP tools
func (s *Server) initWorkflowTools() []server.ServerTool {
	klog.V(1).Info("Initializing workflow orchestration tools")

	return []server.ServerTool{
		{Tool: mcp.NewTool("workflow_execute",
			mcp.WithDescription("Intelligently analyze a user prompt and execute the appropriate workflow automatically. This tool can understand natural language requests and chain together multiple container operations like build, push, and deploy based on the intent."),
			mcp.WithString("prompt", mcp.Description("Natural language description of what you want to accomplish. Examples: 'Build and push my app from https://github.com/user/repo.git to quay.io/user/app:latest', 'Deploy container from source to production namespace', 'Scan my container image for security vulnerabilities'."), mcp.Required()),
			mcp.WithString("source", mcp.Description("Source code location (Git repository URL, local path, or archive URL). Auto-extracted from prompt if not provided.")),
			mcp.WithString("image_name", mcp.Description("Target container image name with registry and tag. Auto-extracted from prompt if not provided.")),
			mcp.WithString("registry", mcp.Description("Container registry URL. Auto-extracted from image_name or prompt if not provided.")),
			mcp.WithString("namespace", mcp.Description("Kubernetes/OpenShift namespace for deployment. Auto-extracted from prompt if not provided.")),
			mcp.WithString("workflow", mcp.Description("Force a specific workflow instead of auto-detection. Available: 'build_and_push', 'complete_cicd', 'security_scan', 'registry_management'.")),
			mcp.WithBoolean("dry_run", mcp.Description("Analyze the prompt and show what would be executed without actually running the workflow. Defaults to false.")),
			mcp.WithBoolean("interactive", mcp.Description("Enable interactive mode for parameter confirmation. Defaults to false.")),
			// Tool annotations
			mcp.WithTitleAnnotation("Workflow: Intelligent Container Operations"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.workflowExecute},

		{Tool: mcp.NewTool("workflow_list",
			mcp.WithDescription("List all available workflows with their descriptions and capabilities. Helps users understand what automated workflows are available for container operations."),
			mcp.WithString("category", mcp.Description("Filter workflows by category: 'build', 'deploy', 'security', 'management'. Shows all if not specified.")),
			mcp.WithString("format", mcp.Description("Output format: 'table' (default), 'json', 'detailed'. Table for human reading, JSON for programmatic use.")),
			// Tool annotations
			mcp.WithTitleAnnotation("Workflow: List Available Workflows"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
		), Handler: s.workflowList},

		{Tool: mcp.NewTool("workflow_analyze",
			mcp.WithDescription("Analyze a user prompt to understand intent and show which workflow would be executed with what parameters. Useful for understanding automation capabilities without executing anything."),
			mcp.WithString("prompt", mcp.Description("Natural language description to analyze. Examples: 'I want to containerize my app and deploy it', 'Check my image for security issues', 'Build from Git and push to registry'."), mcp.Required()),
			mcp.WithBoolean("show_parameters", mcp.Description("Show extracted parameters and how they would be used. Defaults to true.")),
			mcp.WithBoolean("show_confidence", mcp.Description("Show confidence scores for workflow matching. Defaults to false.")),
			// Tool annotations
			mcp.WithTitleAnnotation("Workflow: Analyze User Intent"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
		), Handler: s.workflowAnalyze},

		{Tool: mcp.NewTool("workflow_create",
			mcp.WithDescription("Create a custom workflow by defining a sequence of container operations. Allows users to create reusable automation for their specific use cases."),
			mcp.WithString("name", mcp.Description("Unique name for the workflow. Use lowercase with underscores. Example: 'my_custom_build_flow'."), mcp.Required()),
			mcp.WithString("description", mcp.Description("Human-readable description of what this workflow does."), mcp.Required()),
			mcp.WithString("keywords", mcp.Description("Comma-separated keywords that trigger this workflow. Example: 'build,test,deploy,custom'.")),
			mcp.WithString("steps", mcp.Description("JSON array of workflow steps. Each step should have 'tool', 'description', and 'parameters' fields."), mcp.Required()),
			mcp.WithString("conditions", mcp.Description("JSON array of trigger conditions. Each condition should have 'type', 'pattern', 'required', and 'confidence' fields.")),
			// Tool annotations
			mcp.WithTitleAnnotation("Workflow: Create Custom Workflow"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
		), Handler: s.workflowCreate},
	}
}

// workflowExecute handles intelligent workflow execution based on user prompts
func (s *Server) workflowExecute(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return NewTextResult("", fmt.Errorf("invalid arguments format")), nil
	}

	prompt, ok := args["prompt"].(string)
	if !ok || prompt == "" {
		return NewTextResult("", fmt.Errorf("prompt parameter is required")), nil
	}

	// Get optional parameters
	forcedWorkflow := getStringArg(args, "workflow", "")
	dryRun := getBoolArg(args, "dry_run", false)
	interactive := getBoolArg(args, "interactive", false)

	// Initialize workflow orchestrator if not already done
	if s.workflowOrchestrator == nil {
		s.workflowOrchestrator = NewWorkflowOrchestrator(s)
	}

	klog.V(1).Infof("Executing workflow for prompt: %s", prompt)

	// Override parameters from user input
	userParams := make(map[string]interface{})
	if source := getStringArg(args, "source", ""); source != "" {
		userParams["source"] = source
	}
	if imageName := getStringArg(args, "image_name", ""); imageName != "" {
		userParams["image_name"] = imageName
	}
	if registry := getStringArg(args, "registry", ""); registry != "" {
		userParams["registry"] = registry
	}
	if namespace := getStringArg(args, "namespace", ""); namespace != "" {
		userParams["namespace"] = namespace
	}

	var selectedWorkflow *Workflow
	var extractedParams map[string]interface{}
	var err error

	if forcedWorkflow != "" {
		// Use forced workflow
		var exists bool
		selectedWorkflow, exists = s.workflowOrchestrator.GetWorkflow(forcedWorkflow)
		if !exists {
			return NewTextResult("", fmt.Errorf("workflow not found: %s", forcedWorkflow)), nil
		}
		extractedParams = s.workflowOrchestrator.extractParametersFromPrompt(prompt)
	} else {
		// Analyze prompt to determine workflow
		selectedWorkflow, extractedParams, err = s.workflowOrchestrator.AnalyzePrompt(prompt)
		if err != nil {
			return NewTextResult("", fmt.Errorf("failed to analyze prompt: %v", err)), nil
		}
	}

	// Merge user parameters with extracted parameters
	for k, v := range userParams {
		extractedParams[k] = v
	}

	if dryRun {
		// Return analysis without execution
		result := map[string]interface{}{
			"mode":                 "dry_run",
			"selected_workflow":    selectedWorkflow.Name,
			"workflow_description": selectedWorkflow.Description,
			"extracted_parameters": extractedParams,
			"steps_to_execute":     selectedWorkflow.Steps,
			"message":              fmt.Sprintf("Would execute workflow '%s' with the parameters shown above", selectedWorkflow.Name),
		}
		jsonResult, _ := json.MarshalIndent(result, "", "  ")
		return NewTextResult(string(jsonResult), nil), nil
	}

	if interactive {
		// In a real implementation, this would prompt for confirmation
		klog.V(1).Info("Interactive mode - would prompt for parameter confirmation")
	}

	// Execute the workflow
	workflowResult, err := s.workflowOrchestrator.ExecuteWorkflow(ctx, selectedWorkflow, extractedParams)
	if err != nil {
		return NewTextResult("", fmt.Errorf("workflow execution failed: %v", err)), nil
	}

	jsonResult, _ := json.MarshalIndent(workflowResult, "", "  ")
	return NewTextResult(string(jsonResult), nil), nil
}

// workflowList handles listing available workflows
func (s *Server) workflowList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		args = make(map[string]interface{})
	}

	category := getStringArg(args, "category", "")
	format := getStringArg(args, "format", "table")

	// Initialize workflow orchestrator if not already done
	if s.workflowOrchestrator == nil {
		s.workflowOrchestrator = NewWorkflowOrchestrator(s)
	}

	workflows := s.workflowOrchestrator.ListWorkflows()
	filteredWorkflows := make(map[string]*Workflow)

	// Filter by category if specified
	for name, workflow := range workflows {
		if category == "" {
			filteredWorkflows[name] = workflow
		} else {
			// Simple category matching based on keywords
			for _, keyword := range workflow.Keywords {
				if strings.Contains(keyword, category) {
					filteredWorkflows[name] = workflow
					break
				}
			}
		}
	}

	if format == "json" {
		result := map[string]interface{}{
			"workflows": filteredWorkflows,
			"total":     len(filteredWorkflows),
			"category":  category,
		}
		jsonResult, _ := json.MarshalIndent(result, "", "  ")
		return NewTextResult(string(jsonResult), nil), nil
	}

	// Format as table
	result := "Available Container Workflows:\n"
	result += strings.Repeat("=", 50) + "\n\n"

	for _, workflow := range filteredWorkflows {
		result += fmt.Sprintf("🔧 %s\n", workflow.Name)
		result += fmt.Sprintf("   Description: %s\n", workflow.Description)
		result += fmt.Sprintf("   Keywords: %s\n", strings.Join(workflow.Keywords, ", "))
		result += fmt.Sprintf("   Steps: %d\n", len(workflow.Steps))
		result += "\n"
	}

	result += fmt.Sprintf("Total: %d workflows available\n", len(filteredWorkflows))
	if category != "" {
		result += fmt.Sprintf("Filtered by category: %s\n", category)
	}

	return NewTextResult(result, nil), nil
}

// workflowAnalyze handles prompt analysis without execution
func (s *Server) workflowAnalyze(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return NewTextResult("", fmt.Errorf("invalid arguments format")), nil
	}

	prompt, ok := args["prompt"].(string)
	if !ok || prompt == "" {
		return NewTextResult("", fmt.Errorf("prompt parameter is required")), nil
	}

	showParameters := getBoolArg(args, "show_parameters", true)
	showConfidence := getBoolArg(args, "show_confidence", false)

	// Initialize workflow orchestrator if not already done
	if s.workflowOrchestrator == nil {
		s.workflowOrchestrator = NewWorkflowOrchestrator(s)
	}

	// Analyze the prompt
	selectedWorkflow, extractedParams, err := s.workflowOrchestrator.AnalyzePrompt(prompt)
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to analyze prompt: %v", err)), nil
	}

	result := map[string]interface{}{
		"prompt":               prompt,
		"selected_workflow":    selectedWorkflow.Name,
		"workflow_description": selectedWorkflow.Description,
		"analysis":             "Successfully identified workflow intent",
	}

	if showParameters {
		result["extracted_parameters"] = extractedParams
		result["workflow_steps"] = selectedWorkflow.Steps
	}

	if showConfidence {
		// Calculate confidence scores for all workflows
		confidence := make(map[string]int)
		workflows := s.workflowOrchestrator.ListWorkflows()
		for name, workflow := range workflows {
			score := s.workflowOrchestrator.scoreWorkflow(strings.ToLower(prompt), workflow)
			confidence[name] = score
		}
		result["confidence_scores"] = confidence
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return NewTextResult(string(jsonResult), nil), nil
}

// workflowCreate handles custom workflow creation
func (s *Server) workflowCreate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return NewTextResult("", fmt.Errorf("invalid arguments format")), nil
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return NewTextResult("", fmt.Errorf("name parameter is required")), nil
	}

	description, ok := args["description"].(string)
	if !ok || description == "" {
		return NewTextResult("", fmt.Errorf("description parameter is required")), nil
	}

	stepsJSON, ok := args["steps"].(string)
	if !ok || stepsJSON == "" {
		return NewTextResult("", fmt.Errorf("steps parameter is required")), nil
	}

	// Parse workflow components
	var steps []WorkflowStep
	if err := json.Unmarshal([]byte(stepsJSON), &steps); err != nil {
		return NewTextResult("", fmt.Errorf("invalid steps JSON: %v", err)), nil
	}

	// Parse keywords
	keywordsStr := getStringArg(args, "keywords", "")
	var keywords []string
	if keywordsStr != "" {
		keywords = strings.Split(keywordsStr, ",")
		for i, keyword := range keywords {
			keywords[i] = strings.TrimSpace(keyword)
		}
	}

	// Parse conditions
	conditionsJSON := getStringArg(args, "conditions", "[]")
	var conditions []WorkflowCondition
	if err := json.Unmarshal([]byte(conditionsJSON), &conditions); err != nil {
		return NewTextResult("", fmt.Errorf("invalid conditions JSON: %v", err)), nil
	}

	// Create workflow
	workflow := &Workflow{
		Name:        name,
		Description: description,
		Keywords:    keywords,
		Steps:       steps,
		Conditions:  conditions,
	}

	// Initialize workflow orchestrator if not already done
	if s.workflowOrchestrator == nil {
		s.workflowOrchestrator = NewWorkflowOrchestrator(s)
	}

	// Add the workflow
	s.workflowOrchestrator.AddCustomWorkflow(workflow)

	result := map[string]interface{}{
		"status":               "success",
		"message":              fmt.Sprintf("Custom workflow '%s' created successfully", name),
		"workflow_name":        name,
		"workflow_description": description,
		"steps_count":          len(steps),
		"keywords":             keywords,
		"conditions_count":     len(conditions),
	}

	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return NewTextResult(string(jsonResult), nil), nil
}
