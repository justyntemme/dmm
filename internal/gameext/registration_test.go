package gameext

import (
	"context"
	"os"
	"path/filepath"
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
			r.RegisterInstallPlatform(sdk.InstallPlatformSpec{
				ID:      "windows",
				Name:    "Windows/Proton",
				Markers: []string{"Game.exe"},
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
				Arguments:          []string{"--native"},
				RequiredFiles:      []string{"loader", "loader.dll"},
				Variants: []sdk.LaunchToolVariantSpec{{
					PlatformID:         "windows",
					ExecutableRelative: "loader.exe",
					Arguments:          []string{"--windows"},
					RequiredFiles:      []string{"loader.exe", "loader.dll"},
				}},
				DefaultPrimary:   true,
				ModTypes:         []string{"mod"},
				ProviderModTypes: []string{"loader-mod"},
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
				NativePluginManifests: []string{
					"Sample.ccc",
				},
			})
			r.RegisterConflictIgnore(sdk.ConflictIgnoreSpec{
				ID:       "sample-ignore",
				Name:     "Sample ignored conflict",
				Patterns: []string{"**/ignored.txt"},
			})
			r.RegisterDeployIgnore(sdk.DeployIgnoreSpec{
				ID:       "sample-deploy-ignore",
				Name:     "Sample ignored deploy",
				Patterns: []string{"**/readme*"},
			})
			r.RegisterSteamWorkshop(sdk.SteamWorkshopSpec{
				AllowCoexistence: true,
				Actions: []sdk.SteamWorkshopActionSpec{
					{ID: "sample-workshop-unsubscribe", Name: "Unsubscribe Workshop item", Kind: "unsubscribe"},
				},
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
	gamePath := t.TempDir()
	if err := os.WriteFile(filepath.Join(gamePath, "Game.exe"), []byte("exe"), 0o600); err != nil {
		t.Fatal(err)
	}
	if platform, ok := registry.InstallPlatformForSteamApp("100", gamePath); !ok || platform.ID != "windows" {
		t.Fatalf("platform = %+v ok=%v", platform, ok)
	}
	_, baseTool, _ := registry.RequiredPrimaryLaunchToolForSteamApp("100", []gamehandler.RuntimeMod{{Enabled: true, ModType: "mod"}})
	resolvedTool := registry.ResolveLaunchToolForSteamApp("100", gamePath, baseTool)
	if resolvedTool.ExecutableRelative != "loader.exe" || len(resolvedTool.Arguments) != 1 || resolvedTool.Arguments[0] != "--windows" || len(resolvedTool.RequiredFiles) != 2 || resolvedTool.RequiredFiles[0] != "loader.exe" {
		t.Fatalf("resolved launch tool = %+v", resolvedTool)
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
	if summary.Coverage != CoverageInstaller || summary.CoverageLabel != "Installer support" {
		t.Fatalf("summary coverage = %q/%q", summary.Coverage, summary.CoverageLabel)
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
	if len(summary.Capabilities.InstallPlatforms) != 1 || summary.Capabilities.InstallPlatforms[0].ID != "windows" {
		t.Fatalf("install platform capabilities = %+v", summary.Capabilities.InstallPlatforms)
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
	if len(summary.Capabilities.DeployIgnores) != 1 || summary.Capabilities.DeployIgnores[0].ID != "sample-deploy-ignore" {
		t.Fatalf("deploy ignore capabilities = %+v", summary.Capabilities.DeployIgnores)
	}
	if patterns := registry.DeployIgnorePatternsForSteamApp("100"); len(patterns) != 1 || patterns[0] != "**/readme*" {
		t.Fatalf("deploy ignore patterns = %+v", patterns)
	}
	workshop, ok := registry.SteamWorkshopForSteamApp("100")
	if !ok || !workshop.AllowCoexistence || len(workshop.Actions) != 1 || workshop.Actions[0].Kind != "unsubscribe" {
		t.Fatalf("workshop support = %+v ok=%v", workshop, ok)
	}
	if !registry.SteamWorkshopCoexistenceAllowed("100") {
		t.Fatal("workshop coexistence was not allowed")
	}
	if summary.Capabilities.SteamWorkshop == nil || !summary.Capabilities.SteamWorkshop.AllowCoexistence || len(summary.Capabilities.SteamWorkshop.Actions) != 1 {
		t.Fatalf("workshop capabilities = %+v", summary.Capabilities.SteamWorkshop)
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

func TestCompileExtensionAllowsWorkshopOnlyGame(t *testing.T) {
	extension, err := CompileExtension(sdk.Extension{
		ID:      "workshop-only",
		Name:    "Workshop Only",
		Version: "0.1.0",
		BuildID: "test-build",
		Register: func(r sdk.Registrar) {
			r.RegisterGame(sdk.GameRegistration{
				SteamAppIDs: []string{"108600"},
				Workshop: sdk.SteamWorkshopSpec{
					AllowCoexistence: true,
					Actions:          sdk.StandardSteamWorkshopActions(),
				},
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(extension.NexusDomains) != 0 {
		t.Fatalf("workshop-only extension should not invent Nexus domains: %+v", extension.NexusDomains)
	}
	coverage, label := ExtensionCoverage(extension)
	if coverage != CoverageWorkshopOnly || label != "Workshop only" {
		t.Fatalf("workshop coverage = %q/%q", coverage, label)
	}
	workshop, ok := NewRegistry([]Extension{extension}).SteamWorkshopForSteamApp("108600")
	if !ok || !workshop.AllowCoexistence || len(workshop.Actions) != 5 {
		t.Fatalf("workshop capability = %+v ok=%v", workshop, ok)
	}
}

func TestCompileExtensionAllowsVerifiedMetadataOnlyGame(t *testing.T) {
	extension, err := CompileExtension(sdk.Extension{
		ID:      "metadata-only",
		Name:    "Metadata Only",
		Version: "0.1.0",
		BuildID: "test-build",
		Register: func(r sdk.Registrar) {
			r.RegisterGame(sdk.GameRegistration{
				SteamAppIDs: []string{"26800"},
			})
			r.RegisterSource(sdk.SourceRef{
				Name: "Verified ModDB page",
				URL:  "https://www.moddb.com/games/braid/mods",
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(extension.NexusDomains) != 0 {
		t.Fatalf("metadata-only extension should not invent Nexus domains: %+v", extension.NexusDomains)
	}
	coverage, label := ExtensionCoverage(extension)
	if coverage != CoverageMetadataOnly || label != "Metadata only" {
		t.Fatalf("metadata coverage = %q/%q", coverage, label)
	}
	summary := NewRegistry([]Extension{extension}).ExtensionSummaries()[0]
	if summary.Coverage != CoverageMetadataOnly || len(summary.Sources) != 1 {
		t.Fatalf("metadata summary = %+v", summary)
	}
}

func TestExtensionCoverageReportsResearchBlockedInstallers(t *testing.T) {
	extension, err := CompileExtension(sdk.Extension{
		ID:      "research-game",
		Name:    "Research Game",
		Version: "0.1.0",
		BuildID: "test-build",
		Register: func(r sdk.Registrar) {
			r.RegisterGame(sdk.GameRegistration{
				SteamAppIDs:  []string{"200"},
				NexusDomains: []string{"researchgame"},
				VortexGameID: "researchgame",
				Deployment: installplan.DeploymentSpec{
					AllowNeedsReviewState: true,
				},
			})
			r.RegisterModType(installplan.ModTypeSpec{ID: "research-mod"})
			r.RegisterInstaller(installplan.InstallerSpec{
				ID:                "research:blocked",
				VortexInstallerID: "research-blocked",
				ModType:           "research-mod",
				InstructionMode:   installplan.InstructionUnsupported,
				UnsupportedReason: "source review required",
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	coverage, label := ExtensionCoverage(extension)
	if coverage != CoverageResearchBlocked || label != "Research needed" {
		t.Fatalf("coverage = %q/%q", coverage, label)
	}
	summaries := NewRegistry([]Extension{extension}).ExtensionSummaries()
	if len(summaries) != 1 || summaries[0].Coverage != CoverageResearchBlocked {
		t.Fatalf("summaries = %+v", summaries)
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
			r.RegisterInstallPlatform(sdk.InstallPlatformSpec{
				ID:      "bad/platform",
				Markers: []string{"../Game.exe"},
			})
			r.RegisterLaunchTool(sdk.LaunchToolSpec{
				ID:                 "tool",
				Name:               "Tool",
				ExecutableRelative: "../tool",
				Arguments:          []string{"bad\narg"},
				Variants: []sdk.LaunchToolVariantSpec{{
					PlatformID:         "bad/platform",
					ExecutableRelative: "../tool.exe",
					Arguments:          []string{"bad\rarg"},
				}},
			})
			r.RegisterPluginActivation(sdk.PluginActivationSpec{
				ID:               "plugins",
				Name:             "Plugins",
				GameDataRoot:     "../Data",
				AppDataPath:      "Bad",
				Format:           "weird",
				PluginExtensions: []string{"esp"},
				NativePluginManifests: []string{
					"/Bad.ccc",
				},
			})
			r.RegisterConflictIgnore(sdk.ConflictIgnoreSpec{
				ID:       "ignore",
				Name:     "Bad ignore",
				Patterns: []string{"/abs.txt", "../outside.txt"},
			})
			r.RegisterSteamWorkshop(sdk.SteamWorkshopSpec{
				Actions: []sdk.SteamWorkshopActionSpec{
					{ID: "bad-workshop", Kind: "delete"},
				},
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
		"install platform bad/platform id must be a simple identifier",
		"install platform bad/platform name is required",
		"install platform bad/platform marker: path traversal is not allowed",
		"launch tool tool executable path: path traversal is not allowed",
		"launch tool tool argument: must not contain control line breaks",
		"launch tool tool variant platform id must be a simple identifier",
		"launch tool tool variant executable path: path traversal is not allowed",
		"launch tool tool variant argument: must not contain control line breaks",
		"plugin activation plugins game data root: path traversal is not allowed",
		"plugin activation plugins format must be original or fallout4",
		"plugin activation plugins plugin extension must be a file extension",
		"plugin activation plugins native plugin manifest: absolute path is not allowed",
		"conflict ignore ignore pattern: absolute patterns are not allowed",
		"conflict ignore ignore pattern: path traversal is not allowed",
		"steam workshop action bad-workshop name is required",
		"steam workshop action bad-workshop kind must be subscribe, unsubscribe, enable, disable, or order",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q did not contain %q", err.Error(), want)
		}
	}
}
