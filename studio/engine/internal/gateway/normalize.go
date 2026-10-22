package gateway

import (
	"strconv"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// NormalizeGateway is normalize_gateway(raw): the stored gateway section as a checked
// structure. Point references are checked at Start, so a removed IED never deletes points.
// Python adds a missing point "id" in place; here the points are copied with the id added
// and the input stays untouched.
func NormalizeGateway(raw any) (*pyjson.Object, error) {
	if raw == nil {
		return EmptyGateway(), nil
	}
	section, ok := raw.(*pyjson.Object)
	if !ok {
		return nil, valueError("The gateway section must be an object")
	}
	result := EmptyGateway()
	protocol := getOr(section, "protocol", "iec104")
	if !contains(Protocols, protocol) {
		return nil, valueError("Unknown gateway protocol '" + pytext.Str(protocol) + "'")
	}
	if _, err := ListenAddress(getOr(section, "listen", "this_pc")); err != nil {
		return nil, err
	}
	role, err := ValidateRole(getOr(section, "role", "gateway"))
	if err != nil {
		return nil, err
	}
	master, err := NormalizeMaster(get(section, "master"))
	if err != nil {
		return nil, err
	}
	result.Set("protocol", protocol).Set("listen", getOr(section, "listen", "this_pc")).Set("role", role).
		Set("master", master)
	for _, flag := range []string{"real_commands", "auto_start"} {
		if value, isBool := get(section, flag).(bool); isBool {
			result.Set(flag, value)
		}
	}
	settings := get(result, "settings").(*pyjson.Object)
	stored := get(result, "points").(*pyjson.Object)
	for _, name := range Protocols {
		values, err := orGet(get(section, "settings"), name, nil)
		if err != nil {
			return nil, err
		}
		checked, err := ValidateSettings(name, values)
		if err != nil {
			return nil, err
		}
		settings.Set(name, checked)
		points, err := storedPoints(section, name)
		if err != nil {
			return nil, err
		}
		stored.Set(name, points)
	}
	return result, nil
}

// storedPoints checks one protocol's stored point list and gives each point its id.
func storedPoints(section *pyjson.Object, name string) ([]any, error) {
	value, err := orGet(get(section, "points"), name, []any{})
	if err != nil {
		return nil, err
	}
	list, ok := value.([]any)
	if !ok || len(list) > MaxPoints {
		return nil, valueError("The " + name + " point list must be a list of at most " +
			strconv.Itoa(MaxPoints) + " points")
	}
	points := make([]any, len(list))
	for index, item := range list {
		point, isDict := item.(*pyjson.Object)
		if !isDict || !KnownKind(get(point, "kind")) || !intLike(get(point, "address")) {
			return nil, valueError("A stored " + name + " gateway point has no valid type or address")
		}
		copied := copyObject(point)
		if _, found := copied.Get("id"); !found {
			copied.Set("id", pointIDOf(getOr(point, "ied", ""), getOr(point, "ref", ""), get(point, "kind")))
		}
		points[index] = copied
	}
	return points, nil
}
