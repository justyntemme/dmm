package nexus

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestClientIncludesNexusErrorMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"code":403,"message":"paste the nxm link"}`))
	}))
	defer server.Close()

	client := NewClient("", WithBaseURL(server.URL))
	_, err := client.DownloadLinks(context.Background(), "stardewvalley", "1", "2", "", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "paste the nxm link") {
		t.Fatalf("error = %v", err)
	}
}

func TestSearchModsUsesGraphQLAndFiltersVortexResults(t *testing.T) {
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if r.Header.Get("apikey") != "secret" {
			t.Fatalf("apikey header = %q", r.Header.Get("apikey"))
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{
			"data": {
				"mods": {
					"totalCount": 3,
					"nodes": [
						{"modId":2400,"name":"SMAPI","summary":"Loader","version":"4.5.2","thumbnailUrl":"https://example.test/smapi.png","downloads":10,"endorsements":7,"updatedAt":"2026-01-02T03:04:05Z","supportsVortex":true},
						{"modId":485,"name":"Old Manual Mod","summary":"Manual","version":"1.0","thumbnailUrl":"","downloads":8,"endorsements":2,"updatedAt":"2026-01-01T00:00:00Z","supportsVortex":false},
						{"modId":1915,"name":"Content Patcher","summary":"Patches content","version":"2.9.1","thumbnailUrl":"","downloads":6,"endorsements":4,"updatedAt":"2026-01-03T00:00:00Z","supportsVortex":true}
					]
				}
			}
		}`))
	}))
	defer server.Close()

	client := NewClient("secret", WithGraphQLURL(server.URL))
	got, err := client.SearchMods(context.Background(), ModSearchRequest{
		GameDomain: "stardewvalley",
		Query:      "smapi",
		Sort:       "downloads",
		Count:      2,
		VortexOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.TotalCount != 3 {
		t.Fatalf("total_count = %d", got.TotalCount)
	}
	if len(got.Mods) != 2 {
		t.Fatalf("mods = %+v", got.Mods)
	}
	if got.Mods[0].ModID != 2400 || got.Mods[0].URL != "https://www.nexusmods.com/stardewvalley/mods/2400" {
		t.Fatalf("first mod = %+v", got.Mods[0])
	}
	if got.Mods[1].ModID != 1915 {
		t.Fatalf("second mod = %+v", got.Mods[1])
	}
	variables := captured["variables"].(map[string]any)
	if int(variables["count"].(float64)) != 6 {
		t.Fatalf("count = %v", variables["count"])
	}
	filter := variables["filter"].(map[string]any)
	if filter["op"] != "AND" || filter["gameDomainName"] == nil || filter["nameStemmed"] == nil || filter["supportsVortex"] == nil {
		t.Fatalf("filter = %+v", filter)
	}
}

func TestSearchModsReportsGraphQLError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errors":[{"message":"bad filter"}]}`))
	}))
	defer server.Close()

	client := NewClient("", WithGraphQLURL(server.URL))
	_, err := client.SearchMods(context.Background(), ModSearchRequest{GameDomain: "stardewvalley"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "bad filter") {
		t.Fatalf("error = %v", err)
	}
}
