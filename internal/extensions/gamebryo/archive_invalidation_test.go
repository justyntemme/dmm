package gamebryo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func TestEnsureINISectionKeysPreservesExistingContent(t *testing.T) {
	current := "[General]\r\nsLanguage=en\r\n\r\n[Archive]\r\nSResourceArchiveList=Fallout4 - Misc.ba2\r\n"
	patched := ensureINISectionKeys(current, "Archive", map[string]string{
		"bInvalidateOlderFiles":  "1",
		"sResourceDataDirsFinal": "",
	})

	if !strings.Contains(patched, "[General]\r\nsLanguage=en") {
		t.Fatalf("general section not preserved:\n%s", patched)
	}
	if !strings.Contains(patched, "SResourceArchiveList=Fallout4 - Misc.ba2") {
		t.Fatalf("archive list not preserved:\n%s", patched)
	}
	if !strings.Contains(patched, "bInvalidateOlderFiles=1") || !strings.Contains(patched, "sResourceDataDirsFinal=") {
		t.Fatalf("archive invalidation keys missing:\n%s", patched)
	}
	if !strings.Contains(patched, "\r\n") {
		t.Fatalf("expected CRLF line endings:\n%s", patched)
	}
}

func TestArchiveInvalidationHandlerReturnsPatchMappingWithRestore(t *testing.T) {
	root := t.TempDir()
	gamePath := filepath.Join(root, "steamapps", "common", "Fallout 4")
	documentsRoot := filepath.Join(root, "steamapps", "compatdata", "377160", "pfx", "drive_c", "users", "steamuser", "Documents", "My Games", "Fallout4")
	iniPath := filepath.Join(documentsRoot, "Fallout4.ini")
	if err := os.MkdirAll(filepath.Dir(iniPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(iniPath, []byte("[Archive]\nSResourceArchiveList=Fallout4 - Misc.ba2\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	handler := ArchiveInvalidationHandler(ArchiveInvalidationOptions{
		MyGamesPath: "Fallout4",
		ININame:     "Fallout4.ini",
		DataRoot:    "Data",
	})
	result, err := handler(context.Background(), sdk.EventHandlerInput{
		AppID:       "377160",
		GamePath:    gamePath,
		LibraryPath: root,
		ProfileID:   9,
		StagingRoot: filepath.Join(root, "staging"),
		WorkDir:     filepath.Join(root, "work"),
		Mappings: []deploy.FileMapping{{
			TargetRelative: "Data/Meshes/example.nif",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	mapping := result.Mappings[0]
	if mapping.TargetRoot != documentsRoot || mapping.TargetRelative != "Fallout4.ini" || mapping.TargetPolicy != deploy.TargetPolicyPatchExisting || mapping.RestorePath == "" {
		t.Fatalf("mapping = %+v", mapping)
	}
	patched, err := os.ReadFile(mapping.SourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(patched), "bInvalidateOlderFiles=1") || !strings.Contains(string(patched), "sResourceDataDirsFinal=") {
		t.Fatalf("patched ini missing settings:\n%s", patched)
	}
	restore, err := os.ReadFile(mapping.RestorePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(restore) != "[Archive]\nSResourceArchiveList=Fallout4 - Misc.ba2\n" {
		t.Fatalf("restore body = %q", restore)
	}
	if strings.Contains(filepath.ToSlash(mapping.RestorePath), "/work/") || strings.Contains(filepath.ToSlash(mapping.SourcePath), "/work/") {
		t.Fatalf("generated paths should not use ephemeral work dir: %+v", mapping)
	}
}

func TestArchiveInvalidationProfilePatchesAttachToINI(t *testing.T) {
	files := LocalGameSettingsProfileFiles(LocalGameSettingsOptions{
		GameID:      "fallout4",
		MyGamesPath: "Fallout4",
		Files: []LocalGameSettingFile{
			{Name: "Fallout4.ini"},
			{Name: "Fallout4Prefs.ini"},
		},
		FilePatches: ArchiveInvalidationProfilePatches(ArchiveInvalidationOptions{
			ININame: "Fallout4.ini",
		}),
	})
	if len(files) != 2 {
		t.Fatalf("profile files = %+v", files)
	}
	if len(files[0].Patches) != 2 {
		t.Fatalf("Fallout4.ini patches = %+v", files[0].Patches)
	}
	if len(files[1].Patches) != 0 {
		t.Fatalf("Fallout4Prefs.ini patches = %+v", files[1].Patches)
	}
	seen := map[string]string{}
	for _, patch := range files[0].Patches {
		if patch.Kind != sdk.ProfileFilePatchINIKey || patch.FeatureID != "local_game_settings" || patch.Section != "Archive" {
			t.Fatalf("patch = %+v", patch)
		}
		seen[patch.Key] = patch.Value
	}
	if seen["bInvalidateOlderFiles"] != "1" {
		t.Fatalf("bInvalidateOlderFiles patch missing in %+v", files[0].Patches)
	}
	if value, ok := seen["sResourceDataDirsFinal"]; !ok || value != "" {
		t.Fatalf("sResourceDataDirsFinal patch missing in %+v", files[0].Patches)
	}
}

func TestLocalSavegameManagementDeclaresSafeGamebryoPaths(t *testing.T) {
	spec := LocalSavegameManagement(LocalGameSettingsOptions{
		GameID:      "fallout4",
		MyGamesPath: "Fallout4",
		SaveININame: "Fallout4Custom.ini",
	})
	if spec.ID != "fallout4-gamebryo-savegames" || spec.Path != "My Games/Fallout4" || spec.LocalPath != "Saves/{profile_id}" || spec.GlobalPath != "Saves" {
		t.Fatalf("savegame spec = %+v", spec)
	}
	if spec.LocalFeatureID != "local_saves" || len(spec.SaveExtensions) == 0 || len(spec.SidecarPatterns) == 0 {
		t.Fatalf("savegame spec missing defaults = %+v", spec)
	}
}

func TestArchiveInvalidationHandlerKeepsManagedPatchMapping(t *testing.T) {
	root := t.TempDir()
	documentsRoot := filepath.Join(root, "steamapps", "compatdata", "377160", "pfx", "drive_c", "users", "steamuser", "Documents", "My Games", "Fallout4")
	iniPath := filepath.Join(documentsRoot, "Fallout4.ini")
	stagingRoot := filepath.Join(root, "staging")
	restorePath := filepath.Join(stagingRoot, archiveInvalidationGeneratedDir, "377160", "9", "restore", "Fallout4.ini")
	if err := os.MkdirAll(filepath.Dir(iniPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(restorePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(iniPath, []byte("[Archive]\nbInvalidateOlderFiles=1\nsResourceDataDirsFinal=\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(restorePath, []byte("[Archive]\nSResourceArchiveList=Fallout4 - Misc.ba2\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	handler := ArchiveInvalidationHandler(ArchiveInvalidationOptions{
		MyGamesPath: "Fallout4",
		ININame:     "Fallout4.ini",
		DataRoot:    "Data",
	})
	result, err := handler(context.Background(), sdk.EventHandlerInput{
		AppID:       "377160",
		LibraryPath: root,
		ProfileID:   9,
		StagingRoot: stagingRoot,
		WorkDir:     filepath.Join(root, "staging", "work"),
		Mappings: []deploy.FileMapping{{
			TargetRelative: "Data/Meshes/example.nif",
		}},
		ManagedFiles: []deploy.AppliedFile{{
			TargetPath:  iniPath,
			RestorePath: restorePath,
			Strategy:    deploy.StrategyCopy,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	if result.Mappings[0].RestorePath != restorePath {
		t.Fatalf("restore path = %q, want %q", result.Mappings[0].RestorePath, restorePath)
	}
}

func TestArchiveInvalidationHandlerSkipsWithoutDataMappings(t *testing.T) {
	handler := ArchiveInvalidationHandler(ArchiveInvalidationOptions{
		MyGamesPath: "Fallout4",
		ININame:     "Fallout4.ini",
		DataRoot:    "Data",
	})
	result, err := handler(context.Background(), sdk.EventHandlerInput{
		AppID:       "377160",
		LibraryPath: t.TempDir(),
		WorkDir:     t.TempDir(),
		Mappings: []deploy.FileMapping{{
			TargetRelative: "plugins.txt",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 0 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
}

func TestArchiveInvalidationHandlerSkipsMissingMyGamesFolder(t *testing.T) {
	root := t.TempDir()
	handler := ArchiveInvalidationHandler(ArchiveInvalidationOptions{
		MyGamesPath: "Fallout4",
		ININame:     "Fallout4.ini",
		DataRoot:    "Data",
	})
	result, err := handler(context.Background(), sdk.EventHandlerInput{
		AppID:       "377160",
		LibraryPath: root,
		WorkDir:     filepath.Join(root, "work"),
		Mappings: []deploy.FileMapping{{
			TargetRelative: "Data/Example.esp",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 0 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
}
