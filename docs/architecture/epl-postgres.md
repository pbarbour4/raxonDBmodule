# EPL and Postgres Architecture (Phase 1)

## Module boundaries

- EPL ingest and processing is a deterministic pipeline per Fabric channel.
- Postgres is the authoritative off-chain query model with replay-safe writes.
- Checkpoint state controls restart and reconciliation boundaries.

## Data flow

1. Receive Fabric block events.
2. Parse block and transaction metadata.
3. Normalize chaincode events to canonical envelope.
4. Validate deterministic constraints.
5. Persist block, transaction, and event records.
6. Update balances and encumbrances projections.
7. Write outbound notification intents.
8. Commit checkpoint.

## Idempotency strategy

- At-least-once ingest.
- Unique constraints on transaction and event keys.
- Conflict-safe writes to avoid duplicate materialization.
- Monotonic checkpoint updates.

## Integration contracts

- UI and API consume operations, balances, and event timelines.
- RBAC integrates through requested_by, owner_id, and audit fields.
- Service layer uses checkpoint and mismatch states for reconciliation views.
