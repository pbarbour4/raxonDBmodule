package pipeline

import (
	"context"
	"fmt"
	"strings"

	"raxonplatform/internal/contracts"
)

// materialize projects a persisted chaincode event onto the off-chain query model
// (balances, token_prices, operations, encumbrances, audit_trail).
func (p *Processor) materialize(ctx context.Context, e contracts.EventEnvelope) error {
	switch e.EventType {
	case "MINT":
		return p.applyBalanceEvent(ctx, e, 1)
	case "BURN":
		return p.applyBalanceEvent(ctx, e, -1)
	case "NAV_UPDATED":
		return p.applyNAV(ctx, e)
	default:
		switch {
		case strings.HasSuffix(e.EventType, "_REQUESTED"):
			return p.applyRequested(ctx, e)
		case strings.HasSuffix(e.EventType, "_APPROVED"):
			return p.applyApproved(ctx, e)
		case strings.HasSuffix(e.EventType, "_REJECTED"):
			return p.applyRejected(ctx, e)
		}
	}
	return nil
}

func (p *Processor) applyBalanceEvent(ctx context.Context, e contracts.EventEnvelope, sign float64) error {
	ownerID, _ := e.Payload["owner_id"].(string)
	ownerID, err := p.repo.ResolveFabricIdentity(ctx, ownerID)
	if err != nil {
		return err
	}
	tokenID, _ := e.Payload["token_id"].(string)
	amount, _ := e.Payload["amount"].(float64)
	return p.repo.UpsertBalanceDelta(ctx, ownerID, tokenID, sign*amount, 0, e.BlockHeight)
}

func (p *Processor) applyNAV(ctx context.Context, e contracts.EventEnvelope) error {
	tokenID, _ := e.Payload["token_id"].(string)
	price, _ := e.Payload["price"].(float64)
	changePct, _ := e.Payload["change_pct"].(float64)
	return p.repo.UpsertTokenPrice(ctx, tokenID, price, changePct)
}

func (p *Processor) applyRequested(ctx context.Context, e contracts.EventEnvelope) error {
	requestID, _ := e.Payload["request_id"].(string)
	requestedBy, _ := e.Payload["requested_by"].(string)
	requestedBy, err := p.repo.ResolveFabricIdentity(ctx, requestedBy)
	if err != nil {
		return err
	}
	payload := canonicalizePayload(e.Payload, p.repo, ctx)
	opType := strings.TrimSuffix(e.EventType, "_REQUESTED")
	return p.repo.InsertOperationFromEvent(ctx, requestID, opType, "requested", requestedBy, payload, e.TxID)
}

func (p *Processor) applyApproved(ctx context.Context, e contracts.EventEnvelope) error {
	requestID, _ := e.Payload["request_id"].(string)
	ownerID, _ := e.Payload["owner_id"].(string)
	counterpartyID, _ := e.Payload["counterparty_id"].(string)
	resolvedOwnerID, err := p.repo.ResolveFabricIdentity(ctx, ownerID)
	if err != nil {
		return err
	}
	ownerID = resolvedOwnerID
	if counterpartyID != "" {
		counterpartyID, err = p.repo.ResolveFabricIdentity(ctx, counterpartyID)
		if err != nil {
			return err
		}
	}
	tokenID, _ := e.Payload["token_id"].(string)
	amount, _ := e.Payload["amount"].(float64)
	opType := strings.TrimSuffix(e.EventType, "_APPROVED")

	if err := p.repo.SetOperationStatus(ctx, requestID, "committed"); err != nil {
		return err
	}

	switch opType {
	case "TRANSFER":
		if err := p.repo.UpsertBalanceDelta(ctx, ownerID, tokenID, -amount, 0, e.BlockHeight); err != nil {
			return err
		}
		if err := p.repo.UpsertBalanceDelta(ctx, counterpartyID, tokenID, amount, 0, e.BlockHeight); err != nil {
			return err
		}
	case "ENCUMBRANCE":
		if err := p.repo.UpsertBalanceDelta(ctx, ownerID, tokenID, -amount, amount, e.BlockHeight); err != nil {
			return err
		}
		if err := p.repo.InsertEncumbrance(ctx, ownerID, tokenID, amount, requestID, e.BlockHeight); err != nil {
			return err
		}
	case "UNENCUMBRANCE":
		if err := p.repo.UpsertBalanceDelta(ctx, ownerID, tokenID, amount, -amount, e.BlockHeight); err != nil {
			return err
		}
		if err := p.repo.ReleaseEncumbrance(ctx, requestID, e.BlockHeight); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown approved operation type %q", opType)
	}

	return p.repo.InsertAuditTrail(ctx, "operation", requestID, "STATUS_CHANGE",
		map[string]string{"status": "requested"}, e.Payload, e.TxID, e.BlockHeight)
}

func (p *Processor) applyRejected(ctx context.Context, e contracts.EventEnvelope) error {
	requestID, _ := e.Payload["request_id"].(string)
	if err := p.repo.SetOperationStatus(ctx, requestID, "cancelled"); err != nil {
		return err
	}
	return p.repo.InsertAuditTrail(ctx, "operation", requestID, "STATUS_CHANGE",
		map[string]string{"status": "requested"}, e.Payload, e.TxID, e.BlockHeight)
}

func canonicalizePayload(payload map[string]any, repo interface {
	ResolveFabricIdentity(context.Context, string) (string, error)
}, ctx context.Context) map[string]any {
	canonical := make(map[string]any, len(payload))
	for key, value := range payload {
		canonical[key] = value
	}
	for _, key := range []string{"owner_id", "counterparty_id"} {
		identity, ok := canonical[key].(string)
		if !ok || identity == "" {
			continue
		}
		if userID, err := repo.ResolveFabricIdentity(ctx, identity); err == nil {
			canonical[key] = userID
		}
	}
	return canonical
}
