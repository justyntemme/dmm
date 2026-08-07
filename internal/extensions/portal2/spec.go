package portal2

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
	SteamAppID   = "620"
	VortexGameID = "portal2"
	Name         = "Portal 2"

	portal2DLC3Root = "portal2_dlc3"
	modType         = "portal2-dlc3"
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
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: portal2DLC3Root})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:portal2:portal2-dlc3",
		VortexInstallerID: "portal2-dlc3",
		Priority:          50,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "portal2-dlc3-folder",
		Name:        "Portal 2 portal2_dlc3 folder",
		Kind:        "game-folder",
		Required:    true,
		ModTypes:    []string{modType},
		Message:     "Portal 2 is missing the portal2_dlc3 folder required by the Vortex extension. Deploy the selected profile so DMM can create and populate it.",
		OKMessage:   "Portal 2 has the portal2_dlc3 folder required by the Vortex extension.",
		InstallHint: "Enable a Portal 2 mod and apply the selected profile. DMM will create portal2_dlc3 as part of the managed deployment.",
		Check:       checkDLC3Folder,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func checkDLC3Folder(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil {
		return nil
	}
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return nil
	}
	path := filepath.Join(gamePath, portal2DLC3Root)
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return []string{filepath.ToSlash(path)}
	}
	return nil
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex central extension manifest entry",
			URL:  "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json",
		},
		{
			Name: "Portal 2 Vortex extension page",
			URL:  "https://www.nexusmods.com/site/mods/109",
		},
	}
}
