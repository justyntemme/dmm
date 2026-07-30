package gamebryo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
	"github.com/justyntemme/decky-mod-manager/internal/peversion"
)

type ScriptExtenderInstallerOptions struct {
	ID                string
	VortexInstallerID string
	Name              string
	ModType           string
	LoaderExecutable  string
	ToolID            string
}

type ScriptExtenderRuntimeRequirementOptions struct {
	ID            string
	Name          string
	Kind          string
	ModType       string
	Message       string
	OKMessage     string
	HelpURL       string
	InstallHint   string
	RequiredFiles []string
}

func ScriptExtenderInstaller(opts ScriptExtenderInstallerOptions) installplan.InstallerSpec {
	return installplan.InstallerSpec{
		ID:                strings.TrimSpace(opts.ID),
		VortexInstallerID: strings.TrimSpace(opts.VortexInstallerID),
		Priority:          10,
		ModType:           strings.TrimSpace(opts.ModType),
		NameSource:        installplan.NameSourceArchive,
		InstructionMode:   installplan.InstructionCustom,
		CustomMatch: func(extractedRoot string) bool {
			_, ok := findScriptExtenderLoader(extractedRoot, opts.LoaderExecutable)
			return ok
		},
		CustomBuild: func(input installplan.BuildInput) (installplan.Plan, error) {
			return buildScriptExtenderPlan(input, opts)
		},
	}
}

func ScriptExtenderRuntimeRequirement(opts ScriptExtenderRuntimeRequirementOptions) gamehandler.RuntimeRequirementSpec {
	return gamehandler.RuntimeRequirementSpec{
		ID:          strings.TrimSpace(opts.ID),
		Name:        strings.TrimSpace(opts.Name),
		Kind:        firstNonEmpty(strings.TrimSpace(opts.Kind), "script-extender"),
		Required:    true,
		ModTypes:    []string{strings.TrimSpace(opts.ModType)},
		Message:     strings.TrimSpace(opts.Message),
		OKMessage:   strings.TrimSpace(opts.OKMessage),
		HelpURL:     strings.TrimSpace(opts.HelpURL),
		InstallHint: strings.TrimSpace(opts.InstallHint),
		Check: func(ctx context.Context, gamePath string) []string {
			return requiredGameFiles(ctx, gamePath, opts.RequiredFiles)
		},
	}
}

func buildScriptExtenderPlan(input installplan.BuildInput, opts ScriptExtenderInstallerOptions) (installplan.Plan, error) {
	loaderRel, ok := findScriptExtenderLoader(input.ExtractedRoot, opts.LoaderExecutable)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Vortex installer " + input.Installer.VortexInstallerID + " matched but no script extender loader was found")
	}
	rootRel := filepath.ToSlash(filepath.Dir(loaderRel))
	if rootRel == "." {
		rootRel = ""
	}
	files, err := filesUnderScriptExtenderRoot(input.ExtractedRoot, rootRel)
	if err != nil {
		return installplan.Plan{}, err
	}
	if len(files) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Vortex installer " + input.Installer.VortexInstallerID + " matched but produced no deployable files")
	}

	instructions := make([]installplan.Instruction, 0, len(files))
	for _, rel := range files {
		targetRel := rel
		if rootRel != "" {
			var err error
			targetRel, err = filepath.Rel(filepath.FromSlash(rootRel), filepath.FromSlash(rel))
			if err != nil {
				return installplan.Plan{}, err
			}
			targetRel = filepath.ToSlash(targetRel)
		}
		targetRel = filepath.ToSlash(strings.TrimSpace(targetRel))
		if targetRel == "" || targetRel == "." {
			return installplan.Plan{}, errors.New("script extender installer produced an empty path")
		}
		instructions = append(instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(rel)),
			StagingRelative: targetRel,
			TargetRelative:  targetRel,
		})
	}
	sort.SliceStable(instructions, func(i, j int) bool {
		return instructions[i].TargetRelative < instructions[j].TargetRelative
	})

	metadata := []installplan.ModMetadata{{
		Kind:     "script-extender",
		Name:     strings.TrimSpace(opts.Name),
		UniqueID: strings.TrimSpace(opts.ToolID),
	}}
	if version, err := peversion.FileVersion(filepath.Join(input.ExtractedRoot, filepath.FromSlash(loaderRel))); err == nil && strings.TrimSpace(version) != "" {
		metadata[0].Version = strings.TrimSpace(version)
	}

	return installplan.Plan{
		GameID:     input.GameID,
		ModType:    strings.TrimSpace(input.Installer.ModType),
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-script-extender-installer",
			Path:   loaderRel,
			Reason: "Vortex installer " + input.Installer.VortexInstallerID + " matched " + strings.TrimSpace(opts.LoaderExecutable),
		}},
		Metadata:     metadata,
		Instructions: instructions,
	}, nil
}

func findScriptExtenderLoader(root, loaderExecutable string) (string, bool) {
	loaderExecutable = strings.TrimSpace(loaderExecutable)
	if loaderExecutable == "" {
		return "", false
	}
	for _, file := range mustListScriptExtenderFiles(root) {
		if strings.EqualFold(filepath.Base(file), loaderExecutable) {
			return file, true
		}
	}
	return "", false
}

func filesUnderScriptExtenderRoot(root, rootRel string) ([]string, error) {
	files, err := listScriptExtenderFiles(root)
	if err != nil {
		return nil, err
	}
	if rootRel == "" {
		return files, nil
	}
	prefix := strings.TrimSuffix(filepath.ToSlash(rootRel), "/") + "/"
	out := make([]string, 0, len(files))
	for _, file := range files {
		if file == rootRel || strings.HasPrefix(file, prefix) {
			out = append(out, file)
		}
	}
	return out, nil
}

func listScriptExtenderFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func mustListScriptExtenderFiles(root string) []string {
	files, err := listScriptExtenderFiles(root)
	if err != nil {
		return nil
	}
	return files
}

func requiredGameFiles(ctx context.Context, gamePath string, requiredFiles []string) []string {
	if err := ctx.Err(); err != nil {
		return nil
	}
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" || len(requiredFiles) == 0 {
		return nil
	}
	details := make([]string, 0, len(requiredFiles))
	for _, rel := range requiredFiles {
		rel = filepath.ToSlash(strings.TrimSpace(rel))
		if rel == "" || rel == "." || strings.HasPrefix(rel, "../") {
			return nil
		}
		path := filepath.Join(gamePath, filepath.FromSlash(rel))
		if _, err := os.Stat(path); err != nil {
			return nil
		}
		details = append(details, "Found "+rel)
	}
	return details
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
