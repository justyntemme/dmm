package metalgearsolidvtpp

import (
	"bytes"
	"context"
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
}

func TestSnakeBitePackageIsDetectedAndStagedForPackedArchiveMutation(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "metadata.xml"), `<ModEntry Name="Infinite Heaven" Version="1.0"><MGSVersion Version="1.15.0.0"></MGSVersion><SBVersion Version="0.9.0.0"></SBVersion><QarEntries><QarEntry FilePath="/Assets/tpp/demo/example.fpk" /></QarEntries><FpkEntries><FpkEntry FpkFile="/Assets/tpp/demo/example.fpk" FilePath="/Assets/tpp/demo/file.dat" /></FpkEntries></ModEntry>`)
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
	writeFile(t, filepath.Join(gamePath, filepath.FromSlash(snakeBiteOneArchiveRel)), "not scanned")
	writeFile(t, filepath.Join(stagingPath, "metadata.xml"), `<ModEntry Name="Infinite Heaven" Version="1.0"><MGSVersion Version="1.15.0.0"></MGSVersion><SBVersion Version="0.9.0.0"></SBVersion><QarEntries><QarEntry FilePath="/Assets/tpp/demo/example.fpk" /></QarEntries><FpkEntries><FpkEntry FpkFile="/Assets/tpp/demo/example.fpk" FilePath="/Assets/tpp/demo/mod.bin" /></FpkEntries></ModEntry>`)
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
	if len(result.Mappings) != 1 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	mapping := result.Mappings[0]
	if mapping.TargetRelative != snakeBiteZeroArchiveRel || mapping.TargetPolicy != deploy.TargetPolicyPatchExisting || mapping.Strategy != deploy.StrategyCopy || mapping.RestorePath == "" {
		t.Fatalf("mapping = %+v", mapping)
	}
	fpkData, ok, err := gzs.ExtractQAREntryDataByHash(mapping.SourcePath, gzs.HashFileNameWithExtension(fpkRel))
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
	keepData, ok, err := gzs.ExtractQAREntryDataByHash(mapping.SourcePath, gzs.HashFileNameWithExtension("Assets/tpp/demo/keep.lua"))
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !bytes.Equal(keepData, []byte("keep")) {
		t.Fatalf("unrelated QAR entry = ok %v data %q", ok, keepData)
	}
}

func TestWillDeploySnakeBitePackagesUsesManagedRestoreAsBase(t *testing.T) {
	root := t.TempDir()
	gamePath := filepath.Join(root, "game")
	stagingRoot := filepath.Join(root, "staging")
	stagingPath := filepath.Join(stagingRoot, "mod")
	restorePath := filepath.Join(stagingRoot, "restore", snakeBiteZeroArchiveRel)
	if err := gzs.WriteQAR(restorePath, gzs.QARFile{
		Flags:   3150048,
		Version: 1,
		Entries: []gzs.QAREntry{{FilePath: "Assets/tpp/demo/original.lua", Data: []byte("original")}},
	}); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(gamePath, filepath.FromSlash(snakeBiteZeroArchiveRel)), "patched")
	writeFile(t, filepath.Join(stagingPath, "metadata.xml"), `<ModEntry Name="Loose MGSV File" Version="1.0"><MGSVersion Version="1.15.0.0"></MGSVersion><SBVersion Version="0.9.0.0"></SBVersion><QarEntries><QarEntry FilePath="/Assets/tpp/demo/new.lua" /></QarEntries></ModEntry>`)
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
	if len(result.Mappings) != 1 || result.Mappings[0].RestorePath != restorePath {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	original, ok, err := gzs.ExtractQAREntryDataByHash(result.Mappings[0].SourcePath, gzs.HashFileNameWithExtension("Assets/tpp/demo/original.lua"))
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !bytes.Equal(original, []byte("original")) {
		t.Fatalf("generated archive did not use managed restore as base: ok=%v data=%q", ok, original)
	}
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
