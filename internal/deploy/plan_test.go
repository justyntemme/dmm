package deploy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestBuildPlanSkipsExistingTargetWhenPolicyKeepsExisting(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	target := filepath.Join(root, "game")
	if err := os.MkdirAll(filepath.Join(staging, "mod"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging, "mod", "steam_appid.txt"), []byte("413150"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "steam_appid.txt"), []byte("413150"), 0o600); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(staging, target, StrategySymlink, []FileMapping{{
		SourceRelative: "mod/steam_appid.txt",
		TargetRelative: "steam_appid.txt",
		TargetPolicy:   TargetPolicyKeepExisting,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Conflicts) != 0 {
		t.Fatalf("conflicts = %+v", plan.Conflicts)
	}
	if len(plan.Actions) != 1 || plan.Actions[0].Operation != "skip" || plan.Actions[0].Conflict {
		t.Fatalf("actions = %+v", plan.Actions)
	}
	if !strings.Contains(plan.Actions[0].ConflictReason, "keeping existing") {
		t.Fatalf("skip reason = %q", plan.Actions[0].ConflictReason)
	}
}

func TestBuildPlanUsesMappingStrategyOverride(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	target := filepath.Join(root, "game")
	if err := os.MkdirAll(filepath.Join(staging, "mod"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging, "mod", "launcher"), []byte("mod"), 0o700); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(staging, target, StrategySymlink, []FileMapping{{
		SourceRelative: "mod/launcher",
		TargetRelative: "launcher",
		Strategy:       StrategyCopy,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) != 1 || plan.Actions[0].Strategy != StrategyCopy {
		t.Fatalf("actions = %+v", plan.Actions)
	}
}

func TestBuildPlanJSONUsesEmptyArrays(t *testing.T) {
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
	}})
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) == "" || !json.Valid(body) {
		t.Fatalf("invalid json: %s", string(body))
	}
	if !strings.Contains(string(body), `"conflicts":[]`) {
		t.Fatalf("expected empty conflicts array, json = %s", string(body))
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

func TestBuildPlanRejectsAbsoluteSourceOutsideStagingRoot(t *testing.T) {
	root := t.TempDir()
	_, err := BuildPlanWithManagedFiles(filepath.Join(root, "staging"), filepath.Join(root, "target"), StrategySymlink, []FileMapping{{
		SourcePath:     filepath.Join(root, "outside", "file.txt"),
		TargetRelative: "Mods/file.txt",
	}}, nil)
	if err == nil {
		t.Fatal("expected outside source error")
	}
}

func TestBuildPlanKeepsExistingOwnedSymlink(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	target := filepath.Join(root, "game")
	if err := os.MkdirAll(filepath.Join(staging, "mod"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(target, "Data"), 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(staging, "mod", "file.txt")
	if err := os.WriteFile(source, []byte("mod"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(source, filepath.Join(target, "Data", "file.txt")); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(staging, target, StrategySymlink, []FileMapping{{
		SourceRelative: "mod/file.txt",
		TargetRelative: "Data/file.txt",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Conflicts) != 0 {
		t.Fatalf("conflicts = %+v", plan.Conflicts)
	}
	if len(plan.Actions) != 1 || plan.Actions[0].Operation != "keep" {
		t.Fatalf("actions = %+v", plan.Actions)
	}
}

func TestBuildPlanReplacesExistingManagedTarget(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	target := filepath.Join(root, "game")
	oldSource := filepath.Join(root, "old", "file.txt")
	newSource := filepath.Join(staging, "mod", "file.txt")
	for _, dir := range []string{filepath.Dir(oldSource), filepath.Dir(newSource), filepath.Join(target, "Mods")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(oldSource, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newSource, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(target, "Mods", "file.txt")
	if err := os.Symlink(oldSource, targetPath); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlanWithManagedFiles(staging, target, StrategySymlink, []FileMapping{{
		SourceRelative: "mod/file.txt",
		TargetRelative: "Mods/file.txt",
	}}, []AppliedFile{{
		SourcePath: oldSource,
		TargetPath: targetPath,
		Strategy:   StrategySymlink,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Conflicts) != 0 {
		t.Fatalf("conflicts = %+v", plan.Conflicts)
	}
	if len(plan.Actions) != 1 || plan.Actions[0].Operation != "replace" {
		t.Fatalf("actions = %+v", plan.Actions)
	}
}

func TestBuildPlanKeepsExistingManagedCopyWhenChecksumMatches(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	target := filepath.Join(root, "game")
	source := filepath.Join(staging, "mod", "launcher")
	targetPath := filepath.Join(target, "launcher")
	for _, dir := range []string{filepath.Dir(source), filepath.Dir(targetPath)} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(source, []byte("launcher"), 0o755); err != nil {
		t.Fatal(err)
	}
	sum, err := fileSHA256(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetPath, []byte("launcher"), 0o755); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlanWithManagedFiles(staging, target, StrategySymlink, []FileMapping{{
		SourceRelative: "mod/launcher",
		TargetRelative: "launcher",
		Strategy:       StrategyCopy,
		ChecksumSHA256: sum,
	}}, []AppliedFile{{
		SourcePath:     source,
		TargetPath:     targetPath,
		Strategy:       StrategyCopy,
		ChecksumSHA256: sum,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Conflicts) != 0 {
		t.Fatalf("conflicts = %+v", plan.Conflicts)
	}
	if len(plan.Actions) != 1 || plan.Actions[0].Operation != "keep" {
		t.Fatalf("actions = %+v", plan.Actions)
	}
}

func TestBuildPlanReplacesExistingManagedCopyWhenChecksumDiffers(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	target := filepath.Join(root, "game")
	source := filepath.Join(staging, "mod", "launcher")
	targetPath := filepath.Join(target, "launcher")
	for _, dir := range []string{filepath.Dir(source), filepath.Dir(targetPath)} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(source, []byte("new launcher"), 0o755); err != nil {
		t.Fatal(err)
	}
	sum, err := fileSHA256(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetPath, []byte("old launcher"), 0o755); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlanWithManagedFiles(staging, target, StrategySymlink, []FileMapping{{
		SourceRelative: "mod/launcher",
		TargetRelative: "launcher",
		Strategy:       StrategyCopy,
		ChecksumSHA256: sum,
	}}, []AppliedFile{{
		SourcePath:     source,
		TargetPath:     targetPath,
		Strategy:       StrategyCopy,
		ChecksumSHA256: sum,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Conflicts) != 0 {
		t.Fatalf("conflicts = %+v", plan.Conflicts)
	}
	if len(plan.Actions) != 1 || plan.Actions[0].Operation != "replace" {
		t.Fatalf("actions = %+v", plan.Actions)
	}
}

func TestBuildPlanRemovesManagedTargetAbsentFromDesiredFiles(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	target := filepath.Join(root, "game")
	if err := os.MkdirAll(staging, 0o700); err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(target, "Mods", "old.txt")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetPath, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlanWithManagedFiles(staging, target, StrategySymlink, nil, []AppliedFile{{
		SourcePath: filepath.Join(root, "old", "old.txt"),
		TargetPath: targetPath,
		Strategy:   StrategySymlink,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Conflicts) != 0 {
		t.Fatalf("conflicts = %+v", plan.Conflicts)
	}
	if len(plan.Actions) != 1 || plan.Actions[0].Operation != "remove" {
		t.Fatalf("actions = %+v", plan.Actions)
	}
}

func TestBuildPlanSkipsDuplicateTargetByPriority(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	target := filepath.Join(root, "game")
	for _, dir := range []string{
		filepath.Join(staging, "low"),
		filepath.Join(staging, "high"),
	} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(staging, "low", "file.txt"), []byte("low"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging, "high", "file.txt"), []byte("high"), 0o600); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(staging, target, StrategySymlink, []FileMapping{
		{SourceRelative: "low/file.txt", TargetRelative: "Mods/file.txt", Priority: 10},
		{SourceRelative: "high/file.txt", TargetRelative: "Mods/file.txt", Priority: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Conflicts) != 0 {
		t.Fatalf("conflicts = %+v", plan.Conflicts)
	}
	if len(plan.Actions) != 2 {
		t.Fatalf("actions = %+v", plan.Actions)
	}
	var add, skip Action
	for _, action := range plan.Actions {
		if action.Operation == "add" {
			add = action
		}
		if action.Operation == "skip" {
			skip = action
		}
	}
	if add.SourcePath != filepath.Join(staging, "high", "file.txt") {
		t.Fatalf("winner = %+v", add)
	}
	if skip.Operation != "skip" || skip.TargetRelative != "Mods/file.txt" {
		t.Fatalf("skip = %+v", skip)
	}
}

func TestBuildPlanDuplicateTargetTieKeepsFirstMapping(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	target := filepath.Join(root, "game")
	if err := os.MkdirAll(filepath.Join(staging, "a"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(staging, "b"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging, "a", "file.txt"), []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging, "b", "file.txt"), []byte("b"), 0o600); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(staging, target, StrategySymlink, []FileMapping{
		{SourceRelative: "a/file.txt", TargetRelative: "Mods/file.txt"},
		{SourceRelative: "b/file.txt", TargetRelative: "Mods/file.txt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var add Action
	for _, action := range plan.Actions {
		if action.Operation == "add" {
			add = action
		}
	}
	if add.SourcePath != filepath.Join(staging, "a", "file.txt") {
		t.Fatalf("winner = %+v", add)
	}
}

func TestApplyHandlesReplaceKeepAndRemove(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	keptSource := filepath.Join(root, "kept.txt")
	target := filepath.Join(root, "target.txt")
	keptTarget := filepath.Join(root, "kept-target.txt")
	removedTarget := filepath.Join(root, "removed.txt")
	if err := os.WriteFile(source, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keptSource, []byte("kept"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(keptSource, keptTarget); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(removedTarget, []byte("remove"), 0o600); err != nil {
		t.Fatal(err)
	}

	applied, err := Apply(Plan{Actions: []Action{
		{SourcePath: source, TargetPath: target, Strategy: StrategySymlink, Operation: "replace"},
		{SourcePath: keptSource, TargetPath: keptTarget, Strategy: StrategySymlink, Operation: "keep"},
		{TargetPath: removedTarget, Operation: "remove"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != 2 {
		t.Fatalf("applied = %+v", applied)
	}
	got, err := os.Readlink(target)
	if err != nil {
		t.Fatal(err)
	}
	if got != source {
		t.Fatalf("replace symlink target = %q", got)
	}
	if _, err := os.Stat(removedTarget); !os.IsNotExist(err) {
		t.Fatalf("removed target err = %v", err)
	}
}
