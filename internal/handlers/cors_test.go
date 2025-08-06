package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestCORS_PreflightWildcardNoCreds(t *testing.T){
	os.Setenv("CORS_ENABLED","true")
	os.Setenv("CORS_ALLOWED_ORIGINS","*")
	os.Setenv("CORS_ALLOW_CREDENTIALS","false")
	h := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){ w.WriteHeader(200) }))
	req := httptest.NewRequest(http.MethodOptions, "/api/v2/presence/user", nil)
	req.Header.Set("Origin","http://example.com")
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)
	if rw.Code != http.StatusNoContent { t.Fatalf("expected 204, got %d", rw.Code) }
	if rw.Header().Get("Access-Control-Allow-Origin") != "*" { t.Fatalf("expected wildcard origin") }
}

func TestCORS_BlockDisallowedOrigin(t *testing.T){
	os.Setenv("CORS_ENABLED","true")
	os.Setenv("CORS_ALLOWED_ORIGINS","https://good.example.com")
	os.Setenv("CORS_ALLOW_CREDENTIALS","true")
	h := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){ w.WriteHeader(200) }))
	req := httptest.NewRequest(http.MethodGet, "/api/v2/presence/user", nil)
	req.Header.Set("Origin","https://bad.example.com")
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)
	if rw.Code != http.StatusForbidden { t.Fatalf("expected 403 for disallowed origin, got %d", rw.Code) }
}
