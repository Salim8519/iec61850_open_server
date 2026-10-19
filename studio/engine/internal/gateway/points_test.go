package gateway

import (
	"sort"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pyoracle"
)

// TestPointsAgainstPython replays gateway_selection and gateway_points on public-template
// IEDs with random mutations, including sets that offer more than MaxPoints candidates.
func TestPointsAgainstPython(t *testing.T) {
	replay(t, "points_oracle.py", "2000", func(item pyoracle.Case, fixtures map[string][]*pyjson.Object) {
		args := item.Args
		switch item.Fn {
		case "node_class":
			expect(t, item, NodeClass(args[0].(string)), nil)
		case "smart_kind":
			kind, err := smartKind(object(args[0]))
			var got any
			if kind != "" {
				got = kind
			}
			expect(t, item, got, err)
		case "engineered":
			var refs map[string]bool
			if args[1] != nil {
				refs = map[string]bool{}
				for _, ref := range stringList(args[1]) {
					refs[ref] = true
				}
			}
			expect(t, item, Engineered(object(args[0]), refs), nil)
		case "occupied":
			got := []any{}
			for _, cell := range Occupied(args[0].(string), object(args[1])) {
				got = append(got, []any{cell.Space, cell.Address})
			}
			expect(t, item, got, nil)
		case "point_id":
			expect(t, item, PointID(args[0].(string), args[1].(string), args[2].(string)), nil)
		case "dataset_refs":
			refs := DatasetRefs(fixtures[args[0].(string)][int(args[1].(int64))])
			var got any
			if refs != nil {
				keys := []string{}
				for ref := range refs {
					keys = append(keys, ref)
				}
				sort.Strings(keys)
				got = strs(keys)
			}
			expect(t, item, got, nil)
		case "propose":
			points, notes, err := Propose(fixtures[args[0].(string)], args[1].(string))
			expect(t, item, []any{points, strs(notes)}, err)
		default:
			t.Errorf("unknown oracle function %s", item.Fn)
		}
	})
}
