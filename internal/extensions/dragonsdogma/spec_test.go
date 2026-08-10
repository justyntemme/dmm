package dragonsdogma

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/arctool"
	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
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

func TestExtensionSummaryRecordsARCMergeRuntime(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	summary := gameext.NewRegistry([]gameext.Extension{extension}).ExtensionSummaries()[0]
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != modRoot || summary.Capabilities.GameRegistration.MergeMode != "all" {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.PackedArchiveMutations) != 1 || summary.Capabilities.PackedArchiveMutations[0].RequiresEngine != "mtframework-arc-support" {
		t.Fatalf("packed archive mutations = %+v", summary.Capabilities.PackedArchiveMutations)
	}
	if len(summary.Capabilities.EventHandlers) != 1 || summary.Capabilities.EventHandlers[0].ID != sdk.EventWillDeploy {
		t.Fatalf("event handlers = %+v", summary.Capabilities.EventHandlers)
	}
	if len(summary.Capabilities.StateMigrations) != 1 || summary.Capabilities.StateMigrations[0].Status != sdk.CapabilityStatusReady {
		t.Fatalf("migrations = %+v", summary.Capabilities.StateMigrations)
	}
}

func TestWillDeployARCMergeReplacesModMappingsWithGeneratedArchive(t *testing.T) {
	root := t.TempDir()
	gamePath := filepath.Join(root, "game")
	staging := filepath.Join(root, "staging")
	workDir := filepath.Join(staging, "_generated")
	baseArchive := filepath.Join(gamePath, "nativePC", "rom", "game_main.arc")
	earlyArchive := filepath.Join(staging, "early", "game_main.arc")
	lateArchive := filepath.Join(staging, "late", "game_main.arc")
	writeArchiveFixture(t, baseArchive, map[string]string{
		"base.bin":        "base",
		"shared/item":     "base-item",
		"winner/file.bin": "base-winner",
	})
	writeArchiveFixture(t, earlyArchive, map[string]string{
		"shared/item":    "early-item",
		"early/only.bin": "early",
	})
	writeArchiveFixture(t, lateArchive, map[string]string{
		"shared/item":     "late-item",
		"winner/file.bin": "late-wins",
	})

	result, err := mergeARCMappings(context.Background(), sdk.EventHandlerInput{
		GamePath: gamePath,
		WorkDir:  workDir,
		Mappings: []deploy.FileMapping{
			{SourcePath: earlyArchive, TargetRelative: "nativePC/rom/game_main.arc", InstalledModID: 10, Priority: 10},
			{SourcePath: lateArchive, TargetRelative: "nativePC/rom/game_main.arc", InstalledModID: 20, Priority: 1},
			{SourcePath: filepath.Join(staging, "loose.txt"), TargetRelative: "nativePC/rom/loose.txt", InstalledModID: 30, Priority: 5},
		},
	}, fakeArcRunner{}, arcMergeGroups(sdk.EventHandlerInput{
		GamePath: gamePath,
		Mappings: []deploy.FileMapping{
			{SourcePath: earlyArchive, TargetRelative: "nativePC/rom/game_main.arc", InstalledModID: 10, Priority: 10},
			{SourcePath: lateArchive, TargetRelative: "nativePC/rom/game_main.arc", InstalledModID: 20, Priority: 1},
			{SourcePath: filepath.Join(staging, "loose.txt"), TargetRelative: "nativePC/rom/loose.txt", InstalledModID: 30, Priority: 5},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !result.ReplaceMappings || len(result.Mappings) != 2 {
		t.Fatalf("result = %+v", result)
	}
	var generated deploy.FileMapping
	for _, mapping := range result.Mappings {
		if mapping.TargetRelative == "nativePC/rom/game_main.arc" {
			generated = mapping
		}
	}
	if generated.SourcePath == "" || generated.RestorePath == "" || generated.TargetPolicy != deploy.TargetPolicyPatchExisting || generated.Strategy != deploy.StrategyCopy {
		t.Fatalf("generated mapping = %+v", generated)
	}
	if generated.TargetRoot != gamePath || generated.InstalledModID != 0 || generated.Catalog != "dmm" || generated.ModID != "dragonsdogma-arc-merge" || generated.Priority != -1 {
		t.Fatalf("generated mapping identity = %+v", generated)
	}
	merged := readArchiveFixture(t, generated.SourcePath)
	if merged["base.bin"] != "base" || merged["early/only.bin"] != "early" || merged["shared/item"] != "late-item" || merged["winner/file.bin"] != "late-wins" {
		t.Fatalf("merged archive = %+v", merged)
	}
	restore := readArchiveFixture(t, generated.RestorePath)
	if restore["shared/item"] != "base-item" || restore["winner/file.bin"] != "base-winner" {
		t.Fatalf("restore archive = %+v", restore)
	}
}

func TestWillDeployARCMergeUsesManagedRestoreAsBase(t *testing.T) {
	root := t.TempDir()
	gamePath := filepath.Join(root, "game")
	staging := filepath.Join(root, "staging")
	workDir := filepath.Join(staging, "_generated")
	currentArchive := filepath.Join(gamePath, "nativePC", "rom", "title.arc")
	restoreArchive := filepath.Join(staging, "restore", "title.arc")
	modArchive := filepath.Join(staging, "mod", "title.arc")
	writeArchiveFixture(t, currentArchive, map[string]string{"base.bin": "already-patched"})
	writeArchiveFixture(t, restoreArchive, map[string]string{"base.bin": "original"})
	writeArchiveFixture(t, modArchive, map[string]string{"mod.bin": "enabled"})
	input := sdk.EventHandlerInput{
		GamePath: gamePath,
		WorkDir:  workDir,
		Mappings: []deploy.FileMapping{
			{SourcePath: modArchive, TargetRelative: "nativePC/rom/title.arc", InstalledModID: 10, Priority: 1},
		},
		ManagedFiles: []deploy.AppliedFile{
			{TargetPath: currentArchive, RestorePath: restoreArchive},
		},
	}

	result, err := mergeARCMappings(context.Background(), input, fakeArcRunner{}, arcMergeGroups(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	merged := readArchiveFixture(t, result.Mappings[0].SourcePath)
	if merged["base.bin"] != "original" || merged["mod.bin"] != "enabled" {
		t.Fatalf("merged archive = %+v", merged)
	}
	restore := readArchiveFixture(t, result.Mappings[0].RestorePath)
	if restore["base.bin"] != "original" {
		t.Fatalf("restore archive = %+v", restore)
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

type fakeArcRunner struct{}

func (fakeArcRunner) Run(_ context.Context, op arctool.Operation) (arctool.Result, error) {
	switch op.Type {
	case arctool.OperationExtract:
		files := readArchiveLines(op.ArchivePath)
		for rel, body := range files {
			if err := os.MkdirAll(filepath.Dir(filepath.Join(op.OutputPath, filepath.FromSlash(rel))), 0o700); err != nil {
				return arctool.Result{}, err
			}
			if err := os.WriteFile(filepath.Join(op.OutputPath, filepath.FromSlash(rel)), []byte(body), 0o600); err != nil {
				return arctool.Result{}, err
			}
		}
		return arctool.Result{}, nil
	case arctool.OperationCreate:
		files := map[string]string{}
		err := filepath.WalkDir(op.SourcePath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(op.SourcePath, path)
			if err != nil {
				return err
			}
			body, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			files[filepath.ToSlash(rel)] = string(body)
			return nil
		})
		if err != nil {
			return arctool.Result{}, err
		}
		writeArchiveLines(op.ArchivePath, files)
		return arctool.Result{}, nil
	default:
		return arctool.Result{}, nil
	}
}

func writeArchiveFixture(t *testing.T, path string, files map[string]string) {
	t.Helper()
	writeArchiveLines(path, files)
}

func readArchiveFixture(t *testing.T, path string) map[string]string {
	t.Helper()
	return readArchiveLines(path)
}

func writeArchiveLines(path string, files map[string]string) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		panic(err)
	}
	keys := make([]string, 0, len(files))
	for key := range files {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var body strings.Builder
	for _, key := range keys {
		body.WriteString(key)
		body.WriteByte('=')
		body.WriteString(files[key])
		body.WriteByte('\n')
	}
	if err := os.WriteFile(path, []byte(body.String()), 0o600); err != nil {
		panic(err)
	}
}

func readArchiveLines(path string) map[string]string {
	body, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(body), "\n") {
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if ok {
			out[key] = value
		}
	}
	return out
}
