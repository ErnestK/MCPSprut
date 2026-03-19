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

	ctx := context.Background()
	for i := 0; i < 1000; i++ {
		s := storage.ServerConfig{
			ID:  fmt.Sprintf("sim-%d", i),
			URL: fmt.Sprintf("http://localhost:9093/server/%d/mcp", i),
		}
		if err := store.SaveServer(ctx, s); err != nil {
			log.Fatalf("save server %s: %v", s.ID, err)
		}
	}

	fmt.Println("Done. 1000 servers seeded.")
}
