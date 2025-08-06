package handlers

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	"gopresence/internal/models"
)

type errSvc struct{}

func (e *errSvc) GetPresence(ctx context.Context, userID string) (models.Presence, error) {
	return models.Presence{}, errors.New("db failed")
}
func (e *errSvc) SetPresence(ctx context.Context, userID string, presence models.Presence) error {
	return errors.New("db failed")
}
func (e *errSvc) GetMultiplePresences(ctx context.Context, userIDs []string) (map[string]models.Presence, error) {
	return nil, errors.New("db failed")
}

func TestHandlers_ErrorBranches(t *testing.T) {
	h := NewPresenceHandler(&errSvc{})
	r := mux.NewRouter()
	r.HandleFunc("/api/v2/presence/{user_id}", h.GetPresence).Methods("GET")
	r.HandleFunc("/api/v2/presence/{user_id}", h.SetPresence).Methods("PUT")
	r.HandleFunc("/api/v2/presence", h.GetMultiplePresences).Methods("GET")
	r.HandleFunc("/api/v2/presence/batch", h.BatchPresence).Methods("POST")

	// GetPresence generic error should be 500
	req := httptest.NewRequest("GET", "/api/v2/presence/u1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}

	// SetPresence invalid status
	body := bytes.NewBufferString(`{"status":"bad"}`)
	req = httptest.NewRequest("PUT", "/api/v2/presence/u1", body)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid status, got %d", rr.Code)
	}

	// SetPresence service error
	body = bytes.NewBufferString(`{"status":"online"}`)
	req = httptest.NewRequest("PUT", "/api/v2/presence/u1", body)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on service error, got %d", rr.Code)
	}

	// GetMultiple missing param
	req = httptest.NewRequest("GET", "/api/v2/presence", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing users param, got %d", rr.Code)
	}

	// Batch invalid JSON
	req = httptest.NewRequest("POST", "/api/v2/presence/batch", strings.NewReader("{"))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", rr.Code)
	}

	// Batch empty user_ids
	req = httptest.NewRequest("POST", "/api/v2/presence/batch", strings.NewReader(`{"user_ids":[]}`))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty user_ids, got %d", rr.Code)
	}
}
