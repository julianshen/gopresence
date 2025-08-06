package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"gopresence/internal/models"
)

type notFoundSvc struct{ mockPresenceService }
func (n *notFoundSvc) GetPresence(ctx context.Context, userID string) (models.Presence, error) { return models.Presence{}, fmt.Errorf("%s not found in store", userID) }

func TestGetPresence_StringBasedNotFound(t *testing.T) {
	service := &notFoundSvc{mockPresenceService{}}
	h := NewPresenceHandler(service)
	r := mux.NewRouter()
	r.HandleFunc("/api/v2/presence/{user_id}", h.GetPresence).Methods("GET")
	req := httptest.NewRequest("GET", "/api/v2/presence/u1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for string-based not found, got %d", rr.Code)
	}
}
