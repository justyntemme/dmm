package gamehandler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeRequirementsSkipWhenNoEnabledMods(t *testing.T) {
	reqs := RuntimeRequirements(context.Background(), "413150", t.TempDir(), nil)
	if len(reqs) != 0 {
		t.Fatalf("requirements = %+v", reqs)
	}
}

func TestRuntimeRequirementsSkipWhenEnabledModDoesNotNeedRuntime(t *testing.T) {
	reqs := RuntimeRequirements(context.Background(), "413150", t.TempDir(), []RuntimeMod{{Enabled: true, ModType: "generic-files"}})
	if len(reqs) != 0 {
		t.Fatalf("requirements = %+v", reqs)
	}
}

func TestStardewRuntimeRequirementsMissingSMAPI(t *testing.T) {
	reqs := RuntimeRequirements(context.Background(), "413150", t.TempDir(), []RuntimeMod{{Enabled: true, ModType: "stardew-smapi-mod"}})
	if len(reqs) != 1 {
		t.Fatalf("requirements = %+v", reqs)
	}
	if reqs[0].ID != "stardew-smapi" || reqs[0].Status != RequirementMissing || !reqs[0].Required {
		t.Fatalf("requirement = %+v", reqs[0])
	}
}

func TestStardewRuntimeRequirementsDetectSMAPI(t *testing.T) {
	gamePath := t.TempDir()
	smapi := filepath.Join(gamePath, "StardewModdingAPI")
	if err := os.WriteFile(smapi, []byte("smapi"), 0o700); err != nil {
		t.Fatal(err)
	}
	reqs := RuntimeRequirements(context.Background(), "413150", gamePath, []RuntimeMod{{Enabled: true, ModType: "stardew-smapi-mod"}})
	if len(reqs) != 1 {
		t.Fatalf("requirements = %+v", reqs)
	}
	if reqs[0].Status != RequirementOK || len(reqs[0].Details) != 1 {
		t.Fatalf("requirement = %+v", reqs[0])
	}
}

func TestStardewRuntimeRequirementsDetectSteamLaunchOption(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configPath := filepath.Join(home, ".local", "share", "Steam", "userdata", "1", "config", "localconfig.vdf")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatal(err)
	}
	body := `"Apps" { "413150" { "LaunchOptions" "\"/home/deck/.local/share/Steam/steamapps/common/Stardew Valley/StardewModdingAPI\" %command%" } }`
	if err := os.WriteFile(configPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	reqs := RuntimeRequirements(context.Background(), "413150", t.TempDir(), []RuntimeMod{{Enabled: true, ModType: "stardew-smapi-mod"}})
	if len(reqs) != 1 {
		t.Fatalf("requirements = %+v", reqs)
	}
	if reqs[0].Status != RequirementOK || len(reqs[0].Details) != 1 || !strings.Contains(reqs[0].Message, "launch option") {
		t.Fatalf("requirement = %+v", reqs[0])
	}
}

func TestStardewRuntimeRequirementsIgnoresOtherAppLaunchOption(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configPath := filepath.Join(home, ".local", "share", "Steam", "userdata", "1", "config", "localconfig.vdf")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatal(err)
	}
	body := `"Apps" { "413150" { "LaunchOptions" "" } "999999" { "LaunchOptions" "StardewModdingAPI %command%" } }`
	if err := os.WriteFile(configPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	reqs := RuntimeRequirements(context.Background(), "413150", t.TempDir(), []RuntimeMod{{Enabled: true, ModType: "stardew-smapi-mod"}})
	if len(reqs) != 1 {
		t.Fatalf("requirements = %+v", reqs)
	}
	if reqs[0].Status != RequirementMissing {
		t.Fatalf("requirement = %+v", reqs[0])
	}
}

func TestStardewRuntimeRequirementsWarnForMissingRequiredManifestDependency(t *testing.T) {
	reqs := RuntimeRequirements(context.Background(), "413150", t.TempDir(), []RuntimeMod{{
		Enabled: true,
		ModType: "stardew-smapi-mod",
		Metadata: []ModMetadata{{
			Kind:     "smapi-manifest",
			Name:     "Visible Fish",
			UniqueID: "shekurika.WaterFish",
			ContentPackFor: &ModDependency{
				UniqueID:       "Pathoschild.ContentPatcher",
				MinimumVersion: "2.0.0",
				Required:       true,
			},
			Dependencies: []ModDependency{
				{UniqueID: "spacechase0.GenericModConfigMenu", Required: false},
				{UniqueID: "Pathoschild.LookupAnything", MinimumVersion: "1.55.0", Required: true},
			},
		}},
	}})
	if len(reqs) != 3 {
		t.Fatalf("requirements = %+v", reqs)
	}
	var missingFramework, missingDependency bool
	for _, req := range reqs {
		if req.ID == "stardew-mod-dependency:Pathoschild.ContentPatcher" && req.Kind == "mod-dependency" && req.Status == RequirementMissing {
			missingFramework = true
		}
		if req.ID == "stardew-mod-dependency:Pathoschild.LookupAnything" && len(req.Details) == 2 {
			missingDependency = true
		}
		if strings.Contains(req.ID, "GenericModConfigMenu") {
			t.Fatalf("optional dependency should not be required: %+v", req)
		}
	}
	if !missingFramework || !missingDependency {
		t.Fatalf("requirements = %+v", reqs)
	}
}

func TestStardewRuntimeRequirementsSkipInstalledManifestDependency(t *testing.T) {
	reqs := RuntimeRequirements(context.Background(), "413150", t.TempDir(), []RuntimeMod{
		{
			Enabled: true,
			ModType: "stardew-smapi-mod",
			Metadata: []ModMetadata{{
				Kind:     "smapi-manifest",
				Name:     "Visible Fish",
				UniqueID: "shekurika.WaterFish",
				ContentPackFor: &ModDependency{
					UniqueID: "Pathoschild.ContentPatcher",
					Required: true,
				},
			}},
		},
		{
			Enabled: true,
			ModType: "stardew-smapi-mod",
			Metadata: []ModMetadata{{
				Kind:     "smapi-manifest",
				Name:     "Content Patcher",
				UniqueID: "Pathoschild.ContentPatcher",
			}},
		},
	})
	for _, req := range reqs {
		if strings.HasPrefix(req.ID, "stardew-mod-dependency:") {
			t.Fatalf("dependency should be satisfied: %+v", reqs)
		}
	}
}

func TestStardewLaunchBlockReferencesSMAPIHandlesNestedBracesInQuotes(t *testing.T) {
	vdf := `"Apps" { "413150" { "LaunchOptions" "\"{path}/StardewModdingAPI\" %command%" } }`
	if !stardewLaunchBlockReferencesSMAPI(vdf) {
		t.Fatal("expected SMAPI launch option")
	}
}
