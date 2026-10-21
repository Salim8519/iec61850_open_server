package goosedom

import (
	"strings"
	"unicode"
)

// pyLower is Python's str.lower(): the full lowercase mapping, so U+0130 becomes
// "i̇" and a capital sigma at the end of a word becomes the final sigma.
func pyLower(text string) string {
	ascii := true
	for index := 0; index < len(text); index++ {
		if text[index] >= 0x80 {
			ascii = false
			break
		}
	}
	if ascii {
		return strings.ToLower(text)
	}
	runes := []rune(text)
	var out strings.Builder
	for index, r := range runes {
		switch {
		case r == 0x130:
			out.WriteString("i̇")
		case r == 0x3A3 && finalSigma(runes, index):
			out.WriteRune(0x3C2)
		default:
			out.WriteRune(unicode.ToLower(r))
		}
	}
	return out.String()
}

// finalSigma follows CPython's handle_capital_sigma: a cased letter before (skipping
// case-ignorable characters) and no cased letter after.
func finalSigma(runes []rune, index int) bool {
	before := index - 1
	for before >= 0 && caseIgnorable(runes[before]) {
		before--
	}
	if before < 0 || !cased(runes[before]) {
		return false
	}
	after := index + 1
	for after < len(runes) && caseIgnorable(runes[after]) {
		after++
	}
	return after == len(runes) || !cased(runes[after])
}

func cased(r rune) bool {
	return unicode.IsUpper(r) || unicode.IsLower(r) || unicode.IsTitle(r) ||
		unicode.Is(unicode.Other_Lowercase, r) || unicode.Is(unicode.Other_Uppercase, r)
}

// midLetter lists Word_Break MidLetter, MidNumLet and Single_Quote characters, which
// are case-ignorable besides the Mn, Me, Cf, Lm and Sk categories.
var midLetter = map[rune]bool{0x27: true, 0x2E: true, 0x3A: true, 0xB7: true, 0x387: true,
	0x55F: true, 0x5F4: true, 0x2018: true, 0x2019: true, 0x2024: true, 0x2027: true,
	0xFE13: true, 0xFE52: true, 0xFE55: true, 0xFF07: true, 0xFF0E: true, 0xFF1A: true}

func caseIgnorable(r rune) bool {
	return midLetter[r] || unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf, unicode.Lm, unicode.Sk)
}
