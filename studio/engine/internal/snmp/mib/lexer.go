package mib

import "strings"

// Token kinds of the MIB lexer.
const (
	tokIdent  = iota // name, keyword or module name (letters, digits, '-', '_')
	tokNumber        // decimal number, possibly negative
	tokString        // "quoted text" (text without the quotes)
	tokBinary        // 'hex'H or 'bits'B literal
	tokPunct         // ::= .. { } ( ) [ ] , ; | and any other single character
)

type token struct {
	kind int
	text string
	line int
}

// lex splits MIB text into tokens, dropping comments and white space. It never fails:
// characters it does not understand become single-character punctuation tokens.
func lex(src string) []token {
	toks := make([]token, 0, len(src)/6)
	line := 1
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == '\n':
			line++
			i++
		case c <= ' ' || c >= 0x7f:
			i++
		case c == '-' && i+1 < len(src) && src[i+1] == '-':
			i = skipComment(src, i+2)
		case c == '"':
			start, l := i+1, line
			i = start
			var b strings.Builder
			for i < len(src) {
				if src[i] == '"' {
					if i+1 < len(src) && src[i+1] == '"' {
						b.WriteByte('"')
						i += 2
						continue
					}
					break
				}
				if src[i] == '\n' {
					line++
				}
				b.WriteByte(src[i])
				i++
			}
			toks = append(toks, token{tokString, b.String(), l})
			i++
		case c == '\'':
			end := strings.IndexByte(src[i+1:], '\'')
			if end < 0 {
				i++
				continue
			}
			end += i + 1
			if end+1 < len(src) && strings.ContainsRune("HhBb", rune(src[end+1])) {
				end++
			}
			lit := src[i : end+1]
			toks = append(toks, token{tokBinary, lit, line})
			line += strings.Count(lit, "\n")
			i = end + 1
		case isLetter(c):
			j := i + 1
			for j < len(src) && isIdentChar(src[j]) {
				if src[j] == '-' && j+1 < len(src) && src[j+1] == '-' {
					break
				}
				j++
			}
			toks = append(toks, token{tokIdent, src[i:j], line})
			i = j
		case isDigit(c) || (c == '-' && i+1 < len(src) && isDigit(src[i+1])):
			j := i + 1
			for j < len(src) && isDigit(src[j]) {
				j++
			}
			toks = append(toks, token{tokNumber, src[i:j], line})
			i = j
		case strings.HasPrefix(src[i:], "::="):
			toks = append(toks, token{tokPunct, "::=", line})
			i += 3
		case strings.HasPrefix(src[i:], ".."):
			toks = append(toks, token{tokPunct, "..", line})
			i += 2
		default:
			toks = append(toks, token{tokPunct, src[i : i+1], line})
			i++
		}
	}
	return toks
}

// skipComment returns the index after a comment that started just before i: a comment ends
// at the end of the line (the newline is left for the line counter) or at the next "--".
func skipComment(src string, i int) int {
	for i < len(src) {
		if src[i] == '\n' {
			return i
		}
		if src[i] == '-' && i+1 < len(src) && src[i+1] == '-' {
			return i + 2
		}
		i++
	}
	return i
}

func isLetter(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isIdentChar(c byte) bool { return isLetter(c) || isDigit(c) || c == '-' || c == '_' }

// isUpperIdent reports whether a name starts with a capital letter (a type or module name).
func isUpperIdent(s string) bool { return s != "" && s[0] >= 'A' && s[0] <= 'Z' }
