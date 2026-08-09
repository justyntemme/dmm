package teamfortress2

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "440"
	VortexGameID = "teamfortress2"
	Name         = "Team Fortress 2"

	vpkModType = "teamfortress2-vpk"
	vpkRoot    = "tf/custom"
	infoFile   = "tf/steam.inf"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Kind:     sdk.ExtensionKindGame,
		Version:  "1.0.0-dmm.1",
		BuildID:  "first-party-go",
		Register: Register,
	}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:        []string{SteamAppID},
		NexusDomains:       []string{VortexGameID},
		VortexGameID:       VortexGameID,
		ExecutableRelative: "tf_win64.exe",
		RequiredFiles:      []string{"tf_win64.exe", "tf/gameinfo.txt"},
		QueryModPath:       vpkRoot,
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "hammer",
		Name:               "Hammer",
		ExecutableRelative: "hammer.exe",
		RequiredFiles:      []string{"hammer.exe"},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: vpkModType, TargetRoot: vpkRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:teamfortress2:vpk",
		VortexInstallerID: "teamfortress2-mod",
		Priority:          25,
		ModType:           vpkModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchVPKArchive,
		CustomBuild:       buildVPKArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "teamfortress2-steam-inf",
		Name:     "Team Fortress 2 ClientVersion",
		Provider: gameVersion,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func gameVersion(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.GameVersionResult{}, err
	}
	file, err := os.Open(filepath.Join(input.GamePath, filepath.FromSlash(infoFile)))
	if err != nil {
		return sdk.GameVersionResult{}, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		version, ok := strings.CutPrefix(line, "ClientVersion=")
		if ok && strings.TrimSpace(version) != "" {
			return sdk.GameVersionResult{Version: strings.TrimSpace(version), Source: infoFile}, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return sdk.GameVersionResult{}, err
	}
	return sdk.GameVersionResult{}, os.ErrNotExist
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex game-teamfortress2 extension source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-teamfortress2/src",
		},
	}
}
