package netcfg

import (
	"strings"
	"unicode"
)

// pyIsSpace is str.isspace for one character (Python also counts \x1c-\x1f).
func pyIsSpace(r rune) bool {
	return unicode.IsSpace(r) || (r >= 0x1c && r <= 0x1f)
}

// pyStrip is str.strip() without arguments.
func pyStrip(text string) string {
	return strings.TrimFunc(text, pyIsSpace)
}

func isLineBreak(r rune) bool {
	switch r {
	case '\n', '\r', '\v', '\f', 0x1c, 0x1d, 0x1e, 0x85, 0x2028, 0x2029:
		return true
	}
	return false
}

// pySplitlines is str.splitlines(): every Python line boundary, "\r\n" counted once, no
// trailing empty line.
func pySplitlines(text string) []string {
	lines := []string{}
	start := 0
	runes := []rune(text)
	for index := 0; index < len(runes); index++ {
		if !isLineBreak(runes[index]) {
			continue
		}
		lines = append(lines, string(runes[start:index]))
		if runes[index] == '\r' && index+1 < len(runes) && runes[index+1] == '\n' {
			index++
		}
		start = index + 1
	}
	if start < len(runes) {
		lines = append(lines, string(runes[start:]))
	}
	return lines
}

// cutRunes is text[:limit] on a Python str (characters, not bytes).
func cutRunes(text string, limit int) string {
	count := 0
	for index := range text {
		if count == limit {
			return text[:index]
		}
		count++
	}
	return text
}

// lastLine is (text.strip().splitlines() or [fallback])[-1], used for PowerShell errors.
func lastLine(text, fallback string) string {
	lines := pySplitlines(pyStrip(text))
	if len(lines) == 0 {
		return fallback
	}
	return lines[len(lines)-1]
}
