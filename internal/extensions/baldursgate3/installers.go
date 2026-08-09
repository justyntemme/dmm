package baldursgate3

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

var lslibFiles = map[string]struct{}{
	"divine.exe": {},
	"lslib.dll":  {},
}

var originalPakFiles = map[string]struct{}{
	"assets.pak":           {},
	"effects.pak":          {},
	"engine.pak":           {},
	"engineshaders.pak":    {},
	"game.pak":             {},
	"gameplatform.pak":     {},
	"gustav.pak":           {},
	"gustav_textures.pak":  {},
	"icons.pak":            {},
	"lowtex.pak":           {},
	"materials.pak":        {},
	"minimaps.pak":         {},
	"models.pak":           {},
	"shared.pak":           {},
	"sharedsoundbanks.pak": {},
	"sharedsounds.pak":     {},
	"textures.pak":         {},
	"virtualtextures.pak":  {},
}

func registerInstallers(r interface {
	RegisterInstaller(installplan.InstallerSpec)
}) {
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:bg3:lslib-divine",
		VortexInstallerID: "bg3-lslib-divine-tool",
		Priority:          15,
		ModType:           lslibModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchLSLib,
		CustomBuild:       buildLSLib,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:bg3:bg3se",
		VortexInstallerID: "bg3-bg3se",
		Priority:          15,
		ModType:           bg3seModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchBG3SE,
		CustomBuild:       buildBG3SE,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:bg3:engine-injector",
		VortexInstallerID: "bg3-engine-injector",
		Priority:          20,
		ModType:           engineInjectorModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchEngineInjector,
		CustomBuild:       buildEngineInjector,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:bg3:loose-or-replacer",
		VortexInstallerID: "bg3-replacer",
		Priority:          25,
		ModType:           looseModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchLooseOrReplacer,
		CustomBuild:       buildLooseOrReplacer,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:bg3:modfixer",
		VortexInstallerID: "bg3-modfixer",
		Priority:          25,
		ModType:           pakModType,
		NameSource:        installplan.NameSourceArchive,
		TargetRootID:      bg3ModsRootID,
		CustomMatch:       matchModFixer,
		CustomBuild:       buildModFixer,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:bg3:pak",
		VortexInstallerID: "bg3-default-pak",
		Priority:          35,
		ModType:           pakModType,
		NameSource:        installplan.NameSourceArchive,
		TargetRootID:      bg3ModsRootID,
		CustomMatch:       matchPakArchive,
		CustomBuild:       buildPakArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
}

func matchLSLib(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil {
		return false
	}
	count := 0
	for _, file := range files {
		if _, ok := lslibFiles[strings.ToLower(filepath.Base(file))]; ok {
			count++
		}
	}
	return count >= 2
}

func buildLSLib(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	plan := newPlan(input, lslibModType, "vortex-bg3-lslib", "Vortex BG3 LSLib installer matched divine.exe and lslib.dll")
	for _, file := range files {
		if !pathContainsSegment(file, "tools") {
			continue
		}
		if strings.TrimSpace(filepath.Ext(file)) == "" {
			continue
		}
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: filepath.ToSlash(filepath.Join("tools", filepath.Base(file))),
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, installplan.Unsupported("BG3 LSLib archive matched but no tools files were found")
	}
	plan.Metadata = append(plan.Metadata, installplan.ModMetadata{
		Kind:            "tool",
		Name:            "LSLib/Divine Tool",
		UniqueID:        "bg3-lslib-divine",
		Version:         lslibVersionFromArchive(input.ArchiveName),
		StagingRelative: "tools/divine.exe",
	})
	sortPlan(&plan)
	return plan, nil
}

func matchBG3SE(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil {
		return false
	}
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), "DWrite.dll") {
			return true
		}
	}
	return false
}

func buildBG3SE(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	plan := newPlan(input, bg3seModType, "vortex-bg3-bg3se", "Vortex BG3SE installer matched DWrite.dll")
	for _, file := range files {
		if !strings.EqualFold(filepath.Ext(file), ".dll") {
			continue
		}
		plan.Instructions = append(plan.Instructions, copyInstruction(input, file, filepath.Base(file), "bin", ""))
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, installplan.Unsupported("BG3SE archive matched but no dll files were found")
	}
	sortPlan(&plan)
	return plan, nil
}

func matchEngineInjector(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil {
		return false
	}
	for _, file := range files {
		if pathContainsSegment(file, "bin") {
			return true
		}
	}
	return false
}

func buildEngineInjector(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	plan := newPlan(input, engineInjectorModType, "vortex-bg3-engine-injector", "Vortex BG3 engine injector installer matched a bin folder")
	for _, file := range files {
		rel, ok := fromSegment(file, "bin")
		if !ok || strings.TrimSpace(filepath.Ext(file)) == "" {
			continue
		}
		plan.Instructions = append(plan.Instructions, copyInstruction(input, file, rel, "", ""))
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, installplan.Unsupported("BG3 engine injector archive matched but no deployable bin files were found")
	}
	sortPlan(&plan)
	return plan, nil
}

func matchModFixer(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil {
		return false
	}
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), "ModFixer.pak") {
			return true
		}
	}
	return false
}

func buildModFixer(input installplan.BuildInput) (installplan.Plan, error) {
	plan, err := buildPakFiles(input, "vortex-bg3-modfixer", "Vortex BG3 Mod Fixer installer matched ModFixer.pak")
	if err != nil {
		return installplan.Plan{}, err
	}
	plan.Metadata = append(plan.Metadata, installplan.ModMetadata{Kind: "bg3-mod-fixer", Name: "Baldur's Gate 3 Mod Fixer"})
	return plan, nil
}

func matchPakArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) {
		return false
	}
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), ".pak") {
			return true
		}
	}
	return false
}

func buildPakArchive(input installplan.BuildInput) (installplan.Plan, error) {
	return buildPakFiles(input, "vortex-bg3-pak", "BG3 default pak installer staged pak files into the Vortex BG3 Mods path")
}

func buildPakFiles(input installplan.BuildInput, detectionKind, reason string) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	plan := newPlan(input, pakModType, detectionKind, reason)
	for _, file := range files {
		if !strings.EqualFold(filepath.Ext(file), ".pak") {
			continue
		}
		plan.Instructions = append(plan.Instructions, copyInstruction(input, file, filepath.Base(file), "", bg3ModsRootID))
		plan.Metadata = append(plan.Metadata, installplan.ModMetadata{
			Kind:                       "bg3-pak-file",
			SourcePath:                 file,
			StagingRelative:            filepath.Base(file),
			TargetRelative:             filepath.Base(file),
			Name:                       strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)),
			AdditionalLogicalFileNames: []string{filepath.Base(file)},
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, installplan.Unsupported("BG3 pak archive matched but no pak files were found")
	}
	sortPlan(&plan)
	return plan, nil
}

func matchLooseOrReplacer(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) {
		return false
	}
	pakCount := 0
	hasGeneratedOrPublic := false
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), ".pak") {
			pakCount++
		}
		if pathContainsSegment(file, "Generated") || pathContainsSegment(file, "Public") {
			hasGeneratedOrPublic = true
		}
	}
	return hasGeneratedOrPublic || pakCount == 0
}

func buildLooseOrReplacer(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	dataRoot := dataRootForBG3(files)
	modType := looseModType
	plan := newPlan(input, modType, "vortex-bg3-replacer", "Vortex BG3 replacer installer matched generated/public data or a non-pak archive")
	for _, file := range files {
		if strings.HasSuffix(file, "/") {
			continue
		}
		rel := file
		if dataRoot != "" {
			if !simplearchive.PathWithinRoot(file, dataRoot) {
				continue
			}
			rel = simplearchive.StripRoot(file, dataRoot)
		}
		if rel == "" {
			continue
		}
		if isOriginalFile(rel) {
			modType = replacerModType
		}
		plan.Instructions = append(plan.Instructions, copyInstruction(input, file, rel, "Data", ""))
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, installplan.Unsupported("BG3 loose/replacer archive matched but no deployable files were found")
	}
	plan.ModType = modType
	if modType == replacerModType {
		plan.Warnings = append(plan.Warnings, "This archive looks like a BG3 replacer. Vortex warns that replacers may include version-specific copies of game data and can break after game updates.")
	}
	sortPlan(&plan)
	return plan, nil
}

func newPlan(input installplan.BuildInput, modType, detectionKind, reason string) installplan.Plan {
	return installplan.Plan{
		GameID:     input.GameID,
		ModType:    modType,
		PlannerID:  input.Installer.ID,
		NameSource: input.Installer.NameSource,
		DetectedFrom: []installplan.Detection{{
			Kind:   detectionKind,
			Path:   ".",
			Reason: reason,
		}},
	}
}

func copyInstruction(input installplan.BuildInput, sourceRel, targetRel, targetRoot, targetRootID string) installplan.Instruction {
	targetRelative := filepath.ToSlash(targetRel)
	targetRootID = strings.TrimSpace(targetRootID)
	if targetRootID == "" && strings.TrimSpace(targetRoot) != "" {
		targetRelative = filepath.ToSlash(filepath.Join(targetRoot, targetRel))
	}
	return installplan.Instruction{
		Kind:            installplan.InstructionKindCopy,
		SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(sourceRel)),
		StagingRelative: filepath.ToSlash(targetRel),
		TargetRoot:      targetRootID,
		TargetRelative:  targetRelative,
	}
}

func dataRootForBG3(files []string) string {
	dirs := map[string]string{}
	for _, file := range files {
		dir := filepath.ToSlash(filepath.Dir(file))
		if dir == "." {
			dir = ""
		}
		dirs[strings.ToUpper(dir)] = dir
	}
	var roots []string
	for _, dir := range dirs {
		base := filepath.Base(filepath.FromSlash(dir))
		if strings.EqualFold(base, "Public") || strings.EqualFold(base, "Generated") {
			parent := filepath.ToSlash(filepath.Dir(filepath.FromSlash(dir)))
			if parent == "." {
				parent = ""
			}
			roots = append(roots, parent)
		}
	}
	if len(roots) > 0 {
		sort.Strings(roots)
		return roots[0]
	}
	var dataRoots []string
	for _, dir := range dirs {
		if strings.EqualFold(filepath.Base(filepath.FromSlash(dir)), "Data") {
			dataRoots = append(dataRoots, filepath.ToSlash(dir))
		}
	}
	sort.Slice(dataRoots, func(i, j int) bool {
		leftParts := len(strings.Split(strings.Trim(dataRoots[i], "/"), "/"))
		rightParts := len(strings.Split(strings.Trim(dataRoots[j], "/"), "/"))
		if leftParts != rightParts {
			return leftParts < rightParts
		}
		return dataRoots[i] < dataRoots[j]
	})
	if len(dataRoots) > 0 {
		return dataRoots[0]
	}
	return ""
}

func isOriginalFile(file string) bool {
	_, ok := originalPakFiles[strings.ToLower(filepath.Base(file))]
	return ok
}

func pathContainsSegment(file, segment string) bool {
	for _, part := range strings.Split(filepath.ToSlash(file), "/") {
		if strings.EqualFold(part, segment) {
			return true
		}
	}
	return false
}

func fromSegment(file, segment string) (string, bool) {
	parts := strings.Split(filepath.ToSlash(strings.Trim(file, "/")), "/")
	for idx, part := range parts {
		if strings.EqualFold(part, segment) {
			return strings.Join(parts[idx:], "/"), true
		}
	}
	return "", false
}

func lslibVersionFromArchive(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	for {
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".zip" && ext != ".7z" && ext != ".rar" && ext != ".tar" && ext != ".gz" && ext != ".bz2" && ext != ".xz" {
			break
		}
		name = strings.TrimSuffix(name, filepath.Ext(name))
	}
	idx := strings.LastIndex(strings.ToLower(name), "-v")
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(name[idx+2:])
}

func sortPlan(plan *installplan.Plan) {
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	sort.SliceStable(plan.Metadata, func(i, j int) bool {
		return plan.Metadata[i].StagingRelative < plan.Metadata[j].StagingRelative
	})
}
