package mcp

import (
	"slices"

	"github.com/mark3labs/mcp-go/server"
)

type Profile interface {
	GetName() string
	GetDescription() string
	GetTools(s *Server) []server.ServerTool
}

var Profiles = []Profile{
	&FullProfile{},
	&CicdProfile{},
}

var ProfileNames []string

func ProfileFromString(name string) Profile {
	for _, profile := range Profiles {
		if profile.GetName() == name {
			return profile
		}
	}
	return nil
}

type FullProfile struct{}

func (p *FullProfile) GetName() string {
	return "full"
}
func (p *FullProfile) GetDescription() string {
	return "Complete profile with all tools and extended outputs"
}
func (p *FullProfile) GetTools(s *Server) []server.ServerTool {
	return slices.Concat(
		s.initConfiguration(),
		s.initEvents(),
		s.initNamespaces(),
		s.initPods(),
		s.initResources(),
		s.initHelm(),
		s.initCicdSimple(),
		s.initContainers(),
		s.initRegistryTools(),
		s.initWorkflowTools(),
	)
}

type CicdProfile struct{}

func (p *CicdProfile) GetName() string {
	return "cicd"
}
func (p *CicdProfile) GetDescription() string {
	return "CI/CD profile with intelligent workflow orchestration, Git monitoring, container image building, registry management, and automated deployment tools with natural language processing"
}
func (p *CicdProfile) GetTools(s *Server) []server.ServerTool {
	return slices.Concat(
		s.initConfiguration(),
		s.initNamespaces(),
		s.initPods(),
		s.initResources(),
		s.initCicdSimple(),
		s.initContainers(),
		s.initRegistryTools(),
		s.initWorkflowTools(),
	)
}

func init() {
	ProfileNames = make([]string, 0)
	for _, profile := range Profiles {
		ProfileNames = append(ProfileNames, profile.GetName())
	}
}
