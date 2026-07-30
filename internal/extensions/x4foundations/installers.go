package x4foundations

import (
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const contentXMLFile = "content.xml"

type contentXML struct {
	ID      string `xml:"id,attr"`
	Name    string `xml:"name,attr"`
	Author  string `xml:"author,attr"`
	Version string `xml:"version,attr"`
	Save    string `xml:"save,attr"`
}

type indexDiffXML struct {
	Adds []struct {
		Entries []struct {
			Value string `xml:"value,attr"`
		} `xml:"entry"`
	} `xml:"add"`
}

func matchContentArchive(root string) bool {
	return findContentXML(root) != ""
}

func buildContentArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	contentPath := contentPathFromFiles(files)
	if contentPath == "" {
		return installplan.Plan{}, installplan.Unsupported("X4 archive does not contain content.xml")
	}
	content, err := readContentXML(filepath.Join(input.ExtractedRoot, filepath.FromSlash(contentPath)))
	if err != nil {
		return installplan.Plan{}, err
	}
	if strings.TrimSpace(content.ID) == "" {
		return installplan.Plan{}, installplan.Unsupported("X4 content.xml is missing the required id attribute")
	}
	basePath := filepath.ToSlash(filepath.Dir(contentPath))
	if basePath == "." {
		basePath = ""
	}
	outputPath := x4OutputPath(input.ExtractedRoot, contentPath, content.ID)
	instructions := make([]installplan.Instruction, 0, len(files))
	for _, file := range files {
		if basePath != "" && !strings.HasPrefix(file, basePath+"/") {
			continue
		}
		rel := file
		if basePath != "" {
			rel = strings.TrimPrefix(file, basePath+"/")
		}
		if strings.TrimSpace(rel) == "" {
			continue
		}
		targetRel := filepath.ToSlash(filepath.Join(outputPath, rel))
		instructions = append(instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: targetRel,
			TargetRoot:      input.TargetRootID,
			TargetRelative:  filepath.ToSlash(filepath.Join(input.TargetRoot, targetRel)),
		})
	}
	if len(instructions) == 0 {
		return installplan.Plan{}, errors.New("X4 installer matched but produced no deployable files")
	}
	sort.SliceStable(instructions, func(i, j int) bool {
		return instructions[i].TargetRelative < instructions[j].TargetRelative
	})
	return installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceManifestDisplay,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-custom-installer",
			Path:   contentPath,
			Reason: "Vortex installer x4foundations matched an X4 content.xml archive",
		}},
		Metadata: []installplan.ModMetadata{{
			Kind:     "x4-content",
			Name:     firstNonEmpty(content.Name, content.ID),
			UniqueID: strings.TrimSpace(content.ID),
			Version:  strings.TrimSpace(content.Version),
		}},
		Instructions: instructions,
	}, nil
}

func findContentXML(root string) string {
	files, err := listFiles(root)
	if err != nil {
		return ""
	}
	return contentPathFromFiles(files)
}

func contentPathFromFiles(files []string) string {
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), contentXMLFile) {
			return file
		}
	}
	return ""
}

func readContentXML(path string) (contentXML, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return contentXML{}, err
	}
	var content contentXML
	if err := xml.Unmarshal(data, &content); err != nil {
		return contentXML{}, err
	}
	return content, nil
}

func x4OutputPath(root, contentPath, contentID string) string {
	parent := filepath.ToSlash(filepath.Dir(contentPath))
	if parent != "." && parent != "" {
		return sanitizeSegment(filepath.Base(parent))
	}
	if fromIndex := x4OutputPathFromIndex(filepath.Join(root, "index")); fromIndex != "" {
		return fromIndex
	}
	return sanitizeSegment(contentID)
}

func x4OutputPathFromIndex(indexPath string) string {
	entries, err := os.ReadDir(indexPath)
	if err != nil {
		return ""
	}
	var xmlFiles []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".xml") {
			continue
		}
		xmlFiles = append(xmlFiles, filepath.Join(indexPath, entry.Name()))
	}
	sort.Strings(xmlFiles)
	for _, file := range xmlFiles {
		name := x4OutputPathFromIndexFile(file)
		if name != "" {
			return name
		}
	}
	return ""
}

func x4OutputPathFromIndexFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var diff indexDiffXML
	if err := xml.Unmarshal(data, &diff); err != nil {
		return ""
	}
	for _, add := range diff.Adds {
		for _, entry := range add.Entries {
			value := filepath.ToSlash(strings.TrimSpace(entry.Value))
			segments := strings.FieldsFunc(value, func(r rune) bool { return r == '/' || r == '\\' })
			if len(segments) >= 2 && strings.EqualFold(segments[0], "extensions") {
				return sanitizeSegment(segments[1])
			}
		}
	}
	return ""
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
	value = strings.TrimSpace(filepath.Base(filepath.ToSlash(value)))
	if value == "." || value == string(filepath.Separator) {
		return ""
	}
	var out strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			out.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			out.WriteRune(r)
		case r >= '0' && r <= '9':
			out.WriteRune(r)
		case r == '-', r == '_', r == '.':
			out.WriteRune(r)
		default:
			out.WriteRune('_')
		}
	}
	return strings.Trim(out.String(), "._-")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
