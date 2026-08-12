package neverwinter

import (
	"context"
	"os"
	"path/filepath"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	NWNID    = "nwn"
	NWNEEID  = "nwnee"
	NWN2ID   = "neverwinter2"
	NWNEEApp = "704450"

	nwnDocumentsRootID  = "nwnee-documents"
	nwn2DocumentsRootID = "nwn2-documents"
	nwn2OverrideRootID  = "nwn2-documents-override"
)

func Extensions() []sdk.Extension {
	return []sdk.Extension{
		{
			ID:       NWNID,
			Name:     "Neverwinter Nights",
			Kind:     sdk.ExtensionKindGame,
			Version:  "1.0.0-dmm.1",
			BuildID:  "first-party-go",
			Register: RegisterNWN,
		},
		{
			ID:       NWNEEID,
			Name:     "Neverwinter Nights: Enhanced Edition",
			Kind:     sdk.ExtensionKindGame,
			Version:  "1.0.0-dmm.1",
			BuildID:  "first-party-go",
			Register: RegisterNWNEE,
		},
		{
			ID:       NWN2ID,
			Name:     "Neverwinter Nights 2",
			Kind:     sdk.ExtensionKindGame,
			Version:  "1.0.0-dmm.1",
			BuildID:  "first-party-go",
			Register: RegisterNWN2,
		},
	}
}

func RegisterNWN(r sdk.Registrar) {
	registerNWNGame(r, nwnGameSpec{
		ID:             NWNID,
		Name:           "Neverwinter Nights",
		NexusDomains:   []string{"neverwinter"},
		AllowNoSteamID: true,
		Executable:     "nwmain.exe",
		RequiredFiles:  []string{"nwmain.exe"},
		ModTypePrefix:  "nwn",
		TargetRoot:     "",
	})
}

func RegisterNWNEE(r sdk.Registrar) {
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       nwnDocumentsRootID,
		Name:     "Neverwinter Nights Documents",
		Resolver: documentsRoot("Neverwinter Nights"),
	})
	registerNWNGame(r, nwnGameSpec{
		ID:            NWNEEID,
		Name:          "Neverwinter Nights: Enhanced Edition",
		SteamAppIDs:   []string{NWNEEApp},
		NexusDomains:  []string{"neverwinter"},
		Executable:    "bin/win32/nwmain.exe",
		RequiredFiles: []string{"bin/win32/nwmain.exe"},
		ModTypePrefix: "nwnee",
		TargetRootID:  nwnDocumentsRootID,
	})
}

func RegisterNWN2(r sdk.Registrar) {
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       nwn2DocumentsRootID,
		Name:     "Neverwinter Nights 2 Documents",
		Resolver: documentsRoot("Neverwinter Nights 2"),
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       nwn2OverrideRootID,
		Name:     "Neverwinter Nights 2 Documents Override",
		Resolver: documentsRoot("Neverwinter Nights 2", "override"),
	})
	r.RegisterGame(sdk.GameRegistration{
		NexusDomains:        []string{NWN2ID},
		VortexGameID:        NWN2ID,
		AllowNoSteamAppID:   true,
		ExecutableRelative:  "nwn2.exe",
		RequiredFiles:       []string{"nwn2.exe"},
		QueryModPathDynamic: true,
		MergeMode:           sdk.GameMergeModeAll,
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: "nwn2-module", TargetRootID: nwn2DocumentsRootID})
	r.RegisterModType(installplan.ModTypeSpec{ID: "nwn2-override-mod", TargetRootID: nwn2OverrideRootID})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:neverwinter2:module",
		VortexInstallerID: "moduleinstaller",
		Priority:          25,
		ModType:           "nwn2-module",
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchNWN2ModuleArchive,
		CustomBuild:       buildNWN2ModuleArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:   "neverwinter2-prepare-mod-folders",
		Name: "Prepare Neverwinter Nights 2 module and override folders",
		Actions: append(
			sdk.EnsureTargetRootDirectories(nwn2DocumentsRootID, "modules"),
			sdk.EnsureTargetRootDirectories(nwn2OverrideRootID, ".")...,
		),
	})
	for _, ref := range nwn2Sources() {
		r.RegisterSource(ref)
	}
}

type nwnGameSpec struct {
	ID             string
	Name           string
	SteamAppIDs    []string
	NexusDomains   []string
	AllowNoSteamID bool
	Executable     string
	RequiredFiles  []string
	ModTypePrefix  string
	TargetRoot     string
	TargetRootID   string
}

func registerNWNGame(r sdk.Registrar, spec nwnGameSpec) {
	structuredType := spec.ModTypePrefix + "-structured"
	looseType := spec.ModTypePrefix + "-loose"
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:         spec.SteamAppIDs,
		NexusDomains:        spec.NexusDomains,
		VortexGameID:        spec.ID,
		AllowNoSteamAppID:   spec.AllowNoSteamID,
		ExecutableRelative:  spec.Executable,
		RequiredFiles:       spec.RequiredFiles,
		QueryModPath:        spec.TargetRoot,
		QueryModPathDynamic: spec.TargetRootID != "",
		MergeMode:           sdk.GameMergeModeAll,
		Environment:         steamEnvironment(spec.SteamAppIDs),
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: structuredType, TargetRoot: spec.TargetRoot, TargetRootID: spec.TargetRootID})
	r.RegisterModType(installplan.ModTypeSpec{ID: looseType, TargetRoot: spec.TargetRoot, TargetRootID: spec.TargetRootID})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:" + spec.ID + ":structured",
		VortexInstallerID: "nwn-structured-default",
		Priority:          20,
		ModType:           structuredType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchNWNStructuredArchive,
		CustomBuild:       buildNWNStructuredArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:" + spec.ID + ":loose",
		VortexInstallerID: "nwn-mod",
		Priority:          25,
		ModType:           looseType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchNWNLooseArchive,
		CustomBuild:       buildNWNLooseArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:      spec.ID + "-prepare-mod-folders",
		Name:    "Prepare " + spec.Name + " mod folders",
		Actions: nwnSetupActions(spec.TargetRootID),
	})
	for _, ref := range nwnSources() {
		r.RegisterSource(ref)
	}
}

func nwnSetupActions(rootID string) []sdk.GameSetupActionSpec {
	paths := make([]string, 0, len(nwnDestinations))
	seen := make(map[string]struct{}, len(nwnDestinations))
	for _, destination := range nwnDestinations {
		if _, ok := seen[destination]; ok {
			continue
		}
		seen[destination] = struct{}{}
		paths = append(paths, destination)
	}
	if rootID != "" {
		return sdk.EnsureTargetRootDirectories(rootID, paths...)
	}
	return sdk.EnsureGameDirectories(paths...)
}

func documentsRoot(rel ...string) sdk.TargetRootResolverFunc {
	return func(ctx context.Context, input sdk.TargetRootInput) (sdk.TargetRootResult, error) {
		if err := ctx.Err(); err != nil {
			return sdk.TargetRootResult{}, err
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return sdk.TargetRootResult{}, err
		}
		parts := append([]string{home, "Documents"}, rel...)
		return sdk.TargetRootResult{Path: filepath.Join(parts...), Source: "Vortex Documents path"}, nil
	}
}

func steamEnvironment(appIDs []string) map[string]string {
	if len(appIDs) == 0 {
		return nil
	}
	return map[string]string{"SteamAPPId": appIDs[0]}
}

func nwnSources() []sdk.SourceRef {
	return []sdk.SourceRef{{
		Name: "Vortex game-neverwinter-nights extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-neverwinter-nights/src",
	}}
}

func nwn2Sources() []sdk.SourceRef {
	return []sdk.SourceRef{{
		Name: "Vortex game-neverwinter-nights2 extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-neverwinter-nights2/src",
	}}
}
