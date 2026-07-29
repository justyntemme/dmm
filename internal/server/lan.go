package server

import (
	"net"
	"net/http"
	"net/netip"
)

func lanOnlyMiddleware(enabled func() bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !enabled() {
			next.ServeHTTP(w, r)
			return
		}
		if !isLANRemote(r.RemoteAddr) {
			http.Error(w, "LAN-only mode rejected this request", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isLANRemote(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	addr, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}

	return addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast()
}
