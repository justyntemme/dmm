package stardewvalley

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
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
