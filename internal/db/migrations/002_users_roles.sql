-- Users, sessions, token pricing and loan positions for the Dumastas web platform.

CREATE TABLE IF NOT EXISTS users (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email            text NOT NULL UNIQUE,
    role             text NOT NULL CHECK (role IN ('investor', 'lender', 'custodian')),
    oidc_subject     text UNIQUE,
    institution_name text NOT NULL DEFAULT '',
    created_at       timestamptz DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sessions (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz DEFAULT NOW(),
    expires_at timestamptz NOT NULL,
    revoked    bool NOT NULL DEFAULT FALSE
);

-- Current NAV per token_id, updated by the EPL pipeline or an admin process.
CREATE TABLE IF NOT EXISTS token_prices (
    token_id   text PRIMARY KEY,
    price      numeric(20,8) NOT NULL,
    change_pct numeric(10,4) NOT NULL DEFAULT 0,
    updated_at timestamptz DEFAULT NOW()
);

-- One row per active borrower/loan in the lending pool.
CREATE TABLE IF NOT EXISTS loans (
    loan_id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id          text NOT NULL,
    institution_name  text NOT NULL,
    token_id          text NOT NULL REFERENCES token_prices(token_id),
    collateral_amount numeric(30,8) NOT NULL,
    loan_balance      numeric(20,2) NOT NULL,
    tier              text NOT NULL DEFAULT 'Tier 1',
    status            text NOT NULL CHECK (status IN ('active', 'monitoring', 'closed', 'defaulted')),
    created_at        timestamptz DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires  ON sessions(expires_at) WHERE NOT revoked;
CREATE INDEX IF NOT EXISTS idx_loans_owner       ON loans(owner_id);
CREATE INDEX IF NOT EXISTS idx_loans_status      ON loans(status);
