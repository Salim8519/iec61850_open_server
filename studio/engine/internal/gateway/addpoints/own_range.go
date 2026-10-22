package addpoints

import (
	"strconv"
	"strings"

	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
)

// MaxRange is the most made-up points "Add a range" adds at once.
const MaxRange = 1000

// OwnRange plans count made-up (own) points of kind at consecutive addresses from "from"
// ("Add a range"): each one the next address after the cells of the one before. A used
// address stops the plan with a problem on that row, so nothing moves silently. name
// numbered "<name> 1", "<name> 2", … (empty: no names).
func OwnRange(protocol, kind string, count int, from int64, name string, current []any) []Result {
	if count < 1 || count > MaxRange {
		return []Result{{Number: 1, Point: pyjson.NewObject().Set("kind", kind),
			Problem: "choose how many points to add, from 1 to " + strconv.Itoa(MaxRange)}}
	}
	if len(current)+count > gateway.MaxPoints {
		return []Result{{Number: 1, Point: pyjson.NewObject().Set("kind", kind),
			Problem: "a gateway holds at most " + strconv.Itoa(gateway.MaxPoints) + " points"}}
	}
	if !gateway.KnownKind(kind) {
		return []Result{{Number: 1, Point: pyjson.NewObject().Set("kind", kind), Problem: "choose a type"}}
	}
	addresses := newAddresser(protocol, current)
	own := nextOwnNumber(current)
	results := make([]Result, 0, count)
	at := from
	name = strings.TrimSpace(name)
	for number := 1; number <= count; number++ {
		point := pyjson.NewObject().Set("id", gateway.OwnID(own)).Set("kind", kind).Set("source", gateway.OwnSource).
			Set("ied", "").Set("ref", "").Set("address", at).Set("scale", 1.0).
			Set("description", rangeName(name, number)).Set("ied_name", gateway.OwnName).Set("ied_role", "own").
			Set("monitor_only", false)
		if value := gateway.DefaultOwnValue(kind); value != nil {
			point.Set("value", value)
		}
		problem := addresses.claim(kind, at)
		if problem != "" {
			problem = "Point " + strconv.Itoa(number) + ": " + problem
		}
		results = append(results, Result{Number: number, Point: point, Problem: problem})
		if problem != "" {
			break
		}
		cells := gateway.Cells(protocol, kind, at)
		at = cells[len(cells)-1].Address + 1
		own++
	}
	return results
}

func rangeName(name string, number int) string {
	if name == "" {
		return ""
	}
	return head(name+" "+strconv.Itoa(number), 120)
}
