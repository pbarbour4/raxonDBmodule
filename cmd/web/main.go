package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"raxonplatform/internal/api"
	"raxonplatform/internal/config"
	"raxonplatform/internal/db/postgres"
	"raxonplatform/internal/db/repository"
)

func main() {
	cfg := config.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres pool: %v", err)
	}
	defer pool.Close()

	repo := repository.New(pool)

	srv, err := api.New(repo, cfg)
	if err != nil {
		log.Fatalf("server init: %v", err)
	}

	if err := srv.Start(ctx); err != nil {
		log.Fatalf("server: %v", err)
	}
}
