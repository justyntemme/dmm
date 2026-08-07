package halflife2episodes

import (
	hl2 "github.com/justyntemme/decky-mod-manager/internal/extensions/halflife2"
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
	targetRoot   string
}

func Extensions() []sdk.Extension {
	specs := []episodeSpec{
		{id: "halflife2lostcoast", name: "Half-Life 2: Lost Coast", steamAppID: LostCoastAppID, requiredFile: "lostcoast/gameinfo.txt", targetRoot: "lostcoast/custom"},
		{id: "halflife2episodeone", name: "Half-Life 2: Episode One", steamAppID: EpisodeOneAppID, requiredFile: "episodic/gameinfo.txt", targetRoot: "episodic/custom"},
		{id: "halflife2episodetwo", name: "Half-Life 2: Episode Two", steamAppID: EpisodeTwoAppID, requiredFile: "ep2/gameinfo.txt", targetRoot: "ep2/custom"},
	}
	out := make([]sdk.Extension, 0, len(specs))
	for _, spec := range specs {
		out = append(out, extension(spec))
	}
	return out
}

func extension(spec episodeSpec) sdk.Extension {
	return hl2.SourceVPKExtension(hl2.SourceVPKSpec{
		ID:                spec.id,
		Name:              spec.name,
		Version:           "1.1.0-dmm.2",
		BuildID:           "first-party-go",
		SteamAppIDs:       []string{spec.steamAppID},
		NexusDomains:      []string{nexusDomain},
		VortexGameID:      nexusDomain,
		VPKModType:        spec.id + "-vpk",
		TargetRoot:        spec.targetRoot,
		InstallerID:       "vortex:" + spec.id + ":vpk",
		VortexInstallerID: "half-life2-mod",
		RequiredFiles:     []string{spec.requiredFile},
		Sources: []sdk.SourceRef{
			{Name: "Half-Life 2 Vortex extension package v1.1.0", URL: "https://www.nexusmods.com/site/mods/80?tab=files"},
			{Name: "Nexus API domain verification for halflife2", URL: "https://api.nexusmods.com/v1/games/halflife2.json"},
			{Name: "Live Steam Deck Source gameinfo.txt custom search-path verification", URL: "extensionTargets.md#half-life-2-source-search-paths"},
		},
	})
}
