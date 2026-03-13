package batcher

import (
	"context"
	"log"
	"time"

	"github.com/ErnestK/mcp-sprut/internal/storage"
)

type Batcher struct {
	storage       storage.Storage
	batchSize     int
	flushInterval time.Duration
	updates       chan storage.ToolUpdate
	done          chan struct{}
}

func NewBatcher(store storage.Storage, batchSize int, bufferSize int, flushInterval time.Duration) *Batcher {
	return &Batcher{
		storage:       store,
		batchSize:     batchSize,
		flushInterval: flushInterval,
		updates:       make(chan storage.ToolUpdate, bufferSize),
		done:          make(chan struct{}),
	}
}

func (b *Batcher) Submit(serverID string, tools []storage.Tool) {
	b.updates <- storage.ToolUpdate{ServerID: serverID, Tools: tools}
}

func (b *Batcher) Start(ctx context.Context) {
	go b.loop(ctx)
}

func (b *Batcher) Wait() {
	<-b.done
}

func (b *Batcher) loop(ctx context.Context) {
	defer close(b.done)

	buf := make([]storage.ToolUpdate, 0, b.batchSize)
	ticker := time.NewTicker(b.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			b.drain(&buf)
			b.flush(buf)
			return

		case update := <-b.updates:
			buf = append(buf, update)
			if len(buf) >= b.batchSize {
				b.flush(buf)
				buf = buf[:0]
			}

		case <-ticker.C:
			if len(buf) > 0 {
				b.flush(buf)
				buf = buf[:0]
			}
		}
	}
}

func (b *Batcher) drain(buf *[]storage.ToolUpdate) {
	for {
		select {
		case update := <-b.updates:
			*buf = append(*buf, update)
		default:
			return
		}
	}
}

func (b *Batcher) flush(buf []storage.ToolUpdate) {
	if len(buf) == 0 {
		return
	}
	if err := b.storage.SaveToolsBatch(context.Background(), buf); err != nil {
		log.Printf("Batcher: failed to flush %d updates: %v", len(buf), err)
	} else {
		log.Printf("Batcher: flushed %d updates", len(buf))
	}
}
