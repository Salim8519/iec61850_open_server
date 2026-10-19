package pytext

import (
	"math"
	"testing"

	"studio-engine/internal/pyjson"
)

// Expected texts were printed by CPython 3.12 (the studio venv).
func TestReprAndStr(t *testing.T) {
	object := pyjson.NewObject().Set("a", []any{int64(1), 2.0, nil, true, "x'y"})
	cases := map[string]string{
		Repr("it's"):   `"it's"`,
		Repr("a\"b'c"): `'a"b\'c'`,
		Repr("\x07\xe2\x80\x8b\xc3\xa9\U0001F600\xc2\x85\xc2\xa0"): `'\x07\` + "u200b\xc3\xa9\U0001F600" + `\x85\xa0'`,
		Str(object):       `{'a': [1, 2.0, None, True, "x'y"]}`,
		Str(1e16):         "1e+16",
		Str(math.Inf(-1)): "-inf",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("got %s want %s", got, want)
		}
	}
}

func TestParseInt(t *testing.T) {
	good := map[string]int64{"1_0": 10, " 7 ": 7, "+5": 5, "٣": 3, "-0": 0, "  12　": 12}
	for text, want := range good {
		if got, err := ParseInt(text); err != nil || got != want {
			t.Errorf("int(%q) = %v %v", text, got, err)
		}
	}
	bad := map[string]string{
		"1__0":      `invalid literal for int() with base 10: '1__0'`,
		"0x10":      `invalid literal for int() with base 10: '0x10'`,
		"1.5":       `invalid literal for int() with base 10: '1.5'`,
		"":          `invalid literal for int() with base 10: ''`,
		"\x1c5\x1f": `invalid literal for int() with base 10: '\x1c5\x1f'`,
		"+-1":       `invalid literal for int() with base 10: '+-1'`,
	}
	for text, want := range bad {
		if _, err := ParseInt(text); err == nil || err.Error() != want {
			t.Errorf("int(%q) error %v want %s", text, err, want)
		}
	}
}

func TestParseFloat(t *testing.T) {
	good := map[string]float64{"1_0.5": 10.5, " 1e3 ": 1000, "1.": 1, ".5": 0.5, "١.٥": 1.5,
		"1e1_0": 1e10, "+.5e-3": 0.0005, "-Infinity": math.Inf(-1), "1e999": math.Inf(1)}
	for text, want := range good {
		if got, err := ParseFloat(text); err != nil || got != want {
			t.Errorf("float(%q) = %v %v", text, got, err)
		}
	}
	if got, err := ParseFloat("nAn"); err != nil || !math.IsNaN(got) {
		t.Errorf("float(nAn) = %v %v", got, err)
	}
	for _, text := range []string{"0x1p3", "1e", "_1", "", ".", "1_", "e5", "--1", "inf5"} {
		want := "could not convert string to float: " + ReprString(text)
		if _, err := ParseFloat(text); err == nil || err.Error() != want {
			t.Errorf("float(%q) error %v", text, err)
		}
	}
}

func TestFormatGAndEqual(t *testing.T) {
	cases := map[float64]string{1e16: "1e+16", 0.0001: "0.0001", 123456789.0: "1.23457e+08", 100000: "100000",
		1: "1", 230.5: "230.5", 0.00001: "1e-05"}
	for value, want := range cases {
		if got := FormatG(value, 6); got != want {
			t.Errorf("format(%v) = %s want %s", value, got, want)
		}
	}
	if FormatG(math.Copysign(0, -1), 6) != "-0" || FormatG(math.NaN(), 6) != "nan" {
		t.Error("special values")
	}
	if !Equal(int64(1), 1.0) || !Equal(true, int64(1)) || Equal(int64(1), "1") || Equal(math.NaN(), math.NaN()) {
		t.Error("python equality")
	}
	if Equal(int64(9007199254740993), 9007199254740992.0) {
		t.Error("int == float must compare exactly")
	}
}
