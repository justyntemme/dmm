package sekiro

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const partsDCXExtension = ".partsbnd.dcx"

func matchLoosePartsArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	return err == nil && hasLooseParts(files)
}

func buildLoosePartsArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	dcxFiles := partsDCXFiles(files)
	if len(dcxFiles) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Sekiro archive does not contain loose .partsbnd.dcx files")
	}
	plan := installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: input.Installer.NameSource,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-sekiro-loose-parts",
			Path:   dcxFiles[0],
			Reason: "Vortex Sekiro loose-parts installer copies loose .partsbnd.dcx files into mods/parts",
		}},
	}
	for _, file := range dcxFiles {
		targetRel := filepath.ToSlash(filepath.Join(input.TargetRoot, "parts", filepath.Base(file)))
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: filepath.ToSlash(filepath.Join("parts", filepath.Base(file))),
			TargetRoot:      input.TargetRootID,
			TargetRelative:  targetRel,
		})
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func matchRootArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	return err == nil && hasLooseParts(files) && hasOtherRootFolders(files)
}

func buildRootArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	dcx := firstPartsDCX(files)
	idx := partsSegmentIndex(dcx)
	if dcx == "" || idx < 0 {
		return installplan.Plan{}, installplan.Unsupported("Sekiro archive does not contain a parts folder root")
	}
	plan := installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: input.Installer.NameSource,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-sekiro-root-mod",
			Path:   dcx,
			Reason: "Vortex Sekiro root installer copies file payloads from the parts root onward into mods",
		}},
	}
	for _, file := range files {
		if filepath.Ext(file) == "" {
			continue
		}
		segments := strings.Split(filepath.ToSlash(strings.Trim(file, "/")), "/")
		if len(segments) <= idx {
			continue
		}
		destination := filepath.ToSlash(filepath.Join(segments[idx:]...))
		if strings.TrimSpace(destination) == "" {
			continue
		}
		targetRel := filepath.ToSlash(filepath.Join(input.TargetRoot, destination))
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: destination,
			TargetRoot:      input.TargetRootID,
			TargetRelative:  targetRel,
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New("Sekiro root installer matched but produced no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func hasLooseParts(files []string) bool {
	dcxFiles := partsDCXFiles(files)
	return len(dcxFiles) > 0 && !strings.Contains(strings.ToLower(dcxFiles[0]), "/parts/")
}

func hasOtherRootFolders(files []string) bool {
	idx := partsSegmentIndex(firstPartsDCX(files))
	if idx < 0 {
		return false
	}
	for _, file := range files {
		segments := strings.Split(strings.ToLower(filepath.ToSlash(strings.Trim(file, "/"))), "/")
		if len(segments) > idx+1 && segments[idx] != "parts" {
			return true
		}
	}
	return false
}

func partsDCXFiles(files []string) []string {
	out := []string{}
	for _, file := range files {
		if strings.HasSuffix(strings.ToLower(file), partsDCXExtension) {
			out = append(out, file)
		}
	}
	return out
}

func firstPartsDCX(files []string) string {
	dcxFiles := partsDCXFiles(files)
	if len(dcxFiles) == 0 {
		return ""
	}
	return dcxFiles[0]
}

func partsSegmentIndex(file string) int {
	segments := strings.Split(strings.ToLower(filepath.ToSlash(strings.Trim(file, "/"))), "/")
	for idx, segment := range segments {
		if segment == "parts" {
			return idx
		}
	}
	return -1
}
