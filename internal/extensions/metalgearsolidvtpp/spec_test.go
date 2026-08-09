package metalgearsolidvtpp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/gzs"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersSnakeBitePackedArchiveCapability(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	if extension.ID != VortexGameID {
		t.Fatalf("extension id = %q", extension.ID)
	}
	if len(extension.NexusDomains) != 1 || extension.NexusDomains[0] != VortexGameID {
		t.Fatalf("nexus domains = %+v", extension.NexusDomains)
	}
	if len(extension.InstallPlan.Installers) != 1 {
		t.Fatalf("installers = %+v", extension.InstallPlan.Installers)
	}
	installer := extension.InstallPlan.Installers[0]
	if installer.InstructionMode != installplan.InstructionCustom || installer.ModType != snakeBiteModType {
		t.Fatalf("installer = %+v", installer)
	}
	if installer.CustomMatch == nil || installer.CustomBuild == nil {
		t.Fatalf("installer missing custom hooks: %+v", installer)
	}
	if len(extension.InstallPlan.ModTypes) != 1 || extension.InstallPlan.ModTypes[0].DeploymentMode != installplan.ModTypeDeploymentEventHook {
		t.Fatalf("mod types = %+v", extension.InstallPlan.ModTypes)
	}
	if len(extension.PackedArchiveMutations) != 1 {
		t.Fatalf("packed archive mutations = %+v", extension.PackedArchiveMutations)
	}
	mutation := extension.PackedArchiveMutations[0]
	if mutation.PackageFormat != "snakebite-mgsv" || mutation.RequiresEngine != "gzs-qar-fpk" {
		t.Fatalf("packed archive mutation = %+v", mutation)
	}
	if len(extension.GameVersionProviders) != 1 || extension.GameVersionProviders[0].ID != "metalgearsolidvtpp-exe-version" {
		t.Fatalf("game version providers = %+v", extension.GameVersionProviders)
	}
}

func TestSnakeBitePackageIsDetectedAndStagedForPackedArchiveMutation(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "metadata.xml"), `<ModEntry Name="Infinite Heaven" Version="1.0"><MGSVersion Version="0.0.0.0"></MGSVersion><SBVersion Version="0.9.0.0"></SBVersion><QarEntries><QarEntry FilePath="/Assets/tpp/demo/example.fpk" /></QarEntries><FpkEntries><FpkEntry FpkFile="/Assets/tpp/demo/example.fpk" FilePath="/Assets/tpp/demo/file.dat" /></FpkEntries></ModEntry>`)
	writeFile(t, filepath.Join(root, "Assets", "tpp", "demo", "example.fpk"), "payload")

	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	plan, err := registry.Build(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != snakeBiteModType || plan.PlannerID != "snakebite:metalgearsolidvtpp:mgsv-package" {
		t.Fatalf("plan = %+v", plan)
	}
	if len(plan.Metadata) != 1 || plan.Metadata[0].Name != "Infinite Heaven" {
		t.Fatalf("metadata = %+v", plan.Metadata)
	}
	var sawMetadata, sawPayload bool
	for _, instruction := range plan.Instructions {
		switch instruction.StagingRelative {
		case "metadata.xml":
			sawMetadata = true
		case "Assets/tpp/demo/example.fpk":
			sawPayload = true
		}
	}
	if !sawMetadata || !sawPayload {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
}

func TestNonSnakeBiteMetadataDoesNotClaimArchive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "metadata.xml"), `<Mod><Properties><Name>Other</Name></Properties></Mod>`)

	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	_, err := registry.Build(SteamAppID, root)
	if err == nil {
		t.Fatal("expected unsupported archive")
	}
	if strings.Contains(err.Error(), "QAR/FPK") {
		t.Fatalf("non-SnakeBite metadata matched SnakeBite installer: %v", err)
	}
}

func TestRequiredFilesCheck(t *testing.T) {
	root := t.TempDir()
	for _, rel := range requiredGameFiles {
		writeFile(t, filepath.Join(root, filepath.FromSlash(rel)), "game")
	}
	got := checkRequiredGameFiles(context.Background(), root)
	if len(got) != len(requiredGameFiles) {
		t.Fatalf("required details = %+v", got)
	}
}

func TestWillDeploySnakeBitePackagesGeneratesPatchedZeroDat(t *testing.T) {
	root := t.TempDir()
	gamePath := filepath.Join(root, "game")
	stagingPath := filepath.Join(root, "staging", "mod")
	workDir := filepath.Join(root, "staging", "_generated", "event-hooks", SteamAppID, "1", "will-deploy")
	fpkRel := "Assets/tpp/demo/example.fpk"
	baseFPK := writeFPK(t, root, "base.fpk", []gzs.FPKEntry{
		{FilePath: "/Assets/tpp/demo/base.bin", Data: []byte("base")},
	})
	baseZero := gzs.QARFile{
		Flags:   3150048,
		Version: 1,
		Entries: []gzs.QAREntry{
			{FilePath: fpkRel, Hash: gzs.HashFileNameWithExtension(fpkRel), Compressed: true, Data: baseFPK},
			{FilePath: "Assets/tpp/demo/keep.lua", Data: []byte("keep")},
		},
	}
	if err := gzs.WriteQAR(filepath.Join(gamePath, filepath.FromSlash(snakeBiteZeroArchiveRel)), baseZero); err != nil {
		t.Fatal(err)
	}
	if err := gzs.WriteQAR(filepath.Join(gamePath, filepath.FromSlash(snakeBiteOneArchiveRel)), gzs.QARFile{
		Flags:   3150048,
		Version: 1,
		Entries: []gzs.QAREntry{
			{FilePath: "Assets/tpp/demo/one.lua", Data: []byte("one")},
		},
	}); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(stagingPath, "metadata.xml"), `<ModEntry Name="Infinite Heaven" Version="1.0"><MGSVersion Version="0.0.0.0"></MGSVersion><SBVersion Version="0.9.0.0"></SBVersion><QarEntries><QarEntry FilePath="/Assets/tpp/demo/example.fpk" /></QarEntries><FpkEntries><FpkEntry FpkFile="/Assets/tpp/demo/example.fpk" FilePath="/Assets/tpp/demo/mod.bin" /></FpkEntries></ModEntry>`)
	if err := gzs.WriteFPK(filepath.Join(stagingPath, filepath.FromSlash(fpkRel)), gzs.FPKFile{
		Entries: []gzs.FPKEntry{{FilePath: "/Assets/tpp/demo/mod.bin", Data: []byte("mod")}},
	}); err != nil {
		t.Fatal(err)
	}

	result, err := willDeploySnakeBitePackages(context.Background(), sdk.EventHandlerInput{
		AppID:       SteamAppID,
		GamePath:    gamePath,
		ProfileID:   1,
		StagingRoot: filepath.Join(root, "staging"),
		WorkDir:     workDir,
		Mods: []sdk.DeploymentMod{{
			ID:          7,
			Name:        "Infinite Heaven",
			ModType:     snakeBiteModType,
			Enabled:     true,
			Priority:    0,
			StagingPath: stagingPath,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 2 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	zeroMapping := requireMapping(t, result.Mappings, snakeBiteZeroArchiveRel)
	oneMapping := requireMapping(t, result.Mappings, snakeBiteOneArchiveRel)
	if zeroMapping.TargetPolicy != deploy.TargetPolicyPatchExisting || zeroMapping.Strategy != deploy.StrategyCopy || zeroMapping.RestorePath == "" {
		t.Fatalf("zero mapping = %+v", zeroMapping)
	}
	if oneMapping.TargetPolicy != deploy.TargetPolicyPatchExisting || oneMapping.Strategy != deploy.StrategyCopy || oneMapping.RestorePath == "" {
		t.Fatalf("one mapping = %+v", oneMapping)
	}
	fpkData, ok, err := gzs.ExtractQAREntryDataByHash(zeroMapping.SourcePath, gzs.HashFileNameWithExtension(fpkRel))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("generated 00.dat is missing merged FPK")
	}
	mergedFPK, err := readTestFPK(fpkData)
	if err != nil {
		t.Fatal(err)
	}
	got := map[[16]byte][]byte{}
	for _, entry := range mergedFPK.Entries {
		data, err := gzs.ExportFPKEntryData(entry)
		if err != nil {
			t.Fatal(err)
		}
		got[entry.PathMD5] = data
	}
	if !bytes.Equal(got[gzs.FPKPathMD5("/Assets/tpp/demo/base.bin")], []byte("base")) {
		t.Fatalf("base FPK entry was not preserved: %+v", got)
	}
	if !bytes.Equal(got[gzs.FPKPathMD5("/Assets/tpp/demo/mod.bin")], []byte("mod")) {
		t.Fatalf("mod FPK entry was not merged: %+v", got)
	}
	keepData, ok, err := gzs.ExtractQAREntryDataByHash(zeroMapping.SourcePath, gzs.HashFileNameWithExtension("Assets/tpp/demo/keep.lua"))
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatalf("system QAR entry remained in generated 00.dat: %q", keepData)
	}
	keepData, ok, err = gzs.ExtractQAREntryDataByHash(oneMapping.SourcePath, gzs.HashFileNameWithExtension("Assets/tpp/demo/keep.lua"))
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !bytes.Equal(keepData, []byte("keep")) {
		t.Fatalf("system QAR entry was not moved to generated 01.dat: ok %v data %q", ok, keepData)
	}
	oneData, ok, err := gzs.ExtractQAREntryDataByHash(oneMapping.SourcePath, gzs.HashFileNameWithExtension("Assets/tpp/demo/one.lua"))
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !bytes.Equal(oneData, []byte("one")) {
		t.Fatalf("original 01.dat entry was not preserved: ok %v data %q", ok, oneData)
	}
}

func TestWillDeploySnakeBitePackagesUsesManagedRestoreAsBase(t *testing.T) {
	root := t.TempDir()
	gamePath := filepath.Join(root, "game")
	stagingRoot := filepath.Join(root, "staging")
	stagingPath := filepath.Join(stagingRoot, "mod")
	restorePath := filepath.Join(stagingRoot, "restore", snakeBiteZeroArchiveRel)
	restoreOnePath := filepath.Join(stagingRoot, "restore", snakeBiteOneArchiveRel)
	if err := gzs.WriteQAR(restorePath, gzs.QARFile{
		Flags:   3150048,
		Version: 1,
		Entries: []gzs.QAREntry{{FilePath: "Assets/tpp/demo/original.lua", Data: []byte("original")}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := gzs.WriteQAR(restoreOnePath, gzs.QARFile{
		Flags:   3150048,
		Version: 1,
		Entries: []gzs.QAREntry{{FilePath: "Assets/tpp/demo/original-one.lua", Data: []byte("original-one")}},
	}); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(gamePath, filepath.FromSlash(snakeBiteZeroArchiveRel)), "patched")
	writeFile(t, filepath.Join(gamePath, filepath.FromSlash(snakeBiteOneArchiveRel)), "patched-one")
	writeFile(t, filepath.Join(stagingPath, "metadata.xml"), `<ModEntry Name="Loose MGSV File" Version="1.0"><MGSVersion Version="0.0.0.0"></MGSVersion><SBVersion Version="0.9.0.0"></SBVersion><QarEntries><QarEntry FilePath="/Assets/tpp/demo/new.lua" /></QarEntries></ModEntry>`)
	writeFile(t, filepath.Join(stagingPath, "Assets", "tpp", "demo", "new.lua"), "new")
	result, err := willDeploySnakeBitePackages(context.Background(), sdk.EventHandlerInput{
		AppID:       SteamAppID,
		GamePath:    gamePath,
		ProfileID:   1,
		StagingRoot: stagingRoot,
		WorkDir:     filepath.Join(stagingRoot, "_generated", "event-hooks", SteamAppID, "1", "will-deploy"),
		ManagedFiles: []deploy.AppliedFile{{
			TargetPath:  filepath.Join(gamePath, filepath.FromSlash(snakeBiteZeroArchiveRel)),
			RestorePath: restorePath,
		}, {
			TargetPath:  filepath.Join(gamePath, filepath.FromSlash(snakeBiteOneArchiveRel)),
			RestorePath: restoreOnePath,
		}},
		Mods: []sdk.DeploymentMod{{
			ID:          8,
			Name:        "Loose MGSV File",
			ModType:     snakeBiteModType,
			Enabled:     true,
			StagingPath: stagingPath,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	zeroMapping := requireMapping(t, result.Mappings, snakeBiteZeroArchiveRel)
	oneMapping := requireMapping(t, result.Mappings, snakeBiteOneArchiveRel)
	if zeroMapping.RestorePath != restorePath || oneMapping.RestorePath != restoreOnePath {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	newEntry, ok, err := gzs.ExtractQAREntryDataByHash(zeroMapping.SourcePath, gzs.HashFileNameWithExtension("Assets/tpp/demo/new.lua"))
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !bytes.Equal(newEntry, []byte("new")) {
		t.Fatalf("generated archive did not include staged mod entry: ok=%v data=%q", ok, newEntry)
	}
	original, ok, err := gzs.ExtractQAREntryDataByHash(oneMapping.SourcePath, gzs.HashFileNameWithExtension("Assets/tpp/demo/original.lua"))
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !bytes.Equal(original, []byte("original")) {
		t.Fatalf("generated 01.dat did not use managed 00.dat restore as base for moved system file: ok=%v data=%q", ok, original)
	}
}

func TestWillDeploySnakeBitePackagesBlocksConflictingEnabledPackages(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "staging", "first")
	second := filepath.Join(root, "staging", "second")
	writeFile(t, filepath.Join(first, "metadata.xml"), `<ModEntry Name="First" Version="1.0"><MGSVersion Version="0.0.0.0"></MGSVersion><SBVersion Version="0.9.0.0"></SBVersion><QarEntries><QarEntry FilePath="/Assets/tpp/demo/shared.lua" /></QarEntries></ModEntry>`)
	writeFile(t, filepath.Join(second, "metadata.xml"), `<ModEntry Name="Second" Version="1.0"><MGSVersion Version="0.0.0.0"></MGSVersion><SBVersion Version="0.9.0.0"></SBVersion><QarEntries><QarEntry FilePath="/Assets/tpp/demo/shared.lua" /></QarEntries></ModEntry>`)

	_, err := willDeploySnakeBitePackages(context.Background(), sdk.EventHandlerInput{
		AppID:    SteamAppID,
		GamePath: filepath.Join(root, "missing-game-path"),
		Mods: []sdk.DeploymentMod{{
			ID:          1,
			Name:        "First",
			ModType:     snakeBiteModType,
			Enabled:     true,
			StagingPath: first,
		}, {
			ID:          2,
			Name:        "Second",
			ModType:     snakeBiteModType,
			Enabled:     true,
			StagingPath: second,
		}},
	})
	if err == nil {
		t.Fatal("expected SnakeBite package conflict")
	}
	var blockers sdk.BlockingIssuesError
	if !errors.As(err, &blockers) || len(blockers.Issues) != 1 || blockers.Issues[0].Kind != "packed-archive-conflict" || len(blockers.Issues[0].Details) != 1 {
		t.Fatalf("blocking issues = %+v, err = %v", blockers, err)
	}
	if !strings.Contains(err.Error(), "Disable one") || !strings.Contains(err.Error(), "First") || !strings.Contains(err.Error(), "Second") || !strings.Contains(err.Error(), "/Assets/tpp/demo/shared.lua") {
		t.Fatalf("conflict error = %v", err)
	}
}

func TestValidateSnakeBitePackageCompatibilityBlocksOldSnakeBiteMetadata(t *testing.T) {
	pkg := snakeBiteDeployPackage{
		mod: sdk.DeploymentMod{ID: 1, Name: "Old Package"},
		metadata: snakeBiteMetadataProbe{
			Name: "Old Package",
			SBVersion: struct {
				Version string `xml:"Version,attr"`
			}{Version: "0.7.9.0"},
		},
	}
	_, err := validateSnakeBitePackageCompatibility(sdk.EventHandlerInput{}, []snakeBiteDeployPackage{pkg})
	if err == nil {
		t.Fatal("expected old SnakeBite metadata to be blocked")
	}
	if !strings.Contains(err.Error(), "0.8.0.0") || !strings.Contains(err.Error(), "Old Package") {
		t.Fatalf("compatibility error = %v", err)
	}
}

func TestValidateSnakeBitePackageCompatibilitySkipsWildcardMGSVersion(t *testing.T) {
	pkg := snakeBiteDeployPackage{
		mod: sdk.DeploymentMod{ID: 1, Name: "Wildcard Package"},
		metadata: snakeBiteMetadataProbe{
			Name: "Wildcard Package",
			SBVersion: struct {
				Version string `xml:"Version,attr"`
			}{Version: "0.8.0.0"},
			MGSVersion: struct {
				Version string `xml:"Version,attr"`
			}{Version: "0.0.0.0"},
		},
	}
	messages, err := validateSnakeBitePackageCompatibility(sdk.EventHandlerInput{}, []snakeBiteDeployPackage{pkg})
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 0 {
		t.Fatalf("messages = %+v", messages)
	}
}

func TestCompareDottedVersions(t *testing.T) {
	tests := []struct {
		lhs  string
		rhs  string
		want int
	}{
		{lhs: "1.0", rhs: "1.0.0.0", want: 0},
		{lhs: "0.7.9.0", rhs: "0.8.0.0", want: -1},
		{lhs: "1.15.1.0", rhs: "1.15.0.0", want: 1},
	}
	for _, tt := range tests {
		if got := compareDottedVersions(tt.lhs, tt.rhs); got != tt.want {
			t.Fatalf("compareDottedVersions(%q, %q) = %d, want %d", tt.lhs, tt.rhs, got, tt.want)
		}
	}
}

func requireMapping(t *testing.T, mappings []deploy.FileMapping, targetRelative string) deploy.FileMapping {
	t.Helper()
	for _, mapping := range mappings {
		if mapping.TargetRelative == targetRelative {
			return mapping
		}
	}
	t.Fatalf("mapping for %s not found in %+v", targetRelative, mappings)
	return deploy.FileMapping{}
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeFPK(t *testing.T, root, name string, entries []gzs.FPKEntry) []byte {
	t.Helper()
	path := filepath.Join(root, name)
	if err := gzs.WriteFPK(path, gzs.FPKFile{Entries: entries}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func readTestFPK(data []byte) (gzs.FPKFile, error) {
	reader := bytes.NewReader(data)
	return gzs.ReadFPKReader(io.NewSectionReader(reader, 0, int64(len(data))))
}
