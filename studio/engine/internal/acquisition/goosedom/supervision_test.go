package goosedom

import (
	"testing"

	"studio-engine/internal/acquisition/goldentest"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

func TestSupervisionMatchesPython(t *testing.T) {
	run(t, "goose_supervision.json", map[string]func(goldentest.Case) any{
		"watch_list": func(c goldentest.Case) any { return WatchList(objs(c.Args[0])) },
		"lgos_updates": func(c goldentest.Case) any {
			index := map[string]*pyjson.Object{}
			table := c.Obj(2)
			for _, key := range table.Keys() {
				value, _ := table.Get(key)
				index[key] = pyv.Obj(value)
			}
			return updates(LgosUpdates(c.Args[0], c.Obj(1), index))
		},
		"dataset_values": func(c goldentest.Case) any { return DatasetValues(c.Obj(0), c.Obj(1), c.Args[2]) },
	})
}

func TestInputsMatchPython(t *testing.T) {
	run(t, "goose_inputs.json", map[string]func(goldentest.Case) any{
		"bind_inputs": func(c goldentest.Case) any {
			bindings, problems := BindInputs(c.Obj(0), c.Obj(1))
			return []any{objectList(bindings), texts(problems)}
		},
		"input_updates": func(c goldentest.Case) any {
			writes, problems := InputUpdates(objs(c.Args[0]), pyv.List(c.Args[1]), c.Obj(2))
			return []any{updates(writes), texts(problems)}
		},
		"accept_frame": func(c goldentest.Case) any {
			count, _ := pyv.Int64(c.Args[1])
			return AcceptFrame(c.Obj(0), count)
		},
	})
}

func TestLgosPointsOrder(t *testing.T) {
	body, _ := pyjson.Dumps(LgosPoints())
	want := `{"St.stVal": "receiving", "SimSt.stVal": "test", "LastStNum.stVal": "st_num", ` +
		`"ConfRevNum.stVal": "conf_rev"}`
	if string(body) != want {
		t.Fatalf("LgosPoints = %s", body)
	}
}
