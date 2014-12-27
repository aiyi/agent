package agent

import ()

type AgentdOptions struct {
	TcpAddress string `flag:"tcp-address"`
}

func NewAgentdOptions() *AgentdOptions {
	o := &AgentdOptions{
		TcpAddress: "0.0.0.0:3002",
	}

	return o
}
