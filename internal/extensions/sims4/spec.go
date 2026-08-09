package sims4

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "1222670"
	VortexGameID = "thesims4"
	Name         = "The Sims 4"

	userDataRootID = "sims4-user-data"
	modType        = "sims4mixed"
)

var localeModFolders = []string{
	"The Sims 4",
	"Die Sims 4",
	"Los Sims 4",
	"Les\u00a0Sims\u00a04",
	"De Sims 4",
}

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
		SteamAppIDs:         []string{SteamAppID},
		NexusDomains:        []string{VortexGameID},
		VortexGameID:        VortexGameID,
		AllowNoSteamAppID:   true,
		ExecutableRelative:  "game/bin/TS4_x64.exe",
		RequiredFiles:       []string{"game/bin/TS4_x64.exe"},
		QueryModPathDynamic: true,
		MergeMode:           sdk.GameMergeModeAll,
		Environment:         map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       userDataRootID,
		Name:     "The Sims 4 user data",
		Resolver: userDataRoot,
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRootID: userDataRootID})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:thesims4:mixed",
		VortexInstallerID: "sims4mixed",
		Priority:          25,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		TargetRootID:      userDataRootID,
		CustomMatch:       matchMixedArchive,
		CustomBuild:       buildMixedArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:thesims4:mods",
		VortexInstallerID: "sims4-default-mods",
		Priority:          50,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		TargetRootID:      userDataRootID,
		CustomMatch:       matchModsArchive,
		CustomBuild:       buildMixedArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "exe32bit",
		Name:               "The Sims 4 (32 bit)",
		ExecutableRelative: "game/bin_le/TS4.exe",
		RequiredFiles:      []string{"game/bin/TS4.exe"},
		Relative:           true,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:             "sims4-prepare-modding",
		Name:           "Prepare The Sims 4 mod folder and Resource.cfg",
		GeneratedFiles: []string{"Mods/Resource.cfg", "Options.ini"},
	})
	r.RegisterStateMigration(sdk.StateMigrationSpec{
		ID:          "sims4-vortex-mods-migration-2.0.0",
		Name:        "Purge legacy Vortex Mods deployment paths",
		FromVersion: "0.0.1",
		ToVersion:   "2.0.1",
		Status:      sdk.CapabilityStatusMetadata,
		Message:     "Vortex purges legacy profile-local Vortex Mods paths during migration; DMM records this source behavior but pre-MVP does not import or mutate Vortex state.",
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventWillDeploy,
		Name:    "Prepare The Sims 4 Resource.cfg and Options.ini",
		Handler: willDeploy,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func userDataRoot(ctx context.Context, input sdk.TargetRootInput) (sdk.TargetRootResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.TargetRootResult{}, err
	}
	for _, base := range candidateElectronicArtsRoots(input) {
		if root, ok := existingLocalizedRoot(base); ok {
			return sdk.TargetRootResult{Path: root, Source: "Vortex documents Electronic Arts localized Sims 4 folder"}, nil
		}
	}
	for _, base := range candidateElectronicArtsRoots(input) {
		if strings.TrimSpace(base) != "" {
			return sdk.TargetRootResult{Path: filepath.Join(base, "The Sims 4"), Source: "Vortex default English Sims 4 folder under Electronic Arts"}, nil
		}
	}
	return sdk.TargetRootResult{}, errors.New("unable to resolve The Sims 4 user data folder")
}

func candidateElectronicArtsRoots(input sdk.TargetRootInput) []string {
	var bases []string
	if library := strings.TrimSpace(input.LibraryPath); library != "" {
		bases = append(bases, filepath.Join(library, "steamapps", "compatdata", SteamAppID, "pfx", "drive_c", "users", "steamuser", "Documents", "Electronic Arts"))
	}
	if library := libraryFromGamePath(input.GamePath); library != "" {
		bases = append(bases, filepath.Join(library, "steamapps", "compatdata", SteamAppID, "pfx", "drive_c", "users", "steamuser", "Documents", "Electronic Arts"))
	}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		bases = append(bases, filepath.Join(home, "Documents", "Electronic Arts"))
	}
	return dedupePaths(bases)
}

func libraryFromGamePath(gamePath string) string {
	gamePath = filepath.Clean(strings.TrimSpace(gamePath))
	if gamePath == "" || gamePath == "." {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(gamePath), "/")
	for i := 0; i < len(parts)-1; i++ {
		if strings.EqualFold(parts[i], "steamapps") && i > 0 {
			return filepath.FromSlash(strings.Join(parts[:i], "/"))
		}
	}
	return ""
}

func existingLocalizedRoot(electronicArtsRoot string) (string, bool) {
	for _, folder := range localeModFolders {
		candidate := filepath.Join(electronicArtsRoot, folder)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, true
		}
	}
	return "", false
}

func dedupePaths(paths []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		path = filepath.Clean(strings.TrimSpace(path))
		if path == "" || path == "." {
			continue
		}
		key := strings.ToLower(path)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, path)
	}
	return out
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{{
		Name: "Vortex game-sims4 extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-sims4/src",
	}}
}
