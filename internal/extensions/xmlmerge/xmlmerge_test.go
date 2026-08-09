package xmlmerge

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func TestWillDeployMergesXMLMappingsAndPreservesRestore(t *testing.T) {
	root := t.TempDir()
	gamePath := filepath.Join(root, "game")
	workDir := filepath.Join(root, "work")
	target := filepath.Join(gamePath, "Game", "Loot", "items.xml")
	write(t, target, `<Items><Item id="a" value="1"/></Items>`)
	modA := filepath.Join(root, "stage-a", "items.xml")
	modB := filepath.Join(root, "stage-b", "items.xml")
	write(t, modA, `<Items><Item id="b" value="2"/></Items>`)
	write(t, modB, `<Items><Item id="a" value="3"/></Items>`)

	result, err := WillDeploy(Options{Extensions: []string{".xml", ".mtl"}})(context.Background(), sdk.EventHandlerInput{
		GamePath: gamePath,
		WorkDir:  workDir,
		Mappings: []deploy.FileMapping{
			{SourcePath: filepath.Join(root, "stage-c", "readme.txt"), TargetRelative: "Game/readme.txt"},
			{SourcePath: modA, TargetRelative: "Game/Loot/items.xml", InstalledModID: 1, Priority: 20},
			{SourcePath: modB, TargetRelative: "Game/Loot/items.xml", InstalledModID: 2, Priority: 10},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.ReplaceMappings || len(result.Mappings) != 2 {
		t.Fatalf("result = %+v", result)
	}
	merged := result.Mappings[1]
	if merged.TargetRelative != "Game/Loot/items.xml" || merged.TargetPolicy != deploy.TargetPolicyPatchExisting || merged.RestorePath == "" {
		t.Fatalf("merged mapping = %+v", merged)
	}
	body, err := os.ReadFile(merged.SourcePath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, `id="a"`) || !strings.Contains(text, `value="3"`) || !strings.Contains(text, `id="b"`) {
		t.Fatalf("merged XML = %s", text)
	}
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
