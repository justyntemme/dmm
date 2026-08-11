package codevein

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/unreal"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
	"github.com/justyntemme/decky-mod-manager/internal/peversion"
)

const (
	SteamAppID   = "678960"
	VortexGameID = "codevein"
	Name         = "Code Vein"

	executableRelative = "CodeVein/Binaries/Win64/CodeVein-Win64-Shipping.exe"
	pakModType         = "codevein-pak"
	pakRoot            = "CodeVein/content/paks/~mods"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Kind:     sdk.ExtensionKindGame,
		Version:  "1.0.0-dmm.1",
		BuildID:  "first-party-go",
		Register: Register,
	}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:        []string{SteamAppID},
		NexusDomains:       []string{VortexGameID},
		VortexGameID:       VortexGameID,
		ExecutableRelative: executableRelative,
		RequiredFiles:      []string{executableRelative},
		QueryModPath:       pakRoot,
		MergeMode:          sdk.GameMergeModeDynamic,
		RequiresCleanup:    true,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: pakModType, TargetRoot: pakRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:codevein:pak",
		VortexInstallerID: "codevein-mod",
		Priority:          25,
		ModType:           pakModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchPakArchive,
		CustomBuild:       buildPakArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterMerge(sdk.MergeSpec{ID: "codevein-pak-load-order", Name: "Code Vein pak load order"})
	r.RegisterLoadOrder(sdk.LoadOrderSpec{
		ID:             "codevein-pak-load-order",
		Name:           "Code Vein pak load order",
		TargetRoot:     pakRoot,
		ModTypes:       []string{pakModType},
		FileExtensions: []string{".pak", ".ucas", ".utoc"},
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event: "will-deploy",
		Name:  "Apply Code Vein pak load order prefixes",
		Handler: unreal.SortablePakLoadOrderHandler(unreal.SortablePakLoadOrderOptions{
			TargetRoot: pakRoot,
			ModType:    pakModType,
		}),
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "codevein-executable-version",
		Name:     "Code Vein executable version",
		Provider: gameVersion,
	})
	r.RegisterExternalModAdoption(sdk.ExternalModAdoptionSpec{
		ID:             "codevein-external-pak-adoption",
		Name:           "Import unmanaged Code Vein PAK files",
		TargetRelative: pakRoot,
		ModType:        pakModType,
		FileExtensions: []string{".pak", ".ucas", ".utoc"},
		DeleteOriginal: true,
		Message:        "Source-backed Vortex setup parity: unmanaged files in CodeVein/content/paks/~mods can be adopted into DMM-owned staging and removed from the game folder.",
	})
	r.RegisterStateMigration(sdk.StateMigrationSpec{
		ID:          "codevein-load-order-migration-1.0.0",
		Name:        "Code Vein load-order state migration",
		FromVersion: "0.0.0",
		ToVersion:   "1.0.0",
		Commands: []sdk.StateMigrationCommandSpec{
			{ID: "purge-old-paks", Name: "Purge old Code Vein PAK deployment", Command: sdk.StateMigrationCommandPurgeModsInPath, TargetRelative: pakRoot},
			{ID: "deploy-profile", Name: "Redeploy active profile", Command: sdk.StateMigrationCommandDeployProfile},
		},
		Message: "Mirrors the deploy side of Vortex 1.0.0 migration by purging the old pak deployment path and redeploying; DMM's load-order handler writes current profile order directly from DMM state.",
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func gameVersion(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.GameVersionResult{}, err
	}
	gamePath := strings.TrimSpace(input.GamePath)
	if gamePath == "" {
		return sdk.GameVersionResult{}, nil
	}
	version, err := peversion.FileVersion(filepath.Join(gamePath, filepath.FromSlash(executableRelative)))
	if err != nil {
		return sdk.GameVersionResult{}, err
	}
	return sdk.GameVersionResult{Version: version, Source: executableRelative}, nil
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex game-codevein extension source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-codevein/src",
		},
	}
}
