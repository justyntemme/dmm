package dragonsdogma

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestNativePCArchiveCopiesRecognizedSegmentsIntoNativePC(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "rom", "eq", "armor.arc"), "arc")
	writeFile(t, filepath.Join(root, "readme.txt"), "readme")

	plan, err := registry().BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != modType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTargets(t, plan, []string{
		"nativePC/readme.txt",
		"nativePC/rom/eq/armor.arc",
	})
}

func TestInvalidArchiveRequiresUserConfirmationBeforeCopying(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "loose", "file.txt"), "content")

	_, err := registry().BuildInstallPlanWithGamePathArchiveAndSelections(SteamAppID, root, "", "Loose Mod.zip", nil)
	var choice installplan.ChoiceRequiredError
	if err == nil || !strings.Contains(err.Error(), "expected Vortex packaging pattern") {
		t.Fatalf("err = %v", err)
	}
	if !asChoice(err, &choice) || choice.Kind != invalidChoiceKind {
		t.Fatalf("choice = %+v err=%v", choice, err)
	}

	plan, err := registry().BuildInstallPlanWithGamePathArchiveAndSelections(SteamAppID, root, "", "Loose Mod.zip", map[string][]string{
		invalidChoiceGroupID: {invalidChoiceProceed},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != invalidModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTargets(t, plan, []string{"nativePC/loose/file.txt"})
}

func TestExtensionSummaryRecordsBlockedARCMerge(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	summary := gameext.NewRegistry([]gameext.Extension{extension}).ExtensionSummaries()[0]
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != modRoot || summary.Capabilities.GameRegistration.MergeMode != "all" {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.ExtensionToDos) != 1 || summary.Capabilities.ExtensionToDos[0].Status != "blocked" {
		t.Fatalf("todos = %+v", summary.Capabilities.ExtensionToDos)
	}
	if len(summary.Capabilities.StateMigrations) != 1 || summary.Capabilities.StateMigrations[0].Status != "blocked" {
		t.Fatalf("migrations = %+v", summary.Capabilities.StateMigrations)
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

func assertTargets(t *testing.T, plan installplan.Plan, targets []string) {
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

func asChoice(err error, target *installplan.ChoiceRequiredError) bool {
	if err == nil {
		return false
	}
	if choice, ok := err.(installplan.ChoiceRequiredError); ok {
		*target = choice
		return true
	}
	return false
}
