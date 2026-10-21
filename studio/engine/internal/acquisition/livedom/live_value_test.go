package livedom

import (
	"testing"

	"studio-engine/internal/acquisition/goldentest"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// dumps is json.dumps(value) as text, failing the test on encoder errors.
func dumps(t *testing.T, value any) string {
	t.Helper()
	body, err := pyjson.Dumps(value)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	return string(body)
}

func pyBool(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

func TestLiveValueMatchesPython(t *testing.T) {
	for _, c := range goldentest.Load(t, "livedom_value.json") {
		switch pyv.Str(c.Args[0]) {
		case "convert":
			value, err := Convert(pyv.Str(c.Args[1]), c.Args[2])
			if err != nil {
				goldentest.Text(t, c, "ValueError: "+err.Error())
				continue
			}
			goldentest.Text(t, c, dumps(t, pyv.O("value", value)))
		case "type_matches":
			goldentest.Text(t, c, pyBool(TypeMatches(pyv.Str(c.Args[1]), c.Args[2])))
		case "describe":
			goldentest.Text(t, c, DescribeMMS(c.Args[1]))
		default:
			t.Fatalf("%s: unknown case", c.Name)
		}
	}
}

func TestDescribeDeviceFindsFloatCodes(t *testing.T) {
	cases := map[float64]string{4.0: "integer", -0.0: "array", 99.0: "MMS type 99.0", 4.5: "MMS type 4.5",
		1e20: "MMS type 1e+20"}
	for code, want := range cases {
		if got := describeDevice(code); got != want {
			t.Errorf("describeDevice(%v) = %q, want %q", code, got, want)
		}
	}
}
