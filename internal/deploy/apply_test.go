package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyAndPurgeSymlink(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	target := filepath.Join(root, "game")
	if err := os.MkdirAll(filepath.Join(staging, "mod"), 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(staging, "mod", "file.txt")
	if err := os.WriteFile(source, []byte("mod"), 0o600); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(staging, target, StrategySymlink, []FileMapping{{
		SourceRelative: "mod/file.txt",
		TargetRelative: "Data/file.txt",
	}})
	if err != nil {
		t.Fatal(err)
	}
	applied, err := Apply(plan)
	if err != nil {
		t.Fatal(err)
	}
	targetFile := filepath.Join(target, "Data", "file.txt")
	if st, err := os.Lstat(targetFile); err != nil || st.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected symlink at target, stat=%v err=%v", st, err)
	}
	if err := Purge(applied); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(targetFile); !os.IsNotExist(err) {
		t.Fatalf("expected target removed, err=%v", err)
	}
}

func TestApplySkipsConflicts(t *testing.T) {
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

	plan, err := BuildPlan(staging, target, StrategyCopy, []FileMapping{{
		SourceRelative: "mod/file.txt",
		TargetRelative: "Data/file.txt",
	}})
	if err != nil {
		t.Fatal(err)
	}
	applied, err := Apply(plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != 0 {
		t.Fatalf("applied = %+v", applied)
	}
}
