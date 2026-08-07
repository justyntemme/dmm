package megabonk

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID     = "3405340"
	SteamDemoAppID = "3520070"
	VortexGameID   = "megabonk"
	Name           = "Megabonk"

	platformLinux   = "linux"
	platformWindows = "windows"

	gameExecutableWindows = "Megabonk.exe"
	gameExecutableLinux   = "Megabonk.x86_64"
	dataFolder            = "Megabonk_Data"
	assemblyFile          = "GameAssembly.dll"

	bepInExRuntimeModType       = "megabonk-bepinex"
	melonRuntimeModType         = "megabonk-melonloader"
	rootModType                 = "megabonk-root"
	bepInExConfigManagerModType = "megabonk-bepcfgman"
	assemblyModType             = "megabonk-assemblydll"
	assetsModType               = "megabonk-assets"
	bepInExModType              = "megabonk-bepinexmod"
	bepInExPluginsModType       = "megabonk-bepinex-plugins"
	bepInExPatchersModType      = "megabonk-bepinex-patchers"
	bepInExConfigModType        = "megabonk-bepinex-config"
	melonModType                = "megabonk-melonmod"
	melonPluginsModType         = "megabonk-melonloader-plugins"
	melonModsModType            = "megabonk-melonloader-mods"
	melonConfigModType          = "megabonk-melonloader-config"
	customCharsBepInExModType   = "megabonk-customcharacters-bepinex"
	customCharsMelonModType     = "megabonk-customcharacters-melonloader"
	fallbackModType             = "megabonk-fallback-blocked"

	bepInExRoot             = "BepInEx"
	bepInExPluginsRoot      = "BepInEx/plugins"
	bepInExPatchersRoot     = "BepInEx/patchers"
	bepInExConfigRoot       = "BepInEx/config"
	melonPluginsRoot        = "Plugins"
	melonModsRoot           = "Mods"
	melonConfigRoot         = "UserData"
	customCharsFolder       = "CustomCharacters"
	customCharsBepInExRoot  = "BepInEx/plugins/CustomCharacters"
	customCharsMelonRoot    = "Mods/CustomCharacters"
	bepInExRuntimeMarkerRel = "BepInEx/core/BepInEx.Core.dll"
	melonRuntimeMarkerRel   = "MelonLoader/net6/MelonLoader.dll"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Version:  "0.1.3-dmm.1",
		BuildID:  "first-party-go",
		Register: Register,
	}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:  []string{SteamAppID, SteamDemoAppID},
		NexusDomains: []string{VortexGameID},
		VortexGameID: VortexGameID,
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	for _, platform := range installPlatforms() {
		r.RegisterInstallPlatform(platform)
	}
	for _, modType := range modTypes() {
		r.RegisterModType(modType)
	}
	for _, installer := range installers() {
		r.RegisterInstaller(installer)
	}
	for _, requirement := range runtimeRequirements() {
		r.RegisterRuntimeRequirement(requirement)
	}
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "megabonk-executable",
		Name:     "Megabonk executable marker",
		Provider: gameVersion,
	})
	for _, pattern := range []string{
		"**/manifest.json",
		"**/icon.png",
		"**/CHANGELOG.md",
		"**/readme.txt",
		"**/README.txt",
		"**/ReadMe.txt",
		"**/Readme.txt",
	} {
		r.RegisterConflictIgnore(sdk.ConflictIgnoreSpec{
			ID:       "megabonk-ignore-" + sanitizeID(pattern),
			Name:     "Megabonk ignored metadata " + pattern,
			Patterns: []string{pattern},
		})
	}
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func installPlatforms() []sdk.InstallPlatformSpec {
	return []sdk.InstallPlatformSpec{
		{
			ID:      platformLinux,
			Name:    "Native Linux",
			Markers: []string{gameExecutableLinux, "GameAssembly.so", filepath.ToSlash(filepath.Join(dataFolder, "globalgamemanagers"))},
		},
		{
			ID:      platformWindows,
			Name:    "Windows/Proton",
			Markers: []string{gameExecutableWindows, assemblyFile, filepath.ToSlash(filepath.Join(dataFolder, "globalgamemanagers"))},
		},
	}
}

func modTypes() []installplan.ModTypeSpec {
	return []installplan.ModTypeSpec{
		{ID: bepInExRuntimeModType, TargetRoot: ""},
		{ID: melonRuntimeModType, TargetRoot: ""},
		{ID: rootModType, TargetRoot: ""},
		{ID: bepInExConfigManagerModType, TargetRoot: bepInExRoot},
		{ID: assemblyModType, TargetRoot: ""},
		{ID: assetsModType, TargetRoot: dataFolder},
		{ID: bepInExModType, TargetRoot: bepInExRoot},
		{ID: bepInExPluginsModType, TargetRoot: bepInExPluginsRoot},
		{ID: bepInExPatchersModType, TargetRoot: bepInExPatchersRoot},
		{ID: bepInExConfigModType, TargetRoot: bepInExConfigRoot},
		{ID: melonModType, TargetRoot: ""},
		{ID: melonPluginsModType, TargetRoot: melonPluginsRoot},
		{ID: melonModsModType, TargetRoot: melonModsRoot},
		{ID: melonConfigModType, TargetRoot: melonConfigRoot},
		{ID: customCharsBepInExModType, TargetRoot: customCharsBepInExRoot},
		{ID: customCharsMelonModType, TargetRoot: customCharsMelonRoot},
		{ID: fallbackModType, TargetRoot: ""},
	}
}

func installers() []installplan.InstallerSpec {
	return []installplan.InstallerSpec{
		{
			ID:                "vortex:megabonk:bepinex:native-linux-blocked",
			VortexInstallerID: "megabonk-bepinex",
			PlatformID:        platformLinux,
			Priority:          25,
			ModType:           bepInExRuntimeModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchBepInExRuntime,
			InstructionMode:   installplan.InstructionUnsupported,
			UnsupportedReason: "The source-verified Megabonk Vortex BepInEx installer downloads a Windows x64 payload. This Steam Deck install is native Linux, so DMM blocks that runtime until a Linux Megabonk loader package and launch behavior are verified.",
		},
		{
			ID:                "vortex:megabonk:bepinex",
			VortexInstallerID: "megabonk-bepinex",
			PlatformID:        platformWindows,
			Priority:          25,
			ModType:           bepInExRuntimeModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchBepInExRuntime,
			CustomBuild:       buildBepInExRuntime,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:megabonk:melonloader:native-linux-blocked",
			VortexInstallerID: "megabonk-melonloader",
			PlatformID:        platformLinux,
			Priority:          26,
			ModType:           melonRuntimeModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchMelonRuntime,
			InstructionMode:   installplan.InstructionUnsupported,
			UnsupportedReason: "The source-verified Megabonk Vortex MelonLoader installer downloads a Windows x64 payload. This Steam Deck install is native Linux, so DMM blocks that runtime until a Linux Megabonk loader package and launch behavior are verified.",
		},
		{
			ID:                "vortex:megabonk:melonloader",
			VortexInstallerID: "megabonk-melonloader",
			PlatformID:        platformWindows,
			Priority:          26,
			ModType:           melonRuntimeModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchMelonRuntime,
			CustomBuild:       buildMelonRuntime,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:megabonk:root",
			VortexInstallerID: "megabonk-root",
			Priority:          27,
			ModType:           rootModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchRootDataFolder,
			CustomBuild:       buildRootDataFolder,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:megabonk:bepinex-config-manager",
			VortexInstallerID: "megabonk-bepcfgman",
			Priority:          29,
			ModType:           bepInExConfigManagerModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchBepInExConfigManager,
			CustomBuild:       buildBepInExConfigManager,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:megabonk:assemblydll",
			VortexInstallerID: "megabonk-assemblydll",
			PlatformID:        platformWindows,
			Priority:          31,
			ModType:           assemblyModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchAssemblyMod,
			CustomBuild:       buildAssemblyMod,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:megabonk:plugin",
			VortexInstallerID: "megabonk-plugin",
			Priority:          33,
			ModType:           bepInExPluginsModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchPlugin,
			CustomBuild:       buildPlugin,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:megabonk:assets",
			VortexInstallerID: "megabonk-assets",
			Priority:          37,
			ModType:           assetsModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchAssetsMod,
			CustomBuild:       buildAssetsMod,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:megabonk:customcharacters",
			VortexInstallerID: "megabonk-customcharacters",
			Priority:          39,
			ModType:           customCharsBepInExModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchCustomCharacters,
			CustomBuild:       buildCustomCharacters,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:megabonk:fallback-blocked",
			VortexInstallerID: "megabonk-fallback",
			Priority:          99,
			ModType:           fallbackModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchFallbackBlocked,
			InstructionMode:   installplan.InstructionUnsupported,
			UnsupportedReason: "Megabonk Vortex fallback installer reached. DMM blocks arbitrary root-file placement until a specific extension-owned rule can classify this archive safely.",
		},
	}
}

func runtimeRequirements() []gamehandler.RuntimeRequirementSpec {
	return []gamehandler.RuntimeRequirementSpec{
		{
			ID:          "megabonk-bepinex-installed",
			Name:        "BepInEx",
			Kind:        "mod-loader",
			Required:    true,
			ModTypes:    []string{bepInExConfigManagerModType, bepInExModType, bepInExPluginsModType, bepInExPatchersModType, bepInExConfigModType, customCharsBepInExModType},
			Message:     "BepInEx was not found in the Megabonk install folder. Deployed BepInEx mods will not load until BepInEx is installed.",
			OKMessage:   "BepInEx is present in the Megabonk install folder.",
			InstallHint: "Install a Megabonk-compatible BepInEx IL2CPP runtime for the active platform, then enable and deploy BepInEx mods.",
			HelpURL:     "https://builds.bepinex.dev/projects/bepinex_be",
			Check:       checkBepInExFiles,
		},
		{
			ID:          "megabonk-melonloader-installed",
			Name:        "MelonLoader",
			Kind:        "mod-loader",
			Required:    true,
			ModTypes:    []string{melonModType, melonPluginsModType, melonModsModType, melonConfigModType, customCharsMelonModType},
			Message:     "MelonLoader was not found in the Megabonk install folder. Deployed MelonLoader mods will not load until MelonLoader is installed.",
			OKMessage:   "MelonLoader is present in the Megabonk install folder.",
			InstallHint: "Install a Megabonk-compatible MelonLoader runtime for the active platform, then enable and deploy MelonLoader mods.",
			HelpURL:     "https://github.com/LavaGang/MelonLoader/releases",
			Check:       checkMelonLoaderFiles,
		},
	}
}

func checkBepInExFiles(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil {
		return nil
	}
	for _, rel := range []string{
		bepInExRuntimeMarkerRel,
		"BepInEx/core/BepInEx.dll",
		"BepInEx/core/BepInEx.Preloader.Core.dll",
		"doorstop_config.ini",
		"winhttp.dll",
		"libdoorstop_x64.so",
	} {
		if info, err := os.Stat(filepath.Join(gamePath, filepath.FromSlash(rel))); err == nil && !info.IsDir() {
			return []string{filepath.ToSlash(filepath.Join(gamePath, filepath.FromSlash(rel)))}
		}
	}
	return nil
}

func checkMelonLoaderFiles(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil {
		return nil
	}
	for _, rel := range []string{
		melonRuntimeMarkerRel,
		"MelonLoader/MelonLoader.dll",
		"MelonLoader.dll",
		"version.dll",
	} {
		if info, err := os.Stat(filepath.Join(gamePath, filepath.FromSlash(rel))); err == nil && !info.IsDir() {
			return []string{filepath.ToSlash(filepath.Join(gamePath, filepath.FromSlash(rel)))}
		}
	}
	return nil
}

func gameVersion(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.GameVersionResult{}, err
	}
	for _, rel := range []string{gameExecutableLinux, gameExecutableWindows} {
		if info, err := os.Stat(filepath.Join(input.GamePath, rel)); err == nil && !info.IsDir() {
			return sdk.GameVersionResult{Version: "installed", Source: rel}, nil
		}
	}
	return sdk.GameVersionResult{}, os.ErrNotExist
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Vortex central extension manifest entry site-mod-1495-file-7663", URL: "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json"},
		{Name: "Megabonk Vortex extension package v0.1.3", URL: "https://www.nexusmods.com/site/mods/1495?tab=files"},
		{Name: "Megabonk Vortex extension package source index.js", URL: "https://www.nexusmods.com/site/mods/1495?tab=files"},
		{Name: "Live Steam Deck executable/path verification", URL: "extensionTargets.md#installed-games-snapshot"},
	}
}

func sanitizeID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
