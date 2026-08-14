package deploy

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPurgeChangedTargetsIsAtomic(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	game := filepath.Join(root, "game")
	if err := os.MkdirAll(staging, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"one.txt", "two.txt"} {
		if err := os.WriteFile(filepath.Join(staging, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	plan, err := BuildPlan(staging, game, StrategySymlink, []FileMapping{
		{SourceRelative: "one.txt", TargetRelative: "one.txt"},
		{SourceRelative: "two.txt", TargetRelative: "two.txt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	files, err := Apply(plan)
	if err != nil {
		t.Fatal(err)
	}
	changed := filepath.Join(game, "two.txt")
	if err := os.Remove(changed); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(changed, []byte("external"), 0o600); err != nil {
		t.Fatal(err)
	}

	err = Purge(files)
	var conflict ConflictError
	if !errors.As(err, &conflict) || len(conflict.Issues) != 1 {
		t.Fatalf("purge error = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(game, "one.txt")); err != nil {
		t.Fatalf("unchanged managed target was mutated: %v", err)
	}
	body, err := os.ReadFile(changed)
	if err != nil || string(body) != "external" {
		t.Fatalf("external target = %q, err = %v", body, err)
	}
}

func TestForcePurgeRemovesChangedCopy(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	target := filepath.Join(root, "target.txt")
	if err := os.WriteFile(source, []byte("managed"), 0o600); err != nil {
		t.Fatal(err)
	}
	sum, err := fileSHA256(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("external"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := PurgeWithOptions([]AppliedFile{{SourcePath: source, TargetPath: target, Strategy: StrategyCopy, ChecksumSHA256: sum}}, PurgeOptions{Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Conflicts) != 1 || len(result.Purged) != 1 {
		t.Fatalf("result = %+v", result)
	}
	if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("forced purge target err = %v", err)
	}
}

func TestPurgeRejectsChangedHardlink(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	target := filepath.Join(root, "target.txt")
	if err := os.WriteFile(source, []byte("managed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(source, target); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("steam update"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := Purge([]AppliedFile{{SourcePath: source, TargetPath: target, Strategy: StrategyHardlink}})
	var conflict ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("purge error = %v", err)
	}
}

func TestPurgeRejectsChangedRestoreSource(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "patched.ini")
	restore := filepath.Join(root, "original.ini")
	target := filepath.Join(root, "game.ini")
	for path, body := range map[string]string{source: "patched", restore: "original", target: "patched"} {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	sourceSum, _ := fileSHA256(source)
	restoreSum, _ := fileSHA256(restore)
	file := AppliedFile{SourcePath: source, RestorePath: restore, TargetPath: target, Strategy: StrategyCopy, ChecksumSHA256: sourceSum, RestoreSHA256: restoreSum}
	if err := os.WriteFile(restore, []byte("changed backup"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Purge([]AppliedFile{file}); err == nil {
		t.Fatal("expected changed restore source conflict")
	}
	body, err := os.ReadFile(target)
	if err != nil || string(body) != "patched" {
		t.Fatalf("target = %q, err = %v", body, err)
	}
}

func TestPreparedPurgeRollbackRestoresManagedTarget(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	target := filepath.Join(root, "target.txt")
	if err := os.WriteFile(source, []byte("managed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(source, target); err != nil {
		t.Fatal(err)
	}
	prepared, _, err := PreparePurgeWithOptions([]AppliedFile{{SourcePath: source, TargetPath: target, Strategy: StrategySymlink}}, PurgeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("prepared target err = %v", err)
	}
	if err := prepared.Rollback(); err != nil {
		t.Fatal(err)
	}
	link, err := os.Readlink(target)
	if err != nil || link != source {
		t.Fatalf("restored link = %q, err = %v", link, err)
	}
}

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

func TestApplyAndPurgePatchExistingRestoresOriginal(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	target := filepath.Join(root, "game")
	if err := os.MkdirAll(filepath.Join(staging, "generated"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(staging, "generated", "Fallout4.ini")
	restore := filepath.Join(staging, "generated", "restore-Fallout4.ini")
	targetPath := filepath.Join(target, "Fallout4.ini")
	if err := os.WriteFile(source, []byte("[Archive]\nbInvalidateOlderFiles=1\nsResourceDataDirsFinal=\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(restore, []byte("[Archive]\nSResourceArchiveList=Fallout4 - Misc.ba2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetPath, []byte("[Archive]\nSResourceArchiveList=Fallout4 - Misc.ba2\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(staging, target, StrategySymlink, []FileMapping{{
		SourcePath:     source,
		RestorePath:    restore,
		TargetRelative: "Fallout4.ini",
		TargetPolicy:   TargetPolicyPatchExisting,
		Strategy:       StrategyCopy,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Conflicts) != 0 || len(plan.Actions) != 1 || plan.Actions[0].Operation != "replace" {
		t.Fatalf("plan = %+v", plan)
	}
	applied, err := Apply(plan)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "[Archive]\nbInvalidateOlderFiles=1\nsResourceDataDirsFinal=\n" {
		t.Fatalf("patched target = %q", body)
	}
	if err := Purge(applied); err != nil {
		t.Fatal(err)
	}
	body, err = os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "[Archive]\nSResourceArchiveList=Fallout4 - Misc.ba2\n" {
		t.Fatalf("restored target = %q", body)
	}
}

func TestPurgePatchExistingRefusesChangedTarget(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "patched.ini")
	restore := filepath.Join(root, "restore.ini")
	target := filepath.Join(root, "Fallout4.ini")
	if err := os.WriteFile(source, []byte("patched"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(restore, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("user changed"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := Purge([]AppliedFile{{
		SourcePath:     source,
		RestorePath:    restore,
		TargetPath:     target,
		Strategy:       StrategyCopy,
		ChecksumSHA256: "wrong",
	}})
	if err == nil {
		t.Fatal("expected purge to refuse changed target")
	}
	body, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(body) != "user changed" {
		t.Fatalf("target body = %q", body)
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

func TestApplyPreservesActionModMetadata(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	target := filepath.Join(root, "game")
	if err := os.MkdirAll(filepath.Join(staging, "mod"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging, "mod", "file.txt"), []byte("mod"), 0o600); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(staging, target, StrategySymlink, []FileMapping{{
		SourceRelative: "mod/file.txt",
		TargetRelative: "Mods/file.txt",
		InstalledModID: 42,
		Catalog:        "nexus",
		ModID:          "541",
	}})
	if err != nil {
		t.Fatal(err)
	}
	applied, err := Apply(plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != 1 {
		t.Fatalf("applied = %+v", applied)
	}
	if applied[0].InstalledModID != 42 || applied[0].Catalog != "nexus" || applied[0].ModID != "541" {
		t.Fatalf("applied metadata = %+v", applied[0])
	}
}

func TestApplyCopyPreservesExecutableMode(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	target := filepath.Join(root, "game")
	if err := os.MkdirAll(filepath.Join(staging, "mod"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging, "mod", "launcher"), []byte("mod"), 0o755); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(staging, target, StrategyCopy, []FileMapping{{
		SourceRelative: "mod/launcher",
		TargetRelative: "launcher",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(plan); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(target, "launcher"))
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o755 {
		t.Fatalf("mode = %v, want 0755", got)
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
	replaceIdentity, err := CaptureTargetIdentity(replaceTarget)
	if err != nil {
		t.Fatal(err)
	}
	removeIdentity, err := CaptureTargetIdentity(removeTarget)
	if err != nil {
		t.Fatal(err)
	}

	deployment, err := ApplyPrepared(Plan{Actions: []Action{
		{SourcePath: newSource, TargetPath: replaceTarget, Strategy: StrategySymlink, Operation: "replace", ExistingTarget: &replaceIdentity},
		{SourcePath: addSource, TargetPath: addTarget, Strategy: StrategySymlink, Operation: "add"},
		{TargetPath: removeTarget, Operation: "remove", ExistingTarget: &removeIdentity},
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
	removeIdentity, err := CaptureTargetIdentity(removeTarget)
	if err != nil {
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
		{TargetPath: removeTarget, Operation: "remove", ExistingTarget: &removeIdentity},
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

func TestRepairRestoresChangedManagedCopy(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	target := filepath.Join(root, "Mods", "file.txt")
	if err := os.WriteFile(source, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := Repair([]AppliedFile{{SourcePath: source, TargetPath: target, Strategy: StrategyCopy}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Issues) != 0 || len(result.Repaired) != 1 {
		t.Fatalf("repair result = %+v", result)
	}
	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "source" {
		t.Fatalf("target = %q, want restored source", string(body))
	}
}
