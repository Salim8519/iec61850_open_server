package gateway

import (
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// groups are the discovered point groups in display order (master_points.GROUPS).
var groups = []string{"single", "double", "step", "bitstring", "normalized", "scaled", "float", "counter",
	"binary", "double_binary", "analog", "frozen_counter", "binary_output", "analog_output",
	"coil", "discrete_input", "holding_register", "input_register"}

// Key is key(value): "<group>:<address>".
func Key(value *pyjson.Object) string {
	return pytext.Str(get(value, "group")) + ":" + pytext.Str(get(value, "address"))
}

// Merge is merge(table, values): folds a batch of master_values into table (key -> row,
// changed in place exactly as Python changes it, also when an error stops the batch) and
// returns the new keys. int() of an odd quality fails as in Python.
func Merge(table *pyjson.Object, values []any) ([]string, error) {
	added := []string{}
	for _, item := range values {
		group, err := getAny(item, "group", nil)
		if err != nil {
			return nil, err
		}
		value := item.(*pyjson.Object)
		if !contains(groups, group) || !intLike(get(value, "address")) {
			continue
		}
		name := Key(value)
		var row *pyjson.Object
		if existing, found := table.Get(name); found && existing != nil {
			row, _ = existing.(*pyjson.Object)
		}
		if row == nil {
			row = pyjson.NewObject().Set("key", name).Set("group", group).Set("address", get(value, "address")).
				Set("changes", int64(0))
			table.Set(name, row)
			added = append(added, name)
		} else if changed, err := rowChanged(row, value); err != nil {
			return nil, err
		} else if changed {
			changes, _ := get(row, "changes").(int64)
			row.Set("changes", changes+1) // a repeated interrogation of the same value is not a change
		}
		quality, err := pyInt(getOr(value, "quality", int64(0)))
		if err != nil {
			return nil, err
		}
		row.Set("type", getOr(value, "type", "")).Set("value", get(value, "value")).Set("quality", quality).
			Set("time_ms", get(value, "time_ms")).Set("cause", getOr(value, "cause", ""))
	}
	return added, nil
}

// rowChanged is row["value"] != value.get("value") or row["quality"] != int(quality).
func rowChanged(row, value *pyjson.Object) (bool, error) {
	old, err := subscript(row, "value")
	if err != nil {
		return false, err
	}
	if !pytext.Equal(old, get(value, "value")) {
		return true, nil
	}
	oldQuality, err := subscript(row, "quality")
	if err != nil {
		return false, err
	}
	quality, err := pyInt(getOr(value, "quality", int64(0)))
	if err != nil {
		return false, err
	}
	return !pytext.Equal(oldQuality, quality), nil
}
