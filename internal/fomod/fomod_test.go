package fomod

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"
)

func TestParseAndBuildPlanUsesDefaultSelections(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "fomod", "ModuleConfig.xml"), `<?xml version="1.0"?>
<config>
  <moduleName>Example Installer</moduleName>
  <requiredInstallFiles>
    <file source="Core/base.txt" destination="base.txt" />
  </requiredInstallFiles>
  <installSteps>
    <installStep name="Variant">
      <optionalFileGroups>
        <group name="Texture Variant" type="SelectExactlyOne">
          <plugins>
            <plugin name="High">
              <description>High texture option</description>
              <typeDescriptor><type name="Recommended" /></typeDescriptor>
              <files>
                <folder source="Options/High" destination="textures" />
              </files>
            </plugin>
            <plugin name="Low">
              <typeDescriptor><type name="Optional" /></typeDescriptor>
              <files>
                <folder source="Options/Low" destination="textures" />
              </files>
            </plugin>
          </plugins>
        </group>
      </optionalFileGroups>
    </installStep>
  </installSteps>
</config>`)
	writeFile(t, filepath.Join(root, "Core", "base.txt"), "base")
	writeFile(t, filepath.Join(root, "Options", "High", "variant.txt"), "high")
	writeFile(t, filepath.Join(root, "Options", "Low", "variant.txt"), "low")

	installer, err := Parse(root)
	if err != nil {
		t.Fatal(err)
	}
	if installer.Name != "Example Installer" || installer.ModuleConfig != "fomod/ModuleConfig.xml" {
		t.Fatalf("installer = %+v", installer)
	}
	if len(installer.Steps) != 1 || len(installer.Steps[0].Groups) != 1 || len(installer.Steps[0].Groups[0].Plugins) != 2 {
		t.Fatalf("steps = %+v", installer.Steps)
	}
	defaults := DefaultSelections(installer)
	group := installer.Steps[0].Groups[0]
	if len(defaults[group.ID]) != 1 || defaults[group.ID][0] != group.Plugins[0].ID {
		t.Fatalf("defaults = %+v", defaults)
	}

	plan, err := BuildPlan("example", root, installer, defaults, PlanOptions{
		ModType:   "example-mod",
		PlannerID: "example:fomod",
	})
	if err != nil {
		t.Fatal(err)
	}
	targets := map[string]bool{}
	for _, instruction := range plan.Instructions {
		targets[instruction.TargetRelative] = true
	}
	for _, want := range []string{"base.txt", "textures/variant.txt"} {
		if !targets[want] {
			t.Fatalf("missing target %q in %+v", want, plan.Instructions)
		}
	}
	if targets["Options/Low/variant.txt"] {
		t.Fatalf("unselected option was included: %+v", plan.Instructions)
	}
}

func TestBuildPlanAppliesExtensionTargetRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "fomod", "ModuleConfig.xml"), `<config>
  <requiredInstallFiles>
    <file source="loose.esp" />
    <file source="already-data.esp" destination="Data/already-data.esp" />
  </requiredInstallFiles>
</config>`)
	writeFile(t, filepath.Join(root, "loose.esp"), "plugin")
	writeFile(t, filepath.Join(root, "already-data.esp"), "plugin")
	installer, err := Parse(root)
	if err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan("skyrimse", root, installer, nil, PlanOptions{
		ModType:    "skyrimse-data-root",
		PlannerID:  "vortex:skyrimse:fomod",
		TargetRoot: "Data",
	})
	if err != nil {
		t.Fatal(err)
	}
	targets := map[string]string{}
	for _, instruction := range plan.Instructions {
		targets[instruction.StagingRelative] = instruction.TargetRelative
	}
	if targets["loose.esp"] != "Data/loose.esp" {
		t.Fatalf("loose target = %q, plan = %+v", targets["loose.esp"], plan.Instructions)
	}
	if targets["Data/already-data.esp"] != "Data/already-data.esp" {
		t.Fatalf("prefixed target = %q, plan = %+v", targets["Data/already-data.esp"], plan.Instructions)
	}
}

func TestBuildPlanValidatesSelectExactlyOne(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "fomod", "ModuleConfig.xml"), `<config>
  <installSteps>
    <installStep name="Variant">
      <optionalFileGroups>
        <group name="Variant" type="SelectExactlyOne">
          <plugins>
            <plugin name="A"><files><file source="a.txt" /></files></plugin>
            <plugin name="B"><files><file source="b.txt" /></files></plugin>
          </plugins>
        </group>
      </optionalFileGroups>
    </installStep>
  </installSteps>
</config>`)
	writeFile(t, filepath.Join(root, "a.txt"), "a")
	writeFile(t, filepath.Join(root, "b.txt"), "b")
	installer, err := Parse(root)
	if err != nil {
		t.Fatal(err)
	}
	group := installer.Steps[0].Groups[0]
	_, err = BuildPlan("example", root, installer, map[string][]string{
		group.ID: {group.Plugins[0].ID, group.Plugins[1].ID},
	}, PlanOptions{
		ModType:   "example-mod",
		PlannerID: "example:fomod",
	})
	if err == nil {
		t.Fatal("expected selection validation to fail")
	}
}

func TestParseUTF16ModuleConfig(t *testing.T) {
	root := t.TempDir()
	xml := `<config><moduleName>UTF16 Installer</moduleName></config>`
	var data []byte
	data = append(data, 0xFF, 0xFE)
	for _, word := range utf16.Encode([]rune(xml)) {
		var buf [2]byte
		binary.LittleEndian.PutUint16(buf[:], word)
		data = append(data, buf[:]...)
	}
	path := filepath.Join(root, "fomod", "ModuleConfig.xml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	installer, err := Parse(root)
	if err != nil {
		t.Fatal(err)
	}
	if installer.Name != "UTF16 Installer" {
		t.Fatalf("installer = %+v", installer)
	}
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
