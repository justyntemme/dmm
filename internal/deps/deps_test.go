package deps

import (
	"strings"
	"testing"
)

func TestArchiveToolsExposeInstallGuidance(t *testing.T) {
	tools := CheckArchiveTools()
	if len(tools) != 4 {
		t.Fatalf("tool count = %d, want 4", len(tools))
	}
	for _, tool := range tools {
		if strings.TrimSpace(tool.Name) == "" || strings.TrimSpace(tool.Command) == "" {
			t.Fatalf("dependency is missing identity: %+v", tool)
		}
		if strings.TrimSpace(tool.Description) == "" {
			t.Fatalf("%s missing description", tool.Command)
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
}
