package loadorderjson

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func TestHandlerGeneratesSortedModNamesFromManagedMappings(t *testing.T) {
	gamePath := t.TempDir()
	stagingRoot := t.TempDir()
	writeFile(t, filepath.Join(gamePath, "Game", "Mods", "loadorder.json"), `{"modNames":["old"]}`)

	handler := Handler(Options{
		ID:                     "game-loadorder",
		TargetRelative:         "Game/Mods/loadorder.json",
		EntryRoot:              "Game/Mods",
		ModTypes:               []string{"official"},
		ManifestFileName:       "manifest.json",
		ManifestParentModTypes: []string{"dinput"},
		ExcludedNames:          []string{"Default"},
	})
	result, err := handler(context.Background(), sdk.EventHandlerInput{
		AppID:       "100",
		GamePath:    gamePath,
		ProfileID:   5,
		StagingRoot: stagingRoot,
		Mappings: []deploy.FileMapping{
			{TargetRelative: "Game/Mods/Late/manifest.json", InstalledModID: 2, Priority: 20},
			{TargetRelative: "Game/Mods/Late/file.json", InstalledModID: 2, Priority: 20},
			{TargetRelative: "Game/Mods/Early/manifest.json", InstalledModID: 1, Priority: 10},
			{TargetRelative: "Game/Mods/Default/manifest.json", InstalledModID: 3, Priority: 1},
			{TargetRelative: "Game/Managed/manifest.json", InstalledModID: 4, Priority: 5},
		},
		Mods: []sdk.DeploymentMod{
			{ID: 1, Name: "Early Mod", ModType: "official", Priority: 10},
			{ID: 2, Name: "Late Mod", ModType: "official", Priority: 20},
			{ID: 3, Name: "Default", ModType: "official", Priority: 1},
			{ID: 4, Name: "Injector", ModType: "dinput", Priority: 5},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	mapping := result.Mappings[0]
	if mapping.TargetRelative != "Game/Mods/loadorder.json" || mapping.TargetPolicy != deploy.TargetPolicyPatchExisting || mapping.RestorePath == "" {
		t.Fatalf("mapping = %+v", mapping)
	}
	var body map[string][]string
	if err := json.Unmarshal([]byte(readFile(t, mapping.SourcePath)), &body); err != nil {
		t.Fatal(err)
	}
	assertNames(t, body["modNames"], []string{"Managed", "Early", "Late"})
	assertNames(t, decodeNames(t, readFile(t, mapping.RestorePath)), []string{"old"})
}

func TestHandlerSkipsWhenNoMatchingMappings(t *testing.T) {
	handler := Handler(Options{
		TargetRelative: "Game/Mods/loadorder.json",
		EntryRoot:      "Game/Mods",
		ModTypes:       []string{"official"},
	})
	result, err := handler(context.Background(), sdk.EventHandlerInput{
		AppID:       "100",
		GamePath:    t.TempDir(),
		ProfileID:   5,
		StagingRoot: t.TempDir(),
		Mappings: []deploy.FileMapping{{
			TargetRelative: "Other/Mod/file.txt",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 0 || len(result.Messages) != 1 {
		t.Fatalf("result = %+v", result)
	}
}

func TestHandlerWritesEmptyFileWhenCandidatesAreExcluded(t *testing.T) {
	gamePath := t.TempDir()
	handler := Handler(Options{
		TargetRelative: "Game/Mods/loadorder.json",
		EntryRoot:      "Game/Mods",
		ExcludedNames:  []string{"Default"},
		EmptyMessage:   "generated empty",
	})
	result, err := handler(context.Background(), sdk.EventHandlerInput{
		AppID:       "100",
		GamePath:    gamePath,
		ProfileID:   5,
		StagingRoot: t.TempDir(),
		Mappings: []deploy.FileMapping{{
			TargetRelative: "Game/Mods/Default/manifest.json",
			InstalledModID: 1,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	assertNames(t, decodeNames(t, readFile(t, result.Mappings[0].SourcePath)), nil)
}

func decodeNames(t *testing.T, body string) []string {
	t.Helper()
	var parsed map[string][]string
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatal(err)
	}
	return parsed["modNames"]
}

func assertNames(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("names = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("names = %+v, want %+v", got, want)
		}
	}
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
