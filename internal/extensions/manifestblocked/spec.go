package manifestblocked

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

type Spec struct {
	ID                     string
	Name                   string
	Version                string
	BuildID                string
	SteamAppIDs            []string
	NexusDomains           []string
	VortexGameID           string
	ModTypeID              string
	InstallerID            string
	VortexInstallerID      string
	UnsupportedReason      string
	RequiredFiles          []string
	RequirementID          string
	RequirementName        string
	RequirementMessage     string
	RequirementOKMessage   string
	RequirementInstallHint string
	Sources                []sdk.SourceRef
}

func Extension(spec Spec) sdk.Extension {
	if strings.TrimSpace(spec.Version) == "" {
		spec.Version = "0.1.0"
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
			Register(r, spec)
		},
	}
}

func Register(r sdk.Registrar, spec Spec) {
	modType := strings.TrimSpace(spec.ModTypeID)
	if modType == "" {
		modType = strings.TrimSpace(spec.ID) + "-research-blocked"
	}
	installerID := strings.TrimSpace(spec.InstallerID)
	if installerID == "" {
		installerID = "research:" + strings.TrimSpace(spec.ID) + ":blocked"
	}
	vortexInstallerID := strings.TrimSpace(spec.VortexInstallerID)
	if vortexInstallerID == "" {
		vortexInstallerID = modType
	}
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:  spec.SteamAppIDs,
		NexusDomains: spec.NexusDomains,
		VortexGameID: spec.VortexGameID,
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: ""})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                installerID,
		VortexInstallerID: vortexInstallerID,
		Priority:          10000,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       func(string) bool { return true },
		InstructionMode:   installplan.InstructionUnsupported,
		UnsupportedReason: strings.TrimSpace(spec.UnsupportedReason),
	})
	if len(spec.RequiredFiles) > 0 {
		r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
			ID:          defaultString(spec.RequirementID, strings.TrimSpace(spec.ID)+"-required-files"),
			Name:        defaultString(spec.RequirementName, strings.TrimSpace(spec.Name)+" install files"),
			Kind:        "game-files",
			Required:    true,
			ModTypes:    []string{modType},
			Message:     spec.RequirementMessage,
			OKMessage:   spec.RequirementOKMessage,
			InstallHint: spec.RequirementInstallHint,
			Check:       RequiredFilesCheck(spec.RequiredFiles),
		})
	}
	for _, ref := range spec.Sources {
		r.RegisterSource(ref)
	}
}

func RequiredFilesCheck(required []string) func(context.Context, string) []string {
	cleaned := make([]string, 0, len(required))
	for _, rel := range required {
		rel = strings.TrimSpace(rel)
		if rel != "" {
			cleaned = append(cleaned, filepath.ToSlash(rel))
		}
	}
	return func(ctx context.Context, gamePath string) []string {
		if err := ctx.Err(); err != nil {
			return nil
		}
		gamePath = strings.TrimSpace(gamePath)
		if gamePath == "" {
			return nil
		}
		details := make([]string, 0, len(cleaned))
		for _, rel := range cleaned {
			path := filepath.Join(gamePath, filepath.FromSlash(rel))
			if info, err := os.Stat(path); err == nil {
				if info.IsDir() {
					details = append(details, filepath.ToSlash(path)+"/")
				} else {
					details = append(details, filepath.ToSlash(path))
				}
			}
		}
		if len(details) != len(cleaned) {
			return nil
		}
		return details
	}
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return fallback
}
