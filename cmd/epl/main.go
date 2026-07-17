package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"raxonplatform/internal/config"
	"raxonplatform/internal/db/postgres"
	"raxonplatform/internal/db/repository"
	"raxonplatform/internal/epl/pipeline"
	"raxonplatform/internal/epl/reconcile"
)

func main() {
	cfg := config.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	dbPool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to create postgres pool: %v", err)
	}
	defer dbPool.Close()

	repo := repository.New(dbPool)
	processor := pipeline.NewProcessor(repo)
	reconciler := reconcile.New(repo)

	log.Printf("starting epl service channel=%s", cfg.ChannelName)
	if err := processor.Start(ctx, cfg.ChannelName); err != nil {
		log.Fatalf("processor exited with error: %v", err)
	}

	if err := reconciler.RunOnce(ctx, cfg.ChannelName); err != nil {
		log.Printf("reconciler warning: %v", err)
	}
}
