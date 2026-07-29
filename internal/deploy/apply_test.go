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

func TestApplyRestoresReplacedTargetOnFailure(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	oldSource := filepath.Join(root, "old.txt")
	target := filepath.Join(root, "target.txt")
	missing := filepath.Join(root, "missing.txt")
	missingTarget := filepath.Join(root, "missing-target.txt")
	if err := os.WriteFile(source, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldSource, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(oldSource, target); err != nil {
		t.Fatal(err)
	}

	_, err := Apply(Plan{Actions: []Action{
		{SourcePath: source, TargetPath: target, Strategy: StrategySymlink, Operation: "replace"},
		{SourcePath: missing, TargetPath: missingTarget, Strategy: StrategySymlink, Operation: "add"},
	}})
	if err == nil {
		t.Fatal("expected apply failure")
	}
	got, err := os.Readlink(target)
	if err != nil {
		t.Fatal(err)
	}
	if got != oldSource {
		t.Fatalf("restored target = %q, want %q", got, oldSource)
	}
	if _, err := os.Lstat(missingTarget); !os.IsNotExist(err) {
		t.Fatalf("missing target err = %v", err)
	}
}

func TestPreparedApplyRollbackRestoresUncommittedChanges(t *testing.T) {
	root := t.TempDir()
	newSource := filepath.Join(root, "new.txt")
	oldSource := filepath.Join(root, "old.txt")
	addSource := filepath.Join(root, "add.txt")
	replaceTarget := filepath.Join(root, "target.txt")
	addTarget := filepath.Join(root, "added.txt")
	removeTarget := filepath.Join(root, "removed.txt")
	if err := os.WriteFile(newSource, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldSource, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(addSource, []byte("add"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(oldSource, replaceTarget); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(removeTarget, []byte("remove"), 0o600); err != nil {
		t.Fatal(err)
	}

	deployment, err := ApplyPrepared(Plan{Actions: []Action{
		{SourcePath: newSource, TargetPath: replaceTarget, Strategy: StrategySymlink, Operation: "replace"},
		{SourcePath: addSource, TargetPath: addTarget, Strategy: StrategySymlink, Operation: "add"},
		{TargetPath: removeTarget, Operation: "remove"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := deployment.Rollback(); err != nil {
		t.Fatal(err)
	}
	got, err := os.Readlink(replaceTarget)
	if err != nil {
		t.Fatal(err)
	}
	if got != oldSource {
		t.Fatalf("replace target after rollback = %q, want %q", got, oldSource)
	}
	if _, err := os.Lstat(addTarget); !os.IsNotExist(err) {
		t.Fatalf("added target after rollback err = %v", err)
	}
	if body, err := os.ReadFile(removeTarget); err != nil || string(body) != "remove" {
		t.Fatalf("removed target after rollback body=%q err=%v", string(body), err)
	}
}

func TestApplyPreparedWithProgressReportsDeployableActions(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	addTarget := filepath.Join(root, "added.txt")
	removeTarget := filepath.Join(root, "removed.txt")
	conflictTarget := filepath.Join(root, "conflict.txt")
	if err := os.WriteFile(source, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(removeTarget, []byte("remove"), 0o600); err != nil {
		t.Fatal(err)
	}

	var updates []struct {
		completed int
		total     int
		operation string
	}
	deployment, err := ApplyPreparedWithProgress(Plan{Actions: []Action{
		{SourcePath: source, TargetPath: addTarget, Strategy: StrategySymlink, Operation: "add"},
		{TargetPath: conflictTarget, Operation: "add", Conflict: true},
		{TargetPath: removeTarget, Operation: "remove"},
		{Operation: "skip"},
	}}, func(completed, total int, action Action) {
		updates = append(updates, struct {
			completed int
			total     int
			operation string
		}{completed: completed, total: total, operation: action.Operation})
	})
	if err != nil {
		t.Fatal(err)
	}
	deployment.Commit()
	if len(updates) != 2 {
		t.Fatalf("updates = %+v", updates)
	}
	if updates[0].completed != 1 || updates[0].total != 2 || updates[0].operation != "add" {
		t.Fatalf("first update = %+v", updates[0])
	}
	if updates[1].completed != 2 || updates[1].total != 2 || updates[1].operation != "remove" {
		t.Fatalf("last update = %+v", updates[1])
	}
}

func TestVerifyDetectsBrokenSymlink(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	target := filepath.Join(root, "target.txt")
	if err := os.WriteFile(source, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "other.txt"), target); err != nil {
		t.Fatal(err)
	}
	err := Verify([]AppliedFile{{SourcePath: source, TargetPath: target, Strategy: StrategySymlink}})
	if err == nil {
		t.Fatal("expected verify failure")
	}
}

func TestRepairRecreatesMissingSymlink(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	target := filepath.Join(root, "Mods", "file.txt")
	if err := os.WriteFile(source, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := Repair([]AppliedFile{{SourcePath: source, TargetPath: target, Strategy: StrategySymlink}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Issues) != 0 || len(result.Repaired) != 1 {
		t.Fatalf("repair result = %+v", result)
	}
	got, err := os.Readlink(target)
	if err != nil {
		t.Fatal(err)
	}
	if got != source {
		t.Fatalf("target = %q, want %q", got, source)
	}
}

func TestRepairRefusesUnmanagedRegularFile(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	target := filepath.Join(root, "Mods", "file.txt")
	if err := os.WriteFile(source, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("user"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := Repair([]AppliedFile{{SourcePath: source, TargetPath: target, Strategy: StrategySymlink}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Repaired) != 0 || len(result.Issues) != 1 {
		t.Fatalf("repair result = %+v", result)
	}
	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "user" {
		t.Fatalf("target was overwritten: %q", string(body))
	}
}
