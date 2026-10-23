package signalindex

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"studio-engine/internal/acquisition/pyv"
)

// suggestionLimit is the most points one answer suggests.
const suggestionLimit = 8

// suggestedBy names where a suggestion mentions the component: in its description or reference.
var suggestedBy = [2]string{"description", "reference"}

// suggested is the Go-only "suggested" list (ADR-0113): at most 8 of the matching signals (every
// other filter applied, paging not) whose description mentions near as a whole word, then those
// whose reference does, each in model order and marked "suggested_by". Only text that is in the
// model counts; nothing is guessed. Words are runs of letters and digits, case is ignored:
// "Q11" is in "Q11 open" and "Bay_Q11" but not in "Q110".
func (ix *Index) suggested(q Query, found scanned, values ValueReader) []any {
	byDescription, byReference := ix.mentions(q.Near, found.matches)
	items := make([]any, 0, len(byDescription)+len(byReference))
	for index, positions := range [2][]int{byDescription, byReference} {
		for _, position := range positions {
			if len(items) == suggestionLimit {
				return items
			}
			item := ix.item(position, values, found.codes, q.Annotate)
			item.Set("suggested_by", suggestedBy[index])
			items = append(items, item)
		}
	}
	return items
}

// mentions returns the matches whose description, or else reference, holds near as a word.
func (ix *Index) mentions(near string, matches []int) (byDescription, byReference []int) {
	needle := pyv.Casefold(strings.TrimSpace(near))
	if strings.IndexFunc(needle, isWordRune) < 0 {
		return nil, nil
	}
	candidates := matches
	if containing, ok := ix.text.matches(needle); ok {
		candidates = intersect(containing, matches)
	}
	for _, position := range candidates {
		reference, description := ix.split(position)
		switch {
		case containsWord(description, needle):
			byDescription = append(byDescription, position)
			if len(byDescription) == suggestionLimit {
				return byDescription, byReference
			}
		case len(byReference) < suggestionLimit && containsWord(reference, needle):
			byReference = append(byReference, position)
		}
	}
	return byDescription, byReference
}

// split returns the casefolded reference and description of a signal, cut from its haystack
// ((ref + " " + description).casefold()) without a copy when the reference is ASCII.
func (ix *Index) split(position int) (string, string) {
	item := &ix.signals[position]
	size := len(item.ref)
	for index := 0; index < len(item.ref); index++ {
		if item.ref[index] >= utf8.RuneSelf {
			size = len(pyv.Casefold(item.ref))
			break
		}
	}
	if size < len(item.haystack) && item.haystack[size] == ' ' {
		return item.haystack[:size], item.haystack[size+1:]
	}
	return pyv.Casefold(item.ref), pyv.Casefold(pyv.PyStr(pyv.Get(item.object, "description")))
}

// intersect returns the positions in both sorted lists, in order.
func intersect(a, b []int) []int {
	var both []int
	for i, j := 0, 0; i < len(a) && j < len(b); {
		switch {
		case a[i] < b[j]:
			i++
		case a[i] > b[j]:
			j++
		default:
			both = append(both, a[i])
			i++
			j++
		}
	}
	return both
}

func isWordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

// containsWord is true when needle occurs in text and is not glued to a letter or digit on a side
// where needle itself starts or ends with one (like a regular-expression \b).
func containsWord(text, needle string) bool {
	first, _ := utf8.DecodeRuneInString(needle)
	last, _ := utf8.DecodeLastRuneInString(needle)
	for from := 0; from+len(needle) <= len(text); {
		at := strings.Index(text[from:], needle)
		if at < 0 {
			return false
		}
		at += from
		before, _ := utf8.DecodeLastRuneInString(text[:at])
		after, _ := utf8.DecodeRuneInString(text[at+len(needle):])
		gluedBefore := at > 0 && isWordRune(first) && isWordRune(before)
		gluedAfter := at+len(needle) < len(text) && isWordRune(last) && isWordRune(after)
		if !gluedBefore && !gluedAfter {
			return true
		}
		_, size := utf8.DecodeRuneInString(text[at:])
		from = at + size
	}
	return false
}
