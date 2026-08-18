package api

import "net/http"

// handleLenderSummary returns fund-wide KPIs for the lender dashboard.
func (s *Server) handleLenderSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.repo.GetLenderSummary(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to load lender summary")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

// handleLenderExposure returns paginated investor exposure rows.
func (s *Server) handleLenderExposure(w http.ResponseWriter, r *http.Request) {
	limit, offset := pagination(r)
	rows, err := s.repo.GetLenderExposure(r.Context(), limit, offset)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to load exposure")
		return
	}
	total, _ := s.repo.GetLenderExposureCount(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"rows":  rows,
		"total": total,
	})
}

// handleCollateral returns risk distribution buckets and collateral by tier.
func (s *Server) handleCollateral(w http.ResponseWriter, r *http.Request) {
	risk, err := s.repo.GetRiskDistribution(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to load collateral")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"risk": risk})
}

// handleLenderPending surfaces the pending-approval queue for the lender view
// (read-only — approve/reject is custodian-only).
func (s *Server) handleLenderPending(w http.ResponseWriter, r *http.Request) {
	limit, offset := pagination(r)
	ops, err := s.repo.GetPendingOperations(r.Context(), limit, offset)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to load pending operations")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"operations": ops})
}
