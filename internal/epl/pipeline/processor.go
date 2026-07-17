package pipeline

import (
	"context"
	"log"
	"time"

	"raxonplatform/internal/db/repository"
)

type Processor struct {
	repo *repository.Repository
}

func NewProcessor(repo *repository.Repository) *Processor {
	return &Processor{repo: repo}
}

func (p *Processor) Start(ctx context.Context, channelName string) error {
	log.Printf("epl processor ready channel=%s", channelName)

	// Placeholder heartbeat loop while Fabric listener integration is implemented.
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(30 * time.Second):
			log.Printf("epl processor waiting for fabric listener integration channel=%s", channelName)
		}
	}
}
