package server

import "testing"

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
