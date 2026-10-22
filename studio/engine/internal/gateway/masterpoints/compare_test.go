package masterpoints

import (
	"strconv"
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

func received(group string, address int64, typeID string, value any) *pyjson.Object {
	return obj("key", group+":"+strconv.FormatInt(address, 10), "group", group, "address", address, "type", typeID, "value", value,
		"value_text", "v", "quality_text", "good", "good", true, "stale", false, "time_ms", int64(5), "label", "")
}

func states(t *testing.T, answer *pyjson.Object) []string {
	t.Helper()
	result := []string{}
	for _, raw := range field(answer, "rows").([]any) {
		result = append(result, text(raw.(*pyjson.Object), "state"))
	}
	return result
}

// Received, Missing, Wrong type and command points; what the device sent beyond the list.
func TestCompareClassifiesExpectedAgainstReceived(t *testing.T) {
	list, _ := Validate("iec104", []any{obj("type", "double", "address", int64(20001)), obj("type", "single", "address", int64(10001)),
		obj("type", "double", "address", int64(20002)), obj("type", "command_double", "address", int64(50001)),
		obj("type", "float", "address", int64(30002), "scale", 0.1, "unit", "kV")})
	rows := []any{received("double", 20001, "M_DP_TB_1", int64(2)), received("single", 20002, "M_SP_TB_1", int64(1)),
		received("float", 30001, "M_ME_NC_1", 1.5), received("float", 30002, "M_ME_NC_1", 1100.0)}
	answer := Compare("iec104", list, rows, Link{Connected: true, Asked: true})
	if got := strings.Join(states(t, answer), ","); got != "received,missing,wrong_type,command,received" {
		t.Fatalf("states: %s", got)
	}
	items := field(answer, "rows").([]any)
	if reason := text(items[2].(*pyjson.Object), "reason"); reason != "The device sends a Single point (M_SP_TB_1) at IOA 20002, not a Double point." {
		t.Fatalf("wrong type reason: %q", reason)
	}
	if reason := text(items[1].(*pyjson.Object), "reason"); reason != "Nothing received at IOA 10001 after the general interrogation." {
		t.Fatalf("missing reason: %q", reason)
	}
	if got := text(items[4].(*pyjson.Object), "engineering"); got != "110 kV" {
		t.Fatalf("scale and unit: %q", got)
	}
	if command := jsonText(t, field(items[3].(*pyjson.Object), "command")); command != `{"kind":"double","address":50001,"value":2,"select":true}` {
		t.Fatalf("one-click command: %s", command)
	}
	extra := field(answer, "not_in_list").([]any)
	if len(extra) != 1 || text(extra[0].(*pyjson.Object), "key") != "float:30001" || text(extra[0].(*pyjson.Object), "list_type") != "float" {
		t.Fatalf("not in your list: %s", jsonText(t, extra))
	}
	counts := jsonText(t, field(answer, "counts"))
	if counts != `{"received":2,"missing":1,"wrong_type":1,"waiting":0,"command":1,"not_in_list":1,"total":5}` {
		t.Fatalf("counts: %s", counts)
	}
	waiting := Compare("iec104", list, nil, Link{Connected: true})
	if got := states(t, waiting); got[1] != "waiting" || !strings.Contains(text(field(waiting, "rows").([]any)[1].(*pyjson.Object), "reason"), "Waiting for the general interrogation") {
		t.Fatalf("before the first interrogation nothing is missing: %v", got)
	}
	if got := states(t, Compare("iec104", list, nil, Link{})); got[0] != "waiting" {
		t.Fatalf("not connected: %v", got)
	}
}

func TestCompareModbusDataTypeAndDNP3Spaces(t *testing.T) {
	list, _ := Validate("modbus", []any{obj("type", "holding_float32", "address", int64(100)), obj("type", "coil", "address", int64(0))})
	answer := Compare("modbus", list, []any{received("holding_register", 100, "uint16", int64(17000)),
		received("coil", 0, "bool", int64(1))}, Link{Connected: true, Asked: true})
	items := field(answer, "rows").([]any)
	if got := states(t, answer); got[0] != "wrong_type" || got[1] != "received" ||
		!strings.Contains(text(items[0].(*pyjson.Object), "reason"), "Read as uint16 at address 100, but the list says float32") {
		t.Fatalf("modbus: %v %s", got, jsonText(t, items[0]))
	}
	dnp3, _ := Validate("dnp3", []any{obj("type", "binary", "address", int64(0)), obj("type", "crob", "address", int64(0))})
	answer = Compare("dnp3", dnp3, []any{received("binary", 0, "g1v2", true), received("double_binary", 0, "g3v2", int64(1)),
		received("binary_output", 0, "g10v2", false)}, Link{Connected: true, Asked: true})
	if got := strings.Join(states(t, answer), ","); got != "received,received" {
		t.Fatalf("a CROB shows its output status: %s", got)
	}
	if extra := field(answer, "not_in_list").([]any); len(extra) != 1 || text(extra[0].(*pyjson.Object), "group") != "double_binary" {
		t.Fatalf("the double-bit input at the same index is not hidden: %s", jsonText(t, extra))
	}
}

func TestPollRangesMergeNeighboursAndCapEachRequest(t *testing.T) {
	raw := []any{obj("type", "holding_int16", "address", int64(5)), obj("type", "holding_int16", "address", int64(0)),
		obj("type", "holding_int16", "address", int64(1)), obj("type", "coil", "address", int64(2)),
		obj("type", "coil", "address", int64(1))}
	for at := int64(0); at < 70; at++ {
		raw = append(raw, obj("type", "input_float32", "address", 1000+2*at))
	}
	list, err := Validate("modbus", raw)
	if err != nil {
		t.Fatal(err)
	}
	got := jsonText(t, PollRanges(list))
	want := `[{"table":"coils","start":1,"count":2,"data_type":"bool","word_order":"big"},` +
		`{"table":"holding_registers","start":0,"count":2,"data_type":"int16","word_order":"big"},` +
		`{"table":"holding_registers","start":5,"count":1,"data_type":"int16","word_order":"big"},` +
		`{"table":"input_registers","start":1000,"count":62,"data_type":"float32","word_order":"big"},` +
		`{"table":"input_registers","start":1124,"count":8,"data_type":"float32","word_order":"big"}]`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}
