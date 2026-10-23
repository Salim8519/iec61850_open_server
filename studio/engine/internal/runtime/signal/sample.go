package signal

import (
	"strings"

	"studio-engine/internal/apperr"
)

// Lookup finds a signal of the same IED by reference.
type Lookup func(ref string) (Signal, bool)

// Companions are the closest same-FC quality ("q") and time ("t") attributes (domain/sample.py).
type Companions struct {
	Q, T       Signal
	HasQ, HasT bool
	// Order lists the found companions ('q', 't') in the order Python's dict holds them.
	Order []byte
}

// Refs are the companion references in dict order.
func (c Companions) Refs() []string {
	refs := make([]string, 0, 2)
	for _, key := range c.Order {
		if key == 'q' {
			refs = append(refs, c.Q.Ref)
		} else {
			refs = append(refs, c.T.Ref)
		}
	}
	return refs
}

// FindCompanions walks up the reference like sample.companions.
func FindCompanions(item Signal, lookup Lookup) Companions {
	var result Companions
	parent := item.Ref
	if i := strings.LastIndex(parent, "."); i >= 0 {
		parent = parent[:i]
	}
	for strings.Contains(parent, ".") {
		if !result.HasQ {
			if candidate, ok := lookup(parent + ".q"); ok && candidate.Kind == "Quality" && candidate.FC == item.FC {
				result.Q, result.HasQ = candidate, true
				result.Order = append(result.Order, 'q')
			}
		}
		if !result.HasT {
			if candidate, ok := lookup(parent + ".t"); ok && candidate.Kind == "Timestamp" && candidate.FC == item.FC {
				result.T, result.HasT = candidate, true
				result.Order = append(result.Order, 't')
			}
		}
		parent = parent[:strings.LastIndex(parent, ".")]
	}
	return result
}

// Update is one {ref, value} of a native batch.
type Update struct {
	Ref   string
	Value any
}

// MakeSample is sample.make_sample: the writes for one signal (value, then quality, then
// time) and its companions. quality nil means "not given".
func MakeSample(item Signal, lookup Lookup, value any, timestamp int64, quality any) ([]Update, Companions, error) {
	checked, err := Validate(item.Kind, value)
	if err == nil {
		err = checkEnum(item, checked)
	}
	if err == nil {
		err = checkVisible(item, checked)
	}
	if err != nil {
		return nil, Companions{}, err
	}
	updates := []Update{{item.Ref, checked}}
	related := FindCompanions(item, lookup)
	if item.Kind == "Quality" || item.Kind == "Timestamp" {
		return updates, related, nil
	}
	if quality != nil {
		if !related.HasQ {
			return nil, related, apperr.ValueError("This attribute has no associated SCL quality")
		}
		q, err := checkQuality(related.Q.Kind, quality)
		if err != nil {
			return nil, related, err
		}
		updates = append(updates, Update{related.Q.Ref, q})
	}
	if related.HasT {
		t, err := Validate(related.T.Kind, timestamp)
		if err != nil {
			return nil, related, err
		}
		updates = append(updates, Update{related.T.Ref, t})
	}
	return updates, related, nil
}
