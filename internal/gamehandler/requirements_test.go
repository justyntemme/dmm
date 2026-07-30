package gamehandler_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/stardewvalley"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	. "github.com/justyntemme/decky-mod-manager/internal/gamehandler"
)

var stardewRuntimeRegistry = NewRegistry([]GameSpec{gameext.MustCompileExtension(stardewvalley.Extension()).RuntimeRequirements})

func RuntimeRequirements(ctx context.Context, steamAppID, gamePath string, mods []RuntimeMod) []RuntimeRequirement {
	return stardewRuntimeRegistry.RuntimeRequirements(ctx, steamAppID, gamePath, mods)
}

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
	if len(reqs) != 2 {
		t.Fatalf("requirements = %+v", reqs)
	}
	if reqStatus(reqs, "stardew-smapi-installed") != RequirementMissing || reqStatus(reqs, "stardew-smapi-launch") != RequirementMissing {
		t.Fatalf("requirements = %+v", reqs)
	}
}

func TestStardewRuntimeRequirementsDetectSMAPI(t *testing.T) {
	gamePath := t.TempDir()
	smapi := filepath.Join(gamePath, "StardewModdingAPI")
	if err := os.WriteFile(smapi, []byte("smapi"), 0o700); err != nil {
		t.Fatal(err)
	}
	reqs := RuntimeRequirements(context.Background(), "413150", gamePath, []RuntimeMod{{Enabled: true, ModType: "stardew-smapi-mod"}})
	if len(reqs) != 2 {
		t.Fatalf("requirements = %+v", reqs)
	}
	req, ok := reqByID(reqs, "stardew-smapi-installed")
	if !ok || req.Status != RequirementOK || len(req.Details) != 1 {
		t.Fatalf("requirements = %+v", reqs)
	}
}

func TestStardewRuntimeRequirementsDetectSteamLaunchOption(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	gamePath := filepath.Join(home, ".local", "share", "Steam", "steamapps", "common", "Stardew Valley")
	configPath := filepath.Join(home, ".local", "share", "Steam", "userdata", "1", "config", "localconfig.vdf")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatal(err)
	}
	body := `"UserLocalConfigStore" { "Software" { "Valve" { "Steam" { "apps" { "413150" { "LaunchOptions" "\"` + filepath.ToSlash(filepath.Join(gamePath, "StardewModdingAPI")) + `\" %command%" } } } } } }`
	if err := os.WriteFile(configPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	reqs := RuntimeRequirements(context.Background(), "413150", gamePath, []RuntimeMod{{Enabled: true, ModType: "stardew-smapi-mod"}})
	if len(reqs) != 2 {
		t.Fatalf("requirements = %+v", reqs)
	}
	req, ok := reqByID(reqs, "stardew-smapi-launch")
	if !ok || req.Status != RequirementOK || len(req.Details) != 1 || !strings.Contains(req.Message, "launch option") {
		t.Fatalf("requirements = %+v", reqs)
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
	if len(reqs) != 2 {
		t.Fatalf("requirements = %+v", reqs)
	}
	if reqStatus(reqs, "stardew-smapi-launch") != RequirementMissing {
		t.Fatalf("requirements = %+v", reqs)
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
	if len(reqs) != 4 {
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

func reqByID(reqs []RuntimeRequirement, id string) (RuntimeRequirement, bool) {
	for _, req := range reqs {
		if req.ID == id {
			return req, true
		}
	}
	return RuntimeRequirement{}, false
}

func reqStatus(reqs []RuntimeRequirement, id string) RequirementStatus {
	req, ok := reqByID(reqs, id)
	if !ok {
		return ""
	}
	return req.Status
}
