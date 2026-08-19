ALTER TABLE sessions
    ADD COLUMN IF NOT EXISTS auth_method text NOT NULL DEFAULT 'oidc',
    ADD COLUMN IF NOT EXISTS last_seen_at timestamptz,
    ADD COLUMN IF NOT EXISTS ip_address inet,
    ADD COLUMN IF NOT EXISTS user_agent text,
    ADD COLUMN IF NOT EXISTS revoked_at timestamptz;

CREATE TABLE IF NOT EXISTS login_audit (
    id             bigserial PRIMARY KEY,
    user_id        uuid REFERENCES users(id) ON DELETE SET NULL,
    attempted_email text,
    role           text CHECK (role IN ('investor', 'lender', 'custodian')),
    auth_method    text NOT NULL,
    outcome        text NOT NULL CHECK (outcome IN ('success', 'failure')),
    failure_reason text,
    ip_address     inet,
    user_agent     text,
    created_at     timestamptz NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sessions_active_expiry
    ON sessions (expires_at) WHERE revoked = FALSE AND revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_login_audit_created_at
    ON login_audit (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_login_audit_user_id
    ON login_audit (user_id);
