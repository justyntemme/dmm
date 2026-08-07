package halflife2

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	HalfLife2AppID       = "220"
	VortexGameID         = "halflife2"
	VortexInternalGameID = "half-life2"
	Name                 = "Half-Life 2"

	vpkModType = "halflife2-vpk"
	vpkRoot    = "hl2/custom"
)

var (
	requiredGameFiles = []string{"hl2/gameinfo.txt"}
	executableMarkers = []string{"hl2_linux", "hl2.sh", "hl2.exe"}
)

type SourceVPKSpec struct {
	ID                string
	Name              string
	Version           string
	BuildID           string
	SteamAppIDs       []string
	NexusDomains      []string
	VortexGameID      string
	VPKModType        string
	TargetRoot        string
	InstallerID       string
	VortexInstallerID string
	RequiredFiles     []string
	Sources           []sdk.SourceRef
}

func Extension() sdk.Extension {
	return SourceVPKExtension(SourceVPKSpec{
		ID:                VortexGameID,
		Name:              Name,
		Version:           "1.1.0-dmm.1",
		BuildID:           "first-party-go",
		SteamAppIDs:       []string{HalfLife2AppID},
		NexusDomains:      []string{VortexGameID},
		VortexGameID:      VortexGameID,
		VPKModType:        vpkModType,
		TargetRoot:        vpkRoot,
		InstallerID:       "vortex:halflife2:vpk",
		VortexInstallerID: "half-life2-mod",
		RequiredFiles:     requiredGameFiles,
		Sources:           sources(),
	})
}

func Register(r sdk.Registrar) {
	RegisterSourceVPK(r, SourceVPKSpec{
		ID:                VortexGameID,
		Name:              Name,
		SteamAppIDs:       []string{HalfLife2AppID},
		NexusDomains:      []string{VortexGameID},
		VortexGameID:      VortexGameID,
		VPKModType:        vpkModType,
		TargetRoot:        vpkRoot,
		InstallerID:       "vortex:halflife2:vpk",
		VortexInstallerID: "half-life2-mod",
		RequiredFiles:     requiredGameFiles,
		Sources:           sources(),
	})
}

func SourceVPKExtension(spec SourceVPKSpec) sdk.Extension {
	if strings.TrimSpace(spec.Version) == "" {
		spec.Version = "1.1.0-dmm.1"
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
			RegisterSourceVPK(r, spec)
		},
	}
}

func RegisterSourceVPK(r sdk.Registrar, spec SourceVPKSpec) {
	id := strings.TrimSpace(spec.ID)
	modType := defaultString(spec.VPKModType, id+"-vpk")
	targetRoot := strings.Trim(strings.TrimSpace(spec.TargetRoot), "/")
	installerID := defaultString(spec.InstallerID, "vortex:"+id+":vpk")
	vortexInstallerID := defaultString(spec.VortexInstallerID, "half-life2-mod")
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:  spec.SteamAppIDs,
		NexusDomains: spec.NexusDomains,
		VortexGameID: defaultString(spec.VortexGameID, id),
		Deployment: installplan.DeploymentSpec{
			DefaultStrategy:       installplan.DeployStrategySymlink,
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: targetRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                installerID,
		VortexInstallerID: vortexInstallerID,
		Priority:          25,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchVPKArchive,
		CustomBuild: func(input installplan.BuildInput) (installplan.Plan, error) {
			return buildVPKArchive(input, targetRoot)
		},
		InstructionMode: installplan.InstructionCustom,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          id + "-required-files",
		Name:        spec.Name + " install files",
		Kind:        "game-files",
		Required:    true,
		ModTypes:    []string{modType},
		Message:     spec.Name + " is missing files needed for VPK deployment.",
		OKMessage:   spec.Name + " contains the expected executable marker and Source gameinfo.txt.",
		InstallHint: "Verify " + spec.Name + " files in Steam before testing Source VPK mods.",
		Check:       requiredFilesCheck(spec.RequiredFiles),
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       id + "-executable",
		Name:     spec.Name + " executable marker",
		Provider: gameVersion,
	})
	for _, ref := range spec.Sources {
		r.RegisterSource(ref)
	}
}

func requiredFilesCheck(requiredFiles []string) func(context.Context, string) []string {
	required := make([]string, 0, len(requiredFiles))
	for _, rel := range requiredFiles {
		rel = strings.TrimSpace(rel)
		if rel != "" {
			required = append(required, filepath.ToSlash(rel))
		}
	}
	return func(ctx context.Context, gamePath string) []string {
		if err := ctx.Err(); err != nil {
			return nil
		}
		gamePath = strings.TrimSpace(gamePath)
		marker := firstExistingFile(gamePath, executableMarkers)
		if gamePath == "" || marker == "" {
			return nil
		}
		details := make([]string, 0, len(required)+1)
		details = append(details, filepath.ToSlash(marker))
		for _, rel := range required {
			path := filepath.Join(gamePath, filepath.FromSlash(rel))
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				details = append(details, filepath.ToSlash(path))
				continue
			}
			return nil
		}
		return details
	}
}

func gameVersion(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.GameVersionResult{}, err
	}
	if marker := firstExistingFile(input.GamePath, executableMarkers); marker != "" {
		return sdk.GameVersionResult{Version: "installed", Source: filepath.ToSlash(strings.TrimPrefix(marker, strings.TrimRight(input.GamePath, string(filepath.Separator))+string(filepath.Separator)))}, nil
	}
	return sdk.GameVersionResult{}, os.ErrNotExist
}

func firstExistingFile(gamePath string, rels []string) string {
	for _, rel := range rels {
		path := filepath.Join(gamePath, filepath.FromSlash(rel))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return fallback
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex central extension manifest entry site-mod-80-file-516",
			URL:  "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json",
		},
		{
			Name: "Half-Life 2 Vortex extension package v1.1.0",
			URL:  "https://www.nexusmods.com/site/mods/80?tab=files",
		},
		{
			Name: "Nexus API domain verification for halflife2",
			URL:  "https://api.nexusmods.com/v1/games/halflife2.json",
		},
		{
			Name: "Live Steam Deck executable/path verification",
			URL:  "extensionTargets.md#installed-games-snapshot",
		},
	}
}
