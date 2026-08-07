package starwarsjedisurvivor

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const pakExtension = ".pak"

func matchPakArchive(root string) bool {
	files, err := listFiles(root)
	if err != nil {
		return false
	}
	return len(pakFiles(files, false)) > 0
}

func buildPakArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	paks := pakFiles(files, false)
	if len(paks) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Star Wars Jedi: Survivor archive does not contain a supported .pak file")
	}
	if len(paks) > 1 {
		return installplan.Plan{}, installplan.Unsupported("Star Wars Jedi: Survivor archive contains multiple .pak choices; DMM needs a generic PAK selection UI before it can install this archive safely")
	}
	rootDir := filepath.ToSlash(filepath.Dir(paks[0]))
	if rootDir == "." {
		rootDir = ""
	}
	builder := newPlanBuilder(input, pakModType)
	for _, file := range files {
		if !sameArchiveFolder(file, rootDir) {
			continue
		}
		if err := builder.add(file, filepath.Base(file)); err != nil {
			return installplan.Plan{}, err
		}
	}
	builder.metadata = append(builder.metadata, installplan.ModMetadata{
		Kind:                       "starwarsjedi2-pak-files",
		AdditionalLogicalFileNames: []string{filepath.Base(paks[0])},
	})
	return builder.plan(paks[0], "Vortex installer starwarsjedi2-mod matched a single Star Wars Jedi: Survivor .pak archive")
}

func matchR457Loader(root string) bool {
	files, err := listFiles(root)
	if err != nil {
		return false
	}
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), r457LoaderPak) {
			return true
		}
	}
	return false
}

func buildR457Loader(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	detected := ""
	builder := newPlanBuilder(input, loaderModType)
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), r457LoaderPak) {
			detected = file
		}
		if err := builder.add(file, file); err != nil {
			return installplan.Plan{}, err
		}
	}
	if detected == "" {
		return installplan.Plan{}, installplan.Unsupported("Star Wars Jedi: Survivor R457 loader package is missing " + r457LoaderPak)
	}
	return builder.plan(detected, "Vortex installer starwarsjedi2-r457loader matched the R457 loader package")
}

type planBuilder struct {
	input        installplan.BuildInput
	modType      string
	metadata     []installplan.ModMetadata
	instructions []installplan.Instruction
}

func newPlanBuilder(input installplan.BuildInput, modType string) *planBuilder {
	return &planBuilder{input: input, modType: modType}
}

func (b *planBuilder) add(sourceRel, targetRel string) error {
	sourceRel = filepath.ToSlash(strings.TrimSpace(sourceRel))
	targetRel = filepath.ToSlash(strings.TrimSpace(targetRel))
	if sourceRel == "" || targetRel == "" {
		return errors.New("Star Wars Jedi: Survivor installer produced an empty path")
	}
	b.instructions = append(b.instructions, installplan.Instruction{
		Kind:            installplan.InstructionKindCopy,
		SourcePath:      filepath.Join(b.input.ExtractedRoot, filepath.FromSlash(sourceRel)),
		StagingRelative: targetRel,
		TargetRelative:  filepath.ToSlash(filepath.Join(b.input.TargetRoot, targetRel)),
		DeployStrategy:  installplan.DeployStrategyCopy,
	})
	return nil
}

func (b *planBuilder) plan(detectedPath, reason string) (installplan.Plan, error) {
	if len(b.instructions) == 0 {
		return installplan.Plan{}, errors.New("Star Wars Jedi: Survivor installer matched but produced no deployable files")
	}
	sort.SliceStable(b.instructions, func(i, j int) bool {
		return b.instructions[i].TargetRelative < b.instructions[j].TargetRelative
	})
	return installplan.Plan{
		GameID:       b.input.GameID,
		ModType:      b.modType,
		PlannerID:    b.input.Installer.ID,
		NameSource:   installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{Kind: "vortex-custom-installer", Path: detectedPath, Reason: reason}},
		Metadata:     b.metadata,
		Instructions: b.instructions,
	}, nil
}

func pakFiles(files []string, includeR457 bool) []string {
	var out []string
	for _, file := range files {
		if !strings.EqualFold(filepath.Ext(file), pakExtension) {
			continue
		}
		if !includeR457 && strings.EqualFold(filepath.Base(file), r457LoaderPak) {
			continue
		}
		out = append(out, file)
	}
	sort.Strings(out)
	return out
}

func sameArchiveFolder(file, rootDir string) bool {
	dir := filepath.ToSlash(filepath.Dir(file))
	if dir == "." {
		dir = ""
	}
	return dir == rootDir
}

func listFiles(root string) ([]string, error) {
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
