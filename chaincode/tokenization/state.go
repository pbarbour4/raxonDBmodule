package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// Balance mirrors the Postgres balances table so EPL projections line up 1:1.
type Balance struct {
	OwnerID    string  `json:"owner_id"`
	TokenID    string  `json:"token_id"`
	Available  float64 `json:"available"`
	Encumbered float64 `json:"encumbered"`
	Version    int64   `json:"version"`
}

// RequestType enumerates the actions an investor can ask the custodian to approve.
type RequestType string

const (
	RequestTransfer      RequestType = "TRANSFER"
	RequestEncumbrance   RequestType = "ENCUMBRANCE"
	RequestUnencumbrance RequestType = "UNENCUMBRANCE"
)

// RequestStatus mirrors the Postgres operations.status enum.
type RequestStatus string

const (
	StatusRequested RequestStatus = "requested"
	StatusApproved  RequestStatus = "committed"
	StatusRejected  RequestStatus = "cancelled"
)

// Request represents a pending or resolved investor action awaiting custodian approval.
type Request struct {
	ID             string        `json:"id"`
	Type           RequestType   `json:"type"`
	OwnerID        string        `json:"owner_id"`
	CounterpartyID string        `json:"counterparty_id,omitempty"`
	TokenID        string        `json:"token_id"`
	Amount         float64       `json:"amount"`
	Status         RequestStatus `json:"status"`
	RequestedBy    string        `json:"requested_by"`
	Reason         string        `json:"reason,omitempty"`
}

// NAV mirrors the Postgres token_prices table.
type NAV struct {
	TokenID   string  `json:"token_id"`
	Price     float64 `json:"price"`
	ChangePct float64 `json:"change_pct"`
}

func balanceKey(ctx contractapi.TransactionContextInterface, ownerID, tokenID string) (string, error) {
	return ctx.GetStub().CreateCompositeKey("balance", []string{ownerID, tokenID})
}

func requestKey(ctx contractapi.TransactionContextInterface, requestID string) (string, error) {
	return ctx.GetStub().CreateCompositeKey("request", []string{requestID})
}

func navKey(ctx contractapi.TransactionContextInterface, tokenID string) (string, error) {
	return ctx.GetStub().CreateCompositeKey("nav", []string{tokenID})
}

// readBalanceState fetches ownerID/tokenID's balance, defaulting to a zero balance if unset.
func readBalanceState(ctx contractapi.TransactionContextInterface, ownerID, tokenID string) (Balance, error) {
	key, err := balanceKey(ctx, ownerID, tokenID)
	if err != nil {
		return Balance{}, err
	}
	data, err := ctx.GetStub().GetState(key)
	if err != nil {
		return Balance{}, fmt.Errorf("read balance: %w", err)
	}
	if data == nil {
		return Balance{OwnerID: ownerID, TokenID: tokenID}, nil
	}
	var b Balance
	if err := json.Unmarshal(data, &b); err != nil {
		return Balance{}, fmt.Errorf("unmarshal balance: %w", err)
	}
	return b, nil
}

// writeBalanceState persists b under its owner/token composite key.
func writeBalanceState(ctx contractapi.TransactionContextInterface, b Balance) error {
	key, err := balanceKey(ctx, b.OwnerID, b.TokenID)
	if err != nil {
		return err
	}
	data, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("marshal balance: %w", err)
	}
	return ctx.GetStub().PutState(key, data)
}

// readRequestState fetches a pending/resolved request by ID.
func readRequestState(ctx contractapi.TransactionContextInterface, requestID string) (Request, error) {
	key, err := requestKey(ctx, requestID)
	if err != nil {
		return Request{}, err
	}
	data, err := ctx.GetStub().GetState(key)
	if err != nil {
		return Request{}, fmt.Errorf("read request: %w", err)
	}
	if data == nil {
		return Request{}, fmt.Errorf("request %s not found", requestID)
	}
	var r Request
	if err := json.Unmarshal(data, &r); err != nil {
		return Request{}, fmt.Errorf("unmarshal request: %w", err)
	}
	return r, nil
}

// writeRequestState persists r under its request ID composite key.
func writeRequestState(ctx contractapi.TransactionContextInterface, r Request) error {
	key, err := requestKey(ctx, r.ID)
	if err != nil {
		return err
	}
	data, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	return ctx.GetStub().PutState(key, data)
}
