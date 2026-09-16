#!/usr/bin/env bash
# Registers the 2 investor and 1 custodian test identities against Org1's CA, each carrying a
# custom "role" attribute that the chaincode checks via cid.GetAttributeValue. Run after
# generate-crypto.sh. Produces flat wallets consumed by cmd/fabriccli (Phase D).
export MSYS_NO_PATHCONV=1
set -euo pipefail
cd "$(dirname "$0")/../.."

FABRIC_CA_CLIENT_IMAGE=hyperledger/fabric-ca:1.5
NET=raxon-fabric
ORGS_DIR="$(pwd)/organizations"
WALLETS_DIR="$(pwd)/network/wallets"

register_and_enroll() {
  local name=$1 secret=$2 role=$3
  docker run --rm --network "$NET" \
    -v "$ORGS_DIR:/organizations" \
    -e FABRIC_CA_CLIENT_HOME=/organizations/fabric-ca-client/ca-org1 \
    "$FABRIC_CA_CLIENT_IMAGE" sh -c "
      fabric-ca-client register --caname ca-org1 --id.name $name --id.secret $secret --id.type client \
        --id.attrs 'role=$role:ecert' \
        -u https://ca.org1.example.com:8054 --tls.certfiles /organizations/fabric-ca/org1/tls-cert.pem &&
      fabric-ca-client enroll -u https://$name:$secret@ca.org1.example.com:8054 \
        --caname ca-org1 -M /organizations/peerOrganizations/org1.example.com/users/$name/msp \
        --enrollment.attrs 'role' \
        --tls.certfiles /organizations/fabric-ca/org1/tls-cert.pem
    "
}

register_and_enroll custodian1 custodian1pw custodian
register_and_enroll investor1 investor1pw investor
register_and_enroll investor2 investor2pw investor

for name in custodian1 investor1 investor2; do
  src="$ORGS_DIR/peerOrganizations/org1.example.com/users/$name/msp"
  dst="$WALLETS_DIR/$name"
  mkdir -p "$dst"
  cp "$src"/signcerts/cert.pem "$dst/cert.pem"
  cp "$src"/keystore/*_sk "$dst/key.pem"
done

# Shared TLS root CA cert used by fabriccli to verify the peer's gateway TLS endpoint.
cp "$ORGS_DIR/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt" \
  "$WALLETS_DIR/tls-ca-cert.pem"

echo "Wallets written to $WALLETS_DIR/{custodian1,investor1,investor2}"
