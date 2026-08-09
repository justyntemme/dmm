package mountandblade

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

var overrideDestinations = map[string]string{
	".dds": "textures",
	".brf": "resource",
	".sco": "sceneobj",
	".txt": "",
}

func matchMountAndBladeArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) {
		return false
	}
	return hasModuleINI(files) || hasOverridePayload(files)
}

func buildMountAndBladeArchive(input installplan.BuildInput, spec gameSpec) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	if marker := firstModuleINI(files); marker != "" {
		return buildModuleArchive(input, files, marker)
	}
	if hasOverridePayload(files) {
		return buildOverrideArchive(input, files, spec)
	}
	return installplan.Plan{}, installplan.Unsupported("Mount & Blade archive does not contain module.ini or supported loose override files")
}

func buildModuleArchive(input installplan.BuildInput, files []string, marker string) (installplan.Plan, error) {
	moduleName := archiveModuleName(input.ArchiveName)
	root := filepath.ToSlash(filepath.Dir(marker))
	if root == "." {
		root = ""
	}
	plan := basePlan(input, "vortex-mount-and-blade-module", marker, "Vortex Mount & Blade installer found module.ini and copies the module root under the archive-name module folder")
	for _, file := range files {
		if filepath.Ext(file) == "" || !simplearchive.PathWithinRoot(file, root) {
			continue
		}
		rel := simplearchive.StripRoot(file, root)
		if rel == "" {
			continue
		}
		if err := addInstruction(&plan, input, filepath.ToSlash(filepath.Join(moduleName, rel)), file); err != nil {
			return installplan.Plan{}, err
		}
	}
	return finishPlan(plan, "Mount & Blade module installer matched but produced no deployable files")
}

func buildOverrideArchive(input installplan.BuildInput, files []string, spec gameSpec) (installplan.Plan, error) {
	plan := basePlan(input, "vortex-mount-and-blade-override", firstOverridePayload(files), "Vortex Mount & Blade installer maps loose supported file extensions into the native module folder")
	for _, file := range files {
		folder, ok := overrideDestinations[strings.ToLower(filepath.Ext(file))]
		if !ok {
			continue
		}
		rel := filepath.ToSlash(filepath.Join(spec.NativeModuleName, folder, filepath.Base(file)))
		if err := addInstruction(&plan, input, rel, file); err != nil {
			return installplan.Plan{}, err
		}
	}
	return finishPlan(plan, "Mount & Blade override installer matched but produced no deployable files")
}

func basePlan(input installplan.BuildInput, kind, marker, reason string) installplan.Plan {
	return installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   kind,
			Path:   marker,
			Reason: reason,
		}},
	}
}

func addInstruction(plan *installplan.Plan, input installplan.BuildInput, destination, source string) error {
	destination = filepath.ToSlash(strings.Trim(destination, "/"))
	source = filepath.ToSlash(strings.Trim(source, "/"))
	if destination == "" || source == "" {
		return errors.New("Mount & Blade installer produced an empty path")
	}
	targetRel := filepath.ToSlash(filepath.Join(input.TargetRoot, destination))
	plan.Instructions = append(plan.Instructions, installplan.Instruction{
		Kind:            installplan.InstructionKindCopy,
		SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(source)),
		StagingRelative: targetRel,
		TargetRelative:  targetRel,
	})
	return nil
}

func finishPlan(plan installplan.Plan, emptyReason string) (installplan.Plan, error) {
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New(emptyReason)
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func hasModuleINI(files []string) bool {
	return firstModuleINI(files) != ""
}

func firstModuleINI(files []string) string {
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), moduleFile) {
			return file
		}
	}
	return ""
}

func hasOverridePayload(files []string) bool {
	return firstOverridePayload(files) != ""
}

func firstOverridePayload(files []string) string {
	for _, file := range files {
		if _, ok := overrideDestinations[strings.ToLower(filepath.Ext(file))]; ok {
			return file
		}
	}
	return ""
}

func archiveModuleName(archiveName string) string {
	name := strings.TrimSpace(filepath.Base(archiveName))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "mod"
	}
	name = strings.TrimSuffix(name, filepath.Ext(name))
	name = strings.TrimSuffix(name, ".tar")
	name = strings.TrimSuffix(name, "-installing")
	name = strings.TrimSuffix(name, " installing")
	name = strings.Trim(strings.TrimSpace(name), ".")
	if name == "" {
		return "mod"
	}
	return sanitizeSegment(name)
}

func sanitizeSegment(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.' || r == ' ':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	out := strings.TrimSpace(b.String())
	out = strings.Trim(out, ".")
	if out == "" {
		return "mod"
	}
	return out
}
