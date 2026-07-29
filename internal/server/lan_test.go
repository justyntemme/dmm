package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsLANRemote(t *testing.T) {
	tests := map[string]bool{
		"127.0.0.1:1234":      true,
		"192.168.8.22:1234":   true,
		"10.1.2.3:1234":       true,
		"172.16.2.3:1234":     true,
		"169.254.1.2:1234":    true,
		"8.8.8.8:1234":        false,
		"malformed-address":   false,
		"[::1]:1234":          true,
		"[fe80::1]:1234":      true,
		"[2606:4700::1111]:1": false,
	}
	for remote, want := range tests {
		if got := isLANRemote(remote); got != want {
			t.Fatalf("isLANRemote(%q) = %v, want %v", remote, got, want)
		}
	}
}

func TestLANOnlyMiddleware(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	t.Run("rejects public remote when enabled", func(t *testing.T) {
		handler := lanOnlyMiddleware(func() bool { return true }, next)
		req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
		req.RemoteAddr = "8.8.8.8:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("allows lan remote when enabled", func(t *testing.T) {
		handler := lanOnlyMiddleware(func() bool { return true }, next)
		req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
		req.RemoteAddr = "192.168.8.22:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("allows public remote when disabled", func(t *testing.T) {
		handler := lanOnlyMiddleware(func() bool { return false }, next)
		req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
		req.RemoteAddr = "8.8.8.8:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
	})
}
