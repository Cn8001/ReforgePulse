package app

// AgentService tells the connect panel how an agent starts the MCP server.
type AgentService struct {
	exe string
}

func NewAgentService(exe string) *AgentService {
	return &AgentService{exe: exe}
}

// Executable is the path agents run with the "mcp" argument.
func (s *AgentService) Executable() string {
	return s.exe
}
