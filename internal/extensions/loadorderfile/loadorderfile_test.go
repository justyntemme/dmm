package loadorderfile_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/loadorderfile"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func TestHandlerWritesOrderedSourcePathListAndRestore(t *testing.T) {
	gamePath := t.TempDir()
	stagingRoot := t.TempDir()
	workDir := t.TempDir()
	currentPath := filepath.Join(gamePath, "ConanSandbox", "Mods", "modlist.txt")
	if err := os.MkdirAll(filepath.Dir(currentPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(currentPath, []byte("old.pak\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	alphaSource := filepath.Join(stagingRoot, "alpha", "Alpha.pak")
	betaSource := filepath.Join(stagingRoot, "beta", "Beta.pak")
	handler := loadorderfile.Handler(loadorderfile.Options{
		TargetRelative: "ConanSandbox/Mods/modlist.txt",
		TargetRoot:     "ConanSandbox/Mods",
		ModTypes:       []string{"conanexiles-pak"},
		FileExtensions: []string{".pak"},
		LineMode:       loadorderfile.LineSourcePath,
		ModID:          "conanexiles-modlist",
	})
	result, err := handler(context.Background(), sdk.EventHandlerInput{
		GamePath:    gamePath,
		StagingRoot: stagingRoot,
		WorkDir:     workDir,
		Mappings: []deploy.FileMapping{
			{SourcePath: betaSource, TargetRelative: "ConanSandbox/Mods/Beta.pak", InstalledModID: 20, Priority: 20},
			{SourcePath: alphaSource, TargetRelative: "ConanSandbox/Mods/Alpha.pak", InstalledModID: 10, Priority: 10},
			{SourcePath: filepath.Join(stagingRoot, "readme.txt"), TargetRelative: "ConanSandbox/Mods/readme.txt", InstalledModID: 10, Priority: 10},
			{SourcePath: filepath.Join(stagingRoot, "other.pak"), TargetRelative: "Other/other.pak", InstalledModID: 30, Priority: 1},
		},
		Mods: []sdk.DeploymentMod{
			{ID: 10, Name: "Alpha", ModType: "conanexiles-pak", Priority: 10},
			{ID: 20, Name: "Beta", ModType: "conanexiles-pak", Priority: 20},
			{ID: 30, Name: "Other", ModType: "other", Priority: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	mapping := result.Mappings[0]
	if mapping.TargetRelative != "ConanSandbox/Mods/modlist.txt" || mapping.TargetPolicy != deploy.TargetPolicyPatchExisting || mapping.Strategy != deploy.StrategyCopy {
		t.Fatalf("mapping = %+v", mapping)
	}
	body, err := os.ReadFile(mapping.SourcePath)
	if err != nil {
		t.Fatal(err)
	}
	wantLines := []string{filepath.ToSlash(alphaSource), filepath.ToSlash(betaSource)}
	if strings.TrimSpace(string(body)) != strings.Join(wantLines, "\n") {
		t.Fatalf("modlist = %q", string(body))
	}
	restore, err := os.ReadFile(mapping.RestorePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(restore) != "old.pak\n" {
		t.Fatalf("restore = %q", string(restore))
	}
}

func TestHandlerSkipsWhenNoMatchingMappings(t *testing.T) {
	handler := loadorderfile.Handler(loadorderfile.Options{
		TargetRelative: "Mods/modlist.txt",
		TargetRoot:     "Mods",
		FileExtensions: []string{".pak"},
		EmptyMessage:   "empty",
	})
	result, err := handler(context.Background(), sdk.EventHandlerInput{
		WorkDir: t.TempDir(),
		Mappings: []deploy.FileMapping{
			{TargetRelative: "Mods/readme.txt"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 0 || len(result.Messages) != 1 || result.Messages[0] != "empty" {
		t.Fatalf("result = %+v", result)
	}
}
