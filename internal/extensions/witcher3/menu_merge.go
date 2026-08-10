package witcher3

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf16"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	w3MenuMergeGeneratedDir = "witcher3-menu-merge"
	w3MenuXMLGeneratedDir   = "witcher3-config-xml-merge"
	w3BackupTag             = ".vortex_backup"
)

var (
	w3ConfigXMLNames = map[string]struct{}{
		"audio":        {},
		"display":      {},
		"gameplay":     {},
		"gamma":        {},
		"graphics":     {},
		"graphicsdx11": {},
		"hdr":          {},
		"hidden":       {},
		"hud":          {},
		"input":        {},
		"localization": {},
	}
	xmlEncodingDeclRE = regexp.MustCompile(`(?i)encoding\s*=\s*("[^"]*"|'[^']*')`)
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

type xmlMergeKey struct {
	TargetRoot     string
	TargetRelative string
}

type xmlNode struct {
	Name     xml.Name
	Attrs    []xml.Attr
	Text     string
	Children []*xmlNode
}

func witcherConfigXMLMergeMappings(ctx context.Context, input sdk.EventHandlerInput) ([]deploy.FileMapping, []deploy.FileMapping, []string, bool, error) {
	kept, groups := splitWitcherConfigXMLMappings(input.Mappings)
	if len(groups) == 0 {
		return kept, nil, nil, false, nil
	}
	keys := make([]xmlMergeKey, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].TargetRoot != keys[j].TargetRoot {
			return keys[i].TargetRoot < keys[j].TargetRoot
		}
		return keys[i].TargetRelative < keys[j].TargetRelative
	})
	var generated []deploy.FileMapping
	var messages []string
	for _, key := range keys {
		if err := ctx.Err(); err != nil {
			return nil, nil, nil, true, err
		}
		mapping, message, ok, err := mergeWitcherConfigXMLGroup(input, key, groups[key])
		if err != nil {
			return nil, nil, nil, true, err
		}
		if strings.TrimSpace(message) != "" {
			messages = append(messages, message)
		}
		if ok {
			generated = append(generated, mapping)
		}
	}
	if len(generated) == 0 && len(messages) == 0 {
		messages = append(messages, "Witcher 3 config XML merge skipped because no native config files were found.")
	}
	return kept, generated, messages, true, nil
}

func splitWitcherConfigXMLMappings(mappings []deploy.FileMapping) ([]deploy.FileMapping, map[xmlMergeKey][]deploy.FileMapping) {
	groups := map[xmlMergeKey][]deploy.FileMapping{}
	kept := make([]deploy.FileMapping, 0, len(mappings))
	for _, mapping := range mappings {
		targetRel, ok := witcherConfigXMLTarget(mapping.TargetRelative)
		if !ok {
			kept = append(kept, mapping)
			continue
		}
		key := xmlMergeKey{
			TargetRoot:     strings.TrimSpace(mapping.TargetRoot),
			TargetRelative: targetRel,
		}
		groups[key] = append(groups[key], mapping)
	}
	return kept, groups
}

func witcherConfigXMLTarget(targetRelative string) (string, bool) {
	rel := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(targetRelative))))
	if rel == "." || rel == "" || strings.HasPrefix(rel, "../") {
		return "", false
	}
	lower := strings.ToLower(rel)
	prefix := strings.ToLower(configMatrixRelPath) + "/"
	if !strings.HasPrefix(lower, prefix) {
		return "", false
	}
	base := strings.ToLower(filepath.Base(lower))
	if filepath.Ext(base) != ".xml" {
		return "", false
	}
	name := strings.TrimSuffix(base, ".xml")
	if _, ok := w3ConfigXMLNames[name]; !ok {
		return "", false
	}
	return rel, true
}

func mergeWitcherConfigXMLGroup(input sdk.EventHandlerInput, key xmlMergeKey, mappings []deploy.FileMapping) (deploy.FileMapping, string, bool, error) {
	if len(mappings) == 0 {
		return deploy.FileMapping{}, "", false, nil
	}
	sort.SliceStable(mappings, func(i, j int) bool {
		if mappings[i].Priority != mappings[j].Priority {
			return mappings[i].Priority < mappings[j].Priority
		}
		if mappings[i].InstalledModID != mappings[j].InstalledModID {
			return mappings[i].InstalledModID < mappings[j].InstalledModID
		}
		if strings.TrimSpace(mappings[i].ModID) != strings.TrimSpace(mappings[j].ModID) {
			return strings.TrimSpace(mappings[i].ModID) < strings.TrimSpace(mappings[j].ModID)
		}
		return mappings[i].SourcePath < mappings[j].SourcePath
	})
	targetPath, err := witcherConfigXMLTargetPath(input, key)
	if err != nil {
		return deploy.FileMapping{}, "", false, err
	}
	basePath, base, managedRestore, err := menuSettingsBase(input, targetPath)
	if err != nil {
		return deploy.FileMapping{}, "", false, err
	}
	if len(base) == 0 {
		return deploy.FileMapping{}, fmt.Sprintf("Witcher 3 config XML merge skipped %s because no native config XML exists; run or verify the game before installing menu mods.", key.TargetRelative), false, nil
	}
	merged, err := parseXMLNode(base)
	if err != nil {
		return deploy.FileMapping{}, "", false, fmt.Errorf("parse Witcher 3 config XML base %s: %w", key.TargetRelative, err)
	}
	if !strings.EqualFold(merged.Name.Local, "UserConfig") {
		return deploy.FileMapping{}, "", false, fmt.Errorf("Witcher 3 config XML base %s root is %q, expected UserConfig", key.TargetRelative, merged.Name.Local)
	}
	for _, mapping := range mappings {
		sourcePath := witcherConfigXMLSourcePath(input, mapping)
		if sourcePath == "" {
			return deploy.FileMapping{}, "", false, fmt.Errorf("Witcher 3 config XML mapping %s has no source path", key.TargetRelative)
		}
		body, err := os.ReadFile(sourcePath)
		if err != nil {
			return deploy.FileMapping{}, "", false, fmt.Errorf("read Witcher 3 config XML source %s: %w", filepath.Base(sourcePath), err)
		}
		modXML, err := parseXMLNode(body)
		if err != nil {
			return deploy.FileMapping{}, "", false, fmt.Errorf("parse Witcher 3 config XML source %s: %w", filepath.Base(sourcePath), err)
		}
		if !strings.EqualFold(modXML.Name.Local, "UserConfig") || len(userConfigGroups(modXML)) == 0 {
			return deploy.FileMapping{}, "", false, fmt.Errorf("Witcher 3 config XML source %s does not contain UserConfig.Group entries", filepath.Base(sourcePath))
		}
		mergeUserConfigGroups(merged, modXML)
	}
	sourcePath, err := writeMenuGeneratedFileIn(input, w3MenuXMLGeneratedDir, "patched", key.TargetRelative, renderXMLNode(merged))
	if err != nil {
		return deploy.FileMapping{}, "", false, err
	}
	restorePath := managedRestore
	if restorePath == "" {
		restorePath, err = writeMenuGeneratedFileIn(input, w3MenuXMLGeneratedDir, "restore", key.TargetRelative, base)
		if err != nil {
			return deploy.FileMapping{}, "", false, err
		}
	}
	return deploy.FileMapping{
		SourcePath:     sourcePath,
		RestorePath:    restorePath,
		TargetRoot:     key.TargetRoot,
		TargetRelative: key.TargetRelative,
		TargetPolicy:   deploy.TargetPolicyPatchExisting,
		Strategy:       deploy.StrategyCopy,
		ModID:          "witcher3-config-xml-merge",
		Catalog:        "dmm-generated",
		Priority:       minWitcherMappingPriority(mappings),
	}, fmt.Sprintf("Witcher 3 config XML %s merged from %d DMM-managed file(s) using %s as the base.", key.TargetRelative, len(mappings), filepath.Base(basePath)), true, nil
}

func witcherConfigXMLTargetPath(input sdk.EventHandlerInput, key xmlMergeKey) (string, error) {
	root := strings.TrimSpace(key.TargetRoot)
	if root == "" || root == "." {
		root = strings.TrimSpace(input.GamePath)
	}
	if root == "" {
		return "", errors.New("game path is required to resolve Witcher 3 config XML target")
	}
	if !filepath.IsAbs(root) {
		return "", fmt.Errorf("Witcher 3 config XML target root %q is not absolute", root)
	}
	return filepath.Join(root, filepath.FromSlash(key.TargetRelative)), nil
}

func witcherConfigXMLSourcePath(input sdk.EventHandlerInput, mapping deploy.FileMapping) string {
	if strings.TrimSpace(mapping.SourcePath) != "" {
		return filepath.Clean(mapping.SourcePath)
	}
	if strings.TrimSpace(input.StagingRoot) == "" || strings.TrimSpace(mapping.SourceRelative) == "" {
		return ""
	}
	return filepath.Join(input.StagingRoot, filepath.FromSlash(filepath.ToSlash(mapping.SourceRelative)))
}

func minWitcherMappingPriority(mappings []deploy.FileMapping) int {
	if len(mappings) == 0 {
		return 0
	}
	min := mappings[0].Priority
	for _, mapping := range mappings[1:] {
		if mapping.Priority < min {
			min = mapping.Priority
		}
	}
	return min
}

func parseXMLNode(data []byte) (*xmlNode, error) {
	text, err := decodeXMLText(data)
	if err != nil {
		return nil, err
	}
	decoder := xml.NewDecoder(strings.NewReader(text))
	decoder.Strict = false
	var stack []*xmlNode
	var root *xmlNode
	for {
		token, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		switch tok := token.(type) {
		case xml.StartElement:
			next := &xmlNode{Name: tok.Name, Attrs: append([]xml.Attr(nil), tok.Attr...)}
			if len(stack) == 0 {
				root = next
			} else {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, next)
			}
			stack = append(stack, next)
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			if len(stack) == 0 {
				continue
			}
			text := strings.TrimSpace(string(tok))
			if text == "" {
				continue
			}
			current := stack[len(stack)-1]
			if current.Text != "" {
				current.Text += " "
			}
			current.Text += text
		}
	}
	if root == nil {
		return nil, errors.New("XML document has no root element")
	}
	return root, nil
}

func decodeXMLText(data []byte) (string, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	var text string
	switch {
	case bytes.HasPrefix(data, []byte{0xFF, 0xFE}):
		decoded, err := decodeUTF16XML(data[2:], binary.LittleEndian)
		if err != nil {
			return "", err
		}
		text = decoded
	case bytes.HasPrefix(data, []byte{0xFE, 0xFF}):
		decoded, err := decodeUTF16XML(data[2:], binary.BigEndian)
		if err != nil {
			return "", err
		}
		text = decoded
	case len(data) >= 4 && data[0] == '<' && data[1] == 0:
		decoded, err := decodeUTF16XML(data, binary.LittleEndian)
		if err != nil {
			return "", err
		}
		text = decoded
	case len(data) >= 4 && data[0] == 0 && data[1] == '<':
		decoded, err := decodeUTF16XML(data, binary.BigEndian)
		if err != nil {
			return "", err
		}
		text = decoded
	default:
		text = string(data)
	}
	return normalizeXMLDeclarationEncoding(text), nil
}

func decodeUTF16XML(data []byte, order binary.ByteOrder) (string, error) {
	if len(data)%2 != 0 {
		return "", errors.New("invalid UTF-16 XML length")
	}
	words := make([]uint16, 0, len(data)/2)
	for len(data) > 0 {
		words = append(words, order.Uint16(data[:2]))
		data = data[2:]
	}
	return string(utf16.Decode(words)), nil
}

func normalizeXMLDeclarationEncoding(text string) string {
	headLen := len(text)
	if headLen > 256 {
		headLen = 256
	}
	head := text[:headLen]
	end := strings.Index(head, "?>")
	if end < 0 {
		return text
	}
	decl := xmlEncodingDeclRE.ReplaceAllString(head[:end], `encoding="UTF-8"`)
	return decl + text[end:]
}

func mergeUserConfigGroups(base, mod *xmlNode) {
	for _, modGroup := range userConfigGroups(mod) {
		groupID := xmlAttrValue(modGroup, "id")
		baseGroup := childByAttr(base, "Group", "id", groupID)
		if baseGroup == nil || groupID == "" {
			base.Children = append(base.Children, cloneXMLNode(modGroup))
			continue
		}
		mergeUserConfigVars(baseGroup, modGroup)
	}
}

func userConfigGroups(root *xmlNode) []*xmlNode {
	var out []*xmlNode
	if root == nil {
		return out
	}
	for _, child := range root.Children {
		if strings.EqualFold(child.Name.Local, "Group") {
			out = append(out, child)
		}
	}
	return out
}

func mergeUserConfigVars(baseGroup, modGroup *xmlNode) {
	modVisible := firstChild(modGroup, "VisibleVars")
	if modVisible == nil {
		return
	}
	baseVisible := firstChild(baseGroup, "VisibleVars")
	if baseVisible == nil {
		baseGroup.Children = append(baseGroup.Children, cloneXMLNode(modVisible))
		return
	}
	for _, modVar := range visibleVars(modVisible) {
		varID := xmlAttrValue(modVar, "id")
		idx := childIndexByAttr(baseVisible.Children, "Var", "id", varID)
		if idx >= 0 && varID != "" {
			baseVisible.Children[idx] = cloneXMLNode(modVar)
			continue
		}
		baseVisible.Children = append(baseVisible.Children, cloneXMLNode(modVar))
	}
}

func visibleVars(visible *xmlNode) []*xmlNode {
	var out []*xmlNode
	if visible == nil {
		return out
	}
	for _, child := range visible.Children {
		if strings.EqualFold(child.Name.Local, "Var") {
			out = append(out, child)
		}
	}
	return out
}

func firstChild(node *xmlNode, name string) *xmlNode {
	if node == nil {
		return nil
	}
	for _, child := range node.Children {
		if strings.EqualFold(child.Name.Local, name) {
			return child
		}
	}
	return nil
}

func childByAttr(node *xmlNode, name, attr, value string) *xmlNode {
	if node == nil {
		return nil
	}
	idx := childIndexByAttr(node.Children, name, attr, value)
	if idx < 0 {
		return nil
	}
	return node.Children[idx]
}

func childIndexByAttr(children []*xmlNode, name, attr, value string) int {
	if value == "" {
		return -1
	}
	for idx, child := range children {
		if strings.EqualFold(child.Name.Local, name) && xmlAttrValue(child, attr) == value {
			return idx
		}
	}
	return -1
}

func xmlAttrValue(node *xmlNode, name string) string {
	if node == nil {
		return ""
	}
	for _, attr := range node.Attrs {
		if strings.EqualFold(attr.Name.Local, name) {
			return attr.Value
		}
	}
	return ""
}

func cloneXMLNode(in *xmlNode) *xmlNode {
	if in == nil {
		return nil
	}
	out := &xmlNode{
		Name:  in.Name,
		Attrs: append([]xml.Attr(nil), in.Attrs...),
		Text:  in.Text,
	}
	for _, child := range in.Children {
		out.Children = append(out.Children, cloneXMLNode(child))
	}
	return out
}

func renderXMLNode(root *xmlNode) []byte {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	encoder := xml.NewEncoder(&buf)
	encoder.Indent("", "\t")
	writeXMLNode(encoder, root)
	_ = encoder.Flush()
	buf.WriteByte('\n')
	return buf.Bytes()
}

func writeXMLNode(encoder *xml.Encoder, node *xmlNode) {
	if node == nil {
		return
	}
	start := xml.StartElement{Name: node.Name, Attr: node.Attrs}
	_ = encoder.EncodeToken(start)
	if node.Text != "" {
		_ = encoder.EncodeToken(xml.CharData([]byte(node.Text)))
	}
	for _, child := range node.Children {
		writeXMLNode(encoder, child)
	}
	_ = encoder.EncodeToken(start.End())
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
	return writeMenuGeneratedFileIn(input, w3MenuMergeGeneratedDir, group, name, contents)
}

func writeMenuGeneratedFileIn(input sdk.EventHandlerInput, rootName, group, name string, contents []byte) (string, error) {
	name = filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(name))))
	if name == "." || name == "" || strings.HasPrefix(name, "../") {
		return "", fmt.Errorf("unsafe Witcher 3 generated file name %q", name)
	}
	root := filepath.Join(input.WorkDir, rootName, group)
	path := filepath.Join(root, filepath.FromSlash(name))
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
