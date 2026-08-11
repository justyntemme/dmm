package stardewvalley

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
	"github.com/justyntemme/decky-mod-manager/internal/steam"
)

const (
	SteamAppID          = "413150"
	VortexGameID        = "stardewvalley"
	Name                = "Stardew Valley"
	SettingMergeConfigs = "stardew_merge_configs"

	ModsRelativePath       = "Mods"
	SMAPIExecutable        = "StardewModdingAPI"
	SMAPIWindowsExecutable = "StardewModdingAPI.exe"

	MetadataKindSMAPIManifest = "smapi-manifest"

	platformLinux   = "linux"
	platformWindows = "windows"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Version:  "0.1.0",
		BuildID:  "first-party-go",
		Register: Register,
	}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:  []string{SteamAppID},
		NexusDomains: []string{VortexGameID},
		VortexGameID: VortexGameID,
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
		Workshop: sdk.SteamWorkshopSpec{
			AllowCoexistence: true,
			Actions:          sdk.StandardSteamWorkshopActions(),
		},
	})
	for _, modType := range modTypes() {
		r.RegisterModType(modType)
	}
	for _, platform := range installPlatforms() {
		r.RegisterInstallPlatform(platform)
	}
	for _, installer := range installers() {
		r.RegisterInstaller(installer)
	}
	r.RegisterInstallerChoice(sdk.InstallerChoiceSpec{
		ID:                    "vortex:stardewvalley:fomod",
		Name:                  "FOMOD installer",
		Kind:                  "fomod",
		ModType:               "stardew-smapi-mod",
		TargetRoot:            ModsRelativePath,
		DestinationPrefixMode: sdk.InstallerChoiceDestinationPrefixModuleBaseName,
	})
	for _, requirement := range runtimeRequirements() {
		r.RegisterRuntimeRequirement(requirement)
	}
	r.RegisterRuntimeMetadataDependencies(sdk.RuntimeDependencySpec{
		MetadataKinds:       []string{MetadataKindSMAPIManifest},
		RequirementIDPrefix: "stardew-mod-dependency:",
		RequirementKind:     "mod-dependency",
		RequirementMessage:  "Recommended Stardew mod dependency is not enabled in this profile.",
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "smapi",
		Name:               "SMAPI",
		ExecutableRelative: SMAPIExecutable,
		RequiredFiles: []string{
			SMAPIExecutable,
			"StardewModdingAPI.dll",
			filepath.ToSlash(filepath.Join("smapi-internal", "SMAPI.Toolkit.CoreInterfaces.dll")),
		},
		Variants: []sdk.LaunchToolVariantSpec{
			{
				PlatformID:         platformLinux,
				ExecutableRelative: SMAPIExecutable,
				RequiredFiles: []string{
					SMAPIExecutable,
					"StardewModdingAPI.dll",
					filepath.ToSlash(filepath.Join("smapi-internal", "SMAPI.Toolkit.CoreInterfaces.dll")),
				},
			},
			{
				PlatformID:         platformWindows,
				ExecutableRelative: SMAPIWindowsExecutable,
				RequiredFiles: []string{
					SMAPIWindowsExecutable,
					"StardewModdingAPI.dll",
					filepath.ToSlash(filepath.Join("smapi-internal", "SMAPI.Toolkit.CoreInterfaces.dll")),
				},
			},
		},
		DefaultPrimary: true,
		ModTypes:       []string{"stardew-smapi-mod"},
		ProviderModTypes: []string{
			"SMAPI",
		},
	})
	r.RegisterExtensionSetting(sdk.ExtensionSettingSpec{
		ID:           SettingMergeConfigs,
		Name:         "Preserve generated SMAPI config files",
		Scope:        "profile",
		ValueType:    sdk.ExtensionSettingValueBool,
		DefaultValue: json.RawMessage("true"),
		Message:      "Mirrors Vortex's Stardew merge-configs profile setting. When enabled, DMM adopts generated SMAPI config.json files into profile-owned staging and restores them when mods are re-enabled.",
	})
	r.RegisterExtensionAction(sdk.ExtensionActionSpec{
		ID:      "stardew-smapi-log",
		Name:    "Open SMAPI Log",
		Scope:   "diagnostics",
		Kind:    sdk.ExtensionActionKindOpenPath,
		Status:  sdk.CapabilityStatusReady,
		Message: "Mirrors Vortex's Stardew SMAPI Log action by opening the latest SMAPI log from the user config ErrorLogs folder.",
		OpenPath: &sdk.OpenPathActionSpec{
			Base:             sdk.OpenDirectoryBaseUserConfig,
			RelativePath:     "StardewValley/ErrorLogs/SMAPI-crash.txt",
			FallbackBase:     sdk.OpenDirectoryBaseUserConfig,
			FallbackRelative: "StardewValley/ErrorLogs",
		},
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   "will-deploy",
		Name:    "Preserve generated SMAPI config files",
		Handler: willDeployPreserveConfigs,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		ID:      "stardew-smapi-compatibility",
		Event:   sdk.EventCheckModsVersion,
		Name:    "Check SMAPI.io compatibility",
		Handler: checkSMAPICompatibility,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func installPlatforms() []sdk.InstallPlatformSpec {
	return []sdk.InstallPlatformSpec{
		{
			ID:      platformWindows,
			Name:    "Windows/Proton",
			Markers: []string{"Stardew Valley.exe"},
		},
		{
			ID:      platformLinux,
			Name:    "Native Linux",
			Markers: []string{"StardewValley"},
		},
	}
}

func modTypes() []installplan.ModTypeSpec {
	return []installplan.ModTypeSpec{
		{ID: "SMAPI", TargetRoot: ""},
		{ID: "sdvrootfolder", TargetRoot: ""},
		{ID: "stardew-smapi-mod", TargetRoot: ModsRelativePath},
	}
}

func installers() []installplan.InstallerSpec {
	return []installplan.InstallerSpec{
		smapiInstaller(platformLinux, []string{"linux-install.dat", "install.dat"}, []string{"internal", "linux"}, []string{SMAPIExecutable, "unix-launcher.sh"}),
		smapiInstaller(platformWindows, []string{"windows-install.dat", "install.dat"}, []string{"internal", "windows"}, []string{SMAPIWindowsExecutable, "StardewModdingAPI.exe.config"}),
		{
			ID:                "vortex:stardewvalley:sdvrootfolder",
			VortexInstallerID: "sdvrootfolder",
			Priority:          50,
			ModType:           "sdvrootfolder",
			Match: installplan.MatchSpec{
				RequireTopLevelDirs: []string{"Content"},
			},
			InstructionMode: installplan.InstructionRootFolder,
		},
		{
			ID:                "vortex:stardewvalley:stardew-valley-installer",
			VortexInstallerID: "stardew-valley-installer",
			Priority:          50,
			ModType:           "stardew-smapi-mod",
			NameSource:        installplan.NameSourceManifestDisplay,
			Match: installplan.MatchSpec{
				ManifestFileName:      "manifest.json",
				ExcludeLocaleManifest: true,
				ExcludeTopLevelDirs:   []string{"Content"},
			},
			MetadataExtractors: []installplan.MetadataExtractorSpec{
				smapiManifestExtractor(),
			},
			ComponentChoices: &installplan.ComponentChoiceSpec{
				Kind:       "component-choice",
				Name:       "Stardew Valley Component Selection",
				Reason:     "This Stardew Valley archive contains multiple SMAPI components; choose which components DMM should install.",
				StepID:     "stardew-component-selection",
				StepName:   "Choose Components",
				GroupID:    "stardew-smapi-components",
				GroupName:  "SMAPI components",
				GroupType:  "SelectAtLeastOne",
				DefaultAll: true,
			},
			InstructionMode: installplan.InstructionManifestFolders,
		},
		{
			ID:                "vortex:stardewvalley:generic-mods-folder",
			VortexInstallerID: "generic-stardew-mods-folder",
			Priority:          90,
			ModType:           "stardew-smapi-mod",
			NameSource:        installplan.NameSourceArchive,
			TargetRoot:        ModsRelativePath,
			StripCommonRoot:   true,
			CustomMatch:       stardewGenericModsFolderMatch,
			InstructionMode:   installplan.InstructionArchiveRoot,
		},
		{
			ID:                "vortex:stardewvalley:generic-mods-root",
			VortexInstallerID: "generic-stardew-mods-root",
			Priority:          100,
			ModType:           "stardew-smapi-mod",
			NameSource:        installplan.NameSourceArchive,
			TargetRoot:        ModsRelativePath,
			CustomMatch:       stardewGenericModsRootMatch,
			InstructionMode:   installplan.InstructionArchiveRoot,
		},
	}
}

func stardewGenericModsFolderMatch(extractedRoot string) bool {
	if !stardewArchiveHasDeployableFile(extractedRoot) {
		return false
	}
	if stardewArchiveHasInstallerToolScript(extractedRoot) || stardewArchiveHasFileBasename(extractedRoot, "smapi.installer.dll") {
		return false
	}
	return stardewArchiveHasTopLevelDir(extractedRoot, ModsRelativePath) && !stardewArchiveHasTopLevelDir(extractedRoot, "Content")
}

func stardewGenericModsRootMatch(extractedRoot string) bool {
	if !stardewArchiveHasDeployableFile(extractedRoot) {
		return false
	}
	if stardewArchiveHasInstallerToolScript(extractedRoot) || stardewArchiveHasFileBasename(extractedRoot, "smapi.installer.dll") {
		return false
	}
	if stardewArchiveHasTopLevelDir(extractedRoot, ModsRelativePath) || stardewArchiveHasTopLevelDir(extractedRoot, "Content") {
		return false
	}
	return true
}

func stardewArchiveHasDeployableFile(root string) bool {
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || found || d.IsDir() {
			return nil
		}
		found = true
		return nil
	})
	return found
}

func stardewArchiveHasTopLevelDir(root, name string) bool {
	name = strings.ToLower(strings.TrimSpace(filepath.ToSlash(name)))
	if name == "" {
		return false
	}
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == "." {
			return nil
		}
		first := strings.Split(filepath.ToSlash(rel), "/")[0]
		if strings.EqualFold(first, name) {
			found = true
		}
		if d.IsDir() && strings.Count(filepath.ToSlash(rel), "/") >= 1 {
			return filepath.SkipDir
		}
		return nil
	})
	return found
}

func stardewArchiveHasFileBasename(root, basename string) bool {
	basename = strings.ToLower(strings.TrimSpace(basename))
	if basename == "" {
		return false
	}
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || found || d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Base(path), basename) {
			found = true
		}
		return nil
	})
	return found
}

func stardewArchiveHasInstallerToolScript(root string) bool {
	installerNames := map[string]struct{}{
		"install on linux.sh":      {},
		"install on mac.command":   {},
		"install on macos.command": {},
		"install on windows.bat":   {},
	}
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || found || d.IsDir() {
			return nil
		}
		_, found = installerNames[strings.ToLower(filepath.Base(path))]
		return nil
	})
	return found
}

func smapiInstaller(platformID string, payloadFiles, payloadSegments, platformCopyTargets []string) installplan.InstallerSpec {
	policies := []installplan.TargetPolicySpec{
		{
			TargetRelative: "steam_appid.txt",
			Policy:         installplan.TargetPolicyKeepExisting,
		},
	}
	for _, target := range append([]string{
		"StardewModdingAPI.dll",
		"StardewModdingAPI.deps.json",
		"StardewModdingAPI.runtimeconfig.json",
		"StardewModdingAPI.xml",
	}, platformCopyTargets...) {
		policies = append(policies, installplan.TargetPolicySpec{
			TargetRelative: target,
			DeployStrategy: installplan.DeployStrategyCopy,
		})
	}
	return installplan.InstallerSpec{
		ID:                "vortex:stardewvalley:smapi-installer:" + platformID,
		VortexInstallerID: "smapi-installer",
		PlatformID:        platformID,
		Priority:          30,
		ModType:           "SMAPI",
		NameSource:        installplan.NameSourceArchive,
		Match: installplan.MatchSpec{
			FileBasenames: []string{"smapi.installer.dll"},
		},
		Payload: installplan.PayloadSpec{
			FileBasenames: payloadFiles,
			PathSegments:  payloadSegments,
		},
		GeneratedFiles: []installplan.GeneratedFileSpec{
			{
				FromGameRelative: "Stardew Valley.deps.json",
				Destination:      "StardewModdingAPI.deps.json",
			},
		},
		TargetPolicies: policies,
		MetadataExtractors: []installplan.MetadataExtractorSpec{
			smapiManifestExtractor(),
		},
		InstructionMode: installplan.InstructionEmbeddedZip,
	}
}

func runtimeRequirements() []gamehandler.RuntimeRequirementSpec {
	return []gamehandler.RuntimeRequirementSpec{
		{
			ID:          "stardew-smapi-installed",
			Name:        "SMAPI",
			Kind:        "mod-loader",
			Required:    true,
			ModTypes:    []string{"stardew-smapi-mod"},
			Message:     "SMAPI was not found in the Stardew Valley install folder. Deployed SMAPI mods will not load until SMAPI is installed and deployed.",
			OKMessage:   "SMAPI is present in the Stardew Valley install folder.",
			HelpURL:     "https://smapi.io/",
			InstallHint: "Install SMAPI through the same Nexus Mod Manager Download flow, then apply enabled mods.",
			Check:       smapiMarkers,
		},
		{
			ID:          "stardew-smapi-launch",
			Name:        "SMAPI launch",
			Kind:        "launch-tool",
			Required:    true,
			ModTypes:    []string{"stardew-smapi-mod"},
			Message:     "Steam is not configured to launch Stardew Valley through SMAPI.",
			OKMessage:   "Steam launch options reference SMAPI.",
			HelpURL:     "https://stardewvalleywiki.com/Modding:Installing_SMAPI_on_Steam_Deck",
			InstallHint: "Configure Stardew Valley to launch through SMAPI from DMM after SMAPI is deployed.",
			Check:       smapiLaunchMarkers,
		},
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex Stardew game registration",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-stardewvalley/src/game/StardewValleyGame.ts",
		},
		{
			Name: "Vortex Stardew installers",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-stardewvalley/src/installers",
		},
		{
			Name: "Vortex Stardew config mod feature",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-stardewvalley/src/configMod",
		},
		{
			Name: "Vortex Stardew SMAPI.io compatibility lookup",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-stardewvalley/src/compatibility/updateConflictInfo.ts",
		},
		{
			Name: "SMAPI.io mod compatibility API",
			URL:  "https://smapi.io/api/v3.0/mods",
		},
		{
			Name: "Stardew Wiki SMAPI Steam Deck guide",
			URL:  "https://stardewvalleywiki.com/Modding:Installing_SMAPI_on_Steam_Deck",
		},
	}
}

func smapiManifestExtractor() installplan.MetadataExtractorSpec {
	return installplan.MetadataExtractorSpec{
		Kind:                  MetadataKindSMAPIManifest,
		ManifestFileName:      "manifest.json",
		ExcludeLocaleManifest: true,
		Parse:                 smapiManifestMetadata,
	}
}

func smapiManifestMetadata(path string) installplan.ModMetadata {
	var manifest struct {
		Name              string `json:"Name"`
		UniqueID          string `json:"UniqueID"`
		Version           string `json:"Version"`
		EntryDLL          string `json:"EntryDll"`
		MinimumAPIVersion string `json:"MinimumApiVersion"`
		ContentPackFor    *struct {
			UniqueID       string `json:"UniqueID"`
			MinimumVersion string `json:"MinimumVersion"`
		} `json:"ContentPackFor"`
		Dependencies []struct {
			UniqueID       string `json:"UniqueID"`
			MinimumVersion string `json:"MinimumVersion"`
			IsRequired     *bool  `json:"IsRequired"`
		} `json:"Dependencies"`
	}
	if !installplan.ReadManifestJSON(path, &manifest) {
		return installplan.ModMetadata{}
	}
	metadata := installplan.ModMetadata{
		Kind:              MetadataKindSMAPIManifest,
		Name:              strings.TrimSpace(manifest.Name),
		UniqueID:          strings.TrimSpace(manifest.UniqueID),
		Version:           strings.TrimSpace(manifest.Version),
		ManifestVersion:   strings.TrimSpace(manifest.Version),
		EntryDLL:          strings.TrimSpace(manifest.EntryDLL),
		MinimumAPIVersion: strings.TrimSpace(manifest.MinimumAPIVersion),
	}
	if metadata.UniqueID != "" {
		metadata.AdditionalLogicalFileNames = []string{strings.ToLower(metadata.UniqueID)}
	}
	if manifest.ContentPackFor != nil && strings.TrimSpace(manifest.ContentPackFor.UniqueID) != "" {
		metadata.ContentPackFor = &installplan.ModDependency{
			UniqueID:       strings.TrimSpace(manifest.ContentPackFor.UniqueID),
			MinimumVersion: strings.TrimSpace(manifest.ContentPackFor.MinimumVersion),
			Required:       false,
		}
	}
	for _, dependency := range manifest.Dependencies {
		uniqueID := strings.TrimSpace(dependency.UniqueID)
		if uniqueID == "" {
			continue
		}
		metadata.Dependencies = append(metadata.Dependencies, installplan.ModDependency{
			UniqueID:       uniqueID,
			MinimumVersion: strings.TrimSpace(dependency.MinimumVersion),
			Required:       false,
		})
	}
	return metadata
}

func smapiMarkers(ctx context.Context, gamePath string) []string {
	for _, files := range smapiPlatformRequiredFiles(gamePath) {
		details, ok := existingFiles(ctx, gamePath, files)
		if ok {
			return details
		}
	}
	return nil
}

func smapiLaunchMarkers(ctx context.Context, gamePath string) []string {
	if ctx.Err() != nil {
		return nil
	}
	for _, executable := range smapiPlatformExecutables(gamePath) {
		markers := steam.LaunchOptionsContainTarget(ctx, SteamAppID, filepath.ToSlash(filepath.Join(gamePath, executable)))
		if len(markers) > 0 {
			return markers
		}
	}
	return nil
}

func smapiPlatformRequiredFiles(gamePath string) [][]string {
	linux := []string{
		SMAPIExecutable,
		"StardewModdingAPI.dll",
		filepath.Join("smapi-internal", "SMAPI.Toolkit.CoreInterfaces.dll"),
	}
	windows := []string{
		SMAPIWindowsExecutable,
		"StardewModdingAPI.dll",
		filepath.Join("smapi-internal", "SMAPI.Toolkit.CoreInterfaces.dll"),
	}
	if platformFileExists(gamePath, "Stardew Valley.exe") {
		return [][]string{windows}
	}
	if platformFileExists(gamePath, "StardewValley") {
		return [][]string{linux}
	}
	return [][]string{linux, windows}
}

func smapiPlatformExecutables(gamePath string) []string {
	if platformFileExists(gamePath, "Stardew Valley.exe") {
		return []string{SMAPIWindowsExecutable}
	}
	if platformFileExists(gamePath, "StardewValley") {
		return []string{SMAPIExecutable}
	}
	return []string{SMAPIExecutable, SMAPIWindowsExecutable}
}

func platformFileExists(gamePath, rel string) bool {
	path := filepath.Join(gamePath, filepath.FromSlash(rel))
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func existingFiles(ctx context.Context, gamePath string, rels []string) ([]string, bool) {
	details := make([]string, 0, len(rels))
	for _, rel := range rels {
		if ctx.Err() != nil {
			return nil, false
		}
		path := filepath.Join(gamePath, filepath.FromSlash(rel))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			details = append(details, filepath.ToSlash(path))
			continue
		}
		return nil, false
	}
	return details, true
}
