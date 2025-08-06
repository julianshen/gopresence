package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey is used for storing values in context
type contextKey string

const userIDContextKey contextKey = "user_id"

// JWTMiddleware handles JWT authentication
type JWTMiddleware struct {
	secretKey string
	issuer    string
}

// NewJWTMiddleware creates a new JWT middleware
func NewJWTMiddleware(secretKey, issuer string) *JWTMiddleware {
	return &JWTMiddleware{
		secretKey: secretKey,
		issuer:    issuer,
	}
}

// Authenticate is a middleware that requires valid JWT authentication
func (m *JWTMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := m.validateToken(r)
		if err != nil {
			m.writeUnauthorizedResponse(w, err.Error())
			return
		}

		// Extract user ID from token
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			m.writeUnauthorizedResponse(w, "invalid token claims")
			return
		}

		userID, ok := claims["sub"].(string)
		if !ok || userID == "" {
			m.writeUnauthorizedResponse(w, "missing or invalid user ID in token")
			return
		}

		// Add user ID to context
		ctx := SetUserIDInContext(r.Context(), userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuthenticate is a middleware that allows both authenticated and unauthenticated requests
func (m *JWTMiddleware) OptionalAuthenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := m.validateToken(r)
		if err != nil {
			// If there's no token or it's invalid, continue without authentication
			next.ServeHTTP(w, r)
			return
		}

		// Extract user ID from token if valid
		claims, ok := token.Claims.(jwt.MapClaims)
		if ok {
			if userID, ok := claims["sub"].(string); ok && userID != "" {
				ctx := SetUserIDInContext(r.Context(), userID)
				r = r.WithContext(ctx)
			}
		}

		next.ServeHTTP(w, r)
	})
}

// validateToken extracts and validates the JWT token from the request
func (m *JWTMiddleware) validateToken(r *http.Request) (*jwt.Token, error) {
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("missing authorization header")
	}

	// Check Bearer format
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, fmt.Errorf("invalid authorization header format")
	}

	tokenString := parts[1]
	if tokenString == "" {
		return nil, fmt.Errorf("missing token")
	}

	// Parse and validate token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Validate issuer if specified
	if m.issuer != "" {
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return nil, fmt.Errorf("invalid token claims")
		}

		issuer, ok := claims["iss"].(string)
		if !ok || issuer != m.issuer {
			return nil, fmt.Errorf("invalid token issuer")
		}
	}

	return token, nil
}

// writeUnauthorizedResponse writes an unauthorized error response
func (m *JWTMiddleware) writeUnauthorizedResponse(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	
	response := map[string]interface{}{
		"success": false,
		"error":   message,
	}
	
	json.NewEncoder(w).Encode(response)
}

// SetUserIDInContext adds a user ID to the context
func SetUserIDInContext(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDContextKey, userID)
}

// GetUserIDFromContext retrieves the user ID from the context
func GetUserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value(userIDContextKey).(string); ok {
		return userID
	}
	return ""
}