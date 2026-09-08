#!/usr/bin/env bash
# Vendors, packages, installs, approves, and commits the tokenization chaincode onto
# tokenization-channel. Run after create-channel.sh, with the cli container up.
set -euo pipefail
cd "$(dirname "$0")/.."

CHANNEL_NAME=tokenization-channel
CC_NAME=tokenization
CC_VERSION=1
CC_SEQUENCE=1
CC_LABEL="${CC_NAME}_${CC_VERSION}"
ORDERER_CA=/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls/ca.crt

# peer lifecycle chaincode packaging needs all deps present locally (no network in the cli container).
(cd ../chaincode/tokenization && GO111MODULE=on go mod vendor)

docker exec raxon-fabric-cli peer lifecycle chaincode package "${CC_LABEL}.tar.gz" \
  --path tokenization-chaincode --lang golang --label "$CC_LABEL"

docker exec raxon-fabric-cli peer lifecycle chaincode install "${CC_LABEL}.tar.gz"

PACKAGE_ID=$(docker exec raxon-fabric-cli peer lifecycle chaincode queryinstalled \
  | grep "$CC_LABEL" | sed -n 's/^Package ID: \(.*\), Label:.*$/\1/p')

if [ -z "$PACKAGE_ID" ]; then
  echo "failed to resolve installed package ID for $CC_LABEL" >&2
  exit 1
fi

docker exec raxon-fabric-cli peer lifecycle chaincode approveformyorg \
  --channelID "$CHANNEL_NAME" --name "$CC_NAME" --version "$CC_VERSION" \
  --package-id "$PACKAGE_ID" --sequence "$CC_SEQUENCE" \
  --orderer orderer.example.com:7050 --tls --cafile "$ORDERER_CA"

docker exec raxon-fabric-cli peer lifecycle chaincode checkcommitreadiness \
  --channelID "$CHANNEL_NAME" --name "$CC_NAME" --version "$CC_VERSION" --sequence "$CC_SEQUENCE" \
  --orderer orderer.example.com:7050 --tls --cafile "$ORDERER_CA" --output json

docker exec raxon-fabric-cli peer lifecycle chaincode commit \
  --channelID "$CHANNEL_NAME" --name "$CC_NAME" --version "$CC_VERSION" --sequence "$CC_SEQUENCE" \
  --orderer orderer.example.com:7050 --tls --cafile "$ORDERER_CA" \
  --peerAddresses peer0.org1.example.com:7051 \
  --tlsRootCertFiles /organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt

echo "Chaincode $CC_NAME committed on $CHANNEL_NAME (package $PACKAGE_ID)."
