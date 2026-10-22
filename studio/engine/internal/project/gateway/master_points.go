package gateway

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/gateway/masterpoints"
	"studio-engine/internal/pyjson"
)

// keepPointList keeps the SCADA master's own point list and its "Check against IED" switch
// (ADR-0110, Go engine only) when a stored master block has them. Both keys are written only
// when present, so a project without them normalises to the exact bytes the frozen Python
// engine produces. Each protocol's list is checked as the page saves it; a list that is not
// valid stops the load with the reason instead of being dropped silently.
func keepPointList(stored, result *pyjson.Object) error {
	if raw, found := stored.Get("point_list"); found {
		lists, isDict := raw.(*pyjson.Object)
		if !isDict {
			return apperr.ValueError("The master point list must be an object with one list per protocol")
		}
		kept := pyjson.NewObject()
		for _, name := range MasterProtocols {
			list, present := lists.Get(name)
			if !present {
				continue
			}
			clean, err := masterpoints.Validate(name, list)
			if err != nil {
				return apperr.ValueError("The saved " + masterpoints.ProtocolName(name) + " master point list: " + err.Error())
			}
			kept.Set(name, clean)
		}
		result.Set("point_list", kept)
	}
	if on, found := stored.Get("check_ied"); found {
		result.Set("check_ied", on == true)
	}
	return nil
}
