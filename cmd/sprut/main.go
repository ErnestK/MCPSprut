package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ErnestK/mcp-sprut/internal/batcher"
	"github.com/ErnestK/mcp-sprut/internal/config"
	"github.com/ErnestK/mcp-sprut/internal/hub"
	"github.com/ErnestK/mcp-sprut/internal/mcpclient"
	"github.com/ErnestK/mcp-sprut/internal/storage"
)

func main() {
	cfg := config.Load()

	log.Printf("MCPSprut starting with config: db=%s batch=%d flush=%s timeout=%s retry=%s",
		cfg.DBPath, cfg.BatchSize, cfg.FlushInterval, cfg.ConnectTimeout, cfg.RetryInterval)

	store, err := storage.NewBoltStorage(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open storage: %v", err)
	}
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())

	client := mcpclient.NewClient(cfg.ConnectTimeout)

	bat := batcher.NewBatcher(store, cfg.BatchSize, cfg.BufferSize, cfg.FlushInterval)
	bat.Start(ctx)

	h := hub.NewHub(store, client, bat, cfg.RetryInterval)
	if err := h.Start(ctx); err != nil {
		log.Fatalf("Failed to start hub: %v", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	cancel()

	done := make(chan struct{})
	go func() {
		h.Wait()
		bat.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("Done")
	case <-time.After(5 * time.Second):
		log.Println("Shutdown timed out, forcing exit")
	}
}
