package metalgearsolidvtpp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gzs"
)

const (
	snakeBiteGeneratedZeroModID = "metalgearsolidvtpp-snakebite-00"
	snakeBiteGeneratedOneModID  = "metalgearsolidvtpp-snakebite-01"
	snakeBiteZeroArchiveRel     = "master/0/00.dat"
	snakeBiteOneArchiveRel      = "master/0/01.dat"
	snakeBiteGeneratedQARFlags  = 3150048
	minSnakeBiteMetadataVersion = "0.8.0.0"
)

type snakeBiteDeployPackage struct {
	mod      sdk.DeploymentMod
	metadata snakeBiteMetadataProbe
}

type snakeBiteQARSource struct {
	name string
	path string
}

func willDeploySnakeBitePackages(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	input.ReportProgress("Scanning enabled SnakeBite packages", 1, 1)
	packages, err := enabledSnakeBitePackages(input)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if len(packages) == 0 {
		return sdk.EventHandlerResult{Messages: []string{"MGSV SnakeBite packed archive deployment skipped because this profile has no enabled SnakeBite packages."}}, nil
	}
	compatMessages, err := validateSnakeBitePackageCompatibility(input, packages)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if conflicts := snakeBitePackageConflicts(packages); len(conflicts) > 0 {
		return sdk.EventHandlerResult{}, snakeBiteConflictError(conflicts)
	}
	mappings, messages, err := buildSnakeBiteArchives(ctx, input, packages)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	messages = append(compatMessages, messages...)
	return sdk.EventHandlerResult{
		Mappings: mappings,
		Messages: messages,
	}, nil
}

func enabledSnakeBitePackages(input sdk.EventHandlerInput) ([]snakeBiteDeployPackage, error) {
	var packages []snakeBiteDeployPackage
	for _, mod := range input.Mods {
		if !mod.Enabled || !strings.EqualFold(strings.TrimSpace(mod.ModType), snakeBiteModType) {
			continue
		}
		if missing := missingSnakeBiteStagedPackageFiles(mod.StagingPath); len(missing) > 0 {
			return nil, errors.New("MGSV SnakeBite package " + mod.Name + " is missing staged package files: " + strings.Join(missing, ", "))
		}
		metadata, err := readSnakeBiteMetadata(filepath.Join(mod.StagingPath, "metadata.xml"))
		if err != nil {
			return nil, err
		}
		packages = append(packages, snakeBiteDeployPackage{mod: mod, metadata: metadata})
	}
	sort.SliceStable(packages, func(i, j int) bool {
		if packages[i].mod.Priority != packages[j].mod.Priority {
			return packages[i].mod.Priority > packages[j].mod.Priority
		}
		return packages[i].mod.ID > packages[j].mod.ID
	})
	return packages, nil
}

func validateSnakeBitePackageCompatibility(input sdk.EventHandlerInput, packages []snakeBiteDeployPackage) ([]string, error) {
	var messages []string
	installedVersion, versionSource, versionErr := installedMGSVVersion(input.GamePath)
	for _, pkg := range packages {
		name := snakeBitePackageName(pkg)
		sbVersion := strings.TrimSpace(pkg.metadata.SBVersion.Version)
		if sbVersion != "" && compareDottedVersions(sbVersion, minSnakeBiteMetadataVersion) < 0 {
			return nil, fmt.Errorf("%s was created with SnakeBite metadata %s; DMM supports SnakeBite package metadata %s or newer. Download an updated package before installing", name, sbVersion, minSnakeBiteMetadataVersion)
		}
		requiredMGSVersion := strings.TrimSpace(pkg.metadata.MGSVersion.Version)
		if requiredMGSVersion == "" || compareDottedVersions(requiredMGSVersion, "0.0.0.0") == 0 {
			continue
		}
		if versionErr != nil {
			return nil, fmt.Errorf("%s requires MGSV version %s, but DMM could not read %s: %w", name, requiredMGSVersion, filepath.ToSlash(filepath.Join(input.GamePath, "mgsvtpp.exe")), versionErr)
		}
		if installedVersion == "" {
			return nil, fmt.Errorf("%s requires MGSV version %s, but DMM could not determine the installed MGSV version", name, requiredMGSVersion)
		}
		switch compareDottedVersions(installedVersion, requiredMGSVersion) {
		case -1:
			return nil, fmt.Errorf("%s requires MGSV version %s, but %s reports %s. Update MGSV before installing this package", name, requiredMGSVersion, versionSource, installedVersion)
		case 1:
			messages = append(messages, fmt.Sprintf("%s targets older MGSV version %s; installed %s reports %s.", name, requiredMGSVersion, versionSource, installedVersion))
		}
	}
	return messages, nil
}

func installedMGSVVersion(gamePath string) (version string, source string, err error) {
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return "", "mgsvtpp.exe", nil
	}
	result, err := gameVersion(context.Background(), sdk.GameVersionInput{GamePath: gamePath})
	if err != nil {
		return "", "mgsvtpp.exe", err
	}
	return strings.TrimSpace(result.Version), strings.TrimSpace(result.Source), nil
}

func buildSnakeBiteArchives(ctx context.Context, input sdk.EventHandlerInput, packages []snakeBiteDeployPackage) ([]deploy.FileMapping, []string, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	totalSteps := len(packages) + 6
	input.ReportProgress("Resolving MGSV base archives", 1, totalSteps)
	baseZeroPath, restoreZeroPath, err := snakeBiteBaseArchive(input, snakeBiteZeroArchiveRel)
	if err != nil {
		return nil, nil, err
	}
	baseOnePath, restoreOnePath, err := snakeBiteBaseArchive(input, snakeBiteOneArchiveRel)
	if err != nil {
		return nil, nil, err
	}
	input.ReportProgress("Reading MGSV base archive "+snakeBiteZeroArchiveRel, 2, totalSteps)
	zeroArchive, err := gzs.ReadQAR(baseZeroPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read MGSV base 00.dat: %w", err)
	}
	input.ReportProgress("Reading MGSV base archive "+snakeBiteOneArchiveRel, 3, totalSteps)
	oneArchive, err := gzs.ReadQAR(baseOnePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read MGSV base 01.dat: %w", err)
	}
	zeroEntries := snakeBiteQAREntryMap(zeroArchive.Entries)
	oneEntries := snakeBiteQAREntryMap(oneArchive.Entries)
	input.ReportProgress("Preparing MGSV SnakeBite base archive split", 4, totalSteps)
	movedSystemFiles, err := snakeBiteMoveSystemEntries(baseZeroPath, zeroEntries, oneEntries, snakeBiteModQARHashes(packages))
	if err != nil {
		return nil, nil, err
	}
	sources := snakeBiteArchiveSources(input, baseZeroPath, baseOnePath)
	mergedFPKs := map[uint64]struct{}{}
	var messages []string
	for index, pkg := range packages {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		input.ReportProgress("Merging SnakeBite package "+pkg.mod.Name, index+5, totalSteps)
		pkgMessages, err := applySnakeBitePackage(input, pkg, sources, zeroEntries, mergedFPKs)
		if err != nil {
			return nil, nil, err
		}
		messages = append(messages, pkgMessages...)
	}
	input.ReportProgress("Writing generated MGSV archive "+snakeBiteZeroArchiveRel, totalSteps-1, totalSteps)
	generatedZeroPath := snakeBiteGeneratedArchivePath(input, snakeBiteZeroArchiveRel)
	if err := gzs.WriteQAR(generatedZeroPath, gzs.QARFile{
		Flags:   snakeBiteGeneratedQARFlags,
		Version: zeroArchive.Version,
		Entries: snakeBiteQAREntries(zeroEntries),
	}); err != nil {
		return nil, nil, fmt.Errorf("write generated MGSV 00.dat: %w", err)
	}
	mappings := []deploy.FileMapping{
		snakeBiteGeneratedArchiveMapping(generatedZeroPath, restoreZeroPath, snakeBiteZeroArchiveRel, snakeBiteGeneratedZeroModID),
	}
	if movedSystemFiles > 0 {
		input.ReportProgress("Writing generated MGSV archive "+snakeBiteOneArchiveRel, totalSteps, totalSteps)
		generatedOnePath := snakeBiteGeneratedArchivePath(input, snakeBiteOneArchiveRel)
		if err := gzs.WriteQAR(generatedOnePath, gzs.QARFile{
			Flags:   snakeBiteGeneratedQARFlags,
			Version: oneArchive.Version,
			Entries: snakeBiteQAREntries(oneEntries),
		}); err != nil {
			return nil, nil, fmt.Errorf("write generated MGSV 01.dat: %w", err)
		}
		mappings = append(mappings, snakeBiteGeneratedArchiveMapping(generatedOnePath, restoreOnePath, snakeBiteOneArchiveRel, snakeBiteGeneratedOneModID))
		messages = append(messages, fmt.Sprintf("MGSV SnakeBite moved %d base system file(s) from %s into generated %s.", movedSystemFiles, snakeBiteZeroArchiveRel, snakeBiteOneArchiveRel))
	} else {
		input.ReportProgress("Finalizing MGSV SnakeBite archive plan", totalSteps, totalSteps)
	}
	messages = append(messages, fmt.Sprintf("MGSV SnakeBite generated %s from %d enabled package(s).", snakeBiteZeroArchiveRel, len(packages)))
	return mappings, messages, nil
}

func snakeBiteGeneratedArchivePath(input sdk.EventHandlerInput, archiveRel string) string {
	return filepath.Join(input.WorkDir, "metalgearsolidvtpp-snakebite", filepath.FromSlash(archiveRel))
}

func snakeBiteGeneratedArchiveMapping(sourcePath, restorePath, targetRel, modID string) deploy.FileMapping {
	return deploy.FileMapping{
		SourcePath:     sourcePath,
		RestorePath:    restorePath,
		TargetRelative: targetRel,
		TargetPolicy:   deploy.TargetPolicyPatchExisting,
		Strategy:       deploy.StrategyCopy,
		InstalledModID: 0,
		ModID:          modID,
		Priority:       -1,
		ChecksumSHA256: "",
		SourceRelative: "",
		TargetRoot:     "",
		Catalog:        "dmm",
	}
}

func snakeBiteQAREntryMap(entries []gzs.QAREntry) map[uint64]gzs.QAREntry {
	out := make(map[uint64]gzs.QAREntry, len(entries))
	for _, entry := range entries {
		out[entry.Hash] = entry
	}
	return out
}

func snakeBiteQAREntries(entries map[uint64]gzs.QAREntry) []gzs.QAREntry {
	out := make([]gzs.QAREntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Hash < out[j].Hash
	})
	return out
}

func snakeBiteModQARHashes(packages []snakeBiteDeployPackage) map[uint64]struct{} {
	hashes := map[uint64]struct{}{}
	for _, pkg := range packages {
		for _, entry := range pkg.metadata.QarEntries.Entries {
			qpath := gzs.ToQARPath(entry.FilePath)
			if qpath == "/" {
				continue
			}
			hashes[gzs.HashFileNameWithExtension(qpath)] = struct{}{}
		}
	}
	return hashes
}

type snakeBiteConflictOwner struct {
	modName string
	path    string
}

func snakeBitePackageName(pkg snakeBiteDeployPackage) string {
	name := strings.TrimSpace(pkg.mod.Name)
	if name == "" {
		name = strings.TrimSpace(pkg.metadata.Name)
	}
	if name == "" {
		name = fmt.Sprintf("installed mod %d", pkg.mod.ID)
	}
	return name
}

func snakeBitePackageConflicts(packages []snakeBiteDeployPackage) []string {
	qarOwners := map[uint64]snakeBiteConflictOwner{}
	fpkOwners := map[string]snakeBiteConflictOwner{}
	seenConflicts := map[string]struct{}{}
	var conflicts []string
	addConflict := func(message string) {
		if _, ok := seenConflicts[message]; ok {
			return
		}
		seenConflicts[message] = struct{}{}
		conflicts = append(conflicts, message)
	}
	for _, pkg := range packages {
		modName := snakeBitePackageName(pkg)
		for _, entry := range pkg.metadata.QarEntries.Entries {
			qpath := gzs.ToQARPath(entry.FilePath)
			if qpath == "/" || strings.Contains(strings.ToLower(qpath), ".fpk") {
				continue
			}
			hash := gzs.HashFileNameWithExtension(qpath)
			if current, ok := qarOwners[hash]; ok && current.modName != modName {
				addConflict(fmt.Sprintf("%s and %s both modify %s in 00.dat", current.modName, modName, snakeBiteDisplayPath(current.path, qpath)))
				continue
			}
			qarOwners[hash] = snakeBiteConflictOwner{modName: modName, path: qpath}
		}
		for _, entry := range pkg.metadata.FpkEntries.Entries {
			fpkPath := gzs.ToQARPath(entry.FpkFile)
			filePath := gzs.ToQARPath(entry.FilePath)
			if fpkPath == "/" || filePath == "/" {
				continue
			}
			key := fmt.Sprintf("%016x:%016x", gzs.HashFileNameWithExtension(fpkPath), gzs.HashFileNameWithExtension(filePath))
			display := fmt.Sprintf("%s inside %s", filePath, fpkPath)
			if current, ok := fpkOwners[key]; ok && current.modName != modName {
				addConflict(fmt.Sprintf("%s and %s both modify %s", current.modName, modName, snakeBiteDisplayPath(current.path, display)))
				continue
			}
			fpkOwners[key] = snakeBiteConflictOwner{modName: modName, path: display}
		}
	}
	sort.Strings(conflicts)
	return conflicts
}

func snakeBiteConflictMessage(conflicts []string) string {
	if len(conflicts) == 0 {
		return ""
	}
	var message strings.Builder
	message.WriteString("MGSV SnakeBite package conflicts must be resolved before deployment. Disable one of the conflicting mods or move it to another profile, then apply the profile again.")
	message.WriteString(" Conflicts: ")
	limit := len(conflicts)
	if limit > 6 {
		limit = 6
	}
	message.WriteString(strings.Join(conflicts[:limit], "; "))
	if len(conflicts) > limit {
		message.WriteString(fmt.Sprintf("; %d more", len(conflicts)-limit))
	}
	return message.String()
}

func snakeBiteConflictError(conflicts []string) error {
	if len(conflicts) == 0 {
		return nil
	}
	return sdk.BlockingIssuesError{
		Issues: []sdk.BlockingIssue{{
			Kind:    "packed-archive-conflict",
			Title:   "SnakeBite package conflicts",
			Message: snakeBiteConflictMessage(conflicts),
			Details: append([]string(nil), conflicts...),
			Actions: []string{
				"Disable one of each conflicting MGSV mod.",
				"Move conflicting packages into separate profiles if you want alternate loadouts.",
				"Apply the profile again after resolving the conflict.",
			},
		}},
	}
}

func snakeBiteDisplayPath(first, fallback string) string {
	first = strings.TrimSpace(first)
	if first != "" && first != "/" {
		return first
	}
	return fallback
}

func snakeBiteMoveSystemEntries(baseZeroPath string, zeroEntries, oneEntries map[uint64]gzs.QAREntry, modQARHashes map[uint64]struct{}) (int, error) {
	foxpatchHash := gzs.HashFileNameWithExtension("foxpatch.dat")
	moved := 0
	for hash, entry := range zeroEntries {
		if hash == foxpatchHash {
			continue
		}
		if _, ok := modQARHashes[hash]; ok {
			continue
		}
		data, ok, err := gzs.ExtractQAREntryDataByHash(baseZeroPath, hash)
		if err != nil {
			return 0, fmt.Errorf("extract MGSV 00.dat system entry %016x: %w", hash, err)
		}
		if !ok {
			return 0, fmt.Errorf("extract MGSV 00.dat system entry %016x: entry disappeared", hash)
		}
		oneEntries[hash] = gzs.QAREntry{
			Hash:       hash,
			FilePath:   entry.FilePath,
			Compressed: entry.Compressed,
			Data:       data,
		}
		delete(zeroEntries, hash)
		moved++
	}
	return moved, nil
}

func applySnakeBitePackage(input sdk.EventHandlerInput, pkg snakeBiteDeployPackage, sources []snakeBiteQARSource, entries map[uint64]gzs.QAREntry, mergedFPKs map[uint64]struct{}) ([]string, error) {
	var messages []string
	for _, fpkPath := range distinctSnakeBiteFPKs(pkg.metadata) {
		fpkHash := gzs.HashFileNameWithExtension(fpkPath)
		baseFPKData, sourceName, found, err := snakeBiteFindQARData(sources, fpkHash)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		modFPKPath := filepath.Join(pkg.mod.StagingPath, filepath.FromSlash(snakeBitePackageRelative(fpkPath)))
		modFPKData, err := os.ReadFile(modFPKPath)
		if err != nil {
			return nil, err
		}
		merged, err := mergeSnakeBiteFPK(baseFPKData, modFPKData, fpkPath, snakeBiteFPKEntries(pkg.metadata, fpkPath))
		if err != nil {
			return nil, err
		}
		entries[fpkHash] = gzs.QAREntry{
			Hash:       fpkHash,
			FilePath:   gzs.ToQARPath(fpkPath),
			Compressed: true,
			Data:       merged,
		}
		mergedFPKs[fpkHash] = struct{}{}
		messages = append(messages, fmt.Sprintf("MGSV SnakeBite merged %s from %s for %s.", gzs.ToQARPath(fpkPath), sourceName, pkg.mod.Name))
	}
	for _, qarEntry := range pkg.metadata.QarEntries.Entries {
		qpath := gzs.ToQARPath(qarEntry.FilePath)
		if qpath == "/" {
			continue
		}
		hash := gzs.HashFileNameWithExtension(qpath)
		if _, merged := mergedFPKs[hash]; merged && strings.Contains(strings.ToLower(filepath.Ext(qpath)), ".fpk") {
			continue
		}
		rel := snakeBitePackageRelative(qpath)
		if rel == "" {
			return nil, errors.New("MGSV SnakeBite package contains unsafe QAR path " + qarEntry.FilePath)
		}
		data, err := os.ReadFile(filepath.Join(pkg.mod.StagingPath, filepath.FromSlash(rel)))
		if err != nil {
			return nil, err
		}
		entries[hash] = gzs.QAREntry{
			Hash:       hash,
			FilePath:   qpath,
			Compressed: snakeBiteQAREntryCompressed(qpath, qarEntry.Compressed),
			Data:       data,
		}
	}
	return messages, nil
}

func mergeSnakeBiteFPK(baseData, modData []byte, fpkPath string, metadataEntries []snakeBiteMetadataFPKEntry) ([]byte, error) {
	baseArchive, err := readFPKBytes(baseData)
	if err != nil {
		return nil, fmt.Errorf("read base FPK %s: %w", fpkPath, err)
	}
	modArchive, err := readFPKBytes(modData)
	if err != nil {
		return nil, fmt.Errorf("read mod FPK %s: %w", fpkPath, err)
	}
	replacements := make(map[[16]byte]gzs.FPKEntry, len(metadataEntries))
	modEntries := map[[16]byte]gzs.FPKEntry{}
	for _, entry := range modArchive.Entries {
		modEntries[entry.PathMD5] = entry
	}
	for _, metadataEntry := range metadataEntries {
		path := gzs.ToQARPath(metadataEntry.FilePath)
		hash := gzs.FPKPathMD5(path)
		entry, ok := modEntries[hash]
		if !ok {
			return nil, fmt.Errorf("mod FPK %s is missing metadata entry %s", fpkPath, path)
		}
		data, err := gzs.ExportFPKEntryData(entry)
		if err != nil {
			return nil, err
		}
		replacements[hash] = gzs.FPKEntry{
			FilePath: path,
			PathMD5:  hash,
			Data:     data,
		}
	}
	out := gzs.FPKFile{Type: baseArchive.Type}
	for _, entry := range baseArchive.Entries {
		if replacement, ok := replacements[entry.PathMD5]; ok {
			out.Entries = append(out.Entries, replacement)
			delete(replacements, entry.PathMD5)
			continue
		}
		out.Entries = append(out.Entries, entry)
	}
	for _, entry := range replacements {
		out.Entries = append(out.Entries, entry)
	}
	var writer memoryWriteSeeker
	if err := gzs.WriteFPKWriter(&writer, out); err != nil {
		return nil, err
	}
	return writer.Bytes(), nil
}

func readFPKBytes(data []byte) (gzs.FPKFile, error) {
	reader := bytes.NewReader(data)
	return gzs.ReadFPKReader(io.NewSectionReader(reader, 0, int64(len(data))))
}

func snakeBiteBaseArchive(input sdk.EventHandlerInput, archiveRel string) (string, string, error) {
	targetPath := filepath.Join(input.GamePath, filepath.FromSlash(archiveRel))
	if managed, ok := snakeBiteManagedFile(input.ManagedFiles, targetPath); ok && strings.TrimSpace(managed.RestorePath) != "" {
		return managed.RestorePath, managed.RestorePath, nil
	}
	restorePath := filepath.Join(input.WorkDir, "metalgearsolidvtpp-snakebite", "restore", filepath.FromSlash(archiveRel))
	if err := copyFile(targetPath, restorePath); err != nil {
		return "", "", fmt.Errorf("prepare MGSV %s restore copy: %w", archiveRel, err)
	}
	return targetPath, restorePath, nil
}

func snakeBiteManagedFile(files []deploy.AppliedFile, targetPath string) (deploy.AppliedFile, bool) {
	targetPath = filepath.Clean(targetPath)
	for _, file := range files {
		if filepath.Clean(file.TargetPath) == targetPath {
			return file, true
		}
	}
	return deploy.AppliedFile{}, false
}

func snakeBiteArchiveSources(input sdk.EventHandlerInput, baseZeroPath, baseOnePath string) []snakeBiteQARSource {
	var sources []snakeBiteQARSource
	add := func(name, path string) {
		if strings.TrimSpace(path) == "" {
			return
		}
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			sources = append(sources, snakeBiteQARSource{name: name, path: path})
		}
	}
	add("00.dat", baseZeroPath)
	add("01.dat", baseOnePath)
	add("data1.dat", filepath.Join(input.GamePath, "master", "data1.dat"))
	for i := 0; i <= 4; i++ {
		add(fmt.Sprintf("chunk%d.dat", i), filepath.Join(input.GamePath, "master", fmt.Sprintf("chunk%d.dat", i)))
	}
	return sources
}

func snakeBiteFindQARData(sources []snakeBiteQARSource, hash uint64) ([]byte, string, bool, error) {
	for _, source := range sources {
		data, ok, err := gzs.ExtractQAREntryDataByHash(source.path, hash)
		if err != nil {
			return nil, "", false, fmt.Errorf("scan MGSV archive %s: %w", source.name, err)
		}
		if ok {
			return data, source.name, true, nil
		}
	}
	return nil, "", false, nil
}

func distinctSnakeBiteFPKs(metadata snakeBiteMetadataProbe) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, entry := range metadata.FpkEntries.Entries {
		path := gzs.ToQARPath(entry.FpkFile)
		if path == "/" {
			continue
		}
		key := strings.ToLower(path)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

func snakeBiteFPKEntries(metadata snakeBiteMetadataProbe, fpkPath string) []snakeBiteMetadataFPKEntry {
	fpkPath = strings.ToLower(gzs.ToQARPath(fpkPath))
	var out []snakeBiteMetadataFPKEntry
	for _, entry := range metadata.FpkEntries.Entries {
		if strings.ToLower(gzs.ToQARPath(entry.FpkFile)) == fpkPath {
			out = append(out, snakeBiteMetadataFPKEntry{
				FpkFile:     entry.FpkFile,
				FilePath:    entry.FilePath,
				ContentHash: entry.ContentHash,
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return strings.ToLower(out[i].FilePath) < strings.ToLower(out[j].FilePath)
	})
	return out
}

func snakeBiteQAREntryCompressed(path string, metadataCompressed bool) bool {
	if metadataCompressed {
		return true
	}
	return strings.Contains(strings.ToLower(filepath.Ext(path)), ".fpk")
}

func compareDottedVersions(lhs, rhs string) int {
	left := dottedVersionParts(lhs)
	right := dottedVersionParts(rhs)
	if len(left) < len(right) {
		left = append(left, make([]int, len(right)-len(left))...)
	}
	if len(right) < len(left) {
		right = append(right, make([]int, len(left)-len(right))...)
	}
	for i := 0; i < len(left); i++ {
		switch {
		case left[i] < right[i]:
			return -1
		case left[i] > right[i]:
			return 1
		}
	}
	return 0
}

func dottedVersionParts(value string) []int {
	fields := strings.FieldsFunc(strings.TrimSpace(value), func(r rune) bool {
		return r == '.' || r == '-' || r == '_' || r == ' '
	})
	parts := make([]int, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		n := 0
		for _, r := range field {
			if r < '0' || r > '9' {
				break
			}
			n = n*10 + int(r-'0')
		}
		parts = append(parts, n)
	}
	if len(parts) == 0 {
		return []int{0}
	}
	return parts
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
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

type memoryWriteSeeker struct {
	data []byte
	pos  int64
}

func (w *memoryWriteSeeker) Write(p []byte) (int, error) {
	if w.pos < 0 {
		return 0, errors.New("negative write position")
	}
	end := w.pos + int64(len(p))
	if end < w.pos {
		return 0, errors.New("write position overflow")
	}
	if end > int64(len(w.data)) {
		next := make([]byte, end)
		copy(next, w.data)
		w.data = next
	}
	copy(w.data[w.pos:end], p)
	w.pos = end
	return len(p), nil
}

func (w *memoryWriteSeeker) Seek(offset int64, whence int) (int64, error) {
	var next int64
	switch whence {
	case io.SeekStart:
		next = offset
	case io.SeekCurrent:
		next = w.pos + offset
	case io.SeekEnd:
		next = int64(len(w.data)) + offset
	default:
		return 0, errors.New("invalid seek whence")
	}
	if next < 0 {
		return 0, errors.New("negative seek position")
	}
	w.pos = next
	return w.pos, nil
}

func (w *memoryWriteSeeker) Bytes() []byte {
	return append([]byte(nil), w.data...)
}
