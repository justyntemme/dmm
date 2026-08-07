package halflife2episodes

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestEpisodeExtensionsRegisterSourceBackedVPKInstallers(t *testing.T) {
	extensions := Extensions()
	if len(extensions) != 3 {
		t.Fatalf("extensions = %+v", extensions)
	}
	for _, extensionSpec := range extensions {
		extension := gameext.MustCompileExtension(extensionSpec)
		summary := gameext.NewRegistry([]gameext.Extension{extension}).ExtensionSummaries()[0]
		if summary.Coverage != gameext.CoverageInstaller {
			t.Fatalf("%s coverage = %+v", extension.ID, summary)
		}
		if len(extension.NexusDomains) != 1 || extension.NexusDomains[0] != nexusDomain {
			t.Fatalf("%s nexus domains = %+v", extension.ID, extension.NexusDomains)
		}
		if len(extension.InstallPlan.Installers) != 1 || len(extension.InstallPlan.ModTypes) != 1 {
			t.Fatalf("%s installers = %+v", extension.ID, extension.InstallPlan.Installers)
		}
	}
}

func TestEpisodeVPKInstallersUseEpisodeCustomRoots(t *testing.T) {
	tests := []struct {
		appID string
		want  string
	}{
		{appID: LostCoastAppID, want: "lostcoast/custom/example_dir.vpk"},
		{appID: EpisodeOneAppID, want: "episodic/custom/example_dir.vpk"},
		{appID: EpisodeTwoAppID, want: "ep2/custom/example_dir.vpk"},
	}
	registry := gameext.NewRegistry(mustCompileAll(t, Extensions()))
	for _, tt := range tests {
		t.Run(tt.appID, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "Wrapper", "example_dir.vpk"), "vpk")
			plan, err := registry.BuildInstallPlan(tt.appID, root)
			if err != nil {
				t.Fatal(err)
			}
			if plan.ModType == "" || plan.PlannerID == "" {
				t.Fatalf("plan = %+v", plan)
			}
			assertTarget(t, plan.Instructions, tt.want)
		})
	}
}

func mustCompileAll(t *testing.T, specs []sdk.Extension) []gameext.Extension {
	t.Helper()
	out := make([]gameext.Extension, 0, len(specs))
	for _, spec := range specs {
		out = append(out, gameext.MustCompileExtension(spec))
	}
	return out
}

func assertTarget(t *testing.T, instructions []installplan.Instruction, target string) {
	t.Helper()
	for _, instruction := range instructions {
		if instruction.TargetRelative == target {
			return
		}
	}
	t.Fatalf("missing target %q in %+v", target, instructions)
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
