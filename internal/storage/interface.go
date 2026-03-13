package storage

import "context"

type ServerConfig struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type ToolUpdate struct {
	ServerID string
	Tools    []Tool
}

type Storage interface {
	LoadServers(ctx context.Context) ([]ServerConfig, error)
	SaveServer(ctx context.Context, server ServerConfig) error
	OnNewServer(callback func(ServerConfig))

	SaveToolsBatch(ctx context.Context, updates []ToolUpdate) error
	GetTools(ctx context.Context, serverID string) ([]Tool, error)

	Close() error
}
