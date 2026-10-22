package explorer

import (
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

// typedFake records the typed steps Explorer sends (the wire is covered by tools/control).
type typedFake struct {
	ops    []string
	fields []*pyjson.Object
	reply  *pyjson.Object
}

func (f *typedFake) Operate(string, any, string, any, any) (*pyjson.Object, error) {
	return pyjson.NewObject(), nil
}

func (f *typedFake) Typed(op string, fields *pyjson.Object) (*pyjson.Object, error) {
	f.ops = append(f.ops, op)
	f.fields = append(f.fields, fields)
	return f.reply, nil
}

func connectedService(fake *typedFake) *Service {
	service := NewService("", nil, nil, nil, fake)
	service.state, service.host, service.port = "connected", "127.0.0.1", int64(102)
	return service
}

func TestCommandSendsTypedStepsWithOptions(t *testing.T) {
	fake := &typedFake{reply: pyjson.NewObject().Set("ok", true).Set("selected", true).Set("model", "sbo-enhanced")}
	service := connectedService(fake)
	options := pyjson.NewObject().Set("test", true).Set("orident", "explorer-test")
	answer, err := service.Command("select", "LD/ATCC1.TapChg", "step", "higher", options, true)
	if err != nil {
		t.Fatal(err)
	}
	if summary, _ := answer.Get("summary"); !strings.Contains(summary.(string), "Selected") {
		t.Fatalf("summary: %v", summary)
	}
	data, _ := pyjson.Dumps(fake.fields[0])
	want := `{"ip": "127.0.0.1", "port": 102, "ref": "LD/ATCC1.TapChg", "or_ident": "explorer-test", "or_cat": 3, ` +
		`"interlock_check": false, "synchro_check": false, "test": true, "value": "higher", "kind": "step"}`
	if fake.ops[0] != "typed_select" || string(data) != want {
		t.Fatalf("select fields:\n got %s\nwant %s", data, want)
	}
	if _, err := service.Command("cancel", "LD/ATCC1.TapChg", "step", nil, nil, false); err != nil || fake.ops[1] != "typed_cancel" {
		t.Fatalf("cancel needs no confirmation: %v %v", err, fake.ops)
	}
}

func TestCommandRefusesBeforeSending(t *testing.T) {
	fake := &typedFake{reply: pyjson.NewObject()}
	service := connectedService(fake)
	cases := []struct {
		action, kind string
		value        any
		confirm      bool
		want         string
	}{
		{"operate", "int", int64(1), false, "Confirm the command first"},
		{"jump", "int", int64(1), true, "Action must be select, operate or cancel"},
		{"operate", "int", "x", true, "whole number"},
		{"operate", "step", "up", true, "higher, lower or stop"},
		{"operate", "float", "x", true, "needs a number"},
	}
	for _, c := range cases {
		if _, err := service.Command(c.action, "LD/GGIO1.ISCSO1", c.kind, c.value, nil, c.confirm); err == nil ||
			!strings.Contains(err.Error(), c.want) {
			t.Errorf("%v: %v", c, err)
		}
	}
	if len(fake.ops) != 0 {
		t.Fatalf("nothing may reach the device: %v", fake.ops)
	}
	service.state = "disconnected"
	if _, err := service.Command("operate", "LD/GGIO1.ISCSO1", "int", int64(1), nil, true); err == nil ||
		err.Error() != "Connect to a device first" {
		t.Fatalf("disconnected: %v", err)
	}
}
