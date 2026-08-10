package baldursgate3

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/divine"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func willDeploy(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	paks := bg3PakCandidates(input)
	if len(paks) == 0 {
		return sdk.EventHandlerResult{Messages: []string{"BG3 load-order export skipped because this profile has no enabled BG3 pak mods."}}, nil
	}
	toolPath, ok := findManagedDivineTool(input)
	if !ok {
		return sdk.EventHandlerResult{
			Notices: []sdk.EventNotice{{
				Message:     "BG3 pak files need LSLib/divine before DMM can update modsettings.lsx. Install the BG3 LSLib/Divine tool package, then deploy again.",
				ActionLabel: "Review BG3 setup",
				HelpURL:     "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-baldursgate3/src",
			}},
			Messages: []string{"BG3 modsettings.lsx generation skipped because LSLib/divine was not found in managed tool metadata."},
		}, nil
	}
	mapping, loaded, locked, err := generateModSettingsMapping(ctx, input, toolPath, paks)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if loaded == 0 {
		return sdk.EventHandlerResult{
			Messages: []string{"BG3 load-order export skipped because all enabled pak files were locked/listed packages without meta.lsx."},
		}, nil
	}
	message := fmt.Sprintf("Generated BG3 modsettings.lsx with %d enabled pak mod%s.", loaded, plural(loaded))
	if locked > 0 {
		message += fmt.Sprintf(" %d locked/listed pak%s left out of modsettings.lsx, matching Vortex behavior.", locked, plural(locked))
	}
	return sdk.EventHandlerResult{
		Mappings: []deploy.FileMapping{mapping},
		Messages: []string{message},
	}, nil
}

type pakCandidate struct {
	Mod      sdk.DeploymentMod
	File     sdk.DeploymentModFile
	FilePath string
	Name     string
}

type pakInfo struct {
	Folder        string
	MD5           string
	Name          string
	PublishHandle string
	UUID          string
	Version       string
}

func bg3PakCandidates(input sdk.EventHandlerInput) []pakCandidate {
	var out []pakCandidate
	for _, mod := range input.Mods {
		if !mod.Enabled || !strings.EqualFold(strings.TrimSpace(mod.ModType), pakModType) {
			continue
		}
		for _, file := range mod.Files {
			if !strings.EqualFold(filepath.Ext(file.Path), ".pak") {
				continue
			}
			rel := filepath.Clean(filepath.FromSlash(strings.TrimSpace(file.Path)))
			if rel == "." || filepath.IsAbs(rel) || strings.HasPrefix(filepath.ToSlash(rel), "../") {
				continue
			}
			filePath := filepath.Join(mod.StagingPath, rel)
			out = append(out, pakCandidate{
				Mod:      mod,
				File:     file,
				FilePath: filePath,
				Name:     strings.TrimSuffix(filepath.Base(file.Path), filepath.Ext(file.Path)),
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Mod.Priority != out[j].Mod.Priority {
			return out[i].Mod.Priority < out[j].Mod.Priority
		}
		return strings.ToLower(out[i].File.Path) < strings.ToLower(out[j].File.Path)
	})
	return out
}

func findManagedDivineTool(input sdk.EventHandlerInput) (string, bool) {
	for _, mod := range input.Mods {
		for _, metadata := range mod.Metadata {
			if !strings.EqualFold(strings.TrimSpace(metadata.Kind), "tool") || !strings.EqualFold(strings.TrimSpace(metadata.UniqueID), "bg3-lslib-divine") {
				continue
			}
			for _, rel := range []string{metadata.StagingRelative, metadata.TargetRelative, metadata.SourcePath, "tools/divine.exe"} {
				rel = strings.TrimSpace(rel)
				if rel == "" || filepath.IsAbs(rel) {
					continue
				}
				clean := filepath.Clean(filepath.FromSlash(rel))
				if clean == "." || strings.HasPrefix(filepath.ToSlash(clean), "../") {
					continue
				}
				path := filepath.Join(mod.StagingPath, clean)
				if info, err := os.Stat(path); err == nil && !info.IsDir() {
					return path, true
				}
			}
		}
	}
	return "", false
}

func generateModSettingsMapping(ctx context.Context, input sdk.EventHandlerInput, toolPath string, paks []pakCandidate) (deploy.FileMapping, int, int, error) {
	localData, _, err := localDataPath(ctx, sdk.TargetRootInput{
		AppID:       input.AppID,
		GamePath:    input.GamePath,
		LibraryPath: input.LibraryPath,
	})
	if err != nil {
		return deploy.FileMapping{}, 0, 0, err
	}
	workDir := filepath.Join(input.WorkDir, "bg3-modsettings")
	if err := os.RemoveAll(workDir); err != nil {
		return deploy.FileMapping{}, 0, 0, err
	}
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		return deploy.FileMapping{}, 0, 0, err
	}
	runner := divine.Runner{ExecutablePath: toolPath, WorkDir: workDir}
	infos := make([]pakInfo, 0, len(paks))
	locked := 0
	for idx, pak := range paks {
		input.ReportProgress("Reading BG3 pak metadata: "+pak.Name, idx+1, len(paks))
		info, listed, err := readPakInfo(ctx, runner, workDir, pak)
		if err != nil {
			return deploy.FileMapping{}, 0, locked, err
		}
		if listed {
			locked++
			continue
		}
		infos = append(infos, info)
	}
	sourcePath := filepath.Join(workDir, "PlayerProfiles", "Public", "modsettings.lsx")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
		return deploy.FileMapping{}, 0, locked, err
	}
	body := bg3ModSettingsXML(infos)
	if err := os.WriteFile(sourcePath, []byte(body), 0o600); err != nil {
		return deploy.FileMapping{}, 0, locked, err
	}
	sum, err := fileSHA256(sourcePath)
	if err != nil {
		return deploy.FileMapping{}, 0, locked, err
	}
	return deploy.FileMapping{
		SourcePath:     sourcePath,
		TargetRoot:     localData,
		TargetRelative: "PlayerProfiles/Public/modsettings.lsx",
		TargetPolicy:   deploy.TargetPolicyPatchExisting,
		Strategy:       deploy.StrategyCopy,
		Catalog:        "dmm-generated",
		ModID:          "bg3-modsettings",
		Priority:       -1,
		ChecksumSHA256: sum,
	}, len(infos), locked, nil
}

func readPakInfo(ctx context.Context, runner divine.Runner, workDir string, pak pakCandidate) (pakInfo, bool, error) {
	list, err := runner.Run(ctx, divine.Operation{
		Action:  divine.ActionListPackage,
		Options: divine.Options{Source: pak.FilePath, LogLevel: "off"},
	})
	if err != nil {
		return pakInfo{}, false, err
	}
	listed := !packageListContainsMeta(list.Entries)
	if listed {
		return pakInfo{}, true, nil
	}
	dest := filepath.Join(workDir, "extracted", strconv.FormatInt(pak.Mod.ID, 10)+"-"+sanitizeID(pak.Name))
	if err := os.RemoveAll(dest); err != nil {
		return pakInfo{}, false, err
	}
	if err := os.MkdirAll(dest, 0o700); err != nil {
		return pakInfo{}, false, err
	}
	if _, err := runner.Run(ctx, divine.Operation{
		Action: divine.ActionExtractPackage,
		Options: divine.Options{
			Source:      pak.FilePath,
			Destination: dest,
			Expression:  "*/meta.lsx",
		},
	}); err != nil {
		return pakInfo{}, false, err
	}
	metaPath, err := findMetaLSX(dest)
	if err != nil {
		return pakInfo{}, false, err
	}
	info, err := parsePakMetaLSX(metaPath, pak.Name)
	if err != nil {
		return pakInfo{}, false, err
	}
	return info, false, nil
}

func packageListContainsMeta(entries []string) bool {
	for _, entry := range entries {
		pathPart := strings.TrimSpace(strings.Split(entry, "\t")[0])
		if strings.EqualFold(filepath.Base(filepath.FromSlash(pathPart)), "meta.lsx") {
			return true
		}
	}
	return false
}

func findMetaLSX(root string) (string, error) {
	var found string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.EqualFold(d.Name(), "meta.lsx") {
			return nil
		}
		found = path
		return filepath.SkipAll
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", errors.New("Divine extracted no BG3 meta.lsx file")
	}
	return found, nil
}

func parsePakMetaLSX(path, fallbackName string) (pakInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return pakInfo{}, err
	}
	defer file.Close()
	decoder := xml.NewDecoder(file)
	inModuleInfo := false
	values := map[string]string{}
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return pakInfo{}, err
		}
		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "node" && xmlAttr(t, "id") == "ModuleInfo" {
				inModuleInfo = true
				continue
			}
			if inModuleInfo && t.Name.Local == "attribute" {
				id := xmlAttr(t, "id")
				if id != "" {
					values[id] = xmlAttr(t, "value")
				}
			}
		case xml.EndElement:
			if t.Name.Local == "node" && inModuleInfo {
				inModuleInfo = false
			}
		}
	}
	fallbackName = strings.TrimSpace(fallbackName)
	if fallbackName == "" {
		fallbackName = "BG3 Pak"
	}
	info := pakInfo{
		Folder:        firstNonEmpty(values["Folder"], fallbackName),
		MD5:           values["MD5"],
		Name:          firstNonEmpty(values["Name"], fallbackName),
		PublishHandle: firstNonEmpty(values["PublishHandle"], "0"),
		UUID:          strings.TrimSpace(values["UUID"]),
		Version:       firstNonEmpty(values["Version64"], "1"),
	}
	if info.UUID == "" {
		return pakInfo{}, errors.New("BG3 meta.lsx is missing ModuleInfo UUID")
	}
	return info, nil
}

func xmlAttr(element xml.StartElement, name string) string {
	for _, attr := range element.Attr {
		if attr.Name.Local == name {
			return strings.TrimSpace(attr.Value)
		}
	}
	return ""
}

func bg3ModSettingsXML(infos []pakInfo) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString("<save>\n")
	b.WriteString("    <version major=\"4\" minor=\"8\" revision=\"0\" build=\"10\"/>\n")
	b.WriteString("    <region id=\"ModuleSettings\">\n")
	b.WriteString("        <node id=\"root\">\n")
	b.WriteString("            <children>\n")
	b.WriteString("                <node id=\"Mods\">\n")
	b.WriteString("                    <children>\n")
	writeBG3ModuleShortDesc(&b, pakInfo{
		Folder:        "GustavX",
		MD5:           "",
		Name:          "GustavX",
		PublishHandle: "0",
		UUID:          "cb555efe-2d9e-131f-8195-a89329d218ea",
		Version:       "36028797018963968",
	})
	for _, info := range infos {
		writeBG3ModuleShortDesc(&b, info)
	}
	b.WriteString("                    </children>\n")
	b.WriteString("                </node>\n")
	b.WriteString("            </children>\n")
	b.WriteString("        </node>\n")
	b.WriteString("    </region>\n")
	b.WriteString("</save>\n")
	return b.String()
}

func writeBG3ModuleShortDesc(b *strings.Builder, info pakInfo) {
	b.WriteString("                        <node id=\"ModuleShortDesc\">\n")
	b.WriteString("                            <attribute id=\"Folder\" type=\"LSString\" value=\"" + xmlEscape(info.Folder) + "\"/>\n")
	b.WriteString("                            <attribute id=\"MD5\" type=\"LSString\" value=\"" + xmlEscape(info.MD5) + "\"/>\n")
	b.WriteString("                            <attribute id=\"Name\" type=\"LSString\" value=\"" + xmlEscape(info.Name) + "\"/>\n")
	b.WriteString("                            <attribute id=\"PublishHandle\" type=\"uint64\" value=\"" + xmlEscape(firstNonEmpty(info.PublishHandle, "0")) + "\"/>\n")
	b.WriteString("                            <attribute id=\"UUID\" type=\"guid\" value=\"" + xmlEscape(info.UUID) + "\"/>\n")
	b.WriteString("                            <attribute id=\"Version64\" type=\"int64\" value=\"" + xmlEscape(firstNonEmpty(info.Version, "1")) + "\"/>\n")
	b.WriteString("                        </node>\n")
}

func xmlEscape(value string) string {
	var b strings.Builder
	if err := xml.EscapeText(&b, []byte(strings.TrimSpace(value))); err != nil {
		return ""
	}
	return b.String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
