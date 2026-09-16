package pipeline

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"raxonplatform/internal/contracts"

	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"google.golang.org/protobuf/proto"
)

type decodedBlock struct {
	height       uint64
	hash         string
	previousHash string
	timestamp    time.Time
	transactions []decodedTransaction
}

type decodedTransaction struct {
	txID           string
	index          int
	validationCode int
	timestamp      time.Time
	chaincode      string
	function       string
	args           []string
	events         []contracts.EventEnvelope
}

func decodeBlock(channelName, chaincodeName string, block *common.Block) (decodedBlock, error) {
	if block == nil || block.Header == nil || block.Data == nil {
		return decodedBlock{}, fmt.Errorf("fabric block is incomplete")
	}

	result := decodedBlock{
		height:       block.Header.Number,
		hash:         blockHeaderHash(block.Header),
		previousHash: fmt.Sprintf("%x", block.Header.PreviousHash),
	}
	validationCodes := transactionValidationCodes(block)

	for index, rawEnvelope := range block.Data.Data {
		transaction, ok, err := decodeTransaction(channelName, chaincodeName, rawEnvelope, uint64(block.Header.Number), index, validationCodeAt(validationCodes, index))
		if err != nil {
			return decodedBlock{}, fmt.Errorf("decode transaction %d: %w", index, err)
		}
		if !ok {
			continue
		}
		if result.timestamp.IsZero() {
			result.timestamp = transaction.timestamp
		}
		result.transactions = append(result.transactions, transaction)
	}
	return result, nil
}

func validationCodeAt(codes []int, index int) int {
	if index < len(codes) {
		return codes[index]
	}
	return 0
}

func blockHeaderHash(header *common.BlockHeader) string {
	encoded, err := proto.Marshal(header)
	if err != nil {
		return ""
	}
	hash := sha256.New()
	_, _ = hash.Write(encoded)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func transactionValidationCodes(block *common.Block) []int {
	if block.Metadata == nil || len(block.Metadata.Metadata) <= int(common.BlockMetadataIndex_TRANSACTIONS_FILTER) {
		return nil
	}
	raw := block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER]
	codes := make([]int, len(raw))
	for index, code := range raw {
		codes[index] = int(code)
	}
	return codes
}

func decodeTransaction(channelName, chaincodeName string, rawEnvelope []byte, blockHeight uint64, index int, validationCode int) (decodedTransaction, bool, error) {
	var envelope common.Envelope
	if err := proto.Unmarshal(rawEnvelope, &envelope); err != nil {
		return decodedTransaction{}, false, fmt.Errorf("unmarshal envelope: %w", err)
	}
	var payload common.Payload
	if err := proto.Unmarshal(envelope.Payload, &payload); err != nil {
		return decodedTransaction{}, false, fmt.Errorf("unmarshal payload: %w", err)
	}
	if payload.Header == nil {
		return decodedTransaction{}, false, fmt.Errorf("payload has no channel header")
	}
	header := common.ChannelHeader{}
	if err := proto.Unmarshal(payload.Header.ChannelHeader, &header); err != nil {
		return decodedTransaction{}, false, fmt.Errorf("unmarshal channel header: %w", err)
	}
	if common.HeaderType(header.Type) != common.HeaderType_ENDORSER_TRANSACTION {
		return decodedTransaction{}, false, nil
	}

	timestamp := time.Time{}
	if header.Timestamp != nil {
		timestamp = header.Timestamp.AsTime()
	}
	transaction := decodedTransaction{
		txID:           header.TxId,
		index:          index,
		validationCode: validationCode,
		timestamp:      timestamp,
	}

	var fabricTransaction peer.Transaction
	if err := proto.Unmarshal(payload.Data, &fabricTransaction); err != nil {
		return decodedTransaction{}, false, fmt.Errorf("unmarshal transaction: %w", err)
	}
	for _, action := range fabricTransaction.Actions {
		decodedAction, err := decodeAction(channelName, chaincodeName, transaction.txID, blockHeight, transaction.index, action, transaction.timestamp)
		if err != nil {
			return decodedTransaction{}, false, err
		}
		if decodedAction.chaincode != "" {
			transaction.chaincode = decodedAction.chaincode
		}
		if decodedAction.function != "" {
			transaction.function = decodedAction.function
		}
		transaction.args = decodedAction.args
		transaction.events = append(transaction.events, decodedAction.events...)
	}
	return transaction, true, nil
}

type decodedAction struct {
	chaincode string
	function  string
	args      []string
	events    []contracts.EventEnvelope
}

func decodeAction(channelName, chaincodeName, txID string, blockHeight uint64, txIndex int, action *peer.TransactionAction, timestamp time.Time) (decodedAction, error) {
	var actionPayload peer.ChaincodeActionPayload
	if err := proto.Unmarshal(action.Payload, &actionPayload); err != nil {
		return decodedAction{}, fmt.Errorf("unmarshal chaincode action payload: %w", err)
	}
	if actionPayload.Action == nil {
		return decodedAction{}, nil
	}
	var proposalPayload peer.ChaincodeProposalPayload
	if err := proto.Unmarshal(actionPayload.ChaincodeProposalPayload, &proposalPayload); err != nil {
		return decodedAction{}, fmt.Errorf("unmarshal proposal payload: %w", err)
	}
	var invocation peer.ChaincodeInvocationSpec
	if err := proto.Unmarshal(proposalPayload.Input, &invocation); err != nil {
		return decodedAction{}, fmt.Errorf("unmarshal invocation spec: %w", err)
	}
	result := decodedAction{}
	if invocation.ChaincodeSpec != nil {
		if invocation.ChaincodeSpec.ChaincodeId != nil {
			result.chaincode = invocation.ChaincodeSpec.ChaincodeId.Name
		}
		if invocation.ChaincodeSpec.Input != nil {
			for index, arg := range invocation.ChaincodeSpec.Input.Args {
				if index == 0 {
					result.function = string(arg)
					continue
				}
				result.args = append(result.args, string(arg))
			}
		}
	}

	var proposalResponse peer.ProposalResponsePayload
	if err := proto.Unmarshal(actionPayload.Action.ProposalResponsePayload, &proposalResponse); err != nil {
		return decodedAction{}, fmt.Errorf("unmarshal proposal response: %w", err)
	}
	var chaincodeAction peer.ChaincodeAction
	if err := proto.Unmarshal(proposalResponse.Extension, &chaincodeAction); err != nil {
		return decodedAction{}, fmt.Errorf("unmarshal chaincode action: %w", err)
	}
	if result.chaincode == "" && chaincodeAction.ChaincodeId != nil {
		result.chaincode = chaincodeAction.ChaincodeId.Name
	}
	if result.chaincode != chaincodeName {
		return result, nil
	}
	if len(chaincodeAction.Events) == 0 {
		return result, nil
	}
	var event peer.ChaincodeEvent
	if err := proto.Unmarshal(chaincodeAction.Events, &event); err != nil {
		return decodedAction{}, fmt.Errorf("unmarshal chaincode event: %w", err)
	}
	var payload map[string]any
	if len(event.Payload) > 0 {
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return decodedAction{}, fmt.Errorf("unmarshal chaincode event payload: %w", err)
		}
	}
	result.events = append(result.events, contracts.EventEnvelope{
		ChannelName:        channelName,
		BlockHeight:        int64(blockHeight),
		TxID:               txID,
		TxIndex:            txIndex,
		EventType:          event.EventName,
		ChaincodeEventName: event.EventName,
		Payload:            payload,
		OccurredAt:         timestamp,
		IngestedAt:         time.Now().UTC(),
	})
	return result, nil
}
