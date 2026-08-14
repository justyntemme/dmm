package steinsgate

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
	SteamAppID   = "412830"
	VortexGameID = "steinsgate"
	Name         = "Steins;Gate"

	usrdirModType = "steinsgate-usrdir"
	usrdirRoot    = "USRDIR"
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
	r.RegisterModType(installplan.ModTypeSpec{ID: usrdirModType, TargetRoot: usrdirRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "source:steinsgate:usrdir",
		VortexInstallerID: "steinsgate-usrdir",
		Priority:          30,
		ModType:           usrdirModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchUSRDIRArchive,
		CustomBuild:       buildUSRDIRArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "steinsgate-usrdir-present",
		Name:        "Steins;Gate USRDIR folder",
		Kind:        "game-folder",
		Required:    true,
		ModTypes:    []string{usrdirModType},
		Message:     "Steins;Gate is missing the expected executable or USRDIR folder.",
		OKMessage:   "Steins;Gate has the expected executable and USRDIR folder markers.",
		InstallHint: "Verify Steins;Gate files in Steam before testing USRDIR replacement mods.",
		Check:       requiredFilesCheck,
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "steinsgate-executable",
		Name:     "Steins;Gate executable marker",
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
	required := []string{"Game.exe", "USRDIR"}
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
	if info, err := os.Stat(filepath.Join(input.GamePath, "Game.exe")); err == nil && !info.IsDir() {
		return sdk.GameVersionResult{Version: "installed", Source: "Game.exe"}, nil
	}
	return sdk.GameVersionResult{}, os.ErrNotExist
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Nexus API game list verified the Steins;Gate domain", URL: "https://www.nexusmods.com/steinsgate"},
		{Name: "Steins;Gate Nexus USRDIR replacement instructions", URL: "https://www.nexusmods.com/steinsgate/mods/2"},
		{Name: "Live Steam Deck executable/path verification", URL: "extensionTargets.md#installed-games-snapshot"},
		{Name: "Checked bundled Vortex game extension source; no Steins;Gate game extension ships upstream", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games"},
	}
}
