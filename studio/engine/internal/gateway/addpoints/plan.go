package addpoints

import (
	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
)

// Pick is one ticked signal or command of the "From an IED" list. Address 0 with HasAddress
// false means "next free"; an empty Name means "the SCL description".
type Pick struct {
	Ref        string
	Name       string
	Address    int64
	HasAddress bool
}

// FromIED plans the points for the ticked signals or commands of one IED, all of type kind.
// A command with withStatus also gets its status point (unless the list already has it),
// paired by address. Commands planned here are IED points: SCADA commands on them operate
// the IED through the gateway's command path. current is the point list now.
func FromIED(protocol string, ied *IED, kind string, picks []Pick, withStatus bool, current []any,
	ieds []*pyjson.Object) []Result {
	addresses := newAddresser(protocol, current)
	taken := TakenIDs(current)
	known := addressesByID(current)
	results := []Result{}
	for index, pick := range picks {
		number := index + 1
		ref := pick.Ref
		result := Result{Number: number}
		problem := fitProblem(ied, kind, ref)
		if problem == "" && taken[gateway.PointID(ied.ID, ref, kind)] {
			problem = "this " + typeName(protocol, kind) + " is already in the point list"
		}
		taken[gateway.PointID(ied.ID, ref, kind)] = true
		var status *Result
		at := pick.Address
		if problem == "" {
			if gateway.IsCommandKind(kind) && withStatus {
				status, at, problem = planStatus(protocol, ied, kind, pick, addresses, taken, known)
			} else if pick.HasAddress {
				problem = addresses.claim(kind, at)
			} else if next, ok := addresses.next(kind); ok {
				at = next
			} else {
				problem = "no free address is left for this type"
			}
		}
		result.Point = iedPoint(ied, kind, ref, nameOf(ied, ref, pick.Name), at, 1.0)
		result.Problem = problem
		results = append(results, result)
		if status != nil {
			status.Number = number
			status.StatusFor = gateway.PointID(ied.ID, ref, kind)
			results = append(results, *status)
		}
	}
	checkWithEngine(protocol, results, ieds)
	return results
}

// planStatus gives a command its address paired with its status point: the status point in
// the list already, or a new one planned with it (nil when the list has it or there is none).
func planStatus(protocol string, ied *IED, kind string, pick Pick, addresses *addresser, taken map[string]bool,
	known map[string]int64) (*Result, int64, string) {
	statusRef, statusKind := ied.StatusOf(ied.Control(pick.Ref))
	statusID := gateway.PointID(ied.ID, statusRef, statusKind)
	if statusRef == "" || taken[statusID] { // no status to add, or it is in the list already
		if pick.HasAddress {
			return nil, pick.Address, addresses.claim(kind, pick.Address)
		}
		if at, found := known[statusID]; found {
			command, _ := addresses.follow(kind, statusKind, at)
			return nil, command, ""
		}
		at, ok := addresses.next(kind)
		return nil, at, map[bool]string{true: "", false: "no free address is left for this type"}[ok]
	}
	taken[statusID] = true
	var statusAt, command int64
	ok := true
	if pick.HasAddress {
		if problem := addresses.claim(kind, pick.Address); problem != "" {
			return nil, pick.Address, problem
		}
		command = pick.Address
		statusAt, ok = addresses.follow(statusKind, kind, command)
	} else {
		statusAt, command, ok = addresses.pair(statusKind, kind)
	}
	if !ok {
		return nil, command, "no free address is left for this command and its status"
	}
	known[statusID] = statusAt
	point := iedPoint(ied, statusKind, statusRef, nameOf(ied, statusRef, ""), statusAt, 1.0)
	return &Result{Point: point}, command, ""
}

// fitProblem says why ref of this IED cannot be a point of kind ("" when it can).
func fitProblem(ied *IED, kind, ref string) string {
	if !gateway.KnownKind(kind) {
		return "choose a type"
	}
	if gateway.IsCommandKind(kind) {
		control := ied.Control(ref)
		if control == nil {
			return ied.Name + " has no controllable object " + ref
		}
		if !CommandFits(control, kind) {
			return ref + " is a " + text(control, "cdc") + " object; choose " + commandTypeFor(text(control, "cdc"))
		}
		return ""
	}
	signal := ied.Signal(ref)
	if signal == nil {
		return ied.Name + " has no signal " + ref
	}
	if !Fits(signal, kind) {
		return ref + " is a " + text(signal, "kind") + " " + text(signal, "fc") + " value and does not fit this type"
	}
	return ""
}

func addressesByID(points []any) map[string]int64 {
	result := map[string]int64{}
	for _, raw := range points {
		if point, ok := raw.(*pyjson.Object); ok {
			value, _ := point.Get("address")
			if at, isInt := value.(int64); isInt {
				result[text(point, "id")] = at
			}
		}
	}
	return result
}

// typeName is the catalog name of a kind, for sentences ("Double point").
func typeName(protocol, kind string) string {
	if entry, ok := Entry(protocol, kind); ok {
		return entry.Name
	}
	return kind
}
