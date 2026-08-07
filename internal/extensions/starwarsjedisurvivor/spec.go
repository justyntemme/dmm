package starwarsjedisurvivor

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/unreal"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "1774580"
	VortexGameID = "starwarsjedisurvivor"
	Name         = "Star Wars Jedi: Survivor"

	pakRoot       = "SwGame/Content/Paks/~mods"
	pakModType    = "starwarsjedi2-pak-modtype"
	loaderModType = "starwarsjedi2-r457loader"
	r457LoaderPak = "zR457ModLoader.pak"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Version:  "0.1.0",
		BuildID:  "first-party-go",
		Register: Register,
	}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:  []string{SteamAppID},
		NexusDomains: []string{VortexGameID},
		VortexGameID: VortexGameID,
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
			DefaultStrategy:       installplan.DeployStrategyCopy,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: pakModType, TargetRoot: pakRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: loaderModType, TargetRoot: ""})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:starwarsjedisurvivor:r457loader",
		VortexInstallerID: "starwarsjedi2-r457loader",
		Priority:          25,
		ModType:           loaderModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchR457Loader,
		CustomBuild:       buildR457Loader,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:starwarsjedisurvivor:pak",
		VortexInstallerID: "starwarsjedi2-mod",
		Priority:          25,
		ModType:           pakModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchPakArchive,
		CustomBuild:       buildPakArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterMerge(sdk.MergeSpec{ID: "starwarsjedi2-pak-load-order", Name: "Star Wars Jedi: Survivor pak load order"})
	r.RegisterLoadOrder(sdk.LoadOrderSpec{ID: "starwarsjedi2-pak-load-order", Name: "Star Wars Jedi: Survivor pak load order"})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event: "will-deploy",
		Name:  "Apply Star Wars Jedi: Survivor pak load order prefixes",
		Handler: unreal.SortablePakLoadOrderHandler(unreal.SortablePakLoadOrderOptions{
			TargetRoot: pakRoot,
			ModType:    pakModType,
		}),
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex Star Wars Jedi: Survivor extension source",
			URL:  "https://github.com/Pickysaurus/vortex-jedi-survivor",
		},
		{
			Name: "Live Steam Deck executable/path verification",
			URL:  "extensionTargets.md#priority-queue",
		},
	}
}
