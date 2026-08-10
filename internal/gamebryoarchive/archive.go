package gamebryoarchive

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pierrec/lz4/v4"
)

type Entry struct {
	Path string
	Size uint64
}

type Reader interface {
	Type() string
	Version() uint32
	List() []Entry
	ReadFile(name string) ([]byte, error)
	ExtractAll(outputDir string) error
}

func Open(path string) (Reader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var magic [4]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil {
		return nil, err
	}
	switch string(magic[:]) {
	case ba2Magic:
		return OpenBA2(path)
	case bsaMagic:
		return OpenBSA(path, false)
	default:
		return nil, fmt.Errorf("unsupported Gamebryo archive magic %q", strings.ReplaceAll(string(magic[:]), "\x00", "\\0"))
	}
}

func sortedEntries(entries []Entry) []Entry {
	out := append([]Entry(nil), entries...)
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Path) < strings.ToLower(out[j].Path)
	})
	return out
}

func normalizeArchivePath(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "/", "\\")
	parts := make([]string, 0)
	for _, part := range strings.Split(value, "\\") {
		part = strings.TrimSpace(part)
		if part == "" || part == "." {
			continue
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "\\")
}

func archivePathEqual(a, b string) bool {
	return strings.EqualFold(normalizeArchivePath(a), normalizeArchivePath(b))
}

func safeOutputPath(root, archivePath string) (string, error) {
	archivePath = normalizeArchivePath(archivePath)
	if archivePath == "" {
		return "", errors.New("archive path is empty")
	}
	parts := strings.Split(archivePath, "\\")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("archive path %q is unsafe", archivePath)
		}
	}
	root = filepath.Clean(root)
	out := filepath.Join(append([]string{root}, parts...)...)
	rel, err := filepath.Rel(root, out)
	if err != nil {
		return "", err
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return "", fmt.Errorf("archive path %q escapes output root", archivePath)
	}
	return out, nil
}

func inflateZlib(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func inflateLZ4Frame(data []byte) ([]byte, error) {
	r := lz4.NewReader(bytes.NewReader(data))
	return io.ReadAll(r)
}
