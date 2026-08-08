package gzs

import (
	"encoding/binary"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-faster/city"
)

const (
	metaFlag     uint64 = 0x4000000000000
	hashMask     uint64 = 0x3FFFFFFFFFFFF
	cityHashSeed uint64 = 0x9ae16a3b2f90404f
)

var fileExtensions = []string{
	"1.ftexs", "1.nav2", "2.ftexs", "3.ftexs", "4.ftexs", "5.ftexs", "6.ftexs",
	"ag.evf", "aia", "aib", "aibc", "aig", "aigc", "aim", "aip", "ait", "atsh",
	"bnd", "bnk", "cc.evf", "clo", "csnav", "dat", "des", "dnav", "dnav2",
	"eng.lng", "ese", "evb", "evf", "fag", "fage", "fago", "fagp", "fagx",
	"fclo", "fcnp", "fcnpx", "fdes", "fdmg", "ffnt", "fmdl", "fmdlb", "fmtt",
	"fnt", "fova", "fox", "fox2", "fpk", "fpkd", "fpkl", "frdv", "fre.lng",
	"frig", "frt", "fsd", "fsm", "fsml", "fsop", "fstb", "ftex", "fv2",
	"fx.evf", "fxp", "gani", "geom", "ger.lng", "gpfp", "grxla", "grxoc",
	"gskl", "htre", "info", "ita.lng", "jpn.lng", "json", "lad", "ladb",
	"lani", "las", "lba", "lng", "lpsh", "lua", "mas", "mbl", "mog", "mtar",
	"mtl", "nav2", "nta", "obr", "obrb", "param", "parts", "path", "pftxs",
	"ph", "phep", "phsd", "por.lng", "qar", "rbs", "rdb", "rdf", "rnav",
	"rus.lng", "sad", "sand", "sani", "sbp", "sd.evf", "sdf", "sim", "simep",
	"snav", "spa.lng", "spch", "sub", "subp", "tgt", "tre2", "txt", "uia",
	"uif", "uig", "uigb", "uil", "uilb", "utxl", "veh", "vfx", "vfxbin",
	"vfxdb", "vnav", "vo.evf", "vpc", "wem", "wmv", "xml",
}

var extensionHashes = func() map[string]uint64 {
	out := make(map[string]uint64, len(fileExtensions))
	for _, ext := range fileExtensions {
		out[ext] = HashFileName(ext, false) & 0x1FFF
	}
	return out
}()

func HashFileName(text string, removeExtension bool) uint64 {
	if removeExtension {
		if idx := strings.Index(text, "."); idx != -1 {
			text = text[:idx]
		}
	}
	meta := false
	const assetsPrefix = "/Assets/"
	if strings.HasPrefix(text, assetsPrefix) {
		text = strings.TrimPrefix(text, assetsPrefix)
		if strings.HasPrefix(text, "tpptest") {
			meta = true
		}
	} else {
		meta = true
	}
	text = strings.TrimLeft(text, "/")
	hash := city.Hash64WithSeeds([]byte(text), cityHashSeed, reverseTailSeed(text)) & hashMask
	if meta {
		hash |= metaFlag
	}
	return hash
}

func HashFileNameWithExtension(filePath string) uint64 {
	filePath = ToQARPath(filePath)
	if !strings.Contains(strings.TrimPrefix(filePath, "/"), "/") {
		name := strings.TrimPrefix(filePath, "/")
		idx := strings.Index(name, ".")
		if idx > 0 {
			if parsed, err := strconv.ParseUint(name[:idx], 16, 64); err == nil {
				typeID := extensionHashes[name[idx+1:]]
				return (typeID << 51) | parsed
			}
		}
	}
	hashable := filePath
	extension := ""
	if idx := strings.Index(filePath, "."); idx != -1 {
		hashable = filePath[:idx]
		extension = filePath[idx+1:]
	}
	typeID := extensionHashes[extension]
	return (typeID << 51) | HashFileName(hashable, false)
}

func ToQARPath(path string) string {
	return "/" + strings.TrimLeft(filepath.ToSlash(path), "/")
}

func NormalizeQARPath(path string) string {
	return strings.TrimLeft(filepath.ToSlash(path), "/")
}

func reverseTailSeed(text string) uint64 {
	var seed [8]byte
	for i, j := len(text)-1, 0; i >= 0 && j < len(seed); i, j = i-1, j+1 {
		seed[j] = text[i]
	}
	return binary.LittleEndian.Uint64(seed[:])
}
