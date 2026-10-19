package xmldoc

import (
	"strings"
	"unicode/utf8"
)

// invalidMark replaces bytes that cannot be decoded (undefined code points of an 8-bit
// codec, lone UTF-16 surrogates); it is never valid UTF-8, so the scanner reports
// "not well-formed (invalid token)" at its position as expat does.
const invalidMark = 0xff

// detect returns the document as UTF-8 with a leading byte order mark kept as U+FEFF
// (expat counts it as one column), the offset after the mark, the UTF-16 byte order
// (0 none, 1 little endian, 2 big endian) and whether an odd trailing byte was dropped.
func detect(data []byte) (buf []byte, start int, wide int, oddTail bool) {
	switch {
	case len(data) >= 3 && data[0] == 0xef && data[1] == 0xbb && data[2] == 0xbf:
		return data, 3, 0, false
	case len(data) >= 2 && data[0] == 0xff && data[1] == 0xfe:
		out, odd := fromUTF16(data[2:], false)
		return append([]byte{0xef, 0xbb, 0xbf}, out...), 3, 1, odd
	case len(data) >= 2 && data[0] == 0xfe && data[1] == 0xff:
		out, odd := fromUTF16(data[2:], true)
		return append([]byte{0xef, 0xbb, 0xbf}, out...), 3, 2, odd
	case len(data) >= 2 && data[0] == '<' && data[1] == 0:
		out, odd := fromUTF16(data, false)
		return out, 0, 1, odd
	case len(data) >= 2 && data[0] == 0 && data[1] == '<':
		out, odd := fromUTF16(data, true)
		return out, 0, 2, odd
	}
	return data, 0, 0, false
}

func fromUTF16(data []byte, big bool) ([]byte, bool) {
	out := make([]byte, 0, len(data))
	unit := func(i int) rune {
		if big {
			return rune(data[i])<<8 | rune(data[i+1])
		}
		return rune(data[i+1])<<8 | rune(data[i])
	}
	i := 0
	for ; i+1 < len(data); i += 2 {
		r := unit(i)
		switch {
		case r >= 0xd800 && r < 0xdc00:
			if i+3 < len(data) {
				low := unit(i + 2)
				if low >= 0xdc00 && low < 0xe000 {
					out = utf8.AppendRune(out, 0x10000+(r-0xd800)<<10+(low-0xdc00))
					i += 2
					continue
				}
			}
			out = append(out, invalidMark)
		case r >= 0xdc00 && r < 0xe000:
			out = append(out, invalidMark)
		default:
			out = utf8.AppendRune(out, r)
		}
	}
	return out, i < len(data)
}

// codecChoice is what expat and pyexpat do with a declared encoding name.
type codecChoice struct {
	keep      bool            // same encoding, nothing to do
	incorrect bool            // "encoding specified in XML declaration is incorrect"
	table     func(byte) rune // 8-bit decoding of the bytes after the declaration
	failure   string          // Python exception text that ends the parse (no position)
	unknown   bool            // expat "unknown encoding" at the name
}

func chooseCodec(name string, wide int) codecChoice {
	switch strings.ToUpper(name) {
	case "UTF-8":
		if wide != 0 {
			return codecChoice{incorrect: true}
		}
		return codecChoice{keep: true}
	case "UTF-16":
		if wide != 0 {
			return codecChoice{keep: true}
		}
		return codecChoice{incorrect: true}
	case "UTF-16LE":
		return codecChoice{keep: wide == 1, incorrect: wide != 1}
	case "UTF-16BE":
		return codecChoice{keep: wide == 2, incorrect: wide != 2}
	case "ISO-8859-1":
		if wide != 0 {
			return codecChoice{incorrect: true}
		}
		return codecChoice{table: latin1}
	case "US-ASCII":
		if wide != 0 {
			return codecChoice{incorrect: true}
		}
		return codecChoice{table: asciiOnly}
	}
	return pythonCodec(name)
}

func pythonCodec(name string) codecChoice {
	normal := normalizeEncoding(strings.ToLower(name))
	codec, found := pythonAliases[normal]
	if !found {
		codec, found = pythonAliases[strings.ReplaceAll(normal, ".", "_")]
	}
	switch {
	case !found && (multiByteCodecs[normal] || multiByteCodecs[strings.ReplaceAll(normal, ".", "_")]):
		return codecChoice{failure: "multi-byte encodings are not supported"}
	case !found:
		return codecChoice{failure: "unknown encoding: " + name}
	case codec == "latin_1":
		return codecChoice{table: latin1}
	case codec == "ascii" || codec == "utf_8":
		return codecChoice{table: asciiOnly}
	}
	table := singleByteTablesA[codec]
	if table == nil {
		table = singleByteTablesB[codec]
	}
	return codecChoice{table: func(b byte) rune {
		if b < 0x80 {
			return rune(b)
		}
		return rune(table[b-0x80])
	}}
}

// normalizeEncoding is encodings.normalize_encoding: runs of characters other than ASCII
// letters, digits and '.' become one '_', leading and trailing ones are dropped.
func normalizeEncoding(name string) string {
	var b strings.Builder
	punct := false
	for _, r := range name {
		alnum := r == '.' || (r < 0x80 && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'))
		if !alnum && r >= 0x80 {
			punct = true
			continue
		}
		if !alnum {
			punct = true
			continue
		}
		if punct && b.Len() > 0 {
			b.WriteByte('_')
		}
		b.WriteRune(r)
		punct = false
	}
	return b.String()
}

func latin1(b byte) rune { return rune(b) }

func asciiOnly(b byte) rune {
	if b < 0x80 {
		return rune(b)
	}
	return 0
}

// transcode8 decodes 8-bit bytes with table into UTF-8; undefined bytes become invalidMark.
func transcode8(data []byte, table func(byte) rune) []byte {
	out := make([]byte, 0, len(data)+len(data)/8)
	for _, b := range data {
		r := table(b)
		if r == 0 && b != 0 {
			out = append(out, invalidMark)
			continue
		}
		out = utf8.AppendRune(out, r)
	}
	return out
}
