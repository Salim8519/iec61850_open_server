package acquisition

import (
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// tapSignalFields are the signals a tap-changer transformer shows (ADR-0109): tap position,
// Auto/Manual status and LV voltage. Its commands have no value, so nothing is polled for them.
var tapSignalFields = []string{"tap_source", "mode_source", "voltage_source"}

// tapChangerKeys are the signals of the tap-changer transformers, so a real IED polls them.
func tapChangerKeys(nodes []*pyjson.Object) []PointKey {
	var keys []PointKey
	for _, node := range nodes {
		tap := pyv.GetObj(pyv.GetObj(node, "data"), "tap_changer")
		for _, field := range tapSignalFields {
			if source := pyv.GetObj(tap, field); source != nil {
				keys = append(keys, BindingKeys(pyv.O("mode", "direct", "source", source))...)
			}
		}
	}
	return keys
}
