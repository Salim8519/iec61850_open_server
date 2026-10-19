package gateway

import (
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// Protocol converter (passthrough.py): every watched device point becomes one upstream
// point at the same address when free, else the next free one.
var protocolNames = map[string]string{"iec104": "IEC 104", "dnp3": "DNP3", "modbus": "Modbus"}

var kindFor = map[string]string{
	"single": "single", "binary": "single", "coil": "single", "discrete_input": "single", "binary_output": "single",
	"double": "double", "double_binary": "double",
	"step": "integer", "bitstring": "integer", "counter": "integer", "frozen_counter": "integer",
	"normalized": "float", "scaled": "float", "float": "float", "analog": "float", "analog_output": "float",
	"holding_register": "float", "input_register": "float",
}

// MapPoints is map_points(rows, protocol, routes): the upstream points for the watched
// device rows (master Rows output) and the user's validated command routes, plus notes.
// Rows and routes outside those shapes give the Python crash as a plain error.
func MapPoints(rows []any, protocol string, routes []any) ([]any, []string, error) {
	used, points, notes := map[Cell]bool{}, []any{}, []string{}
	for _, value := range rows {
		ticked, err := getAny(value, "ticked", true)
		if err != nil {
			return nil, nil, err
		}
		if !pytext.Truthy(ticked) {
			continue
		}
		row := value.(*pyjson.Object)
		if err := mapRow(row, protocol, used, &points, &notes); err != nil {
			return nil, nil, err
		}
	}
	for index, value := range routes {
		number := itoa(index + 1)
		route, ok := value.(*pyjson.Object)
		if !ok {
			_, err := subscript(value, "upstream_kind")
			return nil, nil, err
		}
		fields, err := subscripts(route, "upstream_kind", "upstream", "kind", "address")
		if err != nil {
			return nil, nil, err
		}
		link := pyjson.NewObject().Set("kind", fields[2]).Set("address", fields[3]).
			Set("select", getOr(route, "select", true))
		command, err := place(protocol, pyjson.NewObject().Set("id", "route:"+number).Set("kind", fields[0]).
			Set("address", fields[1]).Set("scale", 1.0).Set("ied", "").Set("ref", "route "+number).
			Set("description", "Command to device "+pytext.Str(fields[2])+" "+pytext.Str(fields[3])).
			Set("source", "remote").Set("route", link), used)
		if err != nil {
			return nil, nil, err
		}
		if command == nil || !pytext.Equal(get(command, "address"), fields[1]) {
			notes = append(notes, "Command route "+number+": upstream address "+pytext.Str(fields[1])+" is already used")
			continue
		}
		points = append(points, command)
	}
	return points, notes, nil
}

func subscripts(object *pyjson.Object, keys ...string) ([]any, error) {
	values := make([]any, len(keys))
	for index, key := range keys {
		value, err := subscript(object, key)
		if err != nil {
			return nil, err
		}
		values[index] = value
	}
	return values, nil
}

// mapRow serves one watched row, plus its command when the device command index equals
// its status index (binary outputs and coils).
func mapRow(row *pyjson.Object, protocol string, used map[Cell]bool, points *[]any, notes *[]string) error {
	label := get(row, "label") // the description keeps the label value itself
	if !pytext.Truthy(label) {
		fields, err := subscripts(row, "group_name", "address")
		if err != nil {
			return err
		}
		label = pytext.Str(fields[0]) + " " + pytext.Str(fields[1])
	}
	name := pytext.Str(label)
	key, err := subscript(row, "key")
	if err != nil {
		return err
	}
	kind, err := rowKind(row)
	if err != nil {
		return err
	}
	fields, err := subscripts(row, "address", "key", "key")
	if err != nil {
		return err
	}
	address := fields[0]
	point, err := place(protocol, pyjson.NewObject().Set("id", "remote:"+pytext.Str(key)).Set("kind", kind).
		Set("address", address).Set("scale", 1.0).Set("ied", "").Set("ref", key).Set("description", label).
		Set("source", "remote").Set("remote", key), used)
	if err != nil {
		return err
	}
	if point == nil {
		*notes = append(*notes, name+" is not served: no free address left in this protocol")
		return nil
	}
	if !pytext.Equal(get(point, "address"), address) {
		*notes = append(*notes, name+" is served at address "+pytext.Str(get(point, "address"))+
			" ("+whyMoved(protocol, address)+")")
	}
	*points = append(*points, point)
	group := get(row, "group")
	if group != "binary_output" && group != "coil" {
		return nil
	}
	command, err := place(protocol, pyjson.NewObject().Set("id", "remote-command:"+pytext.Str(key)).
		Set("kind", "command_single").Set("address", address).Set("scale", 1.0).Set("ied", "").Set("ref", key).
		Set("description", "Command "+name).Set("source", "remote").
		Set("route", pyjson.NewObject().Set("kind", group).Set("address", address)), used)
	if err == nil && command != nil {
		*points = append(*points, command)
	}
	return err
}

// rowKind is _kind(row): the upstream type of a device group.
func rowKind(row *pyjson.Object) (string, error) {
	group, err := subscript(row, "group")
	if err != nil {
		return "", err
	}
	if err := hashable(group); err != nil {
		return "", err
	}
	text, _ := group.(string)
	kind, known := kindFor[text]
	if _, isText := group.(string); !known || !isText {
		return "", keyError(group)
	}
	if (text == "holding_register" || text == "input_register") &&
		contains([]string{"uint16", "int16", "uint32", "int32"}, get(row, "type")) {
		return "integer", nil
	}
	return kind, nil
}

func whyMoved(protocol string, address any) string {
	low := AddressRange[protocol][0]
	if cmpInt(address, low) < 0 {
		return protocolNames[protocol] + " has no address " + pytext.Str(address)
	}
	return pytext.Str(address) + " was taken"
}

// place is _place: the point at its address when free, else the next free one; nil when
// the protocol has no room. Cells are taken even when the caller then refuses the point.
func place(protocol string, point *pyjson.Object, used map[Cell]bool) (*pyjson.Object, error) {
	bounds, ok := AddressRange[protocol]
	if !ok {
		return nil, keyError(protocol)
	}
	kind, isText := get(point, "kind").(string)
	address := get(point, "address")
	if !isText || !intLike(address) {
		return nil, crash("TypeError", "the address of a converter point must be an int") // unreachable
	}
	if cmpInt(bounds[0], address) > 0 { // max(address, low) keeps address when equal
		address = bounds[0]
	}
	for cmpInt(address, bounds[1]) <= 0 {
		at, _ := pytext.Int64(address)
		taken := cells(protocol, kind, at)
		if !anyUsed(used, taken) {
			for _, cell := range taken {
				used[cell] = true
			}
			return copyObject(point).Set("address", address), nil
		}
		address = at + 1
	}
	return nil, nil
}

// DeviceValue is device_value(route, value): upstream ON/OFF to the device command value.
func DeviceValue(route *pyjson.Object, value bool) int64 {
	switch {
	case get(route, "kind") == "double" && value:
		return 2
	case get(route, "kind") == "double", value:
		return 1
	}
	return 0
}

// RemoteSample is remote_sample(point, row, protocol_good, link_open); row nil is None.
// int()/float() of an odd device value fails as in Python (ValueError or a plain error).
func RemoteSample(point, row *pyjson.Object, protocolGood, linkOpen bool) (*pyjson.Object, error) {
	if row == nil || get(row, "value") == nil {
		id, err := subscript(point, "id")
		if err != nil {
			return nil, err
		}
		return pyjson.NewObject().Set("id", id).Set("value", int64(0)).Set("quality", int64(0)).
			Set("time_ms", int64(0)).Set("source_ok", false), nil
	}
	kind, err := subscript(point, "kind")
	if err != nil {
		return nil, err
	}
	var value any
	if contains([]string{"single", "double", "integer"}, kind) {
		value, err = pyInt(get(row, "value"))
	} else {
		value, err = pyFloat(get(row, "value"))
	}
	if err != nil {
		return nil, err
	}
	id, err := subscript(point, "id")
	if err != nil {
		return nil, err
	}
	good := linkOpen && protocolGood
	quality, timeMS := int64(2), get(row, "time_ms")
	if good {
		quality = 0
	}
	if !pytext.Truthy(timeMS) {
		timeMS = int64(0)
	}
	return pyjson.NewObject().Set("id", id).Set("value", value).Set("quality", quality).
		Set("time_ms", timeMS).Set("source_ok", good), nil
}
