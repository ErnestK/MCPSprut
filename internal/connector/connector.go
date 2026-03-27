package connector

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ErnestK/mcp-sprut/internal/batcher"
	"github.com/ErnestK/mcp-sprut/internal/mcpclient"
	"github.com/ErnestK/mcp-sprut/internal/storage"
)

const maxRetries = 3

type Connector struct {
	server        storage.ServerConfig
	client        *mcpclient.Client
	batcher       *batcher.Batcher
	retryInterval time.Duration
}

func NewConnector(server storage.ServerConfig, client *mcpclient.Client, bat *batcher.Batcher, retryInterval time.Duration) *Connector {
	return &Connector{
		server:        server,
		client:        client,
		batcher:       bat,
		retryInterval: retryInterval,
	}
}

func (c *Connector) Run(ctx context.Context) {
	failures := 0
	for {
		err := c.connect(ctx)
		if ctx.Err() != nil {
			return
		}

		failures++
		if failures > maxRetries {
			log.Printf("Connector %s: giving up after %d failures, last error: %v", c.server.ID, maxRetries, err)
			return
		}

		log.Printf("Connector %s: %v, retrying in %s (%d/%d)", c.server.ID, err, c.retryInterval, failures, maxRetries)
		select {
		case <-ctx.Done():
			return
		case <-time.After(c.retryInterval):
		}
	}
}

func (c *Connector) connect(ctx context.Context) error {
	result, err := c.client.Initialize(ctx, c.server.URL)
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}
	log.Printf("Connector %s: initialized (%s %s)", c.server.ID, result.ServerInfo.Name, result.ServerInfo.Version)

	if err := c.client.SendInitialized(ctx, c.server.URL); err != nil {
		return fmt.Errorf("send initialized: %w", err)
	}

	ch, err := c.client.SubscribeNotifications(ctx, c.server.URL)
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	tools, err := c.client.ListTools(ctx, c.server.URL)
	if err != nil {
		return fmt.Errorf("list tools: %w", err)
	}
	log.Printf("Connector %s: fetched %d tools", c.server.ID, len(tools))
	if err := c.batcher.Submit(ctx, c.server.ID, tools); err != nil {
		return fmt.Errorf("submit tools: %w", err)
	}

	for method := range ch {
		if method == "notifications/tools/list_changed" {
			tools, err := c.client.ListTools(ctx, c.server.URL)
			if err != nil {
				return fmt.Errorf("re-fetch tools: %w", err)
			}
			log.Printf("Connector %s: tools changed, fetched %d tools", c.server.ID, len(tools))
			if err := c.batcher.Submit(ctx, c.server.ID, tools); err != nil {
				return fmt.Errorf("submit tools: %w", err)
			}
		}
	}

	return fmt.Errorf("sse stream closed")
}
