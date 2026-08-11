package gamebryo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func TestParseScriptExtenderLogMatchesVortexErrorLines(t *testing.T) {
	log := strings.Join([]string{
		`plugin E:\SteamLibrary\steamapps\common\Skyrim Special Edition\Data\SKSE\Plugins\\Fuz Ro D'oh.dll (00000001 Fuz Ro D'oh 010513CC) reported as incompatible during query`,
		`couldn't load plugin Data\F4SE\Plugins\Example.dll (Error 126)`,
		`couldn't load plugin Data\F4SE\Plugins\BadImage.dll (Error 193)`,
	}, "\n")
	errors := ParseScriptExtenderLog(log)
	if len(errors) != 3 {
		t.Fatalf("errors = %+v", errors)
	}
	if errors[0].DLLName != "Fuz Ro D'oh.dll" || errors[0].Message != "reported as incompatible during query" {
		t.Fatalf("status error = %+v", errors[0])
	}
	if errors[1].DLLName != "Example.dll" || errors[1].Message != "dependent dll not found (code 126)" {
		t.Fatalf("code 126 error = %+v", errors[1])
	}
	if errors[2].DLLName != "BadImage.dll" || errors[2].Message != "not a valid dll (code 193)" {
		t.Fatalf("code 193 error = %+v", errors[2])
	}
}

func TestScriptExtenderErrorTestReadsProtonLogAndAttributesManagedMod(t *testing.T) {
	library := t.TempDir()
	logDir := filepath.Join(library, "steamapps", "compatdata", "377160", "pfx", "drive_c", "users", "steamuser", "Documents", "My Games", "Fallout4", "F4SE")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logDir, "f4se.log"), []byte(`couldn't load plugin Data\F4SE\Plugins\Example.dll (Error 126)`), 0o600); err != nil {
		t.Fatal(err)
	}
	opts := ScriptExtenderErrorTestOptions{
		Logs: []ScriptExtenderLogSpec{{
			Base:     ScriptExtenderLogBaseProtonDocuments,
			MyGames:  "Fallout4",
			Relative: "F4SE/f4se.log",
		}},
		Plugins: []string{"F4SE/Plugins"},
	}
	errors, inspected, err := CheckScriptExtenderLogs(context.Background(), sdk.ExtensionTestInput{
		AppID:       "377160",
		LibraryPath: library,
		Mods: []sdk.DeploymentMod{{
			ID:      10,
			Name:    "Example Plugin",
			Enabled: true,
			Files: []sdk.DeploymentModFile{{
				TargetRelative: "Data/F4SE/Plugins/Example.dll",
			}},
		}},
	}, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(inspected) != 1 || !strings.HasSuffix(inspected[0], "f4se.log") {
		t.Fatalf("inspected = %+v", inspected)
	}
	if len(errors) != 1 || errors[0].DLLName != "Example.dll" || errors[0].ModName != "Example Plugin" {
		t.Fatalf("errors = %+v", errors)
	}
}
