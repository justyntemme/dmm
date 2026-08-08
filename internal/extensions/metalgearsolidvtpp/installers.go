package metalgearsolidvtpp

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

type snakeBiteMetadataProbe struct {
	XMLName     xml.Name `xml:"ModEntry"`
	Name        string   `xml:"Name,attr"`
	Version     string   `xml:"Version,attr"`
	Author      string   `xml:"Author,attr"`
	Website     string   `xml:"Website,attr"`
	Description string   `xml:"Description"`
	QarEntries  struct {
		Entries []snakeBiteMetadataQAREntry `xml:"QarEntry"`
	} `xml:"QarEntries"`
	FpkEntries struct {
		Entries []snakeBiteMetadataFPKEntry `xml:"FpkEntry"`
	} `xml:"FpkEntries"`
	SBVersion struct {
		Version string `xml:"Version,attr"`
	} `xml:"SBVersion"`
	MGSVersion struct {
		Version string `xml:"Version,attr"`
	} `xml:"MGSVersion"`
}

type snakeBiteMetadataQAREntry struct {
	Hash        uint64 `xml:"Hash,attr"`
	FilePath    string `xml:"FilePath,attr"`
	Compressed  bool   `xml:"Compressed,attr"`
	ContentHash string `xml:"ContentHash,attr"`
}

type snakeBiteMetadataFPKEntry struct {
	FpkFile     string `xml:"FpkFile,attr"`
	FilePath    string `xml:"FilePath,attr"`
	ContentHash string `xml:"ContentHash,attr"`
}

func matchSnakeBitePackage(extractedRoot string) bool {
	extractedRoot = strings.TrimSpace(extractedRoot)
	if extractedRoot == "" {
		return false
	}
	matched := false
	_ = filepath.WalkDir(extractedRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || matched || d.IsDir() {
			return nil
		}
		if strings.EqualFold(d.Name(), "metadata.xml") && isSnakeBiteMetadata(path) {
			matched = true
			return filepath.SkipAll
		}
		return nil
	})
	return matched
}

func buildSnakeBitePackagePlan(input installplan.BuildInput) (installplan.Plan, error) {
	metadataPath := findSnakeBiteMetadata(input.ExtractedRoot)
	if metadataPath == "" {
		return installplan.Plan{}, installplan.Unsupported("SnakeBite metadata.xml was not found")
	}
	metadata, err := readSnakeBiteMetadata(metadataPath)
	if err != nil {
		return installplan.Plan{}, err
	}
	contentRoot := filepath.Dir(metadataPath)
	missing := missingSnakeBiteReferencedFiles(contentRoot, metadata)
	if len(missing) > 0 {
		return installplan.Plan{}, installplan.Unsupported("SnakeBite package references files that are missing from the archive: " + strings.Join(missing, ", "))
	}
	plan := installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceManifestDisplay,
		DetectedFrom: []installplan.Detection{{
			Kind:   "snakebite-metadata",
			Path:   filepath.ToSlash(mustRel(input.ExtractedRoot, metadataPath)),
			Reason: "SnakeBite metadata.xml declares QAR/FPK package entries",
		}},
		Metadata: []installplan.ModMetadata{snakeBiteMetadataSummary(input.ExtractedRoot, metadataPath, metadata)},
	}
	err = filepath.WalkDir(contentRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(contentRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !safeSnakeBitePackageRelative(rel) {
			return installplan.Unsupported("SnakeBite package contains unsafe path " + rel)
		}
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      path,
			StagingRelative: rel,
		})
		return nil
	})
	if err != nil {
		return installplan.Plan{}, err
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, installplan.Unsupported("SnakeBite package has no files to stage")
	}
	sort.Slice(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].StagingRelative < plan.Instructions[j].StagingRelative
	})
	return plan, nil
}

func findSnakeBiteMetadata(root string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		return ""
	}
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || found != "" || d.IsDir() {
			return nil
		}
		if strings.EqualFold(d.Name(), "metadata.xml") && isSnakeBiteMetadata(path) {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func isSnakeBiteMetadata(path string) bool {
	metadata, err := readSnakeBiteMetadata(path)
	if err != nil {
		return false
	}
	return snakeBiteMetadataHasEntries(metadata)
}

func readSnakeBiteMetadata(path string) (snakeBiteMetadataProbe, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return snakeBiteMetadataProbe{}, err
	}
	var metadata snakeBiteMetadataProbe
	if err := xml.Unmarshal(data, &metadata); err != nil {
		return snakeBiteMetadataProbe{}, err
	}
	if metadata.XMLName.Local != "ModEntry" {
		return snakeBiteMetadataProbe{}, installplan.Unsupported("SnakeBite metadata.xml root is not ModEntry")
	}
	return metadata, nil
}

func snakeBiteMetadataHasEntries(metadata snakeBiteMetadataProbe) bool {
	if len(metadata.QarEntries.Entries) == 0 && len(metadata.FpkEntries.Entries) == 0 {
		return false
	}
	for _, entry := range metadata.QarEntries.Entries {
		if strings.TrimSpace(entry.FilePath) != "" {
			return true
		}
	}
	for _, entry := range metadata.FpkEntries.Entries {
		if strings.TrimSpace(entry.FpkFile) != "" || strings.TrimSpace(entry.FilePath) != "" {
			return true
		}
	}
	return false
}

func snakeBiteMetadataSummary(root, metadataPath string, metadata snakeBiteMetadataProbe) installplan.ModMetadata {
	name := strings.TrimSpace(metadata.Name)
	if name == "" {
		name = filepath.Base(root)
	}
	version := strings.TrimSpace(metadata.Version)
	if version == "" {
		version = strings.TrimSpace(metadata.MGSVersion.Version)
	}
	return installplan.ModMetadata{
		Kind:            "snakebite-metadata",
		SourcePath:      filepath.ToSlash(mustRel(root, metadataPath)),
		StagingRelative: "metadata.xml",
		Name:            name,
		UniqueID:        name,
		Version:         version,
	}
}

func missingSnakeBiteReferencedFiles(root string, metadata snakeBiteMetadataProbe) []string {
	seen := map[string]struct{}{}
	var missing []string
	check := func(value string) {
		rel := snakeBitePackageRelative(value)
		if rel == "" {
			return
		}
		if _, ok := seen[rel]; ok {
			return
		}
		seen[rel] = struct{}{}
		path := filepath.Join(root, filepath.FromSlash(rel))
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			missing = append(missing, rel)
		}
	}
	for _, entry := range metadata.QarEntries.Entries {
		check(entry.FilePath)
	}
	for _, entry := range metadata.FpkEntries.Entries {
		check(entry.FpkFile)
	}
	sort.Strings(missing)
	return missing
}

func snakeBitePackageRelative(value string) string {
	value = strings.TrimSpace(filepath.ToSlash(value))
	value = strings.TrimLeft(value, "/")
	if !safeSnakeBitePackageRelative(value) {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
}

func safeSnakeBitePackageRelative(value string) bool {
	value = strings.TrimSpace(filepath.ToSlash(value))
	if value == "" || strings.HasPrefix(value, "/") {
		return false
	}
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	return cleaned != "." && cleaned != ".." && !strings.HasPrefix(cleaned, "../")
}

func mustRel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.Base(path)
	}
	return rel
}
