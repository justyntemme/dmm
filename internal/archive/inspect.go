package archive

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bodgit/sevenzip"
	"github.com/nwaples/rardecode/v2"
)

type Entry struct {
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	IsDir bool   `json:"is_dir"`
}

type Inspection struct {
	ArchivePath       string   `json:"archive_path"`
	Format            string   `json:"format"`
	InstallerKind     string   `json:"installer_kind,omitempty"`
	Entries           []Entry  `json:"entries"`
	TopLevelDirs      []string `json:"top_level_dirs"`
	Warnings          []string `json:"warnings"`
	Unsafe            bool     `json:"unsafe"`
	RequiresExternal  bool     `json:"requires_external"`
	RequiresInstaller bool     `json:"requires_installer"`
}

func Inspect(filePath string) (Inspection, error) {
	format, err := detectArchiveFormat(filePath)
	if err != nil {
		return Inspection{}, err
	}
	switch format {
	case ".zip":
		return inspectZip(filePath)
	case ".7z":
		return inspect7z(filePath)
	case ".rar":
		return inspectRAR(filePath)
	default:
		return Inspection{}, errors.New("unsupported archive format")
	}
}

func detectArchiveFormat(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".zip", ".7z", ".rar":
		return ext, nil
	}
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	header := make([]byte, 8)
	n, err := io.ReadFull(file, header)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return "", err
	}
	header = header[:n]
	if len(header) >= 4 && header[0] == 'P' && header[1] == 'K' && (header[2] == 0x03 || header[2] == 0x05 || header[2] == 0x07) && (header[3] == 0x04 || header[3] == 0x06 || header[3] == 0x08) {
		return ".zip", nil
	}
	if len(header) >= 6 && string(header[:6]) == "7z\xbc\xaf'\x1c" {
		return ".7z", nil
	}
	if len(header) >= 7 && string(header[:7]) == "Rar!\x1a\x07\x00" {
		return ".rar", nil
	}
	if len(header) >= 8 && string(header[:8]) == "Rar!\x1a\x07\x01\x00" {
		return ".rar", nil
	}
	return "", errors.New("unsupported archive format")
}

func Extract(filePath, destDir string) (Inspection, error) {
	return ExtractContext(context.Background(), filePath, destDir)
}

func ExtractContext(ctx context.Context, filePath, destDir string) (Inspection, error) {
	if err := ctx.Err(); err != nil {
		return Inspection{}, err
	}
	inspection, err := Inspect(filePath)
	if err != nil {
		return Inspection{}, err
	}
	if inspection.Unsafe {
		return Inspection{}, errors.New("archive is unsafe and cannot be extracted")
	}
	if destDir == "" {
		return Inspection{}, errors.New("destination directory is required")
	}
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return Inspection{}, err
	}

	switch inspection.Format {
	case "zip":
		if err := extractZip(ctx, filePath, destDir); err != nil {
			return Inspection{}, err
		}
	case "7z":
		if err := extractWith7z(ctx, filePath, destDir); err != nil {
			return Inspection{}, err
		}
	case "rar":
		if err := extractRAR(ctx, filePath, destDir); err != nil {
			return Inspection{}, err
		}
	default:
		return Inspection{}, errors.New("unsupported archive format")
	}
	if err := validateExtractedTree(ctx, destDir); err != nil {
		return Inspection{}, err
	}
	if inspection.InstallerKind == "" {
		inspection.InstallerKind = detectExtractedInstaller(destDir)
	}
	if inspection.InstallerKind != "" {
		inspection.RequiresInstaller = true
	}
	return inspection, nil
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
		if isFOMODPath(entry.Path) {
			inspection.InstallerKind = "fomod"
			inspection.RequiresInstaller = true
		} else if inspection.InstallerKind == "" && isNestedFOMODArchive(entry.Path, entry.IsDir) {
			inspection.InstallerKind = "nested_fomod"
			inspection.RequiresInstaller = true
		}
	}
	for name := range top {
		inspection.TopLevelDirs = append(inspection.TopLevelDirs, name)
	}
	sort.Strings(inspection.TopLevelDirs)
	return inspection, nil
}

func inspect7z(filePath string) (Inspection, error) {
	inspection := Inspection{
		ArchivePath:  filePath,
		Format:       "7z",
		Entries:      []Entry{},
		TopLevelDirs: []string{},
		Warnings:     []string{},
	}
	reader, err := sevenzip.OpenReader(filePath)
	if err != nil {
		return Inspection{}, err
	}
	defer reader.Close()
	top := map[string]struct{}{}
	for _, file := range reader.File {
		info := file.FileInfo()
		entry := Entry{
			Path:  filepath.ToSlash(file.Name),
			Size:  info.Size(),
			IsDir: info.IsDir(),
		}
		addInspectionEntry(&inspection, top, entry)
	}
	finishTopLevelDirs(&inspection, top)
	return inspection, nil
}

func inspectRAR(filePath string) (Inspection, error) {
	inspection := Inspection{
		ArchivePath:  filePath,
		Format:       "rar",
		Entries:      []Entry{},
		TopLevelDirs: []string{},
		Warnings:     []string{},
	}
	reader, err := rardecode.OpenReader(filePath)
	if err != nil {
		return Inspection{}, err
	}
	defer reader.Close()
	top := map[string]struct{}{}
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Inspection{}, err
		}
		entry := Entry{
			Path:  filepath.ToSlash(header.Name),
			Size:  header.UnPackedSize,
			IsDir: header.IsDir,
		}
		addInspectionEntry(&inspection, top, entry)
	}
	finishTopLevelDirs(&inspection, top)
	return inspection, nil
}

func parse7zListing(inspection Inspection, listing string) Inspection {
	top := map[string]struct{}{}
	current := map[string]string{}
	flush := func() {
		entryPath := strings.TrimSpace(current["Path"])
		if entryPath == "" || entryPath == inspection.ArchivePath {
			current = map[string]string{}
			return
		}
		entry := Entry{
			Path:  filepath.ToSlash(entryPath),
			IsDir: strings.EqualFold(current["Folder"], "+"),
		}
		if size, ok := parseInt64(current["Size"]); ok {
			entry.Size = size
		}
		addInspectionEntry(&inspection, top, entry)
		current = map[string]string{}
	}
	for _, line := range strings.Split(listing, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		key, value, ok := strings.Cut(line, " = ")
		if !ok {
			continue
		}
		current[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	flush()
	finishTopLevelDirs(&inspection, top)
	return inspection
}

func addInspectionEntry(inspection *Inspection, top map[string]struct{}, entry Entry) {
	inspection.Entries = append(inspection.Entries, entry)
	if warning := validateArchivePath(entry.Path); warning != "" {
		inspection.Warnings = append(inspection.Warnings, warning)
		inspection.Unsafe = true
	}
	first := firstPathSegment(entry.Path)
	if first != "" {
		top[first] = struct{}{}
	}
	if isFOMODPath(entry.Path) {
		inspection.InstallerKind = "fomod"
		inspection.RequiresInstaller = true
	} else if inspection.InstallerKind == "" && isNestedFOMODArchive(entry.Path, entry.IsDir) {
		inspection.InstallerKind = "nested_fomod"
		inspection.RequiresInstaller = true
	}
}

func finishTopLevelDirs(inspection *Inspection, top map[string]struct{}) {
	for name := range top {
		inspection.TopLevelDirs = append(inspection.TopLevelDirs, name)
	}
	sort.Strings(inspection.TopLevelDirs)
}

func parseInt64(value string) (int64, bool) {
	var out int64
	for _, r := range strings.TrimSpace(value) {
		if r < '0' || r > '9' {
			return 0, false
		}
		out = out*10 + int64(r-'0')
	}
	return out, strings.TrimSpace(value) != ""
}

func extractZip(ctx context.Context, filePath, destDir string) error {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		name := filepath.ToSlash(file.Name)
		if warning := validateArchivePath(name); warning != "" {
			return errors.New(warning)
		}
		target, err := safeExtractPath(destDir, name)
		if err != nil {
			return err
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o700); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		in, err := file.Open()
		if err != nil {
			return err
		}
		err = writeExtractedFile(ctx, target, in)
		closeErr := in.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func extractWith7z(ctx context.Context, filePath, destDir string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	reader, err := sevenzip.OpenReader(filePath)
	if err != nil {
		return err
	}
	defer reader.Close()
	for _, file := range reader.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := extractArchiveFile(ctx, destDir, file.Name, file.FileInfo().IsDir(), file.Open); err != nil {
			return err
		}
	}
	return nil
}

func extractRAR(ctx context.Context, filePath, destDir string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	reader, err := rardecode.OpenReader(filePath)
	if err != nil {
		return err
	}
	defer reader.Close()
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		name := header.Name
		err = extractArchiveFile(ctx, destDir, name, header.IsDir, func() (io.ReadCloser, error) {
			return io.NopCloser(reader), nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func extractArchiveFile(ctx context.Context, destDir, name string, isDir bool, open func() (io.ReadCloser, error)) error {
	name = filepath.ToSlash(name)
	if warning := validateArchivePath(name); warning != "" {
		return errors.New(warning)
	}
	target, err := safeExtractPath(destDir, name)
	if err != nil {
		return err
	}
	if isDir {
		return os.MkdirAll(target, 0o700)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	in, err := open()
	if err != nil {
		return err
	}
	err = writeExtractedFile(ctx, target, in)
	closeErr := in.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func writeExtractedFile(ctx context.Context, target string, in io.Reader) error {
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, contextReader{ctx: ctx, reader: in}); err != nil {
		return err
	}
	return out.Sync()
}

func safeExtractPath(destDir, name string) (string, error) {
	target := filepath.Join(destDir, filepath.FromSlash(name))
	rel, err := filepath.Rel(destDir, target)
	if err != nil {
		return "", err
	}
	if rel == "." || strings.HasPrefix(filepath.ToSlash(rel), "../") {
		return "", errors.New("archive entry escapes destination: " + name)
	}
	return target, nil
}

func validateExtractedTree(ctx context.Context, destDir string) error {
	return filepath.WalkDir(destDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink == 0 {
			return nil
		}
		return errors.New("extracted archive contains symlink: " + path)
	})
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := r.reader.Read(p)
	if err != nil {
		return n, err
	}
	if ctxErr := r.ctx.Err(); ctxErr != nil {
		return n, ctxErr
	}
	return n, nil
}

func detectExtractedInstaller(destDir string) string {
	found := ""
	_ = filepath.WalkDir(destDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(destDir, path)
		if err != nil {
			return nil
		}
		if isFOMODPath(rel) {
			found = "fomod"
			return nil
		}
		if found == "" && isNestedFOMODArchive(rel, false) {
			found = "nested_fomod"
		}
		return nil
	})
	return found
}

func FindNestedFOMODArchive(destDir string) (string, error) {
	var found string
	err := filepath.WalkDir(destDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || found != "" {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(destDir, path)
		if err != nil {
			return err
		}
		if !isNestedFOMODArchive(rel, false) {
			return nil
		}
		found = path
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", errors.New("nested .fomod archive was not found")
	}
	return found, nil
}

func isFOMODPath(name string) bool {
	parts := strings.Split(strings.ToLower(filepath.ToSlash(name)), "/")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] != "fomod" {
			continue
		}
		switch parts[i+1] {
		case "moduleconfig.xml", "info.xml":
			return true
		}
	}
	return false
}

func isNestedFOMODArchive(name string, isDir bool) bool {
	if isDir {
		return false
	}
	return strings.EqualFold(filepath.Ext(filepath.ToSlash(name)), ".fomod")
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
