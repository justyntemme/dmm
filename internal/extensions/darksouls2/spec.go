package darksouls2

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sharedmodtypes"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppIDOriginal = "236430"
	SteamAppIDScholar  = "335300"
	VortexGameID       = "darksouls2"
	Name               = "Dark Souls II"

	executable = "Game/DarksoulsII.exe"
	modType    = "darksouls2-root"

	gedosatoRootID      = "darksouls2-gedosato-textures"
	gedosatoTexturePath = "DarkSoulsII"
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
		SteamAppIDs:        []string{SteamAppIDOriginal, SteamAppIDScholar},
		NexusDomains:       []string{VortexGameID},
		VortexGameID:       VortexGameID,
		ExecutableRelative: executable,
		RequiredFiles:      []string{executable},
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": SteamAppIDOriginal},
		Deployment:         installplan.DeploymentSpec{AllowNeedsReviewState: true},
	})
	r.RegisterTargetRoot(sharedmodtypes.GeDoSaToTargetRoot(gedosatoRootID, "Dark Souls II GeDoSaTo textures", gedosatoTexturePath))
	r.RegisterModType(sharedmodtypes.GeDoSaToModTypeSpec(gedosatoRootID))
	r.RegisterInstaller(sharedmodtypes.GeDoSaToInstaller("vortex:darksouls2:gedosato", 50, gedosatoRootID))
	r.RegisterRuntimeRequirement(sharedmodtypes.GeDoSaToRuntimeRequirement("darksouls2-gedosato-installed", []string{sharedmodtypes.GeDoSaToType}))
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: ""})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:darksouls2:root",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex game-darksouls2 extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-darksouls2/src",
	})
	r.RegisterExtensionToDo(sdk.ExtensionToDoSpec{
		ID:      "darksouls2-root-query-mod-path",
		Name:    "Dark Souls II root queryModPath metadata",
		Trigger: "source-parity",
		Status:  sdk.CapabilityStatusNotApplicable,
		Message: "Vortex declares queryModPath '.' for root deployment. DMM's extension encodes the equivalent game-root deployment with an empty target root, so the dot-path registration itself is not a separate runtime surface.",
	})
}
