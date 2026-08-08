package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddleware(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	t.Run("allows health without token", func(t *testing.T) {
		handler := authMiddleware(func() string { return "secret" }, next)
		req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("allows api when auth disabled", func(t *testing.T) {
		handler := authMiddleware(func() string { return "" }, next)
		req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("rejects missing token", func(t *testing.T) {
		handler := authMiddleware(func() string { return "secret" }, next)
		req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("allows header token", func(t *testing.T) {
		handler := authMiddleware(func() string { return "secret" }, next)
		req := httptest.NewRequest(http.MethodPost, "/api/captured-installs", nil)
		req.Header.Set(dmmAuthHeader, "secret")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("allows websocket query token", func(t *testing.T) {
		handler := authMiddleware(func() string { return "secret" }, next)
		req := httptest.NewRequest(http.MethodGet, "/api/events/ws?token=secret", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d", rec.Code)
		}
	})
}
