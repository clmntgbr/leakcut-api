package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-api/cmd/worker/di"
	"go-api/internal/infrastructure/config"
	"go-api/internal/infrastructure/persistence/schema"
)

func main() {
	env := config.Load()
	db := config.ConnectDatabase(env)

	if err := schema.AssertModelsMatchDB(db); err != nil {
		log.Fatalf("schema check failed: %v", err)
	}

	container := di.NewContainer(db, env)
	defer container.Conn.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go container.Relay.Start(ctx)
	go expireStaleUploads(ctx, container)

	if err := container.Consumer.Start(ctx); err != nil {
		log.Fatalf("worker stopped with error: %v", err)
	}
}

func expireStaleUploads(ctx context.Context, container *di.Container) {
	interval := container.ExpireUploadsInterval
	if interval <= 0 {
		interval = time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := container.ExpireStaleUploads.Handle(ctx); err != nil {
				log.Printf("expire stale uploads failed: %v", err)
			}
		}
	}
}
