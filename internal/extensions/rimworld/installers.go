package rimworld

import (
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const aboutXMLFile = "about.xml"

var rimWorldGitFiles = map[string]struct{}{
	".gitignore":     {},
	".gitattributes": {},
}

var rimWorldRootFolderFiles = map[string]struct{}{
	"README.MD":       {},
	"LICENSE":         {},
	"CONTRIBUTING.MD": {},
}

type aboutXML struct {
	PackageID string `xml:"packageId"`
	Name      string `xml:"name"`
	Author    string `xml:"author"`
}

func matchSteamMod(root string) bool {
	aboutFiles := rimWorldAboutFiles(root)
	return len(aboutFiles) > 0
}

func buildSteamMod(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	aboutFiles := aboutFilesFromList(files)
	if len(aboutFiles) == 0 {
		return installplan.Plan{}, installplan.Unsupported("RimWorld archive does not contain About.xml")
	}
	if len(aboutFiles) > 1 {
		return installplan.Plan{}, installplan.Unsupported("RimWorld archive contains multiple About.xml files; multi-mod bundles need manual review before DMM can install them")
	}
	aboutFile := aboutFiles[0]
	rootFile := rimWorldRootFile(files, aboutFile)
	rootSegment := firstSegment(rootFile)
	modName, metadata := rimWorldModName(input.ExtractedRoot, aboutFile)
	if modName == "" {
		modName = sanitizeSegment(strings.TrimSuffix(filepath.Base(filepath.Dir(aboutFile)), ".installing"))
	}
	if modName == "" {
		modName = "rimworld-mod"
	}

	instructions := make([]installplan.Instruction, 0, len(files))
	for _, file := range files {
		if !rimWorldDeployableFile(file) {
			continue
		}
		targetRel := file
		segments := splitPath(file)
		if len(segments) > 1 && segments[0] == rootSegment {
			targetRel = filepath.ToSlash(filepath.Join(append([]string{modName}, segments[1:]...)...))
		}
		instructions = append(instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: targetRel,
			TargetRelative:  filepath.ToSlash(filepath.Join(modRoot, targetRel)),
		})
	}
	if len(instructions) == 0 {
		return installplan.Plan{}, errors.New("RimWorld installer matched but produced no deployable files")
	}
	sort.SliceStable(instructions, func(i, j int) bool {
		return instructions[i].TargetRelative < instructions[j].TargetRelative
	})
	return installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-custom-installer",
			Path:   aboutFile,
			Reason: "Vortex installer rimworld-steam-mod matched a RimWorld About.xml archive",
		}},
		Metadata:     metadata,
		Instructions: instructions,
	}, nil
}

func rimWorldAboutFiles(root string) []string {
	files, err := listFiles(root)
	if err != nil {
		return nil
	}
	return aboutFilesFromList(files)
}

func aboutFilesFromList(files []string) []string {
	var out []string
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), aboutXMLFile) {
			out = append(out, file)
		}
	}
	sort.Strings(out)
	return out
}

func rimWorldRootFile(files []string, aboutFile string) string {
	for _, file := range files {
		if _, ok := rimWorldRootFolderFiles[filepath.Base(file)]; ok {
			return file
		}
	}
	return aboutFile
}

func rimWorldModName(root, aboutFile string) (string, []installplan.ModMetadata) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(aboutFile)))
	if err != nil {
		return "", nil
	}
	var parsed aboutXML
	if err := xml.Unmarshal(data, &parsed); err != nil {
		return "", nil
	}
	packageID := strings.TrimSpace(parsed.PackageID)
	name := strings.TrimSpace(parsed.Name)
	metadata := []installplan.ModMetadata{{
		Kind:     "rimworld-about",
		Name:     name,
		UniqueID: packageID,
	}}
	return strings.ReplaceAll(sanitizeSegment(packageID), ".", "_"), metadata
}

func rimWorldDeployableFile(file string) bool {
	base := filepath.Base(file)
	if _, ok := rimWorldGitFiles[base]; ok {
		return false
	}
	return filepath.Ext(base) != ""
}

func firstSegment(file string) string {
	segments := splitPath(file)
	if len(segments) == 0 {
		return ""
	}
	return segments[0]
}

func splitPath(file string) []string {
	file = filepath.ToSlash(strings.TrimSpace(file))
	if file == "" {
		return nil
	}
	return strings.Split(file, "/")
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

func sanitizeSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "." || value == ".." {
		return ""
	}
	value = strings.ReplaceAll(value, "\\", "_")
	value = strings.ReplaceAll(value, "/", "_")
	return value
}
