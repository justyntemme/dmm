package commoninterpreters

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersVortexCommonInterpreters(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{
		gameext.MustCompileExtension(Extension()),
	}).ExtensionSummaries()[0]

	if summary.ID != ID || summary.Kind != gameext.ExtensionKindFramework || summary.Coverage != gameext.CoverageFramework {
		t.Fatalf("summary = %+v", summary)
	}
	if len(summary.Capabilities.Interpreters) != 5 {
		t.Fatalf("interpreters = %+v", summary.Capabilities.Interpreters)
	}
	byID := map[string]gameext.FeatureSummary{}
	for _, interpreter := range summary.Capabilities.Interpreters {
		byID[interpreter.ID] = interpreter
	}
	for _, id := range []string{"jar", "python", "vbs", "cmd", "bat"} {
		if byID[id].ID == "" {
			t.Fatalf("missing interpreter %s in %+v", id, summary.Capabilities.Interpreters)
		}
	}
	if got := byID["cmd"].Platforms; len(got) != 1 || got[0] != "windows" {
		t.Fatalf("cmd platforms = %+v", got)
	}
	if got := byID["jar"].Platforms; len(got) != 2 || got[0] != "linux" || got[1] != "windows" {
		t.Fatalf("jar platforms = %+v", got)
	}
}

func TestCommonInterpretersResolveFromEnvironment(t *testing.T) {
	bin := t.TempDir()
	writeExecutable(t, filepath.Join(bin, "python"))
	writeExecutable(t, filepath.Join(bin, "java"))
	t.Setenv("PATH", bin)
	t.Setenv("JAVA_HOME", "")

	registry := gameext.NewRegistry([]gameext.Extension{
		gameext.MustCompileExtension(Extension()),
	})
	python, ok := registry.ResolveInterpreter("/tmp/install.py", "linux")
	if !ok || python.InterpreterID != "python" || python.Command != filepath.Join(bin, "python") || len(python.Arguments) != 1 || python.Arguments[0] != "/tmp/install.py" {
		t.Fatalf("python resolution = %+v ok=%v", python, ok)
	}
	jar, ok := registry.ResolveInterpreter("/tmp/tool.jar", "linux")
	if !ok || jar.InterpreterID != "jar" || jar.Command != filepath.Join(bin, "java") || len(jar.Arguments) != 2 || jar.Arguments[0] != "-jar" || jar.Arguments[1] != "/tmp/tool.jar" {
		t.Fatalf("jar resolution = %+v ok=%v", jar, ok)
	}
}

func TestJavaInterpreterPrefersJavaHome(t *testing.T) {
	pathBin := t.TempDir()
	javaHome := t.TempDir()
	writeExecutable(t, filepath.Join(pathBin, "java"))
	writeExecutable(t, filepath.Join(javaHome, "bin", "java"))
	t.Setenv("PATH", pathBin)
	t.Setenv("JAVA_HOME", javaHome)

	resolved, ok := gameext.NewRegistry([]gameext.Extension{
		gameext.MustCompileExtension(Extension()),
	}).ResolveInterpreter("/tmp/tool.jar", "linux")
	if !ok || resolved.Command != filepath.Join(javaHome, "bin", "java") {
		t.Fatalf("resolved = %+v ok=%v", resolved, ok)
	}
}

func writeExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatal(err)
	}
}
