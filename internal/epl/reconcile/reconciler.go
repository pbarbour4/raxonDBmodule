package reconcile

import (
	"context"
	"fmt"

	"raxonplatform/internal/db/repository"
)

type Reconciler struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Reconciler {
	return &Reconciler{repo: repo}
}

func (r *Reconciler) RunOnce(ctx context.Context, channelName string) error {
	height, err := r.repo.GetCheckpointHeight(ctx, channelName)
	if err != nil {
		return err
	}

	if err := r.repo.UpdateCheckpoint(ctx, channelName, height, "init"); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) VerifyMonotonic(localHeight, remoteHeight int64) error {
	if remoteHeight < localHeight {
		return fmt.Errorf("remote height %d is behind local height %d", remoteHeight, localHeight)
	}
	return nil
}
