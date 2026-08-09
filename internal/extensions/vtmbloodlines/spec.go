package vtmbloodlines

import (
	"bufio"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "2600"
	VortexGameID = "vampirebloodlines"
	Name         = "Vampire: The Masquerade - Bloodlines"

	defaultRoot    = "Vampire"
	patchRoot      = "Unofficial_Patch"
	defaultModType = "vtmb-vampire-modtype"
	patchModType   = "vtmb-up-modtype"
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
		ExecutableRelative: "Vampire.exe",
		RequiredFiles:      []string{"Vampire.exe"},
		QueryModPath:       defaultRoot,
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: patchModType, TargetRoot: patchRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: defaultModType, TargetRoot: defaultRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:vampirebloodlines:unofficial-patch",
		VortexInstallerID: patchModType,
		Priority:          25,
		ModType:           patchModType,
		NameSource:        installplan.NameSourceArchive,
		InstructionMode:   installplan.InstructionCustom,
		CustomMatch:       matchUnofficialPatchArchive,
		CustomBuild:       buildUnofficialPatchArchive,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:vampirebloodlines:vampire",
		VortexInstallerID: "vampirebloodlines-default",
		Priority:          50,
		ModType:           defaultModType,
		NameSource:        installplan.NameSourceArchive,
		TargetRoot:        defaultRoot,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "vampirebloodlines-steam-launcher",
		Name:     "Steam launcher",
		Launcher: "steam",
		Store:    "steam",
		AppID:    SteamAppID,
		Status:   sdk.CapabilityStatusMetadata,
		Message:  "Vortex attempts to launch Bloodlines through Steam when the Steam app is discoverable.",
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:             "vampirebloodlines-prepare-folders",
		Name:           "Ensure Bloodlines mod folders are writable",
		GeneratedFiles: []string{defaultRoot, patchRoot},
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "vampirebloodlines-version-inf",
		Name:     "version.inf ExtVersion",
		Provider: gameVersion,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func matchUnofficialPatchArchive(root string) bool {
	patchPath, ok := unofficialPatchContentRoot(root)
	return ok && strings.TrimSpace(patchPath) != ""
}

func buildUnofficialPatchArchive(input installplan.BuildInput) (installplan.Plan, error) {
	patchPath, ok := unofficialPatchContentRoot(input.ExtractedRoot)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Unofficial Patch archive marker was not found")
	}
	return buildCopyPlan(input, patchPath, patchRoot, "vortex-modtype", filepath.ToSlash(mustRel(input.ExtractedRoot, patchPath)), "Vortex Bloodlines Unofficial Patch mod type targets Unofficial_Patch")
}

func unofficialPatchContentRoot(root string) (string, bool) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", false
	}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.EqualFold(entry.Name(), patchRoot) {
			continue
		}
		return filepath.Join(root, entry.Name()), true
	}
	return "", false
}

func buildCopyPlan(input installplan.BuildInput, contentRoot, targetRoot, detectionKind, detectionPath, detectionReason string) (installplan.Plan, error) {
	plan := installplan.Plan{
		GameID:       input.GameID,
		ModType:      input.Installer.ModType,
		PlannerID:    input.Installer.ID,
		NameSource:   installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{Kind: detectionKind, Path: detectionPath, Reason: detectionReason}},
	}
	err := filepath.WalkDir(contentRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(contentRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		targetRel := filepath.ToSlash(filepath.Join(targetRoot, rel))
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      path,
			StagingRelative: targetRel,
			TargetRelative:  targetRel,
		})
		return nil
	})
	if err != nil {
		return installplan.Plan{}, err
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New("Unofficial Patch archive contained no deployable files")
	}
	return plan, nil
}

func gameVersion(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.GameVersionResult{}, err
	}
	version, err := readVersionINF(filepath.Join(input.GamePath, "version.inf"))
	if err != nil {
		return sdk.GameVersionResult{}, err
	}
	return sdk.GameVersionResult{Version: version, Source: "version.inf"}, nil
}

func readVersionINF(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	inVersionInfo := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inVersionInfo = strings.EqualFold(strings.TrimSpace(strings.Trim(line, "[]")), "Version Info")
			continue
		}
		if !inVersionInfo {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(key), "ExtVersion") {
			continue
		}
		version := strings.Trim(strings.TrimSpace(value), `"`)
		if version != "" {
			return version, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", errors.New("version.inf did not contain [Version Info] ExtVersion")
}

func mustRel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	if rel == "." {
		return "."
	}
	return rel
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{{
		Name: "Vortex game-vtmbloodlines extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-vtmbloodlines/src",
	}}
}
