package push

import (
	"bytes"
	"strconv"

	"studio-engine/internal/pyjson"
)

// rows is a Keyed body cut into text: head (up to and including the list's "["), the raw
// rows in order with their keys, and tail (from the list's "]"). head + rows joined by ","
// + tail is the body byte for byte, so a client rebuilds the exact GET text from a delta.
type rows struct {
	valid bool
	head  []byte
	tail  []byte
	order []string          // row keys in body order
	byKey map[string][]byte // raw row by key
	ends  []int             // end offset of each row in the body
}

// offset is the position of part (a sub-slice) inside body.
func offset(body, part []byte) int { return cap(body) - cap(part) }

// parse cuts a Keyed body; valid is false when the body has no such list, a row has no
// usable key, keys repeat, or the list is not written compactly (then only full frames).
// Keys equal to prev's at the same position reuse its strings (no allocation per row).
func (t *Topic) parse(body []byte, prev *rows) rows {
	members, ok := objectFields(body, nil)
	if !ok {
		return rows{}
	}
	listKey, key := quote(t.List), quote(t.Key)
	var list []byte
	for _, member := range members {
		if bytes.Equal(member.key, listKey) {
			list = member.value
		}
	}
	if list == nil {
		return rows{}
	}
	items, ok := arrayItems(list, nil)
	if !ok {
		return rows{}
	}
	start := offset(body, list)
	parsed := rows{head: body[:start+1], order: make([]string, len(items)),
		byKey: make(map[string][]byte, len(items)), ends: make([]int, len(items))}
	next := start + 1 // where the next row must begin
	for index, item := range items {
		at := offset(body, item)
		if at != next {
			return rows{} // spaces around rows: the rebuilt text would differ
		}
		raw, ok := memberValue(item, key)
		if !ok {
			return rows{}
		}
		var text string
		if index < len(prev.order) && len(raw) > 1 && raw[0] == '"' && prev.order[index] == string(raw[1:len(raw)-1]) &&
			bytes.IndexByte(raw, '\\') < 0 {
			text = prev.order[index]
		} else if text, ok = keyText(raw); !ok {
			return rows{}
		}
		if _, duplicate := parsed.byKey[text]; duplicate {
			return rows{}
		}
		parsed.order[index], parsed.byKey[text] = text, item
		parsed.ends[index] = at + len(item)
		next = parsed.ends[index] + 1
	}
	closing := start + len(list) - 1
	if (len(items) > 0 && closing != next-1) || (len(items) == 0 && closing != start+1) {
		return rows{}
	}
	parsed.tail, parsed.valid = body[closing:], true
	return parsed
}

// delta writes the members of a delta frame from prev to next; ok is false when only a
// full frame can express the change (a reorder, a row moved, a body that is not a list).
func (t *Topic) delta(prev, next *rows, out []byte) ([]byte, bool) {
	if !prev.valid || !next.valid {
		return out, false
	}
	var removed []string
	kept := 0
	for _, key := range prev.order {
		if _, found := next.byKey[key]; !found {
			removed = append(removed, key)
			continue
		}
		if kept >= len(next.order) || next.order[kept] != key {
			return out, false
		}
		kept++
	}
	for _, key := range next.order[kept:] {
		if _, existed := prev.byKey[key]; existed {
			return out, false
		}
	}
	if !bytes.Equal(prev.head, next.head) {
		out = pyjson.AppendString(append(out, `,"head":`...), string(next.head))
	}
	if !bytes.Equal(prev.tail, next.tail) {
		out = pyjson.AppendString(append(out, `,"tail":`...), string(next.tail))
	}
	upserts := 0
	for _, key := range next.order {
		row := next.byKey[key]
		if old, found := prev.byKey[key]; found && bytes.Equal(old, row) {
			continue
		}
		if upserts == 0 {
			out = append(out, `,"upsert":{`...)
		} else {
			out = append(out, ',')
		}
		upserts++
		out = pyjson.AppendString(append(pyjson.AppendString(out, key), ':'), string(row))
	}
	if upserts > 0 {
		out = append(out, '}')
	}
	out = appendKeys(out, `,"remove":[`, removed)
	return appendKeys(out, `,"add":[`, next.order[kept:]), true
}

// appendRows writes `,"rows":{"head":H,"keys":[...],"ends":[...]}` for a full frame, so the
// client can cut the full text into rows without parsing it (offsets count bytes, which
// equal UTF-16 units because Flask bodies are ASCII).
func appendRows(out []byte, parsed *rows) []byte {
	out = strconv.AppendInt(append(out, `,"rows":{"head":`...), int64(len(parsed.head)), 10)
	out = appendKeys(out, `,"keys":[`, parsed.order)
	if len(parsed.order) == 0 {
		out = append(out, `,"keys":[]`...)
	}
	out = append(out, `,"ends":[`...)
	for index, end := range parsed.ends {
		if index > 0 {
			out = append(out, ',')
		}
		out = strconv.AppendInt(out, int64(end), 10)
	}
	return append(out, "]}"...)
}

func appendKeys(out []byte, member string, keys []string) []byte {
	if len(keys) == 0 {
		return out
	}
	out = append(out, member...)
	for index, key := range keys {
		if index > 0 {
			out = append(out, ',')
		}
		out = pyjson.AppendString(out, key)
	}
	return append(out, ']')
}
