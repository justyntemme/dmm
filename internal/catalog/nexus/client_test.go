package nexus

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientSetsAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("apikey") != "secret" {
			t.Fatalf("apikey header = %q", r.Header.Get("apikey"))
		}
		_, _ = w.Write([]byte(`{"user_id":1,"name":"deck","email":"deck@example.com","is_premium":false,"is_supporter":false}`))
	}))
	defer server.Close()

	client := NewClient("secret", WithBaseURL(server.URL))
	got, err := client.Validate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "deck" {
		t.Fatalf("validate = %+v", got)
	}
}

func TestDownloadLinksIncludesNXMParameters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "abc" || r.URL.Query().Get("expires") != "123" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`[{"name":"Nexus CDN","short_name":"Nexus","URI":"https://example.test/mod.zip"}]`))
	}))
	defer server.Close()

	client := NewClient("", WithBaseURL(server.URL))
	links, err := client.DownloadLinks(context.Background(), "fallout4", "10", "20", "abc", "123")
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 1 || links[0].URI == "" {
		t.Fatalf("links = %+v", links)
	}
}
