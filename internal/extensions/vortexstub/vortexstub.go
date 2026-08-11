package vortexstub

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

type RootSupportModSpec struct {
	GameID       string
	SteamAppIDs  []string
	NexusDomains []string
	SourceName   string
	SourceDir    string
	SupportModID string
}

func RegisterRootSupportMod(r sdk.Registrar, spec RootSupportModSpec) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:  spec.SteamAppIDs,
		NexusDomains: spec.NexusDomains,
		VortexGameID: spec.GameID,
		VortexStub:   true,
		SupportModID: spec.SupportModID,
		MergeMode:    sdk.GameMergeModeNone,
		Deployment:   installplan.DeploymentSpec{AllowNeedsReviewState: true},
	})
	modType := spec.GameID + "-root"
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: ""})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:" + spec.GameID + ":root",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		TargetRoot:        "",
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex " + spec.SourceName + " extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/" + spec.SourceDir + "/src",
	})
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex support mod declared by " + spec.SourceName,
		URL:  "https://www.nexusmods.com/site/mods/" + spec.SupportModID,
	})
}
