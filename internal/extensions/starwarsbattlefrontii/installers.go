package starwarsbattlefrontii

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	fbmodExtension     = ".fbmod"
	fbmodChoiceGroupID = "starwarsbattlefront22017-fbmod-choice"
)

func matchFBModArchive(root string) bool {
	files, err := listFiles(root)
	if err != nil {
		return false
	}
	return len(fbmodFiles(files)) > 0
}

func buildFBModArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	fbmods := fbmodFiles(files)
	if len(fbmods) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Star Wars Battlefront II archive does not contain a supported .fbmod file")
	}
	selectedFromChoice := false
	if len(fbmods) > 1 {
		selected, ok := selectedFBModChoice(input.Selections, fbmods)
		if !ok {
			return installplan.Plan{}, fbmodChoiceRequired(fbmods)
		}
		fbmods = []string{selected}
		selectedFromChoice = true
	}
	rootDir := filepath.ToSlash(filepath.Dir(fbmods[0]))
	if rootDir == "." {
		rootDir = ""
	}
	builder := newPlanBuilder(input)
	for _, file := range files {
		if !sameArchiveFolder(file, rootDir) {
			continue
		}
		if selectedFromChoice && !sameLogicalModFile(file, fbmods[0]) {
			continue
		}
		targetRel := filepath.Base(file)
		if err := builder.add(file, targetRel); err != nil {
			return installplan.Plan{}, err
		}
	}
	builder.metadata = append(builder.metadata, installplan.ModMetadata{
		Kind:                       "starwarsbattlefront22017-fbmod-files",
		AdditionalLogicalFileNames: []string{filepath.Base(fbmods[0])},
	})
	return builder.plan(fbmods[0], "Vortex installer starwarsbattlefront22017-mod matched a Battlefront II .fbmod archive")
}

func selectedFBModChoice(selections map[string][]string, fbmods []string) (string, bool) {
	selected := selections[fbmodChoiceGroupID]
	if len(selected) != 1 {
		return "", false
	}
	allowed := map[string]string{}
	for _, fbmod := range fbmods {
		allowed[fbmodChoiceID(fbmod)] = fbmod
	}
	fbmod, ok := allowed[selected[0]]
	return fbmod, ok
}

func fbmodChoiceRequired(fbmods []string) error {
	options := make([]installplan.ChoiceOption, 0, len(fbmods))
	for _, fbmod := range fbmods {
		options = append(options, installplan.ChoiceOption{
			ID:            fbmodChoiceID(fbmod),
			Name:          filepath.Base(fbmod),
			Description:   fbmod,
			Type:          "Optional",
			EffectiveType: "Optional",
		})
	}
	return installplan.ChoiceRequired(
		"archive-file-choice",
		"Star Wars Battlefront II archive contains multiple .fbmod options; choose the variant Vortex would prompt for before DMM installs it.",
		installplan.ChoiceInstaller{
			Name: "Star Wars Battlefront II FBMod Selection",
			Steps: []installplan.ChoiceStep{{
				ID:   "fbmod-selection",
				Name: "Choose FBMod",
				Groups: []installplan.ChoiceGroup{{
					ID:      fbmodChoiceGroupID,
					Name:    "FBMod file",
					Type:    "SelectExactlyOne",
					Plugins: options,
				}},
			}},
		},
		nil,
	)
}

func fbmodChoiceID(fbmod string) string {
	return "fbmod:" + filepath.ToSlash(fbmod)
}

type planBuilder struct {
	input        installplan.BuildInput
	metadata     []installplan.ModMetadata
	instructions []installplan.Instruction
}

func newPlanBuilder(input installplan.BuildInput) *planBuilder {
	return &planBuilder{input: input}
}

func (b *planBuilder) add(sourceRel, targetRel string) error {
	sourceRel = filepath.ToSlash(strings.TrimSpace(sourceRel))
	targetRel = filepath.ToSlash(strings.TrimSpace(targetRel))
	if sourceRel == "" || targetRel == "" {
		return errors.New("Star Wars Battlefront II installer produced an empty path")
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
		return installplan.Plan{}, errors.New("Star Wars Battlefront II installer matched but produced no deployable files")
	}
	sort.SliceStable(b.instructions, func(i, j int) bool {
		return b.instructions[i].TargetRelative < b.instructions[j].TargetRelative
	})
	return installplan.Plan{
		GameID:       b.input.GameID,
		ModType:      b.input.Installer.ModType,
		PlannerID:    b.input.Installer.ID,
		NameSource:   installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{Kind: "vortex-custom-installer", Path: detectedPath, Reason: reason}},
		Metadata:     b.metadata,
		Instructions: b.instructions,
	}, nil
}

func fbmodFiles(files []string) []string {
	out := []string{}
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), fbmodExtension) {
			out = append(out, file)
		}
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

func sameLogicalModFile(file, modFile string) bool {
	fileBase := filepath.Base(file)
	modBase := filepath.Base(modFile)
	fileStem := strings.TrimSuffix(fileBase, filepath.Ext(fileBase))
	modStem := strings.TrimSuffix(modBase, filepath.Ext(modBase))
	return strings.EqualFold(fileStem, modStem)
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
