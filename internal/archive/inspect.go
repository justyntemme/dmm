package archive

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
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
	case ".7z", ".rar":
		return inspectWith7z(filePath, strings.TrimPrefix(format, "."))
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
		}
	}
	for name := range top {
		inspection.TopLevelDirs = append(inspection.TopLevelDirs, name)
	}
	sort.Strings(inspection.TopLevelDirs)
	return inspection, nil
}

func inspectWith7z(filePath, format string) (Inspection, error) {
	inspection := Inspection{
		ArchivePath:      filePath,
		Format:           format,
		Entries:          []Entry{},
		TopLevelDirs:     []string{},
		Warnings:         []string{},
		RequiresExternal: true,
	}
	if _, err := exec.LookPath("7z"); err != nil {
		inspection.Warnings = append(inspection.Warnings, missing7zMessage("inspect this archive before extraction"))
		return inspection, nil
	}
	cmd := exec.Command("7z", "l", "-slt", "--", filePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Inspection{}, helperCommandError("7z", "inspect archive", out)
	}
	return parse7zListing(inspection, string(out)), nil
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
		}
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
	for name := range top {
		inspection.TopLevelDirs = append(inspection.TopLevelDirs, name)
	}
	sort.Strings(inspection.TopLevelDirs)
	return inspection
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
	if _, err := exec.LookPath("7z"); err != nil {
		return errors.New(missing7zMessage("extract this archive"))
	}
	cmd := exec.CommandContext(ctx, "7z", "x", "-y", "-o"+destDir, filePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return ctx.Err()
		}
		return helperCommandError("7z", "extract archive", out)
	}
	return nil
}

func extractRAR(ctx context.Context, filePath, destDir string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := exec.LookPath("unrar"); err == nil {
		cmd := exec.CommandContext(ctx, "unrar", "x", "-o+", filePath, destDir+string(filepath.Separator))
		out, err := cmd.CombinedOutput()
		if err != nil {
			if errors.Is(ctx.Err(), context.Canceled) {
				return ctx.Err()
			}
			return helperCommandError("unrar", "extract RAR archive", out)
		}
		return nil
	}
	if _, err := exec.LookPath("7z"); err != nil {
		return errors.New("unrar or 7z is required to extract this RAR archive. Install an archive helper from the Decky plugin Dependencies view, then retry the install.")
	}
	return extractWith7z(ctx, filePath, destDir)
}

func missing7zMessage(action string) string {
	return "7z is required to " + action + ". Install 7-Zip/p7zip from the Decky plugin Dependencies view, then retry the install."
}

func helperCommandError(tool, action string, out []byte) error {
	detail := strings.TrimSpace(string(out))
	if detail == "" {
		return errors.New(tool + " failed to " + action + " and did not report details. Check that the archive is valid and the helper is installed correctly.")
	}
	return errors.New(tool + " failed to " + action + ": " + detail)
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
		if err != nil || d.IsDir() || found != "" {
			return nil
		}
		rel, err := filepath.Rel(destDir, path)
		if err != nil {
			return nil
		}
		if isFOMODPath(rel) {
			found = "fomod"
		}
		return nil
	})
	return found
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
