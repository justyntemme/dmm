package stellaris

import "github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"

const (
	SteamAppID = "281990"
	ID         = "stellaris"
	Name       = "Stellaris"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       ID,
		Name:     Name,
		Version:  "0.1.0",
		BuildID:  "first-party-go",
		Register: Register,
	}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs: []string{SteamAppID},
		Workshop: sdk.SteamWorkshopSpec{
			AllowCoexistence: true,
			Actions:          sdk.StandardSteamWorkshopActions(),
		},
	})
	r.RegisterSource(sdk.SourceRef{
		Name: "Steam Deck installed app manifest snapshot; no official Vortex Stellaris extension found in checked-out source",
		URL:  "extensionTargets.md#priority-queue",
	})
}
