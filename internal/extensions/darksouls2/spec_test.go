package darksouls2

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sharedmodtypes"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersSourceBackedGeDoSaToSupport(t *testing.T) {
	compiled := gameext.MustCompileExtension(Extension())
	if len(compiled.TargetRoots) != 1 || compiled.TargetRoots[0].ID != gedosatoRootID {
		t.Fatalf("target roots = %+v", compiled.TargetRoots)
	}

	modTypes := map[string]bool{}
	for _, modType := range compiled.InstallPlan.ModTypes {
		modTypes[modType.ID] = true
	}
	if !modTypes[sharedmodtypes.GeDoSaToType] {
		t.Fatalf("mod types = %+v", compiled.InstallPlan.ModTypes)
	}

	installers := map[string]bool{}
	for _, installer := range compiled.InstallPlan.Installers {
		installers[installer.ID] = true
	}
	if !installers["vortex:darksouls2:gedosato"] {
		t.Fatalf("installers = %+v", compiled.InstallPlan.Installers)
	}
	if len(compiled.RuntimeRequirements.RuntimeRequirements) != 1 {
		t.Fatalf("runtime requirements = %+v", compiled.RuntimeRequirements.RuntimeRequirements)
	}
}
