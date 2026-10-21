package push

import (
	"net/url"
	"sort"
	"strings"
	"time"

	"studio-engine/internal/pyjson"
)

// Kind is how a topic's changes travel after the first full frame.
type Kind uint8

const (
	// Whole sends the full body whenever it changed (small bodies).
	Whole Kind = iota
	// Keyed sends the text of changed rows of List (keyed by String(row[Key])), removed and
	// added keys, and the changed text around the list (head, tail).
	Keyed
)

// Getter answers the topic's polled GET route in-process: status and exact body bytes.
type Getter func(query string) (status int, body []byte)

// Topic is one subscribable view. Its full frame is the exact body of its GET route.
type Topic struct {
	Name string
	// Path is the polled GET path under /api/v1 ("/health") that the topic replaces; the
	// frontend's live feed maps its paths to topics through /capabilities.
	Path string
	Get  Getter
	Kind Kind
	List string // Keyed: the array member
	Key  string // Keyed: the row identity member
	// Params lists the subscription parameters passed to the route as its query string
	// (for example "protocol"); others are ignored.
	Params []string
	// Ignore is a top-level member left out of the "did it change" comparison (health's
	// checked_at, which no view shows). A change of it alone sends nothing.
	Ignore string
	// Gap is the coalescing interval (leading edge, then at most once per Gap).
	Gap time.Duration
	// Every re-reads the route that often while While() is true and someone subscribes
	// (counters and staleness that change without an event). 0: never.
	Every time.Duration
	While func() bool
	// Triggers are change names (OnChange hooks) and bus event kinds that mark the topic
	// dirty; a name ending in "*" is a prefix. "runtime", "ied_runtime", "project_changed"
	// and a value hub resync mark every topic.
	Triggers []string
	// Values selects value hub changes (iedID, ref) that mark the topic; nil: none.
	Values func(iedID, ref string) bool
}

// DefaultGap is the coalescing interval of most topics.
const DefaultGap = 100 * time.Millisecond

// AnyValue marks a topic on every value change.
func AnyValue(string, string) bool { return true }

// triggeredBy reports whether the change name or bus kind marks the topic.
func (t *Topic) triggeredBy(name string) bool {
	for _, trigger := range t.Triggers {
		if trigger == name || (strings.HasSuffix(trigger, "*") && strings.HasPrefix(name, trigger[:len(trigger)-1])) {
			return true
		}
	}
	return false
}

// query builds the route's query string from the allowed parameters (sorted, so equal
// parameters share one computation). Text is sent as is, numbers and booleans as JSON.
func (t *Topic) query(params *pyjson.Object) string {
	if params == nil || len(t.Params) == 0 {
		return ""
	}
	values := url.Values{}
	for _, name := range t.Params {
		raw, found := params.Get(name)
		if !found || raw == nil {
			continue
		}
		if text, isText := raw.(string); isText {
			values.Set(name, text)
		} else if encoded, err := pyjson.Compact(raw); err == nil {
			values.Set(name, string(encoded))
		}
	}
	return values.Encode()
}

// globalTriggers mark every active topic.
var globalTriggers = map[string]bool{"runtime": true, "ied_runtime": true, "project_changed": true, "resync": true}

// Names returns the sorted topic names.
func Names(topics []*Topic) []string {
	names := make([]string, 0, len(topics))
	for _, topic := range topics {
		names = append(names, topic.Name)
	}
	sort.Strings(names)
	return names
}
