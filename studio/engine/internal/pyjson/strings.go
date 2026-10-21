package pyjson

const hexDigits = "0123456789abcdef"

// AppendString writes s as a JSON string with ensure_ascii=True, the default of both
// json.dumps and Flask: printable ASCII stays, '"', '\\' and \b \f \n \r \t use short
// escapes, everything else (controls, DEL, non-ASCII) becomes lowercase \uXXXX, with a
// surrogate pair above U+FFFF. Invalid UTF-8 is written as U+FFFD.
func AppendString(dst []byte, s string) []byte {
	dst = append(dst, '"')
	for _, r := range s {
		switch r {
		case '"':
			dst = append(dst, '\\', '"')
		case '\\':
			dst = append(dst, '\\', '\\')
		case '\b':
			dst = append(dst, '\\', 'b')
		case '\f':
			dst = append(dst, '\\', 'f')
		case '\n':
			dst = append(dst, '\\', 'n')
		case '\r':
			dst = append(dst, '\\', 'r')
		case '\t':
			dst = append(dst, '\\', 't')
		default:
			switch {
			case r >= 0x20 && r < 0x7f:
				dst = append(dst, byte(r))
			case r > 0xffff:
				r -= 0x10000
				dst = appendEscape(dst, 0xd800+(r>>10))
				dst = appendEscape(dst, 0xdc00+(r&0x3ff))
			default:
				dst = appendEscape(dst, r)
			}
		}
	}
	return append(dst, '"')
}

func appendEscape(dst []byte, r rune) []byte {
	return append(dst, '\\', 'u', hexDigits[r>>12&0xf], hexDigits[r>>8&0xf], hexDigits[r>>4&0xf], hexDigits[r&0xf])
}
