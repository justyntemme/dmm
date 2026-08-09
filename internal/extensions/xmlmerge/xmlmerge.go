package xmlmerge

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

type Options struct {
	Extensions []string
	Message    string
}

func WillDeploy(options Options) sdk.EventHandlerFunc {
	extensions := extensionSet(options.Extensions)
	message := strings.TrimSpace(options.Message)
	if message == "" {
		message = "XML merge generated deployment files."
	}
	return func(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
		if err := ctx.Err(); err != nil {
			return sdk.EventHandlerResult{}, err
		}
		kept, groups := splitMergeMappings(input.Mappings, extensions)
		if len(groups) == 0 {
			return sdk.EventHandlerResult{Messages: []string{"XML merge skipped because no matching enabled files were mapped."}}, nil
		}
		generated := make([]deploy.FileMapping, 0, len(groups))
		keys := make([]mergeKey, 0, len(groups))
		for key := range groups {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool {
			if keys[i].targetRoot != keys[j].targetRoot {
				return keys[i].targetRoot < keys[j].targetRoot
			}
			return keys[i].targetRelative < keys[j].targetRelative
		})
		for idx, key := range keys {
			mapping, err := mergeGroup(ctx, input, key, groups[key], idx)
			if err != nil {
				return sdk.EventHandlerResult{}, err
			}
			generated = append(generated, mapping)
			input.ReportProgress("Merged "+key.targetRelative, idx+1, len(keys))
		}
		out := append(kept, generated...)
		return sdk.EventHandlerResult{
			ReplaceMappings: true,
			Mappings:        out,
			Messages:        []string{message},
		}, nil
	}
}

type mergeKey struct {
	targetRoot     string
	targetRelative string
}

func splitMergeMappings(mappings []deploy.FileMapping, extensions map[string]struct{}) ([]deploy.FileMapping, map[mergeKey][]deploy.FileMapping) {
	groups := map[mergeKey][]deploy.FileMapping{}
	kept := make([]deploy.FileMapping, 0, len(mappings))
	for _, mapping := range mappings {
		ext := strings.ToLower(filepath.Ext(mapping.TargetRelative))
		if _, ok := extensions[ext]; !ok {
			kept = append(kept, mapping)
			continue
		}
		key := mergeKey{
			targetRoot:     filepath.Clean(strings.TrimSpace(mapping.TargetRoot)),
			targetRelative: filepath.ToSlash(pathClean(mapping.TargetRelative)),
		}
		if key.targetRelative == "." || strings.TrimSpace(key.targetRelative) == "" {
			kept = append(kept, mapping)
			continue
		}
		groups[key] = append(groups[key], mapping)
	}
	return kept, groups
}

func mergeGroup(ctx context.Context, input sdk.EventHandlerInput, key mergeKey, mappings []deploy.FileMapping, idx int) (deploy.FileMapping, error) {
	if err := ctx.Err(); err != nil {
		return deploy.FileMapping{}, err
	}
	sort.SliceStable(mappings, func(i, j int) bool {
		if mappings[i].Priority != mappings[j].Priority {
			return mappings[i].Priority > mappings[j].Priority
		}
		if mappings[i].InstalledModID != mappings[j].InstalledModID {
			return mappings[i].InstalledModID > mappings[j].InstalledModID
		}
		return mappings[i].SourcePath < mappings[j].SourcePath
	})
	targetPath, err := targetPath(input, key)
	if err != nil {
		return deploy.FileMapping{}, err
	}
	var merged *node
	var restorePath string
	if existing, err := parseFile(targetPath); err == nil {
		merged = existing
		restorePath = filepath.Join(input.WorkDir, "restore", filepath.FromSlash(key.targetRelative))
		if err := copyFile(targetPath, restorePath); err != nil {
			return deploy.FileMapping{}, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return deploy.FileMapping{}, fmt.Errorf("read merge target %s: %w", key.targetRelative, err)
	}
	for _, mapping := range mappings {
		source := strings.TrimSpace(mapping.SourcePath)
		if source == "" {
			return deploy.FileMapping{}, fmt.Errorf("merge mapping for %s has no source path", key.targetRelative)
		}
		modNode, err := parseFile(source)
		if err != nil {
			return deploy.FileMapping{}, fmt.Errorf("parse merge source %s: %w", filepath.Base(source), err)
		}
		if merged == nil {
			merged = modNode
			continue
		}
		merged = mergeNodes(merged, modNode)
	}
	if merged == nil {
		return deploy.FileMapping{}, fmt.Errorf("merge target %s had no mergeable XML inputs", key.targetRelative)
	}
	sourcePath := filepath.Join(input.WorkDir, "merged", fmt.Sprintf("%04d-%s", idx, filepath.Base(key.targetRelative)))
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
		return deploy.FileMapping{}, err
	}
	if err := os.WriteFile(sourcePath, renderDocument(merged), 0o600); err != nil {
		return deploy.FileMapping{}, err
	}
	out := deploy.FileMapping{
		SourcePath:     sourcePath,
		RestorePath:    restorePath,
		TargetRoot:     strings.TrimSpace(key.targetRoot),
		TargetRelative: key.targetRelative,
		TargetPolicy:   deploy.TargetPolicyPatchExisting,
		Strategy:       deploy.StrategyCopy,
		InstalledModID: 0,
		Priority:       minPriority(mappings),
		ChecksumSHA256: "",
		SourceRelative: "",
		Catalog:        "dmm-generated",
		ModID:          "xml-merge",
	}
	if restorePath == "" {
		out.TargetPolicy = ""
	}
	return out, nil
}

func targetPath(input sdk.EventHandlerInput, key mergeKey) (string, error) {
	root := strings.TrimSpace(key.targetRoot)
	if root == "" || root == "." {
		root = strings.TrimSpace(input.GamePath)
	}
	if root == "" {
		return "", errors.New("game path is required for XML merge target resolution")
	}
	if !filepath.IsAbs(root) {
		return "", fmt.Errorf("merge target root %q is not absolute", root)
	}
	return filepath.Join(root, filepath.FromSlash(key.targetRelative)), nil
}

func minPriority(mappings []deploy.FileMapping) int {
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

type node struct {
	Name     xml.Name
	Attrs    []xml.Attr
	Text     string
	Children []*node
}

func parseFile(path string) (*node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if bytes.HasPrefix(data, []byte("CryXml")) {
		return nil, errors.New("CryXml binary XML is not mergeable")
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = false
	var stack []*node
	var root *node
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
			next := &node{Name: tok.Name, Attrs: append([]xml.Attr(nil), tok.Attr...)}
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
			text := strings.TrimSpace(string(tok))
			if text != "" && len(stack) > 0 {
				current := stack[len(stack)-1]
				if current.Text != "" {
					current.Text += " "
				}
				current.Text += text
			}
		}
	}
	if root == nil {
		return nil, errors.New("XML document has no root element")
	}
	return root, nil
}

func mergeNodes(base, mod *node) *node {
	if base == nil {
		return cloneNode(mod)
	}
	if mod == nil {
		return cloneNode(base)
	}
	if base.Name.Local != mod.Name.Local {
		return cloneNode(mod)
	}
	out := cloneNode(base)
	out.Attrs = mergeAttrs(out.Attrs, mod.Attrs)
	if strings.TrimSpace(mod.Text) != "" {
		out.Text = mod.Text
	}
	for _, modChild := range mod.Children {
		matchIdx := matchingChild(out.Children, modChild)
		if matchIdx < 0 {
			out.Children = append(out.Children, cloneNode(modChild))
			continue
		}
		out.Children[matchIdx] = mergeNodes(out.Children[matchIdx], modChild)
	}
	return out
}

func matchingChild(children []*node, modChild *node) int {
	modKey := attrKey(modChild.Attrs)
	for idx, child := range children {
		if child.Name.Local == modChild.Name.Local && attrKey(child.Attrs) == modKey {
			return idx
		}
	}
	if modKey != "" {
		return -1
	}
	for idx, child := range children {
		if child.Name.Local == modChild.Name.Local && attrKey(child.Attrs) == "" {
			return idx
		}
	}
	return -1
}

func mergeAttrs(base, mod []xml.Attr) []xml.Attr {
	out := append([]xml.Attr(nil), base...)
	index := map[string]int{}
	for idx, attr := range out {
		index[attr.Name.Space+"\x00"+attr.Name.Local] = idx
	}
	for _, attr := range mod {
		key := attr.Name.Space + "\x00" + attr.Name.Local
		if idx, ok := index[key]; ok {
			out[idx] = attr
			continue
		}
		index[key] = len(out)
		out = append(out, attr)
	}
	return out
}

func attrKey(attrs []xml.Attr) string {
	if len(attrs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(attrs))
	for _, attr := range attrs {
		parts = append(parts, strings.ToLower(attr.Name.Local)+"="+attr.Value)
	}
	sort.Strings(parts)
	return strings.Join(parts, "\x00")
}

func cloneNode(in *node) *node {
	if in == nil {
		return nil
	}
	out := &node{
		Name:  in.Name,
		Attrs: append([]xml.Attr(nil), in.Attrs...),
		Text:  in.Text,
	}
	for _, child := range in.Children {
		out.Children = append(out.Children, cloneNode(child))
	}
	return out
}

func renderDocument(root *node) []byte {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	encoder := xml.NewEncoder(&buf)
	encoder.Indent("", "  ")
	writeNode(encoder, root)
	_ = encoder.Flush()
	buf.WriteByte('\n')
	return buf.Bytes()
}

func writeNode(encoder *xml.Encoder, item *node) {
	if item == nil {
		return
	}
	start := xml.StartElement{Name: item.Name, Attr: item.Attrs}
	_ = encoder.EncodeToken(start)
	if item.Text != "" {
		_ = encoder.EncodeToken(xml.CharData([]byte(item.Text)))
	}
	for _, child := range item.Children {
		writeNode(encoder, child)
	}
	_ = encoder.EncodeToken(start.End())
}

func extensionSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		values = []string{".xml"}
	}
	out := map[string]struct{}{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if !strings.HasPrefix(value, ".") {
			value = "." + value
		}
		out[value] = struct{}{}
	}
	return out
}

func pathClean(value string) string {
	parts := strings.Split(filepath.ToSlash(strings.TrimSpace(value)), "/")
	stack := make([]string, 0, len(parts))
	for _, part := range parts {
		switch part {
		case "", ".":
			continue
		case "..":
			if len(stack) == 0 {
				return "../" + strings.Join(parts, "/")
			}
			stack = stack[:len(stack)-1]
		default:
			stack = append(stack, part)
		}
	}
	if len(stack) == 0 {
		return "."
	}
	return strings.Join(stack, "/")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
