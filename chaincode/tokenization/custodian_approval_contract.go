package main

import (
	"fmt"

	"github.com/hyperledger/fabric-chaincode-go/v2/pkg/cid"
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// CustodianApprovalContract implements the investor-requests/custodian-approves lifecycle
// for Transfer, Encumbrance, and Unencumbrance.
type CustodianApprovalContract struct {
	contractapi.Contract
}

func newRequest(ctx contractapi.TransactionContextInterface, reqType RequestType, ownerID, counterpartyID, tokenID string, amount float64) (Request, error) {
	if amount <= 0 {
		return Request{}, fmt.Errorf("amount must be positive")
	}
	if err := requireInvestorOwner(ctx, ownerID); err != nil {
		return Request{}, err
	}
	callerID, err := cid.GetID(ctx.GetStub())
	if err != nil {
		return Request{}, fmt.Errorf("read caller id: %w", err)
	}
	r := Request{
		ID:             ctx.GetStub().GetTxID(),
		Type:           reqType,
		OwnerID:        ownerID,
		CounterpartyID: counterpartyID,
		TokenID:        tokenID,
		Amount:         amount,
		Status:         StatusRequested,
		RequestedBy:    callerID,
	}
	if err := writeRequestState(ctx, r); err != nil {
		return Request{}, err
	}
	return r, nil
}

// RequestTransfer asks the custodian to move amount from ownerID to toOwnerID. Investor-only.
func (c *CustodianApprovalContract) RequestTransfer(ctx contractapi.TransactionContextInterface, ownerID, toOwnerID, tokenID string, amount float64) (*Request, error) {
	r, err := newRequest(ctx, RequestTransfer, ownerID, toOwnerID, tokenID, amount)
	if err != nil {
		return nil, err
	}
	if err := emitEvent(ctx, "TRANSFER_REQUESTED", requestEventPayload(r)); err != nil {
		return nil, err
	}
	return &r, nil
}

// RequestEncumbrance asks the custodian to move amount from available to encumbered. Investor-only.
func (c *CustodianApprovalContract) RequestEncumbrance(ctx contractapi.TransactionContextInterface, ownerID, tokenID string, amount float64) (*Request, error) {
	r, err := newRequest(ctx, RequestEncumbrance, ownerID, "", tokenID, amount)
	if err != nil {
		return nil, err
	}
	if err := emitEvent(ctx, "ENCUMBRANCE_REQUESTED", requestEventPayload(r)); err != nil {
		return nil, err
	}
	return &r, nil
}

// RequestUnencumbrance asks the custodian to move amount from encumbered back to available. Investor-only.
func (c *CustodianApprovalContract) RequestUnencumbrance(ctx contractapi.TransactionContextInterface, ownerID, tokenID string, amount float64) (*Request, error) {
	r, err := newRequest(ctx, RequestUnencumbrance, ownerID, "", tokenID, amount)
	if err != nil {
		return nil, err
	}
	if err := emitEvent(ctx, "UNENCUMBRANCE_REQUESTED", requestEventPayload(r)); err != nil {
		return nil, err
	}
	return &r, nil
}

// ApproveRequest applies a pending request's balance mutation. Custodian-only.
func (c *CustodianApprovalContract) ApproveRequest(ctx contractapi.TransactionContextInterface, requestID string) error {
	if err := requireCustodian(ctx); err != nil {
		return err
	}
	r, err := readRequestState(ctx, requestID)
	if err != nil {
		return err
	}
	if r.Status != StatusRequested {
		return fmt.Errorf("request %s is not pending (status=%s)", requestID, r.Status)
	}

	owner, err := readBalanceState(ctx, r.OwnerID, r.TokenID)
	if err != nil {
		return err
	}

	switch r.Type {
	case RequestTransfer:
		if owner.Available < r.Amount {
			return fmt.Errorf("insufficient available balance for transfer")
		}
		counterparty, err := readBalanceState(ctx, r.CounterpartyID, r.TokenID)
		if err != nil {
			return err
		}
		owner.Available -= r.Amount
		owner.Version++
		counterparty.Available += r.Amount
		counterparty.Version++
		if err := writeBalanceState(ctx, owner); err != nil {
			return err
		}
		if err := writeBalanceState(ctx, counterparty); err != nil {
			return err
		}
	case RequestEncumbrance:
		if owner.Available < r.Amount {
			return fmt.Errorf("insufficient available balance to encumber")
		}
		owner.Available -= r.Amount
		owner.Encumbered += r.Amount
		owner.Version++
		if err := writeBalanceState(ctx, owner); err != nil {
			return err
		}
	case RequestUnencumbrance:
		if owner.Encumbered < r.Amount {
			return fmt.Errorf("insufficient encumbered balance to release")
		}
		owner.Encumbered -= r.Amount
		owner.Available += r.Amount
		owner.Version++
		if err := writeBalanceState(ctx, owner); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown request type %s", r.Type)
	}

	r.Status = StatusApproved
	if err := writeRequestState(ctx, r); err != nil {
		return err
	}
	return emitEvent(ctx, string(r.Type)+"_APPROVED", requestEventPayload(r))
}

// RejectRequest marks a pending request as cancelled without mutating balances. Custodian-only.
func (c *CustodianApprovalContract) RejectRequest(ctx contractapi.TransactionContextInterface, requestID, reason string) error {
	if err := requireCustodian(ctx); err != nil {
		return err
	}
	r, err := readRequestState(ctx, requestID)
	if err != nil {
		return err
	}
	if r.Status != StatusRequested {
		return fmt.Errorf("request %s is not pending (status=%s)", requestID, r.Status)
	}
	r.Status = StatusRejected
	r.Reason = reason
	if err := writeRequestState(ctx, r); err != nil {
		return err
	}
	return emitEvent(ctx, string(r.Type)+"_REJECTED", requestEventPayload(r))
}

// QueryRequest returns a request by ID for CLI convenience.
func (c *CustodianApprovalContract) QueryRequest(ctx contractapi.TransactionContextInterface, requestID string) (*Request, error) {
	r, err := readRequestState(ctx, requestID)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func requestEventPayload(r Request) map[string]any {
	return map[string]any{
		"request_id":      r.ID,
		"owner_id":        r.OwnerID,
		"counterparty_id": r.CounterpartyID,
		"token_id":        r.TokenID,
		"amount":          r.Amount,
		"status":          r.Status,
		"requested_by":    r.RequestedBy,
		"reason":          r.Reason,
	}
}
