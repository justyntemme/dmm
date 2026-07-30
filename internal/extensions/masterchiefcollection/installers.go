package masterchiefcollection

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	modInfoJSONFile      = "modinfo.json"
	modConfigFile        = "modpack_config.cfg"
	modConfigDestElement = "$MCC_home\\"
	assemblyExtension    = ".asmp"
	mapExtension         = ".map"
)

type haloGame struct {
	internalID string
	name       string
	modsPath   string
}

var haloGames = map[string]haloGame{
	"halo1":     {internalID: "1", name: "Halo: CE", modsPath: "halo1"},
	"halo2":     {internalID: "2", name: "Halo 2", modsPath: "halo2"},
	"halo3":     {internalID: "3", name: "Halo 3", modsPath: "halo3"},
	"odst":      {internalID: "4", name: "ODST", modsPath: "halo3odst"},
	"halo4":     {internalID: "5", name: "Halo 4", modsPath: "halo4"},
	"haloreach": {internalID: "6", name: "Reach", modsPath: "haloreach"},
}

type modInfoJSON struct {
	ModIdentifier struct {
		ModGUID string `json:"ModGuid"`
	} `json:"ModIdentifier"`
	ModVersion *struct {
		Major int `json:"Major"`
		Minor int `json:"Minor"`
		Patch int `json:"Patch"`
	} `json:"ModVersion"`
	Engine string `json:"Engine"`
	Title  struct {
		Neutral string `json:"Neutral"`
	} `json:"Title"`
}

type modConfig struct {
	Entries []modConfigEntry `json:"entries"`
}

type modConfigEntry struct {
	Source string `json:"src"`
	Dest   string `json:"dest"`
}

func matchPlugAndPlay(root string) bool {
	return findBaseName(root, modInfoJSONFile) != ""
}

func buildPlugAndPlay(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	modInfo := firstFileWithBase(files, modInfoJSONFile)
	if modInfo == "" {
		return installplan.Plan{}, installplan.Unsupported("Halo MCC plug-and-play archive does not contain modinfo.json")
	}
	info, err := readModInfo(filepath.Join(input.ExtractedRoot, filepath.FromSlash(modInfo)))
	if err != nil {
		return installplan.Plan{}, err
	}
	game, ok := haloGames[strings.ToLower(strings.TrimSpace(info.Engine))]
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Halo MCC modinfo.json declares unsupported Engine " + strings.TrimSpace(info.Engine))
	}
	modInfoSegments := splitPath(modInfo)
	modFolderIndex := len(modInfoSegments) - 1
	if modFolderIndex < 0 {
		modFolderIndex = 0
	}
	builder := newPlanBuilder(input, plugAndPlayModType)
	for _, file := range files {
		if filepath.Ext(filepath.Base(file)) == "" {
			continue
		}
		segments := splitPath(file)
		if modFolderIndex > len(segments) {
			continue
		}
		target := filepath.ToSlash(filepath.Join(segments[modFolderIndex:]...))
		if err := builder.add(file, target); err != nil {
			return installplan.Plan{}, err
		}
	}
	builder.metadata = append(builder.metadata, haloModInfoMetadata(info, game, modInfo))
	return builder.plan("vortex-custom-installer", "matched Halo MCC plug-and-play modinfo.json archive")
}

func haloModInfoMetadata(info modInfoJSON, game haloGame, sourcePath string) installplan.ModMetadata {
	return installplan.ModMetadata{
		Kind:       "halo-mcc-modinfo",
		SourcePath: filepath.ToSlash(sourcePath),
		Name:       strings.TrimSpace(info.Title.Neutral),
		UniqueID:   strings.TrimSpace(info.ModIdentifier.ModGUID),
		Version:    haloVersion(info),
		AdditionalLogicalFileNames: []string{
			strings.ToLower(game.name),
			strings.ToLower(game.internalID),
		},
	}
}

func haloVersion(info modInfoJSON) string {
	if info.ModVersion == nil {
		return ""
	}
	return fmt.Sprintf("%d.%d.%d", info.ModVersion.Major, info.ModVersion.Minor, info.ModVersion.Patch)
}

func readModInfo(path string) (modInfoJSON, error) {
	var info modInfoJSON
	if !installplan.ReadManifestJSON(path, &info) {
		return modInfoJSON{}, errors.New("invalid modinfo.json")
	}
	return info, nil
}

func matchModConfig(root string) bool {
	files, err := listFiles(root)
	if err != nil {
		return false
	}
	if firstFileWithBase(files, modConfigFile) == "" {
		return false
	}
	return !isAssemblyOnlyMod(files)
}

func isAssemblyOnlyMod(files []string) bool {
	hasAssembly := false
	hasMap := false
	for _, file := range files {
		switch strings.ToLower(filepath.Ext(file)) {
		case assemblyExtension:
			hasAssembly = true
		case mapExtension:
			hasMap = true
		}
	}
	return hasAssembly && !hasMap
}

func buildModConfig(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	configFile := firstFileWithBase(files, modConfigFile)
	if configFile == "" {
		return installplan.Plan{}, installplan.Unsupported("Halo MCC archive does not contain modpack_config.cfg")
	}
	config, err := readModConfig(filepath.Join(input.ExtractedRoot, filepath.FromSlash(configFile)))
	if err != nil {
		return installplan.Plan{}, err
	}
	if len(config.Entries) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Halo MCC modpack_config.cfg contains no entries")
	}
	entries := map[string]string{}
	for _, entry := range config.Entries {
		source := filepath.ToSlash(strings.TrimSpace(entry.Source))
		dest := strings.TrimSpace(entry.Dest)
		if source == "" || dest == "" || !strings.HasPrefix(strings.ToLower(dest), strings.ToLower(modConfigDestElement)) {
			continue
		}
		target := dest[len(modConfigDestElement):]
		entries[strings.ToLower(source)] = filepath.ToSlash(strings.ReplaceAll(target, "\\", "/"))
	}
	builder := newPlanBuilder(input, rootModType)
	for _, file := range files {
		if file == configFile {
			continue
		}
		ext := strings.ToLower(filepath.Ext(file))
		if ext == "" || ext == ".txt" || ext == assemblyExtension {
			continue
		}
		target, ok := entries[strings.ToLower(file)]
		if !ok {
			continue
		}
		if err := builder.add(file, target); err != nil {
			return installplan.Plan{}, err
		}
	}
	return builder.plan("vortex-custom-installer", "matched Halo MCC modpack_config.cfg archive")
}

func readModConfig(path string) (modConfig, error) {
	var config modConfig
	if !installplan.ReadManifestJSON(path, &config) {
		return modConfig{}, errors.New("invalid modpack_config.cfg")
	}
	return config, nil
}

func matchHaloGameFolder(root string) bool {
	files, err := listFiles(root)
	if err != nil {
		return false
	}
	return len(identifyHaloGames(files)) > 0
}

func buildHaloGameFolder(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	games := identifyHaloGames(files)
	if len(games) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Halo MCC archive does not contain a recognized Halo game folder")
	}
	builder := newPlanBuilder(input, rootModType)
	for _, game := range games {
		for _, file := range files {
			if filepath.Ext(file) == "" {
				continue
			}
			segments := splitPath(file)
			rootIdx := segmentIndex(segments, game.modsPath)
			if rootIdx < 0 {
				continue
			}
			target := filepath.ToSlash(filepath.Join(segments[rootIdx:]...))
			if err := builder.add(file, target); err != nil {
				return installplan.Plan{}, err
			}
		}
	}
	return builder.plan("vortex-custom-installer", "matched Halo MCC game folder archive")
}

func identifyHaloGames(files []string) []haloGame {
	filtered := make([]string, 0, len(files))
	for _, file := range files {
		if filepath.Ext(file) != "" {
			filtered = append(filtered, file)
		}
	}
	var keys []string
	for key := range haloGames {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var out []haloGame
	for _, key := range keys {
		game := haloGames[key]
		for _, file := range filtered {
			if segmentIndex(splitPath(file), game.modsPath) >= 0 {
				out = append(out, game)
				break
			}
		}
	}
	return out
}

type planBuilder struct {
	input        installplan.BuildInput
	modType      string
	instructions []installplan.Instruction
	metadata     []installplan.ModMetadata
}

func newPlanBuilder(input installplan.BuildInput, modType string) *planBuilder {
	return &planBuilder{input: input, modType: modType}
}

func (b *planBuilder) add(sourceRel, targetRel string) error {
	sourceRel = filepath.ToSlash(strings.TrimSpace(sourceRel))
	targetRel = filepath.ToSlash(strings.TrimSpace(targetRel))
	if sourceRel == "" || targetRel == "" {
		return errors.New("Halo MCC installer produced an empty path")
	}
	b.instructions = append(b.instructions, installplan.Instruction{
		Kind:            installplan.InstructionKindCopy,
		SourcePath:      filepath.Join(b.input.ExtractedRoot, filepath.FromSlash(sourceRel)),
		StagingRelative: targetRel,
		TargetRelative:  targetRel,
	})
	return nil
}

func (b *planBuilder) plan(kind, reason string) (installplan.Plan, error) {
	if len(b.instructions) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Halo MCC installer matched but produced no deployable files")
	}
	sort.SliceStable(b.instructions, func(i, j int) bool {
		return b.instructions[i].TargetRelative < b.instructions[j].TargetRelative
	})
	return installplan.Plan{
		GameID:     b.input.GameID,
		ModType:    b.modType,
		PlannerID:  b.input.Installer.ID,
		NameSource: b.input.Installer.NameSource,
		DetectedFrom: []installplan.Detection{{
			Kind:   kind,
			Reason: reason,
		}},
		Metadata:     b.metadata,
		Instructions: b.instructions,
	}, nil
}

func listFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func findBaseName(root, basename string) string {
	files, err := listFiles(root)
	if err != nil {
		return ""
	}
	return firstFileWithBase(files, basename)
}

func firstFileWithBase(files []string, basename string) string {
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), basename) {
			return file
		}
	}
	return ""
}

func splitPath(file string) []string {
	file = filepath.ToSlash(strings.TrimSpace(file))
	if file == "" {
		return nil
	}
	return strings.Split(file, "/")
}

func segmentIndex(segments []string, want string) int {
	for idx, segment := range segments {
		if strings.EqualFold(segment, want) {
			return idx
		}
	}
	return -1
}
