package untitledgoose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/bepinex"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestUntitledGoosePortsEpicBepInExMetadata(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if !summary.AllowNoSteamAppID || summary.Capabilities.GameRegistration == nil {
		t.Fatalf("summary = %+v", summary)
	}
	if summary.Capabilities.GameRegistration.QueryModPath != queryModPath || summary.Capabilities.GameRegistration.ExecutableRelative != executable {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.LauncherRequirements) != 1 || summary.Capabilities.LauncherRequirements[0].AppID != EpicAppID {
		t.Fatalf("launcher requirements = %+v", summary.Capabilities.LauncherRequirements)
	}
	if summary.Capabilities.LauncherRequirements[0].Status != sdk.CapabilityStatusNotApplicable {
		t.Fatalf("launcher requirement status = %+v", summary.Capabilities.LauncherRequirements[0])
	}
	if len(summary.Capabilities.StateMigrations) != 1 || len(summary.Capabilities.ExtensionToDos) != 0 {
		t.Fatalf("migration/todos = %+v / %+v", summary.Capabilities.StateMigrations, summary.Capabilities.ExtensionToDos)
	}
	if len(summary.Capabilities.StateMigrations[0].Commands) != 1 || summary.Capabilities.StateMigrations[0].Commands[0].Command != sdk.StateMigrationCommandPurgeModsInPath {
		t.Fatalf("migration commands = %+v", summary.Capabilities.StateMigrations[0])
	}
	if len(summary.Capabilities.RuntimeRequirements) != 1 {
		t.Fatalf("runtime requirements = %+v", summary.Capabilities.RuntimeRequirements)
	}
	requirement := summary.Capabilities.RuntimeRequirements[0]
	if requirement.Acquisition == nil || requirement.Acquisition.Catalog != "github" || !requirement.Acquisition.AutoAcquire || requirement.Acquisition.SourceModID != bepinex.DefaultRuntimeModID {
		t.Fatalf("runtime acquisition = %+v", requirement.Acquisition)
	}
	if !hasSetupAction(summary.Capabilities.GameSetups, sdk.GameSetupActionEnsureFile, "BepInEx/config/BepInEx.cfg") {
		t.Fatalf("game setup = %+v", summary.Capabilities.GameSetups)
	}
	if !strings.HasPrefix(bepinexConfig, "\ufeff[Caching]") || !strings.Contains(bepinexConfig, "Enabled = true") {
		t.Fatalf("BepInEx config content does not match expected Vortex asset")
	}
}

func TestUntitledGooseBepInExPluginInstallsToPlugins(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CoolGoose", "GooseMod.dll"), "dll")
	writeFile(t, filepath.Join(root, "CoolGoose", "README.md"), "readme")

	plan, err := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())}).BuildInstallPlan(VortexGameID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != pluginModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTarget(t, plan, "BepInEx/plugins/GooseMod.dll")
	assertTarget(t, plan, "BepInEx/plugins/README.md")
}

func assertTarget(t *testing.T, plan installplan.Plan, target string) {
	t.Helper()
	for _, instruction := range plan.Instructions {
		if instruction.TargetRelative == target {
			return
		}
	}
	t.Fatalf("missing target %q in %+v", target, plan.Instructions)
}

func hasSetupAction(setups []gameext.FeatureSummary, kind, relPath string) bool {
	for _, setup := range setups {
		for _, action := range setup.SetupActions {
			if action.Kind == kind && action.RelativePath == relPath {
				return true
			}
		}
	}
	return false
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
