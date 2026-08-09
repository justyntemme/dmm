package dazip

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	ModType      = "dazip"
	AddInsXMLRel = "Settings/AddIns.xml"
)

const daModuleERFSuffix = "_module.erf"

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:      "modtype-dazip",
		Name:    "Vortex DAZIP Mod Type",
		Kind:    sdk.ExtensionKindFramework,
		Version: "1.0.0-dmm.1",
		BuildID: "first-party-go",
		Register: func(r sdk.Registrar) {
			r.RegisterSource(sdk.SourceRef{
				Name: "Vortex modtype-dazip source",
				URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/modtype-dazip/src",
			})
			r.RegisterExtensionToDo(sdk.ExtensionToDoSpec{
				ID:      "dazip-outer-submodule-runtime",
				Name:    "DAZIP nested submodule extraction",
				Trigger: "install",
				Status:  sdk.CapabilityStatusBlocked,
				Message: "DMM implements Vortex dazipInner planning for extracted DAZIP contents, but Vortex dazipOuter needs nested .dazip submodule extraction before archives containing DAZIP files can run automatically.",
			})
			r.RegisterExtensionToDo(sdk.ExtensionToDoSpec{
				ID:      "dazip-migration-runtime",
				Name:    "DAZIP migration purge runtime",
				Trigger: "migration",
				Status:  sdk.CapabilityStatusBlocked,
				Message: "Vortex modtype-dazip can purge historical DA2 DAZIP deployments during migration. DMM has no generic migration/purge-mods-in-path runtime yet.",
			})
		},
	}
}

func RegisterModType(r sdk.Registrar, targetRootID string) {
	r.RegisterModType(installplan.ModTypeSpec{ID: ModType, TargetRootID: targetRootID})
}

func InnerInstaller(id, targetRootID string, priority int) installplan.InstallerSpec {
	return installplan.InstallerSpec{
		ID:                strings.TrimSpace(id),
		VortexInstallerID: "dazipInner",
		Priority:          priority,
		ModType:           ModType,
		NameSource:        installplan.NameSourceArchive,
		TargetRootID:      strings.TrimSpace(targetRootID),
		CustomMatch:       MatchInner,
		CustomBuild:       BuildInner,
		InstructionMode:   installplan.InstructionCustom,
	}
}

func OuterInstallerBlocked(id string, priority int) installplan.InstallerSpec {
	reason := "Vortex treats .dazip files inside an archive as nested submodules. DMM needs nested archive extraction/submodule runtime before this outer DAZIP installer can run."
	return installplan.InstallerSpec{
		ID:                strings.TrimSpace(id),
		VortexInstallerID: "dazipOuter",
		Priority:          priority,
		ModType:           ModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       MatchOuter,
		InstructionMode:   installplan.InstructionUnsupported,
		UnsupportedReason: reason,
		Status:            sdk.CapabilityStatusBlocked,
		Message:           reason,
	}
}

func MatchOuter(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil {
		return false
	}
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), ".dazip") {
			return true
		}
	}
	return false
}

func MatchInner(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil {
		return false
	}
	return innerSupport(files).supported
}

func BuildInner(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	support := innerSupport(files)
	if !support.supported {
		return installplan.Plan{}, installplan.Unsupported("DAZIP inner installer requires a contents folder and a manifest.xml with no files outside the manifest wrapper")
	}
	modName := dazipModName(files)
	plan := installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-dazip-inner",
			Path:   support.manifest,
			Reason: "Vortex dazipInner matched contents plus manifest.xml",
		}},
	}
	for _, file := range files {
		if file == "" {
			continue
		}
		dest := dazipDestination(file, support.manifest, support.basePath, modName)
		if dest == "" {
			continue
		}
		targetRel := filepath.ToSlash(filepath.Join(input.TargetRoot, dest))
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: targetRel,
			TargetRoot:      input.TargetRootID,
			TargetRelative:  targetRel,
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New("DAZIP inner installer matched but produced no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

type innerSupportResult struct {
	supported bool
	manifest  string
	basePath  string
}

func innerSupport(files []string) innerSupportResult {
	hasContents := false
	var manifests []string
	for _, file := range files {
		segments := strings.Split(filepath.ToSlash(file), "/")
		for _, segment := range segments {
			if strings.EqualFold(segment, "contents") {
				hasContents = true
				break
			}
		}
		if strings.EqualFold(filepath.Base(file), "manifest.xml") {
			manifests = append(manifests, filepath.ToSlash(file))
		}
	}
	if !hasContents || len(manifests) == 0 {
		return innerSupportResult{}
	}
	sort.SliceStable(manifests, func(i, j int) bool {
		left := strings.Count(manifests[i], "/")
		right := strings.Count(manifests[j], "/")
		if left != right {
			return left < right
		}
		return manifests[i] < manifests[j]
	})
	shortest := manifests[0]
	basePath := filepath.ToSlash(filepath.Dir(shortest))
	if basePath == "." {
		basePath = ""
	}
	if basePath != "" {
		prefix := strings.ToLower(basePath) + "/"
		for _, file := range files {
			if !strings.HasPrefix(strings.ToLower(filepath.ToSlash(file)), prefix) {
				return innerSupportResult{}
			}
		}
	}
	return innerSupportResult{supported: true, manifest: shortest, basePath: basePath}
}

func dazipModName(files []string) string {
	for _, file := range files {
		segments := strings.Split(filepath.ToSlash(file), "/")
		for idx := 0; idx < len(segments)-1; idx++ {
			if strings.EqualFold(segments[idx], "contents") && idx+2 < len(segments) && strings.EqualFold(segments[idx+1], "addins") {
				return sanitizeSegment(segments[idx+2])
			}
		}
	}
	for _, file := range files {
		base := filepath.Base(file)
		if strings.Contains(base, daModuleERFSuffix) {
			return sanitizeSegment(strings.Replace(base, daModuleERFSuffix, "", 1))
		}
	}
	return ""
}

func dazipDestination(file, manifest, basePath, modName string) string {
	file = filepath.ToSlash(strings.Trim(file, "/"))
	if file == "" {
		return ""
	}
	if file == manifest {
		if modName != "" {
			return filepath.ToSlash(filepath.Join("addins", modName, file))
		}
		return file
	}
	if basePath != "" && strings.HasPrefix(strings.ToLower(file), strings.ToLower(basePath)+"/") {
		file = strings.TrimPrefix(file[len(basePath):], "/")
	}
	parts := strings.Split(file, "/")
	if len(parts) > 0 && strings.EqualFold(parts[0], "contents") {
		parts = parts[1:]
	}
	return filepath.ToSlash(filepath.Join(parts...))
}

func WillDeployAddInsXML(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	type manifestSource struct {
		path       string
		targetRoot string
	}
	var manifests []manifestSource
	for _, mod := range input.Mods {
		if !mod.Enabled || !strings.EqualFold(mod.ModType, ModType) {
			continue
		}
		for _, mapping := range input.Mappings {
			if mapping.InstalledModID != mod.ID || !strings.EqualFold(filepath.Base(mapping.TargetRelative), "manifest.xml") {
				continue
			}
			if strings.TrimSpace(mapping.SourcePath) == "" || strings.TrimSpace(mapping.TargetRoot) == "" {
				continue
			}
			manifests = append(manifests, manifestSource{path: mapping.SourcePath, targetRoot: mapping.TargetRoot})
		}
	}
	if len(manifests) == 0 {
		return sdk.EventHandlerResult{Messages: []string{"Dragon Age AddIns.xml skipped because no enabled DAZIP manifests were mapped."}}, nil
	}
	targetRoot := manifests[0].targetRoot
	items := map[string]addInItem{}
	existingItems, _ := readAddInItems(filepath.Join(targetRoot, filepath.FromSlash(AddInsXMLRel)))
	for _, item := range existingItems {
		items[item.key()] = item
	}
	for _, manifest := range manifests {
		manifestItems, err := readAddInItems(manifest.path)
		if err != nil {
			return sdk.EventHandlerResult{}, err
		}
		for _, item := range manifestItems {
			items[item.key()] = item
		}
	}
	ordered := make([]addInItem, 0, len(items))
	for _, item := range items {
		ordered = append(ordered, item)
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].key() < ordered[j].key()
	})
	sourcePath := filepath.Join(input.WorkDir, filepath.FromSlash(AddInsXMLRel))
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if err := os.WriteFile(sourcePath, renderAddInsXML(ordered), 0o600); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	return sdk.EventHandlerResult{
		Mappings: []deploy.FileMapping{{
			SourcePath:     sourcePath,
			TargetRoot:     targetRoot,
			TargetRelative: AddInsXMLRel,
			Strategy:       deploy.StrategyCopy,
		}},
		Messages: []string{"Dragon Age AddIns.xml generated from enabled DAZIP manifests."},
	}, nil
}

type addInItem struct {
	XMLName xml.Name   `xml:"AddInItem"`
	Attrs   []xml.Attr `xml:",any,attr"`
	Inner   string     `xml:",innerxml"`
}

func (item addInItem) key() string {
	var parts []string
	for _, attr := range item.Attrs {
		parts = append(parts, strings.ToLower(attr.Name.Local)+"="+attr.Value)
	}
	sort.Strings(parts)
	inner := strings.Join(strings.Fields(item.Inner), " ")
	return strings.Join(parts, "\x00") + "\x00" + inner
}

type addInsList struct {
	XMLName xml.Name    `xml:"AddInsList"`
	Items   []addInItem `xml:"AddInItem"`
}

func readAddInItems(path string) ([]addInItem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	decoder.Strict = false
	var out []addInItem
	for {
		token, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		start, ok := token.(xml.StartElement)
		if !ok || !strings.EqualFold(start.Name.Local, "AddInItem") {
			continue
		}
		var item addInItem
		if err := decoder.DecodeElement(&item, &start); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func renderAddInsXML(items []addInItem) []byte {
	data, err := xml.MarshalIndent(addInsList{Items: items}, "", "  ")
	if err != nil {
		return []byte(`<?xml version="1.0" encoding="UTF-8"?><AddInsList></AddInsList>`)
	}
	return append([]byte(xml.Header), data...)
}

func sanitizeSegment(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `/\`)
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, "\\", "_")
	return strings.TrimSpace(value)
}
