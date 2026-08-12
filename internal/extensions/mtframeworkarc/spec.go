package mtframeworkarc

import (
	"errors"
	"os"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/arctool"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	ID      = "mtframework-arc-support"
	Name    = "Capcom MT Framework ARC Support"
	Version = "0.1.0"
	BuildID = "first-party-go"

	ARCToolPathEnv = "DMM_ARCTOOL_PATH"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:      ID,
		Name:    Name,
		Kind:    sdk.ExtensionKindFramework,
		Version: Version,
		BuildID: BuildID,
		Register: func(r sdk.Registrar) {
			Register(r)
		},
	}
}

func Register(r sdk.Registrar) {
	for _, ref := range Sources() {
		r.RegisterSource(ref)
	}
	r.RegisterArchiveType(sdk.ArchiveTypeSpec{
		ID:             "arc",
		Name:           "Capcom MT Framework ARC",
		FileExtensions: []string{".arc"},
		Engine:         ID,
		SupportsWrite:  true,
		Status:         sdk.CapabilityStatusReady,
		Message:        "DMM runs Vortex-compatible ARCtool list/extract/create operations when " + ARCToolPathEnv + " points at ARCtool.exe.",
	})
	r.RegisterExtensionDashlet(sdk.ExtensionDashletSpec{
		ID:      "mtframework-arc-support",
		Name:    "ARC Support",
		Scope:   "archive-runtime",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM mirrors Vortex's MT Framework ARC support dashlet with the source-backed ARC archive runtime, ARCTool environment validation, and extension diagnostics for games that register ARC-backed installers.",
	})
}

func RunnerFromEnvironment() (arctool.Runner, error) {
	path := strings.TrimSpace(os.Getenv(ARCToolPathEnv))
	if path == "" {
		return arctool.Runner{}, errors.New(ARCToolPathEnv + " must point at ARCtool.exe before DMM can merge MT Framework ARC archives")
	}
	return arctool.Runner{ExecutablePath: path}, nil
}

func Sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex mtframework-arc-support source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/mtframework-arc-support/src/index.ts",
		},
	}
}
