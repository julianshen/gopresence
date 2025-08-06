package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"

	"gopresence/internal/auth"
	"gopresence/internal/cache"
	"gopresence/internal/handlers"
	"gopresence/internal/models"
	"gopresence/internal/nats"
	"gopresence/internal/service"
)

const (
	testJWTSecret = "integration-test-secret"
	testIssuer    = "presence-service"
)

// IntegrationTestSuite holds the components for integration testing
type IntegrationTestSuite struct {
	server     *httptest.Server
	service    *service.PresenceService
	middleware *auth.JWTMiddleware
}

func setupIntegrationTest(t *testing.T) *IntegrationTestSuite {
	// Create components
	memCache := cache.NewMemoryCache(1000, time.Minute)

	// Use unique bucket name for each test
	bucketName := fmt.Sprintf("test-%d", time.Now().UnixNano())
	store, err := nats.NewKVStore(nats.KVConfig{
		Embedded:   true,
		BucketName: bucketName,
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	presenceService := service.NewPresenceService(memCache, store, "test-integration-node")
	presenceHandler := handlers.NewPresenceHandler(presenceService)
	jwtMiddleware := auth.NewJWTMiddleware(testJWTSecret, testIssuer)

	// Setup routes (same as main.go)
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v2").Subrouter()

	// Presence endpoints
	api.HandleFunc("/presence/{user_id}", presenceHandler.GetPresence).Methods("GET")
	api.HandleFunc("/presence", presenceHandler.GetMultiplePresences).Methods("GET")
	api.HandleFunc("/presence/batch", presenceHandler.BatchPresence).Methods("POST")
	api.Handle("/presence/{user_id}", jwtMiddleware.Authenticate(http.HandlerFunc(presenceHandler.SetPresence))).Methods("PUT")

	// Health endpoints
	api.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"healthy"}`)
	}).Methods("GET")

	server := httptest.NewServer(router)

	return &IntegrationTestSuite{
		server:     server,
		service:    presenceService,
		middleware: jwtMiddleware,
	}
}

func (suite *IntegrationTestSuite) cleanup() {
	suite.server.Close()
	suite.service.Close()
}

func (suite *IntegrationTestSuite) createValidJWT(userID string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
		"iss": testIssuer,
	})
	tokenString, _ := token.SignedString([]byte(testJWTSecret))
	return tokenString
}

func TestIntegration_HealthEndpoint(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	resp, err := http.Get(suite.server.URL + "/api/v2/health")
	if err != nil {
		t.Fatalf("Failed to call health endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}
}

func TestIntegration_SetAndGetPresence(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	userID := "integration-user-1"
	token := suite.createValidJWT(userID)

	// 1. Try to get non-existent presence
	resp, err := http.Get(suite.server.URL + "/api/v2/presence/" + userID)
	if err != nil {
		t.Fatalf("Failed to get presence: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status %d for non-existent user, got %d", http.StatusNotFound, resp.StatusCode)
	}

	// 2. Set presence
	setRequest := handlers.SetPresenceRequest{
		Status:  models.StatusBusy,
		Message: "Integration testing",
		TTL:     3600,
	}

	setBody, _ := json.Marshal(setRequest)
	req, _ := http.NewRequest("PUT", suite.server.URL+"/api/v2/presence/"+userID, bytes.NewReader(setBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed to set presence: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d for set presence, got %d", http.StatusOK, resp.StatusCode)
	}

	// 3. Get the presence
	resp, err = http.Get(suite.server.URL + "/api/v2/presence/" + userID)
	if err != nil {
		t.Fatalf("Failed to get presence: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d for get presence, got %d", http.StatusOK, resp.StatusCode)
	}

	var presenceResponse models.PresenceResponse
	if err := json.NewDecoder(resp.Body).Decode(&presenceResponse); err != nil {
		t.Fatalf("Failed to decode presence response: %v", err)
	}

	if !presenceResponse.Success {
		t.Error("Expected successful response")
	}

	if len(presenceResponse.Data) != 1 {
		t.Errorf("Expected 1 presence in response, got %d", len(presenceResponse.Data))
	}

	presence, exists := presenceResponse.Data[userID]
	if !exists {
		t.Error("Expected user presence in response")
	}

	if presence.Status != models.StatusBusy {
		t.Errorf("Expected status %s, got %s", models.StatusBusy, presence.Status)
	}

	if presence.Message != "Integration testing" {
		t.Errorf("Expected message 'Integration testing', got '%s'", presence.Message)
	}
}

func TestIntegration_AuthenticationRequired(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	userID := "auth-test-user"

	// Try to set presence without authentication
	setRequest := handlers.SetPresenceRequest{
		Status: models.StatusOnline,
	}

	setBody, _ := json.Marshal(setRequest)
	req, _ := http.NewRequest("PUT", suite.server.URL+"/api/v2/presence/"+userID, bytes.NewReader(setBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status %d for unauthorized request, got %d", http.StatusUnauthorized, resp.StatusCode)
	}
}

func TestIntegration_MultiplePresences(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	// Set up multiple users
	users := []string{"multi-user-1", "multi-user-2", "multi-user-3"}
	statuses := []models.PresenceStatus{models.StatusOnline, models.StatusAway, models.StatusBusy}

	for i, userID := range users {
		token := suite.createValidJWT(userID)

		setRequest := handlers.SetPresenceRequest{
			Status:  statuses[i],
			Message: fmt.Sprintf("User %d message", i+1),
		}

		setBody, _ := json.Marshal(setRequest)
		req, _ := http.NewRequest("PUT", suite.server.URL+"/api/v2/presence/"+userID, bytes.NewReader(setBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to set presence for %s: %v", userID, err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d for %s, got %d", http.StatusOK, userID, resp.StatusCode)
		}
	}

	// Test query parameter format
	resp, err := http.Get(suite.server.URL + "/api/v2/presence?users=multi-user-1,multi-user-2,multi-user-4")
	if err != nil {
		t.Fatalf("Failed to get multiple presences: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var presenceResponse models.PresenceResponse
	if err := json.NewDecoder(resp.Body).Decode(&presenceResponse); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !presenceResponse.Success {
		t.Error("Expected successful response")
	}

	// Should get user-1 and user-2, but not user-4
	if len(presenceResponse.Data) != 2 {
		t.Errorf("Expected 2 presences, got %d", len(presenceResponse.Data))
	}

	if _, exists := presenceResponse.Data["multi-user-1"]; !exists {
		t.Error("Expected multi-user-1 in response")
	}
	if _, exists := presenceResponse.Data["multi-user-2"]; !exists {
		t.Error("Expected multi-user-2 in response")
	}
	if _, exists := presenceResponse.Data["multi-user-4"]; exists {
		t.Error("Did not expect multi-user-4 in response")
	}
}

func TestIntegration_BatchPresence(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	// Set up test users
	users := []string{"batch-user-1", "batch-user-2", "batch-user-3"}

	for _, userID := range users {
		token := suite.createValidJWT(userID)

		setRequest := handlers.SetPresenceRequest{
			Status:  models.StatusOnline,
			Message: "Batch test",
		}

		setBody, _ := json.Marshal(setRequest)
		req, _ := http.NewRequest("PUT", suite.server.URL+"/api/v2/presence/"+userID, bytes.NewReader(setBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to set presence for %s: %v", userID, err)
		}
		resp.Body.Close()
	}

	// Test batch request
	batchRequest := handlers.BatchPresenceRequest{
		UserIDs: []string{"batch-user-1", "batch-user-2", "batch-user-4"}, // user-4 doesn't exist
	}

	batchBody, _ := json.Marshal(batchRequest)
	resp, err := http.Post(suite.server.URL+"/api/v2/presence/batch", "application/json", bytes.NewReader(batchBody))
	if err != nil {
		t.Fatalf("Failed to make batch request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var presenceResponse models.PresenceResponse
	if err := json.NewDecoder(resp.Body).Decode(&presenceResponse); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !presenceResponse.Success {
		t.Error("Expected successful response")
	}

	// Should get user-1 and user-2, but not user-4
	if len(presenceResponse.Data) != 2 {
		t.Errorf("Expected 2 presences, got %d", len(presenceResponse.Data))
	}
}

func TestIntegration_CacheAndStoreConsistency(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	userID := "cache-test-user"
	token := suite.createValidJWT(userID)

	// Set presence
	setRequest := handlers.SetPresenceRequest{
		Status:  models.StatusAway,
		Message: "Cache consistency test",
	}

	setBody, _ := json.Marshal(setRequest)
	req, _ := http.NewRequest("PUT", suite.server.URL+"/api/v2/presence/"+userID, bytes.NewReader(setBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to set presence: %v", err)
	}
	resp.Body.Close()

	// Get presence (should come from cache)
	resp1, err := http.Get(suite.server.URL + "/api/v2/presence/" + userID)
	if err != nil {
		t.Fatalf("Failed to get presence: %v", err)
	}
	defer resp1.Body.Close()

	var response1 models.PresenceResponse
	if err := json.NewDecoder(resp1.Body).Decode(&response1); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Simulate cache clear by accessing service directly
	// In a real test, we might restart the service or wait for TTL
	ctx := context.Background()
	directPresence, err := suite.service.GetPresence(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to get presence from service: %v", err)
	}

	// Compare HTTP response with direct service call
	httpPresence := response1.Data[userID]
	if httpPresence.Status != directPresence.Status {
		t.Errorf("Status mismatch: HTTP=%s, Direct=%s", httpPresence.Status, directPresence.Status)
	}
	if httpPresence.Message != directPresence.Message {
		t.Errorf("Message mismatch: HTTP=%s, Direct=%s", httpPresence.Message, directPresence.Message)
	}
}
