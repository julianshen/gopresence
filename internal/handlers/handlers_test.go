package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"gopresence/internal/models"
)

// mockPresenceService implements the PresenceService interface for testing
type mockPresenceService struct {
	presences map[string]models.Presence
}

func newMockPresenceService() *mockPresenceService {
	return &mockPresenceService{
		presences: make(map[string]models.Presence),
	}
}

func (m *mockPresenceService) GetPresence(ctx context.Context, userID string) (models.Presence, error) {
	if presence, exists := m.presences[userID]; exists {
		return presence, nil
	}
	return models.Presence{}, &PresenceNotFoundError{UserID: userID}
}

func (m *mockPresenceService) SetPresence(ctx context.Context, userID string, presence models.Presence) error {
	m.presences[userID] = presence
	return nil
}

func (m *mockPresenceService) GetMultiplePresences(ctx context.Context, userIDs []string) (map[string]models.Presence, error) {
	result := make(map[string]models.Presence)
	for _, userID := range userIDs {
		if presence, exists := m.presences[userID]; exists {
			result[userID] = presence
		}
	}
	return result, nil
}

func TestGetPresenceHandler(t *testing.T) {
	service := newMockPresenceService()
	handler := NewPresenceHandler(service)

	// Add a test presence
	now := time.Now().UTC().Truncate(time.Second)
	testPresence := models.Presence{
		UserID:    "user1",
		Status:    models.StatusOnline,
		Message:   "Working",
		LastSeen:  now,
		UpdatedAt: now,
		NodeID:    "node1",
	}
	service.presences["user1"] = testPresence

	// Test successful get
	req := httptest.NewRequest("GET", "/api/v2/presence/user1", nil)
	rr := httptest.NewRecorder()
	
	router := mux.NewRouter()
	router.HandleFunc("/api/v2/presence/{user_id}", handler.GetPresence).Methods("GET")
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var response models.PresenceResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success to be true")
	}

	if len(response.Data) != 1 {
		t.Errorf("Expected 1 presence in response, got %d", len(response.Data))
	}

	if presence, exists := response.Data["user1"]; !exists {
		t.Error("Expected user1 in response data")
	} else if presence.Status != models.StatusOnline {
		t.Errorf("Expected status %s, got %s", models.StatusOnline, presence.Status)
	}
}

func TestGetPresenceHandler_NotFound(t *testing.T) {
	service := newMockPresenceService()
	handler := NewPresenceHandler(service)

	req := httptest.NewRequest("GET", "/api/v2/presence/nonexistent", nil)
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v2/presence/{user_id}", handler.GetPresence).Methods("GET")
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, rr.Code)
	}

	var response models.PresenceResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Success {
		t.Error("Expected success to be false")
	}

	if !strings.Contains(response.Error, "not found") {
		t.Errorf("Expected 'not found' in error message, got: %s", response.Error)
	}
}

func TestSetPresenceHandler(t *testing.T) {
	service := newMockPresenceService()
	handler := NewPresenceHandler(service)

	// Test successful set
	setRequest := SetPresenceRequest{
		Status:  models.StatusBusy,
		Message: "In a meeting",
		TTL:     3600,
	}

	body, _ := json.Marshal(setRequest)
	req := httptest.NewRequest("PUT", "/api/v2/presence/user1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v2/presence/{user_id}", handler.SetPresence).Methods("PUT")
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var response models.PresenceResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success to be true")
	}

	// Verify the presence was actually set
	if presence, exists := service.presences["user1"]; !exists {
		t.Error("Expected presence to be set in service")
	} else if presence.Status != models.StatusBusy {
		t.Errorf("Expected status %s, got %s", models.StatusBusy, presence.Status)
	}
}

func TestSetPresenceHandler_InvalidJSON(t *testing.T) {
	service := newMockPresenceService()
	handler := NewPresenceHandler(service)

	req := httptest.NewRequest("PUT", "/api/v2/presence/user1", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v2/presence/{user_id}", handler.SetPresence).Methods("PUT")
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestGetMultiplePresencesHandler(t *testing.T) {
	service := newMockPresenceService()
	handler := NewPresenceHandler(service)

	// Add test presences
	now := time.Now().UTC().Truncate(time.Second)
	for i, userID := range []string{"user1", "user2", "user3"} {
		presence := models.Presence{
			UserID:    userID,
			Status:    models.StatusOnline,
			Message:   "Working",
			LastSeen:  now,
			UpdatedAt: now,
			NodeID:    "node1",
		}
		if i == 1 {
			presence.Status = models.StatusAway
		}
		service.presences[userID] = presence
	}

	// Test query parameter format
	req := httptest.NewRequest("GET", "/api/v2/presence?users=user1,user2,user4", nil)
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v2/presence", handler.GetMultiplePresences).Methods("GET")
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var response models.PresenceResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success to be true")
	}

	// Should return user1 and user2, but not user4 (doesn't exist)
	if len(response.Data) != 2 {
		t.Errorf("Expected 2 presences in response, got %d", len(response.Data))
	}

	if _, exists := response.Data["user1"]; !exists {
		t.Error("Expected user1 in response")
	}
	if _, exists := response.Data["user2"]; !exists {
		t.Error("Expected user2 in response")
	}
	if _, exists := response.Data["user4"]; exists {
		t.Error("Did not expect user4 in response")
	}
}

func TestBatchPresenceHandler(t *testing.T) {
	service := newMockPresenceService()
	handler := NewPresenceHandler(service)

	// Add test presences
	now := time.Now().UTC().Truncate(time.Second)
	for _, userID := range []string{"user1", "user2", "user3"} {
		service.presences[userID] = models.Presence{
			UserID:    userID,
			Status:    models.StatusOnline,
			LastSeen:  now,
			UpdatedAt: now,
			NodeID:    "node1",
		}
	}

	// Test batch request
	batchRequest := BatchPresenceRequest{
		UserIDs: []string{"user1", "user2", "user4"},
	}

	body, _ := json.Marshal(batchRequest)
	req := httptest.NewRequest("POST", "/api/v2/presence/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v2/presence/batch", handler.BatchPresence).Methods("POST")
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var response models.PresenceResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success to be true")
	}

	// Should return user1 and user2, but not user4
	if len(response.Data) != 2 {
		t.Errorf("Expected 2 presences in response, got %d", len(response.Data))
	}
}