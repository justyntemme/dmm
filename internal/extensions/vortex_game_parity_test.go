package extensions

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestFirstPartyCoversBundledVortexGameExtensions(t *testing.T) {
	// Source inventory verified from Nexus-Mods/Vortex extensions/games.
	vortexGameIDs := bundledVortexGameIDs()
	aliases := map[string][]string{
		"7daystodie":              {"sevendaystodie"},
		"dmc5":                    {"devilmaycry5"},
		"dragons-dogma":           {"dragonsdogma"},
		"galciv3":                 {"galacticcivilizations3"},
		"kingdomcome-deliverance": {"kingdomcomedeliverance"},
		"masterchiefcollection":   {"halothemasterchiefcollection"},
		"monster-hunter-world":    {"monsterhunterworld"},
		"mount-and-blade":         {"mountandblade"},
		"mount-and-blade2":        {"mountandblade2bannerlord"},
		"neverwinter-nights":      {"nwn", "neverwinter"},
		"neverwinter-nights2":     {"nwn2", "neverwinter2"},
		"oni":                     {"oxygen-not-included", "oxygennotincluded"},
		"re2remake":               {"residentevil22019"},
		"re3remake":               {"residentevil32020"},
		"sims3":                   {"thesims3"},
		"sims4":                   {"thesims4"},
		"sw-kotor":                {"swkotor", "swkotor2", "kotor", "kotor2"},
		"untitledgoose":           {"untitledgoosegame"},
		"vtmbloodlines":           {"vampirebloodlines"},
		"witcher":                 {"witcherlegacy", "witcher"},
		"witcher2":                {"witcherlegacy", "witcher2"},
		"wolcen":                  {"wolcenlordsofmayhem"},
	}

	seen := map[string]gameext.ExtensionSummary{}
	for _, summary := range gameext.NewRegistry(FirstParty()).ExtensionSummaries() {
		if summary.VortexGameID != "" {
			seen[summary.VortexGameID] = summary
		}
		seen[summary.ID] = summary
		for _, domain := range summary.NexusDomains {
			seen[domain] = summary
		}
	}

	for _, vortexID := range vortexGameIDs {
		if _, ok := seen[vortexID]; ok {
			continue
		}
		found := false
		for _, alias := range aliases[vortexID] {
			if _, ok := seen[alias]; ok {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing DMM counterpart for Vortex game extension %q", vortexID)
		}
	}
}

func bundledVortexGameIDs() []string {
	const upstreamGames = "/tmp/dmm-vortex-upstream/extensions/games"
	if entries, err := os.ReadDir(upstreamGames); err == nil {
		ids := make([]string, 0, len(entries))
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !strings.HasPrefix(name, "game-") {
				continue
			}
			if _, err := os.Stat(filepath.Join(upstreamGames, name, "src")); err != nil {
				continue
			}
			ids = append(ids, strings.TrimPrefix(name, "game-"))
		}
		sort.Strings(ids)
		if len(ids) > 0 {
			return ids
		}
	}
	return []string{
		"7daystodie",
		"ahatintime",
		"baldursgate3",
		"battletech",
		"bladeandsorcery",
		"bloodstainedritualofthenight",
		"breakingwheel",
		"codevein",
		"conanexiles",
		"cyberpunk2077",
		"daggerfallunity",
		"darkestdungeon",
		"darksouls",
		"darksouls2",
		"dawnofman",
		"divinityoriginalsin2",
		"dmc5",
		"dragonage",
		"dragonage2",
		"dragons-dogma",
		"elex",
		"enderal",
		"factorio",
		"fallout3",
		"fallout4",
		"fallout4vr",
		"falloutnv",
		"galciv3",
		"gardenpaws",
		"greedfall",
		"grimdawn",
		"grimrock",
		"kenshi",
		"kerbalspaceprogram",
		"kingdomcome-deliverance",
		"masterchiefcollection",
		"microsoftflightsimulator",
		"monster-hunter-world",
		"morrowind",
		"mount-and-blade",
		"mount-and-blade2",
		"nehrim",
		"neverwinter-nights",
		"neverwinter-nights2",
		"nomanssky",
		"oblivion",
		"oni",
		"palworld",
		"pathfinderkingmaker",
		"pathfinderwrathoftherighteous",
		"pillarsofeternity2",
		"prisonarchitect",
		"re2remake",
		"re3remake",
		"rimworld",
		"sekiro",
		"shadowrunreturns",
		"sims3",
		"sims4",
		"skyrim",
		"skyrimse",
		"skyrimvr",
		"spyroreignitedtrilogy",
		"starbound",
		"stardewvalley",
		"starfield",
		"stateofdecay",
		"subnautica",
		"subnauticabelowzero",
		"survivingmars",
		"sw-kotor",
		"teamfortress2",
		"teso",
		"torchlight2",
		"totalwarthreekingdoms",
		"untitledgoose",
		"vtmbloodlines",
		"warthunder",
		"witcher",
		"witcher2",
		"witcher3",
		"wolcen",
		"worldoftanks",
		"x4foundations",
		"xcom2",
		"xrebirth",
	}
}
