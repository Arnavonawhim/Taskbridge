package config


type ServerConfig struct {
	Addr      string
	AuthToken string
}

type AgentConfig struct {
	ServerURL    string
	AgentID      string
	AuthToken    string
	Capabilities []string
}

