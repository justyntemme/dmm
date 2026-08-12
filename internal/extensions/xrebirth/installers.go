package xrebirth

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
	ID          string `xml:"id,attr"`
	Name        string `xml:"name,attr"`
	Description string `xml:"description,attr"`
	Author      string `xml:"author,attr"`
	Version     string `xml:"version,attr"`
	Save        string `xml:"save,attr"`
}

func installers() []installplan.InstallerSpec {
	return []installplan.InstallerSpec{
		{
			ID:                "vortex:xrebirth:content-xml",
			VortexInstallerID: "xrebirth",
			Priority:          50,
			ModType:           modTypeContent,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchContentArchive,
			CustomBuild:       buildContentArchive,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:xrebirth:savegame",
			VortexInstallerID: "savegame",
			Priority:          60,
			ModType:           modTypeSavegame,
			NameSource:        installplan.NameSourceArchive,
			Match: installplan.MatchSpec{
				RegexPatterns: []string{`(?i)(^|/)(quicksave|save_\d+)\.xml$`},
				RegexMode:     installplan.MatchModeAny,
			},
			InstructionMode: installplan.InstructionArchiveRoot,
		},
		{
			ID:                "vortex:xrebirth:shader-injector",
			VortexInstallerID: "shader-injector",
			Priority:          65,
			ModType:           modTypeShaderInjector,
			NameSource:        installplan.NameSourceArchive,
			Match: installplan.MatchSpec{
				RegexPatterns: []string{
					`(?i)(^|/)d3d9\.dll$`,
					`(?i)(^|/)dxgi\.dll$`,
					`(?i)(^|/)d3d9\.ini$`,
					`(?i)(^|/)SweetFX([\\/]|_)`,
					`(?i)(^|/)reshade-shaders/`,
					`(?i)(^|/)ReShade/`,
				},
				RegexMode: installplan.MatchModeAny,
			},
			StripCommonRoot: true,
			InstructionMode: installplan.InstructionArchiveRoot,
		},
		{
			ID:                "vortex:xrebirth:utility",
			VortexInstallerID: "utility",
			Priority:          70,
			ModType:           modTypeUtility,
			NameSource:        installplan.NameSourceArchive,
			Match: installplan.MatchSpec{
				FileExtensions:    []string{".exe"},
				FileExtensionMode: installplan.MatchModeAny,
			},
			StripCommonRoot: true,
			InstructionMode: installplan.InstructionArchiveRoot,
		},
		{
			ID:                "vortex:xrebirth:dropin",
			VortexInstallerID: "dropin",
			Priority:          75,
			ModType:           modTypeDropIn,
			NameSource:        installplan.NameSourceArchive,
			Match:             installplan.MatchSpec{UseGameStopPatterns: true},
			StripCommonRoot:   true,
			InstructionMode:   installplan.InstructionArchiveRoot,
		},
		{
			ID:                "vortex:xrebirth:save-patch",
			VortexInstallerID: "save-patch",
			Priority:          80,
			ModType:           modTypeSavePatch,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchSavePatchArchive,
			InstructionMode:   installplan.InstructionArchiveRoot,
		},
		{
			ID:                "vortex:xrebirth:documentation",
			VortexInstallerID: "documentation",
			Priority:          90,
			ModType:           modTypeDocumentation,
			NameSource:        installplan.NameSourceArchive,
			Match: installplan.MatchSpec{
				FileExtensions:    documentationExtensions(),
				FileExtensionMode: installplan.MatchModeAll,
			},
			StripCommonRoot: true,
			InstructionMode: installplan.InstructionArchiveRoot,
		},
	}
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
		return installplan.Plan{}, installplan.Unsupported("X Rebirth archive does not contain content.xml")
	}
	content, err := readContentXML(filepath.Join(input.ExtractedRoot, filepath.FromSlash(contentPath)))
	if err != nil {
		return installplan.Plan{}, err
	}
	if strings.TrimSpace(content.ID) == "" {
		return installplan.Plan{}, installplan.Unsupported("X Rebirth content.xml is missing the required id attribute")
	}
	basePath := filepath.ToSlash(filepath.Dir(contentPath))
	if basePath == "." {
		basePath = ""
	}
	outputPath := firstNonEmpty(sanitizeSegment(content.ID), sanitizeSegment(content.Name), "mod")
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
		return installplan.Plan{}, errors.New("X Rebirth installer matched but produced no deployable files")
	}
	sort.SliceStable(instructions, func(i, j int) bool {
		return instructions[i].TargetRelative < instructions[j].TargetRelative
	})
	return installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: input.Installer.NameSource,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-custom-installer",
			Path:   contentPath,
			Reason: "Vortex installer xrebirth matched an X Rebirth content.xml archive",
		}},
		Metadata: []installplan.ModMetadata{{
			Kind:     "xrebirth-content",
			Name:     firstNonEmpty(content.Name, content.ID),
			UniqueID: strings.TrimSpace(content.ID),
			Version:  strings.TrimSpace(content.Version),
		}},
		Instructions: instructions,
	}, nil
}

func matchSavePatchArchive(root string) bool {
	files, err := listFiles(root)
	if err != nil || len(files) == 0 {
		return false
	}
	hasXML := false
	for _, file := range files {
		extension := strings.ToLower(filepath.Ext(file))
		if extension != ".xml" && extension != ".txt" {
			return false
		}
		if extension == ".xml" {
			hasXML = true
		}
	}
	return hasXML
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

func stopPatterns() []string {
	return []string{
		`[^/]*\.cat$`,
		`[^/]*\.dat$`,
		`(^|/)t/[^/]+\.xml$`,
		`(^|/)lang\.dat$`,
		`(^|/)assets/.+`,
		`(^|/)libraries/.+\.xml$`,
		`(^|/)maps/.+\.xml$`,
		`(^|/)md/.+\.xml$`,
		`(^|/)cinematics/.+`,
		`(^|/)aiscripts/.+\.xml$`,
		`(^|/)voice-[^/]+/.+\.(ogg|wav)$`,
		`(^|/)ui/.+`,
		`(^|/)sfx/.+`,
		`[^/]*\.cur$`,
		`[^/]*\.(ogg|mp3|wav)$`,
		`[^/]*\.(mkv|mp4|webm)$`,
		`[^/]*\.ini$`,
	}
}

func documentationExtensions() []string {
	return []string{".pdf", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".xlsx", ".xls", ".docx", ".doc", ".odt", ".ods", ".md", ".rtf"}
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
		case r == '-' || r == '_' || r == '.':
			out.WriteRune(r)
		}
	}
	return out.String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if clean := strings.TrimSpace(value); clean != "" {
			return clean
		}
	}
	return ""
}
