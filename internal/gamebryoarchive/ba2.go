package gamebryoarchive

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	ba2Magic        = "BTDX"
	ba2HeaderSize   = 24
	ba2GnrlEntryLen = 36
)

type BA2Archive struct {
	path        string
	version     uint32
	archiveType string
	names       []string
	gnrl        []ba2GeneralEntry
	dx10        []ba2TextureEntry
}

type ba2GeneralEntry struct {
	offset      uint64
	packedLen   uint32
	unpackedLen uint32
}

type ba2TextureEntry struct {
	numChunks  uint8
	height     uint16
	width      uint16
	numMips    uint8
	dxgiFormat uint8
	chunks     []ba2TextureChunk
}

type ba2TextureChunk struct {
	offset      uint64
	packedLen   uint32
	unpackedLen uint32
}

func OpenBA2(path string) (*BA2Archive, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	header := make([]byte, ba2HeaderSize)
	if _, err := io.ReadFull(f, header); err != nil {
		return nil, err
	}
	if string(header[:4]) != ba2Magic {
		return nil, fmt.Errorf("invalid BA2 file: expected magic %q", ba2Magic)
	}
	version := binary.LittleEndian.Uint32(header[4:8])
	typeRaw := strings.TrimRight(string(header[8:12]), "\x00")
	fileCount := binary.LittleEndian.Uint32(header[12:16])
	nameTableOffset := binary.LittleEndian.Uint64(header[16:24])
	if fileCount > 1_000_000 {
		return nil, fmt.Errorf("BA2 file count %d is unreasonable", fileCount)
	}
	out := &BA2Archive{path: path, version: version}
	switch typeRaw {
	case "GNRL":
		out.archiveType = "general"
		out.gnrl, err = readBA2GeneralEntries(f, int(fileCount))
	case "DX10":
		out.archiveType = "dx10"
		out.dx10, err = readBA2DX10Entries(f, int(fileCount))
	default:
		return nil, fmt.Errorf("unknown BA2 type %q", typeRaw)
	}
	if err != nil {
		return nil, err
	}
	names, err := readBA2NameTable(f, int(fileCount), nameTableOffset)
	if err != nil {
		return nil, err
	}
	out.names = names
	return out, nil
}

func (a *BA2Archive) Type() string {
	return "ba2:" + a.archiveType
}

func (a *BA2Archive) Version() uint32 {
	return a.version
}

func (a *BA2Archive) List() []Entry {
	entries := make([]Entry, 0, len(a.names))
	for i, name := range a.names {
		var size uint64
		if a.archiveType == "general" && i < len(a.gnrl) {
			size = uint64(a.gnrl[i].unpackedLen)
		}
		entries = append(entries, Entry{Path: normalizeArchivePath(name), Size: size})
	}
	return entries
}

func (a *BA2Archive) ReadFile(name string) ([]byte, error) {
	idx := -1
	for i, entry := range a.names {
		if archivePathEqual(entry, name) {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, os.ErrNotExist
	}
	f, err := os.Open(a.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	switch a.archiveType {
	case "general":
		return a.readGeneralFile(f, idx)
	case "dx10":
		return a.readDX10File(f, idx)
	default:
		return nil, errors.New("unknown BA2 archive type")
	}
}

func (a *BA2Archive) ExtractAll(outputDir string) error {
	for _, entry := range a.List() {
		body, err := a.ReadFile(entry.Path)
		if err != nil {
			return err
		}
		out, err := safeOutputPath(outputDir, entry.Path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(out, body, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func (a *BA2Archive) readGeneralFile(f *os.File, idx int) ([]byte, error) {
	if idx >= len(a.gnrl) {
		return nil, io.ErrUnexpectedEOF
	}
	entry := a.gnrl[idx]
	readLen := entry.unpackedLen
	compressed := entry.packedLen != 0 && entry.packedLen != entry.unpackedLen
	if compressed {
		readLen = entry.packedLen
	}
	if readLen == 0 {
		return []byte{}, nil
	}
	buf := make([]byte, readLen)
	if _, err := f.ReadAt(buf, int64(entry.offset)); err != nil {
		return nil, err
	}
	if compressed {
		return inflateZlib(buf)
	}
	return buf, nil
}

func (a *BA2Archive) readDX10File(f *os.File, idx int) ([]byte, error) {
	if idx >= len(a.dx10) {
		return nil, io.ErrUnexpectedEOF
	}
	tex := a.dx10[idx]
	out := buildDDSHeader(tex)
	for _, chunk := range tex.chunks {
		readLen := chunk.unpackedLen
		compressed := chunk.packedLen != 0 && chunk.packedLen != chunk.unpackedLen
		if compressed {
			readLen = chunk.packedLen
		}
		buf := make([]byte, readLen)
		if readLen > 0 {
			if _, err := f.ReadAt(buf, int64(chunk.offset)); err != nil {
				return nil, err
			}
		}
		if compressed {
			inflated, err := inflateZlib(buf)
			if err != nil {
				return nil, err
			}
			out = append(out, inflated...)
		} else {
			out = append(out, buf...)
		}
	}
	return out, nil
}

func readBA2GeneralEntries(f *os.File, count int) ([]ba2GeneralEntry, error) {
	buf := make([]byte, count*ba2GnrlEntryLen)
	if _, err := f.ReadAt(buf, ba2HeaderSize); err != nil {
		return nil, err
	}
	entries := make([]ba2GeneralEntry, 0, count)
	for i := 0; i < count; i++ {
		off := i * ba2GnrlEntryLen
		entries = append(entries, ba2GeneralEntry{
			offset:      binary.LittleEndian.Uint64(buf[off+16 : off+24]),
			packedLen:   binary.LittleEndian.Uint32(buf[off+24 : off+28]),
			unpackedLen: binary.LittleEndian.Uint32(buf[off+28 : off+32]),
		})
	}
	return entries, nil
}

func readBA2DX10Entries(f *os.File, count int) ([]ba2TextureEntry, error) {
	maxEntrySize := 24 + 255*24
	bufSize := count * maxEntrySize
	if bufSize > 10*1024*1024 {
		bufSize = 10 * 1024 * 1024
	}
	buf := make([]byte, bufSize)
	n, err := f.ReadAt(buf, ba2HeaderSize)
	if err != nil && err != io.EOF {
		return nil, err
	}
	buf = buf[:n]
	entries := make([]ba2TextureEntry, 0, count)
	off := 0
	for i := 0; i < count; i++ {
		if off+24 > len(buf) {
			return nil, io.ErrUnexpectedEOF
		}
		numChunks := buf[off+0x0d]
		entry := ba2TextureEntry{
			numChunks:  numChunks,
			height:     binary.LittleEndian.Uint16(buf[off+0x10 : off+0x12]),
			width:      binary.LittleEndian.Uint16(buf[off+0x12 : off+0x14]),
			numMips:    buf[off+0x14],
			dxgiFormat: buf[off+0x15],
		}
		off += 24
		for j := 0; j < int(numChunks); j++ {
			if off+24 > len(buf) {
				return nil, io.ErrUnexpectedEOF
			}
			entry.chunks = append(entry.chunks, ba2TextureChunk{
				offset:      binary.LittleEndian.Uint64(buf[off : off+8]),
				packedLen:   binary.LittleEndian.Uint32(buf[off+8 : off+12]),
				unpackedLen: binary.LittleEndian.Uint32(buf[off+12 : off+16]),
			})
			off += 24
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func readBA2NameTable(f *os.File, count int, offset uint64) ([]string, error) {
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if offset > uint64(stat.Size()) {
		return nil, io.ErrUnexpectedEOF
	}
	size := uint64(stat.Size()) - offset
	if size > 64*1024*1024 {
		return nil, fmt.Errorf("BA2 name table is too large: %d bytes", size)
	}
	buf := make([]byte, size)
	if _, err := f.ReadAt(buf, int64(offset)); err != nil && err != io.EOF {
		return nil, err
	}
	names := make([]string, 0, count)
	for off := 0; off+2 <= len(buf) && len(names) < count; {
		nameLen := int(binary.LittleEndian.Uint16(buf[off : off+2]))
		off += 2
		if off+nameLen > len(buf) {
			return nil, io.ErrUnexpectedEOF
		}
		names = append(names, normalizeArchivePath(string(buf[off:off+nameLen])))
		off += nameLen
	}
	if len(names) != count {
		return nil, fmt.Errorf("BA2 name table had %d entries, expected %d", len(names), count)
	}
	return names, nil
}

func buildDDSHeader(tex ba2TextureEntry) []byte {
	const (
		ddsMagic           = 0x20534444
		ddsHeaderSize      = 124
		ddsPixelFormatSize = 32
		ddsHeaderFlags     = 0x00000001 | 0x00000002 | 0x00000004 | 0x00020000
		ddsSurfaceFlags    = 0x00001000 | 0x00400000 | 0x00000008
		ddpfFourCC         = 0x04
		ddpfRGB            = 0x40
		ddpfAlpha          = 0x01
	)
	type format struct {
		fourCC string
		bpp    uint32
		flags  uint32
		rMask  uint32
		gMask  uint32
		bMask  uint32
		aMask  uint32
	}
	formats := map[uint8]format{
		70: {fourCC: "DXT1", bpp: 4},
		71: {fourCC: "DXT3", bpp: 8},
		72: {fourCC: "DXT5", bpp: 8},
		77: {fourCC: "ATI2", bpp: 8},
		87: {bpp: 32, flags: ddpfRGB | ddpfAlpha, rMask: 0x00ff0000, gMask: 0x0000ff00, bMask: 0x000000ff, aMask: 0xff000000},
		61: {bpp: 8, flags: ddpfRGB, rMask: 0xff},
		98: {fourCC: "DX10", bpp: 8},
	}
	info := formats[tex.dxgiFormat]
	buf := make([]byte, 128)
	binary.LittleEndian.PutUint32(buf[0:4], ddsMagic)
	h := 4
	binary.LittleEndian.PutUint32(buf[h:h+4], ddsHeaderSize)
	binary.LittleEndian.PutUint32(buf[h+4:h+8], ddsHeaderFlags)
	binary.LittleEndian.PutUint32(buf[h+8:h+12], uint32(tex.height))
	binary.LittleEndian.PutUint32(buf[h+12:h+16], uint32(tex.width))
	binary.LittleEndian.PutUint32(buf[h+24:h+28], uint32(tex.numMips))
	pf := h + 72
	binary.LittleEndian.PutUint32(buf[pf:pf+4], ddsPixelFormatSize)
	if info.fourCC != "" {
		binary.LittleEndian.PutUint32(buf[pf+4:pf+8], ddpfFourCC)
		copy(buf[pf+8:pf+12], []byte(info.fourCC))
	} else {
		binary.LittleEndian.PutUint32(buf[pf+4:pf+8], info.flags)
		binary.LittleEndian.PutUint32(buf[pf+12:pf+16], info.bpp)
		binary.LittleEndian.PutUint32(buf[pf+16:pf+20], info.rMask)
		binary.LittleEndian.PutUint32(buf[pf+20:pf+24], info.gMask)
		binary.LittleEndian.PutUint32(buf[pf+24:pf+28], info.bMask)
		binary.LittleEndian.PutUint32(buf[pf+28:pf+32], info.aMask)
	}
	binary.LittleEndian.PutUint32(buf[h+104:h+108], ddsSurfaceFlags)
	return buf
}
