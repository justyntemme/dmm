package civilizationvii

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "1295660"
	VortexGameID = "civilizationvii"
	Name         = "Sid Meier's Civilization VII"

	localModsRootID = "civilizationvii-local-mods"
	modType         = "civilizationvii-mod"
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
			DefaultStrategy:       installplan.DeployStrategyCopy,
		},
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       localModsRootID,
		Name:     "Civilization VII LocalAppData Mods",
		Resolver: localModsRoot,
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRootID: localModsRootID})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:civilizationvii:modinfo-package",
		VortexInstallerID: "civilizationvii-modinfo",
		Priority:          50,
		ModType:           modType,
		NameSource:        installplan.NameSourceManifestDisplay,
		CustomMatch:       matchModInfoPackage,
		CustomBuild:       buildModInfoPackage,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "civilizationvii-executable-version",
		Name:     "Civ7 executable version",
		Provider: gameVersion,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func localModsRoot(ctx context.Context, input sdk.TargetRootInput) (sdk.TargetRootResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.TargetRootResult{}, err
	}
	libraryPath := strings.TrimSpace(input.LibraryPath)
	if libraryPath == "" {
		libraryPath = inferSteamLibraryPath(input.GamePath)
	}
	if libraryPath == "" {
		return sdk.TargetRootResult{}, errors.New("Steam library path is required to resolve Civilization VII Proton LocalAppData Mods path")
	}
	return sdk.TargetRootResult{
		Path: filepath.Join(
			libraryPath,
			"steamapps",
			"compatdata",
			SteamAppID,
			"pfx",
			"drive_c",
			"users",
			"steamuser",
			"AppData",
			"Local",
			"Firaxis Games",
			"Sid Meier's Civilization VII",
			"Mods",
		),
		Source: "Steam Proton LocalAppData",
	}, nil
}

func inferSteamLibraryPath(gamePath string) string {
	gamePath = filepath.Clean(strings.TrimSpace(gamePath))
	marker := string(filepath.Separator) + filepath.Join("steamapps", "common") + string(filepath.Separator)
	idx := strings.Index(gamePath, marker)
	if idx <= 0 {
		return ""
	}
	return gamePath[:idx]
}

func gameVersion(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.GameVersionResult{}, err
	}
	gamePath := strings.TrimSpace(input.GamePath)
	if gamePath == "" {
		return sdk.GameVersionResult{}, nil
	}
	for _, rel := range []string{
		filepath.Join("Base", "Binaries", "Win64", "Civ7_Win64_Vulkan_FinalRelease.exe"),
		filepath.Join("Base", "Binaries", "Win64", "Civ7_Win64_DX12_FinalRelease.exe"),
	} {
		if _, err := os.Stat(filepath.Join(gamePath, rel)); err == nil {
			return sdk.GameVersionResult{Version: "installed", Source: filepath.ToSlash(rel)}, nil
		}
	}
	return sdk.GameVersionResult{}, os.ErrNotExist
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex central extension manifest entry",
			URL:  "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json",
		},
		{
			Name: "Civilization VII Vortex extension page",
			URL:  "https://www.nexusmods.com/site/mods/1182",
		},
	}
}
