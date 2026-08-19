package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type SessionRecord struct {
	ID              string
	UserID          string
	Email           string
	Role            string
	InstitutionName string
	ExpiresAt       time.Time
}

func (r *Repository) GetDevelopmentUser(ctx context.Context, role string) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, role, oidc_subject, institution_name, created_at
		FROM users
		WHERE role = $1 AND oidc_subject = 'demo|' || role
	`, role).Scan(&u.ID, &u.Email, &u.Role, &u.OIDCSubject, &u.InstitutionName, &u.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get development user: %w", err)
	}
	return &u, nil
}

func (r *Repository) CreateSession(ctx context.Context, userID, authMethod, ipAddress, userAgent string, ttl time.Duration) (SessionRecord, error) {
	var s SessionRecord
	err := r.pool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, auth_method, expires_at, last_seen_at, ip_address, user_agent)
		SELECT id, $2, NOW() + ($5 * INTERVAL '1 second'), NOW(), $3, $4
		FROM users WHERE id = $1
		RETURNING id::text, user_id::text, expires_at
	`, userID, authMethod, ipAddress, userAgent, int64(ttl/time.Second)).Scan(&s.ID, &s.UserID, &s.ExpiresAt)
	if err != nil {
		return SessionRecord{}, fmt.Errorf("create session: %w", err)
	}
	return s, nil
}

func (r *Repository) GetActiveSession(ctx context.Context, sessionID string) (SessionRecord, error) {
	var s SessionRecord
	err := r.pool.QueryRow(ctx, `
		SELECT s.id::text, s.user_id::text, u.email, u.role, u.institution_name, s.expires_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.id = $1::uuid
		  AND s.revoked = FALSE
		  AND s.revoked_at IS NULL
		  AND s.expires_at > NOW()
	`, sessionID).Scan(&s.ID, &s.UserID, &s.Email, &s.Role, &s.InstitutionName, &s.ExpiresAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return SessionRecord{}, fmt.Errorf("session is not active")
		}
		return SessionRecord{}, fmt.Errorf("get active session: %w", err)
	}
	_, err = r.pool.Exec(ctx, `UPDATE sessions SET last_seen_at = NOW() WHERE id = $1::uuid`, sessionID)
	if err != nil {
		return SessionRecord{}, fmt.Errorf("touch session: %w", err)
	}
	return s, nil
}

func (r *Repository) RevokeSession(ctx context.Context, sessionID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE sessions SET revoked = TRUE, revoked_at = NOW()
		WHERE id = $1::uuid
	`, sessionID)
	return err
}

func (r *Repository) RecordLoginAudit(ctx context.Context, userID, email, role, authMethod, outcome, reason, ipAddress, userAgent string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO login_audit
			(user_id, attempted_email, role, auth_method, outcome, failure_reason, ip_address, user_agent)
		VALUES (NULLIF($1, '')::uuid, $2, NULLIF($3, ''), $4, $5, NULLIF($6, ''), $7, $8)
	`, userID, email, role, authMethod, outcome, reason, ipAddress, userAgent)
	return err
}
