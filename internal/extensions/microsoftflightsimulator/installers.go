package microsoftflightsimulator

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const replacerTargetGroupID = "msfs-replacer-target"

func matchPackArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) {
		return false
	}
	manifestCount := 0
	for _, file := range files {
		if isManifest(file) {
			manifestCount++
		}
	}
	return manifestCount > 1
}

func buildPackArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	manifests := manifestFiles(files)
	if len(manifests) <= 1 {
		return installplan.Plan{}, installplan.Unsupported("MSFS pack installer requires more than one manifest.json, matching Vortex game-microsoftflightsimulator")
	}
	depth := packStripDepth(manifests)
	plan := installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: input.Installer.NameSource,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-msfs-pack",
			Path:   manifests[0],
			Reason: "Vortex MSFS pack installer matched multiple manifest.json files and stripped the common package parent depth",
		}},
	}
	for _, file := range files {
		rel := stripSegments(file, depth)
		if rel == "" {
			continue
		}
		plan.Instructions = append(plan.Instructions, copyInstruction(input, file, rel))
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, installplan.Unsupported("MSFS pack installer matched but produced no deployable files")
	}
	sortPlan(&plan)
	return plan, nil
}

func matchReplacerArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) || len(files) == 0 {
		return false
	}
	for _, file := range files {
		if isManifest(file) || strings.EqualFold(filepath.Base(file), "layout.json") {
			return false
		}
	}
	return true
}

func buildReplacerArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	if len(files) == 0 {
		return installplan.Plan{}, installplan.Unsupported("MSFS replacer archive has no deployable files")
	}
	targetMap, err := officialFileList(input)
	if err != nil {
		return installplan.Plan{}, err
	}
	targetID, ok := selectedReplacerTarget(files, targetMap, input.Selections)
	if !ok {
		return installplan.Plan{}, msfsTargetChoiceRequired(files, targetMap)
	}
	packageName := safePackageName(input.ArchiveName)
	if packageName == "" {
		packageName = safePackageName(filepath.Base(input.ExtractedRoot))
	}
	if packageName == "" {
		return installplan.Plan{}, errors.New("MSFS replacer archive needs a safe generated Community package folder")
	}
	plan := installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: input.Installer.NameSource,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-msfs-replacer",
			Path:   ".",
			Reason: "Vortex MSFS replacer installer matched an archive without manifest.json or layout.json",
		}},
	}
	if targetID == "" {
		plan.Warnings = append(plan.Warnings, "MSFS replacer archive did not match official content; files were packaged as-is like Vortex's warning fallback.")
	}
	for _, file := range files {
		targetRel := file
		if targetID != "" {
			if mapped, ok := targetMap.destinationFor(file, targetID); ok {
				targetRel = mapped
			}
		}
		plan.Instructions = append(plan.Instructions, copyInstruction(input, file, filepath.ToSlash(filepath.Join(packageName, targetRel))))
	}
	layout, err := layoutJSON(input.ExtractedRoot, files)
	if err != nil {
		return installplan.Plan{}, err
	}
	plan.Instructions = append(plan.Instructions, installplan.Instruction{
		Kind:                    installplan.InstructionKindGenerateFromGameFile,
		GeneratedDefaultContent: layout,
		StagingRelative:         filepath.ToSlash(filepath.Join(packageName, "layout.json")),
		TargetRoot:              input.TargetRootID,
		TargetRelative:          filepath.ToSlash(filepath.Join(packageName, "layout.json")),
	})
	sortPlan(&plan)
	return plan, nil
}

func copyInstruction(input installplan.BuildInput, sourceRel, targetRel string) installplan.Instruction {
	return installplan.Instruction{
		Kind:            installplan.InstructionKindCopy,
		SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(sourceRel)),
		StagingRelative: filepath.ToSlash(targetRel),
		TargetRoot:      input.TargetRootID,
		TargetRelative:  filepath.ToSlash(targetRel),
	}
}

func isManifest(file string) bool {
	return strings.EqualFold(filepath.Base(file), "manifest.json")
}

func manifestFiles(files []string) []string {
	out := make([]string, 0, len(files))
	for _, file := range files {
		if isManifest(file) {
			out = append(out, file)
		}
	}
	sort.Strings(out)
	return out
}

func packStripDepth(manifests []string) int {
	depth := 1000
	for _, manifest := range manifests {
		segments := strings.Split(filepath.ToSlash(strings.Trim(manifest, "/")), "/")
		if len(segments) < depth {
			depth = len(segments)
		}
	}
	depth -= 2
	if depth < 0 {
		return 0
	}
	return depth
}

func stripSegments(file string, depth int) string {
	segments := strings.Split(filepath.ToSlash(strings.Trim(file, "/")), "/")
	if depth >= len(segments) {
		return ""
	}
	return strings.Join(segments[depth:], "/")
}

type officialFiles map[string]map[string][]officialTarget

type officialTarget struct {
	Type   string
	ItemID string
	Rel    string
}

func officialFileList(input installplan.BuildInput) (officialFiles, error) {
	packages, _, err := packagesPath(sdkTargetRootInput(input))
	if err != nil {
		return nil, err
	}
	for _, candidate := range []string{
		filepath.Join(packages, "Official", "OneStore"),
		filepath.Join(packages, "Official", "Steam"),
	} {
		files, err := scanOfficialFiles(candidate)
		if err == nil && len(files) > 0 {
			return files, nil
		}
	}
	return nil, errors.New("failed to find MSFS official package file list for replacer archive")
}

func sdkTargetRootInput(input installplan.BuildInput) sdk.TargetRootInput {
	return sdk.TargetRootInput{
		AppID:       input.GameID,
		GamePath:    input.GamePath,
		LibraryPath: input.LibraryPath,
	}
}

func scanOfficialFiles(root string) (officialFiles, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	out := officialFiles{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		parts := strings.Split(entry.Name(), "-")
		if len(parts) < 3 || !strings.EqualFold(parts[0], "asobo") {
			continue
		}
		contentType := parts[1]
		itemID := strings.Join(parts[2:], "-")
		itemRoot := filepath.Join(root, entry.Name())
		err := filepath.WalkDir(itemRoot, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(itemRoot, path)
			if err != nil {
				return err
			}
			base := strings.ToUpper(filepath.Base(path))
			if out[contentType] == nil {
				out[contentType] = map[string][]officialTarget{}
			}
			out[contentType][base] = append(out[contentType][base], officialTarget{
				Type:   contentType,
				ItemID: itemID,
				Rel:    filepath.ToSlash(rel),
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func selectedReplacerTarget(files []string, official officialFiles, selections map[string][]string) (string, bool) {
	targets := possibleReplacerTargets(files, official)
	switch len(targets) {
	case 0:
		return "", true
	case 1:
		return targets[0], true
	default:
		selected := selections[replacerTargetGroupID]
		if len(selected) == 0 {
			return "", false
		}
		allowed := map[string]struct{}{}
		for _, target := range targets {
			allowed[target] = struct{}{}
		}
		for _, target := range selected {
			if _, ok := allowed[target]; ok {
				return target, true
			}
		}
		return "", false
	}
}

func possibleReplacerTargets(files []string, official officialFiles) []string {
	var possible map[string]struct{}
	for _, file := range files {
		targetIDs := official.targetIDsFor(file)
		if len(targetIDs) == 0 {
			continue
		}
		if possible == nil {
			possible = targetIDs
			continue
		}
		for target := range possible {
			if _, ok := targetIDs[target]; !ok {
				delete(possible, target)
			}
		}
	}
	out := make([]string, 0, len(possible))
	for target := range possible {
		out = append(out, target)
	}
	sort.Strings(out)
	return out
}

func (files officialFiles) targetIDsFor(file string) map[string]struct{} {
	out := map[string]struct{}{}
	fileID := strings.ToUpper(filepath.Base(file))
	for contentType, byName := range files {
		for _, target := range byName[fileID] {
			out[contentType+":"+target.ItemID] = struct{}{}
		}
	}
	return out
}

func (files officialFiles) destinationFor(file, targetID string) (string, bool) {
	contentType, itemID, ok := strings.Cut(targetID, ":")
	if !ok {
		return "", false
	}
	fileID := strings.ToUpper(filepath.Base(file))
	for _, target := range files[contentType][fileID] {
		if target.ItemID == itemID {
			return target.Rel, true
		}
	}
	return "", false
}

func msfsTargetChoiceRequired(files []string, official officialFiles) error {
	targets := possibleReplacerTargets(files, official)
	options := make([]installplan.ChoiceOption, 0, len(targets))
	defaults := map[string][]string{}
	for idx, target := range targets {
		_, name, _ := strings.Cut(target, ":")
		if name == "" {
			name = target
		}
		if idx == 0 {
			defaults[replacerTargetGroupID] = []string{target}
		}
		options = append(options, installplan.ChoiceOption{
			ID:            target,
			Name:          name,
			Description:   target,
			Type:          "Optional",
			EffectiveType: "Optional",
		})
	}
	return installplan.ChoiceRequired(
		"msfs-replacer-target",
		"MSFS replacer archive matches multiple official content packages; choose the target package like Vortex's Pick target dialog.",
		installplan.ChoiceInstaller{
			Name: "Microsoft Flight Simulator Target",
			Steps: []installplan.ChoiceStep{{
				ID:   "target-selection",
				Name: "Pick target",
				Groups: []installplan.ChoiceGroup{{
					ID:       replacerTargetGroupID,
					Name:     "Official content",
					Type:     "SelectExactlyOne",
					Required: true,
					Plugins:  options,
				}},
			}},
		},
		defaults,
	)
}

func layoutJSON(root string, files []string) (string, error) {
	entries := make([]msfsLayoutEntry, 0, len(files))
	for _, file := range files {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			return "", err
		}
		entries = append(entries, msfsLayoutEntry{
			Path: file,
			Size: info.Size(),
			Date: toWinTimestamp(info.ModTime().UnixNano() / 1_000_000),
		})
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type msfsLayoutEntry struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
	Date int64  `json:"date"`
}

func toWinTimestamp(ms int64) int64 {
	return ms*10000 + 116444736000000000
}

func safePackageName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	for {
		ext := strings.ToLower(filepath.Ext(name))
		if ext == "" {
			break
		}
		switch ext {
		case ".zip", ".7z", ".rar", ".tar", ".gz", ".bz2", ".xz":
			name = strings.TrimSuffix(name, filepath.Ext(name))
		default:
			ext = ""
		}
		if ext == "" {
			break
		}
	}
	var b strings.Builder
	for _, r := range name {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', '\r', '\n', '\t':
			b.WriteRune('-')
		default:
			b.WriteRune(r)
		}
	}
	return strings.Trim(strings.TrimSpace(b.String()), ". ")
}

func sortPlan(plan *installplan.Plan) {
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	sort.SliceStable(plan.DetectedFrom, func(i, j int) bool {
		return plan.DetectedFrom[i].Path < plan.DetectedFrom[j].Path
	})
}
