package modcontent

import (
	"reflect"
	"testing"
)

func TestFromFilesMirrorsVortexTypeOrderingAndGameConditions(t *testing.T) {
	got := FromFiles("stardewvalley", []string{
		"Example/manifest.json",
		"Example/config.json",
		"Example/ContentPack.dll",
		"Example/assets/icon.png",
		"Example/music.ogg",
	})
	want := []string{TypePlugin, TypeTexture, TypeMusic, TypeConfig}
	if !reflect.DeepEqual(got.Types, want) {
		t.Fatalf("types = %#v, want %#v", got.Types, want)
	}
	if got.Empty || got.Scanned != 5 {
		t.Fatalf("summary = %+v", got)
	}
}

func TestFromFilesTreatsScriptExtenderDLLsAsExtenders(t *testing.T) {
	got := FromFiles("skyrimse", []string{
		"SKSE/Plugins/example.dll",
		"Data/example.esp",
	})
	want := []string{TypePlugin, TypeExtender}
	if !reflect.DeepEqual(got.Types, want) {
		t.Fatalf("types = %#v, want %#v", got.Types, want)
	}
}

func TestFromFilesReportsEmptyWhenNoFilesExist(t *testing.T) {
	got := FromFiles("fallout4", nil)
	if !got.Empty || len(got.Types) != 0 || got.Scanned != 0 {
		t.Fatalf("summary = %+v", got)
	}
}
