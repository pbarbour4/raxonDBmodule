-- Make the event idempotency key deterministic when no subtype is present.
UPDATE events
SET event_subtype = ''
WHERE event_subtype IS NULL;

ALTER TABLE events
    ALTER COLUMN event_subtype SET DEFAULT '',
    ALTER COLUMN event_subtype SET NOT NULL;