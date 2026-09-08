package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-chaincode-go/v2/pkg/cid"
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// TokenContract implements custodian-only minting, burning, and NAV publication.
// 1 token = 1 share; amounts are whole units.
type TokenContract struct {
	contractapi.Contract
}

// Mint credits ownerID's available balance. Custodian-only.
func (t *TokenContract) Mint(ctx contractapi.TransactionContextInterface, ownerID, tokenID string, amount float64) error {
	if err := requireCustodian(ctx); err != nil {
		return err
	}
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	b, err := readBalanceState(ctx, ownerID, tokenID)
	if err != nil {
		return err
	}
	b.Available += amount
	b.Version++
	if err := writeBalanceState(ctx, b); err != nil {
		return err
	}
	return emitEvent(ctx, "MINT", map[string]any{
		"owner_id": ownerID, "token_id": tokenID, "amount": amount,
	})
}

// Burn debits ownerID's available balance. Custodian-only.
func (t *TokenContract) Burn(ctx contractapi.TransactionContextInterface, ownerID, tokenID string, amount float64) error {
	if err := requireCustodian(ctx); err != nil {
		return err
	}
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	b, err := readBalanceState(ctx, ownerID, tokenID)
	if err != nil {
		return err
	}
	if b.Available < amount {
		return fmt.Errorf("insufficient available balance to burn")
	}
	b.Available -= amount
	b.Version++
	if err := writeBalanceState(ctx, b); err != nil {
		return err
	}
	return emitEvent(ctx, "BURN", map[string]any{
		"owner_id": ownerID, "token_id": tokenID, "amount": amount,
	})
}

// SetNAV publishes the daily custodian-supplied price for tokenID. Custodian-only.
func (t *TokenContract) SetNAV(ctx contractapi.TransactionContextInterface, tokenID string, price, changePct float64) error {
	if err := requireCustodian(ctx); err != nil {
		return err
	}
	if price <= 0 {
		return fmt.Errorf("price must be positive")
	}
	key, err := navKey(ctx, tokenID)
	if err != nil {
		return err
	}
	nav := NAV{TokenID: tokenID, Price: price, ChangePct: changePct}
	data, err := json.Marshal(nav)
	if err != nil {
		return fmt.Errorf("marshal nav: %w", err)
	}
	if err := ctx.GetStub().PutState(key, data); err != nil {
		return fmt.Errorf("write nav: %w", err)
	}
	return emitEvent(ctx, "NAV_UPDATED", map[string]any{
		"token_id": tokenID, "price": price, "change_pct": changePct,
	})
}

// QueryBalance returns the current balance for ownerID/tokenID.
func (t *TokenContract) QueryBalance(ctx contractapi.TransactionContextInterface, ownerID, tokenID string) (*Balance, error) {
	b, err := readBalanceState(ctx, ownerID, tokenID)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// WhoAmI returns the caller's chain identity string, used to seed the ownerID that
// investor-initiated requests must match (see requireInvestorOwner).
func (t *TokenContract) WhoAmI(ctx contractapi.TransactionContextInterface) (string, error) {
	return cid.GetID(ctx.GetStub())
}
