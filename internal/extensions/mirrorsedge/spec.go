package mirrorsedge

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "17410"
	VortexGameID = "mirrorsedge"
	Name         = "Mirror's Edge"

	cookedPCModType = "mirrorsedge-cookedpc"
	blockedModType  = "mirrorsedge-unclassified-blocked"
	cookedPCRoot    = "TdGame/CookedPC"
)

const blockedReason = "Mirror's Edge archive layout is not classified by the verified extension rules. DMM currently supports game-root TdGame/CookedPC Unreal package replacement archives only; user-Documents Published/CookedPC mod-menu flows, executable tools, and unclassified payloads stay blocked until a source-reviewed target-root rule can place them safely."

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
	r.RegisterModType(installplan.ModTypeSpec{ID: cookedPCModType, TargetRoot: cookedPCRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: blockedModType, TargetRoot: ""})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "source:mirrorsedge:cookedpc",
		VortexInstallerID: "mirrorsedge-cookedpc",
		Priority:          30,
		ModType:           cookedPCModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchCookedPCArchive,
		CustomBuild:       buildCookedPCArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "source:mirrorsedge:unclassified-blocked",
		VortexInstallerID: "mirrorsedge-unclassified-blocked",
		Priority:          10000,
		ModType:           blockedModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchAnyArchive,
		InstructionMode:   installplan.InstructionUnsupported,
		UnsupportedReason: blockedReason,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "mirrorsedge-cookedpc-present",
		Name:        "Mirror's Edge CookedPC folder",
		Kind:        "game-folder",
		Required:    true,
		ModTypes:    []string{cookedPCModType},
		Message:     "Mirror's Edge is missing the expected executable or TdGame/CookedPC folder.",
		OKMessage:   "Mirror's Edge has the expected executable and TdGame/CookedPC folder markers.",
		InstallHint: "Verify Mirror's Edge files in Steam before testing CookedPC replacement mods.",
		Check:       requiredFilesCheck,
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "mirrorsedge-executable",
		Name:     "Mirror's Edge executable marker",
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
	required := []string{"Binaries/MirrorsEdge.exe", cookedPCRoot}
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
	if info, err := os.Stat(filepath.Join(input.GamePath, filepath.FromSlash("Binaries/MirrorsEdge.exe"))); err == nil && !info.IsDir() {
		return sdk.GameVersionResult{Version: "installed", Source: "Binaries/MirrorsEdge.exe"}, nil
	}
	return sdk.GameVersionResult{}, os.ErrNotExist
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Nexus API game list verified the Mirror's Edge domain", URL: "https://www.nexusmods.com/mirrorsedge"},
		{Name: "Mirror's Edge Nexus CookedPC/Characters replacement instructions", URL: "https://www.nexusmods.com/mirrorsedge/mods/31"},
		{Name: "Mirror's Edge community Published/CookedPC mod-menu guide kept blocked", URL: "https://steamcommunity.com/sharedfiles/filedetails/?id=1981216701"},
		{Name: "Live Steam Deck executable/path verification", URL: "extensionTargets.md#installed-games-snapshot"},
		{Name: "Checked bundled Vortex game extension source; no reviewed Mirror's Edge handler found", URL: "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games"},
	}
}

func matchAnyArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	return err == nil && !simplearchive.ContainsFOMOD(files) && len(files) > 0
}
