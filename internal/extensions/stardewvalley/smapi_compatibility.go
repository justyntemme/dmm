package stardewvalley

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const smapiIOAPIVersion = "3.0.0"

var (
	smapiCompatibilityEndpoint   = "https://smapi.io/api/v3.0/mods"
	smapiCompatibilityHTTPClient = &http.Client{Timeout: 10 * time.Second}
)

type smapiCompatibilityRequest struct {
	Mods                    []smapiCompatibilityQuery `json:"mods"`
	IncludeExtendedMetadata bool                      `json:"includeExtendedMetadata"`
	APIVersion              string                    `json:"apiVersion"`
}

type smapiCompatibilityQuery struct {
	ID               string `json:"id"`
	InstalledVersion string `json:"installedVersion,omitempty"`
}

type smapiCompatibilityResult struct {
	ID              string `json:"id"`
	SuggestedUpdate *struct {
		Version string `json:"version"`
		URL     string `json:"url"`
	} `json:"suggestedUpdate"`
	Metadata struct {
		ID                   []string `json:"id"`
		Name                 string   `json:"name"`
		CompatibilityStatus  string   `json:"compatibilityStatus"`
		CompatibilitySummary string   `json:"compatibilitySummary"`
		Main                 *struct {
			Version string `json:"version"`
			URL     string `json:"url"`
		} `json:"main"`
	} `json:"metadata"`
	Errors []string `json:"errors"`
}

type smapiModCompatibilityQuery struct {
	Mod     sdk.DeploymentMod
	Queries []smapiCompatibilityQuery
}

func checkSMAPICompatibility(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	modQueries := smapiCompatibilityQueries(input.Mods)
	if len(modQueries) == 0 {
		return sdk.EventHandlerResult{}, nil
	}
	queries := uniqueSMAPIQueries(modQueries)
	results, err := querySMAPICompatibility(ctx, queries)
	if err != nil {
		return sdk.EventHandlerResult{
			Notices: []sdk.EventNotice{{
				Message: "SMAPI.io compatibility lookup failed: " + err.Error(),
				HelpURL: "https://smapi.io/mods",
			}},
		}, nil
	}
	resultByID := smapiCompatibilityResultsByID(results)
	notices := smapiCompatibilityNotices(modQueries, resultByID)
	return sdk.EventHandlerResult{Notices: notices}, nil
}

func smapiCompatibilityQueries(mods []sdk.DeploymentMod) []smapiModCompatibilityQuery {
	var out []smapiModCompatibilityQuery
	for _, mod := range mods {
		if !mod.Enabled {
			continue
		}
		var queries []smapiCompatibilityQuery
		seen := map[string]struct{}{}
		for _, metadata := range mod.Metadata {
			if strings.TrimSpace(metadata.Kind) != MetadataKindSMAPIManifest {
				continue
			}
			version := firstSMAPIMetadataValue(metadata.ManifestVersion, metadata.Version)
			names := append([]string(nil), metadata.AdditionalLogicalFileNames...)
			if uniqueID := strings.TrimSpace(metadata.UniqueID); uniqueID != "" {
				names = append(names, uniqueID)
			}
			for _, name := range names {
				id := strings.ToLower(strings.TrimSpace(name))
				if id == "" {
					continue
				}
				if _, ok := seen[id]; ok {
					continue
				}
				seen[id] = struct{}{}
				queries = append(queries, smapiCompatibilityQuery{
					ID:               id,
					InstalledVersion: version,
				})
			}
		}
		if len(queries) == 0 {
			continue
		}
		out = append(out, smapiModCompatibilityQuery{
			Mod:     mod,
			Queries: queries,
		})
	}
	return out
}

func uniqueSMAPIQueries(mods []smapiModCompatibilityQuery) []smapiCompatibilityQuery {
	type queryKey struct {
		id      string
		version string
	}
	seen := map[queryKey]struct{}{}
	var out []smapiCompatibilityQuery
	for _, mod := range mods {
		for _, query := range mod.Queries {
			key := queryKey{id: strings.ToLower(strings.TrimSpace(query.ID)), version: strings.TrimSpace(query.InstalledVersion)}
			if key.id == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, smapiCompatibilityQuery{ID: key.id, InstalledVersion: key.version})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ID != out[j].ID {
			return out[i].ID < out[j].ID
		}
		return out[i].InstalledVersion < out[j].InstalledVersion
	})
	return out
}

func querySMAPICompatibility(ctx context.Context, queries []smapiCompatibilityQuery) ([]smapiCompatibilityResult, error) {
	if len(queries) == 0 {
		return nil, nil
	}
	requestBody, err := json.Marshal(smapiCompatibilityRequest{
		Mods:                    queries,
		IncludeExtendedMetadata: true,
		APIVersion:              smapiIOAPIVersion,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, smapiCompatibilityEndpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := smapiCompatibilityHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("SMAPI.io returned HTTP %d", resp.StatusCode)
	}
	var results []smapiCompatibilityResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}
	return results, nil
}

func smapiCompatibilityResultsByID(results []smapiCompatibilityResult) map[string][]smapiCompatibilityResult {
	out := map[string][]smapiCompatibilityResult{}
	for _, result := range results {
		ids := append([]string{result.ID}, result.Metadata.ID...)
		for _, id := range ids {
			id = strings.ToLower(strings.TrimSpace(id))
			if id == "" {
				continue
			}
			out[id] = append(out[id], result)
		}
	}
	return out
}

func smapiCompatibilityNotices(modQueries []smapiModCompatibilityQuery, resultByID map[string][]smapiCompatibilityResult) []sdk.EventNotice {
	var notices []sdk.EventNotice
	for _, modQuery := range modQueries {
		worst, ok := worstSMAPICompatibilityResult(modQuery.Queries, resultByID)
		if !ok {
			continue
		}
		notice, ok := smapiCompatibilityNotice(modQuery.Mod, worst)
		if ok {
			notices = append(notices, notice)
		}
	}
	sort.SliceStable(notices, func(i, j int) bool {
		return notices[i].Message < notices[j].Message
	})
	if len(notices) > 8 {
		remaining := len(notices) - 8
		notices = append(notices[:8], sdk.EventNotice{
			Message: fmt.Sprintf("SMAPI.io reported %d more compatibility notice%s. Review installed Stardew mods for details.", remaining, pluralS(remaining)),
			HelpURL: "https://smapi.io/mods",
		})
	}
	return notices
}

func worstSMAPICompatibilityResult(queries []smapiCompatibilityQuery, resultByID map[string][]smapiCompatibilityResult) (smapiCompatibilityResult, bool) {
	var out smapiCompatibilityResult
	found := false
	for _, query := range queries {
		for _, result := range resultByID[strings.ToLower(strings.TrimSpace(query.ID))] {
			if !found || smapiCompatibilityRank(result) < smapiCompatibilityRank(out) {
				out = result
				found = true
			}
		}
	}
	return out, found
}

func smapiCompatibilityNotice(mod sdk.DeploymentMod, result smapiCompatibilityResult) (sdk.EventNotice, bool) {
	status := strings.ToLower(strings.TrimSpace(result.Metadata.CompatibilityStatus))
	update := ""
	updateURL := ""
	if result.SuggestedUpdate != nil {
		update = strings.TrimSpace(result.SuggestedUpdate.Version)
		updateURL = strings.TrimSpace(result.SuggestedUpdate.URL)
	}
	if status == "" && len(result.Errors) == 0 && update == "" {
		return sdk.EventNotice{}, false
	}
	if status == "ok" && update == "" && len(result.Errors) == 0 {
		return sdk.EventNotice{}, false
	}
	name := firstSMAPIMetadataValue(result.Metadata.Name, mod.Name, result.ID, fmt.Sprintf("Mod %d", mod.ID))
	var parts []string
	if status != "" && status != "ok" {
		summary := strings.TrimSpace(result.Metadata.CompatibilitySummary)
		if summary == "" {
			summary = "SMAPI.io reports compatibility status " + status + "."
		}
		parts = append(parts, summary)
	}
	if len(result.Errors) > 0 {
		parts = append(parts, "SMAPI.io reported: "+strings.Join(cleanSMAPIStrings(result.Errors), "; "))
	}
	if update != "" {
		parts = append(parts, "Suggested update: "+update+".")
	}
	if len(parts) == 0 {
		return sdk.EventNotice{}, false
	}
	helpURL := updateURL
	if helpURL == "" && result.Metadata.Main != nil {
		helpURL = strings.TrimSpace(result.Metadata.Main.URL)
	}
	if helpURL == "" {
		helpURL = "https://smapi.io/mods"
	}
	return sdk.EventNotice{
		Message: strings.TrimSpace(name + ": " + strings.Join(parts, " ")),
		HelpURL: helpURL,
	}, true
}

func smapiCompatibilityRank(result smapiCompatibilityResult) int {
	status := strings.ToLower(strings.TrimSpace(result.Metadata.CompatibilityStatus))
	for index, candidate := range []string{"broken", "obsolete", "abandoned", "unofficial", "workaround", "unknown", "optional", "ok"} {
		if status == candidate {
			return index
		}
	}
	if len(result.Errors) > 0 {
		return 0
	}
	return 5
}

func cleanSMAPIStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func firstSMAPIMetadataValue(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func pluralS(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
