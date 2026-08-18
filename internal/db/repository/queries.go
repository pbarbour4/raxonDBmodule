package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ── Model types ──────────────────────────────────────────────────────────────

type User struct {
	ID              string    `json:"id"`
	Email           string    `json:"email"`
	Role            string    `json:"role"`
	OIDCSubject     string    `json:"oidc_subject"`
	InstitutionName string    `json:"institution_name"`
	CreatedAt       time.Time `json:"created_at"`
}

type Operation struct {
	OperationID    string         `json:"operation_id"`
	Type           string         `json:"type"`
	Status         string         `json:"status"`
	RequestedBy    string         `json:"requested_by"`
	RequestPayload map[string]any `json:"request_payload"`
	CreatedAt      time.Time      `json:"created_at"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty"`
}

type InvestorSummary struct {
	Available    float64   `json:"available"`
	Encumbered   float64   `json:"encumbered"`
	TotalValue   float64   `json:"total_value"`
	Allocation   float64   `json:"allocation_pct"`
	TokenPrice   float64   `json:"token_price"`
	LastSyncAt   time.Time `json:"last_sync_at"`
	LastSyncHash string    `json:"last_sync_hash"`
}

type LenderSummary struct {
	TokenPrice      float64 `json:"token_price"`
	TokenChangePct  float64 `json:"token_change_pct"`
	TotalCollateral float64 `json:"total_collateral_usd"`
	WeightedLTV     float64 `json:"weighted_ltv_pct"`
	TotalLoans      float64 `json:"total_loans_usd"`
	InvestorCount   int     `json:"investor_count"`
}

type InvestorExposure struct {
	OwnerID         string  `json:"owner_id"`
	InstitutionName string  `json:"institution_name"`
	LoanBalance     float64 `json:"loan_balance"`
	CollateralDUM   float64 `json:"collateral_dum"`
	LTVRatio        float64 `json:"ltv_ratio_pct"`
	Headroom        float64 `json:"headroom_usd"`
	Status          string  `json:"status"`
}

type RiskBucket struct {
	Label   string  `json:"label"`
	Count   int     `json:"count"`
	Percent float64 `json:"percent"`
}

type AuditEntry struct {
	ID         int64          `json:"id"`
	EntityType string         `json:"entity_type"`
	EntityID   string         `json:"entity_id"`
	ChangeType string         `json:"change_type"`
	After      map[string]any `json:"after,omitempty"`
	Timestamp  time.Time      `json:"timestamp"`
}

type OperationStats struct {
	PendingCount     int     `json:"pending_count"`
	EncumberedVolume float64 `json:"encumbered_volume_usd"`
	AvgResponseHours float64 `json:"avg_response_hours"`
}

// ── User queries ─────────────────────────────────────────────────────────────

func (r *Repository) GetUserByOIDCSubject(ctx context.Context, subject string) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, role, oidc_subject, institution_name, created_at
		FROM users WHERE oidc_subject = $1
	`, subject).Scan(&u.ID, &u.Email, &u.Role, &u.OIDCSubject, &u.InstitutionName, &u.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by oidc subject: %w", err)
	}
	return &u, nil
}

func (r *Repository) UpsertUser(ctx context.Context, u User) (User, error) {
	var out User
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (id, email, role, oidc_subject, institution_name)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (oidc_subject) DO UPDATE SET
			email            = EXCLUDED.email,
			institution_name = EXCLUDED.institution_name
		RETURNING id, email, role, oidc_subject, institution_name, created_at
	`, u.ID, u.Email, u.Role, u.OIDCSubject, u.InstitutionName,
	).Scan(&out.ID, &out.Email, &out.Role, &out.OIDCSubject, &out.InstitutionName, &out.CreatedAt)
	if err != nil {
		return User{}, fmt.Errorf("upsert user: %w", err)
	}
	return out, nil
}

// ── Investor queries ──────────────────────────────────────────────────────────

func (r *Repository) GetInvestorSummary(ctx context.Context, ownerID, channelName string) (InvestorSummary, error) {
	var s InvestorSummary
	err := r.pool.QueryRow(ctx, `
		SELECT
			b.available,
			b.encumbered,
			(b.available + b.encumbered) * p.price                                          AS total_value,
			CASE WHEN b.available + b.encumbered > 0
				 THEN ROUND((b.encumbered / (b.available + b.encumbered) * 100)::numeric, 1)
				 ELSE 0 END                                                                 AS allocation_pct,
			p.price,
			COALESCE(c.updated_at,  NOW())                                                  AS last_sync_at,
			COALESCE(c.last_processed_hash, '')                                             AS last_sync_hash
		FROM   balances b
		JOIN   token_prices p  ON p.token_id = b.token_id
		LEFT   JOIN checkpoint c ON c.channel_name = $2
		WHERE  b.owner_id = $1 AND b.token_id = 'DUM-MMF-001'
	`, ownerID, channelName).Scan(
		&s.Available, &s.Encumbered, &s.TotalValue,
		&s.Allocation, &s.TokenPrice, &s.LastSyncAt, &s.LastSyncHash,
	)
	if err == pgx.ErrNoRows {
		return InvestorSummary{}, nil
	}
	if err != nil {
		return InvestorSummary{}, fmt.Errorf("get investor summary: %w", err)
	}
	return s, nil
}

func (r *Repository) GetOperationsByOwner(ctx context.Context, ownerID string, limit, offset int) ([]Operation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT operation_id, type, status, requested_by, request_payload, created_at, completed_at
		FROM   operations
		WHERE  request_payload->>'owner_id' = $1
		ORDER  BY created_at DESC
		LIMIT  $2 OFFSET $3
	`, ownerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get operations by owner: %w", err)
	}
	return scanOperations(rows)
}

// ── Lender queries ────────────────────────────────────────────────────────────

func (r *Repository) GetLenderSummary(ctx context.Context) (LenderSummary, error) {
	var s LenderSummary
	err := r.pool.QueryRow(ctx, `
		SELECT
			p.price,
			p.change_pct,
			COALESCE(SUM(l.collateral_amount * p.price), 0)                               AS total_collateral,
			CASE WHEN SUM(l.collateral_amount * p.price) > 0
				 THEN ROUND((SUM(l.loan_balance) / SUM(l.collateral_amount * p.price) * 100)::numeric, 1)
				 ELSE 0 END                                                                AS weighted_ltv,
			COALESCE(SUM(l.loan_balance), 0)                                               AS total_loans,
			COUNT(*)::int                                                                   AS investor_count
		FROM   token_prices p
		LEFT   JOIN loans l ON l.token_id = p.token_id AND l.status != 'closed'
		WHERE  p.token_id = 'DUM-MMF-001'
		GROUP  BY p.price, p.change_pct
	`).Scan(&s.TokenPrice, &s.TokenChangePct, &s.TotalCollateral,
		&s.WeightedLTV, &s.TotalLoans, &s.InvestorCount)
	if err != nil {
		return LenderSummary{}, fmt.Errorf("get lender summary: %w", err)
	}
	return s, nil
}

func (r *Repository) GetLenderExposure(ctx context.Context, limit, offset int) ([]InvestorExposure, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			l.owner_id,
			l.institution_name,
			l.loan_balance,
			l.collateral_amount,
			ROUND((l.loan_balance / NULLIF(l.collateral_amount * p.price, 0) * 100)::numeric, 1) AS ltv_ratio,
			l.collateral_amount * p.price - l.loan_balance                                        AS headroom,
			l.status
		FROM   loans l
		JOIN   token_prices p ON p.token_id = l.token_id
		WHERE  l.status != 'closed'
		ORDER  BY l.loan_balance DESC
		LIMIT  $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get lender exposure: %w", err)
	}
	defer rows.Close()
	var out []InvestorExposure
	for rows.Next() {
		var e InvestorExposure
		if err := rows.Scan(&e.OwnerID, &e.InstitutionName, &e.LoanBalance, &e.CollateralDUM,
			&e.LTVRatio, &e.Headroom, &e.Status); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) GetLenderExposureCount(ctx context.Context) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM loans WHERE status != 'closed'`).Scan(&n)
	return n, err
}

func (r *Repository) GetRiskDistribution(ctx context.Context) ([]RiskBucket, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			CASE
				WHEN l.loan_balance / NULLIF(l.collateral_amount * p.price, 0) * 100 < 50 THEN 'Low Risk (LTV < 50%)'
				WHEN l.loan_balance / NULLIF(l.collateral_amount * p.price, 0) * 100 < 70 THEN 'Moderate (LTV 50-70%)'
				ELSE 'Elevated (LTV > 70%)'
			END                                                             AS label,
			COUNT(*)::int                                                   AS cnt,
			ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER (), 1)::float      AS pct
		FROM  loans l
		JOIN  token_prices p ON p.token_id = l.token_id
		WHERE l.status != 'closed'
		GROUP BY 1
		ORDER BY
			CASE label
				WHEN 'Low Risk (LTV < 50%)'    THEN 1
				WHEN 'Moderate (LTV 50-70%)'   THEN 2
				ELSE 3
			END
	`)
	if err != nil {
		return nil, fmt.Errorf("get risk distribution: %w", err)
	}
	defer rows.Close()
	var out []RiskBucket
	for rows.Next() {
		var b RiskBucket
		if err := rows.Scan(&b.Label, &b.Count, &b.Percent); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// ── Custodian queries ─────────────────────────────────────────────────────────

func (r *Repository) GetPendingOperations(ctx context.Context, limit, offset int) ([]Operation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT operation_id, type, status, requested_by, request_payload, created_at, completed_at
		FROM   operations
		WHERE  status = 'requested'
		ORDER  BY created_at ASC
		LIMIT  $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get pending operations: %w", err)
	}
	return scanOperations(rows)
}

func (r *Repository) CountPendingOperations(ctx context.Context) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM operations WHERE status = 'requested'`).Scan(&n)
	return n, err
}

func (r *Repository) GetOperationStats(ctx context.Context) (OperationStats, error) {
	var s OperationStats
	err := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'requested')::int                                AS pending_count,
			COALESCE(SUM((request_payload->>'usd_value')::numeric)
				FILTER (WHERE status IN ('requested','accepted','submitted','committed')), 0) AS encumbered_volume,
			COALESCE(ROUND(AVG(EXTRACT(EPOCH FROM (completed_at - created_at)) / 3600)::numeric
				FILTER (WHERE completed_at IS NOT NULL), 1)::float, 0)                       AS avg_response_hours
		FROM operations
	`).Scan(&s.PendingCount, &s.EncumberedVolume, &s.AvgResponseHours)
	if err != nil {
		return OperationStats{}, fmt.Errorf("get operation stats: %w", err)
	}
	return s, nil
}

func (r *Repository) ApproveOperation(ctx context.Context, operationID, approvedBy string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE operations SET status = 'accepted', completed_at = NOW()
		WHERE  operation_id = $1 AND status = 'requested'
	`, operationID)
	if err != nil {
		return fmt.Errorf("approve operation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("operation %s not found or not in requested state", operationID)
	}
	afterJSON, _ := json.Marshal(map[string]string{
		"status":      "accepted",
		"approved_by": approvedBy,
		"note":        "Operation approved by custodian",
	})
	_, err = r.pool.Exec(ctx, `
		INSERT INTO audit_trail (entity_type, entity_id, change_type, before, after)
		VALUES ('operation', $1, 'STATUS_CHANGE', '{"status":"requested"}', $2)
	`, operationID, afterJSON)
	return err
}

func (r *Repository) RejectOperation(ctx context.Context, operationID, reason, rejectedBy string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE operations SET status = 'cancelled', completed_at = NOW()
		WHERE  operation_id = $1 AND status = 'requested'
	`, operationID)
	if err != nil {
		return fmt.Errorf("reject operation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("operation %s not found or not in requested state", operationID)
	}
	afterJSON, _ := json.Marshal(map[string]string{
		"status":      "cancelled",
		"reason":      reason,
		"rejected_by": rejectedBy,
	})
	_, err = r.pool.Exec(ctx, `
		INSERT INTO audit_trail (entity_type, entity_id, change_type, before, after)
		VALUES ('operation', $1, 'STATUS_CHANGE', '{"status":"requested"}', $2)
	`, operationID, afterJSON)
	return err
}

func (r *Repository) GetAuditTrail(ctx context.Context, limit int) ([]AuditEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, entity_type, entity_id, change_type, after, timestamp
		FROM   audit_trail
		ORDER  BY timestamp DESC
		LIMIT  $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("get audit trail: %w", err)
	}
	defer rows.Close()
	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var afterRaw []byte
		if err := rows.Scan(&e.ID, &e.EntityType, &e.EntityID, &e.ChangeType, &afterRaw, &e.Timestamp); err != nil {
			return nil, err
		}
		if afterRaw != nil {
			_ = json.Unmarshal(afterRaw, &e.After)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ── Shared helpers ────────────────────────────────────────────────────────────

func scanOperations(rows pgx.Rows) ([]Operation, error) {
	defer rows.Close()
	var out []Operation
	for rows.Next() {
		var o Operation
		var payloadRaw []byte
		if err := rows.Scan(&o.OperationID, &o.Type, &o.Status, &o.RequestedBy,
			&payloadRaw, &o.CreatedAt, &o.CompletedAt); err != nil {
			return nil, err
		}
		if payloadRaw != nil {
			_ = json.Unmarshal(payloadRaw, &o.RequestPayload)
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
