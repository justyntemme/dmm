package metadataonly

import (
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

type Spec struct {
	ID          string
	Name        string
	Version     string
	BuildID     string
	SteamAppIDs []string
	Sources     []sdk.SourceRef
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
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs: spec.SteamAppIDs,
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
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
			Name: "Checked bundled Vortex game extension source; no reviewed " + strings.TrimSpace(gameName) + " handler found",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games",
		},
	}
}
