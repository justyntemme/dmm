package neverwinter

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

var nwnDestinations = map[string]string{
	".mod": "modules",
	".tga": "portraits",
	".erf": "erf",
	".hak": "hak",
	".exe": "hak",
	".hif": "hak",
	".tlk": "tlk",
	".bmu": "music",
	".wav": "ambient",
	".cdx": "database",
	".dbf": "database",
	".fpt": "database",
	".nbm": "movies",
	".bik": "movies",
	".2da": "override",
	".uti": "override",
	".txi": "override",
	".mdl": "override",
	".ncs": "override",
	".dlg": "override",
	".utp": "override",
}

var nwnStructuredDirs = map[string]struct{}{
	"ambient":     {},
	"database":    {},
	"development": {},
	"dmvault":     {},
	"hak":         {},
	"localvault":  {},
	"logs":        {},
	"modules":     {},
	"movies":      {},
	"music":       {},
	"nwsync":      {},
	"override":    {},
	"portraits":   {},
	"servervault": {},
	"tempclient":  {},
	"tlk":         {},
}

func matchNWNStructuredArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	return err == nil && firstStructuredFile(files) != ""
}

func buildNWNStructuredArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	marker := firstStructuredFile(files)
	if marker == "" {
		return installplan.Plan{}, installplan.Unsupported("Neverwinter Nights archive does not contain a recognized folder structure")
	}
	plan := basePlan(input, "vortex-nwn-structured", marker, "Vortex skips its loose-file installer when an archive already has Neverwinter folder structure; DMM preserves that structure explicitly")
	for _, file := range files {
		if filepath.Ext(file) == "" {
			continue
		}
		destination, ok := structuredDestination(file)
		if !ok {
			continue
		}
		if err := addInstruction(&plan, input, destination, file); err != nil {
			return installplan.Plan{}, err
		}
	}
	return finishPlan(plan, "Neverwinter Nights structured installer matched but produced no deployable files")
}

func matchNWNLooseArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || firstStructuredFile(files) != "" {
		return false
	}
	return firstNWNLooseFile(files) != ""
}

func buildNWNLooseArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	marker := firstNWNLooseFile(files)
	if marker == "" {
		return installplan.Plan{}, installplan.Unsupported("Neverwinter Nights archive does not contain a supported loose file extension")
	}
	plan := basePlan(input, "vortex-nwn-loose", marker, "Vortex Neverwinter Nights installer maps loose file extensions into their expected mod folders")
	for _, file := range files {
		destination, ok := nwnLooseDestination(file)
		if !ok {
			continue
		}
		if err := addInstruction(&plan, input, destination, file); err != nil {
			return installplan.Plan{}, err
		}
	}
	return finishPlan(plan, "Neverwinter Nights loose installer matched but produced no deployable files")
}

func matchNWN2ModuleArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil {
		return false
	}
	hasModule := false
	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file))
		if ext == ".mod" {
			hasModule = true
			continue
		}
		if ext != "" && ext != ".txt" && ext != ".doc" {
			return false
		}
	}
	return hasModule
}

func buildNWN2ModuleArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	plan := basePlan(input, "vortex-nwn2-module", firstNWN2Module(files), "Vortex Neverwinter Nights 2 module installer copies .mod files into the Documents modules folder")
	for _, file := range files {
		if !strings.EqualFold(filepath.Ext(file), ".mod") {
			continue
		}
		if err := addInstruction(&plan, input, filepath.ToSlash(filepath.Join("modules", filepath.Base(file))), file); err != nil {
			return installplan.Plan{}, err
		}
	}
	return finishPlan(plan, "Neverwinter Nights 2 module installer matched but produced no deployable files")
}

func basePlan(input installplan.BuildInput, kind, marker, reason string) installplan.Plan {
	return installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   kind,
			Path:   filepath.ToSlash(marker),
			Reason: reason,
		}},
	}
}

func addInstruction(plan *installplan.Plan, input installplan.BuildInput, destination, source string) error {
	destination = filepath.ToSlash(strings.Trim(destination, "/"))
	source = filepath.ToSlash(strings.Trim(source, "/"))
	if destination == "" || source == "" {
		return errors.New("Neverwinter installer produced an empty path")
	}
	targetRel := filepath.ToSlash(filepath.Join(input.TargetRoot, destination))
	plan.Instructions = append(plan.Instructions, installplan.Instruction{
		Kind:            installplan.InstructionKindCopy,
		SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(source)),
		StagingRelative: targetRel,
		TargetRoot:      input.TargetRootID,
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

func firstStructuredFile(files []string) string {
	for _, file := range files {
		if _, ok := structuredDestination(file); ok {
			return file
		}
	}
	return ""
}

func structuredDestination(file string) (string, bool) {
	segments := pathSegments(file)
	for idx, segment := range segments {
		segment = strings.ToLower(segment)
		if segment == "override" && idx > 0 {
			continue
		}
		if _, ok := nwnStructuredDirs[segment]; ok && idx < len(segments)-1 {
			return filepath.ToSlash(filepath.Join(segments[idx:]...)), true
		}
	}
	return "", false
}

func firstNWNLooseFile(files []string) string {
	for _, file := range files {
		if _, ok := nwnLooseDestination(file); ok {
			return file
		}
	}
	return ""
}

func nwnLooseDestination(file string) (string, bool) {
	ext := strings.ToLower(filepath.Ext(file))
	destination, ok := nwnDestinations[ext]
	if !ok {
		return "", false
	}
	if containsPathSegment(file, "override") {
		return filepath.ToSlash(file), true
	}
	return filepath.ToSlash(filepath.Join(destination, filepath.Base(file))), true
}

func firstNWN2Module(files []string) string {
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), ".mod") {
			return file
		}
	}
	return ""
}

func containsPathSegment(file, segment string) bool {
	for _, item := range pathSegments(file) {
		if strings.EqualFold(item, segment) {
			return true
		}
	}
	return false
}

func pathSegments(file string) []string {
	file = filepath.ToSlash(strings.Trim(file, "/"))
	if file == "" {
		return nil
	}
	return strings.Split(file, "/")
}
