package stardewvalley

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
	"github.com/justyntemme/decky-mod-manager/internal/steam"
)

const (
	SteamAppID   = "413150"
	VortexGameID = "stardewvalley"
	Name         = "Stardew Valley"

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
	for _, requirement := range runtimeRequirements() {
		r.RegisterRuntimeRequirement(requirement)
	}
	r.RegisterRuntimeMetadataDependencies(sdk.RuntimeDependencySpec{
		MetadataKinds:       []string{MetadataKindSMAPIManifest},
		RequirementIDPrefix: "stardew-mod-dependency:",
		RequirementKind:     "mod-dependency",
		RequirementMessage:  "Required mod dependency is not enabled in this profile.",
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
			InstructionMode: installplan.InstructionManifestFolders,
		},
	}
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
			Required:       true,
		}
	}
	for _, dependency := range manifest.Dependencies {
		uniqueID := strings.TrimSpace(dependency.UniqueID)
		if uniqueID == "" {
			continue
		}
		required := true
		if dependency.IsRequired != nil {
			required = *dependency.IsRequired
		}
		metadata.Dependencies = append(metadata.Dependencies, installplan.ModDependency{
			UniqueID:       uniqueID,
			MinimumVersion: strings.TrimSpace(dependency.MinimumVersion),
			Required:       required,
		})
	}
	return metadata
}

func smapiMarkers(ctx context.Context, gamePath string) []string {
	var details []string
	for _, rel := range []string{
		SMAPIExecutable,
		"StardewModdingAPI.exe",
		"StardewModdingAPI.dll",
		filepath.Join("smapi-internal", "SMAPI.Toolkit.CoreInterfaces.dll"),
	} {
		if ctx.Err() != nil {
			return details
		}
		path := filepath.Join(gamePath, rel)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			details = append(details, filepath.ToSlash(path))
		}
	}
	return details
}

func smapiLaunchMarkers(ctx context.Context, gamePath string) []string {
	if ctx.Err() != nil {
		return nil
	}
	for _, executable := range []string{SMAPIExecutable, SMAPIWindowsExecutable} {
		markers := steam.LaunchOptionsContainTarget(ctx, SteamAppID, filepath.ToSlash(filepath.Join(gamePath, executable)))
		if len(markers) > 0 {
			return markers
		}
	}
	return nil
}
