CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS blocks (
    channel_name text NOT NULL,
    block_height bigint NOT NULL,
    block_hash text NOT NULL,
    prev_hash text NOT NULL,
    tx_count integer NOT NULL,
    block_timestamp timestamptz,
    processed_at timestamptz,
    raw_block_ref text,
    validation_status text,
    created_at timestamptz DEFAULT NOW(),
    PRIMARY KEY (channel_name, block_height)
);

CREATE TABLE IF NOT EXISTS raw_blocks (
    channel_name text NOT NULL,
    block_height bigint NOT NULL,
    raw_payload jsonb NOT NULL,
    stored_at timestamptz DEFAULT NOW(),
    storage_uri text,
    PRIMARY KEY (channel_name, block_height),
    CONSTRAINT fk_raw_blocks_block
        FOREIGN KEY (channel_name, block_height)
        REFERENCES blocks(channel_name, block_height)
);

CREATE TABLE IF NOT EXISTS transactions (
    channel_name text NOT NULL,
    tx_id text NOT NULL,
    block_height bigint NOT NULL,
    tx_index integer NOT NULL,
    chaincode text,
    function text,
    args jsonb,
    read_set jsonb,
    write_set jsonb,
    endorsements jsonb,
    validation_code integer,
    timestamp timestamptz,
    processed boolean DEFAULT FALSE,
    created_at timestamptz DEFAULT NOW(),
    PRIMARY KEY (channel_name, tx_id),
    CONSTRAINT fk_transactions_block
        FOREIGN KEY (channel_name, block_height)
        REFERENCES blocks(channel_name, block_height),
    CONSTRAINT uq_transactions_block_txindex
        UNIQUE (channel_name, block_height, tx_index)
);

CREATE TABLE IF NOT EXISTS events (
    event_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_name text NOT NULL,
    tx_id text NOT NULL,
    event_type text NOT NULL,
    event_subtype text,
    payload jsonb NOT NULL,
    chaincode_event_name text,
    timestamp timestamptz,
    processed_at timestamptz,
    created_at timestamptz DEFAULT NOW(),
    CONSTRAINT fk_events_tx
        FOREIGN KEY (channel_name, tx_id)
        REFERENCES transactions(channel_name, tx_id),
    CONSTRAINT uq_events_idempotent
        UNIQUE (channel_name, tx_id, chaincode_event_name, event_subtype)
);

CREATE TABLE IF NOT EXISTS operations (
    operation_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    type text NOT NULL,
    status text NOT NULL CHECK (status IN ('requested', 'accepted', 'submitted', 'committed', 'failed', 'cancelled')),
    requested_by text,
    request_payload jsonb,
    result_payload jsonb,
    created_at timestamptz DEFAULT NOW(),
    completed_at timestamptz,
    related_tx_id text,
    related_event_id uuid
);

CREATE TABLE IF NOT EXISTS balances (
    owner_id text NOT NULL,
    token_id text NOT NULL,
    available numeric(30,8) DEFAULT 0,
    encumbered numeric(30,8) DEFAULT 0,
    version bigint DEFAULT 0,
    last_updated_block bigint,
    PRIMARY KEY (owner_id, token_id)
);

CREATE TABLE IF NOT EXISTS encumbrances (
    encumbrance_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id text NOT NULL,
    token_id text NOT NULL,
    amount numeric(30,8) NOT NULL,
    status text NOT NULL CHECK (status IN ('active', 'released', 'expired', 'cancelled')),
    expiry timestamptz,
    related_operation_id uuid,
    created_block bigint,
    released_block bigint,
    created_at timestamptz DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS audit_trail (
    id bigserial PRIMARY KEY,
    entity_type text NOT NULL,
    entity_id text NOT NULL,
    change_type text NOT NULL,
    before jsonb,
    after jsonb,
    tx_id text,
    block_height bigint,
    timestamp timestamptz DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS invalid_transactions (
    tx_id text PRIMARY KEY,
    block_height bigint,
    reason text NOT NULL,
    raw_tx jsonb,
    recorded_at timestamptz DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS checkpoint (
    channel_name text PRIMARY KEY,
    last_processed_height bigint DEFAULT 0,
    last_processed_hash text,
    updated_at timestamptz DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS dlq (
    id bigserial PRIMARY KEY,
    source_type text NOT NULL,
    payload jsonb NOT NULL,
    error text,
    retry_count integer DEFAULT 0,
    created_at timestamptz DEFAULT NOW(),
    last_attempt_at timestamptz
);

CREATE TABLE IF NOT EXISTS notifications_log (
    id bigserial PRIMARY KEY,
    event_id uuid,
    destination text,
    payload jsonb,
    status text CHECK (status IN ('pending', 'sent', 'failed', 'acknowledged')),
    sent_at timestamptz,
    response jsonb,
    CONSTRAINT fk_notifications_event
        FOREIGN KEY (event_id)
        REFERENCES events(event_id)
);

CREATE INDEX IF NOT EXISTS idx_blocks_processed ON blocks (channel_name, processed_at);
CREATE INDEX IF NOT EXISTS idx_blocks_timestamp ON blocks (block_timestamp);
CREATE INDEX IF NOT EXISTS idx_transactions_block ON transactions (channel_name, block_height);
CREATE INDEX IF NOT EXISTS idx_transactions_processed ON transactions (processed);
CREATE INDEX IF NOT EXISTS idx_transactions_timestamp ON transactions (timestamp);
CREATE INDEX IF NOT EXISTS idx_events_type_time ON events (event_type, timestamp);
CREATE INDEX IF NOT EXISTS idx_events_tx ON events (channel_name, tx_id);
CREATE INDEX IF NOT EXISTS idx_events_processed_at ON events (processed_at);
CREATE INDEX IF NOT EXISTS idx_events_payload_gin ON events USING GIN (payload);
CREATE INDEX IF NOT EXISTS idx_operations_status_created ON operations (status, created_at);
CREATE INDEX IF NOT EXISTS idx_operations_related_tx ON operations (related_tx_id);
CREATE INDEX IF NOT EXISTS idx_operations_related_event ON operations (related_event_id);
CREATE INDEX IF NOT EXISTS idx_balances_last_updated_block ON balances (last_updated_block);
CREATE INDEX IF NOT EXISTS idx_encumbrances_owner_token_status ON encumbrances (owner_id, token_id, status);
CREATE INDEX IF NOT EXISTS idx_encumbrances_expiry ON encumbrances (expiry);
CREATE INDEX IF NOT EXISTS idx_encumbrances_related_operation ON encumbrances (related_operation_id);
CREATE INDEX IF NOT EXISTS idx_audit_entity ON audit_trail (entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_tx ON audit_trail (tx_id);
CREATE INDEX IF NOT EXISTS idx_audit_block ON audit_trail (block_height);
CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_trail (timestamp);
CREATE INDEX IF NOT EXISTS idx_dlq_source_retry_created ON dlq (source_type, retry_count, created_at);
CREATE INDEX IF NOT EXISTS idx_dlq_retry_pending ON dlq (created_at) WHERE retry_count < 10;
CREATE INDEX IF NOT EXISTS idx_notifications_event ON notifications_log (event_id);
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications_log (status);
