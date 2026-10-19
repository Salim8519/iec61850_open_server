// Package pytext reproduces the few CPython text and file rules the license home and the
// update check depend on: str.isspace/strip, int(str), json.loads input decoding,
// json.dumps(indent=2) and Path.write_text + os.replace, so files and answers stay
// byte-identical to the frozen Python engine.
package pytext

import "strings"

// IsSpace reports whether r is whitespace for CPython's str.isspace(), str.strip(), int()
// and the \s class of str regular expressions (Py_UNICODE_ISSPACE, Unicode 15).
func IsSpace(r rune) bool {
	switch {
	case r >= 0x09 && r <= 0x0d, r >= 0x1c && r <= 0x20:
		return true
	case r < 0x85:
		return false
	case r == 0x85, r == 0xa0, r == 0x1680, r >= 0x2000 && r <= 0x200a:
		return true
	case r == 0x2028, r == 0x2029, r == 0x202f, r == 0x205f, r == 0x3000:
		return true
	}
	return false
}

// Strip returns s without leading and trailing whitespace, as str.strip() does.
func Strip(s string) string { return strings.TrimFunc(s, IsSpace) }

// RemoveSpace removes every whitespace character, as re.sub(r"\s+", "", s) does.
func RemoveSpace(s string) string {
	return strings.Map(func(r rune) rune {
		if IsSpace(r) {
			return -1
		}
		return r
	}, s)
}
