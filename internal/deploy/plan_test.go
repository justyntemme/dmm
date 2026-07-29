package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildPlanDetectsExistingTarget(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	target := filepath.Join(root, "game")
	if err := os.MkdirAll(filepath.Join(staging, "mod"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(target, "Data"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging, "mod", "file.txt"), []byte("mod"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "Data", "file.txt"), []byte("game"), 0o600); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(staging, target, StrategyHardlink, []FileMapping{{
		SourceRelative: "mod/file.txt",
		TargetRelative: "Data/file.txt",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Conflicts) != 1 {
		t.Fatalf("conflicts = %v", plan.Conflicts)
	}
}

func TestBuildPlanRejectsTraversal(t *testing.T) {
	_, err := BuildPlan("staging", "target", StrategyHardlink, []FileMapping{{
		SourceRelative: "../escape",
		TargetRelative: "Data/file.txt",
	}})
	if err == nil {
		t.Fatal("expected traversal error")
	}
}
