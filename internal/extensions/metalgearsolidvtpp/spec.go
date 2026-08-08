package metalgearsolidvtpp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "287700"
	VortexGameID = "metalgearsolidvtpp"
	Name         = "METAL GEAR SOLID V: THE PHANTOM PAIN"

	snakeBiteModType = "metalgearsolidvtpp-snakebite"
)

var requiredGameFiles = []string{
	"mgsvtpp.exe",
	"master/0/00.dat",
	"master/0/01.dat",
}

const snakeBiteDeployBlockedReason = "MGSV SnakeBite packages patch packed QAR/FPK game archives. DMM has staged this package safely, but enabling it requires the generic packed-archive mutation engine before DMM can rebuild master/0/00.dat without corrupting game data."

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
			DefaultStrategy: installplan.DeployStrategyCopy,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{
		ID:             snakeBiteModType,
		DeploymentMode: installplan.ModTypeDeploymentEventHook,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "snakebite:metalgearsolidvtpp:mgsv-package",
		VortexInstallerID: "metalgearsolidvtpp-snakebite-mgsv",
		Priority:          10,
		ModType:           snakeBiteModType,
		NameSource:        installplan.NameSourceManifestDisplay,
		CustomMatch:       matchSnakeBitePackage,
		CustomBuild:       buildSnakeBitePackagePlan,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterPackedArchiveMutation(sdk.PackedArchiveMutationSpec{
		ID:                "snakebite:metalgearsolidvtpp:qar-fpk",
		Name:              "SnakeBite QAR/FPK package deployment",
		PackageFormat:     "snakebite-mgsv",
		StateFileRelative: "snakebite.xml",
		TargetArchives:    []string{"master/0/00.dat", "master/0/01.dat"},
		RequiresEngine:    "gzs-qar-fpk",
		ModTypes:          []string{snakeBiteModType},
	})
	r.RegisterMerge(sdk.MergeSpec{ID: "snakebite:metalgearsolidvtpp:qar-fpk", Name: "SnakeBite QAR/FPK merge"})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   "will-deploy",
		Name:    "Apply SnakeBite packed archive mutations",
		Handler: willDeploySnakeBitePackages,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "metalgearsolidvtpp-required-files",
		Name:        "MGSV install files",
		Kind:        "game-files",
		Required:    true,
		ModTypes:    []string{snakeBiteModType},
		Message:     "The MGSV game folder is missing files required by SnakeBite-compatible mod support.",
		OKMessage:   "The MGSV game folder contains the packed archives required by SnakeBite-compatible mod support.",
		InstallHint: "Verify the game files in Steam before installing MGSV mods.",
		Check:       checkRequiredGameFiles,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func willDeploySnakeBitePackages(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	for _, mod := range input.Mods {
		if !mod.Enabled || !strings.EqualFold(strings.TrimSpace(mod.ModType), snakeBiteModType) {
			continue
		}
		if missing := missingSnakeBiteStagedPackageFiles(mod.StagingPath); len(missing) > 0 {
			return sdk.EventHandlerResult{}, errors.New("MGSV SnakeBite package " + mod.Name + " is missing staged package files: " + strings.Join(missing, ", "))
		}
		return sdk.EventHandlerResult{}, installplan.Unsupported(snakeBiteDeployBlockedReason)
	}
	return sdk.EventHandlerResult{Messages: []string{"MGSV SnakeBite packed archive deployment skipped because no SnakeBite package is enabled in this profile."}}, nil
}

func missingSnakeBiteStagedPackageFiles(stagingPath string) []string {
	stagingPath = strings.TrimSpace(stagingPath)
	if stagingPath == "" {
		return []string{"staging path"}
	}
	var missing []string
	for _, rel := range []string{"metadata.xml"} {
		path := filepath.Join(stagingPath, filepath.FromSlash(rel))
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			missing = append(missing, filepath.ToSlash(rel))
		}
	}
	return missing
}

func checkRequiredGameFiles(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil {
		return nil
	}
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return nil
	}
	details := make([]string, 0, len(requiredGameFiles))
	for _, rel := range requiredGameFiles {
		path := filepath.Join(gamePath, filepath.FromSlash(rel))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			details = append(details, filepath.ToSlash(path))
		}
	}
	if len(details) != len(requiredGameFiles) {
		return nil
	}
	return details
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "SnakeBite README and source",
			URL:  "https://github.com/topher-au/SnakeBite",
		},
		{
			Name: "SnakeBite Nexus page",
			URL:  "https://www.nexusmods.com/metalgearsolidvtpp/mods/106",
		},
		{
			Name: "MGSV Modding Wiki SnakeBite page",
			URL:  "https://mgsvmoddingwiki.github.io/SnakeBite_Mod_Manager/",
		},
		{
			Name: "Nexus game domain",
			URL:  "https://www.nexusmods.com/metalgearsolidvtpp",
		},
		{
			Name: "Live Steam Deck executable/path verification",
			URL:  "extensionTargets.md#installed-games-snapshot",
		},
	}
}
