package deps

import (
	"strings"
	"testing"
)

func TestArchiveToolsExposeInstallGuidance(t *testing.T) {
	tools := CheckArchiveTools()
	if len(tools) != 5 {
		t.Fatalf("tool count = %d, want 5", len(tools))
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
		if strings.TrimSpace(tool.PackageName) == "" {
			t.Fatalf("%s missing package name", tool.Command)
		}
		if !strings.Contains(tool.InstallCommand, tool.PackageName) {
			t.Fatalf("%s install command %q does not include package %q", tool.Command, tool.InstallCommand, tool.PackageName)
		}
		if !strings.HasPrefix(tool.DocsURL, "https://") {
			t.Fatalf("%s docs URL = %q", tool.Command, tool.DocsURL)
		}
	}
	if !foundSorter {
		t.Fatal("dmm-loot-sorter helper was not reported")
	}
}
