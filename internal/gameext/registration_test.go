package gameext

import (
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestNewExtensionRegistersVortexStyleDomains(t *testing.T) {
	extension, err := NewExtension("sample", "Sample Game", func(r *Registrar) {
		r.RegisterGame(GameRegistration{
			SteamAppIDs:  []string{"100", "100"},
			NexusDomains: []string{"samplegame", "samplegame"},
			VortexGameID: "samplegame",
		})
		r.RegisterModType(installplan.ModTypeSpec{ID: "mod", TargetRoot: "Mods"})
		r.RegisterInstaller(installplan.InstallerSpec{
			ID:                "sample:installer",
			VortexInstallerID: "sample-installer",
			ModType:           "mod",
			InstructionMode:   installplan.InstructionManifestFolders,
		})
		r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
			ID:       "sample-loader",
			Name:     "Sample Loader",
			Kind:     "mod-loader",
			Required: true,
			ModTypes: []string{"mod"},
		})
		r.RegisterLaunchTool(LaunchToolSpec{
			ID:                 "loader",
			Name:               "Sample Loader",
			ExecutableRelative: "loader",
			RequiredFiles:      []string{"loader", "loader.dll"},
			DefaultPrimary:     true,
			ModTypes:           []string{"mod"},
			ProviderModTypes:   []string{"loader-mod"},
		})
		r.RegisterMerge(MergeSpec{ID: "merge", Name: "Merge"})
		r.RegisterLoadOrder(LoadOrderSpec{ID: "load-order", Name: "Load Order"})
		r.RegisterEventHandler(EventHandlerSpec{Event: "will-deploy", Name: "Prepare"})
	})
	if err != nil {
		t.Fatal(err)
	}
	if extension.ID != "sample" || extension.Name != "Sample Game" {
		t.Fatalf("extension identity = %+v", extension)
	}
	if len(extension.SteamAppIDs) != 1 || extension.SteamAppIDs[0] != "100" {
		t.Fatalf("steam app ids = %#v", extension.SteamAppIDs)
	}
	if extension.InstallPlan.VortexGameID != "samplegame" || extension.RuntimeRequirements.SteamAppID != "100" {
		t.Fatalf("registered specs = %+v %+v", extension.InstallPlan, extension.RuntimeRequirements)
	}

	registry := NewRegistry([]Extension{extension})
	if !registry.SupportsSteamApp("100") {
		t.Fatal("registry did not support registered Steam app")
	}
	if appID, ok := registry.SteamAppIDForNexusDomain("samplegame"); !ok || appID != "100" {
		t.Fatalf("nexus domain mapping = %q %v", appID, ok)
	}
	if _, _, ok := registry.RequiredPrimaryLaunchToolForSteamApp("100", []gamehandler.RuntimeMod{{Enabled: true, ModType: "mod"}}); !ok {
		t.Fatal("primary launch tool did not match enabled mod type")
	}
	if tool, ok := registry.ModTypeProvidesLaunchTool("100", "loader-mod"); !ok || tool.ID != "loader" {
		t.Fatalf("launch tool provider lookup = %+v %v", tool, ok)
	}
	summaries := registry.ExtensionSummaries()
	if len(summaries) != 1 {
		t.Fatalf("summaries = %+v", summaries)
	}
	summary := summaries[0]
	if summary.ID != "sample" || summary.VortexGameID != "samplegame" {
		t.Fatalf("summary identity = %+v", summary)
	}
	if len(summary.Capabilities.Installers) != 1 || summary.Capabilities.Installers[0].ID != "sample:installer" {
		t.Fatalf("installer capabilities = %+v", summary.Capabilities.Installers)
	}
	if len(summary.Capabilities.LaunchTools) != 1 || summary.Capabilities.LaunchTools[0].ID != "loader" {
		t.Fatalf("launch tool capabilities = %+v", summary.Capabilities.LaunchTools)
	}
}

func TestNewExtensionRejectsUnsafeExtensionOutputs(t *testing.T) {
	_, err := NewExtension("bad", "Bad Game", func(r *Registrar) {
		r.RegisterGame(GameRegistration{
			SteamAppIDs:  []string{"200"},
			NexusDomains: []string{"badgame"},
			VortexGameID: "badgame",
		})
		r.RegisterModType(installplan.ModTypeSpec{ID: "mod", TargetRoot: "../outside"})
		r.RegisterInstaller(installplan.InstallerSpec{
			ID:                "bad:installer",
			VortexInstallerID: "bad-installer",
			ModType:           "missing-type",
			GeneratedFiles: []installplan.GeneratedFileSpec{{
				FromGameRelative: "/abs/source.json",
				Destination:      "ok.json",
			}},
		})
		r.RegisterLaunchTool(LaunchToolSpec{
			ID:                 "tool",
			Name:               "Tool",
			ExecutableRelative: "../tool",
		})
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	for _, want := range []string{
		"mod type mod target root: path traversal is not allowed",
		"references undeclared mod type missing-type",
		"generated source path: absolute path is not allowed",
		"launch tool tool executable path: path traversal is not allowed",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q did not contain %q", err.Error(), want)
		}
	}
}
