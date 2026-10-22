package addpoints

import (
	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
)

// withTypedCommands adds the typed controls (ADR-0104) of the proposed IEDs to a smart
// proposal: a BSC or ISC as a step command, an APC or INC as a set point (ADR-0107), the
// same kinds Add points offers for them. gateway.Propose itself stays the Python selection
// (SPC/DPC only, byte-identical); this is the Go-only smart path (change 0178). Each typed
// command gets its status (tap position or measured value) through withStatuses, and the
// set point the protocol's default format when the list is checked. A kind the protocol
// does not carry (a step on Modbus) is left out.
func withTypedCommands(protocol string, raw []*pyjson.Object, index *Index, proposal []any) []any {
	ids := TakenIDs(proposal)
	result := append([]any{}, proposal...)
	for _, item := range raw {
		ied := index.ByID(text(item, "id"))
		if ied == nil || ied.Role == "off" {
			continue
		}
		for _, value := range gateway.TypedControlsOf(item) {
			control, ok := value.(*pyjson.Object)
			if !ok || text(control, "ref") == "" {
				continue
			}
			kind := typedKindOf(control)
			id := gateway.PointID(ied.ID, text(control, "ref"), kind)
			if _, served := Entry(protocol, kind); !served || ids[id] { // Modbus has no step command
				continue
			}
			ids[id] = true
			result = append(result, iedPoint(ied, kind, text(control, "ref"), "", -1, 1.0))
		}
	}
	return result
}

// typedKindOf is the gateway command kind of a typed control ("" for ENC, BAC and others).
func typedKindOf(control *pyjson.Object) string {
	for _, kind := range []string{gateway.KindStep, gateway.KindSetpoint} {
		if CommandFits(control, kind) {
			return kind
		}
	}
	return ""
}
