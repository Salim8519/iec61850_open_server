package acquisition

import (
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// pointPanelKeys are the signals the points of point panels show (ADR-0102): status, double and
// value points. A command point has a control and no signal, so a real IED polls nothing for it.
func pointPanelKeys(nodes []*pyjson.Object) []PointKey {
	var keys []PointKey
	for _, node := range nodes {
		for _, entry := range pyv.Objects(pyv.GetList(pyv.GetObj(node, "data"), "points")) {
			if source := pyv.GetObj(entry, "source"); source != nil {
				keys = append(keys, BindingKeys(pyv.O("mode", "direct", "source", source))...)
			}
		}
	}
	return keys
}
