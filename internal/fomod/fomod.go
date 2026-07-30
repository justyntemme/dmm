package fomod

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

type Installer struct {
	Name                string               `json:"name"`
	ModuleConfig        string               `json:"module_config"`
	RequiredFiles       []FileEntry          `json:"required_files,omitempty"`
	ConditionalPatterns []ConditionalPattern `json:"conditional_patterns,omitempty"`
	Steps               []Step               `json:"steps"`
}

type Step struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Groups []Group `json:"groups"`
}

type Group struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Plugins []Plugin `json:"plugins"`
}

type Plugin struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Type        string      `json:"type,omitempty"`
	Flags       []Flag      `json:"flags,omitempty"`
	Files       []FileEntry `json:"files,omitempty"`
}

type FileEntry struct {
	Source      string `json:"source"`
	Destination string `json:"destination,omitempty"`
	Priority    int    `json:"priority,omitempty"`
	IsFolder    bool   `json:"is_folder,omitempty"`
}

type Flag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ConditionalPattern struct {
	Dependencies DependencyGroup `json:"dependencies"`
	Files        []FileEntry     `json:"files"`
}

type DependencyGroup struct {
	Operator              string            `json:"operator,omitempty"`
	FlagDependencies      []FlagDependency  `json:"flag_dependencies,omitempty"`
	FileDependencies      []FileDependency  `json:"file_dependencies,omitempty"`
	NestedDependencies    []DependencyGroup `json:"nested_dependencies,omitempty"`
	UnsupportedDependency bool              `json:"unsupported_dependency,omitempty"`
}

type FlagDependency struct {
	Flag  string `json:"flag"`
	Value string `json:"value"`
}

type FileDependency struct {
	File  string `json:"file"`
	State string `json:"state"`
}

type FileStateResolver func(relative string) string

type PlanOptions struct {
	ModType           string
	PlannerID         string
	TargetRoot        string
	StopFolders       []string
	FileStates        map[string]string
	FileStateResolver FileStateResolver
}

type configXML struct {
	ModuleName              string                     `xml:"moduleName"`
	RequiredInstallFiles    fileContainerXML           `xml:"requiredInstallFiles"`
	ConditionalFileInstalls conditionalFileInstallsXML `xml:"conditionalFileInstalls"`
	InstallSteps            installStepsXML            `xml:"installSteps"`
}

type installStepsXML struct {
	Steps []installStepXML `xml:"installStep"`
}

type installStepXML struct {
	Name               string                `xml:"name,attr"`
	OptionalFileGroups optionalFileGroupsXML `xml:"optionalFileGroups"`
}

type optionalFileGroupsXML struct {
	Groups []groupXML `xml:"group"`
}

type groupXML struct {
	Name    string     `xml:"name,attr"`
	Type    string     `xml:"type,attr"`
	Plugins pluginsXML `xml:"plugins"`
}

type pluginsXML struct {
	Plugins []pluginXML `xml:"plugin"`
}

type pluginXML struct {
	Name           string            `xml:"name,attr"`
	Description    string            `xml:"description"`
	ConditionFlags conditionFlagsXML `xml:"conditionFlags"`
	TypeDescriptor typeDescriptorXML `xml:"typeDescriptor"`
	Files          fileContainerXML  `xml:"files"`
}

type typeDescriptorXML struct {
	Type typeXML `xml:"type"`
}

type typeXML struct {
	Name string `xml:"name,attr"`
}

type fileContainerXML struct {
	Files   []fileXML `xml:"file"`
	Folders []fileXML `xml:"folder"`
}

type fileXML struct {
	Source      string `xml:"source,attr"`
	Destination string `xml:"destination,attr"`
	Priority    string `xml:"priority,attr"`
}

type conditionFlagsXML struct {
	Flags []flagXML `xml:"flag"`
}

type flagXML struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

type conditionalFileInstallsXML struct {
	Patterns conditionalPatternsXML `xml:"patterns"`
}

type conditionalPatternsXML struct {
	Patterns []conditionalPatternXML `xml:"pattern"`
}

type conditionalPatternXML struct {
	Dependencies dependenciesXML  `xml:"dependencies"`
	Files        fileContainerXML `xml:"files"`
}

type dependenciesXML struct {
	Operator         string              `xml:"operator,attr"`
	FlagDependencies []flagDependencyXML `xml:"flagDependency"`
	FileDependencies []fileDependencyXML `xml:"fileDependency"`
	GameDependencies []gameDependencyXML `xml:"gameDependency"`
	Dependencies     []dependenciesXML   `xml:"dependencies"`
}

type flagDependencyXML struct {
	Flag  string `xml:"flag,attr"`
	Value string `xml:"value,attr"`
}

type fileDependencyXML struct {
	File  string `xml:"file,attr"`
	State string `xml:"state,attr"`
}

type gameDependencyXML struct {
	Version string `xml:"version,attr"`
}

func Parse(root string) (Installer, error) {
	moduleConfig, err := findModuleConfig(root)
	if err != nil {
		return Installer{}, err
	}
	data, err := os.ReadFile(moduleConfig)
	if err != nil {
		return Installer{}, err
	}
	text, err := decodeXML(data)
	if err != nil {
		return Installer{}, err
	}
	var cfg configXML
	decoder := xml.NewDecoder(strings.NewReader(text))
	if err := decoder.Decode(&cfg); err != nil {
		return Installer{}, err
	}
	rel, err := filepath.Rel(root, moduleConfig)
	if err != nil {
		return Installer{}, err
	}
	installer := Installer{
		Name:                strings.TrimSpace(cfg.ModuleName),
		ModuleConfig:        filepath.ToSlash(rel),
		RequiredFiles:       cfg.RequiredInstallFiles.entries(),
		ConditionalPatterns: cfg.ConditionalFileInstalls.patterns(),
		Steps:               []Step{},
	}
	if installer.Name == "" {
		installer.Name = "FOMOD installer"
	}
	for stepIndex, stepXML := range cfg.InstallSteps.Steps {
		step := Step{
			ID:     fmt.Sprintf("step-%d", stepIndex+1),
			Name:   strings.TrimSpace(stepXML.Name),
			Groups: []Group{},
		}
		if step.Name == "" {
			step.Name = fmt.Sprintf("Step %d", stepIndex+1)
		}
		for groupIndex, groupXML := range stepXML.OptionalFileGroups.Groups {
			group := Group{
				ID:      fmt.Sprintf("%s-group-%d", step.ID, groupIndex+1),
				Name:    strings.TrimSpace(groupXML.Name),
				Type:    normalizeGroupType(groupXML.Type),
				Plugins: []Plugin{},
			}
			if group.Name == "" {
				group.Name = fmt.Sprintf("Group %d", groupIndex+1)
			}
			for pluginIndex, pluginXML := range groupXML.Plugins.Plugins {
				name := strings.TrimSpace(pluginXML.Name)
				if name == "" {
					name = fmt.Sprintf("Option %d", pluginIndex+1)
				}
				group.Plugins = append(group.Plugins, Plugin{
					ID:          fmt.Sprintf("%s-plugin-%d", group.ID, pluginIndex+1),
					Name:        name,
					Description: strings.TrimSpace(pluginXML.Description),
					Type:        strings.TrimSpace(pluginXML.TypeDescriptor.Type.Name),
					Flags:       pluginXML.ConditionFlags.flags(),
					Files:       pluginXML.Files.entries(),
				})
			}
			step.Groups = append(step.Groups, group)
		}
		installer.Steps = append(installer.Steps, step)
	}
	return installer, nil
}

func DefaultSelections(installer Installer) map[string][]string {
	out := map[string][]string{}
	for _, step := range installer.Steps {
		for _, group := range step.Groups {
			switch strings.ToLower(group.Type) {
			case "selectall":
				for _, plugin := range group.Plugins {
					out[group.ID] = append(out[group.ID], plugin.ID)
				}
			case "selectexactlyone", "selectatleastone":
				if plugin, ok := preferredPlugin(group.Plugins); ok {
					out[group.ID] = []string{plugin.ID}
				}
			case "selectatmostone", "selectany":
				for _, plugin := range group.Plugins {
					if isPreferredPluginType(plugin.Type) {
						out[group.ID] = append(out[group.ID], plugin.ID)
					}
				}
			default:
				if plugin, ok := preferredPlugin(group.Plugins); ok {
					out[group.ID] = []string{plugin.ID}
				}
			}
		}
	}
	return out
}

func BuildPlan(gameID, root string, installer Installer, selections map[string][]string, options PlanOptions) (installplan.Plan, error) {
	if strings.TrimSpace(gameID) == "" {
		return installplan.Plan{}, errors.New("game id is required")
	}
	if strings.TrimSpace(root) == "" {
		return installplan.Plan{}, errors.New("extracted root is required")
	}
	modType := strings.TrimSpace(options.ModType)
	if modType == "" {
		return installplan.Plan{}, errors.New("FOMOD plan mod type is required")
	}
	plannerID := strings.TrimSpace(options.PlannerID)
	if plannerID == "" {
		plannerID = "fomod"
	}
	targetRoot, err := cleanOptionalRoot(options.TargetRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	if selections == nil {
		selections = DefaultSelections(installer)
	}
	plan := installplan.Plan{
		GameID:       strings.TrimSpace(gameID),
		ModType:      modType,
		PlannerID:    plannerID,
		NameSource:   installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{Kind: "fomod-module-config", Path: installer.ModuleConfig, Reason: "FOMOD ModuleConfig.xml parsed"}},
		Instructions: []installplan.Instruction{},
	}
	stopFolders := normalizedStopFolders(options.StopFolders)
	for _, entry := range installer.RequiredFiles {
		if err := appendEntryInstructions(&plan, root, targetRoot, stopFolders, entry); err != nil {
			return installplan.Plan{}, err
		}
	}
	selectedFlags := map[string]string{}
	for _, step := range installer.Steps {
		for _, group := range step.Groups {
			selected, err := selectedPlugins(group, selections[group.ID])
			if err != nil {
				return installplan.Plan{}, err
			}
			for _, plugin := range selected {
				for _, flag := range plugin.Flags {
					name := strings.TrimSpace(flag.Name)
					if name != "" {
						selectedFlags[name] = strings.TrimSpace(flag.Value)
					}
				}
				for _, entry := range plugin.Files {
					if err := appendEntryInstructions(&plan, root, targetRoot, stopFolders, entry); err != nil {
						return installplan.Plan{}, err
					}
				}
			}
		}
	}
	for _, pattern := range installer.ConditionalPatterns {
		if !dependencyGroupMatches(pattern.Dependencies, selectedFlags, options.FileStates, options.FileStateResolver) {
			continue
		}
		for _, entry := range pattern.Files {
			if err := appendEntryInstructions(&plan, root, targetRoot, stopFolders, entry); err != nil {
				return installplan.Plan{}, err
			}
		}
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func selectedPlugins(group Group, ids []string) ([]Plugin, error) {
	selected := map[string]struct{}{}
	for _, id := range ids {
		selected[strings.TrimSpace(id)] = struct{}{}
	}
	var out []Plugin
	for _, plugin := range group.Plugins {
		if _, ok := selected[plugin.ID]; ok {
			out = append(out, plugin)
		}
	}
	switch strings.ToLower(group.Type) {
	case "selectall":
		if len(out) != len(group.Plugins) {
			return nil, fmt.Errorf("group %q requires all options", group.Name)
		}
	case "selectexactlyone":
		if len(out) != 1 {
			return nil, fmt.Errorf("group %q requires exactly one option", group.Name)
		}
	case "selectatleastone":
		if len(out) < 1 {
			return nil, fmt.Errorf("group %q requires at least one option", group.Name)
		}
	case "selectatmostone":
		if len(out) > 1 {
			return nil, fmt.Errorf("group %q allows at most one option", group.Name)
		}
	}
	return out, nil
}

func appendEntryInstructions(plan *installplan.Plan, root string, targetRoot string, stopFolders []string, entry FileEntry) error {
	sourceRel, err := cleanRel(entry.Source)
	if err != nil {
		return err
	}
	sourcePath := filepath.Join(root, filepath.FromSlash(sourceRel))
	info, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}
	destRel := strings.TrimSpace(filepath.ToSlash(entry.Destination))
	if destRel == "" {
		destRel = sourceRel
	}
	destRel = stripBeforeStopFolder(destRel, stopFolders)
	destRel, err = cleanRel(destRel)
	if err != nil {
		return err
	}
	if info.IsDir() || entry.IsFolder {
		return filepath.WalkDir(sourcePath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(sourcePath, path)
			if err != nil {
				return err
			}
			targetRel := filepath.ToSlash(filepath.Join(destRel, rel))
			plan.Instructions = append(plan.Instructions, installplan.Instruction{
				Kind:            installplan.InstructionKindCopy,
				SourcePath:      path,
				StagingRelative: targetRel,
				TargetRelative:  targetRelative(targetRoot, targetRel),
			})
			return nil
		})
	}
	plan.Instructions = append(plan.Instructions, installplan.Instruction{
		Kind:            installplan.InstructionKindCopy,
		SourcePath:      sourcePath,
		StagingRelative: destRel,
		TargetRelative:  targetRelative(targetRoot, destRel),
	})
	return nil
}

func normalizedStopFolders(values []string) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(filepath.ToSlash(value))
		if value == "" || strings.Contains(value, "/") {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func stripBeforeStopFolder(rel string, stopFolders []string) string {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" || len(stopFolders) == 0 {
		return rel
	}
	stop := map[string]struct{}{}
	for _, folder := range stopFolders {
		stop[strings.ToLower(strings.TrimSpace(folder))] = struct{}{}
	}
	segments := strings.Split(rel, "/")
	for index, segment := range segments {
		if _, ok := stop[strings.ToLower(strings.TrimSpace(segment))]; !ok {
			continue
		}
		return filepath.ToSlash(filepath.Join(segments[index:]...))
	}
	return rel
}

func targetRelative(targetRoot, destRel string) string {
	targetRoot = strings.TrimSpace(filepath.ToSlash(targetRoot))
	destRel = strings.TrimSpace(filepath.ToSlash(destRel))
	if targetRoot == "" || pathHasRoot(destRel, targetRoot) {
		return destRel
	}
	return filepath.ToSlash(filepath.Join(targetRoot, destRel))
}

func pathHasRoot(rel, root string) bool {
	rel = strings.Trim(filepath.ToSlash(rel), "/")
	root = strings.Trim(filepath.ToSlash(root), "/")
	if root == "" {
		return false
	}
	if strings.EqualFold(rel, root) {
		return true
	}
	return strings.HasPrefix(strings.ToLower(rel), strings.ToLower(root)+"/")
}

func findModuleConfig(root string) (string, error) {
	var found string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || found != "" || d.IsDir() {
			return err
		}
		if !strings.EqualFold(filepath.Base(path), "ModuleConfig.xml") {
			return nil
		}
		if !strings.EqualFold(filepath.Base(filepath.Dir(path)), "fomod") {
			return nil
		}
		found = path
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", errors.New("fomod/ModuleConfig.xml was not found")
	}
	return found, nil
}

func (c fileContainerXML) entries() []FileEntry {
	var out []FileEntry
	for _, file := range c.Files {
		out = append(out, file.entry(false))
	}
	for _, folder := range c.Folders {
		out = append(out, folder.entry(true))
	}
	return out
}

func (c conditionFlagsXML) flags() []Flag {
	var out []Flag
	for _, flag := range c.Flags {
		name := strings.TrimSpace(flag.Name)
		if name == "" {
			continue
		}
		out = append(out, Flag{
			Name:  name,
			Value: strings.TrimSpace(flag.Value),
		})
	}
	return out
}

func (c conditionalFileInstallsXML) patterns() []ConditionalPattern {
	var out []ConditionalPattern
	for _, pattern := range c.Patterns.Patterns {
		files := pattern.Files.entries()
		if len(files) == 0 {
			continue
		}
		out = append(out, ConditionalPattern{
			Dependencies: pattern.Dependencies.group(),
			Files:        files,
		})
	}
	return out
}

func (d dependenciesXML) group() DependencyGroup {
	group := DependencyGroup{
		Operator:              normalizeDependencyOperator(d.Operator),
		UnsupportedDependency: len(d.GameDependencies) > 0,
	}
	for _, dependency := range d.FlagDependencies {
		flag := strings.TrimSpace(dependency.Flag)
		if flag == "" {
			continue
		}
		group.FlagDependencies = append(group.FlagDependencies, FlagDependency{
			Flag:  flag,
			Value: strings.TrimSpace(dependency.Value),
		})
	}
	for _, nested := range d.Dependencies {
		group.NestedDependencies = append(group.NestedDependencies, nested.group())
	}
	for _, dependency := range d.FileDependencies {
		file := strings.TrimSpace(filepath.ToSlash(dependency.File))
		if file == "" {
			continue
		}
		group.FileDependencies = append(group.FileDependencies, FileDependency{
			File:  file,
			State: normalizeDependencyState(dependency.State),
		})
	}
	return group
}

func (f fileXML) entry(isFolder bool) FileEntry {
	priority, _ := strconv.Atoi(strings.TrimSpace(f.Priority))
	return FileEntry{
		Source:      strings.TrimSpace(filepath.ToSlash(f.Source)),
		Destination: strings.TrimSpace(filepath.ToSlash(f.Destination)),
		Priority:    priority,
		IsFolder:    isFolder,
	}
}

func dependencyGroupMatches(group DependencyGroup, flags map[string]string, fileStates map[string]string, fileStateResolver FileStateResolver) bool {
	if group.UnsupportedDependency {
		return false
	}
	var results []bool
	for _, dependency := range group.FlagDependencies {
		results = append(results, flagDependencyMatches(dependency, flags))
	}
	for _, dependency := range group.FileDependencies {
		results = append(results, fileDependencyMatches(dependency, fileStates, fileStateResolver))
	}
	for _, nested := range group.NestedDependencies {
		results = append(results, dependencyGroupMatches(nested, flags, fileStates, fileStateResolver))
	}
	if len(results) == 0 {
		return true
	}
	switch normalizeDependencyOperator(group.Operator) {
	case "or":
		for _, result := range results {
			if result {
				return true
			}
		}
		return false
	default:
		for _, result := range results {
			if !result {
				return false
			}
		}
		return true
	}
}

func flagDependencyMatches(dependency FlagDependency, flags map[string]string) bool {
	flag := strings.TrimSpace(dependency.Flag)
	if flag == "" {
		return false
	}
	want := strings.TrimSpace(dependency.Value)
	got, ok := flags[flag]
	return ok && strings.EqualFold(strings.TrimSpace(got), want)
}

func fileDependencyMatches(dependency FileDependency, fileStates map[string]string, fileStateResolver FileStateResolver) bool {
	file := strings.TrimSpace(filepath.ToSlash(dependency.File))
	if file == "" {
		return false
	}
	state := dependencyStateForFile(file, fileStates, fileStateResolver)
	want := normalizeDependencyState(dependency.State)
	return state == want
}

func dependencyStateForFile(file string, fileStates map[string]string, fileStateResolver FileStateResolver) string {
	key, err := cleanRel(file)
	if err != nil {
		return "missing"
	}
	if state, ok := lookupDependencyFileState(fileStates, key); ok {
		return state
	}
	if fileStateResolver != nil {
		return normalizeDependencyState(fileStateResolver(key))
	}
	return "missing"
}

func lookupDependencyFileState(fileStates map[string]string, file string) (string, bool) {
	if len(fileStates) == 0 {
		return "", false
	}
	candidates := []string{file, filepath.ToSlash(filepath.Clean(filepath.FromSlash(file)))}
	for _, candidate := range candidates {
		if state, ok := fileStates[candidate]; ok {
			return normalizeDependencyState(state), true
		}
		for key, state := range fileStates {
			if strings.EqualFold(filepath.ToSlash(filepath.Clean(filepath.FromSlash(key))), candidate) {
				return normalizeDependencyState(state), true
			}
		}
	}
	return "", false
}

func normalizeDependencyState(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "inactive":
		return "inactive"
	case "missing":
		return "missing"
	default:
		return "active"
	}
}

func normalizeDependencyOperator(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "or":
		return "or"
	default:
		return "and"
	}
}

func preferredPlugin(plugins []Plugin) (Plugin, bool) {
	for _, plugin := range plugins {
		if isPreferredPluginType(plugin.Type) {
			return plugin, true
		}
	}
	if len(plugins) == 0 {
		return Plugin{}, false
	}
	return plugins[0], true
}

func isPreferredPluginType(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "required", "recommended":
		return true
	default:
		return false
	}
}

func normalizeGroupType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "SelectAny"
	}
	return value
}

func cleanRel(value string) (string, error) {
	value = strings.TrimSpace(filepath.ToSlash(value))
	if value == "" {
		return "", errors.New("relative path is required")
	}
	if strings.HasPrefix(value, "/") {
		return "", errors.New("absolute paths are not allowed")
	}
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", errors.New("path traversal is not allowed")
	}
	return cleaned, nil
}

func cleanOptionalRoot(value string) (string, error) {
	value = strings.TrimSpace(filepath.ToSlash(value))
	if value == "" {
		return "", nil
	}
	return cleanRel(value)
}

func decodeXML(data []byte) (string, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	if len(data) >= 2 {
		switch {
		case data[0] == 0xFF && data[1] == 0xFE:
			return decodeUTF16(data[2:], binary.LittleEndian)
		case data[0] == 0xFE && data[1] == 0xFF:
			return decodeUTF16(data[2:], binary.BigEndian)
		}
	}
	return string(data), nil
}

func decodeUTF16(data []byte, order binary.ByteOrder) (string, error) {
	if len(data)%2 != 0 {
		return "", errors.New("invalid UTF-16 XML length")
	}
	words := make([]uint16, len(data)/2)
	for i := range words {
		words[i] = order.Uint16(data[i*2:])
	}
	return string(utf16.Decode(words)), nil
}
