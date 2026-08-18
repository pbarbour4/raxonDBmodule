-- Realistic seed data for the Dumastas MMF platform.
-- Run after 001_init.sql and 002_users_roles.sql.

-- ── Token price ──────────────────────────────────────────────────────────────
INSERT INTO token_prices (token_id, price, change_pct) VALUES
    ('DUM-MMF-001', 1.00049, 0.12)
ON CONFLICT (token_id) DO UPDATE SET
    price      = EXCLUDED.price,
    change_pct = EXCLUDED.change_pct,
    updated_at = NOW();

-- ── Demo users (oidc_subject filled on first real OIDC login) ─────────────
INSERT INTO users (id, email, role, oidc_subject, institution_name) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'investor@dumastas.com',  'investor',  'demo|investor',  'Dumastas Fund'),
    ('a0000000-0000-0000-0000-000000000002', 'lender@dumastas.com',    'lender',    'demo|lender',    'Dumastas Capital'),
    ('a0000000-0000-0000-0000-000000000003', 'custodian@dumastas.com', 'custodian', 'demo|custodian', 'Dumastas Custody')
ON CONFLICT DO NOTHING;

-- ── Investor MMF token balance ────────────────────────────────────────────
INSERT INTO balances (owner_id, token_id, available, encumbered, version, last_updated_block) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'DUM-MMF-001', 8100000.00, 4350291.50, 1, 1000)
ON CONFLICT DO NOTHING;

-- ── Borrower collateral balances ──────────────────────────────────────────
INSERT INTO balances (owner_id, token_id, available, encumbered, version, last_updated_block) VALUES
    ('astra-capital-001',  'DUM-MMF-001', 0,  5630.00,  1, 1000),
    ('nexus-liq-001',      'DUM-MMF-001', 0, 10200.50,  1, 1000),
    ('vanguard-dig-001',   'DUM-MMF-001', 0,  2100.00,  1, 1000),
    ('meridian-am-001',    'DUM-MMF-001', 0,  1500.00,  1, 1000),
    ('quantum-cap-001',    'DUM-MMF-001', 0,   800.00,  1, 1000),
    ('atlas-sov-001',      'DUM-MMF-001', 0,  1200.00,  1, 1000),
    ('pacific-rim-001',    'DUM-MMF-001', 0,   600.00,  1, 1000),
    ('nordic-alpha-001',   'DUM-MMF-001', 0,  2000.00,  1, 1000),
    ('continental-001',    'DUM-MMF-001', 0,   900.00,  1, 1000),
    ('delta-sov-001',      'DUM-MMF-001', 0,  3000.00,  1, 1000),
    ('emerald-st-001',     'DUM-MMF-001', 0,  1400.00,  1, 1000),
    ('harbor-pt-001',      'DUM-MMF-001', 0,   700.00,  1, 1000),
    ('keystone-001',       'DUM-MMF-001', 0,  1100.00,  1, 1000),
    ('summit-dig-001',     'DUM-MMF-001', 0,   985.00,  1, 1000)
ON CONFLICT DO NOTHING;

-- ── Active encumbrances (investor user: lending 3.2M + frozen 1.1M) ──────
INSERT INTO encumbrances (owner_id, token_id, amount, status) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'DUM-MMF-001', 3200000.00,  'active'),
    ('a0000000-0000-0000-0000-000000000001', 'DUM-MMF-001', 1150291.50,  'active'),
    ('astra-capital-001',                   'DUM-MMF-001',    5630.00,   'active'),
    ('nexus-liq-001',                        'DUM-MMF-001',  10200.50,   'active'),
    ('vanguard-dig-001',                     'DUM-MMF-001',   2100.00,   'active')
ON CONFLICT DO NOTHING;

-- ── Loan positions (14 borrowers) ─────────────────────────────────────────
INSERT INTO loans (owner_id, institution_name, token_id, collateral_amount, loan_balance, tier, status) VALUES
    ('astra-capital-001', 'Astra Capital Partners', 'DUM-MMF-001',  5630.00,  4200000.00, 'Tier 2', 'active'),
    ('nexus-liq-001',     'Nexus Liquidity Fund',   'DUM-MMF-001', 10200.50,  8150000.00, 'Tier 1', 'active'),
    ('vanguard-dig-001',  'Vanguard Digital LLC',   'DUM-MMF-001',  2100.00,  2000000.00, 'Tier 3', 'monitoring'),
    ('meridian-am-001',   'Meridian Asset Mgmt',    'DUM-MMF-001',  1500.00,  1150000.00, 'Tier 2', 'active'),
    ('quantum-cap-001',   'Quantum Capital Group',  'DUM-MMF-001',   800.00,   600000.00, 'Tier 3', 'active'),
    ('atlas-sov-001',     'Atlas Sovereign Fund',   'DUM-MMF-001',  1200.00,   900000.00, 'Tier 2', 'active'),
    ('pacific-rim-001',   'Pacific Rim Trust',      'DUM-MMF-001',   600.00,   460000.00, 'Tier 4', 'active'),
    ('nordic-alpha-001',  'Nordic Alpha Fund',      'DUM-MMF-001',  2000.00,  1500000.00, 'Tier 1', 'active'),
    ('continental-001',   'Continental Bridge Cap', 'DUM-MMF-001',   900.00,   680000.00, 'Tier 3', 'active'),
    ('delta-sov-001',     'Delta Sovereign LP',     'DUM-MMF-001',  3000.00,  2200000.00, 'Tier 1', 'active'),
    ('emerald-st-001',    'Emerald Street Capital', 'DUM-MMF-001',  1400.00,  1050000.00, 'Reserves', 'active'),
    ('harbor-pt-001',     'Harbor Point Trust',     'DUM-MMF-001',   700.00,   520000.00, 'Tier 4', 'active'),
    ('keystone-001',      'Keystone Lending LLC',   'DUM-MMF-001',  1100.00,   820000.00, 'Tier 3', 'active'),
    ('summit-dig-001',    'Summit Digital LLC',     'DUM-MMF-001',   985.00,   720000.00, 'Reserves', 'active');

-- ── Pending operations (custodian approval queue) ─────────────────────────
INSERT INTO operations (operation_id, type, status, requested_by, request_payload) VALUES
    ('c0000000-0000-0000-0000-000000000001', 'ENCUMBRANCE',   'requested', '0x8823...1a2c',
     '{"owner_id":"astra-capital-001","institution":"Argon Capital Partners","token_id":"DUM-MMF-001","amount":1250000,"usd_value":1250000}'),
    ('c0000000-0000-0000-0000-000000000002', 'UNENCUMBRANCE', 'requested', '0xf441...99e8',
     '{"owner_id":"vanguard-dig-001","institution":"BlackForest Holdings","token_id":"DUM-MMF-001","amount":450000,"usd_value":450000}'),
    ('c0000000-0000-0000-0000-000000000003', 'ENCUMBRANCE',   'requested', '0x229b...ff11',
     '{"owner_id":"nexus-liq-001","institution":"Nexus Prime LP","token_id":"DUM-MMF-001","amount":2000000,"usd_value":2000000}'),
    ('c0000000-0000-0000-0000-000000000004', 'ENCUMBRANCE',   'requested', '0x3f1a...22b3',
     '{"owner_id":"meridian-am-001","institution":"Meridian Asset Mgmt","token_id":"DUM-MMF-001","amount":500000,"usd_value":500000}'),
    ('c0000000-0000-0000-0000-000000000005', 'UNENCUMBRANCE', 'requested', '0x8c7d...44fa',
     '{"owner_id":"nordic-alpha-001","institution":"Nordic Alpha Fund","token_id":"DUM-MMF-001","amount":300000,"usd_value":300000}')
ON CONFLICT DO NOTHING;

-- ── Historical completed operations (for audit trail) ────────────────────
INSERT INTO operations (operation_id, type, status, requested_by, request_payload, completed_at) VALUES
    ('d0000000-0000-0000-0000-000000000001', 'ENCUMBRANCE', 'committed', '0x9a3c...3c2e',
     '{"owner_id":"astra-capital-001","institution":"Institutional Vault A","token_id":"DUM-MMF-001","amount":500000,"usd_value":500000}',
     NOW() - INTERVAL '15 minutes'),
    ('d0000000-0000-0000-0000-000000000002', 'ENCUMBRANCE', 'cancelled', '0x7b2f...f1d4',
     '{"owner_id":"delta-sov-001","institution":"Global Trust","token_id":"DUM-MMF-001","amount":100000,"usd_value":100000}',
     NOW() - INTERVAL '1 hour'),
    ('d0000000-0000-0000-0000-000000000003', 'MAINTENANCE', 'committed', 'system',
     '{"description":"Node synchronization complete","version":"2.4.1"}',
     NOW() - INTERVAL '3 hours')
ON CONFLICT DO NOTHING;

-- ── Audit trail ───────────────────────────────────────────────────────────
INSERT INTO audit_trail (entity_type, entity_id, change_type, before, after, timestamp) VALUES
    ('operation', 'd0000000-0000-0000-0000-000000000001', 'STATUS_CHANGE',
     '{"status":"requested"}',
     '{"status":"committed","note":"Encumbrance Confirmed: 500k DUM for Institutional Vault A","tx_hash":"0x9a3c...3c2e"}',
     NOW() - INTERVAL '15 minutes'),
    ('operation', 'd0000000-0000-0000-0000-000000000002', 'STATUS_CHANGE',
     '{"status":"requested"}',
     '{"status":"cancelled","reason":"Compliance Check Failure","note":"Request Rejected: Limit exceeded for Global Trust"}',
     NOW() - INTERVAL '1 hour'),
    ('operation', 'd0000000-0000-0000-0000-000000000003', 'STATUS_CHANGE',
     '{"status":"requested"}',
     '{"status":"committed","note":"System Maintenance: Node synchronization complete — Version 2.4.1"}',
     NOW() - INTERVAL '3 hours');

-- ── Checkpoint ────────────────────────────────────────────────────────────
INSERT INTO checkpoint (channel_name, last_processed_height, last_processed_hash, updated_at) VALUES
    ('tokenization-channel', 1000,
     'a1b2c3d4e5f67890abcdef1234567890abcdef1234567890abcdef1234567890',
     NOW())
ON CONFLICT (channel_name) DO UPDATE SET
    last_processed_height = EXCLUDED.last_processed_height,
    last_processed_hash   = EXCLUDED.last_processed_hash,
    updated_at            = EXCLUDED.updated_at;
