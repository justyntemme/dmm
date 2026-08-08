package server

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

const dmmAuthHeader = "X-DMM-Token"

func authMiddleware(token func() string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api/health" {
			next.ServeHTTP(w, r)
			return
		}

		required := strings.TrimSpace(token())
		if required == "" {
			next.ServeHTTP(w, r)
			return
		}
		if authTokenEqual(authTokenFromRequest(r), required) {
			next.ServeHTTP(w, r)
			return
		}
		http.Error(w, "DMM API authentication required", http.StatusUnauthorized)
	})
}

func authTokenFromRequest(r *http.Request) string {
	if token := strings.TrimSpace(r.Header.Get(dmmAuthHeader)); token != "" {
		return token
	}
	if value := strings.TrimSpace(r.Header.Get("Authorization")); value != "" {
		if token, ok := strings.CutPrefix(value, "Bearer "); ok {
			return strings.TrimSpace(token)
		}
	}
	if token := strings.TrimSpace(r.URL.Query().Get("token")); token != "" {
		return token
	}
	return strings.TrimSpace(r.URL.Query().Get("dmm_token"))
}

func authTokenEqual(provided, required string) bool {
	provided = strings.TrimSpace(provided)
	required = strings.TrimSpace(required)
	if provided == "" || required == "" || len(provided) != len(required) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(required)) == 1
}
