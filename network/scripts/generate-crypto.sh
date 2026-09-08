#!/usr/bin/env bash
# Generates crypto material for the orderer org and Org1 using the two Fabric CA
# containers defined in docker-compose.fabric.yml. Run once, after the CAs are up:
#   docker compose -f ../docker-compose.yml -f docker-compose.fabric.yml up -d ca.orderer.example.com ca.org1.example.com
#   network/scripts/generate-crypto.sh
set -euo pipefail
cd "$(dirname "$0")/.."

FABRIC_CA_CLIENT_IMAGE=hyperledger/fabric-ca:1.5
NET=raxon-fabric
ORGS_DIR="$(pwd)/organizations"

run_ca_client() {
  local ca_name=$1; shift
  docker run --rm --network "$NET" \
    -v "$ORGS_DIR:/organizations" \
    -e FABRIC_CA_CLIENT_HOME=/organizations/fabric-ca-client \
    "$FABRIC_CA_CLIENT_IMAGE" sh -c "$*"
}

mkdir -p "$ORGS_DIR"

# ── Orderer org ──────────────────────────────────────────────────────────────
run_ca_client ca-orderer "
  fabric-ca-client enroll -u https://admin:adminpw@ca.orderer.example.com:7054 --tls.certfiles /organizations/fabric-ca/ordererOrg/tls-cert.pem &&
  fabric-ca-client register --caname ca-orderer --id.name orderer --id.secret ordererpw --id.type orderer \
    -u https://ca.orderer.example.com:7054 --tls.certfiles /organizations/fabric-ca/ordererOrg/tls-cert.pem &&
  fabric-ca-client register --caname ca-orderer --id.name ordererAdmin --id.secret ordererAdminpw --id.type admin \
    -u https://ca.orderer.example.com:7054 --tls.certfiles /organizations/fabric-ca/ordererOrg/tls-cert.pem &&
  fabric-ca-client enroll -u https://orderer:ordererpw@ca.orderer.example.com:7054 \
    --caname ca-orderer -M /organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp \
    --tls.certfiles /organizations/fabric-ca/ordererOrg/tls-cert.pem &&
  fabric-ca-client enroll -u https://orderer:ordererpw@ca.orderer.example.com:7054 \
    --caname ca-orderer -M /organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls \
    --enrollment.profile tls --csr.hosts orderer.example.com --csr.hosts localhost \
    --tls.certfiles /organizations/fabric-ca/ordererOrg/tls-cert.pem &&
  fabric-ca-client enroll -u https://ordererAdmin:ordererAdminpw@ca.orderer.example.com:7054 \
    --caname ca-orderer -M /organizations/ordererOrganizations/example.com/users/Admin@example.com/msp \
    --tls.certfiles /organizations/fabric-ca/ordererOrg/tls-cert.pem
"

# Fabric expects tls/{server.key,server.crt,ca.crt}; the CA client writes keystore/*_sk and
# signcerts/cert.pem, so normalize names for the components that read fixed filenames.
for d in "$ORGS_DIR/ordererOrganizations/example.com/orderers/orderer.example.com/tls"; do
  cp "$d"/signcerts/cert.pem "$d/server.crt"
  cp "$d"/keystore/*_sk "$d/server.key"
  cp "$d"/tlscacerts/*.pem "$d/ca.crt"
done

# ── Org1 ─────────────────────────────────────────────────────────────────────
run_ca_client ca-org1 "
  fabric-ca-client enroll -u https://admin:adminpw@ca.org1.example.com:8054 --tls.certfiles /organizations/fabric-ca/org1/tls-cert.pem &&
  fabric-ca-client register --caname ca-org1 --id.name peer0 --id.secret peer0pw --id.type peer \
    -u https://ca.org1.example.com:8054 --tls.certfiles /organizations/fabric-ca/org1/tls-cert.pem &&
  fabric-ca-client register --caname ca-org1 --id.name org1admin --id.secret org1adminpw --id.type admin \
    -u https://ca.org1.example.com:8054 --tls.certfiles /organizations/fabric-ca/org1/tls-cert.pem &&
  fabric-ca-client enroll -u https://peer0:peer0pw@ca.org1.example.com:8054 \
    --caname ca-org1 -M /organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/msp \
    --tls.certfiles /organizations/fabric-ca/org1/tls-cert.pem &&
  fabric-ca-client enroll -u https://peer0:peer0pw@ca.org1.example.com:8054 \
    --caname ca-org1 -M /organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls \
    --enrollment.profile tls --csr.hosts peer0.org1.example.com --csr.hosts localhost \
    --tls.certfiles /organizations/fabric-ca/org1/tls-cert.pem &&
  fabric-ca-client enroll -u https://org1admin:org1adminpw@ca.org1.example.com:8054 \
    --caname ca-org1 -M /organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp \
    --tls.certfiles /organizations/fabric-ca/org1/tls-cert.pem
"

for d in "$ORGS_DIR/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls"; do
  cp "$d"/signcerts/cert.pem "$d/server.crt"
  cp "$d"/keystore/*_sk "$d/server.key"
  cp "$d"/tlscacerts/*.pem "$d/ca.crt"
done

echo "Crypto material generated under $ORGS_DIR"
echo "Next: network/scripts/register-identities.sh, then network/scripts/create-channel.sh"
