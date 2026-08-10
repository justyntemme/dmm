package sharedmodtypes

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	ID      = "vortex-shared-modtypes"
	Name    = "Vortex Shared Mod Types"
	Version = "0.1.0"
	BuildID = "first-party-go"
)

const blockedMessage = "Vortex source defines this shared mod-type behavior, but DMM has not implemented the reusable runtime/helper for game extensions yet."

const (
	DInputModType = "dinput"
	ENBModType    = "enb"
	GeDoSaToType  = "gedosato"
)

const (
	GeDoSaToPathEnv       = "DMM_GEDOSATO_PATH"
	GeDoSaToLegacyPathEnv = "GEDOSATO_PATH"

	dinputTrustGroupID  = "dinput-trust"
	dinputTrustChoiceID = "dinput-trust-continue"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:      ID,
		Name:    Name,
		Kind:    sdk.ExtensionKindFramework,
		Version: Version,
		BuildID: BuildID,
		Register: func(r sdk.Registrar) {
			Register(r)
		},
	}
}

func Register(r sdk.Registrar) {
	for _, ref := range Sources() {
		r.RegisterSource(ref)
	}
	for _, modType := range []installplan.ModTypeSpec{
		DInputModTypeSpec(),
		{ID: ENBModType, TargetRoot: "", Status: sdk.CapabilityStatusMetadata, Message: "Vortex registers the ENB mod type for game-root deployment, but the upstream automatic ENB installer is commented out. DMM keeps this as source metadata until a game extension declares a concrete ENB runtime path."},
		{ID: GeDoSaToType, TargetRoot: "", Status: sdk.CapabilityStatusMetadata, Message: "GeDoSaTo helper support is implemented for game extensions that declare a concrete texture target root and runtime requirement."},
	} {
		if modType.ID == DInputModType {
			r.RegisterModType(modType)
			continue
		}
		if modType.Status == "" {
			modType.Status = sdk.CapabilityStatusBlocked
		}
		if modType.Message == "" {
			modType.Message = blockedMessage
		}
		r.RegisterModType(modType)
	}
	r.RegisterInstaller(DInputInstaller("dinput", 50))
	for _, installer := range []installplan.InstallerSpec{
		{ID: "gedosato", VortexInstallerID: "gedosato", ModType: GeDoSaToType, UnsupportedReason: "GeDoSaTo installer planning is implemented as an opt-in helper; a game extension must declare the target root before DMM can run it."},
	} {
		installer.InstructionMode = installplan.InstructionUnsupported
		installer.Status = sdk.CapabilityStatusBlocked
		installer.Message = installer.UnsupportedReason
		r.RegisterInstaller(installer)
	}
}

func DInputModTypeSpec() installplan.ModTypeSpec {
	return installplan.ModTypeSpec{
		ID:         DInputModType,
		TargetRoot: "",
		Message:    "Vortex DInput injector support deploys files beside the selected game executable and requires user trust for injected DLLs.",
	}
}

func DInputInstaller(id string, priority int) installplan.InstallerSpec {
	return installplan.InstallerSpec{
		ID:                strings.TrimSpace(id),
		VortexInstallerID: "dinput",
		Priority:          priority,
		ModType:           DInputModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       MatchDInputArchive,
		CustomBuild:       BuildDInputArchive,
		InstructionMode:   installplan.InstructionCustom,
	}
}

func MatchDInputArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil {
		return false
	}
	return dinputReferenceFile(files) != ""
}

func BuildDInputArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	ref := dinputReferenceFile(files)
	if ref == "" {
		return installplan.Plan{}, installplan.Unsupported("Vortex dinput installer matched but no dinput8.dll was found")
	}
	if !dinputTrustConfirmed(input.Selections) {
		return installplan.Plan{}, dinputTrustChoiceRequired(ref)
	}
	basePath := filepath.ToSlash(filepath.Dir(ref))
	if basePath == "." {
		basePath = ""
	}
	executableDir := filepath.ToSlash(filepath.Dir(strings.TrimSpace(input.ExecutableRelative)))
	if executableDir == "." {
		executableDir = ""
	}
	instructions := make([]installplan.Instruction, 0, len(files))
	for _, file := range files {
		if !simplearchive.PathWithinRoot(file, basePath) {
			continue
		}
		rel := simplearchive.StripRoot(file, basePath)
		if strings.TrimSpace(rel) == "" || rel == "." {
			continue
		}
		targetRel := filepath.ToSlash(filepath.Join(executableDir, rel))
		instructions = append(instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: targetRel,
			TargetRelative:  targetRel,
		})
	}
	if len(instructions) == 0 {
		return installplan.Plan{}, errors.New("DInput installer matched but produced no deployable files")
	}
	sort.SliceStable(instructions, func(i, j int) bool {
		return instructions[i].TargetRelative < instructions[j].TargetRelative
	})
	return installplan.Plan{
		GameID:     input.GameID,
		ModType:    DInputModType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-dinput-installer",
			Path:   ref,
			Reason: "Vortex dinput installer matched dinput8.dll and routes the archive beside the game executable",
		}},
		Warnings:     []string{"DInput mods run injected DLL code inside the game process. Install only from trusted sources."},
		Instructions: instructions,
	}, nil
}

func dinputTrustConfirmed(selections map[string][]string) bool {
	for _, choice := range selections[dinputTrustGroupID] {
		if choice == dinputTrustChoiceID {
			return true
		}
	}
	return false
}

func dinputTrustChoiceRequired(ref string) error {
	return installplan.ChoiceRequired(
		"unsafe-dll-confirmation",
		"DInput injector mods run DLL code inside the game process. Confirm this mod came from a source you trust before DMM installs it.",
		installplan.ChoiceInstaller{
			Name: "Confirm DInput Injector",
			Steps: []installplan.ChoiceStep{{
				ID:   "dinput-confirmation",
				Name: "Trust this injector",
				Groups: []installplan.ChoiceGroup{{
					ID:          dinputTrustGroupID,
					Name:        "Injected DLL warning",
					Type:        "SelectAtLeastOne",
					Description: "This archive contains " + filepath.Base(ref) + ". It will run with the same access as the game process.",
					Required:    true,
					Plugins: []installplan.ChoiceOption{{
						ID:            dinputTrustChoiceID,
						Name:          "I trust this mod and want DMM to install it",
						Description:   "Only continue for mods from authors and pages you trust.",
						Type:          "Optional",
						EffectiveType: "Optional",
					}},
				}},
			}},
		},
		nil,
	)
}

func dinputReferenceFile(files []string) string {
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), "dinput8.dll") {
			return file
		}
	}
	return ""
}

func GeDoSaToTargetRoot(id, name, texturePath string) sdk.TargetRootSpec {
	texturePath = cleanTexturePath(texturePath)
	if strings.TrimSpace(name) == "" {
		name = "GeDoSaTo textures"
	}
	return sdk.TargetRootSpec{
		ID:   strings.TrimSpace(id),
		Name: name,
		Resolver: func(ctx context.Context, input sdk.TargetRootInput) (sdk.TargetRootResult, error) {
			if err := ctx.Err(); err != nil {
				return sdk.TargetRootResult{}, err
			}
			base, source, err := GeDoSaToInstallPath()
			if err != nil {
				return sdk.TargetRootResult{}, err
			}
			if texturePath == "" {
				return sdk.TargetRootResult{}, errors.New("GeDoSaTo texture path is required")
			}
			return sdk.TargetRootResult{
				Path:   filepath.Join(base, "textures", filepath.FromSlash(texturePath)),
				Source: source + " textures/" + texturePath,
			}, nil
		},
	}
}

func GeDoSaToModTypeSpec(targetRootID string) installplan.ModTypeSpec {
	return installplan.ModTypeSpec{
		ID:           GeDoSaToType,
		TargetRootID: strings.TrimSpace(targetRootID),
		Message:      "Vortex GeDoSaTo support deploys texture files into the detected GeDoSaTo textures folder for the game.",
	}
}

func GeDoSaToInstaller(id string, priority int, targetRootID string) installplan.InstallerSpec {
	return installplan.InstallerSpec{
		ID:                strings.TrimSpace(id),
		VortexInstallerID: "gedosato",
		Priority:          priority,
		ModType:           GeDoSaToType,
		NameSource:        installplan.NameSourceArchive,
		TargetRootID:      strings.TrimSpace(targetRootID),
		CustomMatch:       MatchGeDoSaToArchive,
		CustomBuild:       BuildGeDoSaToArchive,
		InstructionMode:   installplan.InstructionCustom,
	}
}

func GeDoSaToRuntimeRequirement(id string, modTypes []string) gamehandler.RuntimeRequirementSpec {
	if strings.TrimSpace(id) == "" {
		id = "gedosato-installed"
	}
	return gamehandler.RuntimeRequirementSpec{
		ID:          id,
		Name:        "GeDoSaTo",
		Kind:        "external-tool",
		Required:    true,
		ModTypes:    modTypes,
		Message:     "GeDoSaTo is required before enabled GeDoSaTo texture mods can deploy. Set DMM_GEDOSATO_PATH to the GeDoSaTo install folder.",
		OKMessage:   "GeDoSaTo install path is configured.",
		HelpURL:     "https://community.pcgamingwiki.com/files/file/897-gedosato/",
		InstallHint: "Install GeDoSaTo, then set DMM_GEDOSATO_PATH to its install directory before deploying these texture mods.",
		Check: func(ctx context.Context, gamePath string) []string {
			if err := ctx.Err(); err != nil {
				return nil
			}
			path, source, err := GeDoSaToInstallPath()
			if err != nil {
				return nil
			}
			return []string{source + ": " + path}
		},
	}
}

func MatchGeDoSaToArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil {
		return false
	}
	textures := textureFiles(files)
	return len(textures) > 0 && len(textures) == len(files)
}

func BuildGeDoSaToArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	textures := textureFiles(files)
	if len(textures) == 0 || len(textures) != len(files) {
		return installplan.Plan{}, installplan.Unsupported("Vortex GeDoSaTo installer matched only all-texture archives")
	}
	basePath := filepath.ToSlash(filepath.Dir(textures[0]))
	if basePath == "." {
		basePath = ""
	}
	instructions := make([]installplan.Instruction, 0, len(textures))
	for _, file := range textures {
		if !simplearchive.PathWithinRoot(file, basePath) {
			continue
		}
		rel := simplearchive.StripRoot(file, basePath)
		if strings.TrimSpace(rel) == "" || rel == "." {
			continue
		}
		instructions = append(instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: filepath.ToSlash(rel),
			TargetRoot:      strings.TrimSpace(input.TargetRootID),
			TargetRelative:  filepath.ToSlash(rel),
		})
	}
	if len(instructions) == 0 {
		return installplan.Plan{}, errors.New("GeDoSaTo installer matched but produced no deployable files")
	}
	sort.SliceStable(instructions, func(i, j int) bool {
		return instructions[i].TargetRelative < instructions[j].TargetRelative
	})
	return installplan.Plan{
		GameID:     input.GameID,
		ModType:    GeDoSaToType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-gedosato-installer",
			Path:   textures[0],
			Reason: "Vortex GeDoSaTo installer matched an all-texture archive and routes files into the game texture folder",
		}},
		Warnings:     []string{"GeDoSaTo texture mods require GeDoSaTo to be installed and configured before deployment."},
		Instructions: instructions,
	}, nil
}

func GeDoSaToInstallPath() (string, string, error) {
	for _, key := range []string{GeDoSaToPathEnv, GeDoSaToLegacyPathEnv} {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			continue
		}
		if !filepath.IsAbs(value) {
			return "", "", errors.New(key + " must be an absolute path")
		}
		info, err := os.Stat(value)
		if err != nil {
			return "", "", err
		}
		if !info.IsDir() {
			return "", "", errors.New(key + " does not point to a directory")
		}
		return filepath.Clean(value), key, nil
	}
	return "", "", errors.New("GeDoSaTo install path is not configured")
}

func textureFiles(files []string) []string {
	out := make([]string, 0, len(files))
	for _, file := range files {
		if strings.HasSuffix(file, "/") {
			continue
		}
		ext := strings.ToLower(filepath.Ext(file))
		if ext != ".dds" && ext != ".png" {
			continue
		}
		out = append(out, file)
	}
	return out
}

func cleanTexturePath(value string) string {
	value = filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(value))))
	if value == "." || value == ".." || strings.HasPrefix(value, "../") || strings.HasPrefix(value, "/") {
		return ""
	}
	return value
}

func Sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex modtype-dinput source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/modtype-dinput/src/index.ts",
		},
		{
			Name: "Vortex modtype-enb source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/modtype-enb/src/index.ts",
		},
		{
			Name: "Vortex modtype-gedosato source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/modtype-gedosato/src/index.ts",
		},
	}
}
