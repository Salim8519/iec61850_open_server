package push

import (
	"encoding/json"
	"fmt"
	"strings"
)

// mirror is the documented client algorithm (lane F's push client) in Go: it keeps the
// exact GET text of one topic from full and delta frames.
type mirror struct {
	rev        int64
	text       string
	head, tail string
	keys       []string
	rows       map[string]string
	keyed      bool
}

// apply takes one frame text; it returns an error where the client would ask for resync.
func (m *mirror) apply(frame string) error {
	var f struct {
		Rev    int64  `json:"rev"`
		Base   *int64 `json:"base"`
		Status int    `json:"status"`
		Rows   *struct {
			Head int      `json:"head"`
			Keys []string `json:"keys"`
			Ends []int    `json:"ends"`
		} `json:"rows"`
		Head   *string           `json:"head"`
		Tail   *string           `json:"tail"`
		Upsert map[string]string `json:"upsert"`
		Remove []string          `json:"remove"`
		Add    []string          `json:"add"`
	}
	if err := json.Unmarshal([]byte(frame), &f); err != nil {
		return err
	}
	if f.Base == nil { // full, always taken: the body is the text between `"full":` and the last "}"
		at := strings.Index(frame, `"full":`)
		m.rev, m.text, m.keyed = f.Rev, frame[at+7:len(frame)-1], f.Rows != nil
		if m.keyed {
			m.head, m.keys, m.rows = m.text[:f.Rows.Head], f.Rows.Keys, map[string]string{}
			start := f.Rows.Head
			for index, end := range f.Rows.Ends {
				m.rows[m.keys[index]] = m.text[start:end]
				start = end + 1
			}
			if len(f.Rows.Ends) > 0 {
				start--
			}
			m.tail = m.text[start:]
		}
		return nil
	}
	if *f.Base != m.rev || !m.keyed {
		if f.Rev <= m.rev {
			return nil // already shown through a newer full frame
		}
		return fmt.Errorf("resync: base %d, have %d", *f.Base, m.rev)
	}
	if f.Head != nil {
		m.head = *f.Head
	}
	if f.Tail != nil {
		m.tail = *f.Tail
	}
	removed := map[string]bool{}
	for _, key := range f.Remove {
		removed[key] = true
		delete(m.rows, key)
	}
	kept := m.keys[:0:0]
	for _, key := range m.keys {
		if !removed[key] {
			kept = append(kept, key)
		}
	}
	m.keys = append(kept, f.Add...)
	for key, row := range f.Upsert {
		m.rows[key] = row
	}
	parts := make([]string, len(m.keys))
	for index, key := range m.keys {
		parts[index] = m.rows[key]
	}
	m.rev, m.text = f.Rev, m.head+strings.Join(parts, ",")+m.tail
	return nil
}
