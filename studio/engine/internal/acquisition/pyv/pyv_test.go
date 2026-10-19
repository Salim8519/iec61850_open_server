package pyv

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
)

func TestCasefoldMatchesPython(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "testdata", "generated", "casefold.json"))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := pyjson.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	want := map[rune]string{}
	for _, pair := range GetList(Obj(decoded), "pairs") {
		items := List(pair)
		code, _ := Int64(items[0])
		var folded []rune
		for _, item := range List(items[1]) {
			value, _ := Int64(item)
			folded = append(folded, rune(value))
		}
		want[rune(code)] = string(folded)
	}
	for r := rune(0); r <= 0x10FFFF; r++ {
		if r >= 0xD800 && r <= 0xDFFF {
			continue
		}
		expected, found := want[r]
		if !found {
			expected = string(r)
		}
		if got := Casefold(string(r)); got != expected {
			t.Fatalf("casefold U+%04X: got %q want %q", r, got, expected)
		}
	}
	if Casefold("Straße XCBR") != "strasse xcbr" {
		t.Fatal("mixed text")
	}
}

func TestEqualFollowsPython(t *testing.T) {
	cases := []struct {
		a, b any
		want bool
	}{
		{int64(1), 1.0, true}, {true, int64(1), true}, {false, 0.0, true}, {"1", int64(1), false},
		{nil, nil, true}, {nil, false, false}, {[]any{int64(1)}, []any{1.0}, true},
		{O("a", int64(1)), O("a", true), true}, {O("a", 1), O("b", 1), false},
	}
	for _, c := range cases {
		if Equal(c.a, c.b) != c.want {
			t.Errorf("Equal(%v, %v) != %v", c.a, c.b, c.want)
		}
	}
}

func TestReprAndTime(t *testing.T) {
	if got := Repr([]any{"a'b", int64(2), 1.5, nil, true, O("k", "v")}); got != `["a'b", 2, 1.5, None, True, {'k': 'v'}]` {
		t.Fatal(got)
	}
	stamp := time.Date(2026, 10, 20, 8, 15, 2, 999, time.UTC)
	if ISOSeconds(stamp) != "2026-10-20T08:15:02+00:00" {
		t.Fatal(ISOSeconds(stamp))
	}
	if parsed, ok := ParseAware("2026-10-20T08:15:02+00:00"); !ok || !parsed.Equal(stamp.Truncate(time.Second)) {
		t.Fatal("parse")
	}
	if _, ok := ParseAware("2026-10-20T08:15:02"); ok {
		t.Fatal("naive must not parse")
	}
	if RoundHalfEven(2.5) != 2 || RoundHalfEven(3.5) != 4 || Format04X(int64(4096)) != "1000" {
		t.Fatal("round/format")
	}
}
