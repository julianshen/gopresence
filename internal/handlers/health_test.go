package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type okChecker struct{}
func (o *okChecker) Ready(ctx context.Context) error { return nil }

type errChecker struct{}
func (e *errChecker) Ready(ctx context.Context) error { return context.DeadlineExceeded }

func TestHealth_Liveness(t *testing.T){
	h := NewHealthHandler(&okChecker{})
	r := httptest.NewRequest(http.MethodGet, "/health/liveness", nil)
	rw := httptest.NewRecorder()
	h.Liveness(rw, r)
	if rw.Code != http.StatusOK { t.Fatalf("expected 200, got %d", rw.Code) }
}

func TestHealth_Readiness_OK(t *testing.T){
	h := NewHealthHandler(&okChecker{})
	r := httptest.NewRequest(http.MethodGet, "/health/readiness", nil)
	rw := httptest.NewRecorder()
	h.Readiness(rw, r)
	if rw.Code != http.StatusOK { t.Fatalf("expected 200, got %d", rw.Code) }
}

func TestHealth_Readiness_Fail(t *testing.T){
	h := NewHealthHandler(&errChecker{})
	r := httptest.NewRequest(http.MethodGet, "/health/readiness", nil)
	rw := httptest.NewRecorder()
	h.Readiness(rw, r)
	if rw.Code != http.StatusServiceUnavailable { t.Fatalf("expected 503, got %d", rw.Code) }
}
