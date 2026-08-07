package metalgearsolidvtpp

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
)

type snakeBiteMetadataProbe struct {
	XMLName    xml.Name `xml:"ModEntry"`
	QarEntries struct {
		Entries []struct {
			FilePath string `xml:"FilePath,attr"`
		} `xml:"QarEntry"`
	} `xml:"QarEntries"`
	FpkEntries struct {
		Entries []struct {
			FpkFile  string `xml:"FpkFile,attr"`
			FilePath string `xml:"FilePath,attr"`
		} `xml:"FpkEntry"`
	} `xml:"FpkEntries"`
	SBVersion  struct{} `xml:"SBVersion"`
	MGSVersion struct{} `xml:"MGSVersion"`
}

func matchSnakeBitePackage(extractedRoot string) bool {
	extractedRoot = strings.TrimSpace(extractedRoot)
	if extractedRoot == "" {
		return false
	}
	matched := false
	_ = filepath.WalkDir(extractedRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || matched || d.IsDir() {
			return nil
		}
		if strings.EqualFold(d.Name(), "metadata.xml") && isSnakeBiteMetadata(path) {
			matched = true
			return filepath.SkipAll
		}
		return nil
	})
	return matched
}

func isSnakeBiteMetadata(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return false
	}
	var metadata snakeBiteMetadataProbe
	if err := xml.Unmarshal(data, &metadata); err != nil {
		return false
	}
	if metadata.XMLName.Local != "ModEntry" {
		return false
	}
	if len(metadata.QarEntries.Entries) == 0 && len(metadata.FpkEntries.Entries) == 0 {
		return false
	}
	for _, entry := range metadata.QarEntries.Entries {
		if strings.TrimSpace(entry.FilePath) != "" {
			return true
		}
	}
	for _, entry := range metadata.FpkEntries.Entries {
		if strings.TrimSpace(entry.FpkFile) != "" || strings.TrimSpace(entry.FilePath) != "" {
			return true
		}
	}
	return false
}
