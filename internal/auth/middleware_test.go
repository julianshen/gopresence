package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-key"

func TestJWTMiddleware_ValidToken(t *testing.T) {
	// Create a valid token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user1",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
		"iss": "presence-service",
	})
	tokenString, _ := token.SignedString([]byte(testSecret))

	middleware := NewJWTMiddleware(testSecret, "presence-service")

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if user ID is in context
		userID := GetUserIDFromContext(r.Context())
		if userID != "user1" {
			t.Errorf("Expected user ID 'user1', got '%s'", userID)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Wrap handler with middleware
	handler := middleware.Authenticate(testHandler)

	// Create request with valid token
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if rr.Body.String() != "success" {
		t.Errorf("Expected 'success', got '%s'", rr.Body.String())
	}
}

func TestJWTMiddleware_NoToken(t *testing.T) {
	middleware := NewJWTMiddleware(testSecret, "presence-service")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Authenticate(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestJWTMiddleware_InvalidToken(t *testing.T) {
	middleware := NewJWTMiddleware(testSecret, "presence-service")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Authenticate(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestJWTMiddleware_ExpiredToken(t *testing.T) {
	// Create an expired token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user1",
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
		"exp": time.Now().Add(-time.Hour).Unix(), // Expired 1 hour ago
		"iss": "presence-service",
	})
	tokenString, _ := token.SignedString([]byte(testSecret))

	middleware := NewJWTMiddleware(testSecret, "presence-service")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Authenticate(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestJWTMiddleware_WrongIssuer(t *testing.T) {
	// Create a token with wrong issuer
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user1",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
		"iss": "wrong-issuer",
	})
	tokenString, _ := token.SignedString([]byte(testSecret))

	middleware := NewJWTMiddleware(testSecret, "presence-service")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Authenticate(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestJWTMiddleware_MissingSubject(t *testing.T) {
	// Create a token without subject
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
		"iss": "presence-service",
	})
	tokenString, _ := token.SignedString([]byte(testSecret))

	middleware := NewJWTMiddleware(testSecret, "presence-service")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Authenticate(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestJWTMiddleware_InvalidAuthHeader(t *testing.T) {
	middleware := NewJWTMiddleware(testSecret, "presence-service")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Authenticate(testHandler)

	// Test various invalid authorization headers
	testCases := []string{
		"InvalidFormat",
		"Basic token123",
		"Bearer",
		"Bearer ",
	}

	for _, authHeader := range testCases {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", authHeader)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d for auth header '%s', got %d", http.StatusUnauthorized, authHeader, rr.Code)
		}
	}
}

func TestGetUserIDFromContext(t *testing.T) {
	ctx := context.Background()

	// Test with no user ID in context
	userID := GetUserIDFromContext(ctx)
	if userID != "" {
		t.Errorf("Expected empty string, got '%s'", userID)
	}

	// Test with user ID in context
	ctx = SetUserIDInContext(ctx, "user123")
	userID = GetUserIDFromContext(ctx)
	if userID != "user123" {
		t.Errorf("Expected 'user123', got '%s'", userID)
	}
}

func TestOptionalJWTMiddleware(t *testing.T) {
	middleware := NewJWTMiddleware(testSecret, "presence-service")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserIDFromContext(r.Context())
		if userID == "" {
			w.Write([]byte("anonymous"))
		} else {
			w.Write([]byte("authenticated:" + userID))
		}
	})

	handler := middleware.OptionalAuthenticate(testHandler)

	// Test without token (should allow through)
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if rr.Body.String() != "anonymous" {
		t.Errorf("Expected 'anonymous', got '%s'", rr.Body.String())
	}

	// Test with valid token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user1",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
		"iss": "presence-service",
	})
	tokenString, _ := token.SignedString([]byte(testSecret))

	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if rr.Body.String() != "authenticated:user1" {
		t.Errorf("Expected 'authenticated:user1', got '%s'", rr.Body.String())
	}
}
