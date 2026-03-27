package hub

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ErnestK/mcp-sprut/internal/batcher"
	"github.com/ErnestK/mcp-sprut/internal/connector"
	"github.com/ErnestK/mcp-sprut/internal/mcpclient"
	"github.com/ErnestK/mcp-sprut/internal/storage"
)

type Hub struct {
	storage       storage.Storage
	client        *mcpclient.Client
	batcher       *batcher.Batcher
	retryInterval time.Duration
	wg            sync.WaitGroup
	shutdownMu    sync.Mutex
	shutdown      bool
}

func NewHub(store storage.Storage, client *mcpclient.Client, bat *batcher.Batcher, retryInterval time.Duration) *Hub {
	return &Hub{
		storage:       store,
		client:        client,
		batcher:       bat,
		retryInterval: retryInterval,
	}
}

func (h *Hub) Start(ctx context.Context) error {
	servers, err := h.storage.LoadServers(ctx)
	if err != nil {
		return fmt.Errorf("load servers: %w", err)
	}

	for _, server := range servers {
		h.startConnector(ctx, server)
	}

	h.storage.OnNewServer(func(server storage.ServerConfig) {
		log.Printf("Hub: new server detected: %s", server.ID)
		h.startConnector(ctx, server)
	})

	log.Printf("Hub: started with %d servers", len(servers))
	return nil
}

func (h *Hub) Wait() {
	h.shutdownMu.Lock()
	h.shutdown = true
	h.shutdownMu.Unlock()
	h.wg.Wait()
}

func (h *Hub) startConnector(ctx context.Context, server storage.ServerConfig) {
	h.shutdownMu.Lock()
	if h.shutdown {
		h.shutdownMu.Unlock()
		log.Printf("Hub: ignoring new server %s, shutting down", server.ID)
		return
	}
	h.wg.Add(1)
	h.shutdownMu.Unlock()
	go func() {
		defer h.wg.Done()
		conn := connector.NewConnector(server, h.client, h.batcher, h.retryInterval)
		conn.Run(ctx)
	}()
}
