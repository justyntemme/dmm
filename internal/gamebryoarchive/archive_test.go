package gamebryoarchive

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestBA2ReaderListsVortexFixtureArchives(t *testing.T) {
	expected := readJSON[struct {
		GNRL struct {
			Type     string   `json:"type"`
			Version  uint32   `json:"version"`
			FileList []string `json:"fileList"`
		} `json:"gnrl"`
		DX10 struct {
			Type     string   `json:"type"`
			Version  uint32   `json:"version"`
			FileList []string `json:"fileList"`
		} `json:"dx10"`
	}](t, "expected-ba2.json")

	gnrl, err := OpenBA2(testdataPath("test-gnrl.ba2"))
	if err != nil {
		t.Fatal(err)
	}
	if gnrl.Type() != "ba2:"+expected.GNRL.Type || gnrl.Version() != expected.GNRL.Version {
		t.Fatalf("gnrl header = %s/%d", gnrl.Type(), gnrl.Version())
	}
	assertEntryPaths(t, gnrl.List(), expected.GNRL.FileList)

	dx10, err := OpenBA2(testdataPath("test-dx10.ba2"))
	if err != nil {
		t.Fatal(err)
	}
	if dx10.Type() != "ba2:"+expected.DX10.Type || dx10.Version() != expected.DX10.Version {
		t.Fatalf("dx10 header = %s/%d", dx10.Type(), dx10.Version())
	}
	assertEntryPaths(t, dx10.List(), expected.DX10.FileList)
}

func TestBSAHashMatchesVortexBehavior(t *testing.T) {
	if CalculateBSAHash("Test.TXT") != CalculateBSAHash("test.txt") {
		t.Fatal("hash should be case-insensitive")
	}
	if CalculateBSAHash("meshes/weapon.nif") != CalculateBSAHash("meshes\\weapon.nif") {
		t.Fatal("hash should normalize slashes")
	}
	if CalculateBSAHash("test.nif")&0x8000 == 0 {
		t.Fatal("nif flag missing")
	}
	if CalculateBSAHash("test.dds")&0x8080 != 0x8080 {
		t.Fatal("dds flags missing")
	}
	if CalculateBSAHash("test.kf")&0x80 == 0 {
		t.Fatal("kf flag missing")
	}
}

func TestBSAReaderListsAndExtractsVortexFixtureArchives(t *testing.T) {
	expected := readJSON[map[string]struct {
		Version uint32            `json:"version"`
		Files   map[string]string `json:"files"`
	}](t, "expected-bsa.json")
	for _, tc := range []struct {
		key  string
		file string
	}{
		{key: "v103", file: "test-v103.bsa"},
		{key: "v104", file: "test-v104.bsa"},
		{key: "v105", file: "test-v105.bsa"},
	} {
		t.Run(tc.key, func(t *testing.T) {
			want := expected[tc.key]
			archive, err := OpenBSA(testdataPath(tc.file), true)
			if err != nil {
				t.Fatal(err)
			}
			if archive.Version() != want.Version {
				t.Fatalf("version = %d, want %d", archive.Version(), want.Version)
			}
			wantPaths := make([]string, 0, len(want.Files))
			for path := range want.Files {
				wantPaths = append(wantPaths, path)
			}
			assertEntryPaths(t, archive.List(), wantPaths)

			out := t.TempDir()
			if err := archive.ExtractAll(out); err != nil {
				t.Fatal(err)
			}
			for archivePath, wantBody := range want.Files {
				got, err := os.ReadFile(filepath.Join(append([]string{out}, strings.Split(archivePath, "\\")...)...))
				if err != nil {
					t.Fatal(err)
				}
				if string(got) != wantBody {
					t.Fatalf("%s body mismatch", archivePath)
				}
			}
		})
	}
}

func TestBSAWriterCreatesReadableArchives(t *testing.T) {
	root := t.TempDir()
	mesh := filepath.Join(root, "mesh.nif")
	readme := filepath.Join(root, "readme.txt")
	rootFile := filepath.Join(root, "root.xml")
	for path, body := range map[string]string{
		mesh:     "mesh-body",
		readme:   "readme-body",
		rootFile: "<root/>",
	} {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name    string
		version uint32
	}{
		{name: "oblivion", version: bsaVersionOblivion},
		{name: "fallout3", version: bsaVersionFallout3},
		{name: "skyrimse", version: bsaVersionSkyrimSE},
	} {
		t.Run(tc.name, func(t *testing.T) {
			archivePath := filepath.Join(root, tc.name+".bsa")
			if err := WriteBSA(archivePath, tc.version, []BSAWriteFile{
				{ArchivePath: "meshes/weapons/example.nif", SourcePath: mesh},
				{ArchivePath: "docs/readme.txt", SourcePath: readme},
				{ArchivePath: "root.xml", SourcePath: rootFile},
			}); err != nil {
				t.Fatal(err)
			}
			archive, err := OpenBSA(archivePath, true)
			if err != nil {
				t.Fatal(err)
			}
			if archive.Version() != tc.version {
				t.Fatalf("version = %d, want %d", archive.Version(), tc.version)
			}
			assertEntryPaths(t, archive.List(), []string{
				"meshes\\weapons\\example.nif",
				"docs\\readme.txt",
				"root.xml",
			})
			body, err := archive.ReadFile("meshes/weapons/example.nif")
			if err != nil {
				t.Fatal(err)
			}
			if string(body) != "mesh-body" {
				t.Fatalf("mesh body = %q", string(body))
			}
			out := filepath.Join(root, tc.name+"-out")
			if err := archive.ExtractAll(out); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(out, "docs", "readme.txt"))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != "readme-body" {
				t.Fatalf("readme body = %q", string(got))
			}
		})
	}
}

func TestOpenDetectsArchiveType(t *testing.T) {
	reader, err := Open(testdataPath("test-v103.bsa"))
	if err != nil {
		t.Fatal(err)
	}
	if reader.Type() != "bsa" {
		t.Fatalf("type = %q", reader.Type())
	}
	reader, err = Open(testdataPath("test-gnrl.ba2"))
	if err != nil {
		t.Fatal(err)
	}
	if reader.Type() != "ba2:general" {
		t.Fatalf("type = %q", reader.Type())
	}
}

func assertEntryPaths(t *testing.T, entries []Entry, want []string) {
	t.Helper()
	got := make([]string, 0, len(entries))
	for _, entry := range entries {
		got = append(got, strings.ToLower(normalizeArchivePath(entry.Path)))
	}
	wantLower := make([]string, 0, len(want))
	for _, entry := range want {
		wantLower = append(wantLower, strings.ToLower(normalizeArchivePath(entry)))
	}
	slices.Sort(got)
	slices.Sort(wantLower)
	if !slices.Equal(got, wantLower) {
		t.Fatalf("entries = %+v, want %+v", got, wantLower)
	}
}

func readJSON[T any](t *testing.T, name string) T {
	t.Helper()
	var out T
	body, err := os.ReadFile(testdataPath(name))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func testdataPath(name string) string {
	return filepath.Join("testdata", name)
}
