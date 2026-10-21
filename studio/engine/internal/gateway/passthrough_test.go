package gateway

import (
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pyoracle"
)

// TestPassthroughAgainstPython replays the protocol converter mapping on display rows built
// by merge + rows, raw rows with bool, negative and huge addresses, and command routes.
func TestPassthroughAgainstPython(t *testing.T) {
	replay(t, "passthrough_oracle.py", "2000", func(item pyoracle.Case, _ map[string][]*pyjson.Object) {
		args := item.Args
		switch item.Fn {
		case "map_points":
			points, notes, err := MapPoints(list(args[0]), args[1].(string), list(args[2]))
			expect(t, item, []any{points, strs(notes)}, err)
		case "device_value":
			expect(t, item, DeviceValue(object(args[0]), args[1].(bool)), nil)
		case "remote_sample":
			got, err := RemoteSample(object(args[0]), object(args[1]), args[2].(bool), args[3].(bool))
			expect(t, item, got, err)
		default:
			t.Errorf("unknown oracle function %s", item.Fn)
		}
	})
}
