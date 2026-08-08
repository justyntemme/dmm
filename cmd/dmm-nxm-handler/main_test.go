package main

import (
	"strings"
	"testing"
)

func TestRedactTextRemovesNXMCredentialsFromJSONBodies(t *testing.T) {
	in := `{"download_links":[{"URI":"https://supporter-files.nexus-cdn.com/file.zip?expires=1785310399\u0026md5=abc123\u0026user_id=5316165"}],"resolved":{"source_url":"nxm://stardewvalley/mods/239/files/165575?key=secret-key\u0026expires=1785468791","nxm_key":"secret-key","expires":"1785468791"},"token":"secret-token"}`

	got := redactText(in)
	for _, leaked := range []string{"secret-key", "1785468791", "1785310399", "abc123", "secret-token"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("redacted text leaked %q: %s", leaked, got)
		}
	}
	for _, want := range []string{"key=[redacted]", "expires=[redacted]", "md5=[redacted]", `"nxm_key":"[redacted]"`, `"token":"[redacted]"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("redacted text missing %q: %s", want, got)
		}
	}
}
