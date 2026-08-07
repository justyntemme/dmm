package metroexodus

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
	SteamAppID       = "1449560"
	LegacySteamAppID = "412020"
	VortexGameID     = "metroexodus"
	Name             = "Metro Exodus"

	rootModType = "metroexodus-root"
)

var requiredGameFiles = []string{"MetroExodus.exe"}

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Version:  "0.2.0-dmm.1",
		BuildID:  "first-party-go",
		Register: Register,
	}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:  []string{SteamAppID, LegacySteamAppID},
		NexusDomains: []string{VortexGameID},
		VortexGameID: VortexGameID,
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: rootModType, TargetRoot: ""})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:metroexodus:root",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           rootModType,
		NameSource:        installplan.NameSourceArchive,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterConflictIgnore(sdk.ConflictIgnoreSpec{
		ID:       "metroexodus-readme-changelog-conflicts",
		Name:     "Metro Exodus readme/changelog conflict ignore",
		Patterns: []string{"**/changelog*", "**/readme*"},
	})
	r.RegisterDeployIgnore(sdk.DeployIgnoreSpec{
		ID:       "metroexodus-readme-changelog-deploy",
		Name:     "Metro Exodus readme/changelog deploy ignore",
		Patterns: []string{"**/changelog*", "**/readme*"},
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "metroexodus-required-files",
		Name:        "Metro Exodus install files",
		Kind:        "game-files",
		Required:    true,
		ModTypes:    []string{rootModType},
		Message:     "The Metro Exodus game folder is missing files required by the Vortex extension.",
		OKMessage:   "The Metro Exodus game folder contains the executable required by the Vortex extension.",
		InstallHint: "Verify Metro Exodus files in Steam before testing Metro Exodus mods.",
		Check:       checkRequiredGameFiles,
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "MetroExodusSDK",
		Name:               "Metro Exodus SDK",
		ExecutableRelative: "SDK/bin_x64/Exodus_SDK.exe",
		RequiredFiles:      []string{"SDK/bin_x64/Exodus_SDK.exe"},
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "metroexodus-executable",
		Name:     "Metro Exodus executable marker",
		Provider: gameVersion,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func checkRequiredGameFiles(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil {
		return nil
	}
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return nil
	}
	details := make([]string, 0, len(requiredGameFiles))
	for _, rel := range requiredGameFiles {
		path := filepath.Join(gamePath, filepath.FromSlash(rel))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			details = append(details, filepath.ToSlash(path))
			continue
		}
		return nil
	}
	return details
}

func gameVersion(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.GameVersionResult{}, err
	}
	for _, rel := range requiredGameFiles {
		if info, err := os.Stat(filepath.Join(input.GamePath, filepath.FromSlash(rel))); err == nil && !info.IsDir() {
			return sdk.GameVersionResult{Version: "installed", Source: rel}, nil
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
			Name: "Metro Exodus Vortex extension page",
			URL:  "https://www.nexusmods.com/site/mods/907",
		},
		{
			Name: "Verified Vortex extension package file",
			URL:  "https://www.nexusmods.com/site/mods/907?tab=files&file_id=8800",
		},
		{
			Name: "Live Steam Deck executable/path verification",
			URL:  "extensionTargets.md#installed-games-snapshot",
		},
	}
}
