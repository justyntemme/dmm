package gamebryoplugin

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"strings"
)

const (
	flagMaster             = 0x00000001
	flagLight              = 0x00000200
	flagStarfieldLight     = 0x00000100
	flagMedium             = 0x00000400
	flagStarfieldBlueprint = 0x00000800

	tagTES4 = 0x34534554
	tagHEDR = 0x52444548
	tagMAST = 0x5453414d
	tagXXXX = 0x58585858
)

type Info struct {
	IsMaster    bool
	IsLight     bool
	IsMedium    bool
	IsBlueprint bool
	Masters     []string
}

func ReadHeader(path, gameID string) (Info, error) {
	file, err := os.Open(path)
	if err != nil {
		return Info{}, err
	}
	defer file.Close()

	header := make([]byte, 24)
	if n, err := file.ReadAt(header, 0); err != nil || n < len(header) {
		if err == nil {
			err = errors.New("file incomplete")
		}
		return Info{}, err
	}
	if binary.LittleEndian.Uint32(header[0:4]) != tagTES4 {
		return Info{}, errors.New("invalid Gamebryo plugin header")
	}
	dataSize := binary.LittleEndian.Uint32(header[4:8])
	if dataSize > 16*1024*1024 {
		return Info{}, errors.New("Gamebryo plugin header is too large")
	}
	total := int(24 + dataSize)
	buf := make([]byte, total)
	copy(buf, header)
	if dataSize > 0 {
		if n, err := file.ReadAt(buf[24:], 24); err != nil || n < int(dataSize) {
			if err == nil {
				err = errors.New("header subrecords incomplete")
			}
			return Info{}, err
		}
	}

	return parseHeader(buf, gameID)
}

func parseHeader(buf []byte, gameID string) (Info, error) {
	if len(buf) < 24 {
		return Info{}, errors.New("file incomplete")
	}
	if binary.LittleEndian.Uint32(buf[0:4]) != tagTES4 {
		return Info{}, errors.New("invalid Gamebryo plugin header")
	}
	flags := binary.LittleEndian.Uint32(buf[8:12])
	revision := binary.LittleEndian.Uint32(buf[20:24])
	offset := 24
	if revision == tagHEDR {
		offset = 20
	}
	dataSize := int(binary.LittleEndian.Uint32(buf[4:8]))
	dataEnd := offset + dataSize
	if dataEnd > len(buf) {
		dataEnd = len(buf)
	}

	info := Info{
		IsMaster: (flags & flagMaster) != 0,
		IsMedium: (flags & flagMedium) != 0,
	}
	if strings.EqualFold(strings.TrimSpace(gameID), "starfield") {
		info.IsLight = (flags & flagStarfieldLight) != 0
		info.IsBlueprint = (flags & flagStarfieldBlueprint) != 0
	} else {
		info.IsLight = (flags & flagLight) != 0
	}

	sizeOverride := 0
	for offset+6 <= dataEnd {
		tag := binary.LittleEndian.Uint32(buf[offset : offset+4])
		subSize := int(binary.LittleEndian.Uint16(buf[offset+4 : offset+6]))
		offset += 6
		if tag == tagXXXX {
			if subSize != 4 || offset+4 > dataEnd {
				return Info{}, errors.New("invalid XXXX subrecord")
			}
			sizeOverride = int(binary.LittleEndian.Uint32(buf[offset : offset+4]))
			offset += 4
			continue
		}
		payloadSize := subSize
		if sizeOverride > 0 {
			payloadSize = sizeOverride
			sizeOverride = 0
		}
		if payloadSize < 0 || offset+payloadSize > dataEnd {
			return Info{}, errors.New("subrecord incomplete")
		}
		if tag == tagMAST && payloadSize > 0 {
			info.Masters = append(info.Masters, readNullTerminatedASCII(buf[offset:offset+payloadSize]))
		}
		offset += payloadSize
	}
	return info, nil
}

func readNullTerminatedASCII(buf []byte) string {
	if idx := bytes.IndexByte(buf, 0); idx >= 0 {
		buf = buf[:idx]
	}
	return strings.TrimSpace(string(buf))
}
