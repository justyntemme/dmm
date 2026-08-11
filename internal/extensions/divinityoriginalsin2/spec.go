package divinityoriginalsin2

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/targetroots"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID = "435150"

	OriginalVortexGameID   = "divinityoriginalsin2"
	DefinitiveVortexGameID = "divinityoriginalsin2definitiveedition"

	OriginalName   = "Divinity: Original Sin 2 Original Edition"
	DefinitiveName = "Divinity: Original Sin 2 Definitive Edition"

	originalModType   = "divinityoriginalsin2-pak"
	definitiveModType = "divinityoriginalsin2definitiveedition-pak"

	originalRootID   = "divinityoriginalsin2-documents-mods"
	definitiveRootID = "divinityoriginalsin2definitiveedition-documents-mods"
)

type variant struct {
	ID                 string
	Name               string
	ExecutableRelative string
	RequiredFiles      []string
	VersionFiles       []string
	DocumentsFolder    string
	ModType            string
	TargetRootID       string
	Logo               string
}

var variants = []variant{
	{
		ID:                 OriginalVortexGameID,
		Name:               OriginalName,
		ExecutableRelative: "bin/SupportTool.exe",
		RequiredFiles:      []string{"bin/SupportTool.exe"},
		VersionFiles:       []string{"bin/EoCApp.exe", "Classic/EoCApp.exe"},
		DocumentsFolder:    "Divinity Original Sin 2",
		ModType:            originalModType,
		TargetRootID:       originalRootID,
		Logo:               "gameart.jpg",
	},
	{
		ID:                 DefinitiveVortexGameID,
		Name:               DefinitiveName,
		ExecutableRelative: "DefEd/bin/SupportTool.exe",
		RequiredFiles:      []string{"DefEd/bin/SupportTool.exe"},
		VersionFiles:       []string{"DefEd/bin/EoCApp.exe"},
		DocumentsFolder:    "Divinity Original Sin 2 Definitive Edition",
		ModType:            definitiveModType,
		TargetRootID:       definitiveRootID,
		Logo:               "gameartDE.png",
	},
}

func Extensions() []sdk.Extension {
	out := make([]sdk.Extension, 0, len(variants))
	for _, current := range variants {
		current := current
		out = append(out, sdk.Extension{
			ID:      current.ID,
			Name:    current.Name,
			Kind:    sdk.ExtensionKindGame,
			Version: "1.0.0-dmm.1",
			BuildID: "first-party-go",
			Register: func(r sdk.Registrar) {
				registerVariant(r, current)
			},
		})
	}
	return out
}

func registerVariant(r sdk.Registrar, current variant) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:         []string{SteamAppID},
		NexusDomains:        []string{current.ID},
		VortexGameID:        current.ID,
		ExecutableRelative:  current.ExecutableRelative,
		RequiredFiles:       current.RequiredFiles,
		QueryModPathDynamic: true,
		MergeMode:           sdk.GameMergeModeAll,
		Environment:         map[string]string{"SteamAPPId": SteamAppID},
		Deployment:          installplan.DeploymentSpec{AllowNeedsReviewState: true},
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       current.TargetRootID,
		Name:     current.Name + " Proton Documents Mods",
		Resolver: targetroots.ProtonDocuments(SteamAppID, "Larian Studios", current.DocumentsFolder, "Mods"),
	})
	r.RegisterModType(installplan.ModTypeSpec{
		ID:           current.ModType,
		TargetRootID: current.TargetRootID,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:" + current.ID + ":mods",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           current.ModType,
		NameSource:        installplan.NameSourceArchive,
		TargetRootID:      current.TargetRootID,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:      current.ID + "-ensure-documents-mods",
		Name:    "Ensure " + current.Name + " Documents Mods folder",
		Actions: sdk.EnsureTargetRootDirectories(current.TargetRootID, "."),
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       current.ID + "-exe-version-marker",
		Name:     current.Name + " executable version marker",
		Provider: versionMarkerProvider(current),
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventDidDeploy,
		Name:    current.Name + " in-game enable reminder",
		Handler: didDeployPakReminder,
	})
	r.RegisterGameStore(sdk.GameStoreSpec{
		ID:      current.ID + "-gog-registry",
		Name:    current.Name + " GOG registry discovery",
		Status:  sdk.CapabilityStatusNotApplicable,
		Message: "Vortex can discover this game from the Windows GOG registry key before falling back to Steam. DMM's Steam Deck MVP does not discover GOG installs yet.",
	})
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex game-divinityoriginalsin2 extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-divinityoriginalsin2/src",
	})
}

func versionMarkerProvider(current variant) sdk.GameVersionProviderFunc {
	return func(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
		if err := ctx.Err(); err != nil {
			return sdk.GameVersionResult{}, err
		}
		gamePath := strings.TrimSpace(input.GamePath)
		if gamePath == "" {
			return sdk.GameVersionResult{}, nil
		}
		for _, rel := range current.VersionFiles {
			path := filepath.Join(gamePath, filepath.FromSlash(rel))
			info, err := os.Stat(path)
			if err != nil || info.IsDir() {
				continue
			}
			return sdk.GameVersionResult{Version: "installed", Source: rel}, nil
		}
		return sdk.GameVersionResult{}, nil
	}
}

func didDeployPakReminder(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if !containsPakDeployment(input.Mappings, input.ManagedFiles) {
		return sdk.EventHandlerResult{}, nil
	}
	return sdk.EventHandlerResult{
		Notices: []sdk.EventNotice{{
			Message: "Please remember to enable mods in-game",
		}},
	}, nil
}

func containsPakDeployment(mappings []deploy.FileMapping, files []deploy.AppliedFile) bool {
	for _, mapping := range mappings {
		if strings.EqualFold(filepath.Ext(mapping.TargetRelative), ".pak") {
			return true
		}
	}
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file.TargetPath), ".pak") {
			return true
		}
	}
	return false
}
