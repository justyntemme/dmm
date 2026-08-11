package x4foundations

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "392160"
	VortexGameID = "x4foundations"
	Name         = "X4: Foundations"

	gameExtensionsRoot = "extensions"
	documentsRootID    = "x4-documents-extensions"
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
		},
		Workshop: sdk.SteamWorkshopSpec{
			AllowCoexistence: true,
			Actions:          sdk.StandardSteamWorkshopActions(),
		},
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       documentsRootID,
		Name:     "X4 Documents extensions",
		Resolver: documentsExtensionsRoot,
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: "x4-extensions", TargetRoot: gameExtensionsRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: "x4-documents-modtype", TargetRootID: documentsRootID})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:x4foundations:content",
		VortexInstallerID: "x4foundations",
		Priority:          50,
		ModType:           "x4-extensions",
		NameSource:        installplan.NameSourceManifestDisplay,
		CustomMatch:       matchContentArchive,
		CustomBuild:       buildContentArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "x4-version-file",
		Name:     "X4 version.dat",
		Provider: gameVersion,
	})
	r.RegisterStateMigration(sdk.StateMigrationSpec{
		ID:          "x4foundations-1.0.1-invalid-folder-migration",
		Name:        "X4 invalid ws_ folder reinstall warning",
		FromVersion: "0.0.0",
		ToVersion:   "1.0.1",
		Status:      sdk.CapabilityStatusNotApplicable,
		Message:     "Vortex scans historical staged X4 mods for invalid ws_ folders and warns users to reinstall affected mods. DMM-created state stages content.xml mods with the corrected extension-owned folder rules from the first install; post-MVP Vortex import must scan imported Vortex staging and surface the same reinstall warning when needed.",
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func documentsExtensionsRoot(ctx context.Context, input sdk.TargetRootInput) (sdk.TargetRootResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.TargetRootResult{}, err
	}
	libraryPath := strings.TrimSpace(input.LibraryPath)
	if libraryPath == "" {
		return sdk.TargetRootResult{}, nil
	}
	steamID := firstSteamUserID32(libraryPath)
	segments := []string{
		libraryPath,
		"steamapps",
		"compatdata",
		SteamAppID,
		"pfx",
		"drive_c",
		"users",
		"steamuser",
		"Documents",
		"Egosoft",
		"X4",
	}
	if steamID != "" {
		segments = append(segments, steamID)
	}
	segments = append(segments, "extensions")
	return sdk.TargetRootResult{Path: filepath.Join(segments...), Source: "Steam userdata"}, nil
}

func firstSteamUserID32(libraryPath string) string {
	candidates := []string{
		filepath.Join(libraryPath, "userdata"),
		filepath.Join(filepath.Dir(filepath.Dir(libraryPath)), "userdata"),
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".local", "share", "Steam", "userdata"),
			filepath.Join(home, ".steam", "steam", "userdata"),
		)
	}
	for _, root := range candidates {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		var ids []string
		for _, entry := range entries {
			if !entry.IsDir() || !isDecimal(entry.Name()) {
				continue
			}
			ids = append(ids, entry.Name())
		}
		sort.Strings(ids)
		if len(ids) > 0 {
			return ids[0]
		}
	}
	return ""
}

func isDecimal(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func gameVersion(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.GameVersionResult{}, err
	}
	gamePath := strings.TrimSpace(input.GamePath)
	if gamePath == "" {
		return sdk.GameVersionResult{}, nil
	}
	data, err := os.ReadFile(filepath.Join(gamePath, "version.dat"))
	if err != nil {
		return sdk.GameVersionResult{}, err
	}
	return sdk.GameVersionResult{Version: strings.TrimSpace(string(data)), Source: "version.dat"}, nil
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex X4: Foundations game extension",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-x4foundations/src/index.js",
		},
	}
}
