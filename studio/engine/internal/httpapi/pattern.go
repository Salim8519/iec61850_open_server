package httpapi

import (
	"fmt"
	"strings"
)

// segment kinds, in Werkzeug's matching preference (static parts win over variables).
const (
	segmentStatic = iota
	segmentPartial
	segmentVariable
	segmentRest
)

// segment is one path part of a route pattern: "name", "{key}", "{edge_id}.csv" (a
// variable with literal prefix/suffix, Werkzeug's "<edge_id>.csv") or "{path...}".
type segment struct {
	kind           int
	literal        string
	name           string
	prefix, suffix string
}

type pattern struct {
	text     string
	segments []segment
}

// parsePattern reads a Go 1.22 style path pattern ("/api/v1/ieds/{key}/settings").
func parsePattern(text string) (pattern, error) {
	if !strings.HasPrefix(text, "/") {
		return pattern{}, fmt.Errorf("route pattern %q must start with /", text)
	}
	result := pattern{text: text}
	parts := strings.Split(text[1:], "/")
	for index, part := range parts {
		open, close := strings.IndexByte(part, '{'), strings.IndexByte(part, '}')
		switch {
		case open < 0 && close < 0:
			result.segments = append(result.segments, segment{kind: segmentStatic, literal: part})
		case open < 0 || close < open || strings.Count(part, "{") > 1:
			return pattern{}, fmt.Errorf("route pattern %q has a malformed segment %q", text, part)
		case strings.HasSuffix(part[open+1:close], "..."):
			if open != 0 || close != len(part)-1 || index != len(parts)-1 {
				return pattern{}, fmt.Errorf("route pattern %q: {name...} must be the last segment", text)
			}
			result.segments = append(result.segments, segment{kind: segmentRest,
				name: strings.TrimSuffix(part[1:close], "...")})
		default:
			kind := segmentVariable
			if open > 0 || close < len(part)-1 {
				kind = segmentPartial
			}
			result.segments = append(result.segments, segment{kind: kind, name: part[open+1 : close],
				prefix: part[:open], suffix: part[close+1:]})
		}
	}
	return result, nil
}

// match returns the variable values when path matches, Werkzeug-style: a variable is a
// non-empty part without '/', a rest variable is non-empty and does not start with '/'.
func (p pattern) match(path string) (map[string]string, bool) {
	if !strings.HasPrefix(path, "/") {
		return nil, false
	}
	parts := strings.Split(path[1:], "/")
	values := map[string]string{}
	for index, seg := range p.segments {
		if index >= len(parts) {
			return nil, false
		}
		part := parts[index]
		switch seg.kind {
		case segmentStatic:
			if part != seg.literal {
				return nil, false
			}
		case segmentRest:
			rest := strings.Join(parts[index:], "/")
			if rest == "" || strings.HasPrefix(rest, "/") {
				return nil, false
			}
			values[seg.name] = rest
			return values, true
		default:
			inner := strings.TrimSuffix(strings.TrimPrefix(part, seg.prefix), seg.suffix)
			if len(part) < len(seg.prefix)+len(seg.suffix)+1 || !strings.HasPrefix(part, seg.prefix) ||
				!strings.HasSuffix(part, seg.suffix) || inner == "" {
				return nil, false
			}
			values[seg.name] = inner
		}
	}
	return values, len(parts) == len(p.segments)
}

// before reports whether p is preferred over other when both match one path.
func (p pattern) before(other pattern) bool {
	for index := 0; index < len(p.segments) && index < len(other.segments); index++ {
		if a, b := p.segments[index].kind, other.segments[index].kind; a != b {
			return a < b
		}
	}
	return len(p.segments) > len(other.segments)
}
