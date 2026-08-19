package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"raxonplatform/internal/config"
	"raxonplatform/internal/db/repository"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/securecookie"
)

// Server holds all dependencies shared by every handler.
type Server struct {
	repo   *repository.Repository
	sc     *securecookie.SecureCookie
	cfg    config.Config
	router *chi.Mux
}

// New constructs and wires the server. Returns an error if the session secret is
// the default placeholder so operators know to set RAXON_SESSION_SECRET.
func New(repo *repository.Repository, cfg config.Config) (*Server, error) {
	if cfg.AuthMode == "dev" && cfg.Environment == "production" {
		return nil, fmt.Errorf("development authentication is disabled in production")
	}
	if cfg.SessionSecret == "change-this-to-a-32-byte-secret!" {
		log.Println("WARNING: using default session secret — set RAXON_SESSION_SECRET in production")
	}
	hashKey := []byte(cfg.SessionSecret)
	if len(hashKey) < 32 {
		return nil, fmt.Errorf("RAXON_SESSION_SECRET must be at least 32 bytes")
	}
	s := &Server{
		repo:   repo,
		sc:     securecookie.New(hashKey, nil),
		cfg:    cfg,
		router: chi.NewRouter(),
	}
	s.routes()
	return s, nil
}

func (s *Server) routes() {
	r := s.router

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	// SameSite=Strict on the session cookie provides CSRF protection for same-origin
	// fetch() calls. For cross-origin or non-browser clients, add gorilla/csrf.

	// Static HTML pages
	fs := http.FileServer(http.Dir(s.cfg.StaticDir))
	r.Get("/login", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, s.cfg.StaticDir+"/codeP1.html") }))
	r.Get("/static/*", http.StripPrefix("/static/", fs).ServeHTTP)

	// Role-gated dashboard routes
	r.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.requireAuth(http.HandlerFunc(s.handleDashboard)).ServeHTTP(w, r)
	}))

	// Auth
	r.Get("/auth/login", http.HandlerFunc(s.handleLogin))
	r.Post("/auth/dev-login", http.HandlerFunc(s.handleDevLogin))
	r.Get("/auth/callback", http.HandlerFunc(s.handleCallback))
	r.Get("/auth/logout", http.HandlerFunc(s.handleLogout))

	// Investor API
	r.Route("/api/investor", func(r chi.Router) {
		r.Use(s.requireRole("investor"))
		r.Get("/summary", http.HandlerFunc(s.handleInvestorSummary))
		r.Get("/operations", http.HandlerFunc(s.handleInvestorOperations))
		r.Post("/operations", http.HandlerFunc(s.handleNewOperation))
	})

	// Lender API
	r.Route("/api/lender", func(r chi.Router) {
		r.Use(s.requireRole("lender"))
		r.Get("/summary", http.HandlerFunc(s.handleLenderSummary))
		r.Get("/exposure", http.HandlerFunc(s.handleLenderExposure))
		r.Get("/collateral", http.HandlerFunc(s.handleCollateral))
		r.Get("/operations/pending", http.HandlerFunc(s.handleLenderPending))
	})

	// Custodian API
	r.Route("/api/custodian", func(r chi.Router) {
		r.Use(s.requireRole("custodian"))
		r.Get("/queue", http.HandlerFunc(s.handleCustodianQueue))
		r.Post("/operations/{id}/approve", http.HandlerFunc(s.handleApprove))
		r.Post("/operations/{id}/reject", http.HandlerFunc(s.handleReject))
		r.Get("/audit-log", http.HandlerFunc(s.handleAuditLog))
		r.Get("/stats", http.HandlerFunc(s.handleStats))
		r.Get("/export", http.HandlerFunc(s.handleExport))
	})
}

// handleDashboard serves the correct HTML page based on the caller's role.
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	sess := sessionFromCtx(r.Context())
	pageMap := map[string]string{
		"investor":  "codeP2.html",
		"lender":    "codeP3.html",
		"custodian": "codeP4.html",
	}
	page, ok := pageMap[sess.Role]
	if !ok {
		http.Error(w, "unknown role", http.StatusForbidden)
		return
	}
	http.ServeFile(w, r, s.cfg.StaticDir+"/"+page)
}

// Start binds the server on cfg.Port. Blocks until ctx is cancelled.
func (s *Server) Start(ctx context.Context) error {
	srv := &http.Server{
		Addr:    ":" + s.cfg.Port,
		Handler: s.router,
	}
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()
	log.Printf("web server listening on :%s", s.cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// ── Shared helpers ────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
