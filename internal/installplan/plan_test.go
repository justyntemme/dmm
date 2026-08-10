package installplan_test

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/stardewvalley"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	. "github.com/justyntemme/decky-mod-manager/internal/installplan"
	"github.com/justyntemme/decky-mod-manager/internal/integrity"
)

var stardewPlanner = NewRegistry([]GameSpec{gameext.MustCompileExtension(stardewvalley.Extension()).InstallPlan})

func Build(gameID, extractedRoot string) (Plan, error) {
	return stardewPlanner.Build(gameID, extractedRoot)
}

func SteamAppIDForVortexGameID(gameID string) (string, bool) {
	return stardewPlanner.SteamAppIDForVortexGameID(gameID)
}

func VortexGameIDForSteamAppID(appID string) (string, bool) {
	return stardewPlanner.VortexGameIDForSteamAppID(appID)
}

func DeploymentAllowedForSteamAppState(appID, state string) (bool, string) {
	return stardewPlanner.DeploymentAllowedForSteamAppState(appID, state)
}

func TestStardewPlannerBuildsInstructionsForSMAPIModFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "LookupAnything", "manifest.json"), `{"Name":"Lookup Anything","UniqueID":"Pathoschild.LookupAnything"}`)
	writeFile(t, filepath.Join(root, "LookupAnything", "LookupAnything.dll"), "dll")
	writeFile(t, filepath.Join(root, "LookupAnything", "i18n", "default.json"), "{}")

	plan, err := Build("413150", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "stardew-smapi-mod" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	if plan.PlannerID != "vortex:stardewvalley:stardew-valley-installer" {
		t.Fatalf("planner id = %q", plan.PlannerID)
	}
	if len(plan.DetectedFrom) != 1 || plan.DetectedFrom[0].Kind != "vortex-manifest" || plan.DetectedFrom[0].Path != "LookupAnything/manifest.json" {
		t.Fatalf("detected from = %+v", plan.DetectedFrom)
	}
	if len(plan.Instructions) != 3 {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
	want := "Mods/LookupAnything/LookupAnything.dll"
	found := false
	for _, instruction := range plan.Instructions {
		if instruction.TargetRelative == want {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing target %q in %+v", want, plan.Instructions)
	}
}

func TestPlannerAcceptsVortexGameIDAlias(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "LookupAnything", "manifest.json"), `{"Name":"Lookup Anything","UniqueID":"Pathoschild.LookupAnything"}`)
	writeFile(t, filepath.Join(root, "LookupAnything", "LookupAnything.dll"), "dll")

	plan, err := Build("stardewvalley", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:stardewvalley:stardew-valley-installer" {
		t.Fatalf("planner id = %q", plan.PlannerID)
	}
}

func TestDefaultRegistryMapsVortexGameIDAndSteamAppID(t *testing.T) {
	appID, ok := SteamAppIDForVortexGameID("stardewvalley")
	if !ok || appID != "413150" {
		t.Fatalf("steam app id = %q, %v", appID, ok)
	}
	gameID, ok := VortexGameIDForSteamAppID("413150")
	if !ok || gameID != "stardewvalley" {
		t.Fatalf("vortex game id = %q, %v", gameID, ok)
	}
}

func TestDefaultRegistryControlsDeploymentEligibility(t *testing.T) {
	if ok, reason := DeploymentAllowedForSteamAppState("413150", "needs_review"); !ok || reason != "" {
		t.Fatalf("stardew deployment eligibility = %v %q", ok, reason)
	}
	if ok, reason := DeploymentAllowedForSteamAppState("287700", "clean_candidate"); ok || reason == "" {
		t.Fatalf("unsupported deployment eligibility = %v %q", ok, reason)
	}

	registry := NewRegistry([]GameSpec{{
		SteamAppIDs:  []string{"999999"},
		VortexGameID: "examplegame",
	}})
	if ok, reason := registry.DeploymentAllowedForSteamAppState("999999", "needs_review"); ok || reason == "" {
		t.Fatalf("dirty supported deployment eligibility = %v %q", ok, reason)
	}
}

func TestStardewPlannerSupportsMultipleSMAPIModFolders(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "ModA", "manifest.json"), `{
  "Name":"Mod A",
  "UniqueID":"author.ModA",
  "Version":"1.2.3",
  "Description":"Adds the A component.",
  "EntryDll":"ModA.dll",
  "MinimumApiVersion":"4.0.0",
  "Dependencies":[{"UniqueID":"spacechase0.JsonAssets","MinimumVersion":"1.11.0","IsRequired":true}]
}`)
	writeFile(t, filepath.Join(root, "ModA", "ModA.dll"), "a")
	writeFile(t, filepath.Join(root, "ModB", "manifest.json"), `{
  "Name":"Mod B",
  "UniqueID":"author.ModB",
  "Description":"Adds the B content pack.",
  "ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher","MinimumVersion":"2.0.0"}
}`)
	writeFile(t, filepath.Join(root, "ModB", "ModB.dll"), "b")

	_, err := Build("413150", root)
	var choice ChoiceRequiredError
	if !errors.As(err, &choice) {
		t.Fatalf("expected component choice error, got %v", err)
	}
	if choice.Kind != "component-choice" {
		t.Fatalf("choice kind = %q", choice.Kind)
	}
	if len(choice.Installer.Steps) != 1 || len(choice.Installer.Steps[0].Groups) != 1 {
		t.Fatalf("choice installer = %+v", choice.Installer)
	}
	group := choice.Installer.Steps[0].Groups[0]
	if group.ID != "stardew-smapi-components" || group.Type != "SelectAtLeastOne" || len(group.Plugins) != 2 {
		t.Fatalf("choice group = %+v", group)
	}
	if got := choice.DefaultSelections["stardew-smapi-components"]; len(got) != 2 {
		t.Fatalf("default selections = %+v", choice.DefaultSelections)
	}
	var modADescription string
	for _, option := range group.Plugins {
		if option.Name == "Mod A" {
			modADescription = option.Description
			break
		}
	}
	for _, want := range []string{
		"Adds the A component.",
		"ID: author.ModA",
		"Version: 1.2.3",
		"Entry: ModA.dll",
		"Requires SMAPI: 4.0.0",
		"Recommended dependency spacechase0.JsonAssets 1.11.0+",
		"Path: ModA",
	} {
		if !strings.Contains(modADescription, want) {
			t.Fatalf("component description missing %q: %q", want, modADescription)
		}
	}

	plan, err := stardewPlanner.BuildWithOptions("413150", root, BuildOptions{
		Selections: map[string][]string{
			"stardew-smapi-components": []string{"component:ModB"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Instructions) != 2 {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
	if len(plan.DetectedFrom) != 1 || plan.DetectedFrom[0].Path != "ModB/manifest.json" {
		t.Fatalf("detected from = %+v", plan.DetectedFrom)
	}
	for _, instruction := range plan.Instructions {
		if strings.Contains(instruction.TargetRelative, "ModA") {
			t.Fatalf("unselected component was staged: %+v", plan.Instructions)
		}
	}
}

func TestStardewPlannerAcceptsRelaxedSMAPIManifest(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "VisibleFish", "manifest.json"), "\ufeff"+`{
  "Name": "Visible Fish",
  "UniqueID": "shekurika.WaterFish",
  "Version": "0.4.2",
  "ContentPackFor": {
    "UniqueID": "Pathoschild.ContentPatcher",
    "MinimumVersion": "2.0.0",
  },
  "Dependencies": [
    {"UniqueID": "spacechase0.GenericModConfigMenu", "IsRequired": false},
    {"UniqueID": "Pathoschild.LookupAnything", "MinimumVersion": "1.55.0"},
  ],
}`)
	writeFile(t, filepath.Join(root, "VisibleFish", "showFishInWater.dll"), "dll")

	plan, err := Build("413150", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:stardewvalley:stardew-valley-installer" {
		t.Fatalf("planner id = %q", plan.PlannerID)
	}
	found := false
	for _, instruction := range plan.Instructions {
		if instruction.TargetRelative == "Mods/VisibleFish/showFishInWater.dll" {
			found = true
		}
	}
	if !found {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
	if len(plan.Metadata) != 1 {
		t.Fatalf("metadata = %+v", plan.Metadata)
	}
	metadata := plan.Metadata[0]
	if metadata.Kind != stardewvalley.MetadataKindSMAPIManifest || metadata.Name != "Visible Fish" || metadata.UniqueID != "shekurika.WaterFish" || metadata.Version != "0.4.2" {
		t.Fatalf("metadata = %+v", metadata)
	}
	if metadata.ManifestVersion != "0.4.2" || len(metadata.AdditionalLogicalFileNames) != 1 || metadata.AdditionalLogicalFileNames[0] != "shekurika.waterfish" {
		t.Fatalf("vortex attribute metadata = %+v", metadata)
	}
	if metadata.ContentPackFor == nil || metadata.ContentPackFor.UniqueID != "Pathoschild.ContentPatcher" || metadata.ContentPackFor.MinimumVersion != "2.0.0" || metadata.ContentPackFor.Required {
		t.Fatalf("content pack metadata = %+v", metadata.ContentPackFor)
	}
	if len(metadata.Dependencies) != 2 || metadata.Dependencies[0].Required || metadata.Dependencies[1].Required {
		t.Fatalf("dependencies = %+v", metadata.Dependencies)
	}
}

func TestStardewPlannerAcceptsCommentedSMAPIManifestLikeVortex(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "GenericModConfigMenu", "manifest.json"), `{
  /*
   | This file is automatically updated by ModManifestBuilder.
   */
  "$schema": "https://smapi.io/schemas/manifest.json",
  "UniqueID": "spacechase0.GenericModConfigMenu",
  "Name": "Generic Mod Config Menu",
  "Version": "1.16.0",
  "MinimumApiVersion": "4.1",
  "EntryDll": "GenericModConfigMenu.dll"
}`)
	writeFile(t, filepath.Join(root, "GenericModConfigMenu", "GenericModConfigMenu.dll"), "dll")

	plan, err := Build("413150", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:stardewvalley:stardew-valley-installer" {
		t.Fatalf("planner id = %q", plan.PlannerID)
	}
	found := false
	for _, instruction := range plan.Instructions {
		if instruction.TargetRelative == "Mods/GenericModConfigMenu/GenericModConfigMenu.dll" {
			found = true
		}
	}
	if !found {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
	if len(plan.Metadata) != 1 || plan.Metadata[0].UniqueID != "spacechase0.GenericModConfigMenu" {
		t.Fatalf("metadata = %+v", plan.Metadata)
	}
}

func TestStardewPlannerBuildsRootFolderInstructionsFromVortexMetadata(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "LegacyXNB", "Content", "Characters", "Abigail.xnb"), "xnb")
	writeFile(t, filepath.Join(root, "LegacyXNB", "readme.txt"), "readme")

	plan, err := Build("413150", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "sdvrootfolder" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	if plan.PlannerID != "vortex:stardewvalley:sdvrootfolder" {
		t.Fatalf("planner id = %q", plan.PlannerID)
	}
	if len(plan.Instructions) != 1 {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
	if plan.Instructions[0].TargetRelative != "Content/Characters/Abigail.xnb" {
		t.Fatalf("target = %q", plan.Instructions[0].TargetRelative)
	}
	if len(plan.DetectedFrom) != 1 || plan.DetectedFrom[0].Kind != "vortex-root-folder" {
		t.Fatalf("detected from = %+v", plan.DetectedFrom)
	}
}

func TestStardewPlannerRejectsInstallerToolArchiveLayout(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "SMAPI 4.5.2 installer", "install on Linux.sh"), "install")
	writeFile(t, filepath.Join(root, "SMAPI 4.5.2 installer", "README.txt"), "readme")

	_, err := Build("413150", root)
	if err == nil {
		t.Fatal("expected unsupported layout")
	}
}

func TestStardewPlannerBuildsGenericModsRootConfigBundle(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Automate", "config.json"), `{}`)
	writeFile(t, filepath.Join(root, "ChestsAnywhere", "config.json"), `{}`)

	plan, err := Build("413150", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:stardewvalley:generic-mods-root" || plan.ModType != "stardew-smapi-mod" {
		t.Fatalf("plan = %+v", plan)
	}
	wantTargets := map[string]bool{
		"Mods/Automate/config.json":       false,
		"Mods/ChestsAnywhere/config.json": false,
	}
	for _, instruction := range plan.Instructions {
		if _, ok := wantTargets[instruction.TargetRelative]; ok {
			wantTargets[instruction.TargetRelative] = true
		}
	}
	for target, found := range wantTargets {
		if !found {
			t.Fatalf("missing target %q in %+v", target, plan.Instructions)
		}
	}
}

func TestStardewPlannerBuildsGenericArchiveWithModsFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Mods", "Automate", "config.json"), `{}`)

	plan, err := Build("413150", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:stardewvalley:generic-mods-folder" || plan.ModType != "stardew-smapi-mod" {
		t.Fatalf("plan = %+v", plan)
	}
	if len(plan.Instructions) != 1 || plan.Instructions[0].TargetRelative != "Mods/Automate/config.json" {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
}

func TestStardewPlannerBuildsSMAPIInstallerPayloadFromVortexMetadata(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "internal", "SMAPI.Installer.dll"), "dll")
	writeZip(t, filepath.Join(root, "internal", "linux", "install.dat"), map[string]string{
		"StardewModdingAPI":                  "exe",
		"smapi-internal/config.json":         "{}",
		"Mods/ConsoleCommands/manifest.json": `{"Name":"Console Commands","UniqueID":"SMAPI.ConsoleCommands"}`,
		"steam_appid.txt":                    "413150",
	})

	plan, err := Build("413150", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "SMAPI" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	if plan.PlannerID != "vortex:stardewvalley:smapi-installer:linux" {
		t.Fatalf("planner id = %q", plan.PlannerID)
	}
	if len(plan.DetectedFrom) != 1 || plan.DetectedFrom[0].Kind != "vortex-embedded-payload" {
		t.Fatalf("detected from = %+v", plan.DetectedFrom)
	}
	wantTargets := map[string]bool{
		"StardewModdingAPI":                  false,
		"smapi-internal/config.json":         false,
		"Mods/ConsoleCommands/manifest.json": false,
		"StardewModdingAPI.deps.json":        false,
		"steam_appid.txt":                    false,
	}
	copyTargets := map[string]bool{
		"StardewModdingAPI":           true,
		"StardewModdingAPI.deps.json": true,
	}
	for _, instruction := range plan.Instructions {
		if _, ok := wantTargets[instruction.TargetRelative]; ok {
			wantTargets[instruction.TargetRelative] = true
		}
		if copyTargets[instruction.TargetRelative] && instruction.DeployStrategy != DeployStrategyCopy {
			t.Fatalf("copy strategy missing for %+v", instruction)
		}
		if instruction.TargetRelative == "StardewModdingAPI.deps.json" && instruction.Kind != InstructionKindGenerateFromGameFile {
			t.Fatalf("generated deps instruction = %+v", instruction)
		}
		if instruction.TargetRelative == "steam_appid.txt" && instruction.TargetPolicy != TargetPolicyKeepExisting {
			t.Fatalf("steam app id policy = %+v", instruction)
		}
	}
	foundManifestMetadata := false
	for _, metadata := range plan.Metadata {
		if metadata.UniqueID == "SMAPI.ConsoleCommands" && metadata.TargetRelative == "Mods/ConsoleCommands/manifest.json" {
			foundManifestMetadata = true
		}
	}
	if !foundManifestMetadata {
		t.Fatalf("metadata = %+v", plan.Metadata)
	}
	for target, found := range wantTargets {
		if !found {
			t.Fatalf("missing target %q in %+v", target, plan.Instructions)
		}
	}
}

func TestStardewPlannerBuildsWindowsSMAPIInstallerPayloadFromVortexMetadata(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "internal", "SMAPI.Installer.dll"), "dll")
	writeZip(t, filepath.Join(root, "internal", "linux", "install.dat"), map[string]string{
		"StardewModdingAPI": "linux",
	})
	writeZip(t, filepath.Join(root, "internal", "windows", "install.dat"), map[string]string{
		"StardewModdingAPI.exe":              "exe",
		"StardewModdingAPI.exe.config":       "config",
		"smapi-internal/config.json":         "{}",
		"Mods/ConsoleCommands/manifest.json": `{"Name":"Console Commands","UniqueID":"SMAPI.ConsoleCommands"}`,
		"steam_appid.txt":                    "413150",
	})

	plan, err := stardewPlanner.BuildWithOptions("413150", root, BuildOptions{PlatformID: "windows"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:stardewvalley:smapi-installer:windows" || plan.ModType != "SMAPI" {
		t.Fatalf("plan = %+v", plan)
	}
	wantTargets := map[string]bool{
		"StardewModdingAPI.exe":              false,
		"StardewModdingAPI.exe.config":       false,
		"smapi-internal/config.json":         false,
		"Mods/ConsoleCommands/manifest.json": false,
		"StardewModdingAPI.deps.json":        false,
		"steam_appid.txt":                    false,
	}
	copyTargets := map[string]bool{
		"StardewModdingAPI.exe":        true,
		"StardewModdingAPI.exe.config": true,
		"StardewModdingAPI.deps.json":  true,
	}
	for _, instruction := range plan.Instructions {
		if _, ok := wantTargets[instruction.TargetRelative]; ok {
			wantTargets[instruction.TargetRelative] = true
		}
		if copyTargets[instruction.TargetRelative] && instruction.DeployStrategy != DeployStrategyCopy {
			t.Fatalf("copy strategy missing for %+v", instruction)
		}
		if instruction.TargetRelative == "StardewModdingAPI" {
			t.Fatalf("linux executable leaked into windows plan: %+v", instruction)
		}
		if instruction.TargetRelative == "StardewModdingAPI.deps.json" && instruction.Kind != InstructionKindGenerateFromGameFile {
			t.Fatalf("generated deps instruction = %+v", instruction)
		}
	}
	for target, found := range wantTargets {
		if !found {
			t.Fatalf("missing target %q in %+v", target, plan.Instructions)
		}
	}
}

func TestRegistryBuildsGenericManifestPlanFromVortexMetadataSpec(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "ExampleMod", "mod.json"), `{"displayName":"Example Mod","id":"author.example","version":2}`)
	writeFile(t, filepath.Join(root, "ExampleMod", "payload.dat"), "payload")

	registry := NewRegistry([]GameSpec{{
		SteamAppIDs:  []string{"999999"},
		VortexGameID: "examplegame",
		ModTypes: []ModTypeSpec{{
			ID:         "example-mod",
			TargetRoot: "ConfiguredMods",
		}},
		Installers: []InstallerSpec{{
			ID:                "vortex:examplegame:manifest-installer",
			VortexInstallerID: "manifest-installer",
			Priority:          50,
			ModType:           "example-mod",
			Match: MatchSpec{
				ManifestFileName: "mod.json",
			},
			MetadataExtractors: []MetadataExtractorSpec{{
				Kind:             MetadataKindJSONManifest,
				ManifestFileName: "mod.json",
				NameField:        "displayName",
				UniqueIDField:    "id",
				VersionField:     "version",
			}},
			InstructionMode: InstructionManifestFolders,
		}},
	}})

	plan, err := registry.Build("examplegame", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:examplegame:manifest-installer" || plan.ModType != "example-mod" {
		t.Fatalf("plan = %+v", plan)
	}
	if len(plan.Metadata) != 1 {
		t.Fatalf("metadata = %+v", plan.Metadata)
	}
	foundPayloadTarget := false
	for _, instruction := range plan.Instructions {
		if instruction.TargetRelative == "ConfiguredMods/ExampleMod/payload.dat" {
			foundPayloadTarget = true
		}
	}
	if !foundPayloadTarget {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
	metadata := plan.Metadata[0]
	if metadata.Kind != MetadataKindJSONManifest || metadata.Name != "Example Mod" || metadata.UniqueID != "author.example" || metadata.Version != "2" {
		t.Fatalf("metadata = %+v", metadata)
	}
	if len(metadata.AdditionalLogicalFileNames) != 1 || metadata.AdditionalLogicalFileNames[0] != "author.example" {
		t.Fatalf("logical names = %+v", metadata.AdditionalLogicalFileNames)
	}
}

func TestManifestComponentChoiceSelectExactlyOneRejectsMultipleSelections(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "RedChicken", "mod.json"), `{"displayName":"Red Chickens","id":"author.redchickens","Description":"Makes chickens red."}`)
	writeFile(t, filepath.Join(root, "RedChicken", "payload.dat"), "red")
	writeFile(t, filepath.Join(root, "BlueChicken", "mod.json"), `{"displayName":"Blue Chickens","id":"author.bluechickens","Description":"Makes chickens blue."}`)
	writeFile(t, filepath.Join(root, "BlueChicken", "payload.dat"), "blue")

	registry := NewRegistry([]GameSpec{{
		SteamAppIDs:  []string{"999999"},
		VortexGameID: "examplegame",
		ModTypes: []ModTypeSpec{{
			ID:         "example-mod",
			TargetRoot: "ConfiguredMods",
		}},
		Installers: []InstallerSpec{{
			ID:                "vortex:examplegame:manifest-variants",
			VortexInstallerID: "manifest-variants",
			Priority:          50,
			ModType:           "example-mod",
			Match: MatchSpec{
				ManifestFileName: "mod.json",
			},
			MetadataExtractors: []MetadataExtractorSpec{{
				Kind:             MetadataKindJSONManifest,
				ManifestFileName: "mod.json",
				NameField:        "displayName",
				UniqueIDField:    "id",
			}},
			ComponentChoices: &ComponentChoiceSpec{
				Kind:      "component-choice",
				Name:      "Variant Selection",
				GroupID:   "example-variants",
				GroupName: "Variant",
				GroupType: "SelectExactlyOne",
			},
			InstructionMode: InstructionManifestFolders,
		}},
	}})

	_, err := registry.BuildWithOptions("examplegame", root, BuildOptions{
		Selections: map[string][]string{
			"example-variants": {"component:RedChicken", "component:BlueChicken"},
		},
	})
	var choice ChoiceRequiredError
	if !errors.As(err, &choice) {
		t.Fatalf("expected exactly-one choice error, got %v", err)
	}

	plan, err := registry.BuildWithOptions("examplegame", root, BuildOptions{
		Selections: map[string][]string{
			"example-variants": {"component:RedChicken"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.DetectedFrom) != 1 || plan.DetectedFrom[0].Path != "RedChicken/mod.json" {
		t.Fatalf("detected from = %+v", plan.DetectedFrom)
	}
	for _, instruction := range plan.Instructions {
		if strings.Contains(instruction.TargetRelative, "BlueChicken") {
			t.Fatalf("unselected variant staged: %+v", plan.Instructions)
		}
	}
}

func TestRegistrySupportIsSpecDriven(t *testing.T) {
	registry := NewRegistry([]GameSpec{{
		SteamAppIDs:  []string{"999999"},
		VortexGameID: "examplegame",
	}})

	if !registry.SupportsGame("999999") || !registry.SupportsGame("examplegame") {
		t.Fatal("expected registered Steam and Vortex IDs to be supported")
	}
	if registry.SupportsGame("missinggame") {
		t.Fatal("unexpected support for unregistered game")
	}
}

func TestManifestDisplayNameFromPlanUsesGenericMetadata(t *testing.T) {
	plan := Plan{Metadata: []ModMetadata{
		{UniqueID: "author.default"},
		{Name: "Display Name", UniqueID: "author.display"},
	}}

	if got := ManifestDisplayNameFromPlan(plan); got != "Display Name" {
		t.Fatalf("display name = %q", got)
	}
}

func TestManifestDisplayNameFromPlanFallsBackToUniqueID(t *testing.T) {
	plan := Plan{Metadata: []ModMetadata{{UniqueID: "author.default"}}}

	if got := ManifestDisplayNameFromPlan(plan); got != "author.default" {
		t.Fatalf("display name = %q", got)
	}
}

func TestStardewPlannerAcceptsManifestWithoutIdentityLikeVortex(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Mod", "manifest.json"), `{}`)
	writeFile(t, filepath.Join(root, "Mod", "mod.dll"), "dll")

	plan, err := Build("413150", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:stardewvalley:stardew-valley-installer" {
		t.Fatalf("planner id = %q", plan.PlannerID)
	}
	if len(plan.Metadata) != 0 {
		t.Fatalf("metadata = %+v", plan.Metadata)
	}
	found := false
	for _, instruction := range plan.Instructions {
		if instruction.TargetRelative == "Mods/Mod/mod.dll" {
			found = true
		}
	}
	if !found {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
}

func TestArchiveRootInstallerTargetsConfiguredModPath(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Meshes", "weapon.nif"), "mesh")
	writeFile(t, filepath.Join(root, "Example.esp"), "plugin")

	registry := NewRegistry([]GameSpec{{
		SteamAppIDs:  []string{"377160"},
		VortexGameID: "fallout4",
		ModTypes: []ModTypeSpec{
			{ID: "fallout4-data-root", TargetRoot: "Data"},
		},
		Installers: []InstallerSpec{{
			ID:                "vortex:fallout4:data-root",
			VortexInstallerID: "game-query-mod-path",
			ModType:           "fallout4-data-root",
			InstructionMode:   InstructionArchiveRoot,
		}},
	}})

	plan, err := registry.Build("fallout4", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:fallout4:data-root" || plan.ModType != "fallout4-data-root" {
		t.Fatalf("plan = %+v", plan)
	}
	want := map[string]bool{
		"Data/Example.esp":       false,
		"Data/Meshes/weapon.nif": false,
	}
	for _, instruction := range plan.Instructions {
		if _, ok := want[instruction.TargetRelative]; ok {
			want[instruction.TargetRelative] = true
		}
	}
	for target, found := range want {
		if !found {
			t.Fatalf("missing target %q in %+v", target, plan.Instructions)
		}
	}
}

func TestArchiveRootInstallerCanStripSingleCommonWrapper(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Nice Weapon", "Meshes", "weapon.nif"), "mesh")
	writeFile(t, filepath.Join(root, "Nice Weapon", "NiceWeapon.esp"), "plugin")

	registry := NewRegistry([]GameSpec{{
		SteamAppIDs:  []string{"377160"},
		VortexGameID: "fallout4",
		ModTypes: []ModTypeSpec{
			{ID: "fallout4-data-root", TargetRoot: "Data"},
		},
		Installers: []InstallerSpec{{
			ID:                "vortex:fallout4:data-root",
			VortexInstallerID: "game-query-mod-path",
			ModType:           "fallout4-data-root",
			StripCommonRoot:   true,
			InstructionMode:   InstructionArchiveRoot,
		}},
	}})

	plan, err := registry.Build("fallout4", root)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.DetectedFrom) != 1 || plan.DetectedFrom[0].Path != "Nice Weapon" {
		t.Fatalf("detected from = %+v", plan.DetectedFrom)
	}
	want := map[string]bool{
		"Data/NiceWeapon.esp":    false,
		"Data/Meshes/weapon.nif": false,
	}
	for _, instruction := range plan.Instructions {
		if strings.Contains(instruction.TargetRelative, "Nice Weapon") {
			t.Fatalf("wrapper was not stripped from %+v", instruction)
		}
		if _, ok := want[instruction.TargetRelative]; ok {
			want[instruction.TargetRelative] = true
		}
	}
	for target, found := range want {
		if !found {
			t.Fatalf("missing target %q in %+v", target, plan.Instructions)
		}
	}
}

func TestArchiveRootInstallerDoesNotStripWhenFilesDoNotShareAWrapper(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Meshes", "weapon.nif"), "mesh")
	writeFile(t, filepath.Join(root, "readme.txt"), "readme")

	registry := NewRegistry([]GameSpec{{
		SteamAppIDs:  []string{"377160"},
		VortexGameID: "fallout4",
		ModTypes: []ModTypeSpec{
			{ID: "fallout4-data-root", TargetRoot: "Data"},
		},
		Installers: []InstallerSpec{{
			ID:                "vortex:fallout4:data-root",
			VortexInstallerID: "game-query-mod-path",
			ModType:           "fallout4-data-root",
			StripCommonRoot:   true,
			InstructionMode:   InstructionArchiveRoot,
		}},
	}})

	plan, err := registry.Build("fallout4", root)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.DetectedFrom) != 1 || plan.DetectedFrom[0].Path != "." {
		t.Fatalf("detected from = %+v", plan.DetectedFrom)
	}
	want := map[string]bool{
		"Data/Meshes/weapon.nif": false,
		"Data/readme.txt":        false,
	}
	for _, instruction := range plan.Instructions {
		if _, ok := want[instruction.TargetRelative]; ok {
			want[instruction.TargetRelative] = true
		}
	}
	for target, found := range want {
		if !found {
			t.Fatalf("missing target %q in %+v", target, plan.Instructions)
		}
	}
}

func TestCustomInstallerBuildsPlanFromExtensionHook(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "content", "file.txt"), "content")
	called := false
	registry := NewRegistry([]GameSpec{{
		SteamAppIDs:  []string{"999999"},
		VortexGameID: "customgame",
		ModTypes: []ModTypeSpec{
			{ID: "custom-mod", TargetRoot: "Mods"},
		},
		Installers: []InstallerSpec{{
			ID:                "vortex:customgame:custom",
			VortexInstallerID: "custom",
			ModType:           "custom-mod",
			InstructionMode:   InstructionCustom,
			CustomBuild: func(input BuildInput) (Plan, error) {
				called = true
				return Plan{
					GameID:    input.GameID,
					ModType:   input.Installer.ModType,
					PlannerID: input.Installer.ID,
					Instructions: []Instruction{{
						Kind:            InstructionKindCopy,
						SourcePath:      filepath.Join(input.ExtractedRoot, "content", "file.txt"),
						StagingRelative: "custom/file.txt",
						TargetRelative:  filepath.ToSlash(filepath.Join(input.TargetRoot, "custom", "file.txt")),
					}},
				}, nil
			},
		}},
	}})

	plan, err := registry.Build("customgame", root)
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("custom build hook was not called")
	}
	if plan.PlannerID != "vortex:customgame:custom" || plan.ModType != "custom-mod" || len(plan.Instructions) != 1 {
		t.Fatalf("plan = %+v", plan)
	}
	if plan.Instructions[0].TargetRelative != "Mods/custom/file.txt" {
		t.Fatalf("target = %q", plan.Instructions[0].TargetRelative)
	}
}

func TestPlannerVerifiesExpectedExtractedFileHashes(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "content", "file.txt")
	writeFile(t, filePath, "content")
	registry := NewRegistry([]GameSpec{{
		SteamAppIDs:  []string{"999999"},
		VortexGameID: "hashgame",
		ModTypes:     []ModTypeSpec{{ID: "hash-mod"}},
		Installers: []InstallerSpec{{
			ID:                "vortex:hashgame:hash",
			VortexInstallerID: "hash",
			ModType:           "hash-mod",
			Match:             MatchSpec{FileBasenames: []string{"file.txt"}},
			ExpectedExtractedFileHashes: []ExtractedFileHashSpec{{
				RelativePath: "content/file.txt",
				Expected: []integrity.ExpectedHash{{
					Algorithm: integrity.AlgorithmMD5,
					Value:     "9a0364b9e99bb480dd25e1f0284c8555",
					Label:     "test extracted file",
				}},
			}},
			InstructionMode: InstructionArchiveRoot,
		}},
	}})

	if _, err := registry.Build("hashgame", root); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filePath, "changed")
	_, err := registry.Build("hashgame", root)
	if err == nil {
		t.Fatal("expected extracted file hash mismatch")
	}
	if !strings.Contains(err.Error(), "extracted file integrity validation failed") {
		t.Fatalf("error = %v", err)
	}
}

func TestPlannerMatchesVortexStyleStopPatternsIntoQueryModPath(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "wrapper", "t", "0001.xml"), "<language />")
	writeFile(t, filepath.Join(root, "wrapper", "readme.txt"), "docs")

	registry := NewRegistry([]GameSpec{{
		SteamAppIDs:  []string{"2870"},
		VortexGameID: "xrebirth",
		QueryModPath: "extensions",
		StopPatterns: []string{
			`(^|/)t/[^/]+\.xml$`,
		},
		Installers: []InstallerSpec{{
			ID:                "vortex:xrebirth:dropin",
			VortexInstallerID: "dropin",
			Priority:          75,
			ModType:           "xrebirth-dropin",
			Match:             MatchSpec{UseGameStopPatterns: true},
			StripCommonRoot:   true,
			InstructionMode:   InstructionArchiveRoot,
		}},
	}})

	plan, err := registry.Build("2870", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:xrebirth:dropin" || len(plan.Instructions) != 2 {
		t.Fatalf("plan = %+v", plan)
	}
	for _, instruction := range plan.Instructions {
		if !strings.HasPrefix(instruction.TargetRelative, "extensions/") {
			t.Fatalf("target did not use queryModPath default: %+v", instruction)
		}
	}
}

func TestPlannerMatchesVortexStyleFileExtensionsAllMode(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "docs", "readme.md"), "readme")
	writeFile(t, filepath.Join(root, "docs", "preview.png"), "image")

	registry := NewRegistry([]GameSpec{{
		SteamAppIDs:  []string{"2870"},
		VortexGameID: "xrebirth",
		QueryModPath: "extensions",
		Installers: []InstallerSpec{{
			ID:                "vortex:xrebirth:docs",
			VortexInstallerID: "documentation",
			Priority:          90,
			ModType:           "xrebirth-documentation",
			Match: MatchSpec{
				FileExtensions:    []string{".md", ".png"},
				FileExtensionMode: MatchModeAll,
			},
			StripCommonRoot: true,
			InstructionMode: InstructionArchiveRoot,
		}},
	}})

	plan, err := registry.Build("2870", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:xrebirth:docs" || len(plan.Instructions) != 2 {
		t.Fatalf("plan = %+v", plan)
	}
}

func TestPlannerMatchesVortexStyleRegexAnyMode(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "ReShade", "preset.ini"), "preset")

	registry := NewRegistry([]GameSpec{{
		SteamAppIDs:  []string{"2870"},
		VortexGameID: "xrebirth",
		Installers: []InstallerSpec{{
			ID:                "vortex:xrebirth:shader",
			VortexInstallerID: "shader-injector",
			Priority:          65,
			ModType:           "xrebirth-shader-injector",
			Match: MatchSpec{
				RegexPatterns: []string{`(^|/)ReShade/`},
				RegexMode:     MatchModeAny,
			},
			StripCommonRoot: true,
			InstructionMode: InstructionArchiveRoot,
		}},
	}})

	plan, err := registry.Build("2870", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:xrebirth:shader" {
		t.Fatalf("planner = %q", plan.PlannerID)
	}
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zipper := zip.NewWriter(out)
	for name, contents := range files {
		writer, err := zipper.Create(strings.TrimLeft(filepath.ToSlash(name), "/"))
		if err != nil {
			_ = zipper.Close()
			_ = out.Close()
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(contents)); err != nil {
			_ = zipper.Close()
			_ = out.Close()
			t.Fatal(err)
		}
	}
	if err := zipper.Close(); err != nil {
		_ = out.Close()
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
}
