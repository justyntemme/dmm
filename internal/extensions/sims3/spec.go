package sims3

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/gameversiontext"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/targetroots"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "47890"
	VortexGameID = "thesims3"
	Name         = "The Sims 3"

	executable     = "game/bin/TS3.exe"
	modsRootID     = "sims3-documents-mods"
	packagesRootID = "sims3-documents-packages"
	packageModType = "sims3-package"
)

const resourceCfg = `Priority 500
PackedFile DCCache/*.dbc
PackedFile Packages/*.package
PackedFile Packages/*/*.package
PackedFile Packages/*/*/*.package
PackedFile Packages/*/*/*/*.package
PackedFile Packages/*/*/*/*/*.package
`

func Extension() sdk.Extension {
	return sdk.Extension{ID: VortexGameID, Name: Name, Kind: sdk.ExtensionKindGame, Version: "1.0.0-dmm.1", BuildID: "first-party-go", Register: Register}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:         []string{SteamAppID},
		NexusDomains:        []string{VortexGameID},
		VortexGameID:        VortexGameID,
		ExecutableRelative:  executable,
		RequiredFiles:       []string{executable},
		QueryModPathDynamic: true,
		MergeMode:           sdk.GameMergeModeAll,
		Deployment:          installplan.DeploymentSpec{AllowNeedsReviewState: true},
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       modsRootID,
		Name:     "The Sims 3 Documents Mods",
		Resolver: targetroots.ProtonDocuments(SteamAppID, "Electronic Arts", "The Sims 3", "Mods"),
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       packagesRootID,
		Name:     "The Sims 3 Documents Packages",
		Resolver: targetroots.ProtonDocuments(SteamAppID, "Electronic Arts", "The Sims 3", "Mods", "Packages"),
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: packageModType, TargetRootID: packagesRootID})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:thesims3:packages",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           packageModType,
		NameSource:        installplan.NameSourceArchive,
		TargetRootID:      packagesRootID,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:   "sims3-resource-cfg",
		Name: "Prepare The Sims 3 Resource.cfg",
		Actions: append(
			sdk.EnsureTargetRootDirectories(packagesRootID, "."),
			sdk.EnsureTargetRootFiles(modsRootID, resourceCfg, "Resource.cfg")...,
		),
	})
	r.RegisterGameVersionProvider(gameversiontext.Provider(gameversiontext.Options{
		ID:        "sims3-sku-version",
		Name:      "The Sims 3 SKU version",
		Paths:     []string{"game/bin/skuversion.txt"},
		Extractor: gameversiontext.KeyValueLine("GameVersion", "="),
	}))
	r.RegisterGameStore(sdk.GameStoreSpec{ID: "registry", Name: "Windows registry", Message: "Vortex discovers The Sims 3 through the Windows registry. DMM represents this store identity through manual install-path registration for extension-backed games, while the Steam Deck extension adapts the install target to the Proton Documents path for Steam app 47890."})
	r.RegisterSource(sdk.SourceRef{Name: "Vortex game-sims3 extension source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-sims3/src"})
}
