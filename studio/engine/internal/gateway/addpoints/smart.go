package addpoints

import (
	"strconv"

	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
)

// SmartOptions are the choices of the smart Propose preview (ADR-0105).
type SmartOptions struct {
	// AddNew keeps the list as it is (with the user's edits) and adds only points it does not
	// have yet; otherwise the proposal replaces the IED points and own points stay (KeepOwn).
	AddNew bool
	// Ticked are the IEDs to propose from; nil means every IED that is not switched off.
	Ticked map[string]bool
	Current []any
}

// IEDSummary is one row of the preview: what an IED offers and how much of it is new.
type IEDSummary struct {
	ID, Name, Role string
	Offered, New   int
	Ticked         bool
}

// Smart is Propose from IEDs in smart mode: the same selection as gateway.Propose for the
// ticked IEDs plus their typed step and set point commands (withTypedCommands), with names
// from the SCL descriptions, every command's status point added and each command's address
// paired with its status. It answers the new list, notes and the per-IED summary for the
// preview.
func Smart(raw []*pyjson.Object, protocol string, options SmartOptions) ([]any, []string, []IEDSummary, error) {
	all, _, err := gateway.Propose(raw, protocol)
	if err != nil {
		return nil, nil, nil, err
	}
	index := NewIndex(raw)
	all = withTypedCommands(protocol, raw, index, all)
	existing := TakenIDs(options.Current)
	summary := summarise(index, all, existing, options.Ticked)
	ticked := []*pyjson.Object{}
	for _, ied := range raw {
		if options.Ticked == nil || options.Ticked[text(ied, "id")] {
			ticked = append(ticked, ied)
		}
	}
	proposal, notes, err := gateway.Propose(ticked, protocol)
	if err != nil {
		return nil, nil, nil, err
	}
	proposal = withStatuses(index, withTypedCommands(protocol, ticked, index, proposal))
	if !options.AddNew {
		addressed := readdress(protocol, index, nil, proposal)
		kept, moved := gateway.KeepOwn(protocol, addressed, ownOf(options.Current))
		return kept, append(notes, moved...), summary, nil
	}
	fresh := []any{}
	for _, value := range proposal {
		if !existing[text(value.(*pyjson.Object), "id")] {
			fresh = append(fresh, value)
		}
	}
	added := readdress(protocol, index, options.Current, fresh)
	notes = []string{addedNote(len(added), len(options.Current))}
	return append(append([]any{}, options.Current...), added...), notes, summary, nil
}

func addedNote(added, kept int) string {
	if added == 0 {
		return "No new points: every proposed point is already in the list. Your list is unchanged."
	}
	return strconv.Itoa(added) + " new points added; the " + strconv.Itoa(kept) +
		" points already in the list kept their addresses and names."
}

// withStatuses adds the status point of every proposed command that the proposal lacks and
// fills empty names from the SCL descriptions.
func withStatuses(index *Index, proposal []any) []any {
	ids := TakenIDs(proposal)
	result := []any{}
	for _, value := range proposal {
		point := value.(*pyjson.Object)
		ied := index.ByID(text(point, "ied"))
		if ied == nil {
			result = append(result, point)
			continue
		}
		point.Set("description", nameOf(ied, text(point, "ref"), text(point, "description")))
		result = append(result, point)
		if !gateway.IsCommandKind(text(point, "kind")) {
			continue
		}
		statusRef, statusKind := ied.StatusOf(ied.Control(text(point, "ref")))
		if id := gateway.PointID(ied.ID, statusRef, statusKind); statusRef != "" && !ids[id] {
			ids[id] = true
			result = append(result, iedPoint(ied, statusKind, statusRef, nameOf(ied, statusRef, ""), -1, 1.0))
		}
	}
	return result
}

// readdress gives the points their addresses around the current list: status and measured
// points keep a proposed address that is still free (else the next free one), and each
// command is paired with its status point.
func readdress(protocol string, index *Index, current, points []any) []any {
	addresses := newAddresser(protocol, current)
	known := addressesByID(current)
	for _, value := range points {
		point := value.(*pyjson.Object)
		kind := text(point, "kind")
		if gateway.IsCommandKind(kind) {
			continue
		}
		at, _ := point.Get("address")
		address, ok := at.(int64)
		if ok && address >= 0 && addresses.free(kind, address) {
			addresses.take(kind, address)
		} else {
			address, _ = addresses.next(kind)
			point.Set("address", address)
		}
		known[text(point, "id")] = address
	}
	for _, value := range points {
		point := value.(*pyjson.Object)
		kind := text(point, "kind")
		if !gateway.IsCommandKind(kind) {
			continue
		}
		var address int64
		if ied := index.ByID(text(point, "ied")); ied != nil {
			statusRef, statusKind := ied.StatusOf(ied.Control(text(point, "ref")))
			if at, found := known[gateway.PointID(ied.ID, statusRef, statusKind)]; found && statusRef != "" {
				address, _ = addresses.follow(kind, statusKind, at)
				point.Set("address", address)
				continue
			}
		}
		address, _ = addresses.next(kind)
		point.Set("address", address)
	}
	return points
}

func summarise(index *Index, proposal []any, existing map[string]bool, ticked map[string]bool) []IEDSummary {
	counts, fresh := map[string]int{}, map[string]int{}
	for _, value := range proposal {
		point := value.(*pyjson.Object)
		counts[text(point, "ied")]++
		if !existing[text(point, "id")] {
			fresh[text(point, "ied")]++
		}
	}
	summary := []IEDSummary{}
	for _, ied := range index.List {
		if ied.Role == "off" {
			continue
		}
		summary = append(summary, IEDSummary{ID: ied.ID, Name: ied.Name, Role: ied.Role, Offered: counts[ied.ID],
			New: fresh[ied.ID], Ticked: ticked == nil || ticked[ied.ID]})
	}
	return summary
}

func ownOf(points []any) []any {
	own := []any{}
	for _, value := range points {
		if point, ok := value.(*pyjson.Object); ok && gateway.IsOwn(point) {
			own = append(own, point)
		}
	}
	return own
}
