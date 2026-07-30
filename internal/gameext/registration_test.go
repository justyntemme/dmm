package gameext

import (
	"context"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestCompileExtensionRegistersVortexStyleDomains(t *testing.T) {
	extension, err := CompileExtension(sdk.Extension{
		ID:      "sample",
		Name:    "Sample Game",
		Version: "1.2.3",
		BuildID: "test-build",
		Register: func(r sdk.Registrar) {
			r.RegisterGame(sdk.GameRegistration{
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
			r.RegisterInstallerChoice(sdk.InstallerChoiceSpec{
				ID:         "sample:fomod",
				Name:       "FOMOD installer",
				Kind:       "fomod",
				ModType:    "mod",
				TargetRoot: "Mods",
			})
			r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
				ID:       "sample-loader",
				Name:     "Sample Loader",
				Kind:     "mod-loader",
				Required: true,
				ModTypes: []string{"mod"},
			})
			r.RegisterLaunchTool(sdk.LaunchToolSpec{
				ID:                 "loader",
				Name:               "Sample Loader",
				ExecutableRelative: "loader",
				RequiredFiles:      []string{"loader", "loader.dll"},
				DefaultPrimary:     true,
				ModTypes:           []string{"mod"},
				ProviderModTypes:   []string{"loader-mod"},
			})
			r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
				ID:   "sample-version",
				Name: "Sample version detector",
				Provider: func(_ context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
					if input.AppID != "100" || input.GamePath != "/games/sample" || input.SteamBuildID != "build-1" {
						t.Fatalf("version input = %+v", input)
					}
					return sdk.GameVersionResult{Version: "1.2.0", Source: "test"}, nil
				},
			})
			r.RegisterPluginActivation(sdk.PluginActivationSpec{
				ID:               "sample-plugins",
				Name:             "Sample plugins.txt",
				GameDataRoot:     "Data",
				AppDataPath:      "Sample Game",
				PluginsFile:      "plugins.txt",
				LoadOrderFile:    "loadorder.txt",
				Format:           "fallout4",
				PluginExtensions: []string{".esm", ".esp", ".esl"},
			})
			r.RegisterConflictIgnore(sdk.ConflictIgnoreSpec{
				ID:       "sample-ignore",
				Name:     "Sample ignored conflict",
				Patterns: []string{"**/ignored.txt"},
			})
			r.RegisterMerge(sdk.MergeSpec{ID: "merge", Name: "Merge"})
			r.RegisterLoadOrder(sdk.LoadOrderSpec{ID: "load-order", Name: "Load Order"})
			r.RegisterEventHandler(sdk.EventHandlerSpec{
				Event: "will-deploy",
				Name:  "Prepare",
				Handler: func(_ context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
					if input.AppID != "100" || input.Event != "will-deploy" {
						t.Fatalf("hook input = %+v", input)
					}
					return sdk.EventHandlerResult{
						Mappings: []deploy.FileMapping{{SourceRelative: "generated.txt", TargetRelative: "generated.txt"}},
						Messages: []string{"prepared"},
					}, nil
				},
			})
		},
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
	if summary.Version != "1.2.3" || summary.BuildID != "test-build" {
		t.Fatalf("summary version/build = %+v", summary)
	}
	if len(summary.Capabilities.Installers) != 1 || summary.Capabilities.Installers[0].ID != "sample:installer" {
		t.Fatalf("installer capabilities = %+v", summary.Capabilities.Installers)
	}
	if len(summary.Capabilities.InstallerChoices) != 1 || summary.Capabilities.InstallerChoices[0].ID != "sample:fomod" {
		t.Fatalf("installer choice capabilities = %+v", summary.Capabilities.InstallerChoices)
	}
	if choice, ok := registry.InstallerChoiceForSteamApp("100", "fomod"); !ok || choice.TargetRoot != "Mods" {
		t.Fatalf("installer choice lookup = %+v %v", choice, ok)
	}
	if len(summary.Capabilities.LaunchTools) != 1 || summary.Capabilities.LaunchTools[0].ID != "loader" {
		t.Fatalf("launch tool capabilities = %+v", summary.Capabilities.LaunchTools)
	}
	if len(summary.Capabilities.GameVersions) != 1 || summary.Capabilities.GameVersions[0].ID != "sample-version" {
		t.Fatalf("game version capabilities = %+v", summary.Capabilities.GameVersions)
	}
	version, ran, err := registry.DetectGameVersion(context.Background(), "100", sdk.GameVersionInput{
		GamePath:     "/games/sample",
		SteamBuildID: "build-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ran || version.Version != "1.2.0" || version.Source != "test" {
		t.Fatalf("detected version = %+v, ran = %v", version, ran)
	}
	if len(summary.Capabilities.PluginActivations) != 1 || summary.Capabilities.PluginActivations[0].ID != "sample-plugins" {
		t.Fatalf("plugin activation capabilities = %+v", summary.Capabilities.PluginActivations)
	}
	if len(summary.Capabilities.ConflictIgnores) != 1 || summary.Capabilities.ConflictIgnores[0].ID != "sample-ignore" {
		t.Fatalf("conflict ignore capabilities = %+v", summary.Capabilities.ConflictIgnores)
	}
	if patterns := registry.ConflictIgnorePatternsForSteamApp("100"); len(patterns) != 1 || patterns[0] != "**/ignored.txt" {
		t.Fatalf("conflict ignore patterns = %+v", patterns)
	}
	if !registry.HasEventHandlerForSteamApp("100", "will-deploy") || registry.HasEventHandlerForSteamApp("100", "did-deploy") {
		t.Fatalf("event handler predicates are wrong")
	}
	hookResult, err := registry.RunEventHandlers(context.Background(), "100", "will-deploy", sdk.EventHandlerInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(hookResult.Mappings) != 1 || hookResult.Mappings[0].TargetRelative != "generated.txt" || len(hookResult.Messages) != 1 {
		t.Fatalf("hook result = %+v", hookResult)
	}
}

func TestCompileExtensionRejectsUnsafeExtensionOutputs(t *testing.T) {
	_, err := CompileExtension(sdk.Extension{
		ID:   "bad",
		Name: "Bad Game",
		Register: func(r sdk.Registrar) {
			r.RegisterGame(sdk.GameRegistration{
				SteamAppIDs:  []string{"200"},
				NexusDomains: []string{"badgame"},
				VortexGameID: "badgame",
			})
			r.RegisterModType(installplan.ModTypeSpec{ID: "mod", TargetRoot: "../outside"})
			r.RegisterInstaller(installplan.InstallerSpec{
				ID:                "bad:installer",
				VortexInstallerID: "bad-installer",
				ModType:           "missing-type",
				InstructionMode:   installplan.InstructionCustom,
				GeneratedFiles: []installplan.GeneratedFileSpec{{
					FromGameRelative: "/abs/source.json",
					Destination:      "ok.json",
				}},
			})
			r.RegisterInstallerChoice(sdk.InstallerChoiceSpec{
				ID:          "bad:fomod",
				Kind:        "fomod",
				ModType:     "missing-choice-type",
				TargetRoot:  "../Data",
				StopFolders: []string{"bad/path"},
			})
			r.RegisterLaunchTool(sdk.LaunchToolSpec{
				ID:                 "tool",
				Name:               "Tool",
				ExecutableRelative: "../tool",
			})
			r.RegisterPluginActivation(sdk.PluginActivationSpec{
				ID:               "plugins",
				Name:             "Plugins",
				GameDataRoot:     "../Data",
				AppDataPath:      "Bad",
				Format:           "weird",
				PluginExtensions: []string{"esp"},
			})
			r.RegisterConflictIgnore(sdk.ConflictIgnoreSpec{
				ID:       "ignore",
				Name:     "Bad ignore",
				Patterns: []string{"/abs.txt", "../outside.txt"},
			})
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	for _, want := range []string{
		"extension version is required",
		"extension build id is required",
		"mod type mod target root: path traversal is not allowed",
		"installer bad:installer custom builder is required",
		"references undeclared mod type missing-type",
		"generated source path: absolute path is not allowed",
		"installer choice bad:fomod references undeclared mod type missing-choice-type",
		"installer choice bad:fomod target root: path traversal is not allowed",
		"installer choice bad:fomod stop folder: must be a single relative path segment",
		"launch tool tool executable path: path traversal is not allowed",
		"plugin activation plugins game data root: path traversal is not allowed",
		"plugin activation plugins format must be original or fallout4",
		"plugin activation plugins plugin extension must be a file extension",
		"conflict ignore ignore pattern: absolute patterns are not allowed",
		"conflict ignore ignore pattern: path traversal is not allowed",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q did not contain %q", err.Error(), want)
		}
	}
}
