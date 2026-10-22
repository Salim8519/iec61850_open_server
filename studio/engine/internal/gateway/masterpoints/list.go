package masterpoints

import (
	"strconv"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/gateway/addpoints"
	"studio-engine/internal/pyjson"
)

// Cell is one address a point occupies in its address space (a Modbus float32 takes two).
type Cell struct {
	Space   string
	Address int64
}

// Space is the address space of a type: IEC 104 has one IOA space for every type; DNP3 numbers
// each object group by itself (binary input 0 and double-bit input 0 are two points; commands by
// their output); Modbus numbers each table.
func Space(protocol string, entry addpoints.MasterType) string {
	switch protocol {
	case "iec104":
		return "ioa"
	case "modbus":
		return entry.Table
	}
	if entry.Group != "" {
		return entry.Group
	}
	if entry.Command == "binary_output" {
		return "crob"
	}
	return "analog_command"
}

// Cells are the cells a stored point occupies; nil for a point of no known type.
func Cells(protocol string, point *pyjson.Object) []Cell {
	entry, known := addpoints.MasterEntry(protocol, text(point, "type"))
	address, isInt := field(point, "address").(int64)
	if !known || !isInt {
		return nil
	}
	cells := []Cell{}
	for offset := int64(0); offset < entry.Registers(); offset++ {
		cells = append(cells, Cell{Space: Space(protocol, entry), Address: address + offset})
	}
	return cells
}

// Use is which point of a list holds each cell (its 1-based number in the list).
type Use map[Cell]int

// UseOf is the cell use of a stored list.
func UseOf(protocol string, list []any) Use {
	use := Use{}
	for at, raw := range list {
		if point, ok := raw.(*pyjson.Object); ok {
			use.claim(Cells(protocol, point), at+1)
		}
	}
	return use
}

func (u Use) claim(cells []Cell, number int) {
	for _, cell := range cells {
		if _, taken := u[cell]; !taken {
			u[cell] = number
		}
	}
}

// Taken is the number of the point already holding one of cells (0 when all are free).
func (u Use) Taken(cells []Cell) int {
	for _, cell := range cells {
		if number, taken := u[cell]; taken {
			return number
		}
	}
	return 0
}

// Validate checks a whole list as the page saves it: every point (CheckPoint), no two points
// on one address of one space, at most MaxPoints. Ids are kept or given ("m1", "m2", ...).
func Validate(protocol string, raw any) ([]any, error) {
	if _, known := addressLimit[protocol]; !known {
		return nil, apperr.ValueError("Choose IEC 104, DNP3 or Modbus TCP")
	}
	list, ok := raw.([]any)
	if !ok || len(list) > MaxPoints {
		return nil, apperr.ValueError("Send the point list as a list of at most " + strconv.Itoa(MaxPoints) + " points")
	}
	clean, problems, use := []any{}, []string{}, Use{}
	for at, item := range list {
		point, problem := CheckPoint(protocol, item)
		if problem == "" {
			if other := use.Taken(Cells(protocol, point)); other > 0 {
				problem = AddressWord[protocol] + " " + strconv.FormatInt(Address(point), 10) +
					" is already used by point " + strconv.Itoa(other)
			}
		}
		if problem != "" {
			problems = append(problems, "Point "+strconv.Itoa(at+1)+": "+problem)
			continue
		}
		use.claim(Cells(protocol, point), at+1)
		clean = append(clean, point)
	}
	if len(problems) > 0 {
		if len(problems) > 3 {
			problems = append(problems[:3], "and "+strconv.Itoa(len(problems)-3)+" more")
		}
		return nil, apperr.ValueError(strings.Join(problems, "; "))
	}
	GiveIDs(clean)
	return clean, nil
}

// GiveIDs gives every point without an id (or with a repeated one) the next free "m<n>".
func GiveIDs(list []any) {
	seen, next := map[string]bool{}, 1
	for _, raw := range list {
		id := text(raw.(*pyjson.Object), "id")
		if number, err := strconv.Atoi(strings.TrimPrefix(id, "m")); err == nil && number >= next {
			next = number + 1
		}
	}
	for _, raw := range list {
		point := raw.(*pyjson.Object)
		if id := text(point, "id"); id == "" || seen[id] {
			point.Set("id", "m"+strconv.Itoa(next))
			next++
		}
		seen[text(point, "id")] = true
	}
}

// StoredList is the master block's list for a protocol ("point_list" holds one per protocol).
func StoredList(master *pyjson.Object, protocol string) []any {
	lists, _ := field(master, "point_list").(*pyjson.Object)
	list, _ := field(lists, protocol).([]any)
	return list
}

// WithList is a copy of the master block with the list of a protocol replaced.
func WithList(master *pyjson.Object, protocol string, list []any) *pyjson.Object {
	result := copyObject(master)
	lists := copyObject(field(master, "point_list"))
	return result.Set("point_list", lists.Set(protocol, list))
}

// Address is a stored point's address.
func Address(point *pyjson.Object) int64 {
	address, _ := field(point, "address").(int64)
	return address
}

// CheckIED reports whether the optional "Check against IED" column is on (off by default).
func CheckIED(master *pyjson.Object) bool { return field(master, "check_ied") == true }

func copyObject(raw any) *pyjson.Object {
	result := pyjson.NewObject()
	if source, ok := raw.(*pyjson.Object); ok {
		for _, key := range source.Keys() {
			value, _ := source.Get(key)
			result.Set(key, value)
		}
	}
	return result
}
