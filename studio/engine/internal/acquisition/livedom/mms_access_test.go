package livedom

import (
	"errors"
	"testing"
	"unicode"

	"studio-engine/internal/acquisition/goldentest"
	"studio-engine/internal/acquisition/pyv"
)

// number returns a recorded `now` as float64 (Python passes floats).
func number(t *testing.T, value any) float64 {
	t.Helper()
	if f, ok := value.(float64); ok {
		return f
	}
	if n, ok := pyv.Int64(value); ok {
		return float64(n)
	}
	t.Fatalf("not a number: %v", value)
	return 0
}

func TestMmsAccessMatchesPython(t *testing.T) {
	for _, c := range goldentest.Load(t, "livedom_mms.json") {
		clients := NewClients()
		outputs := []any{}
		for _, raw := range pyv.List(c.Args[0]) {
			step := pyv.List(raw)
			var err error
			switch pyv.Str(step[0]) {
			case "merge":
				err = MergeSummary(clients, pyv.List(step[1]), number(t, step[2]))
				outputs = append(outputs, clients.Object())
			case "own":
				MarkOwnPeer(clients, pyv.Str(step[1]), number(t, step[2]))
				outputs = append(outputs, clients.Object())
			default:
				outputs = append(outputs, ClientView(clients, number(t, step[1])))
			}
			var failure *PyError
			if errors.As(err, &failure) {
				outputs[len(outputs)-1] = pyv.O("error", failure.Text(), "clients", clients.Object())
			}
		}
		goldentest.Text(t, c, dumps(t, outputs))
	}
}

func TestIntMatchesPython(t *testing.T) {
	for _, c := range goldentest.Load(t, "livedom_int.json") {
		switch pyv.Str(c.Args[0]) {
		case "text":
			value, err := ParseIntText(pyv.Str(c.Args[1]))
			if err != nil {
				goldentest.Text(t, c, err.(*PyError).Text())
				continue
			}
			goldentest.Text(t, c, value.String())
		case "value":
			value, err := PyInt(c.Args[1])
			if err != nil {
				goldentest.Text(t, c, err.(*PyError).Text())
				continue
			}
			goldentest.Text(t, c, pyv.Repr(value))
		case "spaces":
			checkSpaces(t, pyv.List(c.Args[1]))
		case "digits":
			checkDigits(t, pyv.List(c.Args[1]))
		}
	}
}

func checkSpaces(t *testing.T, codes []any) {
	want := map[rune]bool{}
	for _, code := range codes {
		value, _ := pyv.Int64(code)
		want[rune(value)] = true
	}
	for r := rune(128); r <= unicode.MaxRune; r++ {
		if isPySpace(r) != want[r] {
			t.Errorf("isPySpace(U+%04X) = %v", r, isPySpace(r))
		}
	}
}

func checkDigits(t *testing.T, pairs []any) {
	want := map[rune]int{}
	for _, pair := range pairs {
		code, _ := pyv.Int64(pyv.List(pair)[0])
		value, _ := pyv.Int64(pyv.List(pair)[1])
		want[rune(code)] = int(value)
	}
	for r := rune(128); r <= unicode.MaxRune; r++ {
		if r >= 0xD800 && r <= 0xDFFF {
			continue
		}
		digit, ok := decimalValue(r)
		expected, found := want[r]
		if ok != found || digit != expected {
			t.Fatalf("decimalValue(U+%04X) = %d %v, want %d %v", r, digit, ok, expected, found)
		}
	}
}
