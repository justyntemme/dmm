package brawlhalla

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/manifestblocked"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID   = "291550"
	VortexGameID = "brawlhalla"
	Name         = "Brawlhalla"
)

func Extension() sdk.Extension {
	return manifestblocked.Extension(manifestblocked.Spec{
		ID:             VortexGameID,
		Name:           Name,
		SteamAppIDs:    []string{SteamAppID},
		NexusDomains:   []string{VortexGameID},
		VortexGameID:   VortexGameID,
		ResearchReason: "Brawlhalla has a Nexus API-verified game domain, but no Vortex extension source has been verified for its archive layouts yet. DMM does not claim install support until Brawlhalla file replacement roots, anti-cheat risk, and rollback rules are reviewed and encoded in this extension.",
		Sources:        manifestblocked.NexusResearchSources(SteamAppID, Name, VortexGameID),
	})
}
