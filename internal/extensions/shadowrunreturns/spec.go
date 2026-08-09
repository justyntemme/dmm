package shadowrunreturns

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "234650"
	VortexGameID = "shadowrunreturns"
	Name         = "Shadowrun Returns"

	executable = "Shadowrun.exe"
	modRoot    = "Shadowrun_Data/StreamingAssets/ContentPacks"
	modType    = "shadowrunreturns-content-pack"
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
		ExecutableRelative: executable,
		RequiredFiles:      []string{executable},
		QueryModPath:       modRoot,
		MergeMode:          sdk.GameMergeModeNone,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment:         installplan.DeploymentSpec{AllowNeedsReviewState: true},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: modRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:shadowrunreturns:content-pack",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		TargetRoot:        modRoot,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "shadowruneditor",
		Name:               "Editor",
		ExecutableRelative: "ShadowrunEditor.exe",
		RequiredFiles:      []string{"ShadowrunEditor.exe"},
		Relative:           true,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:             "shadowrunreturns-ensure-content-packs",
		Name:           "Ensure Shadowrun Returns ContentPacks folder exists",
		GeneratedFiles: []string{modRoot},
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:      "shadowrunreturns-hash-version",
		Name:    "Shadowrun Returns managed assembly hash version",
		Status:  sdk.CapabilityStatusBlocked,
		Message: "Vortex hashes several managed assemblies and the editor executable for game-version metadata. DMM needs a reusable hash-version provider runtime before this can run.",
	})
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex game-shadowrunreturns extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-shadowrunreturns/src",
	})
}
