# RaxonDBModule End-to-End Testing

This guide tests the complete local flow:

1. PostgreSQL starts in Docker.
2. Hyperledger Fabric starts in Docker.
3. Fabric certificates, channel, and chaincode are created.
4. Dummy investor and custodian identities submit transactions through `fabriccli`.
5. EPL consumes committed Fabric blocks and updates PostgreSQL.
6. The web service reads PostgreSQL and displays the updated data in HTML dashboards.

Run commands from the repository root:

```text
C:\Users\paulb\Dev\repos\raxonDBmodule
```

Use Git Bash for the Fabric shell scripts. Use PowerShell or Git Bash for Go services.

## Prerequisites

Install and start:

- Docker Desktop
- Git Bash or WSL
- Go 1.25
- A web browser

Check versions:

```bash
docker version
docker compose version
go version
```

## Clean Start Options

### Normal stop and restart

This removes containers but preserves Docker volumes, certificates, wallets, channel artifacts, and ledger state:

```bash
docker compose -f docker-compose.yml -f network/docker-compose.fabric.yml -f network/docker-compose.clients.yml down
```

Restart without rebuilding:

```bash
docker compose -f docker-compose.yml -f network/docker-compose.fabric.yml up -d
```

### Full destructive reset

Use this when certificates are stale, channel artifacts are incompatible, the ledger is corrupted, or you want to repeat the test from zero. This deletes the PostgreSQL database volume and Fabric ledger volumes.

Run from Git Bash:

```bash
docker compose -f docker-compose.yml -f network/docker-compose.clients.yml down -v
docker compose -f docker-compose.yml -f network/docker-compose.fabric.yml -f network/docker-compose.clients.yml down -v
rm -rf organizations
rm -rf network/organizations
rm -rf network/wallets
rm -rf channel-artifacts
```

If a fixed-name container remains:

```bash
docker rm -f ca.orderer.example.com ca.org1.example.com orderer.example.com peer0.org1.example.com raxon-fabric-cli raxon-cli-custodian raxon-cli-investor1 raxon-cli-investor2
```

The `docker rm` command may report that a container does not exist. That is harmless.

PowerShell equivalent for deleting generated directories:

```powershell
Remove-Item -Recurse -Force organizations, network\organizations, network\wallets, channel-artifacts -ErrorAction SilentlyContinue
```

Do not use the destructive reset if you need to preserve existing ledger or database data.

## 1. Start PostgreSQL

From the repository root:

```bash
docker compose up -d postgres
```

Verify that PostgreSQL is ready:

```bash
docker ps --filter name=raxon-postgres
docker exec raxon-postgres pg_isready -U postgres -d raxon
```

Expected readiness output:

```text
accepting connections
```

## 2. Apply Database Migrations

Apply every migration in filename order. The `-v ON_ERROR_STOP=1` option stops immediately if a migration fails.

Git Bash:

```bash
for file in internal/db/migrations/*.sql; do
  echo "Applying $file..."
  docker exec -i raxon-postgres psql -v ON_ERROR_STOP=1 -U postgres -d raxon < "$file" || exit 1
done
```

PowerShell:

```powershell
Get-ChildItem .\internal\db\migrations\*.sql | Sort-Object Name | ForEach-Object {
  Write-Host "Applying $($_.Name)..."
  Get-Content $_.FullName | docker exec -i raxon-postgres psql -v ON_ERROR_STOP=1 -U postgres -d raxon
  if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}
```

Verify the schema and seed data:

```bash
docker exec raxon-postgres psql -U postgres -d raxon -c "\\dt"
docker exec raxon-postgres psql -U postgres -d raxon -c "SELECT email, role, id FROM users ORDER BY role, email;"
docker exec raxon-postgres psql -U postgres -d raxon -c "SELECT fabric_identity, identity_label, msp_id, user_id FROM fabric_identity_mappings;"
docker exec raxon-postgres psql -U postgres -d raxon -c "SELECT token_id, price FROM token_prices;"
```

The identity mapping should include:

```text
investor1
investor2
custodian1
```

When using interactive `psql`, enter `\\dt` by itself. Do not paste SQL statements on the same line as `\\dt`.

## 3. Start the Fabric CA Containers

The CA containers must be running before crypto material is generated:

```bash
docker compose -f docker-compose.yml -f network/docker-compose.fabric.yml up -d ca.orderer.example.com ca.org1.example.com
```

Check status:

```bash
docker compose -f docker-compose.yml -f network/docker-compose.fabric.yml ps
docker logs ca.orderer.example.com
docker logs ca.org1.example.com
```

The compose file includes explicit CA hostnames in the TLS certificates. If you changed those settings after the CAs were first initialized, perform the full destructive reset before continuing.

The setup scripts use separate Fabric CA client homes for `ca-orderer` and
`ca-org1`. This is required so Org1 identity registration does not reuse the
Orderer CA administrator credentials.

## 4. Generate Fabric Crypto Material

Run:

```bash
bash network/scripts/generate-crypto.sh
```

The script creates organization MSPs, orderer and peer MSPs, TLS certificates, and Node OU configuration. It also sets `MSYS_NO_PATHCONV=1` for Git Bash compatibility.

Verify key directories:

```bash
test -f organizations/fabric-ca/ordererOrg/tls-cert.pem
test -f organizations/fabric-ca/org1/tls-cert.pem
test -d organizations/ordererOrganizations/example.com/msp/cacerts
test -d organizations/peerOrganizations/org1.example.com/msp/cacerts
test -f organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls/server.crt
test -f organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/server.crt
```

If the command reports a TLS hostname error, reset the generated `organizations` directory and recreate the CA containers before rerunning this step.

## 5. Register Dummy Identities

Run:

```bash
bash network/scripts/register-identities.sh
```

This registers and enrolls:

| Identity | Role |
|---|---|
| `custodian1` | `custodian` |
| `investor1` | `investor` |
| `investor2` | `investor` |

Verify the wallets:

```bash
ls -l network/wallets/custodian1
ls -l network/wallets/investor1
ls -l network/wallets/investor2
ls -l network/wallets/tls-ca-cert.pem
```

Each identity should have `cert.pem` and `key.pem`.

## 6. Start the Orderer, Peer, and Fabric CLI Container

```bash
docker compose -f docker-compose.yml -f network/docker-compose.fabric.yml up -d orderer.example.com peer0.org1.example.com cli
```

The orderer is configured with `ORDERER_GENERAL_BOOTSTRAPMETHOD=none` and
`ORDERER_CHANNELPARTICIPATION_ENABLED=true`; it is joined to the application
channel by `osnadmin` in the next step rather than using a system-channel
genesis block.

Verify:

```bash
docker compose -f docker-compose.yml -f network/docker-compose.fabric.yml ps
docker logs orderer.example.com
docker logs peer0.org1.example.com
```

If Docker reports that a fixed container name is already in use, remove the stale container and retry:

```bash
docker rm -f orderer.example.com peer0.org1.example.com raxon-fabric-cli
```

## 7. Create the Channel

Run:

```bash
bash network/scripts/create-channel.sh
```

Expected output:

```text
Channel tokenization-channel created; orderer and peer0.org1 have joined.
```

If you manually run `osnadmin` from Git Bash, disable MSYS path conversion so
container paths such as `/organizations/...` are not changed into Windows host paths:

```bash
export MSYS_NO_PATHCONV=1
```

Then rerun the `osnadmin channel join` command. The certificate arguments must remain
container paths, for example:

```text
/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls/ca.crt
```

Verify channel membership:

```bash
docker exec raxon-fabric-cli peer channel list
```

Expected channel:

```text
tokenization-channel
```

If `configtxgen` reports a missing MSP CA certificate, verify these directories exist:

```bash
ls organizations/ordererOrganizations/example.com/msp/cacerts
ls organizations/peerOrganizations/org1.example.com/msp/cacerts
```

If `osnadmin channel join` reports that the consenter certificate is signed by an
unknown authority, the channel block and the mounted orderer certificates were
generated from different crypto material. Do a complete Fabric-only reset, then
regenerate crypto and the channel block without reusing the old artifacts:

```bash
docker compose -f docker-compose.yml -f network/docker-compose.fabric.yml -f network/docker-compose.clients.yml down
docker volume rm raxonDBmodule_orderer_data raxonDBmodule_peer0_org1_data 2>/dev/null || true
rm -rf organizations network/wallets channel-artifacts
docker compose -f docker-compose.yml -f network/docker-compose.fabric.yml up -d ca.orderer.example.com ca.org1.example.com
bash network/scripts/generate-crypto.sh
bash network/scripts/register-identities.sh
docker compose -f docker-compose.yml -f network/docker-compose.fabric.yml up -d orderer.example.com peer0.org1.example.com cli
bash network/scripts/create-channel.sh
```

Use the exact volume names shown by `docker volume ls` if the Compose project
name is different. Do not reuse an old `tokenization-channel.block` after
regenerating certificates.

## 8. Deploy the Tokenization Chaincode

Run:

```bash
bash network/scripts/deploy-chaincode.sh
```

The script vendors, packages, installs, approves, and commits the `tokenization` chaincode.

The Fabric CLI container mounts the repository path
`./chaincode/tokenization` at `/opt/gopath/src/github.com/hyperledger/fabric/peer/tokenization-chaincode`.
If the package command says `package tokenization-chaincode is not in std`,
recreate the CLI container so the current mount is applied:

```bash
docker compose -f docker-compose.yml -f network/docker-compose.fabric.yml up -d --force-recreate cli
```

The deployment script vendors dependencies on the host before packaging. If
Git Bash cannot find Go, add the Windows installation to the current shell:

```bash
export PATH="$PATH:/c/Program Files/Go/bin"
go version
```

Verify commitment:

```bash
docker exec raxon-fabric-cli peer lifecycle chaincode querycommitted --channelID tokenization-channel
```

The output should identify chaincode named `tokenization`.

## 9. Start the Interactive CLI Containers

Build and start the three identity-specific CLI containers:

```bash
docker compose -f docker-compose.yml -f network/docker-compose.fabric.yml -f network/docker-compose.clients.yml up -d --build
```

Verify:

```bash
docker ps --filter name=raxon-cli
```

Open three terminals and attach to the containers.

Custodian terminal:

```bash
docker attach raxon-cli-custodian
```

Investor 1 terminal:

```bash
docker attach raxon-cli-investor1
```

Investor 2 terminal:

```bash
docker attach raxon-cli-investor2
```

Detach without stopping a container by pressing `Ctrl-P`, then `Ctrl-Q`.

## 10. Get the Fabric Owner IDs

In the investor 1 CLI, run:

```text
whoami
```

Copy the complete result as `INVESTOR1_ID`.

In the investor 2 CLI, run:

```text
whoami
```

Copy the complete result as `INVESTOR2_ID`.

Use the exact returned values. The chaincode compares the supplied owner ID with the caller's Fabric identity.

## 11. Submit Custodian Transactions

In the custodian CLI, set the token NAV:

```text
setnav DUM-MMF-001 1.00049 0.12
```

Mint tokens to investor 1:

```text
mint <INVESTOR1_ID> DUM-MMF-001 1000
```

Mint tokens to investor 2:

```text
mint <INVESTOR2_ID> DUM-MMF-001 500
```

Query investor 1's balance from investor 1's CLI:

```text
query-balance <INVESTOR1_ID> DUM-MMF-001
```

Query investor 2's balance from investor 2's CLI:

```text
query-balance <INVESTOR2_ID> DUM-MMF-001
```

Expected initial transaction results:

```text
Investor 1: available 1000, encumbered 0
Investor 2: available 500, encumbered 0
```

## 12. Submit and Approve an Encumbrance

In investor 1's CLI:

```text
request-encumbrance <INVESTOR1_ID> DUM-MMF-001 200
```

Copy the request ID returned by the command.

In the custodian CLI:

```text
approve <REQUEST_ID>
```

Query investor 1's balance again:

```text
query-balance <INVESTOR1_ID> DUM-MMF-001
```

Expected:

```text
available 800
encumbered 200
```

## 13. Submit and Approve a Transfer

In investor 1's CLI:

```text
request-transfer <INVESTOR1_ID> <INVESTOR2_ID> DUM-MMF-001 250
```

Copy the request ID.

In the custodian CLI:

```text
approve <REQUEST_ID>
```

Query both balances:

```text
query-balance <INVESTOR1_ID> DUM-MMF-001
query-balance <INVESTOR2_ID> DUM-MMF-001
```

Expected balances after the encumbrance and transfer:

```text
Investor 1: available 550, encumbered 200
Investor 2: available 750, encumbered 0
```

## 14. Start the EPL Processor

The EPL service currently runs locally. Open a new PowerShell terminal and set its Fabric and database configuration:

```powershell
$env:RAXON_DATABASE_URL="postgres://postgres:postgres@localhost:5432/raxon?sslmode=disable"
$env:RAXON_CHANNEL_NAME="tokenization-channel"
$env:RAXON_FABRIC_PEER_ENDPOINT="localhost:7051"
$env:RAXON_FABRIC_GATEWAY_PEER="peer0.org1.example.com"
$env:RAXON_FABRIC_MSP_ID="Org1MSP"
$env:RAXON_FABRIC_CHAINCODE_NAME="tokenization"
$env:RAXON_FABRIC_TLS_CA_CERT_PATH="network/wallets/tls-ca-cert.pem"
$env:RAXON_FABRIC_CERT_PATH="network/wallets/custodian1/cert.pem"
$env:RAXON_FABRIC_KEY_PATH="network/wallets/custodian1/key.pem"
go run ./cmd/epl
```

Expected startup messages include:

```text
starting epl service channel=tokenization-channel
epl processor ready channel=tokenization-channel
```

The processor consumes committed blocks, creates block and transaction rows, stores events with PostgreSQL UUIDs, materializes projections, and updates the checkpoint.

Check the database from another terminal:

```bash
docker exec raxon-postgres psql -U postgres -d raxon -c "SELECT channel_name, block_height, block_hash FROM blocks ORDER BY block_height DESC LIMIT 5;"
docker exec raxon-postgres psql -U postgres -d raxon -c "SELECT channel_name, tx_id, block_height, tx_index, validation_code FROM transactions ORDER BY block_height DESC, tx_index DESC LIMIT 10;"
docker exec raxon-postgres psql -U postgres -d raxon -c "SELECT event_id, tx_id, event_type, processed_at FROM events ORDER BY created_at DESC LIMIT 10;"
docker exec raxon-postgres psql -U postgres -d raxon -c "SELECT channel_name, last_processed_height, last_processed_hash FROM checkpoint;"
docker exec raxon-postgres psql -U postgres -d raxon -c "SELECT owner_id, token_id, available, encumbered FROM balances WHERE token_id = 'DUM-MMF-001';"
```

If transactions were submitted before EPL started, it should replay from the stored checkpoint.

## 15. Start the Web Server

Open another PowerShell terminal:

```powershell
$env:RAXON_DATABASE_URL="postgres://postgres:postgres@localhost:5432/raxon?sslmode=disable"
$env:RAXON_PORT="8080"
$env:RAXON_STATIC_DIR="Stitch-files"
$env:RAXON_AUTH_MODE="dev"
$env:RAXON_ENV="development"
$env:RAXON_SESSION_SECRET="local-development-secret-at-least-32-bytes"
go run ./cmd/web
```

Expected output:

```text
web server listening on :8080
```

Open:

```text
http://localhost:8080/login
```

## 16. Verify the HTML Dashboards

### Investor 1

On the login page select:

- Role: `Investor`
- Development identity: `Investor 1`

Submit the form and refresh the dashboard with `Ctrl+F5`.

The investor dashboard reads:

```text
GET /api/investor/summary
GET /api/investor/operations
```

Investor 1 should see only the balance and operations mapped to investor 1's web user UUID.

### Investor 2

Log out at:

```text
http://localhost:8080/auth/logout
```

Return to `/login`, select:

- Role: `Investor`
- Development identity: `Investor 2`

Investor 2 should see the separate investor 2 balance and operations.

### Custodian

Log out and return to `/login`. Select:

- Role: `Custodian`
- Development identity: `Custodian 1`

The custodian dashboard reads:

```text
GET /api/custodian/queue
GET /api/custodian/stats
GET /api/custodian/audit-log
```

It should show processed operations and audit entries after EPL has consumed the corresponding blocks.

## 17. Test Rejection and Invalid Transactions

Invalid investor request:

```text
request-encumbrance <INVESTOR1_ID> DUM-MMF-001 -50
```

Expected error:

```text
amount must be positive
```

Unauthorized mint from investor 1:

```text
mint <INVESTOR1_ID> DUM-MMF-001 100
```

Expected error:

```text
caller is not authorized as custodian
```

A failed transaction must not change the successful balance projections.

## 18. Diagnostics

Container status:

```bash
docker ps -a
docker compose -f docker-compose.yml -f network/docker-compose.fabric.yml ps
```

Fabric logs:

```bash
docker logs ca.orderer.example.com
docker logs ca.org1.example.com
docker logs orderer.example.com
docker logs peer0.org1.example.com
docker logs raxon-fabric-cli
```

Fabric channel and chaincode status:

```bash
docker exec raxon-fabric-cli peer channel list
docker exec raxon-fabric-cli peer lifecycle chaincode querycommitted --channelID tokenization-channel
```

PostgreSQL readiness:

```bash
docker exec raxon-postgres pg_isready -U postgres -d raxon
```

## 19. Stop the Test Without Deleting Data

```bash
docker compose -f docker-compose.yml -f network/docker-compose.clients.yml down
docker compose -f docker-compose.yml -f network/docker-compose.fabric.yml down
```

## 20. Complete Cleanup

To remove all containers and volumes:

```bash
docker compose -f docker-compose.yml -f network/docker-compose.clients.yml down -v
docker compose -f docker-compose.yml -f network/docker-compose.fabric.yml down -v
```

Then remove generated Fabric state in Git Bash:

```bash
rm -rf organizations network/organizations network/wallets channel-artifacts
```

Repeat the test from Step 1 for a clean run.

## Expected End State

A successful test has all of the following:

- Fabric CLI transactions succeed for the correct identities.
- `blocks`, `transactions`, and `events` contain corresponding records.
- Event rows have valid PostgreSQL UUIDs.
- `checkpoint` advances after processed blocks.
- Investor 1 and investor 2 balances remain isolated.
- The custodian dashboard shows the request and approval activity.
- Investor dashboards show their mapped balances and operations.
