package api

import (
	"context"
	"fmt"
	"net/http"
)

type contextKey string

const sessionCtxKey contextKey = "session"

type SessionData struct {
	SessionID   string
	UserID      string
	Email       string
	Role        string
	Institution string
}

func sessionFromCtx(ctx context.Context) SessionData {
	v, _ := ctx.Value(sessionCtxKey).(SessionData)
	return v
}

// requireAuth decodes the session cookie and stores the SessionData in context.
// Redirects to /login on any failure.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, err := s.decodeSession(r)
		if err != nil {
			s.clearSession(w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		ctx := context.WithValue(r.Context(), sessionCtxKey, sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireRole is a chi middleware that enforces a specific role; must be placed
// after requireAuth (or inside a route group that already has requireAuth).
func (s *Server) requireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return s.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess := sessionFromCtx(r.Context())
			if sess.Role != role {
				writeErr(w, http.StatusForbidden, "insufficient role")
				return
			}
			next.ServeHTTP(w, r)
		}))
	}
}

// decodeSession reads and decrypts the session cookie.
func (s *Server) decodeSession(r *http.Request) (SessionData, error) {
	cookie, err := r.Cookie("raxon_session")
	if err != nil {
		return SessionData{}, err
	}
	var sess SessionData
	if err := s.sc.Decode("raxon_session", cookie.Value, &sess); err != nil {
		return SessionData{}, err
	}
	if sess.SessionID == "" {
		return SessionData{}, fmt.Errorf("session id is missing")
	}
	record, err := s.repo.GetActiveSession(r.Context(), sess.SessionID)
	if err != nil {
		return SessionData{}, err
	}
	if record.UserID != sess.UserID || record.Role != sess.Role {
		return SessionData{}, fmt.Errorf("session claims do not match database")
	}
	sess.Email = record.Email
	sess.Institution = record.InstitutionName
	return sess, nil
}

// setSession encrypts sess and writes it as a cookie.
func (s *Server) setSession(w http.ResponseWriter, sess SessionData) error {
	encoded, err := s.sc.Encode("raxon_session", sess)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "raxon_session",
		Value:    encoded,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		// Secure: true in production (TLS required)
	})
	return nil
}

// clearSession removes the session cookie.
func (s *Server) clearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "raxon_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}
