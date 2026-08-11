package modcontent

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	TypePlugin     = "plugin"
	TypeInterface  = "interface"
	TypeTexture    = "texture"
	TypeMesh       = "mesh"
	TypeAnimation  = "animation"
	TypeMap        = "map"
	TypeMusic      = "music"
	TypeShader     = "shader"
	TypeArchive    = "archive"
	TypeScript     = "script"
	TypeExtender   = "extender"
	TypeConfig     = "config"
	TypeExecutable = "executable"
	TypeFOMOD      = "fomod"
)

type Summary struct {
	Types   []string
	Empty   bool
	Scanned int
}

type fileTypeRule struct {
	Type      string
	Condition func(gameID, path string) bool
}

var typeOrder = map[string]int{
	TypePlugin:     0,
	TypeInterface:  1,
	TypeTexture:    2,
	TypeMesh:       3,
	TypeAnimation:  4,
	TypeMap:        5,
	TypeMusic:      6,
	TypeShader:     7,
	TypeArchive:    8,
	TypeScript:     9,
	TypeExtender:   10,
	TypeConfig:     11,
	TypeExecutable: 12,
	TypeFOMOD:      13,
}

var scriptExtenderGames = set("oblivion", "skyrim", "skyrimse", "skyrimvr", "fallout3", "falloutnv", "fallout4", "fallout4vr", "enderal", "enderalspecialedition")
var gamesUsingPythonScripting = set("thesims4")
var gamesUsingDLLPlugins = set("stardewvalley")
var gamesUsingImagesAsTextures = set("stardewvalley", "darksouls2", "intothebreach")

var fileTypes = map[string][]fileTypeRule{
	".dds": {{Type: TypeTexture}},
	".exe": {{Type: TypeExecutable}},
	".bat": {{Type: TypeExecutable}},
	".cmd": {{Type: TypeExecutable}},
	".jar": {{Type: TypeExecutable}},
	".py": {
		{Type: TypeExecutable, Condition: func(gameID, _ string) bool { return !gamesUsingPythonScripting[gameID] }},
		{Type: TypeScript, Condition: func(gameID, _ string) bool { return gamesUsingPythonScripting[gameID] }},
	},
	".swf":     {{Type: TypeInterface}},
	".xml":     {{Type: TypeConfig}},
	".json":    {{Type: TypeConfig, Condition: func(_ string, path string) bool { return strings.ToLower(filepath.Base(path)) != "manifest.json" }}},
	".ini":     {{Type: TypeConfig}},
	".wav":     {{Type: TypeMusic}},
	".mp3":     {{Type: TypeMusic}},
	".ogg":     {{Type: TypeMusic}},
	".png":     {{Type: TypeTexture, Condition: func(gameID, _ string) bool { return gamesUsingImagesAsTextures[gameID] }}},
	".jpg":     {{Type: TypeTexture, Condition: func(gameID, _ string) bool { return gamesUsingImagesAsTextures[gameID] }}},
	".tga":     {{Type: TypeTexture}},
	".unity3d": {{Type: TypeArchive}},
	".arc":     {{Type: TypeArchive}},
	".tri":     {{Type: TypeMesh}},
	".pak":     {{Type: TypeArchive}},
	".nif":     {{Type: TypeMesh}},
	".xwm":     {{Type: TypeMusic}},
	".fuz":     {{Type: TypeMusic}},
	".bsa":     {{Type: TypeArchive}},
	".ba2":     {{Type: TypeArchive}},
	".esp":     {{Type: TypePlugin}},
	".esm":     {{Type: TypePlugin}},
	".esl":     {{Type: TypePlugin}},
	".pex":     {{Type: TypeScript}},
	".dll": {
		{Type: TypeExtender, Condition: func(gameID, _ string) bool { return scriptExtenderGames[gameID] }},
		{Type: TypePlugin, Condition: func(gameID, _ string) bool { return gamesUsingDLLPlugins[gameID] }},
	},
	".hkx":             {{Type: TypeAnimation}},
	".ts4script":       {{Type: TypeScript}},
	".package":         {{Type: TypeArchive}},
	".bpi":             {{Type: TypePlugin}},
	".blueprint":       {{Type: TypePlugin}},
	".trayitem":        {{Type: TypePlugin}},
	".sfx":             {{Type: TypeMusic}},
	".ion":             {{Type: TypePlugin}},
	".householdbinary": {{Type: TypePlugin}},
	".sgi":             {{Type: TypePlugin}},
	".hhi":             {{Type: TypePlugin}},
	".room":            {{Type: TypePlugin}},
	".midi":            {{Type: TypeMusic}},
	".rmi":             {{Type: TypePlugin}},
	".tbin":            {{Type: TypeTexture}},
	".mod":             {{Type: TypePlugin}},
	".hak":             {{Type: TypeArchive}},
	".bmu":             {{Type: TypeMusic}},
	".ani":             {{Type: TypeAnimation}},
	".rpf":             {{Type: TypeArchive}},
	".asi":             {{Type: TypeExtender}},
	".ytd":             {{Type: TypeTexture}},
	".awc":             {{Type: TypeMusic}},
	".ymt":             {{Type: TypeConfig}},
	".gfx":             {{Type: TypeInterface}},
	".meta":            {{Type: TypeConfig}},
	".lua":             {{Type: TypeScript}},
}

func FromFiles(gameID string, files []string) Summary {
	gameID = strings.ToLower(strings.TrimSpace(gameID))
	found := map[string]struct{}{}
	scanned := 0
	for _, file := range files {
		file = filepath.ToSlash(strings.TrimSpace(file))
		if file == "" {
			continue
		}
		scanned++
		for _, rule := range fileTypes[strings.ToLower(filepath.Ext(file))] {
			if rule.Condition == nil || rule.Condition(gameID, file) {
				found[rule.Type] = struct{}{}
				break
			}
		}
	}
	return Summary{Types: orderedTypes(found), Empty: scanned == 0, Scanned: scanned}
}

func ScanDirectory(root, gameID string) (Summary, error) {
	files := []string{}
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
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
	}); err != nil {
		return Summary{}, err
	}
	return FromFiles(gameID, files), nil
}

func orderedTypes(found map[string]struct{}) []string {
	out := make([]string, 0, len(found))
	for typ := range found {
		out = append(out, typ)
	}
	sort.Slice(out, func(i, j int) bool {
		left, leftOK := typeOrder[out[i]]
		right, rightOK := typeOrder[out[j]]
		if leftOK && rightOK && left != right {
			return left < right
		}
		if leftOK != rightOK {
			return leftOK
		}
		return out[i] < out[j]
	})
	return out
}

func set(values ...string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}
