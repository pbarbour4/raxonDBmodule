#!/usr/bin/env bash
# Creates the tokenization-channel genesis block and joins the orderer and peer0.org1 to it.
# Run after generate-crypto.sh, with the orderer/peer/cli containers up.
export MSYS_NO_PATHCONV=1
set -euo pipefail
cd "$(dirname "$0")/../.."

CHANNEL_NAME=tokenization-channel
TOOLS_IMAGE=hyperledger/fabric-tools:2.5
NET=raxon-fabric
ROOT="$(pwd)"

mkdir -p "$ROOT/channel-artifacts"

docker run --rm --network "$NET" \
  -v "$ROOT/network/configtx:/configtx" \
  -v "$ROOT/organizations:/organizations" \
  -v "$ROOT/channel-artifacts:/channel-artifacts" \
  -e FABRIC_CFG_PATH=/configtx \
  "$TOOLS_IMAGE" configtxgen \
    -profile TokenizationChannel -outputBlock "/channel-artifacts/${CHANNEL_NAME}.block" -channelID "$CHANNEL_NAME"

docker run --rm --network "$NET" \
  -v "$ROOT/organizations:/organizations" \
  -v "$ROOT/channel-artifacts:/channel-artifacts" \
  "$TOOLS_IMAGE" osnadmin channel join \
    --channelID "$CHANNEL_NAME" \
    --config-block "/channel-artifacts/${CHANNEL_NAME}.block" \
    -o orderer.example.com:7053 \
    --ca-file /organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls/ca.crt \
    --client-cert /organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls/server.crt \
    --client-key /organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls/server.key

docker exec raxon-fabric-cli peer channel join -b "channel-artifacts/${CHANNEL_NAME}.block"

echo "Channel $CHANNEL_NAME created; orderer and peer0.org1 have joined."
