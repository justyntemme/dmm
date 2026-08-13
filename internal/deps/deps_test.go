package deps

import (
	"strings"
	"testing"
)

func TestRuntimeHelpersExposeInstallGuidance(t *testing.T) {
	tools := CheckArchiveTools()
	if len(tools) != 1 {
		t.Fatalf("tool count = %d, want 1", len(tools))
	}
	foundSorter := false
	for _, tool := range tools {
		if strings.TrimSpace(tool.Name) == "" || strings.TrimSpace(tool.Command) == "" {
			t.Fatalf("dependency is missing identity: %+v", tool)
		}
		if strings.TrimSpace(tool.Description) == "" {
			t.Fatalf("%s missing description", tool.Command)
		}
		if tool.Command == "dmm-loot-sorter" {
			foundSorter = true
			if strings.TrimSpace(tool.InstallHint) == "" {
				t.Fatalf("%s missing bundled helper guidance", tool.Command)
			}
			continue
		}
		t.Fatalf("unexpected external dependency after pure-Go archive extraction: %+v", tool)
	}
	if !foundSorter {
		t.Fatal("dmm-loot-sorter helper was not reported")
	}
}
