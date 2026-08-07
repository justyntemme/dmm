package workshoponly

import (
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

type Spec struct {
	ID          string
	Name        string
	Version     string
	BuildID     string
	SteamAppIDs []string
	Sources     []sdk.SourceRef
}

func Extension(spec Spec) sdk.Extension {
	if strings.TrimSpace(spec.Version) == "" {
		spec.Version = "0.1.0"
	}
	if strings.TrimSpace(spec.BuildID) == "" {
		spec.BuildID = "first-party-go"
	}
	return sdk.Extension{
		ID:      spec.ID,
		Name:    spec.Name,
		Version: spec.Version,
		BuildID: spec.BuildID,
		Register: func(r sdk.Registrar) {
			Register(r, spec)
		},
	}
}

func Register(r sdk.Registrar, spec Spec) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs: spec.SteamAppIDs,
		Workshop: sdk.SteamWorkshopSpec{
			AllowCoexistence: true,
			Actions:          sdk.StandardSteamWorkshopActions(),
		},
	})
	for _, ref := range spec.Sources {
		r.RegisterSource(ref)
	}
}
