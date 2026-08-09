package darkestdungeon

import (
	"bytes"
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	projectFile        = "project.xml"
	heroPortraitSuffix = "_portrait_roster.png"
)

var alphaOnly = regexp.MustCompile(`[^A-Za-z]`)

type projectXML struct {
	Title string `xml:"Title"`
}

type noProjectMapping struct {
	source      string
	destination string
}

func matchProjectArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	return err == nil && projectPath(files) != ""
}

func buildProjectArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	projectRel := projectPath(files)
	if projectRel == "" {
		return installplan.Plan{}, installplan.Unsupported("Darkest Dungeon archive does not contain project.xml")
	}
	modName := darkestDungeonModName(input.ArchiveName, input.ExtractedRoot)
	if modName == "" {
		return installplan.Plan{}, installplan.Unsupported("Darkest Dungeon archive is missing a safe mod folder name")
	}
	projectRoot := filepath.ToSlash(filepath.Dir(projectRel))
	if projectRoot == "." {
		projectRoot = ""
	}
	title := readProjectTitle(filepath.Join(input.ExtractedRoot, filepath.FromSlash(projectRel)), modName)
	projectContent, err := renderProjectXML(title, expectedModPath(input.GamePath, modName))
	if err != nil {
		return installplan.Plan{}, err
	}
	plan := newPlan(input, "darkestdungeon-project", projectRel, "Vortex Darkest Dungeon project installer matched project.xml")
	projectStagingRel := filepath.ToSlash(filepath.Join(modName, projectFile))
	projectTargetRel := filepath.ToSlash(filepath.Join(input.TargetRoot, projectStagingRel))
	plan.Instructions = append(plan.Instructions, generatedProjectInstruction(projectContent, projectStagingRel, projectTargetRel))
	plan.Metadata = append(plan.Metadata, installplan.ModMetadata{
		Kind:            "darkestdungeon-project",
		SourcePath:      projectRel,
		StagingRelative: projectStagingRel,
		TargetRelative:  projectTargetRel,
		Name:            title,
		UniqueID:        modName,
	})
	for _, file := range files {
		if !simplearchive.PathWithinRoot(file, projectRoot) || strings.EqualFold(filepath.Base(file), projectFile) {
			continue
		}
		rel := simplearchive.StripRoot(file, projectRoot)
		if rel == "" {
			continue
		}
		plan.Instructions = append(plan.Instructions, copyInstruction(input, file, filepath.ToSlash(filepath.Join(modName, rel))))
	}
	sortPlan(&plan)
	return plan, nil
}

func matchNoProjectArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || projectPath(files) != "" || simplearchive.ContainsFOMOD(files) {
		return false
	}
	return len(files) > 0
}

func buildNoProjectArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	if projectPath(files) != "" {
		return installplan.Plan{}, installplan.Unsupported("Darkest Dungeon no-project installer excludes archives that already contain project.xml")
	}
	dirs, err := gameDirectoryStruct(input.GamePath)
	if err != nil {
		return installplan.Plan{}, err
	}
	mappings := noProjectMappings(files, dirs)
	if len(mappings) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Darkest Dungeon archive does not match Vortex project, game-folder, or hero portrait installer rules")
	}
	modName := darkestDungeonModName(input.ArchiveName, input.ExtractedRoot)
	if modName == "" {
		return installplan.Plan{}, installplan.Unsupported("Darkest Dungeon archive is missing a safe mod folder name")
	}
	projectContent, err := renderProjectXML(modName, expectedModPath(input.GamePath, modName))
	if err != nil {
		return installplan.Plan{}, err
	}
	plan := newPlan(input, "darkestdungeon-no-project", "", "Vortex Darkest Dungeon no-project installer matched game-folder or hero portrait structure")
	projectStagingRel := filepath.ToSlash(filepath.Join(modName, projectFile))
	projectTargetRel := filepath.ToSlash(filepath.Join(input.TargetRoot, projectStagingRel))
	plan.Instructions = append(plan.Instructions, generatedProjectInstruction(projectContent, projectStagingRel, projectTargetRel))
	plan.Metadata = append(plan.Metadata, installplan.ModMetadata{
		Kind:            "darkestdungeon-generated-project",
		StagingRelative: projectStagingRel,
		TargetRelative:  projectTargetRel,
		Name:            modName,
		UniqueID:        modName,
	})
	for _, mapping := range mappings {
		plan.Instructions = append(plan.Instructions, copyInstruction(input, mapping.source, filepath.ToSlash(filepath.Join(modName, mapping.destination))))
	}
	sortPlan(&plan)
	return plan, nil
}

func newPlan(input installplan.BuildInput, detectionKind, detectionPath, detectionReason string) installplan.Plan {
	plan := installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: input.Installer.NameSource,
	}
	if strings.TrimSpace(detectionPath) != "" {
		plan.DetectedFrom = append(plan.DetectedFrom, installplan.Detection{
			Kind:   detectionKind,
			Path:   detectionPath,
			Reason: detectionReason,
		})
	} else {
		plan.DetectedFrom = append(plan.DetectedFrom, installplan.Detection{
			Kind:   detectionKind,
			Reason: detectionReason,
		})
	}
	return plan
}

func copyInstruction(input installplan.BuildInput, sourceRel, stagingRel string) installplan.Instruction {
	targetRel := filepath.ToSlash(filepath.Join(input.TargetRoot, stagingRel))
	return installplan.Instruction{
		Kind:            installplan.InstructionKindCopy,
		SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(sourceRel)),
		StagingRelative: stagingRel,
		TargetRoot:      input.TargetRootID,
		TargetRelative:  targetRel,
	}
}

func generatedProjectInstruction(content, stagingRel, targetRel string) installplan.Instruction {
	return installplan.Instruction{
		Kind:                    installplan.InstructionKindGenerateFromGameFile,
		GeneratedDefaultContent: content,
		StagingRelative:         stagingRel,
		TargetRelative:          targetRel,
	}
}

func projectPath(files []string) string {
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), projectFile) {
			return file
		}
	}
	return ""
}

func readProjectTitle(path, fallback string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fallback
	}
	var project projectXML
	if err := xml.Unmarshal(data, &project); err != nil {
		return fallback
	}
	if title := strings.TrimSpace(project.Title); title != "" {
		return title
	}
	return fallback
}

func renderProjectXML(title, modPath string) (string, error) {
	if strings.TrimSpace(title) == "" {
		return "", errors.New("Darkest Dungeon project title is required")
	}
	if strings.TrimSpace(modPath) == "" {
		return "", errors.New("Darkest Dungeon game path is required to generate project.xml ModDataPath")
	}
	return `<?xml version="1.0" encoding="utf-8"?>
<project>
    <PreviewIconFile>preview_icon.png</PreviewIconFile>
    <ItemDescriptionShort/>
    <ModDataPath>` + escapeXML(modPath) + `</ModDataPath>
    <Title>` + escapeXML(title) + `</Title>
    <Language>english</Language>
    <UpdateDetails/>
    <Visibility>public</Visibility>
    <UploadMode>direct_upload</UploadMode>
    <VersionMajor>1</VersionMajor>
    <VersionMinor>0</VersionMinor>
    <TargetBuild>0</TargetBuild>
    <Tags>
    </Tags>
    <ItemDescription>File generated by Vortex</ItemDescription>
</project>`, nil
}

func escapeXML(value string) string {
	var buf bytes.Buffer
	if err := xml.EscapeText(&buf, []byte(value)); err != nil {
		return value
	}
	return buf.String()
}

func expectedModPath(gamePath, modName string) string {
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" || strings.TrimSpace(modName) == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(gamePath, modsDir, modName))
}

func darkestDungeonModName(archiveName, extractedRoot string) string {
	base := strings.TrimSpace(archiveName)
	if base == "" {
		base = filepath.Base(strings.TrimRight(extractedRoot, string(filepath.Separator)))
	}
	for {
		ext := filepath.Ext(base)
		if ext == "" {
			break
		}
		base = strings.TrimSuffix(base, ext)
	}
	base = alphaOnly.ReplaceAllString(base, "")
	if base == "" {
		return "DarkestDungeonMod"
	}
	return base
}

func gameDirectoryStruct(gamePath string) ([]string, error) {
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return nil, installplan.Unsupported("Darkest Dungeon game path is required for Vortex no-project archive matching")
	}
	dirs := []string{}
	err := filepath.WalkDir(gamePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() || path == gamePath {
			return nil
		}
		rel, err := filepath.Rel(gamePath, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if rel != "." {
			dirs = append(dirs, rel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !containsFold(dirs, "dlc") {
		dirs = append(dirs, "dlc")
	}
	sort.Strings(dirs)
	return dirs, nil
}

func noProjectMappings(files, gameDirs []string) []noProjectMapping {
	onlyFiles := filesWithExtension(files)
	portraits := portraitFiles(onlyFiles)
	mappings := matchingDirectoryFiles(onlyFiles, gameDirs, portraits)
	if len(mappings) == 0 {
		for _, portrait := range portraits {
			mappings = append(mappings, heroPortraitMappings(onlyFiles, portrait, mappings)...)
		}
	} else {
		mappings = append(mappings, leftoverHeroFiles(onlyFiles, mappings)...)
	}
	deduped := map[string]noProjectMapping{}
	for _, mapping := range mappings {
		if strings.TrimSpace(mapping.source) == "" || strings.TrimSpace(mapping.destination) == "" {
			continue
		}
		deduped[mapping.source+"=>"+mapping.destination] = mapping
	}
	out := make([]noProjectMapping, 0, len(deduped))
	for _, mapping := range deduped {
		out = append(out, mapping)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].destination == out[j].destination {
			return out[i].source < out[j].source
		}
		return out[i].destination < out[j].destination
	})
	return out
}

func filesWithExtension(files []string) []string {
	out := []string{}
	for _, file := range files {
		if filepath.Ext(filepath.Base(file)) != "" {
			out = append(out, file)
		}
	}
	return out
}

func portraitFiles(files []string) []string {
	out := []string{}
	for _, file := range files {
		if strings.Contains(file, heroPortraitSuffix) {
			out = append(out, file)
		}
	}
	return out
}

func matchingDirectoryFiles(files, gameDirs, portraits []string) []noProjectMapping {
	out := []noProjectMapping{}
	for _, file := range files {
		if len(portraits) > 0 && sameVortexParent(portraits[0], file) {
			continue
		}
		if archiveDirMatchesGame(file, gameDirs) {
			out = append(out, noProjectMapping{source: file, destination: file})
		}
	}
	return out
}

func archiveDirMatchesGame(file string, gameDirs []string) bool {
	dir := strings.ToLower(filepath.ToSlash(filepath.Dir(file)))
	for _, gameDir := range gameDirs {
		gameDir = strings.ToLower(filepath.ToSlash(strings.Trim(gameDir, "/")))
		if gameDir != "" && strings.Contains(dir, gameDir) {
			return true
		}
	}
	return false
}

func sameVortexParent(lhs, rhs string) bool {
	return vortexParent(lhs) == vortexParent(rhs)
}

func vortexParent(pathRel string) string {
	pathRel = filepath.ToSlash(strings.Trim(pathRel, "/"))
	if pathRel == "" {
		return ""
	}
	dir := pathRel
	if filepath.Ext(filepath.Base(pathRel)) != "" {
		dir = filepath.ToSlash(filepath.Dir(pathRel))
	}
	if dir == "." || dir == "" {
		return ""
	}
	parent := filepath.ToSlash(filepath.Dir(dir))
	if parent == "." {
		return ""
	}
	return parent
}

func heroPortraitMappings(files []string, portrait string, existing []noProjectMapping) []noProjectMapping {
	portraitDir := filepath.ToSlash(filepath.Dir(portrait))
	if portraitDir == "." || portraitDir == "" {
		return nil
	}
	heroSuffix := ""
	if len(portraitDir) >= 2 {
		heroSuffix = portraitDir[len(portraitDir)-2:]
	}
	heroName := heroNameFromPortraitDir(portraitDir)
	if heroName == "" {
		return nil
	}
	heroPath := filepath.ToSlash(filepath.Join("heroes", heroName))
	out := externalHeroFiles(files, portraitDir, heroPath, existing)
	prefix := strings.TrimSuffix(portrait, filepath.Base(portrait))
	for _, file := range files {
		if !strings.HasPrefix(file, portraitDir+"/") && file != portrait {
			continue
		}
		suffix := strings.TrimPrefix(file, prefix)
		out = append(out, noProjectMapping{
			source:      file,
			destination: filepath.ToSlash(filepath.Join("heroes", heroName, heroName+heroSuffix, suffix)),
		})
	}
	return out
}

func externalHeroFiles(files []string, portraitDir, heroPath string, existing []noProjectMapping) []noProjectMapping {
	existingSources := map[string]struct{}{}
	for _, mapping := range existing {
		existingSources[mapping.source] = struct{}{}
	}
	out := []noProjectMapping{}
	for _, file := range files {
		if _, ok := existingSources[file]; ok {
			continue
		}
		if sameVortexParent(file, portraitDir) {
			out = append(out, noProjectMapping{
				source:      file,
				destination: filepath.ToSlash(filepath.Join(heroPath, file)),
			})
		}
	}
	return out
}

func leftoverHeroFiles(files []string, existing []noProjectMapping) []noProjectMapping {
	existingSources := map[string]struct{}{}
	for _, mapping := range existing {
		existingSources[mapping.source] = struct{}{}
	}
	out := []noProjectMapping{}
	for _, file := range files {
		if _, ok := existingSources[file]; ok {
			continue
		}
		segments := strings.Split(filepath.ToSlash(file), "/")
		for idx, segment := range segments {
			if segment == "heroes" && idx < len(segments)-1 {
				out = append(out, noProjectMapping{source: file, destination: filepath.ToSlash(filepath.Join(segments[idx:]...))})
				break
			}
		}
	}
	return out
}

func heroNameFromPortraitDir(dir string) string {
	re := regexp.MustCompile(`_[A-Z]`)
	parts := strings.Split(re.ReplaceAllString(filepath.ToSlash(dir), ""), "/")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[len(parts)-1])
}

func containsFold(values []string, want string) bool {
	for _, value := range values {
		if strings.EqualFold(value, want) {
			return true
		}
	}
	return false
}

func sortPlan(plan *installplan.Plan) {
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	sort.SliceStable(plan.DetectedFrom, func(i, j int) bool {
		return plan.DetectedFrom[i].Path < plan.DetectedFrom[j].Path
	})
	sort.SliceStable(plan.Metadata, func(i, j int) bool {
		return plan.Metadata[i].TargetRelative < plan.Metadata[j].TargetRelative
	})
}
