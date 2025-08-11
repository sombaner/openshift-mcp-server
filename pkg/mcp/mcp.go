package mcp

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"k8s.io/klog/v2"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	authenticationapiv1 "k8s.io/api/authentication/v1"
	"k8s.io/utils/ptr"

	"github.com/manusa/kubernetes-mcp-server/pkg/config"
	internalk8s "github.com/manusa/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/manusa/kubernetes-mcp-server/pkg/output"
	"github.com/manusa/kubernetes-mcp-server/pkg/version"
)

type Configuration struct {
	Profile    Profile
	ListOutput output.Output

	StaticConfig *config.StaticConfig
}

func (c *Configuration) isToolApplicable(tool server.ServerTool) bool {
	if c.StaticConfig.ReadOnly && !ptr.Deref(tool.Tool.Annotations.ReadOnlyHint, false) {
		return false
	}
	if c.StaticConfig.DisableDestructive && ptr.Deref(tool.Tool.Annotations.DestructiveHint, false) {
		return false
	}
	if c.StaticConfig.EnabledTools != nil && !slices.Contains(c.StaticConfig.EnabledTools, tool.Tool.Name) {
		return false
	}
	if c.StaticConfig.DisabledTools != nil && slices.Contains(c.StaticConfig.DisabledTools, tool.Tool.Name) {
		return false
	}
	return true
}

type Server struct {
	configuration *Configuration
	server        *server.MCPServer
	k             *internalk8s.Manager
}

func NewServer(configuration Configuration) (*Server, error) {
	s := &Server{
		configuration: &configuration,
		server: server.NewMCPServer(
			version.BinaryName,
			version.Version,
			server.WithResourceCapabilities(true, true),
			server.WithPromptCapabilities(true),
			server.WithToolCapabilities(true),
			server.WithLogging(),
			server.WithToolHandlerMiddleware(toolCallLoggingMiddleware),
		),
	}
	if err := s.reloadKubernetesClient(); err != nil {
		return nil, err
	}
	s.k.WatchKubeConfig(s.reloadKubernetesClient)

	return s, nil
}

func (s *Server) reloadKubernetesClient() error {
	k, err := internalk8s.NewManager(s.configuration.StaticConfig)
	if err != nil {
		return err
	}
	s.k = k
	applicableTools := make([]server.ServerTool, 0)
	for _, tool := range s.configuration.Profile.GetTools(s) {
		if !s.configuration.isToolApplicable(tool) {
			continue
		}
		applicableTools = append(applicableTools, tool)
	}
	s.server.SetTools(applicableTools...)
	return nil
}

func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.server)
}

func (s *Server) ServeSse(baseUrl string, httpServer *http.Server) *server.SSEServer {
	options := make([]server.SSEOption, 0)
	options = append(options, server.WithSSEContextFunc(contextFunc), server.WithHTTPServer(httpServer))
	if baseUrl != "" {
		options = append(options, server.WithBaseURL(baseUrl))
	}
	return server.NewSSEServer(s.server, options...)
}

func (s *Server) ServeHTTP(httpServer *http.Server) *server.StreamableHTTPServer {
	options := []server.StreamableHTTPOption{
		server.WithHTTPContextFunc(contextFunc),
		server.WithStreamableHTTPServer(httpServer),
		server.WithStateLess(true),
	}
	return server.NewStreamableHTTPServer(s.server, options...)
}

// VerifyTokenAPIServer verifies the given token with the audience by
// sending an TokenReview request to API Server.
func (s *Server) VerifyTokenAPIServer(ctx context.Context, token string, audience string) (*authenticationapiv1.UserInfo, []string, error) {
	if s.k == nil {
		return nil, nil, fmt.Errorf("kubernetes manager is not initialized")
	}
	return s.k.VerifyToken(ctx, token, audience)
}

// GetKubernetesAPIServerHost returns the Kubernetes API server host from the configuration.
func (s *Server) GetKubernetesAPIServerHost() string {
	if s.k == nil {
		return ""
	}
	return s.k.GetAPIServerHost()
}

func (s *Server) Close() {
	if s.k != nil {
		s.k.Close()
	}
}

func NewTextResult(content string, err error) *mcp.CallToolResult {
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: err.Error(),
				},
			},
		}
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: content,
			},
		},
	}
}

func contextFunc(ctx context.Context, r *http.Request) context.Context {
	// Get the standard Authorization header (OAuth compliant)
	authHeader := r.Header.Get(string(internalk8s.OAuthAuthorizationHeader))
	if authHeader != "" {
		return context.WithValue(ctx, internalk8s.OAuthAuthorizationHeader, authHeader)
	}

	// Fallback to custom header for backward compatibility
	customAuthHeader := r.Header.Get(string(internalk8s.CustomAuthorizationHeader))
	if customAuthHeader != "" {
		return context.WithValue(ctx, internalk8s.OAuthAuthorizationHeader, customAuthHeader)
	}

	return ctx
}

func toolCallLoggingMiddleware(next server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, ctr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		klog.V(5).Infof("mcp tool call: %s(%v)", ctr.Params.Name, ctr.Params.Arguments)
		return next(ctx, ctr)
	}
}
