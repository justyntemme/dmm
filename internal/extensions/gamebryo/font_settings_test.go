package gamebryo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func TestSkyrimFontSettingsReportsMissingFontConfigReferences(t *testing.T) {
	root := t.TempDir()
	data := filepath.Join(root, "Data")
	if err := os.MkdirAll(filepath.Join(data, "interface"), 0o755); err != nil {
		t.Fatal(err)
	}
	copyFile(t, filepath.Join("..", "..", "gamebryoarchive", "testdata", "test-v103.bsa"), filepath.Join(data, "Skyrim - Interface.bsa"))
	writeFile(t, filepath.Join(data, "interface", "fontconfig.txt"), `fontlib "interface/missing.swf"`)

	test := SkyrimFontSettingsTest(SkyrimFontSettingsOptions{})
	result, err := test.Check(context.Background(), sdk.ExtensionTestInput{AppID: "72850", GamePath: root})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != sdk.HealthCheckStatusFailed || result.Severity != sdk.HealthCheckSeverityError {
		t.Fatalf("result = %+v", result)
	}
	if !strings.Contains(result.Details, "missing.swf") {
		t.Fatalf("details = %q", result.Details)
	}
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	body, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, body, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
