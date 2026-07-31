package fomod

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
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

func TestBuildPlanUsesExtensionStopFolders(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "fomod", "ModuleConfig.xml"), `<config>
  <requiredInstallFiles>
    <file source="Wrapper/Data/Plugin.esp" />
    <file source="Wrapper/meshes/weapon.nif" />
  </requiredInstallFiles>
</config>`)
	writeFile(t, filepath.Join(root, "Wrapper", "Data", "Plugin.esp"), "plugin")
	writeFile(t, filepath.Join(root, "Wrapper", "meshes", "weapon.nif"), "mesh")
	installer, err := Parse(root)
	if err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan("fallout4", root, installer, nil, PlanOptions{
		ModType:     "fallout4-data-root",
		PlannerID:   "vortex:fallout4:fomod",
		TargetRoot:  "Data",
		StopFolders: []string{"Data", "meshes"},
	})
	if err != nil {
		t.Fatal(err)
	}
	targets := map[string]string{}
	for _, instruction := range plan.Instructions {
		targets[instruction.StagingRelative] = instruction.TargetRelative
	}
	if targets["Data/Plugin.esp"] != "Data/Plugin.esp" {
		t.Fatalf("data target = %q, plan = %+v", targets["Data/Plugin.esp"], plan.Instructions)
	}
	if targets["meshes/weapon.nif"] != "Data/meshes/weapon.nif" {
		t.Fatalf("meshes target = %q, plan = %+v", targets["meshes/weapon.nif"], plan.Instructions)
	}
}

func TestBuildPlanAppliesConditionalFilesFromSelectedFlags(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "fomod", "ModuleConfig.xml"), `<config>
  <installSteps>
    <installStep name="Variant">
      <optionalFileGroups>
        <group name="Texture Variant" type="SelectExactlyOne">
          <plugins>
            <plugin name="High">
              <conditionFlags><flag name="texture">high</flag></conditionFlags>
              <typeDescriptor><type name="Recommended" /></typeDescriptor>
              <files><file source="high.txt" /></files>
            </plugin>
            <plugin name="Low">
              <conditionFlags><flag name="texture">low</flag></conditionFlags>
              <files><file source="low.txt" /></files>
            </plugin>
          </plugins>
        </group>
      </optionalFileGroups>
    </installStep>
  </installSteps>
  <conditionalFileInstalls>
    <patterns>
      <pattern>
        <dependencies operator="And">
          <flagDependency flag="texture" value="high" />
        </dependencies>
        <files>
          <file source="conditional-high.txt" destination="textures/conditional.txt" />
        </files>
      </pattern>
    </patterns>
  </conditionalFileInstalls>
</config>`)
	writeFile(t, filepath.Join(root, "high.txt"), "high")
	writeFile(t, filepath.Join(root, "low.txt"), "low")
	writeFile(t, filepath.Join(root, "conditional-high.txt"), "conditional")
	installer, err := Parse(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(installer.ConditionalPatterns) != 1 {
		t.Fatalf("conditional patterns = %+v", installer.ConditionalPatterns)
	}

	plan, err := BuildPlan("fallout4", root, installer, nil, PlanOptions{
		ModType:    "fallout4-data-root",
		PlannerID:  "vortex:fallout4:fomod",
		TargetRoot: "Data",
	})
	if err != nil {
		t.Fatal(err)
	}
	targets := map[string]bool{}
	for _, instruction := range plan.Instructions {
		targets[instruction.TargetRelative] = true
	}
	if !targets["Data/high.txt"] || !targets["Data/textures/conditional.txt"] {
		t.Fatalf("conditional files missing from %+v", plan.Instructions)
	}
	if targets["Data/low.txt"] {
		t.Fatalf("unselected files included in %+v", plan.Instructions)
	}
}

func TestBuildPlanAppliesConditionalFilesFromActiveFileDependencies(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "fomod", "ModuleConfig.xml"), `<config>
  <conditionalFileInstalls>
    <patterns>
      <pattern>
        <dependencies operator="And">
          <fileDependency file="Data/SomeOtherMod.esp" state="Active" />
        </dependencies>
        <files>
          <file source="conditional.txt" />
        </files>
      </pattern>
    </patterns>
  </conditionalFileInstalls>
</config>`)
	writeFile(t, filepath.Join(root, "conditional.txt"), "conditional")
	installer, err := Parse(root)
	if err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan("fallout4", root, installer, nil, PlanOptions{
		ModType:    "fallout4-data-root",
		PlannerID:  "vortex:fallout4:fomod",
		TargetRoot: "Data",
		FileStates: map[string]string{
			"Data/SomeOtherMod.esp": "Active",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Instructions) != 1 || plan.Instructions[0].TargetRelative != "Data/conditional.txt" {
		t.Fatalf("conditional dependency did not install expected file: %+v", plan.Instructions)
	}
}

func TestBuildPlanSkipsConditionalFilesFromMissingFileDependencies(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "fomod", "ModuleConfig.xml"), `<config>
  <conditionalFileInstalls>
    <patterns>
      <pattern>
        <dependencies operator="And">
          <fileDependency file="Data/SomeOtherMod.esp" state="Active" />
        </dependencies>
        <files>
          <file source="conditional.txt" />
        </files>
      </pattern>
    </patterns>
  </conditionalFileInstalls>
</config>`)
	writeFile(t, filepath.Join(root, "conditional.txt"), "conditional")
	installer, err := Parse(root)
	if err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan("fallout4", root, installer, nil, PlanOptions{
		ModType:    "fallout4-data-root",
		PlannerID:  "vortex:fallout4:fomod",
		TargetRoot: "Data",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Instructions) != 0 {
		t.Fatalf("missing conditional dependency installed files: %+v", plan.Instructions)
	}
}

func TestBuildPlanAppliesConditionalFilesFromGameDependency(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "fomod", "ModuleConfig.xml"), `<config>
  <conditionalFileInstalls>
    <patterns>
      <pattern>
        <dependencies operator="And">
          <gameDependency version="1.10.2" />
        </dependencies>
        <files>
          <file source="current-game.txt" />
        </files>
      </pattern>
    </patterns>
  </conditionalFileInstalls>
</config>`)
	writeFile(t, filepath.Join(root, "current-game.txt"), "current")
	installer, err := Parse(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(installer.ConditionalPatterns) != 1 || len(installer.ConditionalPatterns[0].Dependencies.GameDependencies) != 1 {
		t.Fatalf("game dependencies were not parsed: %+v", installer.ConditionalPatterns)
	}

	plan, err := BuildPlan("fallout4", root, installer, nil, PlanOptions{
		ModType:     "fallout4-data-root",
		PlannerID:   "vortex:fallout4:fomod",
		TargetRoot:  "Data",
		GameVersion: "1.10.984.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Instructions) != 1 || plan.Instructions[0].TargetRelative != "Data/current-game.txt" {
		t.Fatalf("game dependency did not install expected file: %+v", plan.Instructions)
	}
}

func TestBuildPlanSkipsConditionalFilesWhenGameVersionIsUnavailable(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "fomod", "ModuleConfig.xml"), `<config>
  <conditionalFileInstalls>
    <patterns>
      <pattern>
        <dependencies operator="And">
          <gameDependency version="1.10.2" />
        </dependencies>
        <files>
          <file source="current-game.txt" />
        </files>
      </pattern>
    </patterns>
  </conditionalFileInstalls>
</config>`)
	writeFile(t, filepath.Join(root, "current-game.txt"), "current")
	installer, err := Parse(root)
	if err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan("fallout4", root, installer, nil, PlanOptions{
		ModType:    "fallout4-data-root",
		PlannerID:  "vortex:fallout4:fomod",
		TargetRoot: "Data",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Instructions) != 0 {
		t.Fatalf("missing game version satisfied dependency: %+v", plan.Instructions)
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

func TestBuildPlanEvaluatesDependencyTypeDefaults(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "fomod", "ModuleConfig.xml"), `<config>
  <installSteps>
    <installStep name="Variant">
      <optionalFileGroups>
        <group name="Base" type="SelectExactlyOne">
          <plugins>
            <plugin name="A">
              <conditionFlags><flag name="variant">A</flag></conditionFlags>
              <files><file source="a.txt" /></files>
              <typeDescriptor><type name="Recommended" /></typeDescriptor>
            </plugin>
            <plugin name="B">
              <conditionFlags><flag name="variant">B</flag></conditionFlags>
              <files><file source="b.txt" /></files>
              <typeDescriptor><type name="Optional" /></typeDescriptor>
            </plugin>
          </plugins>
        </group>
        <group name="Patch" type="SelectAny">
          <plugins>
            <plugin name="Patch A">
              <files><file source="patch-a.txt" /></files>
              <typeDescriptor>
                <dependencyType>
                  <defaultType name="NotUsable" />
                  <patterns>
                    <pattern>
                      <dependencies><flagDependency flag="variant" value="A" /></dependencies>
                      <type name="Required" />
                    </pattern>
                  </patterns>
                </dependencyType>
              </typeDescriptor>
            </plugin>
            <plugin name="Patch B">
              <files><file source="patch-b.txt" /></files>
              <typeDescriptor>
                <dependencyType>
                  <defaultType name="NotUsable" />
                  <patterns>
                    <pattern>
                      <dependencies><flagDependency flag="variant" value="B" /></dependencies>
                      <type name="Required" />
                    </pattern>
                  </patterns>
                </dependencyType>
              </typeDescriptor>
            </plugin>
          </plugins>
        </group>
      </optionalFileGroups>
    </installStep>
  </installSteps>
</config>`)
	writeFile(t, filepath.Join(root, "a.txt"), "a")
	writeFile(t, filepath.Join(root, "b.txt"), "b")
	writeFile(t, filepath.Join(root, "patch-a.txt"), "patch-a")
	writeFile(t, filepath.Join(root, "patch-b.txt"), "patch-b")

	installer, err := Parse(root)
	if err != nil {
		t.Fatal(err)
	}
	defaults := DefaultSelections(installer)
	patchGroup := installer.Steps[0].Groups[1]
	if len(defaults[patchGroup.ID]) != 1 || defaults[patchGroup.ID][0] != patchGroup.Plugins[0].ID {
		t.Fatalf("defaults = %+v", defaults)
	}
	evaluated := EvaluatedInstaller(installer, defaults, PlanOptions{})
	evaluatedPatchGroup := evaluated.Steps[0].Groups[1]
	if evaluatedPatchGroup.Plugins[0].Type != "NotUsable" || evaluatedPatchGroup.Plugins[1].Type != "NotUsable" {
		t.Fatalf("source dependencyType defaults were mutated: %+v", evaluatedPatchGroup.Plugins)
	}
	if evaluatedPatchGroup.Plugins[0].EffectiveType != "Required" || evaluatedPatchGroup.Plugins[1].EffectiveType != "NotUsable" {
		t.Fatalf("evaluated dependencyType = %+v", evaluatedPatchGroup.Plugins)
	}
	variantGroup := installer.Steps[0].Groups[0]
	evaluatedB := EvaluatedInstaller(evaluated, map[string][]string{
		variantGroup.ID: {variantGroup.Plugins[1].ID},
	}, PlanOptions{})
	evaluatedBPatchGroup := evaluatedB.Steps[0].Groups[1]
	if evaluatedBPatchGroup.Plugins[0].EffectiveType != "NotUsable" || evaluatedBPatchGroup.Plugins[1].EffectiveType != "Required" {
		t.Fatalf("reevaluated dependencyType after changed selection = %+v", evaluatedBPatchGroup.Plugins)
	}
	plan, err := BuildPlan("fallout4", root, installer, nil, PlanOptions{
		ModType:    "fallout4-data-root",
		PlannerID:  "vortex:fallout4:fomod",
		TargetRoot: "Data",
	})
	if err != nil {
		t.Fatal(err)
	}
	targets := instructionTargets(plan)
	if !targets["Data/a.txt"] || !targets["Data/patch-a.txt"] || targets["Data/patch-b.txt"] {
		t.Fatalf("dynamic dependencyType targets = %+v", plan.Instructions)
	}
}

func TestBuildPlanRejectsNotUsableDependencyTypeSelection(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "fomod", "ModuleConfig.xml"), `<config>
  <installSteps>
    <installStep name="Variant">
      <optionalFileGroups>
        <group name="Patch" type="SelectAny">
          <plugins>
            <plugin name="Patch">
              <files><file source="patch.txt" /></files>
              <typeDescriptor>
                <dependencyType>
                  <defaultType name="NotUsable" />
                  <patterns>
                    <pattern>
                      <dependencies><flagDependency flag="missing" value="yes" /></dependencies>
                      <type name="Required" />
                    </pattern>
                  </patterns>
                </dependencyType>
              </typeDescriptor>
            </plugin>
          </plugins>
        </group>
      </optionalFileGroups>
    </installStep>
  </installSteps>
</config>`)
	writeFile(t, filepath.Join(root, "patch.txt"), "patch")
	installer, err := Parse(root)
	if err != nil {
		t.Fatal(err)
	}
	group := installer.Steps[0].Groups[0]
	_, err = BuildPlan("fallout4", root, installer, map[string][]string{
		group.ID: {group.Plugins[0].ID},
	}, PlanOptions{
		ModType:    "fallout4-data-root",
		PlannerID:  "vortex:fallout4:fomod",
		TargetRoot: "Data",
	})
	if err == nil {
		t.Fatal("expected NotUsable selection to fail")
	}
}

func TestBuildPlanSkipsInvisibleSteps(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "fomod", "ModuleConfig.xml"), `<config>
  <installSteps>
    <installStep name="Base">
      <optionalFileGroups>
        <group name="Base" type="SelectExactlyOne">
          <plugins>
            <plugin name="A">
              <conditionFlags><flag name="variant">A</flag></conditionFlags>
              <files><file source="a.txt" /></files>
              <typeDescriptor><type name="Recommended" /></typeDescriptor>
            </plugin>
            <plugin name="B">
              <conditionFlags><flag name="variant">B</flag></conditionFlags>
              <files><file source="b.txt" /></files>
            </plugin>
          </plugins>
        </group>
      </optionalFileGroups>
    </installStep>
    <installStep name="Patch">
      <visible><flagDependency flag="variant" value="B" /></visible>
      <optionalFileGroups>
        <group name="Patch" type="SelectAny">
          <plugins>
            <plugin name="Patch"><files><file source="patch.txt" /></files><typeDescriptor><type name="Recommended" /></typeDescriptor></plugin>
          </plugins>
        </group>
      </optionalFileGroups>
    </installStep>
  </installSteps>
</config>`)
	writeFile(t, filepath.Join(root, "a.txt"), "a")
	writeFile(t, filepath.Join(root, "b.txt"), "b")
	writeFile(t, filepath.Join(root, "patch.txt"), "patch")
	installer, err := Parse(root)
	if err != nil {
		t.Fatal(err)
	}
	if installer.Steps[1].Visibility == nil {
		t.Fatalf("visibility was not parsed: %+v", installer.Steps[1])
	}
	defaults := DefaultSelections(installer)
	if _, ok := defaults[installer.Steps[1].Groups[0].ID]; ok {
		t.Fatalf("hidden step received default selections: %+v", defaults)
	}
	evaluated := EvaluatedInstaller(installer, defaults, PlanOptions{})
	if !evaluated.Steps[0].Visible || evaluated.Steps[1].Visible {
		t.Fatalf("visible flags = %+v", evaluated.Steps)
	}
	plan, err := BuildPlan("fallout4", root, installer, nil, PlanOptions{
		ModType:    "fallout4-data-root",
		PlannerID:  "vortex:fallout4:fomod",
		TargetRoot: "Data",
	})
	if err != nil {
		t.Fatal(err)
	}
	targets := instructionTargets(plan)
	if !targets["Data/a.txt"] || targets["Data/patch.txt"] {
		t.Fatalf("invisible step targets = %+v", plan.Instructions)
	}
}

func TestBuildPlanHonorsAlwaysInstallAndInstallIfUsable(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "fomod", "ModuleConfig.xml"), `<config>
  <installSteps>
    <installStep name="Options">
      <optionalFileGroups>
        <group name="Options" type="SelectAny">
          <plugins>
            <plugin name="Unselected">
              <files>
                <file source="always.txt" alwaysInstall="true" />
                <file source="usable.txt" installIfUsable="true" />
                <file source="normal.txt" />
              </files>
              <typeDescriptor><type name="Optional" /></typeDescriptor>
            </plugin>
            <plugin name="Blocked">
              <files><file source="blocked.txt" installIfUsable="true" /></files>
              <typeDescriptor><type name="NotUsable" /></typeDescriptor>
            </plugin>
          </plugins>
        </group>
      </optionalFileGroups>
    </installStep>
  </installSteps>
</config>`)
	for _, name := range []string{"always.txt", "usable.txt", "normal.txt", "blocked.txt"} {
		writeFile(t, filepath.Join(root, name), name)
	}
	installer, err := Parse(root)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := BuildPlan("fallout4", root, installer, map[string][]string{
		installer.Steps[0].Groups[0].ID: {},
	}, PlanOptions{
		ModType:    "fallout4-data-root",
		PlannerID:  "vortex:fallout4:fomod",
		TargetRoot: "Data",
	})
	if err != nil {
		t.Fatal(err)
	}
	targets := instructionTargets(plan)
	if !targets["Data/always.txt"] || !targets["Data/usable.txt"] {
		t.Fatalf("always/installIfUsable files missing from %+v", plan.Instructions)
	}
	if targets["Data/normal.txt"] || targets["Data/blocked.txt"] {
		t.Fatalf("unwanted files installed from %+v", plan.Instructions)
	}
}

func TestBuildPlanOrdersFOMODFilePriorityLowToHigh(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "fomod", "ModuleConfig.xml"), `<config>
  <requiredInstallFiles>
    <file source="low.txt" destination="same.txt" priority="0" />
    <file source="high.txt" destination="same.txt" priority="10" />
  </requiredInstallFiles>
</config>`)
	writeFile(t, filepath.Join(root, "low.txt"), "low")
	writeFile(t, filepath.Join(root, "high.txt"), "high")
	installer, err := Parse(root)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := BuildPlan("fallout4", root, installer, nil, PlanOptions{
		ModType:    "fallout4-data-root",
		PlannerID:  "vortex:fallout4:fomod",
		TargetRoot: "Data",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Instructions) != 2 {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
	if plan.Instructions[0].Priority != 0 || plan.Instructions[1].Priority != 10 {
		t.Fatalf("priority order = %+v", plan.Instructions)
	}
	if filepath.Base(plan.Instructions[1].SourcePath) != "high.txt" {
		t.Fatalf("higher priority file should be staged last: %+v", plan.Instructions)
	}
}

func TestBuildPlanRequiresSatisfiedModuleDependencies(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "fomod", "ModuleConfig.xml"), `<config>
  <moduleDependencies operator="And">
    <fommDependency version="5.0" />
    <gameDependency version="1.10.0" />
  </moduleDependencies>
  <requiredInstallFiles>
    <file source="base.txt" />
  </requiredInstallFiles>
</config>`)
	writeFile(t, filepath.Join(root, "base.txt"), "base")
	installer, err := Parse(root)
	if err != nil {
		t.Fatal(err)
	}
	if installer.ModuleDependencies == nil || len(installer.ModuleDependencies.FOMMDependencies) != 1 || len(installer.ModuleDependencies.GameDependencies) != 1 {
		t.Fatalf("module dependencies were not parsed: %+v", installer.ModuleDependencies)
	}
	plan, err := BuildPlan("fallout4", root, installer, nil, PlanOptions{
		ModType:     "fallout4-data-root",
		PlannerID:   "vortex:fallout4:fomod",
		TargetRoot:  "Data",
		GameVersion: "1.10.984.0",
		HostVersion: "5.1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Instructions) != 1 || plan.Instructions[0].TargetRelative != "Data/base.txt" {
		t.Fatalf("module dependency plan = %+v", plan.Instructions)
	}
}

func TestBuildPlanBlocksUnsatisfiedModuleDependencies(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "fomod", "ModuleConfig.xml"), `<config>
  <moduleDependencies operator="And">
    <gameDependency version="2.0.0" />
  </moduleDependencies>
  <requiredInstallFiles>
    <file source="base.txt" />
  </requiredInstallFiles>
</config>`)
	writeFile(t, filepath.Join(root, "base.txt"), "base")
	installer, err := Parse(root)
	if err != nil {
		t.Fatal(err)
	}
	_, err = BuildPlan("fallout4", root, installer, nil, PlanOptions{
		ModType:     "fallout4-data-root",
		PlannerID:   "vortex:fallout4:fomod",
		TargetRoot:  "Data",
		GameVersion: "1.10.984.0",
		HostVersion: "5.1",
	})
	if err == nil {
		t.Fatal("expected unsatisfied module dependencies to block the plan")
	}
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %T %v, want UnsupportedError", err, err)
	}
}

func TestParseAppliesFOMODListOrdering(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "fomod", "ModuleConfig.xml"), `<config>
  <installSteps order="Descending">
    <installStep name="Alpha">
      <optionalFileGroups order="Explicit">
        <group name="Zeta" type="SelectAny">
          <plugins order="Ascending">
            <plugin name="Beta"><files><file source="beta.txt" /></files></plugin>
            <plugin name="Alpha"><files><file source="alpha.txt" /></files></plugin>
          </plugins>
        </group>
        <group name="Alpha" type="SelectAny">
          <plugins><plugin name="Only"><files><file source="only.txt" /></files></plugin></plugins>
        </group>
      </optionalFileGroups>
    </installStep>
    <installStep name="Omega">
      <optionalFileGroups>
        <group name="Only" type="SelectAny">
          <plugins><plugin name="Only"><files><file source="only.txt" /></files></plugin></plugins>
        </group>
      </optionalFileGroups>
    </installStep>
  </installSteps>
</config>`)
	for _, name := range []string{"alpha.txt", "beta.txt", "only.txt"} {
		writeFile(t, filepath.Join(root, name), name)
	}
	installer, err := Parse(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := []string{installer.Steps[0].Name, installer.Steps[1].Name}; got[0] != "Omega" || got[1] != "Alpha" {
		t.Fatalf("step order = %+v", got)
	}
	alphaStep := installer.Steps[1]
	if got := []string{alphaStep.Groups[0].Name, alphaStep.Groups[1].Name}; got[0] != "Zeta" || got[1] != "Alpha" {
		t.Fatalf("explicit group order = %+v", got)
	}
	if got := []string{alphaStep.Groups[0].Plugins[0].Name, alphaStep.Groups[0].Plugins[1].Name}; got[0] != "Alpha" || got[1] != "Beta" {
		t.Fatalf("plugin order = %+v", got)
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

func instructionTargets(plan installplan.Plan) map[string]bool {
	out := make(map[string]bool, len(plan.Instructions))
	for _, instruction := range plan.Instructions {
		out[instruction.TargetRelative] = true
	}
	return out
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
