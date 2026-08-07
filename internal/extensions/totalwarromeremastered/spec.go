package totalwarromeremastered

import "github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"

const (
	SteamAppID = "885970"
	ID         = "totalwarromeremastered"
	Name       = "Total War: ROME REMASTERED"
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
		Name: "Steam Deck installed app and Workshop manifest snapshot; no verified Nexus/Vortex extension assigned",
		URL:  "extensionTargets.md#priority-queue",
	})
}
