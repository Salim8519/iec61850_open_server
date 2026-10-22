package masterpoints

import (
	"math"
	"strconv"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// Mapped is what the station gateway's own mapping says one gateway point carries now. The
// gateway service reads it (the IED value behind that gateway point); the master never does.
// "Check against IED" is a test aid outside the master: it compares that with what the master
// received for a point copied from the gateway.
type Mapped struct {
	Found  bool    // the gateway list still has a point at that protocol, address and kind
	Online bool    // the IED (or the made-up value) is current
	Value  any     // the value as the IED holds it (on/off, position 0-3, number)
	Scale  float64 // the gateway point's scale: SCADA receives Value x Scale for numbers
	Source string  // "FEED1 XCBR1.Pos" or "made-up value": who holds the value
}

var positions = map[int64]string{0: "Intermediate", 1: "Open", 2: "Closed", 3: "Bad state"}

// CheckAgainstIED is the column of one compared row: {ok, text} with ok true (match), false
// (a plain reason what is wrong) or nil (nothing to compare yet, and why).
func CheckAgainstIED(protocol string, item *pyjson.Object, mapped Mapped) *pyjson.Object {
	point, _ := field(item, "point").(*pyjson.Object)
	_, address, kind, copied := FromOf(point)
	answer := func(ok any, words string) *pyjson.Object {
		return pyjson.NewObject().Set("ok", ok).Set("text", words).Set("source", mapped.Source)
	}
	where := AddressWord[protocol] + " " + strconv.FormatInt(address, 10)
	switch {
	case !copied:
		return nil
	case len(kind) > 8 && kind[:8] == "command_":
		return answer(nil, "A command point: use Send and watch the IED.")
	case !mapped.Found:
		return answer(nil, "The gateway list has no "+kind+" point at "+where+" any more.")
	case !mapped.Online:
		return answer(nil, "The IED has no value now (it is not running or not connected).")
	}
	switch text(item, "state") {
	case StateWrongType:
		return answer(false, "The master got another type at this address: check the gateway point type.")
	case StateReceived:
	default:
		return answer(nil, "The master has not received this point yet.")
	}
	got := field(item, "value")
	switch kind {
	case "single":
		want, have := onOff(mapped.Value), onOff(got)
		if want == have {
			return answer(true, "Both say "+want+".")
		}
		return answer(false, "IED says "+want+", master got "+have+": check the single point mapping.")
	case "double":
		want, have := position(mapped.Value), position(got)
		if want == have {
			return answer(true, "Both say "+want+".")
		}
		return answer(false, "IED says "+want+", master got "+have+": check the double point mapping.")
	}
	return numberCheck(kind, mapped, got, answer)
}

func numberCheck(kind string, mapped Mapped, got any, answer func(any, string) *pyjson.Object) *pyjson.Object {
	raw, okWant := pytext.Float(mapped.Value)
	have, okHave := pytext.Float(got)
	if !okWant || !okHave {
		return answer(nil, "One of the values is not a number.")
	}
	want := raw * mapped.Scale
	tolerance := math.Max(0.01, 1e-4*math.Abs(want))
	if kind == "integer" {
		tolerance += 0.5 // a scaled value is a whole number
	}
	wantText, haveText := pytext.FormatG(want, 6), pytext.FormatG(have, 6)
	if math.Abs(want-have) <= tolerance {
		return answer(true, "Both say "+haveText+".")
	}
	return answer(false, "IED says "+wantText+", master got "+haveText+": check the scale and the type of this point.")
}

func onOff(value any) string {
	if pytext.Truthy(value) {
		return "ON"
	}
	return "OFF"
}

func position(value any) string {
	number, ok := pytext.Int64(value)
	if word, known := positions[number]; ok && known {
		return word
	}
	if f, isFloat := value.(float64); isFloat {
		if word, known := positions[int64(f)]; known && f == math.Trunc(f) {
			return word
		}
	}
	return pytext.Str(value)
}
