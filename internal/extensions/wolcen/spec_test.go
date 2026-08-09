package wolcen

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestWolcenInstallsGameFolderPayloadAndMergesXML(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "wrapped", "Loot", "items.xml"), `<Items><Item id="mod" /></Items>`)

	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
	plan, err := registry.BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan.Instructions, "Game/Loot/items.xml")

	gamePath := filepath.Join(t.TempDir(), "Wolcen")
	write(t, filepath.Join(gamePath, "Game", "Loot", "items.xml"), `<Items><Item id="base" /></Items>`)
	result, err := registry.RunEventHandlers(context.Background(), SteamAppID, sdk.EventWillDeploy, sdk.EventHandlerInput{
		GamePath: gamePath,
		WorkDir:  filepath.Join(t.TempDir(), "work"),
		Mappings: []deploy.FileMapping{{
			SourcePath:     plan.Instructions[0].SourcePath,
			TargetRelative: plan.Instructions[0].TargetRelative,
			InstalledModID: 1,
			Priority:       10,
			TargetRoot:     plan.Instructions[0].TargetRoot,
			ChecksumSHA256: "",
			Catalog:        "nexus",
			ModID:          "1",
			SourceRelative: plan.Instructions[0].StagingRelative,
			TargetPolicy:   plan.Instructions[0].TargetPolicy,
			Strategy:       deploy.Strategy(plan.Instructions[0].DeployStrategy),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.ReplaceMappings || len(result.Mappings) != 1 {
		t.Fatalf("event result = %+v", result)
	}
	body, err := os.ReadFile(result.Mappings[0].SourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `id="base"`) || !strings.Contains(string(body), `id="mod"`) {
		t.Fatalf("merged body = %s", string(body))
	}
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

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
