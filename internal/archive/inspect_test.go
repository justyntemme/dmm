package archive

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectZip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mod.zip")
	if err := CreateTestZip(path, map[string]string{
		"modExample/file.txt": "ok",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := Inspect(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Unsafe {
		t.Fatalf("expected safe archive: %+v", got.Warnings)
	}
	if len(got.TopLevelDirs) != 1 || got.TopLevelDirs[0] != "modExample" {
		t.Fatalf("top level dirs = %v", got.TopLevelDirs)
	}
}

func TestInspectExtensionlessZip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dfb0c986-2260-47f9-ae8a-543f4eabe8d4")
	if err := CreateTestZip(path, map[string]string{
		"modExample/file.txt": "ok",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := Inspect(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Format != "zip" {
		t.Fatalf("format = %q", got.Format)
	}
}

func TestInspectZipDetectsTraversal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.zip")
	if err := CreateTestZip(path, map[string]string{
		"../escape.txt": "bad",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := Inspect(path)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Unsafe {
		t.Fatalf("expected unsafe archive")
	}
}

func TestInspectZipDetectsFOMOD(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fomod.zip")
	if err := CreateTestZip(path, map[string]string{
		"fomod/ModuleConfig.xml": "<config />",
		"Data/file.txt":          "ok",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := Inspect(path)
	if err != nil {
		t.Fatal(err)
	}
	if !got.RequiresInstaller || got.InstallerKind != "fomod" {
		t.Fatalf("inspection = %+v", got)
	}
}

func TestInspectZipDetectsNestedFOMOD(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested.zip")
	if err := CreateTestZip(path, map[string]string{
		"Packages/Example.fomod": "nested package",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := Inspect(path)
	if err != nil {
		t.Fatal(err)
	}
	if !got.RequiresInstaller || got.InstallerKind != "nested_fomod" {
		t.Fatalf("inspection = %+v", got)
	}
}

func TestInspectZipPrefersDirectFOMODOverNestedPackage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mixed.zip")
	if err := CreateTestZip(path, map[string]string{
		"Packages/Example.fomod": "nested package",
		"fomod/ModuleConfig.xml": "<config />",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := Inspect(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.InstallerKind != "fomod" {
		t.Fatalf("installer kind = %q", got.InstallerKind)
	}
}

func TestExtractZipWritesSafeEntries(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "mod.zip")
	if err := CreateTestZip(archivePath, map[string]string{
		"Content/file.txt": "ok",
	}); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "staging")
	got, err := Extract(archivePath, dest)
	if err != nil {
		t.Fatal(err)
	}
	if got.Format != "zip" {
		t.Fatalf("format = %q", got.Format)
	}
	body, err := os.ReadFile(filepath.Join(dest, "Content", "file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "ok" {
		t.Fatalf("body = %q", string(body))
	}
}

func TestExtractZipCarriesFOMODDetection(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "fomod.zip")
	if err := CreateTestZip(archivePath, map[string]string{
		"Nested/fomod/info.xml": "<fomod />",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := Extract(archivePath, filepath.Join(dir, "staging"))
	if err != nil {
		t.Fatal(err)
	}
	if !got.RequiresInstaller || got.InstallerKind != "fomod" {
		t.Fatalf("inspection = %+v", got)
	}
}

func TestExtractZipCarriesNestedFOMODDetection(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "nested.zip")
	if err := CreateTestZip(archivePath, map[string]string{
		"Packages/Example.fomod": "nested package",
	}); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(dir, "staging")
	got, err := Extract(archivePath, dest)
	if err != nil {
		t.Fatal(err)
	}
	if !got.RequiresInstaller || got.InstallerKind != "nested_fomod" {
		t.Fatalf("inspection = %+v", got)
	}
	nested, err := FindNestedFOMODArchive(dest)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.ToSlash(nested) != filepath.ToSlash(filepath.Join(dest, "Packages", "Example.fomod")) {
		t.Fatalf("nested path = %q", nested)
	}
}

func TestExtractZipRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "bad.zip")
	if err := CreateTestZip(archivePath, map[string]string{
		"../escape.txt": "bad",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := Extract(archivePath, filepath.Join(dir, "staging")); err == nil {
		t.Fatal("expected traversal archive to fail extraction")
	}
}

func TestExtractContextWithLimitsRejectsEntryCountAndCleansDestination(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "entries.zip")
	if err := CreateTestZip(archivePath, map[string]string{"one.txt": "1", "two.txt": "2"}); err != nil {
		t.Fatal(err)
	}
	limits := DefaultLimits
	limits.MaxEntries = 1
	dest := filepath.Join(dir, "staging")
	_, err := ExtractContextWithLimits(context.Background(), archivePath, dest, limits)
	if !IsLimitExceeded(err) {
		t.Fatalf("error = %v", err)
	}
	if _, statErr := os.Lstat(dest); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("partial destination remains: %v", statErr)
	}
}

func TestExtractContextWithLimitsRejectsEntryAndExpandedSizes(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "sizes.zip")
	if err := CreateTestZip(archivePath, map[string]string{"one.txt": "1234", "two.txt": "5678"}); err != nil {
		t.Fatal(err)
	}

	entryLimits := DefaultLimits
	entryLimits.MaxEntryBytes = 3
	if _, err := ExtractContextWithLimits(context.Background(), archivePath, filepath.Join(dir, "entry"), entryLimits); !IsLimitExceeded(err) {
		t.Fatalf("entry size error = %v", err)
	}

	totalLimits := DefaultLimits
	totalLimits.MaxExpandedBytes = 7
	if _, err := ExtractContextWithLimits(context.Background(), archivePath, filepath.Join(dir, "total"), totalLimits); !IsLimitExceeded(err) {
		t.Fatalf("expanded size error = %v", err)
	}
}

func TestInspectContextWithLimitsRejectsCompressionRatio(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "ratio.zip")
	out, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(out)
	entry, err := zw.CreateHeader(&zip.FileHeader{Name: "zeros.txt", Method: zip.Deflate})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(entry, strings.Repeat("0", 64<<10)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
	limits := DefaultLimits
	limits.MaxCompressionRatio = 2
	if _, err := InspectContextWithLimits(context.Background(), archivePath, limits); !IsLimitExceeded(err) {
		t.Fatalf("compression ratio error = %v", err)
	}
}

func TestExtractContextWithLimitsRejectsPathDepth(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "depth.zip")
	if err := CreateTestZip(archivePath, map[string]string{"one/two/three/file.txt": "ok"}); err != nil {
		t.Fatal(err)
	}
	limits := DefaultLimits
	limits.MaxPathDepth = 3
	if _, err := ExtractContextWithLimits(context.Background(), archivePath, filepath.Join(dir, "staging"), limits); !IsLimitExceeded(err) {
		t.Fatalf("path depth error = %v", err)
	}
}

func TestExtractContextWithLimitsPreservesExistingDestination(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "mod.zip")
	if err := CreateTestZip(archivePath, map[string]string{"file.txt": "new"}); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "staging")
	if err := os.MkdirAll(dest, 0o700); err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(dest, "existing.txt")
	if err := os.WriteFile(existing, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Extract(archivePath, dest); err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Fatalf("error = %v", err)
	}
	body, err := os.ReadFile(existing)
	if err != nil || string(body) != "keep" {
		t.Fatalf("existing destination changed: body=%q err=%v", body, err)
	}
}

func TestExtractContextWithLimitsHonorsCanceledContextWithoutOutput(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "mod.zip")
	if err := CreateTestZip(archivePath, map[string]string{"file.txt": "content"}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	dest := filepath.Join(dir, "staging")
	if _, err := ExtractContextWithLimits(ctx, archivePath, dest, DefaultLimits); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
	if _, err := os.Lstat(dest); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("destination exists after cancellation: %v", err)
	}
}

func TestParse7zListing(t *testing.T) {
	got := parse7zListing(Inspection{ArchivePath: "/tmp/mod.7z", Format: "7z"}, `
Path = /tmp/mod.7z
Type = 7z

Path = Content
Folder = +
Size = 0

Path = Content/file.txt
Folder = -
Size = 12

Path = fomod/ModuleConfig.xml
Folder = -
Size = 9
`)
	if len(got.Entries) != 3 {
		t.Fatalf("entries = %+v", got.Entries)
	}
	if len(got.TopLevelDirs) != 2 || got.TopLevelDirs[0] != "Content" || got.TopLevelDirs[1] != "fomod" {
		t.Fatalf("top level dirs = %+v", got.TopLevelDirs)
	}
	if !got.RequiresInstaller || got.InstallerKind != "fomod" {
		t.Fatalf("installer detection = %+v", got)
	}
}

func TestParse7zListingDetectsNestedFOMOD(t *testing.T) {
	got := parse7zListing(Inspection{ArchivePath: "/tmp/mod.7z", Format: "7z"}, `
Path = /tmp/mod.7z
Type = 7z

Path = Packages
Folder = +

Path = Packages/Example.fomod
Folder = -
Size = 99
`)
	if !got.RequiresInstaller || got.InstallerKind != "nested_fomod" {
		t.Fatalf("installer detection = %+v", got)
	}
}

func TestParse7zListingDetectsTraversal(t *testing.T) {
	got := parse7zListing(Inspection{ArchivePath: "/tmp/mod.7z", Format: "7z"}, `
Path = ../escape.txt
Folder = -
Size = 4
`)
	if !got.Unsafe || len(got.Warnings) == 0 {
		t.Fatalf("inspection = %+v", got)
	}
}
