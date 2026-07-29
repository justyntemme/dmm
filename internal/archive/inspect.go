package archive

import (
	"archive/zip"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type Entry struct {
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	IsDir bool   `json:"is_dir"`
}

type Inspection struct {
	ArchivePath      string   `json:"archive_path"`
	Format           string   `json:"format"`
	Entries          []Entry  `json:"entries"`
	TopLevelDirs     []string `json:"top_level_dirs"`
	Warnings         []string `json:"warnings"`
	Unsafe           bool     `json:"unsafe"`
	RequiresExternal bool     `json:"requires_external"`
}

func Inspect(filePath string) (Inspection, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".zip":
		return inspectZip(filePath)
	case ".7z", ".rar":
		return Inspection{
			ArchivePath:      filePath,
			Format:           strings.TrimPrefix(ext, "."),
			RequiresExternal: true,
			Warnings:         []string{"Archive listing requires external helper support."},
		}, nil
	default:
		return Inspection{}, errors.New("unsupported archive format")
	}
}

func inspectZip(filePath string) (Inspection, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return Inspection{}, err
	}
	defer reader.Close()

	inspection := Inspection{
		ArchivePath:  filePath,
		Format:       "zip",
		Entries:      []Entry{},
		TopLevelDirs: []string{},
		Warnings:     []string{},
	}
	top := map[string]struct{}{}
	for _, file := range reader.File {
		entry := Entry{
			Path:  filepath.ToSlash(file.Name),
			Size:  int64(file.UncompressedSize64),
			IsDir: file.FileInfo().IsDir(),
		}
		inspection.Entries = append(inspection.Entries, entry)
		if warning := validateArchivePath(entry.Path); warning != "" {
			inspection.Warnings = append(inspection.Warnings, warning)
			inspection.Unsafe = true
		}
		first := firstPathSegment(entry.Path)
		if first != "" {
			top[first] = struct{}{}
		}
	}
	for name := range top {
		inspection.TopLevelDirs = append(inspection.TopLevelDirs, name)
	}
	sort.Strings(inspection.TopLevelDirs)
	return inspection, nil
}

func validateArchivePath(name string) string {
	clean := path.Clean(filepath.ToSlash(name))
	if strings.HasPrefix(name, "/") {
		return "Archive contains absolute path: " + name
	}
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "Archive contains path traversal: " + name
	}
	if filepath.IsAbs(name) {
		return "Archive contains absolute path: " + name
	}
	return ""
}

func firstPathSegment(name string) string {
	name = strings.Trim(filepath.ToSlash(name), "/")
	if name == "" || strings.HasPrefix(name, "..") {
		return ""
	}
	parts := strings.Split(name, "/")
	return parts[0]
}

func CreateTestZip(filePath string, files map[string]string) error {
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	writer := zip.NewWriter(out)
	defer writer.Close()

	for name, contents := range files {
		entry, err := writer.Create(name)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(entry, contents); err != nil {
			return err
		}
	}
	return nil
}
