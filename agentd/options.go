package agentd

import ()

type AgentdOptions struct {
	TcpAddress string `flag:"tcp-address"`
}

func NewAgentdOptions() *AgentdOptions {
	o := &AgentdOptions{
		TcpAddress: "0.0.0.0:4050",
	}

	return o
}
