package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ErnestK/mcp-sprut/internal/storage"
)

func main() {
	dbPath := "sprut.db"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	store, err := storage.NewBoltStorage(dbPath)
	if err != nil {
		log.Fatalf("open storage: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			log.Printf("close storage: %v", err)
		}
	}()

	servers := make([]storage.ServerConfig, 10000)
	for i := range servers {
		servers[i] = storage.ServerConfig{
			ID:  fmt.Sprintf("sim-%d", i),
			URL: fmt.Sprintf("http://localhost:9093/server/%d/mcp", i),
		}
	}

	if err := store.SaveServersBatch(context.Background(), servers); err != nil {
		log.Fatalf("seed servers: %v", err)
	}

	fmt.Println("Done. 10000 servers seeded.")
}
