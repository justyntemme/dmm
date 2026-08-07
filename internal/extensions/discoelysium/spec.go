package discoelysium

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
	SteamAppID   = "632470"
	VortexGameID = "discoelysium"
	Name         = "Disco Elysium"

	dataFolder    = "disco_Data"
	altDataFolder = "Disco Elysium_Data"

	rootModType            = "discoelysium-root"
	bepinexInjectorModType = "discoelysium-bepinex-injector"
	bepinexRootModType     = "discoelysium-bepinex-root"
	bepinexPluginModType   = "discoelysium-bepinex-plugin"
	bepinexConfigModType   = "discoelysium-bepinex-config-manager"
	assemblyModType        = "discoelysium-assemblydll"
	assetsModType          = "discoelysium-assets"
	unclassifiedModType    = "discoelysium-unclassified-blocked"

	bepinexRoot       = "BepInEx"
	bepinexPluginRoot = bepinexRoot + "/plugins"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Version:  "0.1.4-dmm.1",
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
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: rootModType, TargetRoot: ""})
	r.RegisterModType(installplan.ModTypeSpec{ID: bepinexInjectorModType, TargetRoot: ""})
	r.RegisterModType(installplan.ModTypeSpec{ID: bepinexRootModType, TargetRoot: bepinexRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: bepinexPluginModType, TargetRoot: bepinexPluginRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: bepinexConfigModType, TargetRoot: bepinexRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: assemblyModType, TargetRoot: ""})
	r.RegisterModType(installplan.ModTypeSpec{ID: assetsModType, TargetRoot: dataFolder})
	r.RegisterModType(installplan.ModTypeSpec{ID: unclassifiedModType, TargetRoot: ""})

	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:discoelysium:root",
		VortexInstallerID: "discoelysium-root",
		Priority:          8,
		ModType:           rootModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchRootDataFolder,
		CustomBuild:       buildRootDataFolder,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:discoelysium:bepinex-config-manager",
		VortexInstallerID: "discoelysium-bepcfgman",
		Priority:          9,
		ModType:           bepinexConfigModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchBepInExConfigManager,
		CustomBuild:       buildBepInExConfigManager,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:discoelysium:bepinex-injector",
		VortexInstallerID: "bepis-injector-extensible",
		Priority:          10,
		ModType:           bepinexInjectorModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchBepInExInjector,
		CustomBuild:       buildBepInExInjector,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:discoelysium:bepinex-root",
		VortexInstallerID: "bepinex-root",
		Priority:          10,
		ModType:           bepinexRootModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchBepInExRootMod,
		CustomBuild:       buildBepInExRootMod,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:discoelysium:bepinex-plugin",
		VortexInstallerID: "bepinex-plugin",
		Priority:          13,
		ModType:           bepinexPluginModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchBepInExPlugin,
		CustomBuild:       buildBepInExPlugin,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:discoelysium:assemblydll",
		VortexInstallerID: "discoelysium-assemblydll",
		Priority:          25,
		ModType:           assemblyModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchAssemblyMod,
		CustomBuild:       buildAssemblyMod,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:discoelysium:assets",
		VortexInstallerID: "discoelysium-assets",
		Priority:          27,
		ModType:           assetsModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchAssetsMod,
		CustomBuild:       buildAssetsMod,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:discoelysium:unclassified-blocked",
		VortexInstallerID: "discoelysium-unclassified",
		Priority:          49,
		ModType:           unclassifiedModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchUnclassifiedArchive,
		InstructionMode:   installplan.InstructionUnsupported,
		UnsupportedReason: "Disco Elysium archive layout is not classified by the verified extension rules. DMM blocks it until a specific extension-owned rule can place the files safely.",
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "discoelysium-bepinex-installed",
		Name:        "BepInEx",
		Kind:        "mod-loader",
		Required:    true,
		ModTypes:    []string{bepinexRootModType, bepinexPluginModType, bepinexConfigModType},
		Message:     "BepInEx is required before enabled Disco Elysium BepInEx mods can load.",
		OKMessage:   "BepInEx is present in the Disco Elysium game folder.",
		InstallHint: "Install the BepInEx Unity IL2CPP x64 runtime for Disco Elysium, then enable and deploy it from DMM before enabling BepInEx plugin mods.",
		HelpURL:     "https://builds.bepinex.dev/projects/bepinex_be",
		Check:       checkBepInExFiles,
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "discoelysium-executable",
		Name:     "Disco Elysium executable marker",
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
			ID:       "discoelysium-ignore-" + sanitizeID(pattern),
			Name:     "Disco Elysium ignored metadata " + pattern,
			Patterns: []string{pattern},
		})
	}
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func checkBepInExFiles(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil {
		return nil
	}
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return nil
	}
	for _, rel := range []string{
		filepath.Join("BepInEx", "core", "BepInEx.Core.dll"),
		filepath.Join("BepInEx", "core", "BepInEx.Preloader.Core.dll"),
		filepath.Join("BepInEx", "core", "BepInEx.Unity.IL2CPP.dll"),
		filepath.Join("BepInEx", "core", "BepInEx.dll"),
		filepath.Join("BepinEx", "core", "BepInEx.Core.dll"),
		filepath.Join("BepinEx", "core", "BepInEx.Preloader.Core.dll"),
		filepath.Join("winhttp.dll"),
	} {
		if info, err := os.Stat(filepath.Join(gamePath, rel)); err == nil && !info.IsDir() {
			return []string{filepath.ToSlash(filepath.Join(gamePath, rel))}
		}
	}
	return nil
}

func gameVersion(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.GameVersionResult{}, err
	}
	for _, rel := range []string{"disco.exe", "Disco Elysium.exe", "gamelaunchhelper.exe"} {
		if info, err := os.Stat(filepath.Join(input.GamePath, rel)); err == nil && !info.IsDir() {
			return sdk.GameVersionResult{Version: "installed", Source: rel}, nil
		}
	}
	return sdk.GameVersionResult{}, os.ErrNotExist
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

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Vortex central extension manifest entry site-mod-1643-file-7265", URL: "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json"},
		{Name: "Disco Elysium Vortex extension package v0.1.4", URL: "https://www.nexusmods.com/site/mods/1643?tab=files"},
		{Name: "Vortex modtype-bepinex shared source", URL: "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/modtype-bepinex"},
		{Name: "Live Steam Deck executable/path verification", URL: "extensionTargets.md#installed-games-snapshot"},
	}
}
