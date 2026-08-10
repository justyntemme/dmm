package witcher3

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	w3MenuMergeGeneratedDir = "witcher3-menu-merge"
	w3BackupTag             = ".vortex_backup"
)

type menuFragment struct {
	ModID      int64
	ModName    string
	Priority   int
	SourcePath string
	TargetName string
}

type iniDocument struct {
	SectionOrder []string
	Sections     map[string]*iniSection
}

type iniSection struct {
	KeyOrder []string
	Values   map[string]string
}

func witcherMenuMergeMappings(ctx context.Context, input sdk.EventHandlerInput, documentsRoot string) ([]deploy.FileMapping, []string, error) {
	fragments, err := witcherMenuFragments(ctx, input)
	if err != nil {
		return nil, nil, err
	}
	if len(fragments) == 0 {
		return nil, []string{"Witcher 3 menu settings merge skipped because this profile has no enabled menu fragments."}, nil
	}
	sort.SliceStable(fragments, func(i, j int) bool {
		if fragments[i].TargetName != fragments[j].TargetName {
			return fragments[i].TargetName < fragments[j].TargetName
		}
		if fragments[i].Priority != fragments[j].Priority {
			return fragments[i].Priority < fragments[j].Priority
		}
		if strings.ToLower(fragments[i].ModName) != strings.ToLower(fragments[j].ModName) {
			return strings.ToLower(fragments[i].ModName) < strings.ToLower(fragments[j].ModName)
		}
		return fragments[i].SourcePath < fragments[j].SourcePath
	})

	var mappings []deploy.FileMapping
	var messages []string
	grouped := groupMenuFragments(fragments)
	targetNames := make([]string, 0, len(grouped))
	for targetName := range grouped {
		targetNames = append(targetNames, targetName)
	}
	sort.Strings(targetNames)
	for _, targetName := range targetNames {
		group := grouped[targetName]
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		targetPath := filepath.Join(documentsRoot, targetName)
		basePath, base, managedRestore, err := menuSettingsBase(input, targetPath)
		if err != nil {
			return nil, nil, err
		}
		if len(base) == 0 {
			messages = append(messages, fmt.Sprintf("Witcher 3 menu settings merge skipped %s because no base Documents file exists.", targetName))
			continue
		}
		doc, err := parseINIDocument(string(base))
		if err != nil {
			return nil, nil, fmt.Errorf("parse Witcher 3 menu settings base %s: %w", targetName, err)
		}
		for _, fragment := range group {
			body, err := os.ReadFile(fragment.SourcePath)
			if err != nil {
				return nil, nil, fmt.Errorf("read Witcher 3 menu fragment %s: %w", fragment.SourcePath, err)
			}
			part, err := parseINIDocument(string(body))
			if err != nil {
				return nil, nil, fmt.Errorf("parse Witcher 3 menu fragment %s: %w", filepath.Base(fragment.SourcePath), err)
			}
			doc.merge(part)
		}
		sourcePath, err := writeMenuGeneratedFile(input, "patched", targetName, []byte(doc.render()))
		if err != nil {
			return nil, nil, err
		}
		restorePath := managedRestore
		if restorePath == "" {
			restorePath, err = writeMenuGeneratedFile(input, "restore", targetName, base)
			if err != nil {
				return nil, nil, err
			}
		}
		mappings = append(mappings, deploy.FileMapping{
			SourcePath:     sourcePath,
			RestorePath:    restorePath,
			TargetRoot:     documentsRoot,
			TargetRelative: targetName,
			TargetPolicy:   deploy.TargetPolicyPatchExisting,
			Strategy:       deploy.StrategyCopy,
			ModID:          "witcher3-menu-merge",
			Priority:       -1,
		})
		messages = append(messages, fmt.Sprintf("Witcher 3 menu settings %s merged from %d DMM-managed fragment(s) using %s as the base.", targetName, len(group), filepath.Base(basePath)))
	}
	if len(mappings) == 0 && len(messages) == 0 {
		messages = append(messages, "Witcher 3 menu settings merge skipped because no mergeable base files were found.")
	}
	return mappings, messages, nil
}

func witcherMenuFragments(ctx context.Context, input sdk.EventHandlerInput) ([]menuFragment, error) {
	var fragments []menuFragment
	for _, mod := range input.Mods {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !mod.Enabled || !strings.EqualFold(mod.ModType, "witcher3menumodroot") || strings.TrimSpace(mod.StagingPath) == "" {
			continue
		}
		modFragments, err := scanMenuFragments(ctx, mod)
		if err != nil {
			return nil, err
		}
		fragments = append(fragments, modFragments...)
	}
	return fragments, nil
}

func scanMenuFragments(ctx context.Context, mod sdk.DeploymentMod) ([]menuFragment, error) {
	root := strings.TrimSpace(mod.StagingPath)
	var fragments []menuFragment
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
				return filepath.SkipDir
			}
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() || !witcherMenuFragmentPath(path) {
			return nil
		}
		targetName := menuFragmentTargetName(path)
		if targetName == "" {
			return nil
		}
		fragments = append(fragments, menuFragment{
			ModID:      mod.ID,
			ModName:    mod.Name,
			Priority:   mod.Priority,
			SourcePath: path,
			TargetName: targetName,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return fragments, nil
}

func menuFragmentTargetName(pathValue string) string {
	name := strings.ToLower(filepath.Base(pathValue))
	if !strings.HasSuffix(name, partSuffix) {
		return ""
	}
	name = strings.TrimSuffix(name, partSuffix)
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return ""
	}
	return name
}

func groupMenuFragments(fragments []menuFragment) map[string][]menuFragment {
	out := map[string][]menuFragment{}
	for _, fragment := range fragments {
		out[fragment.TargetName] = append(out[fragment.TargetName], fragment)
	}
	return out
}

func menuSettingsBase(input sdk.EventHandlerInput, targetPath string) (string, []byte, string, error) {
	if managed, ok := managedMenuRestoreForTarget(input.ManagedFiles, targetPath); ok {
		body, err := os.ReadFile(managed.RestorePath)
		if err != nil {
			return "", nil, "", err
		}
		return managed.RestorePath, body, managed.RestorePath, nil
	}
	for _, candidate := range []string{targetPath + w3BackupTag, targetPath} {
		body, err := os.ReadFile(candidate)
		if err == nil {
			return candidate, body, "", nil
		}
		if !os.IsNotExist(err) {
			return "", nil, "", err
		}
	}
	return targetPath, nil, "", nil
}

func managedMenuRestoreForTarget(files []deploy.AppliedFile, targetPath string) (deploy.AppliedFile, bool) {
	targetPath = filepath.Clean(targetPath)
	for _, file := range files {
		if strings.TrimSpace(file.RestorePath) == "" {
			continue
		}
		if filepath.Clean(file.TargetPath) == targetPath {
			return file, true
		}
	}
	return deploy.AppliedFile{}, false
}

func writeMenuGeneratedFile(input sdk.EventHandlerInput, group, name string, contents []byte) (string, error) {
	name = menuFragmentTargetName(name + partSuffix)
	if name == "" {
		return "", fmt.Errorf("unsafe Witcher 3 menu generated file name %q", name)
	}
	root := filepath.Join(input.WorkDir, w3MenuMergeGeneratedDir, group)
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func parseINIDocument(body string) (*iniDocument, error) {
	doc := &iniDocument{Sections: map[string]*iniSection{}}
	current := doc.section("")
	scanner := bufio.NewScanner(strings.NewReader(strings.ReplaceAll(body, "\r\n", "\n")))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.Contains(line, "]") {
			end := strings.Index(line, "]")
			current = doc.section(strings.TrimSpace(line[1:end]))
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		current.set(strings.TrimSpace(key), strings.TrimSpace(value))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return doc, nil
}

func (doc *iniDocument) section(name string) *iniSection {
	name = strings.TrimSpace(name)
	key := strings.ToLower(name)
	section, ok := doc.Sections[key]
	if !ok {
		section = &iniSection{Values: map[string]string{}}
		doc.Sections[key] = section
		doc.SectionOrder = append(doc.SectionOrder, name)
	}
	return section
}

func (section *iniSection) set(key, value string) {
	if key == "" {
		return
	}
	canonical := strings.ToLower(key)
	if _, ok := section.Values[canonical]; !ok {
		section.KeyOrder = append(section.KeyOrder, key)
	}
	section.Values[canonical] = value
}

func (doc *iniDocument) merge(other *iniDocument) {
	if other == nil {
		return
	}
	for _, sectionName := range other.SectionOrder {
		source := other.Sections[strings.ToLower(sectionName)]
		target := doc.section(sectionName)
		for _, key := range source.KeyOrder {
			target.set(key, source.Values[strings.ToLower(key)])
		}
	}
}

func (doc *iniDocument) render() string {
	var b strings.Builder
	for _, sectionName := range doc.SectionOrder {
		section := doc.Sections[strings.ToLower(sectionName)]
		if section == nil || len(section.KeyOrder) == 0 {
			continue
		}
		if sectionName != "" {
			b.WriteString("[")
			b.WriteString(sectionName)
			b.WriteString("]")
			b.WriteString(w3LoadOrderLineBreak)
		}
		for _, key := range section.KeyOrder {
			b.WriteString(key)
			b.WriteString("=")
			b.WriteString(section.Values[strings.ToLower(key)])
			b.WriteString(w3LoadOrderLineBreak)
		}
		b.WriteString(w3LoadOrderLineBreak)
	}
	return b.String()
}
