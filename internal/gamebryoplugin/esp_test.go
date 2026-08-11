package gamebryoplugin

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestReadHeaderParsesMastersAndFlags(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Example.esp")
	body := gamebryoHeader(flagLight, "Fallout4.esm", "DLCRobot.esm")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := ReadHeader(path, "fallout4")
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsLight || info.IsMedium || info.IsBlueprint {
		t.Fatalf("flags = %+v", info)
	}
	if len(info.Masters) != 2 || info.Masters[0] != "Fallout4.esm" || info.Masters[1] != "DLCRobot.esm" {
		t.Fatalf("masters = %+v", info.Masters)
	}
}

func TestReadHeaderUsesStarfieldFlagLayout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Blueprint.esm")
	body := gamebryoHeader(flagStarfieldLight|flagStarfieldBlueprint|flagMedium, "Starfield.esm")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := ReadHeader(path, "starfield")
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsLight || !info.IsMedium || !info.IsBlueprint {
		t.Fatalf("flags = %+v", info)
	}
}

func TestIsBlueprintPluginMirrorsVortexAPISemantics(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Blueprint.esm")
	body := gamebryoHeader(flagStarfieldBlueprint, "Starfield.esm")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	if !IsBlueprintPlugin(path, "starfield") {
		t.Fatal("expected Starfield blueprint plugin")
	}
	if IsBlueprintPlugin(path, "fallout4") {
		t.Fatal("non-Starfield games must not report blueprint plugins")
	}
	if IsBlueprintPlugin(filepath.Join(t.TempDir(), "missing.esm"), "starfield") {
		t.Fatal("parse errors must report false")
	}
}

func gamebryoHeader(flags uint32, masters ...string) []byte {
	subrecords := []byte{}
	hedr := make([]byte, 6+12)
	binary.LittleEndian.PutUint32(hedr[0:4], tagHEDR)
	binary.LittleEndian.PutUint16(hedr[4:6], 12)
	subrecords = append(subrecords, hedr...)
	for _, master := range masters {
		payload := append([]byte(master), 0)
		record := make([]byte, 6+len(payload))
		binary.LittleEndian.PutUint32(record[0:4], tagMAST)
		binary.LittleEndian.PutUint16(record[4:6], uint16(len(payload)))
		copy(record[6:], payload)
		subrecords = append(subrecords, record...)
	}
	header := make([]byte, 24)
	binary.LittleEndian.PutUint32(header[0:4], tagTES4)
	binary.LittleEndian.PutUint32(header[4:8], uint32(len(subrecords)))
	binary.LittleEndian.PutUint32(header[8:12], flags)
	out := append([]byte{}, header...)
	return append(out, subrecords...)
}
