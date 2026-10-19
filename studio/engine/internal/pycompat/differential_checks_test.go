package pycompat

import (
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"time"

	"studio-engine/internal/pyjson"
)

func checkTable(c vector) bool {
	codes, _ := c.get("codes").([]any)
	name := c.str("name")
	expected := map[rune]string{}
	for _, item := range codes {
		switch entry := item.(type) {
		case int64:
			expected[rune(entry)] = "1"
		case []any:
			lo := rune(entry[0].(int64))
			switch value := entry[1].(type) {
			case int64:
				if name == "printable" {
					for r := lo; r <= rune(value); r++ {
						expected[r] = "1"
					}
				} else {
					expected[lo] = strconv.FormatInt(value, 10)
				}
			case string:
				expected[lo] = value
			}
		}
	}
	differences := 0
	for r := rune(0); r < 0x110000; r++ {
		if r >= 0xd800 && r <= 0xdfff {
			continue
		}
		got, want := tableValue(name, r), expected[r]
		if name == "casefold" && want == "" {
			want = string(r)
		}
		if got != want {
			differences++
			if differences < 10 {
				c.t.Errorf("table %s U+%04X: got %q, CPython %q", name, r, got, want)
			}
		}
	}
	return differences == 0
}

func tableValue(name string, r rune) string {
	switch name {
	case "isspace":
		return flag(IsSpace(r))
	case "printable":
		return flag(IsPrintable(r))
	case "decimal":
		if value := DecimalValue(r); value >= 0 {
			return strconv.Itoa(value)
		}
		return ""
	}
	return CasefoldRune(r)
}

func flag(on bool) string {
	if on {
		return "1"
	}
	return ""
}

func checkInt(c vector) bool {
	value, err := ParseInt(c.str("in"))
	got := ""
	switch number := value.(type) {
	case int64:
		got = strconv.FormatInt(number, 10)
	case *big.Int:
		got = number.String()
	}
	return c.expect(fmt.Sprintf("int(%q)", truncate(c.str("in"))), got, err)
}

func truncate(text string) string {
	if len(text) > 60 {
		return text[:60] + "..."
	}
	return text
}

func checkRound(c vector) bool {
	x := c.number("x")
	if n, has := c.get("n").(int64); has {
		rounded, err := RoundDigits(x, int(n))
		got := ""
		if err == nil {
			got = floatRepr(rounded)
		}
		return c.expect(fmt.Sprintf("round(%v, %d)", x, n), got, err)
	}
	value, err := Round(x)
	got := ""
	switch number := value.(type) {
	case int64:
		got = strconv.FormatInt(number, 10)
	case *big.Int:
		got = number.String()
	}
	return c.expect(fmt.Sprintf("round(%v)", x), got, err)
}

// floatRepr is repr(float), with Python's nan/inf spellings.
func floatRepr(value float64) string {
	switch {
	case math.IsNaN(value):
		return "nan"
	case math.IsInf(value, 1):
		return "inf"
	case math.IsInf(value, -1):
		return "-inf"
	}
	return pyjson.Repr(value)
}

func checkFromIsoformat(c vector) bool {
	d, err := FromIsoformat(c.str("in"))
	got := ""
	if err == nil {
		got = fmt.Sprint(dateTimeList(d))
	}
	want := c.get("out")
	if list, ok := want.([]any); ok {
		c.o.Set("out", fmt.Sprint(list))
	}
	return c.expect(fmt.Sprintf("fromisoformat(%q)", c.str("in")), got, err)
}

func dateTimeList(d DateTime) []any {
	var offset any
	if d.Aware {
		offset = int64(d.Offset / time.Microsecond)
	}
	return []any{int64(d.Year), int64(d.Month), int64(d.Day), int64(d.Hour), int64(d.Minute),
		int64(d.Second), int64(d.Microsecond), offset}
}

func checkIsoformat(c vector) bool {
	fields := c.get("dt").([]any)
	n := func(i int) int { return int(fields[i].(int64)) }
	d := DateTime{Year: n(0), Month: n(1), Day: n(2), Hour: n(3), Minute: n(4), Second: n(5), Microsecond: n(6)}
	if micro, aware := fields[7].(int64); aware {
		d.Aware, d.Offset = true, time.Duration(micro)*time.Microsecond
	}
	got, err := Isoformat(d, c.str("spec"))
	return c.expect(fmt.Sprintf("isoformat(%v, %s)", fields, c.str("spec")), got, err)
}

func checkFromTimestamp(c vector) bool {
	x := c.number("x")
	d, err := FromTimestamp(x)
	got := ""
	if err == nil {
		got, err = Isoformat(d, "auto")
	}
	return c.expect(fmt.Sprintf("fromtimestamp(%v)", x), got, err)
}

func checkUTF8(c vector) bool {
	data, _ := hex.DecodeString(c.str("hex"))
	ok := true
	if replaced := DecodeUTF8Replace(data); replaced != c.str("replace") {
		c.t.Errorf("decode(%x, replace): got %q, CPython %q", data, replaced, c.str("replace"))
		ok = false
	}
	sig := vector{t: c.t, o: c.get("sig").(*pyjson.Object)}
	got, err := DecodeUTF8Sig(data)
	return sig.expect(fmt.Sprintf("decode(%x, utf-8-sig)", data), got, err) && ok
}

func checkText(c vector) bool {
	in := c.str("in")
	split := []any{}
	for _, field := range Split(in) {
		split = append(split, field)
	}
	got := map[string]any{"repr": Repr(in), "strip": Strip(in), "lstrip": LStrip(in),
		"rstrip": RStrip(in), "casefold": Casefold(in), "split": split}
	ok := true
	for key, value := range got {
		if want := c.get(key); !reflect.DeepEqual(value, want) {
			c.t.Errorf("%s(%q): got %q, CPython %q", key, in, value, want)
			ok = false
		}
	}
	return ok
}
