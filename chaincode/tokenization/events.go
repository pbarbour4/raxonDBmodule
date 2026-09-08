package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// emitEvent marshals payload to JSON and sets it as the chaincode event for Gateway subscribers.
func emitEvent(ctx contractapi.TransactionContextInterface, name string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event %s: %w", name, err)
	}
	if err := ctx.GetStub().SetEvent(name, data); err != nil {
		return fmt.Errorf("emit event %s: %w", name, err)
	}
	return nil
}
