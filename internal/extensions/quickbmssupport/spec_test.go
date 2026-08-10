package quickbmssupport

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/quickbms"
)

func TestExtensionRegistersSourceBackedQuickBMSAPIRuntime(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{
		gameext.MustCompileExtension(Extension()),
	}).ExtensionSummaries()[0]

	if summary.ID != ID || summary.Kind != gameext.ExtensionKindFramework {
		t.Fatalf("summary = %+v", summary)
	}
	byID := map[string]gameext.FeatureSummary{}
	for _, api := range summary.Capabilities.ExtensionAPIs {
		byID[api.ID] = api
	}
	for _, id := range []string{"qbmsRegisterGame", "qbmsList", "qbmsExtract", "qbmsWrite", "qbmsReimport"} {
		if byID[id].Status != sdk.CapabilityStatusReady || byID[id].Message == "" {
			t.Fatalf("%s capability = %+v", id, byID[id])
		}
	}
}

func TestAPIGatesOperationsByRegisteredGame(t *testing.T) {
	api := NewAPI(quickbms.Runner{})
	if api.GameRegistered("example") {
		t.Fatal("game should not start registered")
	}
	_, err := api.List(context.Background(), "example", quickbms.Operation{})
	if err == nil {
		t.Fatal("expected unregistered game error")
	}
	api.RegisterGame("example")
	if !api.GameRegistered("example") {
		t.Fatal("game should be registered")
	}
}

func TestAPIListRunsTypedQuickBMSBridge(t *testing.T) {
	root := t.TempDir()
	exe := filepath.Join(root, "fake-qbms")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\nprintf '00000010 20 assets/mesh.nif\\n'\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	api := NewAPI(quickbms.Runner{
		ExecutablePath: exe,
		DataDir:        root,
		Timeout:        5 * time.Second,
	})
	api.RegisterGame("example")
	result, err := api.List(context.Background(), "example", quickbms.Operation{
		BMSScriptPath: filepath.Join(root, "example.bms"),
		ArchivePath:   filepath.Join(root, "archive.pak"),
		OperationPath: filepath.Join(root, "out"),
		Options:       quickbms.Options{WildCards: []string{"assets/{}"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 1 || result.Entries[0].FilePath != "assets/mesh.nif" {
		t.Fatalf("entries = %+v", result.Entries)
	}
}
