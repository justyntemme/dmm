package darkestdungeon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestProjectArchiveRewritesProjectIntoArchiveNamedModFolder(t *testing.T) {
	root := t.TempDir()
	gamePath := t.TempDir()
	writeFile(t, filepath.Join(root, "wrapped", "project.xml"), `<project><Title>Better Ruins</Title></project>`)
	writeFile(t, filepath.Join(root, "wrapped", "preview_icon.png"), "icon")
	writeFile(t, filepath.Join(root, "wrapped", "raid", "ruins.txt"), "ruins")

	plan, err := registry().BuildInstallPlanWithGamePathArchiveAndSelections(SteamAppID, root, gamePath, "Better Ruins-1-2.zip", nil)
	if err != nil {
		t.Fatal(err)
	}
	assertInstructionTargets(t, plan, []string{
		"mods/BetterRuins/project.xml",
		"mods/BetterRuins/preview_icon.png",
		"mods/BetterRuins/raid/ruins.txt",
	})
	projectInstruction := findInstruction(plan, "mods/BetterRuins/project.xml")
	if projectInstruction.Kind != installplan.InstructionKindGenerateFromGameFile {
		t.Fatalf("project instruction = %+v", projectInstruction)
	}
	if !strings.Contains(projectInstruction.GeneratedDefaultContent, "<Title>Better Ruins</Title>") {
		t.Fatalf("project content = %s", projectInstruction.GeneratedDefaultContent)
	}
	if !strings.Contains(projectInstruction.GeneratedDefaultContent, filepath.ToSlash(filepath.Join(gamePath, "mods", "BetterRuins"))) {
		t.Fatalf("project content missing mod path: %s", projectInstruction.GeneratedDefaultContent)
	}
}

func TestNoProjectHeroArchiveGeneratesProjectAndHeroPaths(t *testing.T) {
	root := t.TempDir()
	gamePath := t.TempDir()
	writeFile(t, filepath.Join(gamePath, "heroes", "crusader", "base.txt"), "base")
	writeFile(t, filepath.Join(root, "Some Hero_A", "Some Hero_A_portrait_roster.png"), "portrait")
	writeFile(t, filepath.Join(root, "Some Hero_A", "anim.png"), "anim")

	plan, err := registry().BuildInstallPlanWithGamePathArchiveAndSelections(SteamAppID, root, gamePath, "Some Hero-2.zip", nil)
	if err != nil {
		t.Fatal(err)
	}
	assertInstructionTargets(t, plan, []string{
		"mods/SomeHero/project.xml",
		"mods/SomeHero/heroes/Some Hero/Some Hero_A/Some Hero_A_portrait_roster.png",
		"mods/SomeHero/heroes/Some Hero/Some Hero_A/anim.png",
	})
}

func TestNoProjectArchiveRequiresGamePathForVortexDirectoryMatching(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "raid", "ruins.txt"), "ruins")
	_, err := registry().BuildInstallPlanWithGamePathArchiveAndSelections(SteamAppID, root, "", "Raid.zip", nil)
	if err == nil || !strings.Contains(err.Error(), "game path is required") {
		t.Fatalf("err = %v", err)
	}
}

func registry() gameext.Registry {
	return gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertInstructionTargets(t *testing.T, plan installplan.Plan, targets []string) {
	t.Helper()
	found := map[string]bool{}
	for _, target := range targets {
		found[target] = false
	}
	for _, instruction := range plan.Instructions {
		if _, ok := found[instruction.TargetRelative]; ok {
			found[instruction.TargetRelative] = true
		}
	}
	for target, ok := range found {
		if !ok {
			t.Fatalf("missing target %q in %+v", target, plan.Instructions)
		}
	}
}

func findInstruction(plan installplan.Plan, target string) installplan.Instruction {
	for _, instruction := range plan.Instructions {
		if instruction.TargetRelative == target {
			return instruction
		}
	}
	return installplan.Instruction{}
}
