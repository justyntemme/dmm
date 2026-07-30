package unreal_test

import (
	"context"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/unreal"
)

func TestSortablePakLoadOrderRewritesManagedPakFoldersByPriority(t *testing.T) {
	handler := unreal.SortablePakLoadOrderHandler(unreal.SortablePakLoadOrderOptions{
		TargetRoot: "End/Content/Paks/~mods",
		ModType:    "ff7rebirth-pak",
	})
	result, err := handler(context.Background(), sdk.EventHandlerInput{
		Mappings: []deploy.FileMapping{
			{InstalledModID: 20, ModID: "200", TargetRelative: "End/Content/Paks/~mods/Late_P.pak", Priority: 20},
			{InstalledModID: 10, ModID: "100", TargetRelative: "End/Content/Paks/~mods/Early_P.pak", Priority: 10},
			{InstalledModID: 10, ModID: "100", TargetRelative: "End/Content/Paks/~mods/Early_P.utoc", Priority: 10},
			{InstalledModID: 30, ModID: "300", TargetRelative: "End/Mods/Other/file.txt", Priority: 1},
		},
		Mods: []sdk.DeploymentMod{
			{ID: 10, Name: "Early", ModType: "ff7rebirth-pak", Priority: 10},
			{ID: 20, Name: "Late", ModType: "ff7rebirth-pak", Priority: 20},
			{ID: 30, Name: "Other", ModType: "ff7rebirth-ff7rml", Priority: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.ReplaceMappings {
		t.Fatal("expected hook to replace the deploy mapping set")
	}
	want := map[string]bool{
		"End/Content/Paks/~mods/AAB-mod-20/Late_P.pak":   false,
		"End/Content/Paks/~mods/AAA-mod-10/Early_P.pak":  false,
		"End/Content/Paks/~mods/AAA-mod-10/Early_P.utoc": false,
		"End/Mods/Other/file.txt":                        false,
	}
	for _, mapping := range result.Mappings {
		if _, ok := want[mapping.TargetRelative]; ok {
			want[mapping.TargetRelative] = true
		}
	}
	for target, found := range want {
		if !found {
			t.Fatalf("missing rewritten target %q in %+v", target, result.Mappings)
		}
	}
}

func TestSortablePakLoadOrderSkipsNonMatchingModTypes(t *testing.T) {
	handler := unreal.SortablePakLoadOrderHandler(unreal.SortablePakLoadOrderOptions{
		TargetRoot: "End/Content/Paks/~mods",
		ModType:    "ff7rebirth-pak",
	})
	result, err := handler(context.Background(), sdk.EventHandlerInput{
		Mappings: []deploy.FileMapping{
			{InstalledModID: 10, TargetRelative: "End/Content/Paks/~mods/Tool.dll", Priority: 1},
			{InstalledModID: 11, TargetRelative: "End/Content/Paks/~mods/Mod_P.pak", Priority: 1},
		},
		Mods: []sdk.DeploymentMod{
			{ID: 10, ModType: "ff7rebirth-binaries", Priority: 1},
			{ID: 11, ModType: "ff7rebirth-ff7rml", Priority: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ReplaceMappings || len(result.Mappings) != 0 {
		t.Fatalf("unexpected rewrite result = %+v", result)
	}
}

func TestMakeLoadOrderPrefixMatchesVortexPattern(t *testing.T) {
	tests := map[int]string{
		0:  "AAA",
		1:  "AAB",
		2:  "AAC",
		24: "AAY",
		25: "ABA",
	}
	for input, want := range tests {
		if got := unreal.MakeLoadOrderPrefix(input); got != want {
			t.Fatalf("prefix(%d) = %q, want %q", input, got, want)
		}
	}
}
