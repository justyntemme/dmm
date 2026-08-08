package metalgearsolidvtpp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
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

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
