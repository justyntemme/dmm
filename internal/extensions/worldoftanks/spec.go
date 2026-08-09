package worldoftanks

import (
	"context"
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	VortexGameID = "worldoftanks"
	Name         = "World of Tanks"

	versionedResModsRootID = "worldoftanks-versioned-res-mods"
	modType                = "worldoftanks-versioned-res-mods"
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
		NexusDomains:        []string{VortexGameID},
		VortexGameID:        VortexGameID,
		AllowNoSteamAppID:   true,
		ExecutableRelative:  "WorldOfTanks.exe",
		RequiredFiles:       []string{"WorldOfTanks.exe", "version.xml"},
		QueryModPathDynamic: true,
		MergeMode:           sdk.GameMergeModeAll,
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       versionedResModsRootID,
		Name:     "World of Tanks versioned res_mods",
		Resolver: versionedResModsRoot,
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRootID: versionedResModsRootID})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:worldoftanks:default",
		VortexInstallerID: "worldoftanks-default",
		Priority:          25,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		TargetRootID:      versionedResModsRootID,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

type versionXML struct {
	Version string `xml:"version"`
}

func versionedResModsRoot(ctx context.Context, input sdk.TargetRootInput) (sdk.TargetRootResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.TargetRootResult{}, err
	}
	gamePath := strings.TrimSpace(input.GamePath)
	if gamePath == "" {
		return sdk.TargetRootResult{}, errors.New("game path is required to resolve World of Tanks versioned res_mods path")
	}
	version, err := readGameVersion(filepath.Join(gamePath, "version.xml"))
	if err != nil {
		return sdk.TargetRootResult{}, err
	}
	return sdk.TargetRootResult{
		Path:   filepath.Join(gamePath, "res_mods", version),
		Source: "Vortex version.xml res_mods path",
	}, nil
}

func readGameVersion(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var parsed versionXML
	if err := xml.Unmarshal(data, &parsed); err != nil {
		return "", err
	}
	version := normalizeVersion(parsed.Version)
	if version == "" {
		return "", errors.New("World of Tanks version.xml did not contain a usable version")
	}
	return version, nil
}

var versionPattern = regexp.MustCompile(`v\.([0-9.]+)`)

func normalizeVersion(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if match := versionPattern.FindStringSubmatch(value); len(match) == 2 {
		return strings.Trim(match[1], ".")
	}
	value = strings.TrimPrefix(value, "v")
	value = strings.TrimPrefix(value, ".")
	return strings.TrimSpace(value)
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{{
		Name: "Vortex game-worldoftanks extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-worldoftanks/src",
	}}
}
