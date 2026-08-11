package stardewvalley

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersFOMODInstallerChoice(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})

	choice, ok := registry.InstallerChoiceForSteamApp(SteamAppID, "fomod")
	if !ok {
		t.Fatal("missing FOMOD installer choice capability")
	}
	if choice.ID != "vortex:stardewvalley:fomod" {
		t.Fatalf("choice id = %q", choice.ID)
	}
	if choice.ModType != "stardew-smapi-mod" || choice.TargetRoot != ModsRelativePath {
		t.Fatalf("choice target = %+v", choice)
	}
	if choice.DestinationPrefixMode != sdk.InstallerChoiceDestinationPrefixModuleBaseName {
		t.Fatalf("destination prefix mode = %q", choice.DestinationPrefixMode)
	}
	if !registry.HasEventHandlerForSteamApp(SteamAppID, sdk.EventCheckModsVersion) {
		t.Fatal("missing SMAPI compatibility version-check handler")
	}
}

func TestExtensionRegistersSMAPILogAction(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})

	_, action, ok := registry.ExtensionActionForSteamApp(SteamAppID, "stardew-smapi-log")
	if !ok {
		t.Fatal("missing SMAPI log action")
	}
	if action.Kind != sdk.ExtensionActionKindOpenPath || action.OpenPath == nil {
		t.Fatalf("SMAPI log action target = %+v", action)
	}
	if action.OpenPath.Base != sdk.OpenDirectoryBaseUserConfig || action.OpenPath.FallbackBase != sdk.OpenDirectoryBaseUserConfig {
		t.Fatalf("SMAPI log action bases = %+v", action.OpenPath)
	}
	if action.OpenPath.RelativePath != "StardewValley/ErrorLogs/SMAPI-crash.txt" || action.OpenPath.FallbackRelative != "StardewValley/ErrorLogs" {
		t.Fatalf("SMAPI log action paths = %+v", action.OpenPath)
	}
}

func TestSMAPIManifestDependenciesAreRecommendations(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	body := []byte(`{
		"Name": "Example Content Pack",
		"UniqueID": "example.contentpack",
		"Version": "1.0.0",
		"ContentPackFor": {
			"UniqueID": "Pathoschild.ContentPatcher",
			"MinimumVersion": "2.0.0"
		},
		"Dependencies": [
			{"UniqueID": "spacechase0.GenericModConfigMenu", "IsRequired": true},
			{"UniqueID": "Pathoschild.LookupAnything", "IsRequired": false}
		]
	}`)
	if err := os.WriteFile(manifestPath, body, 0o600); err != nil {
		t.Fatal(err)
	}

	metadata := smapiManifestMetadata(manifestPath)
	if metadata.ContentPackFor == nil || metadata.ContentPackFor.Required {
		t.Fatalf("content pack dependency = %+v", metadata.ContentPackFor)
	}
	if len(metadata.Dependencies) != 2 {
		t.Fatalf("dependencies = %+v", metadata.Dependencies)
	}
	for _, dependency := range metadata.Dependencies {
		if dependency.Required {
			t.Fatalf("SMAPI manifest dependency should be non-blocking to match Vortex recommendation behavior: %+v", dependency)
		}
	}
}

func TestSMAPICompatibilityCheckQueuesNoticesFromSMAPIIO(t *testing.T) {
	var captured smapiCompatibilityRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{
			"id": "pathoschild.lookupanything",
			"suggestedUpdate": {"version": "2.0.0", "url": "https://example.test/update"},
			"metadata": {
				"id": ["Pathoschild.LookupAnything"],
				"name": "Lookup Anything",
				"compatibilityStatus": "obsolete",
				"compatibilitySummary": "Use the latest version.",
				"main": {"url": "https://smapi.io/mods"}
			},
			"errors": []
		}]`))
	}))
	defer server.Close()
	restoreSMAPICompatibilityEndpoint(t, server.URL, server.Client())

	result, err := checkSMAPICompatibility(context.Background(), sdk.EventHandlerInput{
		Mods: []sdk.DeploymentMod{{
			ID:      42,
			Name:    "Lookup Anything",
			Enabled: true,
			Metadata: []installplan.ModMetadata{{
				Kind:                       MetadataKindSMAPIManifest,
				UniqueID:                   "Pathoschild.LookupAnything",
				Version:                    "1.0.0",
				ManifestVersion:            "1.0.0",
				AdditionalLogicalFileNames: []string{"pathoschild.lookupanything"},
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(captured.Mods) != 1 || captured.Mods[0].ID != "pathoschild.lookupanything" || captured.Mods[0].InstalledVersion != "1.0.0" || captured.APIVersion != smapiIOAPIVersion || !captured.IncludeExtendedMetadata {
		t.Fatalf("captured request = %+v", captured)
	}
	if len(result.Notices) != 1 {
		t.Fatalf("notices = %+v", result.Notices)
	}
	notice := result.Notices[0]
	if !strings.Contains(notice.Message, "Lookup Anything") || !strings.Contains(notice.Message, "Use the latest version.") || !strings.Contains(notice.Message, "Suggested update: 2.0.0") {
		t.Fatalf("notice = %+v", notice)
	}
	if notice.HelpURL != "https://example.test/update" {
		t.Fatalf("notice help URL = %q", notice.HelpURL)
	}
}

func TestSMAPICompatibilityCheckSkipsModsWithoutSMAPIManifest(t *testing.T) {
	restoreSMAPICompatibilityEndpoint(t, "http://127.0.0.1:1", http.DefaultClient)
	result, err := checkSMAPICompatibility(context.Background(), sdk.EventHandlerInput{
		Mods: []sdk.DeploymentMod{{
			ID:      7,
			Name:    "Plain Mod",
			Enabled: true,
			Metadata: []installplan.ModMetadata{{
				Kind:     "other",
				UniqueID: "example.Plain",
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Notices) != 0 {
		t.Fatalf("notices = %+v", result.Notices)
	}
}

func restoreSMAPICompatibilityEndpoint(t *testing.T, endpoint string, client *http.Client) {
	t.Helper()
	oldEndpoint := smapiCompatibilityEndpoint
	oldClient := smapiCompatibilityHTTPClient
	smapiCompatibilityEndpoint = endpoint
	smapiCompatibilityHTTPClient = client
	t.Cleanup(func() {
		smapiCompatibilityEndpoint = oldEndpoint
		smapiCompatibilityHTTPClient = oldClient
	})
}
