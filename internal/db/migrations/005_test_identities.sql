-- Second investor login for testing the tokenization chaincode's 2-investor / 1-custodian
-- flow end-to-end against the web dashboard. Run after 001-004.
--
-- NOTE: on-chain, investor-initiated requests (RequestTransfer/RequestEncumbrance/
-- RequestUnencumbrance) require ownerID to equal the caller's chain identity string
-- (from the chaincode's WhoAmI query, run via fabriccli as investor2). To see chain
-- activity reflected against this user's balance, mint to that exact owner_id string
-- rather than the placeholder UUID below, or update this row's owner references to match.
INSERT INTO users (id, email, role, oidc_subject, institution_name) VALUES
    ('a0000000-0000-0000-0000-000000000004', 'investor2@dumastas.com', 'investor', 'demo|investor2', 'Dumastas Fund II')
ON CONFLICT DO NOTHING;

INSERT INTO balances (owner_id, token_id, available, encumbered, version, last_updated_block) VALUES
    ('a0000000-0000-0000-0000-000000000004', 'DUM-MMF-001', 0, 0, 0, 1000)
ON CONFLICT DO NOTHING;
