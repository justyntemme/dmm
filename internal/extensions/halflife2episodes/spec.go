package halflife2episodes

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/manifestblocked"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	LostCoastAppID  = "340"
	EpisodeOneAppID = "380"
	EpisodeTwoAppID = "420"

	nexusDomain = "halflife2"
)

type episodeSpec struct {
	id           string
	name         string
	steamAppID   string
	requiredFile string
}

func Extensions() []sdk.Extension {
	specs := []episodeSpec{
		{id: "halflife2lostcoast", name: "Half-Life 2: Lost Coast", steamAppID: LostCoastAppID, requiredFile: "lostcoast/gameinfo.txt"},
		{id: "halflife2episodeone", name: "Half-Life 2: Episode One", steamAppID: EpisodeOneAppID, requiredFile: "episodic/gameinfo.txt"},
		{id: "halflife2episodetwo", name: "Half-Life 2: Episode Two", steamAppID: EpisodeTwoAppID, requiredFile: "ep2/gameinfo.txt"},
	}
	out := make([]sdk.Extension, 0, len(specs))
	for _, spec := range specs {
		out = append(out, extension(spec))
	}
	return out
}

func extension(spec episodeSpec) sdk.Extension {
	return manifestblocked.Extension(manifestblocked.Spec{
		ID:                spec.id,
		Name:              spec.name,
		SteamAppIDs:       []string{spec.steamAppID},
		NexusDomains:      []string{nexusDomain},
		VortexGameID:      spec.id,
		UnsupportedReason: spec.name + " shares the live Half-Life 2 install folder, but the inspected Vortex Half-Life 2 extension only registers AppID 220 and routes VPKs to hl2/custom. DMM blocks episode archive installs until episode-specific VPK/custom/sourcemod routing is source-verified.",
		RequiredFiles: []string{
			"hl2_linux",
			spec.requiredFile,
		},
		RequirementName:        spec.name + " install files",
		RequirementMessage:     spec.name + " is missing the expected shared Linux executable or episode gameinfo.txt.",
		RequirementOKMessage:   spec.name + " has the expected shared Linux executable and episode gameinfo.txt.",
		RequirementInstallHint: "Verify " + spec.name + " files in Steam before testing episode mods.",
		Sources: []sdk.SourceRef{
			{Name: "Half-Life 2 Vortex extension package v1.1.0", URL: "https://www.nexusmods.com/site/mods/80?tab=files"},
			{Name: "Nexus API domain verification for halflife2", URL: "https://api.nexusmods.com/v1/games/halflife2.json"},
			{Name: "Live Steam Deck shared Half-Life 2 install-dir verification", URL: "extensionTargets.md#installed-games-snapshot"},
		},
	})
}
