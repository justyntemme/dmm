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
	variantShell := false
	variantDetach := false
	variantExclusive := false
	extension, err := CompileExtension(sdk.Extension{
		ID:      "sample",
		Name:    "Sample Game",
		Version: "1.2.3",
		BuildID: "test-build",
		Register: func(r sdk.Registrar) {
			r.RegisterGame(sdk.GameRegistration{
				SteamAppIDs:        []string{"100", "100"},
				NexusDomains:       []string{"samplegame", "samplegame"},
				VortexGameID:       "samplegame",
				ExecutableRelative: "Game.exe",
				RequiredFiles:      []string{"Game.exe", "Data/sample.dat"},
				QueryModPath:       "Mods",
				MergeMode:          sdk.GameMergeModeAll,
				RequiresCleanup:    true,
				StopPatterns:       []string{"(^|/)Data/.+"},
				CompatibleDownloads: []string{
					"sampleaddon",
				},
				Environment: map[string]string{"SteamAPPId": "100"},
			})
			r.RegisterModType(installplan.ModTypeSpec{ID: "mod", TargetRoot: "Mods"})
			r.RegisterInstaller(installplan.InstallerSpec{
				ID:                "sample:installer",
				VortexInstallerID: "sample-installer",
				ModType:           "mod",
				InstructionMode:   installplan.InstructionManifestFolders,
			})
			r.RegisterInstallerChoice(sdk.InstallerChoiceSpec{
				ID:                    "sample:fomod",
				Name:                  "FOMOD installer",
				Kind:                  "fomod",
				ModType:               "mod",
				TargetRoot:            "Mods",
				DestinationPrefixMode: sdk.InstallerChoiceDestinationPrefixModuleBaseName,
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
				DynamicInputs: []sdk.LaunchToolDynamicInputSpec{{
					ID:             "profile-packages",
					Name:           "Enabled profile packages",
					Kind:           sdk.LaunchToolDynamicInputEnabledModFileList,
					SourceModTypes: []string{"mod"},
					OutputRelative: "DMM/profile-packages.ini",
					ArgumentToken:  "--package-list={path}",
				}},
				Variants: []sdk.LaunchToolVariantSpec{{
					PlatformID:         "windows",
					ExecutableRelative: "loader.exe",
					Arguments:          []string{"--windows"},
					RequiredFiles:      []string{"loader.exe", "loader.dll"},
					Shell:              &variantShell,
					Detach:             &variantDetach,
					Exclusive:          &variantExclusive,
				}},
				Shell:            true,
				Detach:           true,
				Exclusive:        true,
				DefaultPrimary:   true,
				ModTypes:         []string{"mod"},
				ProviderModTypes: []string{"loader-mod"},
			})
			r.RegisterSupportedTool(sdk.SupportedToolSpec{
				ID:                 "sample-editor",
				Name:               "Sample Editor",
				ShortName:          "Editor",
				ExecutableRelative: "Tools/Editor.exe",
				Arguments:          []string{"--game", "{game_path}"},
				RequiredFiles:      []string{"Tools/Editor.exe"},
				Relative:           true,
				Exclusive:          true,
			})
			r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
				ID:       "sample-epic-launcher",
				Name:     "Epic Games launcher",
				Launcher: "epic",
				Store:    "epic",
				AppID:    "sample-epic-app",
				Parameters: []sdk.LauncherParameterSpec{{
					Name:  "appExecName",
					Value: "Game",
				}},
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
			r.RegisterGameInfoProvider(sdk.GameInfoProviderSpec{
				ID:           "sample-info",
				Name:         "Sample info",
				Tags:         []string{"game_version"},
				CacheSeconds: 60,
				Priority:     42,
				Provider: func(_ context.Context, input sdk.GameInfoInput) (sdk.GameInfoResult, error) {
					if input.AppID != "100" || input.GamePath != "/games/sample" || input.GameVersion != "1.2.0" {
						t.Fatalf("game info input = %+v", input)
					}
					return sdk.GameInfoResult{Details: []sdk.GameInfoDetail{{
						ID:     "game_version",
						Title:  "Installed Version",
						Value:  input.GameVersion,
						Source: "sample-info",
					}}}, nil
				},
			})
			r.RegisterPluginActivation(sdk.PluginActivationSpec{
				ID:               "sample-plugins",
				Name:             "Sample plugins.txt",
				GameDataRoot:     "Data",
				AppDataPath:      "Sample Game",
				PluginsFile:      "plugins.txt",
				LoadOrderFile:    "loadorder.txt",
				Format:           sdk.PluginActivationFormatAsterisked,
				PluginExtensions: []string{".esm", ".esp", ".esl"},
				NativePluginManifests: []string{
					"Sample.ccc",
				},
			})
			r.RegisterUnmanagedMarker(sdk.UnmanagedMarkerSpec{
				ID:       "sample-unmanaged",
				Name:     "Sample unmanaged files",
				Patterns: []string{"loader.exe", "Data/External/*.dll"},
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
			r.RegisterArchiveType(sdk.ArchiveTypeSpec{
				ID:             "ba2",
				Name:           "Bethesda BA2",
				FileExtensions: []string{"ba2"},
				Engine:         "gamebryo-archive-support",
				SupportsWrite:  true,
			})
			r.RegisterInterpreter(sdk.InterpreterSpec{
				ID:             "jar",
				Name:           "Java archive",
				FileExtensions: []string{".jar"},
				Command:        "java",
				Arguments:      []string{"-jar", "{path}"},
			})
			r.RegisterGameStore(sdk.GameStoreSpec{ID: "gog", Name: "GOG"})
			r.RegisterGameSetup(sdk.GameSetupSpec{
				ID:      "prepare",
				Name:    "Prepare for modding",
				Actions: append(sdk.RequireGamePaths("Game.exe"), sdk.EnsureGameFiles("ready\n", "Mods/.dmm-ready")...),
			})
			r.RegisterExtensionAction(sdk.ExtensionActionSpec{ID: "manage-rules", Name: "Manage Rules", Scope: "profile", Kind: "dialog"})
			r.RegisterExtensionSetting(sdk.ExtensionSettingSpec{ID: "rules", Name: "Rules", Scope: "game"})
			r.RegisterExtensionTest(sdk.ExtensionTestSpec{ID: "loader-missing", Name: "Loader missing", Trigger: "gamemode-activated"})
			r.RegisterExtensionToDo(sdk.ExtensionToDoSpec{ID: "archive-invalidation", Name: "Archive invalidation", Trigger: "gamemode-activated"})
			r.RegisterExtensionAPI(sdk.ExtensionAPISpec{ID: "lootSortAsync", Name: "LOOT sort"})
			r.RegisterProfileFeature(sdk.ProfileFeatureSpec{ID: "plugins", Name: "Plugins"})
			r.RegisterCollectionFeature(sdk.CollectionFeatureSpec{ID: "rules", Name: "Rules"})
			r.RegisterStateStore(sdk.StateStoreSpec{ID: "load-order", Name: "Load order", Scope: "profile"})
			r.RegisterStateMigration(sdk.StateMigrationSpec{
				ID:          "load-order-1",
				Name:        "Load order migration",
				FromVersion: "0.0.1",
				ToVersion:   "0.0.2",
				Commands: []sdk.StateMigrationCommandSpec{{
					ID:             "purge-old-mods",
					Name:           "Purge old managed mods",
					Command:        sdk.StateMigrationCommandPurgeModsInPath,
					ModType:        "mod",
					TargetRelative: "Mods/Legacy",
				}},
			})
			r.RegisterHistoryStack(sdk.HistoryStackSpec{ID: "plugins", Name: "Plugin history", Scope: "plugins"})
			r.RegisterHealthCheck(sdk.HealthCheckSpec{
				ID:   "sample-health",
				Name: "Sample health",
				CheckMod: func(_ context.Context, input sdk.ModHealthCheckInput) (sdk.HealthCheckResult, error) {
					return sdk.HealthCheckResult{
						InstalledModID: input.Mod.ID,
						Status:         sdk.HealthCheckStatusPassed,
						Severity:       sdk.HealthCheckSeverityInfo,
						Message:        "ok",
					}, nil
				},
			})
			r.RegisterAttributeExtractor(sdk.AttributeExtractorSpec{ID: "manifest", Name: "Manifest metadata", Target: "mods"})
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
	if appID, ok := registry.SteamAppIDForNexusDomain("sampleaddon"); ok || appID != "" {
		t.Fatalf("compatible download domain should not own reverse mapping = %q %v", appID, ok)
	}
	if domains := registry.NexusDomainsForSteamAppID("100"); !contains(domains, "samplegame") || !contains(domains, "sampleaddon") {
		t.Fatalf("nexus domains for app should include primary and compatible domains = %+v", domains)
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
	if resolvedTool.ExecutableRelative != "loader.exe" || len(resolvedTool.Arguments) != 1 || resolvedTool.Arguments[0] != "--windows" || len(resolvedTool.RequiredFiles) != 2 || resolvedTool.RequiredFiles[0] != "loader.exe" || resolvedTool.Shell || resolvedTool.Detach || resolvedTool.Exclusive {
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
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != "Mods" || summary.Capabilities.GameRegistration.MergeMode != sdk.GameMergeModeAll || !summary.Capabilities.GameRegistration.RequiresCleanup {
		t.Fatalf("game registration metadata = %+v", summary.Capabilities.GameRegistration)
	}
	if !contains(summary.Capabilities.GameRegistration.RequiredFiles, "Data/sample.dat") || !contains(summary.Capabilities.GameRegistration.CompatibleDownloads, "sampleaddon") {
		t.Fatalf("game registration lists = %+v", summary.Capabilities.GameRegistration)
	}
	if summary.Capabilities.GameRegistration.Environment["SteamAPPId"] != "100" {
		t.Fatalf("game environment = %+v", summary.Capabilities.GameRegistration.Environment)
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
	if len(summary.Capabilities.ModTypes) != 1 || summary.Capabilities.ModTypes[0].ID != "mod" || summary.Capabilities.ModTypes[0].DeploymentMode != installplan.ModTypeDeploymentDirect {
		t.Fatalf("mod type capabilities = %+v", summary.Capabilities.ModTypes)
	}
	if len(summary.Capabilities.InstallerChoices) != 1 || summary.Capabilities.InstallerChoices[0].ID != "sample:fomod" {
		t.Fatalf("installer choice capabilities = %+v", summary.Capabilities.InstallerChoices)
	}
	if choice, ok := registry.InstallerChoiceForSteamApp("100", "fomod"); !ok || choice.TargetRoot != "Mods" || choice.DestinationPrefixMode != sdk.InstallerChoiceDestinationPrefixModuleBaseName {
		t.Fatalf("installer choice lookup = %+v %v", choice, ok)
	}
	if len(summary.Capabilities.LaunchTools) != 1 || summary.Capabilities.LaunchTools[0].ID != "loader" {
		t.Fatalf("launch tool capabilities = %+v", summary.Capabilities.LaunchTools)
	}
	if len(summary.Capabilities.SupportedTools) != 1 || summary.Capabilities.SupportedTools[0].ID != "sample-editor" || !summary.Capabilities.SupportedTools[0].Relative || !summary.Capabilities.SupportedTools[0].Exclusive {
		t.Fatalf("supported tool capabilities = %+v", summary.Capabilities.SupportedTools)
	}
	if len(summary.Capabilities.LauncherRequirements) != 1 || summary.Capabilities.LauncherRequirements[0].Launcher != "epic" || len(summary.Capabilities.LauncherRequirements[0].Parameters) != 1 {
		t.Fatalf("launcher requirement capabilities = %+v", summary.Capabilities.LauncherRequirements)
	}
	if !summary.Capabilities.LaunchTools[0].Shell || !summary.Capabilities.LaunchTools[0].Detach || !summary.Capabilities.LaunchTools[0].Exclusive {
		t.Fatalf("launch tool flags = %+v", summary.Capabilities.LaunchTools[0])
	}
	dynamicInputs := summary.Capabilities.LaunchTools[0].DynamicInputs
	if len(dynamicInputs) != 1 || dynamicInputs[0].ID != "profile-packages" || dynamicInputs[0].Kind != sdk.LaunchToolDynamicInputEnabledModFileList || dynamicInputs[0].OutputRelative != "DMM/profile-packages.ini" || dynamicInputs[0].ArgumentToken != "--package-list={path}" {
		t.Fatalf("launch tool dynamic inputs = %+v", dynamicInputs)
	}
	if len(summary.Capabilities.InstallPlatforms) != 1 || summary.Capabilities.InstallPlatforms[0].ID != "windows" {
		t.Fatalf("install platform capabilities = %+v", summary.Capabilities.InstallPlatforms)
	}
	if len(summary.Capabilities.GameVersions) != 1 || summary.Capabilities.GameVersions[0].ID != "sample-version" {
		t.Fatalf("game version capabilities = %+v", summary.Capabilities.GameVersions)
	}
	if len(summary.Capabilities.GameInfoProviders) != 1 || summary.Capabilities.GameInfoProviders[0].ID != "sample-info" || summary.Capabilities.GameInfoProviders[0].Priority != 42 {
		t.Fatalf("game info capabilities = %+v", summary.Capabilities.GameInfoProviders)
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
	info, ran, err := registry.QueryGameInfo(context.Background(), "100", sdk.GameInfoInput{
		GamePath:     "/games/sample",
		SteamBuildID: "build-1",
		GameVersion:  version.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ran || len(info) != 1 || info[0].ID != "game_version" || info[0].Value != "1.2.0" {
		t.Fatalf("game info = %+v, ran = %v", info, ran)
	}
	if len(summary.Capabilities.PluginActivations) != 1 || summary.Capabilities.PluginActivations[0].ID != "sample-plugins" {
		t.Fatalf("plugin activation capabilities = %+v", summary.Capabilities.PluginActivations)
	}
	if len(summary.Capabilities.UnmanagedMarkers) != 1 || summary.Capabilities.UnmanagedMarkers[0].ID != "sample-unmanaged" || len(summary.Capabilities.UnmanagedMarkers[0].Patterns) != 2 {
		t.Fatalf("unmanaged marker capabilities = %+v", summary.Capabilities.UnmanagedMarkers)
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
	if len(summary.Capabilities.ArchiveTypes) != 1 || summary.Capabilities.ArchiveTypes[0].ID != "ba2" || !summary.Capabilities.ArchiveTypes[0].SupportsWrite || summary.Capabilities.ArchiveTypes[0].Status != sdk.CapabilityStatusReady {
		t.Fatalf("archive type capabilities = %+v", summary.Capabilities.ArchiveTypes)
	}
	if len(summary.Capabilities.Interpreters) != 1 || summary.Capabilities.Interpreters[0].ID != "jar" || summary.Capabilities.Interpreters[0].Command != "java" {
		t.Fatalf("interpreter capabilities = %+v", summary.Capabilities.Interpreters)
	}
	if len(summary.Capabilities.GameStores) != 1 || summary.Capabilities.GameStores[0].ID != "gog" {
		t.Fatalf("game store capabilities = %+v", summary.Capabilities.GameStores)
	}
	if len(summary.Capabilities.GameSetups) != 1 || summary.Capabilities.GameSetups[0].ID != "prepare" || len(summary.Capabilities.GameSetups[0].SetupActions) != 2 {
		t.Fatalf("game setup capabilities = %+v", summary.Capabilities.GameSetups)
	}
	if len(summary.Capabilities.ExtensionActions) != 1 || summary.Capabilities.ExtensionActions[0].Kind != "dialog" {
		t.Fatalf("extension action capabilities = %+v", summary.Capabilities.ExtensionActions)
	}
	if len(summary.Capabilities.ExtensionSettings) != 1 || summary.Capabilities.ExtensionSettings[0].Scope != "game" {
		t.Fatalf("extension setting capabilities = %+v", summary.Capabilities.ExtensionSettings)
	}
	if len(summary.Capabilities.ExtensionTests) != 1 || summary.Capabilities.ExtensionTests[0].Trigger != "gamemode-activated" {
		t.Fatalf("extension test capabilities = %+v", summary.Capabilities.ExtensionTests)
	}
	if len(summary.Capabilities.ExtensionToDos) != 1 || summary.Capabilities.ExtensionToDos[0].ID != "archive-invalidation" {
		t.Fatalf("extension todo capabilities = %+v", summary.Capabilities.ExtensionToDos)
	}
	if len(summary.Capabilities.ExtensionAPIs) != 1 || summary.Capabilities.ExtensionAPIs[0].ID != "lootSortAsync" {
		t.Fatalf("extension api capabilities = %+v", summary.Capabilities.ExtensionAPIs)
	}
	if len(summary.Capabilities.ProfileFeatures) != 1 || summary.Capabilities.ProfileFeatures[0].ID != "plugins" {
		t.Fatalf("profile feature capabilities = %+v", summary.Capabilities.ProfileFeatures)
	}
	if len(summary.Capabilities.CollectionFeatures) != 1 || summary.Capabilities.CollectionFeatures[0].ID != "rules" {
		t.Fatalf("collection feature capabilities = %+v", summary.Capabilities.CollectionFeatures)
	}
	if len(summary.Capabilities.StateStores) != 1 || summary.Capabilities.StateStores[0].Scope != "profile" {
		t.Fatalf("state store capabilities = %+v", summary.Capabilities.StateStores)
	}
	if len(summary.Capabilities.StateMigrations) != 1 || summary.Capabilities.StateMigrations[0].FromVersion != "0.0.1" ||
		len(summary.Capabilities.StateMigrations[0].Commands) != 1 || summary.Capabilities.StateMigrations[0].Commands[0].Command != sdk.StateMigrationCommandPurgeModsInPath {
		t.Fatalf("state migration capabilities = %+v", summary.Capabilities.StateMigrations)
	}
	if len(summary.Capabilities.HistoryStacks) != 1 || summary.Capabilities.HistoryStacks[0].ID != "plugins" {
		t.Fatalf("history stack capabilities = %+v", summary.Capabilities.HistoryStacks)
	}
	if len(summary.Capabilities.HealthChecks) != 1 || summary.Capabilities.HealthChecks[0].ID != "sample-health" {
		t.Fatalf("health check capabilities = %+v", summary.Capabilities.HealthChecks)
	}
	if len(summary.Capabilities.AttributeExtractors) != 1 || summary.Capabilities.AttributeExtractors[0].Target != "mods" {
		t.Fatalf("attribute extractor capabilities = %+v", summary.Capabilities.AttributeExtractors)
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

func TestRegistryBuildInstallPlanUsesNexusDomainForSharedSteamApp(t *testing.T) {
	extensions := []Extension{
		MustCompileExtension(sdk.Extension{
			ID:      "shared-original",
			Name:    "Shared Original",
			Version: "1.0.0",
			BuildID: "test",
			Register: func(r sdk.Registrar) {
				r.RegisterGame(sdk.GameRegistration{
					SteamAppIDs:  []string{"777"},
					NexusDomains: []string{"sharedoriginal"},
					VortexGameID: "sharedoriginal",
				})
				r.RegisterTargetRoot(sdk.TargetRootSpec{
					ID:       "original-root",
					Name:     "Original Root",
					Resolver: staticTestTargetRoot,
				})
				r.RegisterModType(installplan.ModTypeSpec{ID: "original-mod", TargetRootID: "original-root"})
				r.RegisterInstaller(installplan.InstallerSpec{
					ID:                "original-installer",
					VortexInstallerID: "game-query-mod-path",
					Priority:          100,
					ModType:           "original-mod",
					TargetRootID:      "original-root",
					StripCommonRoot:   true,
					InstructionMode:   installplan.InstructionArchiveRoot,
				})
			},
		}),
		MustCompileExtension(sdk.Extension{
			ID:      "shared-definitive",
			Name:    "Shared Definitive",
			Version: "1.0.0",
			BuildID: "test",
			Register: func(r sdk.Registrar) {
				r.RegisterGame(sdk.GameRegistration{
					SteamAppIDs:  []string{"777"},
					NexusDomains: []string{"shareddefinitive"},
					VortexGameID: "shareddefinitive",
				})
				r.RegisterTargetRoot(sdk.TargetRootSpec{
					ID:       "definitive-root",
					Name:     "Definitive Root",
					Resolver: staticTestTargetRoot,
				})
				r.RegisterModType(installplan.ModTypeSpec{ID: "definitive-mod", TargetRootID: "definitive-root"})
				r.RegisterInstaller(installplan.InstallerSpec{
					ID:                "definitive-installer",
					VortexInstallerID: "game-query-mod-path",
					Priority:          100,
					ModType:           "definitive-mod",
					TargetRootID:      "definitive-root",
					StripCommonRoot:   true,
					InstructionMode:   installplan.InstructionArchiveRoot,
				})
			},
		}),
	}
	registry := NewRegistry(extensions)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "mod.pak"), []byte("pak"), 0o600); err != nil {
		t.Fatal(err)
	}
	original, err := registry.BuildInstallPlanForNexusDomainWithGamePathArchiveAndSelections("777", "sharedoriginal", root, "", "mod.zip", nil)
	if err != nil {
		t.Fatal(err)
	}
	if original.ModType != "original-mod" || len(original.Instructions) != 1 || original.Instructions[0].TargetRoot != "original-root" {
		t.Fatalf("original plan = %+v", original)
	}
	definitive, err := registry.BuildInstallPlanForNexusDomainWithGamePathArchiveAndSelections("777", "shareddefinitive", root, "", "mod.zip", nil)
	if err != nil {
		t.Fatal(err)
	}
	if definitive.ModType != "definitive-mod" || len(definitive.Instructions) != 1 || definitive.Instructions[0].TargetRoot != "definitive-root" {
		t.Fatalf("definitive plan = %+v", definitive)
	}
}

func staticTestTargetRoot(ctx context.Context, input sdk.TargetRootInput) (sdk.TargetRootResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.TargetRootResult{}, err
	}
	return sdk.TargetRootResult{Path: filepath.Join(os.TempDir(), "dmm-test-target-root"), Source: "test"}, nil
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

func TestCompileExtensionAllowsFrameworkExtension(t *testing.T) {
	extension, err := CompileExtension(sdk.Extension{
		ID:      "common-interpreters",
		Name:    "Common Interpreters",
		Kind:    sdk.ExtensionKindFramework,
		Version: "0.1.0",
		BuildID: "test-build",
		Register: func(r sdk.Registrar) {
			r.RegisterInterpreter(sdk.InterpreterSpec{
				ID:             "python",
				Name:           "Python",
				FileExtensions: []string{".py"},
				Command:        "python",
				Arguments:      []string{"{path}", "--verbose"},
				Platforms:      []string{"linux"},
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if extension.Kind != ExtensionKindFramework || len(extension.SteamAppIDs) != 0 {
		t.Fatalf("framework extension = %+v", extension)
	}
	summary := NewRegistry([]Extension{extension}).ExtensionSummaries()[0]
	if summary.Kind != ExtensionKindFramework || summary.Coverage != CoverageFramework || summary.CoverageLabel != "Framework capability" {
		t.Fatalf("framework summary = %+v", summary)
	}
	if len(summary.Capabilities.Interpreters) != 1 || summary.Capabilities.Interpreters[0].ID != "python" {
		t.Fatalf("framework interpreter = %+v", summary.Capabilities.Interpreters)
	}
	resolved, ok := NewRegistry([]Extension{extension}).ResolveInterpreter("/tmp/install.py", "linux")
	if !ok || resolved.Command != "python" || len(resolved.Arguments) != 2 || resolved.Arguments[0] != "/tmp/install.py" {
		t.Fatalf("resolved interpreter = %+v ok=%v", resolved, ok)
	}
	if _, ok := NewRegistry([]Extension{extension}).ResolveInterpreter("/tmp/install.py", "windows"); ok {
		t.Fatal("linux-only interpreter matched windows")
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
	if len(summaries[0].Capabilities.Installers) != 0 || len(summaries[0].Capabilities.UnsupportedInstallers) != 1 {
		t.Fatalf("research-blocked installer capabilities = %+v", summaries[0].Capabilities)
	}
}

func TestBuildInstallPlanPassesGamePathToCustomInstaller(t *testing.T) {
	gamePath := t.TempDir()
	extractRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(extractRoot, "mod.txt"), []byte("mod"), 0o600); err != nil {
		t.Fatal(err)
	}

	extension, err := CompileExtension(sdk.Extension{
		ID:      "custom-path",
		Name:    "Custom Path",
		Version: "1.0.0",
		BuildID: "test-build",
		Register: func(r sdk.Registrar) {
			r.RegisterGame(sdk.GameRegistration{
				SteamAppIDs:        []string{"777777"},
				NexusDomains:       []string{"custompath"},
				VortexGameID:       "custom-path",
				ExecutableRelative: "bin/custom.exe",
				QueryModPath:       "Mods",
			})
			r.RegisterModType(installplan.ModTypeSpec{ID: "custom-path-mod", TargetRoot: "Mods"})
			r.RegisterInstaller(installplan.InstallerSpec{
				ID:                "custom-path-installer",
				VortexInstallerID: "custom-path",
				ModType:           "custom-path-mod",
				InstructionMode:   installplan.InstructionCustom,
				CustomBuild: func(input installplan.BuildInput) (installplan.Plan, error) {
					if input.GamePath != gamePath {
						t.Fatalf("game path = %q, want %q", input.GamePath, gamePath)
					}
					if input.ExecutableRelative != "bin/custom.exe" {
						t.Fatalf("executable relative = %q", input.ExecutableRelative)
					}
					return installplan.Plan{
						GameID:    input.GameID,
						ModType:   input.Installer.ModType,
						PlannerID: input.Installer.ID,
						Instructions: []installplan.Instruction{{
							Kind:            installplan.InstructionKindCopy,
							SourcePath:      filepath.Join(input.ExtractedRoot, "mod.txt"),
							StagingRelative: "mod.txt",
							TargetRelative:  "Mods/mod.txt",
						}},
					}, nil
				},
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := NewRegistry([]Extension{extension}).BuildInstallPlanWithGamePath("777777", extractRoot, gamePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Instructions) != 1 {
		t.Fatalf("plan = %+v", plan)
	}
}

func TestCompileExtensionRejectsUnsafeExtensionOutputs(t *testing.T) {
	_, err := CompileExtension(sdk.Extension{
		ID:   "bad",
		Name: "Bad Game",
		Register: func(r sdk.Registrar) {
			r.RegisterGame(sdk.GameRegistration{
				SteamAppIDs:        []string{"200"},
				NexusDomains:       []string{"badgame"},
				VortexGameID:       "badgame",
				ExecutableRelative: "../Bad.exe",
				RequiredFiles:      []string{"/Bad.dat"},
				QueryModPath:       "bad\npath",
				MergeMode:          "maybe",
				StopPatterns:       []string{""},
				CompatibleDownloads: []string{
					"bad\ndomain",
				},
				Environment: map[string]string{"bad\nkey": "value"},
			})
			r.RegisterModType(installplan.ModTypeSpec{ID: "mod", TargetRoot: "../outside", DeploymentMode: "magic"})
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
				ID:                    "bad:fomod",
				Kind:                  "fomod",
				ModType:               "missing-choice-type",
				TargetRoot:            "../Data",
				StopFolders:           []string{"bad/path"},
				DestinationPrefixMode: "bad-mode",
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
				DynamicInputs: []sdk.LaunchToolDynamicInputSpec{{
					ID:             "bad/id",
					Kind:           "texmod",
					SourceModTypes: []string{""},
					OutputRelative: "../bad.ini",
					ArgumentToken:  "bad\narg",
				}},
				Variants: []sdk.LaunchToolVariantSpec{{
					PlatformID:         "bad/platform",
					ExecutableRelative: "../tool.exe",
					Arguments:          []string{"bad\rarg"},
				}},
			})
			r.RegisterSupportedTool(sdk.SupportedToolSpec{
				ID:                 "bad/tool",
				Name:               "Bad Tool",
				ExecutableRelative: "../tool.exe",
				Arguments:          []string{"bad\narg"},
				Environment:        map[string]string{"bad\nkey": "value"},
				RequiredFiles:      []string{"/bad.exe"},
			})
			r.RegisterSupportedTool(sdk.SupportedToolSpec{
				ID:     "no-exe",
				Name:   "No Executable",
				Status: sdk.CapabilityStatusReady,
			})
			r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
				ID:       "bad/launcher",
				Name:     "Bad Launcher",
				Launcher: "",
				Store:    "bad\nstore",
				AppID:    "bad\napp",
				Parameters: []sdk.LauncherParameterSpec{{
					Value: "missing-name",
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
			r.RegisterUnmanagedMarker(sdk.UnmanagedMarkerSpec{
				ID:       "bad/marker",
				Patterns: []string{"/abs", "../bad"},
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
			r.RegisterStateMigration(sdk.StateMigrationSpec{
				ID:          "bad-migration",
				Name:        "Bad migration",
				FromVersion: "0.0.0",
				ToVersion:   "1.0.0",
				Commands: []sdk.StateMigrationCommandSpec{{
					ID:             "bad/command",
					Name:           "Bad command",
					Command:        "magic",
					SteamAppID:     "bad\napp",
					ModType:        "bad/type",
					TargetRootID:   "missing-root",
					TargetRelative: "../outside",
				}},
			})
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	for _, want := range []string{
		"extension version is required",
		"extension build id is required",
		"game executable path: path traversal is not allowed",
		"game required file: absolute path is not allowed",
		"game query mod path must not contain control line breaks",
		"game merge mode must be none, all, or dynamic",
		"game stop pattern is required",
		"game compatible download domain must not contain control line breaks",
		"game environment entries must not contain control line breaks",
		"mod type mod target root: path traversal is not allowed",
		"mod type mod deployment mode must be direct, event-hook, or tool-only",
		"installer bad:installer custom builder is required",
		"references undeclared mod type missing-type",
		"generated source path: absolute path is not allowed",
		"installer choice bad:fomod references undeclared mod type missing-choice-type",
		"installer choice bad:fomod target root: path traversal is not allowed",
		"installer choice bad:fomod stop folder: must be a single relative path segment",
		"installer choice bad:fomod destination prefix mode: unsupported value bad-mode",
		"install platform bad/platform id must be a simple identifier",
		"install platform bad/platform name is required",
		"install platform bad/platform marker: path traversal is not allowed",
		"launch tool tool executable path: path traversal is not allowed",
		"launch tool tool argument: must not contain control line breaks",
		"launch tool tool dynamic input bad/id id must be a simple identifier",
		"launch tool tool dynamic input bad/id name is required",
		"launch tool tool dynamic input bad/id kind must be generated-config or enabled-mod-file-list",
		"launch tool tool dynamic input bad/id source mod type is required",
		"launch tool tool dynamic input bad/id output path: path traversal is not allowed",
		"launch tool tool dynamic input bad/id argument token: must not contain control line breaks",
		"launch tool tool variant platform id must be a simple identifier",
		"launch tool tool variant executable path: path traversal is not allowed",
		"launch tool tool variant argument: must not contain control line breaks",
		"supported tool bad/tool id must be a simple identifier",
		"supported tool bad/tool executable path: path traversal is not allowed",
		"supported tool bad/tool argument: must not contain control line breaks",
		"supported tool bad/tool required file: absolute path is not allowed",
		"supported tool bad/tool environment entries must not contain control line breaks",
		"supported tool no-exe executable path is required",
		"launcher requirement bad/launcher id must be a simple identifier",
		"launcher requirement bad/launcher launcher is required",
		"launcher requirement bad/launcher store must not contain control line breaks",
		"launcher requirement bad/launcher app id must not contain control line breaks",
		"launcher requirement bad/launcher parameter name is required",
		"plugin activation plugins game data root: path traversal is not allowed",
		"plugin activation plugins format must be original or asterisked",
		"plugin activation plugins plugin extension must be a file extension",
		"plugin activation plugins native plugin manifest: absolute path is not allowed",
		"unmanaged marker bad/marker id must be a simple identifier",
		"unmanaged marker bad/marker name is required",
		"unmanaged marker bad/marker pattern: absolute patterns are not allowed",
		"unmanaged marker bad/marker pattern: path traversal is not allowed",
		"conflict ignore ignore pattern: absolute patterns are not allowed",
		"conflict ignore ignore pattern: path traversal is not allowed",
		"steam workshop action bad-workshop name is required",
		"steam workshop action bad-workshop kind must be subscribe, unsubscribe, enable, disable, or order",
		"state migration bad-migration command bad/command id must be a simple identifier",
		"state migration bad-migration command bad/command has unsupported command magic",
		"state migration bad-migration command bad/command steam app id must be a simple identifier",
		"state migration bad-migration command bad/command mod type must be a simple identifier",
		"state migration bad-migration command bad/command references undeclared target root missing-root",
		"state migration bad-migration command bad/command target relative path: path traversal is not allowed",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q did not contain %q", err.Error(), want)
		}
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
