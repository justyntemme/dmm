package spelunky

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
	SteamAppID   = "239350"
	VortexGameID = "spelunky"
	Name         = "Spelunky"

	dataModType = "spelunky-data"
	dataRoot    = "Data"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Version:  "1.0.0-dmm.1",
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
	r.RegisterModType(installplan.ModTypeSpec{ID: dataModType, TargetRoot: dataRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "source:spelunky:data",
		VortexInstallerID: "spelunky-data",
		Priority:          30,
		ModType:           dataModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchDataArchive,
		CustomBuild:       buildDataArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "spelunky-data-present",
		Name:        "Spelunky Data folder",
		Kind:        "game-folder",
		Required:    true,
		ModTypes:    []string{dataModType},
		Message:     "Spelunky is missing the expected executable or Data folder.",
		OKMessage:   "Spelunky has the expected executable and Data folder markers.",
		InstallHint: "Verify Spelunky files in Steam before testing Data-folder mods.",
		Check:       requiredFilesCheck,
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "spelunky-executable",
		Name:     "Spelunky executable marker",
		Provider: gameVersion,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func requiredFilesCheck(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil || strings.TrimSpace(gamePath) == "" {
		return nil
	}
	required := []string{"Spelunky.exe", "Data"}
	var details []string
	for _, rel := range required {
		path := filepath.Join(gamePath, filepath.FromSlash(rel))
		if info, err := os.Stat(path); err == nil {
			if info.IsDir() {
				details = append(details, filepath.ToSlash(path)+"/")
			} else {
				details = append(details, filepath.ToSlash(path))
			}
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
	if info, err := os.Stat(filepath.Join(input.GamePath, "Spelunky.exe")); err == nil && !info.IsDir() {
		return sdk.GameVersionResult{Version: "installed", Source: "Spelunky.exe"}, nil
	}
	return sdk.GameVersionResult{}, os.ErrNotExist
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Nexus API game list verified the Spelunky domain", URL: "https://www.nexusmods.com/spelunky"},
		{Name: "Spelunky Nexus Data-folder replacement instructions", URL: "https://www.nexusmods.com/spelunky/mods/10"},
		{Name: "Spelunky Nexus Localization/Textures Data-folder instructions", URL: "https://www.nexusmods.com/spelunky/mods/7"},
		{Name: "Spelunky community Data-folder modding note", URL: "https://www.reddit.com/r/spelunky/comments/1u4lh4/just_got_spelunky_on_steam_and_how_do_i_install/"},
		{Name: "Live Steam Deck executable/path verification", URL: "extensionTargets.md#installed-games-snapshot"},
		{Name: "Checked bundled Vortex game extension source; no reviewed Spelunky handler found", URL: "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games"},
	}
}
