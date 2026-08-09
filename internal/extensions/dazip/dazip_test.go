package dazip

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestBuildInnerCopiesContentsAndManifestLikeVortex(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "manifest.xml"), `<Manifest><AddInsList><AddInItem UID="demo" /></AddInsList></Manifest>`)
	writeFile(t, filepath.Join(root, "contents", "addins", "DemoModule", "demo_module.erf"), "erf")
	writeFile(t, filepath.Join(root, "contents", "packages", "core", "data.txt"), "data")

	plan, err := BuildInner(installplan.BuildInput{
		GameID:        "dragonage",
		ExtractedRoot: root,
		Installer:     InnerInstaller("test-dazip-inner", "dragonage-documents", 15),
		TargetRootID:  "dragonage-documents",
	})
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan, "addins/DemoModule/manifest.xml")
	assertTarget(t, plan, "addins/DemoModule/demo_module.erf")
	assertTarget(t, plan, "packages/core/data.txt")
}

func TestWillDeployAddInsXMLGeneratesMergedFile(t *testing.T) {
	root := t.TempDir()
	targetRoot := filepath.Join(root, "documents", "BioWare", "Dragon Age")
	if err := os.MkdirAll(filepath.Join(targetRoot, "Settings"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(targetRoot, "Settings", "AddIns.xml"), `<AddInsList><AddInItem UID="existing" /></AddInsList>`)
	manifest := filepath.Join(root, "staging", "manifest.xml")
	writeFile(t, manifest, `<Manifest><AddInsList><AddInItem UID="managed"><Name>Managed</Name></AddInItem></AddInsList></Manifest>`)

	result, err := WillDeployAddInsXML(context.Background(), sdk.EventHandlerInput{
		WorkDir: root,
		Mods: []sdk.DeploymentMod{{
			ID:      10,
			Enabled: true,
			ModType: ModType,
		}},
		Mappings: []deploy.FileMapping{{
			InstalledModID: 10,
			SourcePath:     manifest,
			TargetRoot:     targetRoot,
			TargetRelative: "addins/DemoModule/manifest.xml",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 || result.Mappings[0].TargetRelative != AddInsXMLRel || result.Mappings[0].TargetRoot != targetRoot {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	generated, err := os.ReadFile(result.Mappings[0].SourcePath)
	if err != nil {
		t.Fatal(err)
	}
	body := string(generated)
	if !strings.Contains(body, `UID="existing"`) || !strings.Contains(body, `UID="managed"`) {
		t.Fatalf("generated AddIns.xml = %s", body)
	}
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
