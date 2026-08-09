package untitledgoose

import (
	"os"
	"path/filepath"
	"testing"

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
	if len(summary.Capabilities.StateMigrations) != 1 || len(summary.Capabilities.ExtensionToDos) != 1 {
		t.Fatalf("migration/todos = %+v / %+v", summary.Capabilities.StateMigrations, summary.Capabilities.ExtensionToDos)
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

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
