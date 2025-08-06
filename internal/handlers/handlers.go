package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"gopresence/internal/models"
)

// PresenceService defines the interface for presence operations
type PresenceService interface {
	GetPresence(ctx context.Context, userID string) (models.Presence, error)
	SetPresence(ctx context.Context, userID string, presence models.Presence) error
	GetMultiplePresences(ctx context.Context, userIDs []string) (map[string]models.Presence, error)
}

// PresenceNotFoundError represents an error when a presence is not found
type PresenceNotFoundError struct {
	UserID string
}

func (e *PresenceNotFoundError) Error() string {
	return fmt.Sprintf("presence not found for user %s", e.UserID)
}

// SetPresenceRequest represents the request body for setting presence
type SetPresenceRequest struct {
	Status  models.PresenceStatus `json:"status"`
	Message string                `json:"message,omitempty"`
	TTL     int64                 `json:"ttl,omitempty"`
}

// BatchPresenceRequest represents the request body for batch presence queries
type BatchPresenceRequest struct {
	UserIDs []string `json:"user_ids"`
}

// PresenceHandler handles HTTP requests for presence operations
type PresenceHandler struct {
	service PresenceService
}

// NewPresenceHandler creates a new PresenceHandler
func NewPresenceHandler(service PresenceService) *PresenceHandler {
	return &PresenceHandler{
		service: service,
	}
}

// GetPresence handles GET /api/v2/presence/{user_id}
func (h *PresenceHandler) GetPresence(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["user_id"]

	if userID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "user_id is required")
		return
	}

	presence, err := h.service.GetPresence(r.Context(), userID)
	if err != nil {
		// Check for PresenceNotFoundError from different packages
		if _, ok := err.(*PresenceNotFoundError); ok {
			h.writeErrorResponse(w, http.StatusNotFound, err.Error())
			return
		}
		// Also check by error message content
		if strings.Contains(err.Error(), "not found") {
			h.writeErrorResponse(w, http.StatusNotFound, err.Error())
			return
		}
		h.writeErrorResponse(w, http.StatusInternalServerError, "failed to get presence")
		return
	}

	response := models.PresenceResponse{
		Success: true,
		Data: map[string]models.Presence{
			userID: presence,
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// SetPresence handles PUT /api/v2/presence/{user_id}
func (h *PresenceHandler) SetPresence(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["user_id"]

	if userID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "user_id is required")
		return
	}

	var req SetPresenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Validate status
	if !req.Status.IsValid() {
		h.writeErrorResponse(w, http.StatusBadRequest, "invalid status")
		return
	}

	// Create presence object
	now := time.Now().UTC()
	presence := models.Presence{
		UserID:    userID,
		Status:    req.Status,
		Message:   req.Message,
		LastSeen:  now,
		UpdatedAt: now,
		NodeID:    "current-node", // This would be set from config in real implementation
	}

	if req.TTL > 0 {
		presence.TTL = time.Duration(req.TTL) * time.Second
	}

	if err := h.service.SetPresence(r.Context(), userID, presence); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "failed to set presence")
		return
	}

	response := models.PresenceResponse{
		Success: true,
		Data: map[string]models.Presence{
			userID: presence,
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// GetMultiplePresences handles GET /api/v2/presence?users=user1,user2,user3
func (h *PresenceHandler) GetMultiplePresences(w http.ResponseWriter, r *http.Request) {
	usersParam := r.URL.Query().Get("users")
	if usersParam == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "users parameter is required")
		return
	}

	userIDs := strings.Split(usersParam, ",")
	for i, userID := range userIDs {
		userIDs[i] = strings.TrimSpace(userID)
	}

	presences, err := h.service.GetMultiplePresences(r.Context(), userIDs)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "failed to get presences")
		return
	}

	response := models.PresenceResponse{
		Success: true,
		Data:    presences,
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// BatchPresence handles POST /api/v2/presence/batch
func (h *PresenceHandler) BatchPresence(w http.ResponseWriter, r *http.Request) {
	var req BatchPresenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if len(req.UserIDs) == 0 {
		h.writeErrorResponse(w, http.StatusBadRequest, "user_ids is required")
		return
	}

	presences, err := h.service.GetMultiplePresences(r.Context(), req.UserIDs)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "failed to get presences")
		return
	}

	response := models.PresenceResponse{
		Success: true,
		Data:    presences,
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// writeJSONResponse writes a JSON response
func (h *PresenceHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// writeErrorResponse writes an error response
func (h *PresenceHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	response := models.PresenceResponse{
		Success: false,
		Error:   message,
	}
	h.writeJSONResponse(w, statusCode, response)
}