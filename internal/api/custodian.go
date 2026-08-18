package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// handleCustodianQueue returns the paginated pending-approval queue.
func (s *Server) handleCustodianQueue(w http.ResponseWriter, r *http.Request) {
	limit, offset := pagination(r)
	ops, err := s.repo.GetPendingOperations(r.Context(), limit, offset)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to load queue")
		return
	}
	total, _ := s.repo.CountPendingOperations(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"operations": ops,
		"total":      total,
	})
}

// handleApprove marks a pending operation as accepted and writes an audit entry.
func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	sess := sessionFromCtx(r.Context())
	opID := chi.URLParam(r, "id")
	if err := s.repo.ApproveOperation(r.Context(), opID, sess.Email); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

// handleReject marks a pending operation as cancelled and writes an audit entry.
func (s *Server) handleReject(w http.ResponseWriter, r *http.Request) {
	sess := sessionFromCtx(r.Context())
	opID := chi.URLParam(r, "id")

	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Reason == "" {
		writeErr(w, http.StatusBadRequest, "reason is required")
		return
	}
	if err := s.repo.RejectOperation(r.Context(), opID, body.Reason, sess.Email); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// handleAuditLog returns the most recent audit trail entries.
func (s *Server) handleAuditLog(w http.ResponseWriter, r *http.Request) {
	entries, err := s.repo.GetAuditTrail(r.Context(), 50)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to load audit log")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
}

// handleStats returns custodian queue statistics.
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.repo.GetOperationStats(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to load stats")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// handleExport streams a CSV of all operations for regulatory filing.
func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	ops, err := s.repo.GetPendingOperations(r.Context(), 1000, 0)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "export failed")
		return
	}
	filename := fmt.Sprintf("dumastas-operations-%s.csv", time.Now().UTC().Format("20060102-150405"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"operation_id", "type", "status", "requested_by", "institution", "token_id", "amount", "usd_value", "created_at"})
	for _, op := range ops {
		institution, _ := op.RequestPayload["institution"].(string)
		tokenID, _ := op.RequestPayload["token_id"].(string)
		amount := fmt.Sprintf("%v", op.RequestPayload["amount"])
		usdVal := fmt.Sprintf("%v", op.RequestPayload["usd_value"])
		_ = cw.Write([]string{
			op.OperationID, op.Type, op.Status, op.RequestedBy,
			institution, tokenID, amount, usdVal,
			op.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	cw.Flush()
}
