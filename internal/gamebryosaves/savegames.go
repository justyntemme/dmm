package gamebryosaves

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

type Slot string

const (
	SlotProfile Slot = "profile"
	SlotGlobal  Slot = "global"
)

type Save struct {
	ID           string    `json:"id"`
	FileName     string    `json:"file_name"`
	FilePath     string    `json:"file_path"`
	Size         int64     `json:"size"`
	ModifiedAt   time.Time `json:"modified_at"`
	Slot         Slot      `json:"slot"`
	ProfileID    int64     `json:"profile_id,omitempty"`
	Character    string    `json:"character,omitempty"`
	Level        int       `json:"level,omitempty"`
	Location     string    `json:"location,omitempty"`
	PlayTime     string    `json:"play_time,omitempty"`
	Plugins      []string  `json:"plugins,omitempty"`
	Corrupted    bool      `json:"corrupted,omitempty"`
	CorruptError string    `json:"corrupt_error,omitempty"`
}

type Service struct {
	Spec       sdk.SavegameManagementSpec
	Documents  string
	ProfileID  int64
	LocalSaves bool
}

func (s Service) List(slot Slot) ([]Save, error) {
	root, err := s.slotPath(slot)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	allowed := extensionsSet(s.Spec.SaveExtensions)
	out := make([]Save, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !allowed[strings.ToLower(filepath.Ext(entry.Name()))] {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		path := filepath.Join(root, entry.Name())
		save := Save{
			ID:         entry.Name(),
			FileName:   entry.Name(),
			FilePath:   path,
			Size:       info.Size(),
			ModifiedAt: info.ModTime(),
			Slot:       slot,
			ProfileID:  s.ProfileID,
		}
		if meta, err := Parse(path); err == nil {
			save.Character = meta.Character
			save.Level = meta.Level
			save.Location = meta.Location
			save.PlayTime = meta.PlayTime
			save.Plugins = meta.Plugins
		} else {
			save.Corrupted = true
			save.CorruptError = err.Error()
		}
		out = append(out, save)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ModifiedAt.After(out[j].ModifiedAt)
	})
	return out, nil
}

func (s Service) Transfer(source Slot, names []string, keepSource bool) ([]string, error) {
	src, err := s.slotPath(source)
	if err != nil {
		return nil, err
	}
	dst, err := s.slotPath(SlotProfile)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return nil, err
	}
	var failed []string
	for _, name := range expandSaveFiles(names, s.Spec.SidecarPatterns) {
		clean, ok := cleanSaveName(name)
		if !ok {
			failed = append(failed, name+" - unsafe path")
			continue
		}
		from := filepath.Join(src, clean)
		to := filepath.Join(dst, clean)
		if keepSource {
			if err := copyFile(from, to); err != nil && !errors.Is(err, fs.ErrNotExist) {
				failed = append(failed, clean+" - "+err.Error())
			}
			continue
		}
		if err := os.Rename(from, to); err != nil && !errors.Is(err, fs.ErrNotExist) {
			failed = append(failed, clean+" - "+err.Error())
		}
	}
	return failed, nil
}

func (s Service) Delete(slot Slot, names []string) ([]string, error) {
	root, err := s.slotPath(slot)
	if err != nil {
		return nil, err
	}
	var failed []string
	for _, name := range expandSaveFiles(names, s.Spec.SidecarPatterns) {
		clean, ok := cleanSaveName(name)
		if !ok {
			failed = append(failed, name+" - unsafe path")
			continue
		}
		if err := os.Remove(filepath.Join(root, clean)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			failed = append(failed, clean+" - "+err.Error())
		}
	}
	return failed, nil
}

func (s Service) Path(slot Slot) (string, error) {
	return s.slotPath(slot)
}

func (s Service) slotPath(slot Slot) (string, error) {
	base := strings.TrimSpace(s.Documents)
	if base == "" {
		return "", errors.New("documents path is unavailable")
	}
	switch slot {
	case SlotGlobal:
		return filepath.Join(base, filepath.FromSlash(s.Spec.Path), filepath.FromSlash(s.Spec.GlobalPath)), nil
	case SlotProfile:
		path := s.Spec.GlobalPath
		if s.LocalSaves {
			path = strings.ReplaceAll(s.Spec.LocalPath, "{profile_id}", int64String(s.ProfileID))
		}
		return filepath.Join(base, filepath.FromSlash(s.Spec.Path), filepath.FromSlash(path)), nil
	default:
		return "", errors.New("unsupported savegame slot")
	}
}

type Metadata struct {
	Character string
	Level     int
	Location  string
	PlayTime  string
	Plugins   []string
}

func Parse(path string) (Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Metadata{}, err
	}
	if len(data) == 0 {
		return Metadata{}, errors.New("empty savegame")
	}
	if !(bytes.HasPrefix(data, []byte("TES4SAVEGAME")) || bytes.HasPrefix(data, []byte("TESV_SAVEGAME")) || bytes.HasPrefix(data, []byte("FO3SAVEGAME")) || bytes.HasPrefix(data, []byte("FO4_SAVEGAME"))) {
		return Metadata{}, errors.New("invalid file header")
	}
	return Metadata{Plugins: extractPlugins(data)}, nil
}

var pluginNamePattern = regexp.MustCompile(`(?i)[A-Za-z0-9_ .'\-\[\]\(\)]+?\.(?:esm|esp|esl)`)

func extractPlugins(data []byte) []string {
	matches := pluginNamePattern.FindAll(data, -1)
	seen := map[string]bool{}
	var plugins []string
	for _, raw := range matches {
		name := strings.TrimSpace(strings.Map(func(r rune) rune {
			if r < 32 || r == 127 {
				return -1
			}
			return r
		}, string(raw)))
		key := strings.ToLower(name)
		if name == "" || seen[key] {
			continue
		}
		seen[key] = true
		plugins = append(plugins, name)
	}
	return plugins
}

func expandSaveFiles(names, sidecars []string) []string {
	var out []string
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		out = append(out, name)
		base := strings.TrimSuffix(name, filepath.Ext(name))
		for _, suffix := range sidecars {
			suffix = strings.TrimSpace(suffix)
			if suffix == "" {
				continue
			}
			if !strings.HasPrefix(suffix, ".") {
				suffix = "." + suffix
			}
			out = append(out, base+suffix)
		}
	}
	return out
}

func cleanSaveName(value string) (string, bool) {
	clean := filepath.Clean(filepath.FromSlash(strings.TrimSpace(value)))
	if clean == "." || clean == ".." || filepath.IsAbs(clean) || strings.Contains(clean, string(filepath.Separator)) {
		return "", false
	}
	return clean, true
}

func extensionsSet(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			out[value] = true
		}
	}
	return out
}

func copyFile(from, to string) error {
	data, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	return os.WriteFile(to, data, 0o644)
}

func int64String(value int64) string {
	if value == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	v := value
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
