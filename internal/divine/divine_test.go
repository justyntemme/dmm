package divine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestBuildArgsMatchesVortexDivineCoreShape(t *testing.T) {
	args := BuildArgs(ActionExtractPackage, Options{
		Source:      "/tmp/mod.pak",
		Destination: "/tmp/out",
		Expression:  "*/meta.lsx",
	})
	want := []string{
		"--action", "extract-package",
		"--source", "/tmp/mod.pak",
		"--game", "bg3",
		"--loglevel", "error",
		"--destination", "/tmp/out",
		"--expression", "*/meta.lsx",
	}
	if !slices.Equal(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestParsePackageListOutputKeepsNonEmptyLines(t *testing.T) {
	entries := ParsePackageListOutput("\nMods/Example/meta.lsx\t1759\t0\n\nPublic/Data/file.txt\t10\t0\n")
	if len(entries) != 2 || entries[0] != "Mods/Example/meta.lsx\t1759\t0" || entries[1] != "Public/Data/file.txt\t10\t0" {
		t.Fatalf("entries = %+v", entries)
	}
}

func TestRunnerListUsesExecutableAndParsesOutput(t *testing.T) {
	root := t.TempDir()
	exe := fakeExecutable(t, root, "#!/bin/sh\nprintf 'Mods/Example/meta.lsx\\t1759\\t0\\nPublic/Data/file.txt\\t10\\t0\\n'\n")
	result, err := (Runner{ExecutablePath: exe, Timeout: time.Second}).Run(context.Background(), Operation{
		Action: ActionListPackage,
		Options: Options{
			Source:   filepath.Join(root, "Example.pak"),
			LogLevel: "off",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 2 || result.Entries[0] != "Mods/Example/meta.lsx\t1759\t0" {
		t.Fatalf("entries = %+v", result.Entries)
	}
	if got, want := result.Args[1], "list-package"; got != want {
		t.Fatalf("action arg = %q, want %q in %+v", got, want, result.Args)
	}
}

func TestRunnerClassifiesPakInvalidOutput(t *testing.T) {
	root := t.TempDir()
	exe := fakeExecutable(t, root, "#!/bin/sh\nprintf '[ERROR] package malformed\\n'\n")
	_, err := (Runner{ExecutablePath: exe, Timeout: time.Second}).Run(context.Background(), Operation{
		Action:  ActionListPackage,
		Options: Options{Source: filepath.Join(root, "bad.pak")},
	})
	var pakErr PakInvalidError
	if !errors.As(err, &pakErr) {
		t.Fatalf("err = %T %[1]v", err)
	}
}

func TestRunnerClassifiesMissingDotNet(t *testing.T) {
	root := t.TempDir()
	exe := fakeExecutable(t, root, "#!/bin/sh\nprintf 'You must install or update .NET to run this application.\\n' >&2\nexit 1\n")
	_, err := (Runner{ExecutablePath: exe, Timeout: time.Second}).Run(context.Background(), Operation{
		Action:  ActionListPackage,
		Options: Options{Source: filepath.Join(root, "mod.pak")},
	})
	var dotnetErr MissingDotNetError
	if !errors.As(err, &dotnetErr) {
		t.Fatalf("err = %T %[1]v", err)
	}
}

func TestRunnerClassifiesMissingExecutable(t *testing.T) {
	_, err := (Runner{ExecutablePath: filepath.Join(t.TempDir(), "missing-divine")}).Run(context.Background(), Operation{
		Action:  ActionListPackage,
		Options: Options{Source: filepath.Join(t.TempDir(), "mod.pak")},
	})
	var missing ExecMissingError
	if !errors.As(err, &missing) {
		t.Fatalf("err = %T %[1]v", err)
	}
}

func fakeExecutable(t *testing.T, root, body string) string {
	t.Helper()
	path := filepath.Join(root, "divine")
	if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}
