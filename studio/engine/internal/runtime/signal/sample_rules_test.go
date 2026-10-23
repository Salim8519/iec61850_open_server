package signal

import (
	"strings"
	"testing"
)

// Writes the wire would carry wrongly are refused in plain words (change 0186): an Enum value
// outside its EnumType (TMODE-10), non-ASCII text in a VisString (ISO 646) and a bad quality.
func TestSampleRefusesValuesTheTypeCannotHold(t *testing.T) {
	mod := Signal{Ref: "LD/LLN0.Mod.stVal", Kind: "Enum", FC: "ST", Writable: true,
		Enums: []EnumChoice{{"on", 1}, {"on-blocked", 2}, {"test", 3}, {"test/blocked", 4}, {"off", 5}}}
	name := Signal{Ref: "LD/LLN0.NamPlt.vendor", Kind: "VisString255", FC: "DC", Writable: true}
	stVal := Signal{Ref: "LD/GGIO1.Ind1.stVal", Kind: "BOOLEAN", FC: "ST", Writable: true}
	quality := Signal{Ref: "LD/GGIO1.Ind1.q", Kind: "Quality", FC: "ST", Writable: true}
	known := map[string]Signal{quality.Ref: quality}
	lookup := func(ref string) (Signal, bool) { item, ok := known[ref]; return item, ok }
	cases := []struct {
		item    Signal
		value   any
		quality any
		want    string // "" = accepted
	}{
		{mod, int64(3), nil, ""},
		{mod, int64(9), nil, "9 is not a choice of LD/LLN0.Mod.stVal. Use one of: 1 (on), 2 (on-blocked), 3 (test), 4 (test/blocked), 5 (off)"},
		{mod, int64(0), nil, "0 is not a choice"},
		{Signal{Ref: "LD/X.Y.stVal", Kind: "Enum", FC: "ST", Writable: true}, int64(9), nil, ""}, // no EnumType known: range only
		{name, "ABB ~ 1.0", nil, ""},
		{name, "Café", nil, "VisString text takes plain ASCII characters only (letters, digits, space and ! to ~); remove \"é\""},
		{name, "tab\there", nil, "VisString text takes plain ASCII characters only"},
		{stVal, true, "bad", "Quality must be a whole number from 0 to 8191 (the 13 quality bits)"},
		{stVal, true, int64(9000), "Quality must be a whole number from 0 to 8191 (the 13 quality bits)"},
		{stVal, true, int64(64), ""},
	}
	for _, test := range cases {
		_, _, err := MakeSample(test.item, lookup, test.value, 1, test.quality)
		switch {
		case test.want == "" && err != nil:
			t.Errorf("%s=%v refused: %v", test.item.Ref, test.value, err)
		case test.want != "" && (err == nil || !strings.HasPrefix(err.Error(), test.want)):
			t.Errorf("%s=%v q=%v: got %v, want %q", test.item.Ref, test.value, test.quality, err, test.want)
		}
	}
}
