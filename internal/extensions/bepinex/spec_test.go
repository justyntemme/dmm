package bepinex_test

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/bepinex"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersSourceBackedBepInExFrameworkSurface(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{
		gameext.MustCompileExtension(bepinex.Extension()),
	}).ExtensionSummaries()[0]

	if summary.ID != bepinex.ID || summary.Kind != gameext.ExtensionKindFramework {
		t.Fatalf("summary = %+v", summary)
	}
	if len(summary.Sources) != 1 {
		t.Fatalf("sources = %+v", summary.Sources)
	}
	assertReadyWithMessage(t, "mod type", summary.Capabilities.ModTypes, "bepinex-patcher")
	assertReadyWithMessage(t, "extension API", summary.Capabilities.ExtensionAPIs, "register-bepinex-unity-game")
	assertReadyWithMessage(t, "extension dashlet", summary.Capabilities.ExtensionDashlets, "bepinex-support")
	if len(summary.Capabilities.ExtensionTests) != 0 {
		t.Fatalf("framework must not advertise per-game BepInEx tests without validators: %+v", summary.Capabilities.ExtensionTests)
	}
	for _, event := range []string{sdk.EventDidInstallMod, sdk.EventProfileWillChange, sdk.EventGamemodeActivated, sdk.EventWillDeploy, sdk.EventCheckModsVersion} {
		assertReadyTrigger(t, summary.Capabilities.EventHandlers, event)
	}
}

func assertReadyTrigger(t *testing.T, kind []gameext.FeatureSummary, trigger string) {
	t.Helper()
	for _, feature := range kind {
		if feature.Trigger == trigger {
			if feature.Status != sdk.CapabilityStatusReady || feature.Message == "" {
				t.Fatalf("event %s = %+v", trigger, feature)
			}
			return
		}
	}
	t.Fatalf("event %s missing from %+v", trigger, kind)
}

func assertReadyWithMessage(t *testing.T, kind string, features []gameext.FeatureSummary, id string) {
	t.Helper()
	for _, feature := range features {
		if feature.ID != id {
			continue
		}
		if feature.Status != sdk.CapabilityStatusReady || feature.Message == "" {
			t.Fatalf("%s %s = %+v", kind, id, feature)
		}
		return
	}
	t.Fatalf("%s %s missing from %+v", kind, id, features)
}
