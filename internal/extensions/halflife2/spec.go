package halflife2

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
	HalfLife2AppID       = "220"
	VortexGameID         = "halflife2"
	VortexInternalGameID = "half-life2"
	Name                 = "Half-Life 2"

	vpkModType = "halflife2-vpk"
	vpkRoot    = "hl2/custom"
)

var (
	requiredGameFiles = []string{"hl2/gameinfo.txt"}
	executableMarkers = []string{"hl2_linux", "hl2.sh", "hl2.exe"}
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Version:  "1.1.0-dmm.1",
		BuildID:  "first-party-go",
		Register: Register,
	}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:  []string{HalfLife2AppID},
		NexusDomains: []string{VortexGameID},
		VortexGameID: VortexGameID,
		Deployment: installplan.DeploymentSpec{
			DefaultStrategy:       installplan.DeployStrategySymlink,
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: vpkModType, TargetRoot: vpkRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:halflife2:vpk",
		VortexInstallerID: "half-life2-mod",
		Priority:          25,
		ModType:           vpkModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchVPKArchive,
		CustomBuild:       buildVPKArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "halflife2-required-files",
		Name:        "Half-Life 2 install files",
		Kind:        "game-files",
		Required:    true,
		ModTypes:    []string{vpkModType},
		Message:     "The Half-Life 2 game folder is missing files needed for VPK deployment.",
		OKMessage:   "The Half-Life 2 game folder contains the expected executable marker and hl2/gameinfo.txt.",
		InstallHint: "Verify Half-Life 2 files in Steam before testing Half-Life 2 VPK mods.",
		Check:       checkRequiredGameFiles,
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "halflife2-executable",
		Name:     "Half-Life 2 executable marker",
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
	if gamePath == "" || firstExistingFile(gamePath, executableMarkers) == "" {
		return nil
	}
	details := make([]string, 0, len(requiredGameFiles)+1)
	details = append(details, filepath.ToSlash(firstExistingFile(gamePath, executableMarkers)))
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
	if marker := firstExistingFile(input.GamePath, executableMarkers); marker != "" {
		return sdk.GameVersionResult{Version: "installed", Source: filepath.ToSlash(strings.TrimPrefix(marker, strings.TrimRight(input.GamePath, string(filepath.Separator))+string(filepath.Separator)))}, nil
	}
	return sdk.GameVersionResult{}, os.ErrNotExist
}

func firstExistingFile(gamePath string, rels []string) string {
	for _, rel := range rels {
		path := filepath.Join(gamePath, filepath.FromSlash(rel))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex central extension manifest entry site-mod-80-file-516",
			URL:  "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json",
		},
		{
			Name: "Half-Life 2 Vortex extension package v1.1.0",
			URL:  "https://www.nexusmods.com/site/mods/80?tab=files",
		},
		{
			Name: "Nexus API domain verification for halflife2",
			URL:  "https://api.nexusmods.com/v1/games/halflife2.json",
		},
		{
			Name: "Live Steam Deck executable/path verification",
			URL:  "extensionTargets.md#installed-games-snapshot",
		},
	}
}
