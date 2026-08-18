package api

import (
	"encoding/json"
	"net/http"
)

// handleInvestorSummary returns portfolio totals for the logged-in investor.
func (s *Server) handleInvestorSummary(w http.ResponseWriter, r *http.Request) {
	sess := sessionFromCtx(r.Context())
	summary, err := s.repo.GetInvestorSummary(r.Context(), sess.UserID, s.cfg.ChannelName)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to load summary")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

// handleInvestorOperations returns paginated operations for the logged-in investor.
func (s *Server) handleInvestorOperations(w http.ResponseWriter, r *http.Request) {
	sess := sessionFromCtx(r.Context())
	limit, offset := pagination(r)
	ops, err := s.repo.GetOperationsByOwner(r.Context(), sess.UserID, limit, offset)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to load operations")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"operations": ops})
}

// handleNewOperation creates a new ENCUMBRANCE/UNENCUMBRANCE request.
func (s *Server) handleNewOperation(w http.ResponseWriter, r *http.Request) {
	sess := sessionFromCtx(r.Context())

	var body struct {
		Type     string  `json:"type"`
		TokenID  string  `json:"token_id"`
		Amount   float64 `json:"amount"`
		USDValue float64 `json:"usd_value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Type == "" || body.Amount <= 0 {
		writeErr(w, http.StatusBadRequest, "type and amount are required")
		return
	}
	if body.TokenID == "" {
		body.TokenID = "DUM-MMF-001"
	}

	payload := map[string]any{
		"owner_id":    sess.UserID,
		"institution": sess.Institution,
		"token_id":    body.TokenID,
		"amount":      body.Amount,
		"usd_value":   body.USDValue,
	}
	env := buildEventEnvelope(body.Type, payload, sess.UserID)
	if err := s.repo.SaveEvent(r.Context(), env); err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to save event")
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
