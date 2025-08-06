package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// ReadinessChecker defines minimal readiness check for dependencies
type ReadinessChecker interface {
	Ready(ctx context.Context) error
}

type HealthHandler struct {
	checker ReadinessChecker
}

func NewHealthHandler(checker ReadinessChecker) *HealthHandler { return &HealthHandler{checker: checker} }

func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type","application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status":"ok","ts": time.Now().UTC()})
}

func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type","application/json")
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if h.checker != nil {
		if err := h.checker.Ready(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{"status":"unready","error": err.Error()})
			return
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"status":"ready","ts": time.Now().UTC()})
}
