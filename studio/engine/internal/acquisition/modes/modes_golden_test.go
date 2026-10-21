package modes

import (
	"strings"
	"testing"

	"studio-engine/internal/acquisition/goldentest"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// dumps compares json.dumps(result) (insertion order) recorded by gen_modes.py.
func dumps(t *testing.T, c goldentest.Case, got any) {
	t.Helper()
	body, err := pyjson.Dumps(got)
	if err != nil {
		t.Fatalf("%s: encode: %v", c.Name, err)
	}
	goldentest.Text(t, c, string(body))
}

// sessionMap turns the recorded {id: status} object into the Go sessions argument.
func sessionMap(object *pyjson.Object) map[string]*pyjson.Object {
	sessions := map[string]*pyjson.Object{}
	if object == nil {
		return sessions
	}
	for _, key := range object.Keys() {
		value, _ := object.Get(key)
		sessions[key] = pyv.Obj(value)
	}
	return sessions
}

func pointsValue(devices []Device, simulation []string) any {
	object := pyjson.NewObject()
	for _, device := range devices {
		var mod any
		if device.Mod != "" {
			mod = device.Mod
		}
		beh := []any{}
		for _, item := range device.Beh {
			beh = append(beh, []any{item.Ref, item.OwnMod})
		}
		object.Set(device.Domain, pyv.O("mod", mod, "beh", beh))
	}
	return []any{object, pyv.Strings(simulation)}
}

func writesValue(writes []Write) any {
	items := []any{}
	for _, write := range writes {
		items = append(items, []any{write.Ref, write.Value})
	}
	return items
}

func run(t *testing.T, c goldentest.Case) {
	t.Helper()
	function, _, _ := strings.Cut(c.Name, "/")
	switch function {
	case "gocb_object":
		dumps(t, c, GocbObject(pyv.Str(c.Args[0])))
	case "gocb_items":
		dumps(t, c, GocbItems(c.Obj(0)))
	case "mode_items":
		dumps(t, c, ModeItems(c.Obj(0)))
	case "parse_answer":
		values, errors := ParseAnswer(c.Obj(0))
		dumps(t, c, []any{values, errors})
	case "mode_name":
		var got any
		if name, ok := ModeName(c.Args[0]); ok {
			got = name
		}
		dumps(t, c, got)
	case "mode_signals":
		dumps(t, c, ModeSignals(pyv.List(c.Args[0])))
	case "derive_beh":
		dumps(t, c, DeriveBeh(c.Args[0], c.Args[1]))
	case "mode_points":
		dumps(t, c, pointsValue(ModePoints(c.Obj(0))))
	case "validate_test_mode":
		settings, err := ValidateTestMode(c.Args[0])
		if err != nil {
			goldentest.Text(t, c, "error:"+err.Error())
			return
		}
		dumps(t, c, settings)
	case "mode_writes":
		dumps(t, c, writesValue(ModeWrites(c.Obj(0), c.Obj(1), c.Obj(2))))
	case "mode_summary":
		dumps(t, c, ModeSummary(c.Obj(0), c.Obj(1)))
	case "mode_badges":
		running, _ := c.Args[3].(bool)
		dumps(t, c, ModeBadges(pyv.Objects(pyv.List(c.Args[0])), sessionMap(c.Obj(1)), c.Obj(2), running))
	case "simulation_sources":
		dumps(t, c, SimulationSources(c.Obj(0), c.Obj(1)))
	default:
		t.Fatalf("%s: no Go function for this golden case", c.Name)
	}
}

func TestGoldenReads(t *testing.T) {
	for _, c := range goldentest.Load(t, "modes_reads.json") {
		run(t, c)
	}
}

func TestGoldenTestMode(t *testing.T) {
	for _, c := range goldentest.Load(t, "modes_test_mode.json") {
		run(t, c)
	}
}

func TestGoldenBadges(t *testing.T) {
	for _, c := range goldentest.Load(t, "modes_badges.json") {
		run(t, c)
	}
}
