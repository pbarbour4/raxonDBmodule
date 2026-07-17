package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"raxonplatform/internal/contracts"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) SaveEvent(ctx context.Context, e contracts.EventEnvelope) error {
	payload, err := json.Marshal(e.Payload)
	if err != nil {
		return fmt.Errorf("marshal event payload: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO events (
			event_id,
			channel_name,
			tx_id,
			event_type,
			event_subtype,
			payload,
			chaincode_event_name,
			timestamp,
			created_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
		ON CONFLICT (channel_name, tx_id, chaincode_event_name, event_subtype)
		DO UPDATE SET payload = EXCLUDED.payload
	`, e.EventID, e.ChannelName, e.TxID, e.EventType, e.EventSubtype, payload, e.ChaincodeEventName, e.OccurredAt)
	if err != nil {
		return fmt.Errorf("save event: %w", err)
	}

	return nil
}

func (r *Repository) UpdateCheckpoint(ctx context.Context, channelName string, height int64, hash string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO checkpoint (channel_name, last_processed_height, last_processed_hash, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (channel_name)
		DO UPDATE SET
			last_processed_height = GREATEST(checkpoint.last_processed_height, EXCLUDED.last_processed_height),
			last_processed_hash = EXCLUDED.last_processed_hash,
			updated_at = NOW()
	`, channelName, height, hash)
	if err != nil {
		return fmt.Errorf("update checkpoint: %w", err)
	}
	return nil
}

func (r *Repository) GetCheckpointHeight(ctx context.Context, channelName string) (int64, error) {
	var height int64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(last_processed_height, 0)
		FROM checkpoint
		WHERE channel_name = $1
	`, channelName).Scan(&height)
	if err == pgx.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get checkpoint height: %w", err)
	}
	return height, nil
}
