package stardewvalley

import (
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
