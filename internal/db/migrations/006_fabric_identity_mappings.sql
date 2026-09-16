-- Maps Fabric identity values emitted by chaincode to canonical web users.
-- The raw identity may be populated after `whoami` if it differs from the label.
CREATE TABLE IF NOT EXISTS fabric_identity_mappings (
    fabric_identity text PRIMARY KEY,
    identity_label  text NOT NULL,
    msp_id          text NOT NULL,
    user_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at      timestamptz NOT NULL DEFAULT NOW(),
    UNIQUE (identity_label, msp_id)
);

INSERT INTO fabric_identity_mappings (fabric_identity, identity_label, msp_id, user_id)
SELECT 'investor1', 'investor1', 'Org1MSP', id
FROM users
WHERE oidc_subject = 'demo|investor'
ON CONFLICT (fabric_identity) DO NOTHING;

INSERT INTO fabric_identity_mappings (fabric_identity, identity_label, msp_id, user_id)
SELECT 'investor2', 'investor2', 'Org1MSP', id
FROM users
WHERE oidc_subject = 'demo|investor2'
ON CONFLICT (fabric_identity) DO NOTHING;

INSERT INTO fabric_identity_mappings (fabric_identity, identity_label, msp_id, user_id)
SELECT 'custodian1', 'custodian1', 'Org1MSP', id
FROM users
WHERE oidc_subject = 'demo|custodian'
ON CONFLICT (fabric_identity) DO NOTHING;