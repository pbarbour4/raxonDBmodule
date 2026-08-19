package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"raxonplatform/internal/db/repository"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// oidcProvider is initialised lazily so the server can start without OIDC config
// (useful in local development before credentials are issued).
type oidcProvider struct {
	provider *gooidc.Provider
	oauth2   oauth2.Config
	verifier *gooidc.IDTokenVerifier
}

func (s *Server) initOIDC(ctx context.Context) (*oidcProvider, error) {
	if s.cfg.OIDCIssuerURL == "" {
		return nil, fmt.Errorf("RAXON_OIDC_ISSUER_URL is not configured")
	}
	p, err := gooidc.NewProvider(ctx, s.cfg.OIDCIssuerURL)
	if err != nil {
		return nil, fmt.Errorf("OIDC provider discovery: %w", err)
	}
	oc := oauth2.Config{
		ClientID:     s.cfg.OIDCClientID,
		ClientSecret: s.cfg.OIDCClientSecret,
		RedirectURL:  s.cfg.OIDCRedirectURL,
		Endpoint:     p.Endpoint(),
		Scopes:       []string{gooidc.ScopeOpenID, "email", "profile"},
	}
	return &oidcProvider{
		provider: p,
		oauth2:   oc,
		verifier: p.Verifier(&gooidc.Config{ClientID: s.cfg.OIDCClientID}),
	}, nil
}

// handleLogin initiates the OIDC authorization code flow.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if s.cfg.AuthMode == "dev" {
		s.handleDevLogin(w, r)
		return
	}
	// Show the login page; the Secure Authorize button posts here to kick off OIDC.
	// If OIDC is not configured, surface a clear error rather than panicking.
	prov, err := s.initOIDC(r.Context())
	if err != nil {
		log.Printf("OIDC not configured: %v", err)
		http.Redirect(w, r, "/login?error=oidc_not_configured", http.StatusSeeOther)
		return
	}

	state, err := randomState()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	// Store the state value in a short-lived cookie to validate on callback.
	http.SetCookie(w, &http.Cookie{
		Name:     "raxon_oauth_state",
		Value:    state,
		Path:     "/auth",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})
	http.Redirect(w, r, prov.oauth2.AuthCodeURL(state), http.StatusFound)
}

// handleCallback exchanges the OIDC code, verifies the ID token, upserts the
// user in postgres, writes a session cookie, then redirects to the dashboard.
func (s *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	prov, err := s.initOIDC(r.Context())
	if err != nil {
		http.Redirect(w, r, "/login?error=oidc_not_configured", http.StatusSeeOther)
		return
	}

	stateCookie, err := r.Cookie("raxon_oauth_state")
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		http.Redirect(w, r, "/login?error=invalid_state", http.StatusSeeOther)
		return
	}
	// Consume the state cookie.
	http.SetCookie(w, &http.Cookie{Name: "raxon_oauth_state", MaxAge: -1, Path: "/auth"})

	token, err := prov.oauth2.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		log.Printf("OIDC token exchange failed: %v", err)
		http.Redirect(w, r, "/login?error=token_exchange", http.StatusSeeOther)
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Redirect(w, r, "/login?error=no_id_token", http.StatusSeeOther)
		return
	}
	idToken, err := prov.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		log.Printf("OIDC ID token verification failed: %v", err)
		http.Redirect(w, r, "/login?error=token_invalid", http.StatusSeeOther)
		return
	}

	var claims struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		http.Redirect(w, r, "/login?error=claims", http.StatusSeeOther)
		return
	}

	// Look up the user by their OIDC subject; the role is set in the DB, not the token.
	user, err := s.repo.GetUserByOIDCSubject(r.Context(), idToken.Subject)
	if err != nil {
		log.Printf("DB lookup failed: %v", err)
		http.Redirect(w, r, "/login?error=db_error", http.StatusSeeOther)
		return
	}
	if user == nil {
		// No user pre-provisioned for this subject — access denied.
		http.Redirect(w, r, "/login?error=not_provisioned", http.StatusSeeOther)
		return
	}

	// Refresh email / institution name from the token.
	upserted, err := s.repo.UpsertUser(r.Context(), repository.User{
		ID:              user.ID,
		Email:           claims.Email,
		Role:            user.Role,
		OIDCSubject:     idToken.Subject,
		InstitutionName: user.InstitutionName,
	})
	if err != nil {
		log.Printf("upsert user failed: %v", err)
		http.Redirect(w, r, "/login?error=db_error", http.StatusSeeOther)
		return
	}
	session, err := s.repo.CreateSession(r.Context(), upserted.ID, "oidc", clientIP(r), r.UserAgent(), s.sessionTTL())
	if err != nil {
		log.Printf("create OIDC session failed: %v", err)
		http.Redirect(w, r, "/login?error=db_error", http.StatusSeeOther)
		return
	}

	if err := s.setSession(w, SessionData{
		SessionID:   session.ID,
		UserID:      upserted.ID,
		Email:       upserted.Email,
		Role:        upserted.Role,
		Institution: upserted.InstitutionName,
	}); err != nil {
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleLogout clears the session and redirects to the login page.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if sess, err := s.decodeSessionCookie(r); err == nil {
		_ = s.repo.RevokeSession(r.Context(), sess.SessionID)
	}
	s.clearSession(w)
	// Optionally redirect to the OIDC provider's logout endpoint if configured.
	if s.cfg.OIDCIssuerURL != "" {
		logoutURL := s.cfg.OIDCIssuerURL + "/v2/logout?returnTo=" +
			url.QueryEscape("http://localhost:"+s.cfg.Port+"/login") +
			"&client_id=" + url.QueryEscape(s.cfg.OIDCClientID)
		http.Redirect(w, r, logoutURL, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) handleDevLogin(w http.ResponseWriter, r *http.Request) {
	if s.cfg.AuthMode != "dev" || s.cfg.Environment == "production" {
		http.Error(w, "development login is disabled", http.StatusNotFound)
		return
	}
	role := r.FormValue("role")
	if role != "investor" && role != "lender" && role != "custodian" {
		_ = s.repo.RecordLoginAudit(r.Context(), "", "", "", "dev", "failure", "invalid_role", clientIP(r), r.UserAgent())
		http.Redirect(w, r, "/login?error=invalid_role", http.StatusSeeOther)
		return
	}
	user, err := s.repo.GetDevelopmentUser(r.Context(), role)
	if err != nil || user == nil {
		_ = s.repo.RecordLoginAudit(r.Context(), "", "", role, "dev", "failure", "test_user_not_found", clientIP(r), r.UserAgent())
		http.Redirect(w, r, "/login?error=test_user_not_found", http.StatusSeeOther)
		return
	}
	session, err := s.repo.CreateSession(r.Context(), user.ID, "dev", clientIP(r), r.UserAgent(), s.sessionTTL())
	if err != nil {
		_ = s.repo.RecordLoginAudit(r.Context(), user.ID, user.Email, user.Role, "dev", "failure", "session_creation_failed", clientIP(r), r.UserAgent())
		http.Redirect(w, r, "/login?error=db_error", http.StatusSeeOther)
		return
	}
	if err := s.repo.RecordLoginAudit(r.Context(), user.ID, user.Email, user.Role, "dev", "success", "", clientIP(r), r.UserAgent()); err != nil {
		http.Error(w, "audit error", http.StatusInternalServerError)
		return
	}
	if err := s.setSession(w, SessionData{SessionID: session.ID, UserID: user.ID, Email: user.Email, Role: user.Role, Institution: user.InstitutionName}); err != nil {
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) sessionTTL() time.Duration {
	ttl, err := time.ParseDuration(s.cfg.SessionTTL)
	if err != nil || ttl <= 0 {
		return 8 * time.Hour
	}
	return ttl
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func (s *Server) decodeSessionCookie(r *http.Request) (SessionData, error) {
	cookie, err := r.Cookie("raxon_session")
	if err != nil {
		return SessionData{}, err
	}
	var sess SessionData
	if err := s.sc.Decode("raxon_session", cookie.Value, &sess); err != nil {
		return SessionData{}, err
	}
	return sess, nil
}

func randomState() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
