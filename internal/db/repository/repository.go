package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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

func (r *Repository) ResolveFabricIdentity(ctx context.Context, fabricIdentity string) (string, error) {
	var userID string
	err := r.pool.QueryRow(ctx, `
		SELECT user_id::text
		FROM fabric_identity_mappings
		WHERE fabric_identity = $1
		   OR fabric_identity = split_part(split_part($1, 'CN=', 2), '::', 1)
		LIMIT 1
	`, fabricIdentity).Scan(&userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("fabric identity %q is not mapped", fabricIdentity)
		}
		return "", fmt.Errorf("resolve fabric identity: %w", err)
	}
	return userID, nil
}

func (r *Repository) SaveBlock(ctx context.Context, channelName string, height int64, blockHash, previousHash string, txCount int, timestamp time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO blocks (channel_name, block_height, block_hash, prev_hash, tx_count, block_timestamp)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, '0001-01-01 00:00:00+00'::timestamptz))
		ON CONFLICT (channel_name, block_height) DO UPDATE SET
			block_hash = EXCLUDED.block_hash,
			prev_hash = EXCLUDED.prev_hash,
			tx_count = EXCLUDED.tx_count,
			block_timestamp = COALESCE(EXCLUDED.block_timestamp, blocks.block_timestamp)
	`, channelName, height, blockHash, previousHash, txCount, timestamp)
	if err != nil {
		return fmt.Errorf("save block: %w", err)
	}
	return nil
}

func (r *Repository) SaveTransaction(ctx context.Context, channelName, txID string, blockHeight int64, txIndex, validationCode int, chaincode, function string, args []string, timestamp time.Time) error {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("marshal transaction args: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO transactions (
			channel_name, tx_id, block_height, tx_index, chaincode, function, args,
			validation_code, timestamp
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,NULLIF($8, 0),NULLIF($9, '0001-01-01 00:00:00+00'::timestamptz))
		ON CONFLICT (channel_name, tx_id) DO UPDATE SET
			block_height = EXCLUDED.block_height,
			tx_index = EXCLUDED.tx_index,
			chaincode = EXCLUDED.chaincode,
			function = EXCLUDED.function,
			args = EXCLUDED.args,
			validation_code = EXCLUDED.validation_code,
			timestamp = EXCLUDED.timestamp
	`, channelName, txID, blockHeight, txIndex, chaincode, function, argsJSON, validationCode, timestamp)
	if err != nil {
		return fmt.Errorf("save transaction: %w", err)
	}
	return nil
}

func (r *Repository) SaveInvalidTransaction(ctx context.Context, txID string, blockHeight int64, reason string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO invalid_transactions (tx_id, block_height, reason)
		VALUES ($1, $2, $3)
		ON CONFLICT (tx_id) DO UPDATE SET
			block_height = EXCLUDED.block_height,
			reason = EXCLUDED.reason
	`, txID, blockHeight, reason)
	if err != nil {
		return fmt.Errorf("save invalid transaction: %w", err)
	}
	return nil
}

func (r *Repository) SaveEvent(ctx context.Context, e contracts.EventEnvelope) (string, bool, error) {
	payload, err := json.Marshal(e.Payload)
	if err != nil {
		return "", false, fmt.Errorf("marshal event payload: %w", err)
	}

	var eventID string
	var shouldProcess bool
	err = r.pool.QueryRow(ctx, `
		INSERT INTO events (
			channel_name,
			tx_id,
			event_type,
			event_subtype,
			payload,
			chaincode_event_name,
			timestamp,
			created_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,NOW())
		ON CONFLICT (channel_name, tx_id, chaincode_event_name, event_subtype)
		DO NOTHING
		RETURNING event_id::text
	`, e.ChannelName, e.TxID, e.EventType, e.EventSubtype, payload, e.ChaincodeEventName, e.OccurredAt).Scan(&eventID)
	if err == pgx.ErrNoRows {
		err = r.pool.QueryRow(ctx, `
			SELECT event_id::text, processed_at IS NULL
			FROM events
			WHERE channel_name = $1 AND tx_id = $2
			  AND chaincode_event_name = $3 AND event_subtype = $4
		`, e.ChannelName, e.TxID, e.ChaincodeEventName, e.EventSubtype).Scan(&eventID, &shouldProcess)
		if err != nil {
			return "", false, fmt.Errorf("check event processing state: %w", err)
		}
		return eventID, shouldProcess, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("save event: %w", err)
	}

	return eventID, true, nil
}

func (r *Repository) MarkEventProcessed(ctx context.Context, e contracts.EventEnvelope) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE events
		SET processed_at = NOW()
		WHERE channel_name = $1 AND tx_id = $2
		  AND chaincode_event_name = $3 AND event_subtype = $4
	`, e.ChannelName, e.TxID, e.ChaincodeEventName, e.EventSubtype)
	if err != nil {
		return fmt.Errorf("mark event processed: %w", err)
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
