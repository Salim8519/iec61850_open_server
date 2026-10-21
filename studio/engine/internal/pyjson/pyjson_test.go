package pyjson

import (
	"math"
	"math/big"
	"strings"
	"testing"
)

// py turns the placeholder %u into a JSON \u escape, so expected Python output stays readable.
func py(text string) string { return strings.ReplaceAll(text, "%u", "\\"+"u") }

func TestFloatReprMatchesPython(t *testing.T) {
	cases := map[float64]string{
		1: "1.0", 0.1: "0.1", 1e15: "1000000000000000.0", 1e16: "1e+16", 1.5e16: "1.5e+16",
		9999999999999998: "9999999999999998.0", 0.0001: "0.0001", 0.00001: "1e-05", 1.25e-7: "1.25e-07",
		5e-324: "5e-324", 1.7976931348623157e308: "1.7976931348623157e+308", 123456.789: "123456.789",
	}
	for value, want := range cases {
		if got := Repr(value); got != want {
			t.Errorf("Repr(%v) = %q, want %q", value, got, want)
		}
	}
	tenth, fifth := 0.1, 0.2
	if got := Repr(tenth + fifth); got != "0.30000000000000004" {
		t.Errorf("Repr(0.1+0.2) = %q", got)
	}
	if got := string(AppendFloat(nil, math.Copysign(0, -1))); got != "-0.0" {
		t.Errorf("negative zero = %q", got)
	}
	for value, want := range map[float64]string{math.NaN(): "NaN", math.Inf(1): "Infinity", math.Inf(-1): "-Infinity"} {
		if got := string(AppendFloat(nil, value)); got != want {
			t.Errorf("AppendFloat(%v) = %q, want %q", value, got, want)
		}
	}
}

func TestFlaskBodyMatchesRecordedFlaskOutput(t *testing.T) {
	// Recorded from Flask 3.1.2 jsonify in this repository's venv.
	inner := NewObject().Set("z", 2).Set("y", math.NaN())
	value := NewObject().Set("b", 1.0).Set("a", []any{1, 1e-05, 1e16, "é\x7f<&>", nil, true}).Set("c", inner)
	got, err := Flask(value)
	if err != nil {
		t.Fatal(err)
	}
	want := py(`{"a":[1,1e-05,1e+16,"%u00e9%u007f<&>",null,true],"b":1.0,"c":{"y":NaN,"z":2}}`) + "\n"
	if string(got) != want {
		t.Errorf("Flask body\n got %s\nwant %s", got, want)
	}
	dumps, _ := Dumps(value)
	if want := py(`{"b": 1.0, "a": [1, 1e-05, 1e+16, "%u00e9%u007f<&>", null, true], "c": {"z": 2, "y": NaN}}`); string(dumps) != want {
		t.Errorf("Dumps\n got %s\nwant %s", dumps, want)
	}
}

func TestStringEscapes(t *testing.T) {
	got := string(AppendString(nil, "\"\\\b\f\n\r\t\x01\x1f é\U0001F600/"))
	want := py(`"\"\\\b\f\n\r\t%u0001%u001f %u00e9%ud83d%ude00/"`)
	if got != want {
		t.Errorf("got %s want %s", got, want)
	}
}

func TestDecodeKeepsPythonSemantics(t *testing.T) {
	input := py(` {"b": 1, "a": 1.0, "b": [true, null, -0, -0.0, 1e400, 123456789012345678901234567890, NaN, -Infinity, "%ud83d%ude00%u00e9"]} `)
	value, err := Decode([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	object := value.(*Object)
	if keys := object.Keys(); len(keys) != 2 || keys[0] != "b" || keys[1] != "a" {
		t.Fatalf("a repeated key keeps its first place: %v", keys)
	}
	if a, _ := object.Get("a"); a != 1.0 {
		t.Errorf("1.0 must stay a float, got %#v", a)
	}
	items, _ := object.Get("b")
	list := items.([]any)
	if list[2] != int64(0) || !math.Signbit(list[3].(float64)) || !math.IsInf(list[4].(float64), 1) {
		t.Errorf("numbers: %#v", list[:5])
	}
	if large, ok := list[5].(*big.Int); !ok || large.String() != "123456789012345678901234567890" {
		t.Errorf("big integer: %#v", list[5])
	}
	if !math.IsNaN(list[6].(float64)) || !math.IsInf(list[7].(float64), -1) || list[8] != "\U0001F600é" {
		t.Errorf("literals and escapes: %#v", list[6:])
	}
	compact, _ := Compact(value)
	want := py(`{"b":[true,null,0,-0.0,Infinity,123456789012345678901234567890,NaN,-Infinity,"%ud83d%ude00%u00e9"],"a":1.0}`)
	if string(compact) != want {
		t.Errorf("round trip\n got %s\nwant %s", compact, want)
	}
}

func TestDecodeRejectsWhatPythonRejects(t *testing.T) {
	for _, input := range []string{``, `01`, `1.`, `[1,]`, `{"a" 1}`, `{a:1}`, "\"tab\there\"", `"\x"`, `[1] 2`, `-`, `tru`, `1e`} {
		if _, err := Decode([]byte(input)); err == nil {
			t.Errorf("Decode(%q) accepted; Python rejects it", input)
		}
	}
}

func TestObjectOrderAfterDelete(t *testing.T) {
	object := NewObject().Set("a", 1).Set("b", 2).Set("c", 3)
	object.Delete("a")
	object.Set("a", 4)
	if out, _ := Compact(object); string(out) != `{"b":2,"c":3,"a":4}` {
		t.Errorf("got %s", out)
	}
}
