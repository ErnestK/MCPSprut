package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	bolt "go.etcd.io/bbolt"
)

var (
	bucketServers = []byte("servers")
	bucketTools   = []byte("tools")
)

type BoltStorage struct {
	db      *bolt.DB
	mu      sync.Mutex
	onNewFn func(ServerConfig)
}

func NewBoltStorage(path string) (*BoltStorage, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("open bolt db: %w", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(bucketServers); err != nil {
			return err
		}
		_, err := tx.CreateBucketIfNotExists(bucketTools)
		return err
	})
	if err != nil {
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("close bolt db after bucket error: %v", closeErr)
		}
		return nil, fmt.Errorf("create buckets: %w", err)
	}

	return &BoltStorage{db: db}, nil
}

func (s *BoltStorage) LoadServers(_ context.Context) ([]ServerConfig, error) {
	var servers []ServerConfig

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketServers)
		return b.ForEach(func(k, v []byte) error {
			var sc ServerConfig
			if err := json.Unmarshal(v, &sc); err != nil {
				return fmt.Errorf("unmarshal server %s: %w", k, err)
			}
			servers = append(servers, sc)
			return nil
		})
	})

	return servers, err
}

func (s *BoltStorage) SaveServer(_ context.Context, server ServerConfig) error {
	data, err := json.Marshal(server)
	if err != nil {
		return fmt.Errorf("marshal server: %w", err)
	}

	err = s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketServers).Put([]byte(server.ID), data)
	})
	if err != nil {
		return err
	}

	s.mu.Lock()
	cb := s.onNewFn
	s.mu.Unlock()

	if cb != nil {
		cb(server)
	}

	return nil
}

func (s *BoltStorage) OnNewServer(callback func(ServerConfig)) {
	s.mu.Lock()
	s.onNewFn = callback
	s.mu.Unlock()
}

func (s *BoltStorage) SaveServersBatch(_ context.Context, servers []ServerConfig) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketServers)
		for _, srv := range servers {
			data, err := json.Marshal(srv)
			if err != nil {
				return fmt.Errorf("marshal server %s: %w", srv.ID, err)
			}
			if err := b.Put([]byte(srv.ID), data); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *BoltStorage) SaveToolsBatch(_ context.Context, updates []ToolUpdate) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketTools)
		for _, u := range updates {
			data, err := json.Marshal(u.Tools)
			if err != nil {
				return fmt.Errorf("marshal tools for %s: %w", u.ServerID, err)
			}
			if err := b.Put([]byte(u.ServerID), data); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *BoltStorage) GetTools(_ context.Context, serverID string) ([]Tool, error) {
	var tools []Tool

	err := s.db.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bucketTools).Get([]byte(serverID))
		if v == nil {
			return nil
		}
		return json.Unmarshal(v, &tools)
	})

	return tools, err
}

func (s *BoltStorage) Close() error {
	return s.db.Close()
}
