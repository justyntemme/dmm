package microsoftflightsimulator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const mergedPackageName = "ZZZZ-merged-config"

func willDeploy(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if len(input.Mappings) == 0 {
		return sdk.EventHandlerResult{}, nil
	}
	prefixes := msfsPackagePrefixes(input.Mappings)
	mergedRoot := filepath.Join(input.WorkDir, "msfs", "merged-config")
	if err := os.RemoveAll(mergedRoot); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	var out []deploy.FileMapping
	var aircraft []deploy.FileMapping
	for _, mapping := range input.Mappings {
		next := mapping
		next.TargetRelative = msfsPrefixedTarget(mapping.TargetRelative, prefixes[mapping.InstalledModID])
		if isAircraftConfigTarget(mapping.TargetRelative) {
			aircraft = append(aircraft, mapping)
			continue
		}
		out = append(out, next)
	}
	generated, err := mergedAircraftMappings(mergedRoot, aircraft)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	out = append(out, generated...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority < out[j].Priority
		}
		return out[i].TargetRelative < out[j].TargetRelative
	})
	messages := []string{"MSFS Community package load-order prefixes applied."}
	if len(generated) > 0 {
		messages = append(messages, "MSFS aircraft.cfg files merged into "+mergedPackageName+".")
	}
	return sdk.EventHandlerResult{ReplaceMappings: true, Mappings: out, Messages: messages}, nil
}

func msfsPackagePrefixes(mappings []deploy.FileMapping) map[int64]string {
	type modOrder struct {
		id       int64
		priority int
		name     string
	}
	byID := map[int64]modOrder{}
	for _, mapping := range mappings {
		if mapping.InstalledModID <= 0 {
			continue
		}
		if _, ok := byID[mapping.InstalledModID]; !ok {
			byID[mapping.InstalledModID] = modOrder{id: mapping.InstalledModID, priority: mapping.Priority, name: firstPathSegment(mapping.TargetRelative)}
		}
	}
	ordered := make([]modOrder, 0, len(byID))
	for _, entry := range byID {
		ordered = append(ordered, entry)
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].priority != ordered[j].priority {
			return ordered[i].priority < ordered[j].priority
		}
		if ordered[i].name != ordered[j].name {
			return ordered[i].name < ordered[j].name
		}
		return ordered[i].id < ordered[j].id
	})
	out := map[int64]string{}
	for idx, entry := range ordered {
		out[entry.id] = makePrefix(idx) + "-"
	}
	return out
}

func makePrefix(input int) string {
	if input < 0 {
		return "ZZZ"
	}
	res := ""
	rest := input
	for rest > 0 {
		res = string(rune('A'+(rest%25))) + res
		rest /= 25
	}
	for len(res) < 3 {
		res = "A" + res
	}
	return res
}

func msfsPrefixedTarget(targetRelative, prefix string) string {
	target := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(targetRelative))))
	if target == "" || target == "." || strings.HasPrefix(target, "../") {
		return targetRelative
	}
	parts := strings.Split(target, "/")
	if len(parts) == 0 || parts[0] == mergedPackageName || strings.HasPrefix(parts[0], prefix) {
		return target
	}
	if strings.TrimSpace(prefix) == "" {
		prefix = "ZZZ-"
	}
	parts[0] = prefix + parts[0]
	return strings.Join(parts, "/")
}

func firstPathSegment(targetRelative string) string {
	target := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(targetRelative))))
	if target == "" || target == "." || strings.HasPrefix(target, "../") {
		return ""
	}
	return strings.Split(target, "/")[0]
}

func isAircraftConfigTarget(targetRelative string) bool {
	return strings.EqualFold(filepath.Base(filepath.FromSlash(targetRelative)), "aircraft.cfg")
}

func mergedAircraftMappings(root string, aircraft []deploy.FileMapping) ([]deploy.FileMapping, error) {
	if len(aircraft) == 0 {
		return nil, nil
	}
	sort.SliceStable(aircraft, func(i, j int) bool {
		if aircraft[i].Priority != aircraft[j].Priority {
			return aircraft[i].Priority < aircraft[j].Priority
		}
		return aircraft[i].TargetRelative < aircraft[j].TargetRelative
	})
	merged := map[string]iniFile{}
	locPakPaths := map[string]string{}
	for _, mapping := range aircraft {
		rel := aircraftMergeRelative(mapping.TargetRelative)
		if rel == "" {
			continue
		}
		incoming, err := readINIFile(mapping.SourcePath)
		if err != nil {
			return nil, err
		}
		locID := strconv.FormatInt(mapping.InstalledModID, 36)
		for locPakName, locPakPath := range mergeLocalizations(root, packageRootForAircraft(mapping.SourcePath), localizationTokens(incoming), locID) {
			locPakPaths[locPakName] = locPakPath
		}
		current := merged[rel]
		if current.Sections == nil {
			current = iniFile{Order: []string{}, Sections: map[string]map[string]string{}}
		}
		merged[rel] = mergeAircraftINI(current, incoming, locID)
	}
	var mappings []deploy.FileMapping
	var layout []msfsLayoutEntry
	for rel, ini := range merged {
		source := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
			return nil, err
		}
		if err := os.WriteFile(source, []byte(ini.String()), 0o600); err != nil {
			return nil, err
		}
		mapping, entry, err := generatedMapping(source, filepath.ToSlash(filepath.Join(mergedPackageName, rel)), -1000)
		if err != nil {
			return nil, err
		}
		mappings = append(mappings, mapping)
		layout = append(layout, entry)
	}
	for locPakName, locPakPath := range locPakPaths {
		mapping, entry, err := generatedMapping(locPakPath, filepath.ToSlash(filepath.Join(mergedPackageName, locPakName)), -1000)
		if err != nil {
			return nil, err
		}
		mappings = append(mappings, mapping)
		layout = append(layout, entry)
	}
	layoutPath := filepath.Join(root, "layout.json")
	body, err := json.MarshalIndent(msfsLayout{Content: layout}, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(layoutPath, body, 0o600); err != nil {
		return nil, err
	}
	mapping, _, err := generatedMapping(layoutPath, mergedPackageName+"/layout.json", -1001)
	if err != nil {
		return nil, err
	}
	mappings = append(mappings, mapping)
	sort.SliceStable(mappings, func(i, j int) bool {
		return mappings[i].TargetRelative < mappings[j].TargetRelative
	})
	return mappings, nil
}

func aircraftMergeRelative(targetRelative string) string {
	target := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(targetRelative))))
	parts := strings.Split(target, "/")
	if len(parts) <= 1 || strings.HasPrefix(target, "../") {
		return ""
	}
	return strings.Join(parts[1:], "/")
}

func generatedMapping(source, target string, priority int) (deploy.FileMapping, msfsLayoutEntry, error) {
	info, err := os.Stat(source)
	if err != nil {
		return deploy.FileMapping{}, msfsLayoutEntry{}, err
	}
	sum, err := fileSHA256(source)
	if err != nil {
		return deploy.FileMapping{}, msfsLayoutEntry{}, err
	}
	return deploy.FileMapping{
		SourcePath:     source,
		TargetRelative: target,
		Strategy:       deploy.StrategyCopy,
		Priority:       priority,
		ChecksumSHA256: sum,
	}, msfsLayoutEntry{Path: strings.TrimPrefix(target, mergedPackageName+"/"), Size: info.Size(), Date: toWinTimestamp(info.ModTime().UnixNano() / int64(time.Millisecond))}, nil
}

func fileSHA256(path string) (string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}

type msfsLayout struct {
	Content []msfsLayoutEntry `json:"content"`
}

type iniFile struct {
	Order    []string
	Sections map[string]map[string]string
}

func readINIFile(path string) (iniFile, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return iniFile{}, err
	}
	out := iniFile{Sections: map[string]map[string]string{}}
	section := ""
	for _, line := range strings.Split(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") && strings.Contains(trimmed, "]") {
			section = strings.ToUpper(strings.TrimSpace(trimmed[1:strings.Index(trimmed, "]")]))
			if _, ok := out.Sections[section]; !ok {
				out.Order = append(out.Order, section)
				out.Sections[section] = map[string]string{}
			}
			continue
		}
		key, value, ok := strings.Cut(trimmed, "=")
		if !ok || section == "" {
			continue
		}
		out.Sections[section][strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return out, nil
}

func packageRootForAircraft(path string) string {
	root := filepath.Dir(path)
	for i := 0; i < 3; i++ {
		next := filepath.Dir(root)
		if next == root {
			break
		}
		root = next
	}
	return root
}

func localizationTokens(ini iniFile) []string {
	seen := map[string]struct{}{}
	var out []string
	for section, values := range ini.Sections {
		if !strings.HasPrefix(section, "FLTSIM.") {
			continue
		}
		for _, key := range []string{"description", "ui_manufacturer", "ui_type", "ui_variation"} {
			token := ttToken(values[key])
			if token == "" {
				continue
			}
			if _, ok := seen[token]; ok {
				continue
			}
			seen[token] = struct{}{}
			out = append(out, token)
		}
	}
	sort.Strings(out)
	return out
}

func mergeLocalizations(outputRoot, packageRoot string, tokens []string, locID string) map[string]string {
	if len(tokens) == 0 {
		return nil
	}
	entries, err := os.ReadDir(packageRoot)
	if err != nil {
		return nil
	}
	out := map[string]string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".locPak") {
			continue
		}
		inPath := filepath.Join(packageRoot, entry.Name())
		in, ok := readLocPak(inPath)
		if !ok {
			continue
		}
		outPath := filepath.Join(outputRoot, entry.Name())
		merged := locPak{LocalisationPackage: locPakBody{Language: in.LocalisationPackage.Language, Strings: map[string]string{}}}
		if existing, ok := readLocPak(outPath); ok {
			merged = existing
			if merged.LocalisationPackage.Strings == nil {
				merged.LocalisationPackage.Strings = map[string]string{}
			}
		}
		for _, token := range tokens {
			if value, ok := in.LocalisationPackage.Strings[token]; ok {
				merged.LocalisationPackage.Strings[token+"."+locID] = value
			}
		}
		if len(merged.LocalisationPackage.Strings) == 0 {
			continue
		}
		body, err := json.MarshalIndent(merged, "", "  ")
		if err != nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(outPath), 0o700); err != nil {
			continue
		}
		if err := os.WriteFile(outPath, body, 0o600); err != nil {
			continue
		}
		out[entry.Name()] = outPath
	}
	return out
}

type locPak struct {
	LocalisationPackage locPakBody `json:"LocalisationPackage"`
}

type locPakBody struct {
	Language string            `json:"Language"`
	Strings  map[string]string `json:"Strings"`
}

func readLocPak(path string) (locPak, bool) {
	body, err := os.ReadFile(path)
	if err != nil {
		return locPak{}, false
	}
	var parsed locPak
	if err := json.Unmarshal(body, &parsed); err != nil {
		return locPak{}, false
	}
	return parsed, true
}

func mergeAircraftINI(existing, incoming iniFile, locID string) iniFile {
	next := existing.clone()
	offset := next.fltsimCount()
	for _, section := range incoming.Order {
		data := incoming.Sections[section]
		if strings.HasPrefix(section, "FLTSIM.") {
			if section == "FLTSIM.0" && next.hasFLTSIM(data) {
				continue
			}
			name := "FLTSIM." + strconv.Itoa(offset)
			offset++
			next.addSection(name, renameLocalizationKeys(data, locID))
			continue
		}
		next.addSection(section, data)
	}
	return next
}

func (ini iniFile) clone() iniFile {
	out := iniFile{Order: append([]string(nil), ini.Order...), Sections: map[string]map[string]string{}}
	for section, values := range ini.Sections {
		out.Sections[section] = copyStringMap(values)
	}
	return out
}

func (ini iniFile) fltsimCount() int {
	count := 0
	for _, section := range ini.Order {
		if strings.HasPrefix(section, "FLTSIM.") {
			count++
		}
	}
	return count
}

func (ini iniFile) hasFLTSIM(values map[string]string) bool {
	for section, current := range ini.Sections {
		if strings.HasPrefix(section, "FLTSIM.") && mapsEqual(current, values) {
			return true
		}
	}
	return false
}

func (ini *iniFile) addSection(section string, values map[string]string) {
	if _, ok := ini.Sections[section]; !ok {
		ini.Order = append(ini.Order, section)
		ini.Sections[section] = map[string]string{}
	}
	for key, value := range values {
		ini.Sections[section][key] = value
	}
}

func (ini iniFile) String() string {
	var body strings.Builder
	for _, section := range ini.Order {
		body.WriteString("[")
		body.WriteString(section)
		body.WriteString("]\n")
		keys := make([]string, 0, len(ini.Sections[section]))
		for key := range ini.Sections[section] {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			body.WriteString(key)
			body.WriteString("=")
			body.WriteString(ini.Sections[section][key])
			body.WriteByte('\n')
		}
	}
	return body.String()
}

func renameLocalizationKeys(values map[string]string, locID string) map[string]string {
	out := copyStringMap(values)
	for _, key := range []string{"description", "ui_manufacturer", "ui_type", "ui_variation"} {
		value := out[key]
		token := ttToken(value)
		if token == "" {
			continue
		}
		out[key] = strings.Replace(value, "TT:"+token, "TT:"+token+"."+locID, 1)
	}
	out["vortex_merged"] = locID
	return out
}

func ttToken(value string) string {
	idx := strings.Index(value, "TT:")
	if idx < 0 {
		return ""
	}
	start := idx + len("TT:")
	end := start
	for end < len(value) {
		r := value[end]
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_' || r == '.' {
			end++
			continue
		}
		break
	}
	return strings.TrimSpace(value[start:end])
}

func copyStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func mapsEqual(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}
