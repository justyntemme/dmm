package simpleexternal

import (
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

type Spec struct {
	ID           string
	Name         string
	Version      string
	BuildID      string
	SteamAppIDs  []string
	Sources      []sdk.SourceRef
	ModTypeID    string
	InstallerID  string
	InstallerMsg string
}

func Extension(spec Spec) sdk.Extension {
	if strings.TrimSpace(spec.Version) == "" {
		spec.Version = "0.1.0"
	}
	if strings.TrimSpace(spec.BuildID) == "" {
		spec.BuildID = "first-party-go"
	}
	return sdk.Extension{
		ID:      spec.ID,
		Name:    spec.Name,
		Version: spec.Version,
		BuildID: spec.BuildID,
		Register: func(r sdk.Registrar) {
			Register(r, spec)
		},
	}
}

func Register(r sdk.Registrar, spec Spec) {
	modType := strings.TrimSpace(spec.ModTypeID)
	if modType == "" {
		modType = strings.TrimSpace(spec.ID) + "-game-root"
	}
	installerID := strings.TrimSpace(spec.InstallerID)
	if installerID == "" {
		installerID = "dmm:" + strings.TrimSpace(spec.ID) + ":archive-root"
	}
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs: spec.SteamAppIDs,
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{
		ID:         modType,
		TargetRoot: "",
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                installerID,
		VortexInstallerID: installerID,
		Priority:          100,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
		Message:           strings.TrimSpace(spec.InstallerMsg),
	})
	for _, ref := range spec.Sources {
		r.RegisterSource(ref)
	}
}

func ModDBSources(appID, gameName, gameSlug string) []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "ModDB game mods page for " + strings.TrimSpace(gameName),
			URL:  "https://www.moddb.com/games/" + strings.TrimSpace(gameSlug) + "/mods",
		},
		{
			Name: "Steam Deck installed app manifest snapshot for " + strings.TrimSpace(appID),
			URL:  "extensionTargets.md#installed-games-snapshot",
		},
		{
			Name: "No bundled Vortex game extension exists; DMM uses an explicit simple external-source archive-root profile for " + strings.TrimSpace(gameName),
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games",
		},
	}
}
