package handlers

import (
	"net/http"
	"os"
	"strconv"
	"strings"
)

// CORSConfig holds CORS settings loaded from env
type CORSConfig struct {
	Enabled          bool
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

func loadCORSConfigFromEnv() CORSConfig {
	enabled := true
	if v := os.Getenv("CORS_ENABLED"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			enabled = b
		}
	}
	origins := strings.Split(getEnvDefault("CORS_ALLOWED_ORIGINS", "*"), ",")
	methods := strings.Split(getEnvDefault("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,OPTIONS"), ",")
	headers := strings.Split(getEnvDefault("CORS_ALLOWED_HEADERS", "Authorization,Content-Type"), ",")
	allowCreds := false
	if v := os.Getenv("CORS_ALLOW_CREDENTIALS"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			allowCreds = b
		}
	}
	maxAge := 600
	if v := os.Getenv("CORS_MAX_AGE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			maxAge = n
		}
	}
	return CORSConfig{
		Enabled:          enabled,
		AllowedOrigins:   trimAll(origins),
		AllowedMethods:   trimAll(methods),
		AllowedHeaders:   trimAll(headers),
		AllowCredentials: allowCreds,
		MaxAge:           maxAge,
	}
}

func trimAll(ss []string) []string {
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		out = append(out, strings.TrimSpace(s))
	}
	return out
}

func getEnvDefault(k, d string) string {
	if v := os.Getenv(k); v != "" { return v }
	return d
}

// CORSMiddleware returns an http.Handler that adds CORS headers and handles preflight
func CORSMiddleware(next http.Handler) http.Handler {
	cfg := loadCORSConfigFromEnv()
	if !cfg.Enabled { return next }

	allowedMethods := strings.Join(cfg.AllowedMethods, ", ")
	allowedHeaders := strings.Join(cfg.AllowedHeaders, ", ")
	allowedOrigins := cfg.AllowedOrigins
	wildcard := len(allowedOrigins) == 1 && allowedOrigins[0] == "*"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			// Non-CORS request
			next.ServeHTTP(w, r)
			return
		}

		// Determine allowed origin
		allowOrigin := ""
		if wildcard && !cfg.AllowCredentials {
			allowOrigin = "*"
		} else {
			for _, o := range allowedOrigins {
				if o == origin {
					allowOrigin = origin
					break
				}
			}
		}

		if allowOrigin == "" {
			// Origin not allowed
			w.WriteHeader(http.StatusForbidden)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		if cfg.AllowCredentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Vary", "Origin")

		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
			w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
			w.Header().Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
