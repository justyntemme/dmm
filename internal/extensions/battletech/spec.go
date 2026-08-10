package battletech

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/targetroots"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "637090"
	VortexGameID = "battletech"
	Name         = "BattleTech"

	executable      = "BattleTech.exe"
	launcher        = "BattleTechLauncher.exe"
	modsRootID      = "battletech-documents-mods"
	modType         = "battletech-mods"
	versionFilePath = "BattleTech_Data/StreamingAssets/version.json"
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
		SteamAppIDs:         []string{SteamAppID},
		NexusDomains:        []string{VortexGameID},
		VortexGameID:        VortexGameID,
		ExecutableRelative:  executable,
		RequiredFiles:       []string{executable, launcher},
		QueryModPathDynamic: true,
		MergeMode:           sdk.GameMergeModeAll,
		RequiresCleanup:     true,
		Environment:         map[string]string{"SteamAPPId": SteamAppID},
		Deployment:          installplan.DeploymentSpec{AllowNeedsReviewState: true},
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       modsRootID,
		Name:     "BattleTech Documents mods",
		Resolver: targetroots.ProtonDocuments(SteamAppID, "My Games", "BattleTech", "mods"),
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRootID: modsRootID})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:battletech:mods",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		TargetRootID:      modsRootID,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:   "battletech-ensure-mods-folders",
		Name: "Ensure BattleTech mod folders exist",
		Actions: append(
			sdk.EnsureGameDirectories("Mods"),
			sdk.EnsureTargetRootDirectories(modsRootID, ".")...,
		),
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "battletech-product-version",
		Name:     "BattleTech ProductVersion",
		Provider: gameVersion,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventAddedFiles,
		Name:    "Adopt generated BattleTech mod files",
		Handler: adoptGeneratedFiles,
	})
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex game-battletech extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-battletech/src",
	})
}

func gameVersion(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.GameVersionResult{}, err
	}
	gamePath := strings.TrimSpace(input.GamePath)
	if gamePath == "" {
		return sdk.GameVersionResult{}, nil
	}
	data, err := os.ReadFile(filepath.Join(gamePath, filepath.FromSlash(versionFilePath)))
	if err != nil {
		return sdk.GameVersionResult{}, err
	}
	var payload struct {
		ProductVersion string `json:"ProductVersion"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return sdk.GameVersionResult{}, err
	}
	version := strings.TrimSpace(payload.ProductVersion)
	if version == "" {
		return sdk.GameVersionResult{}, nil
	}
	return sdk.GameVersionResult{Version: version, Source: versionFilePath}, nil
}
