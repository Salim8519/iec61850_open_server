package archive

import (
	"encoding/binary"
	"time"
)

// EntryName is the only member of a project archive.
const EntryName = "project.json"

// The header fields Python's zipfile.ZipFile.writestr(name, bytes) writes into a seekable
// stream with ZIP_DEFLATED on Windows: versions 20, create_system 0, flag bits 0 (no data
// descriptor, no UTF-8 flag), external_attr 0o600 << 16, no extra field, no comment, no ZIP64
// (a project is at most 100 MiB, far below Python's ZIP64 threshold).
const (
	zipVersion     = 20
	createSystem   = 0
	methodDeflated = 8
	externalAttr   = 0o600 << 16
	localHeaderLen = 30
	centralLen     = 46
	endRecordLen   = 22
)

// dosDateTime packs t like ZipInfo.FileHeader: date_time = time.localtime()[:6], seconds / 2.
func dosDateTime(t time.Time) (date, clock uint16) {
	local := t.In(time.Local)
	date = uint16((local.Year()-1980)<<9 | int(local.Month())<<5 | local.Day())
	clock = uint16(local.Hour()<<11 | local.Minute()<<5 | local.Second()/2)
	return date, clock
}

// zipEntry describes the single stored member.
type zipEntry struct {
	date, clock uint16
	crc         uint32
	compressed  uint32
	size        uint32
}

// appendLocalHeader writes structFileHeader "<4s2B4HL2L2H" and the file name.
func appendLocalHeader(dst []byte, e zipEntry) []byte {
	dst = binary.LittleEndian.AppendUint32(dst, 0x04034b50)
	dst = append(dst, zipVersion, 0) // extract_version, reserved
	dst = binary.LittleEndian.AppendUint16(dst, 0)
	dst = binary.LittleEndian.AppendUint16(dst, methodDeflated)
	dst = binary.LittleEndian.AppendUint16(dst, e.clock)
	dst = binary.LittleEndian.AppendUint16(dst, e.date)
	dst = binary.LittleEndian.AppendUint32(dst, e.crc)
	dst = binary.LittleEndian.AppendUint32(dst, e.compressed)
	dst = binary.LittleEndian.AppendUint32(dst, e.size)
	dst = binary.LittleEndian.AppendUint16(dst, uint16(len(EntryName)))
	dst = binary.LittleEndian.AppendUint16(dst, 0)
	return append(dst, EntryName...)
}

// appendDirectory writes the central directory record (structCentralDir "<4s4B4HL2L5H2L")
// for an entry at offset 0 and the end record (structEndArchive "<4s4H2LH").
func appendDirectory(dst []byte, e zipEntry) []byte {
	start := len(dst)
	dst = binary.LittleEndian.AppendUint32(dst, 0x02014b50)
	dst = append(dst, zipVersion, createSystem, zipVersion, 0)
	dst = binary.LittleEndian.AppendUint16(dst, 0) // flag bits
	dst = binary.LittleEndian.AppendUint16(dst, methodDeflated)
	dst = binary.LittleEndian.AppendUint16(dst, e.clock)
	dst = binary.LittleEndian.AppendUint16(dst, e.date)
	dst = binary.LittleEndian.AppendUint32(dst, e.crc)
	dst = binary.LittleEndian.AppendUint32(dst, e.compressed)
	dst = binary.LittleEndian.AppendUint32(dst, e.size)
	dst = binary.LittleEndian.AppendUint16(dst, uint16(len(EntryName)))
	dst = binary.LittleEndian.AppendUint16(dst, 0) // extra
	dst = binary.LittleEndian.AppendUint16(dst, 0) // comment
	dst = binary.LittleEndian.AppendUint16(dst, 0) // disk number start
	dst = binary.LittleEndian.AppendUint16(dst, 0) // internal attributes
	dst = binary.LittleEndian.AppendUint32(dst, externalAttr)
	dst = binary.LittleEndian.AppendUint32(dst, 0) // local header offset
	dst = append(dst, EntryName...)
	directorySize := uint32(len(dst) - start)
	dst = binary.LittleEndian.AppendUint32(dst, 0x06054b50)
	dst = binary.LittleEndian.AppendUint16(dst, 0)
	dst = binary.LittleEndian.AppendUint16(dst, 0)
	dst = binary.LittleEndian.AppendUint16(dst, 1)
	dst = binary.LittleEndian.AppendUint16(dst, 1)
	dst = binary.LittleEndian.AppendUint32(dst, directorySize)
	dst = binary.LittleEndian.AppendUint32(dst, uint32(start))
	return binary.LittleEndian.AppendUint16(dst, 0)
}

// finalBlock is an empty final stored deflate block. Every segment ends with a sync flush
// (byte aligned, not final), so appending these bytes closes the concatenated stream.
var finalBlock = []byte{0x01, 0x00, 0x00, 0xff, 0xff}

// assemble builds the archive bytes from the compressed segments of project.json.
func assemble(segments [][]byte, crc uint32, size int64, at time.Time) []byte {
	compressed := len(finalBlock)
	for _, segment := range segments {
		compressed += len(segment)
	}
	date, clock := dosDateTime(at)
	entry := zipEntry{date: date, clock: clock, crc: crc, compressed: uint32(compressed), size: uint32(size)}
	out := make([]byte, 0, localHeaderLen+len(EntryName)*2+compressed+centralLen+endRecordLen)
	out = appendLocalHeader(out, entry)
	for _, segment := range segments {
		out = append(out, segment...)
	}
	out = append(out, finalBlock...)
	return appendDirectory(out, entry)
}
