package masterpoints

import "testing"

// Gateway tap positions, step commands and set points (ADR-0107) copy to the master type that
// reads or sends them; a set point follows its format (merge of 0173 and 0174).
func TestCopyFromGatewayTypedKinds(t *testing.T) {
	setpoint := func(format string, address int64) any {
		point := gatewayPoint("command_setpoint", address, "set point")
		if format != "" {
			point.Set("format", format)
		}
		return point
	}
	cases := map[string]struct {
		source []any
		want   []string
	}{
		"iec104": {[]any{gatewayPoint("step_position", 30001, "tap"), gatewayPoint("command_step", 60001, "raise"),
			setpoint("", 60002), setpoint("scaled", 60003)}, []string{"step", "regulating", "setpoint_float", "setpoint_scaled"}},
		"dnp3": {[]any{gatewayPoint("step_position", 5, "tap"), gatewayPoint("command_step", 6, "raise"),
			setpoint("int32", 7)}, []string{"analog", "crob_trip_close", "ao_int32"}},
		"modbus": {[]any{gatewayPoint("step_position", 3000, "tap"), setpoint("int16", 4000)},
			[]string{"input_int16", "holding_int16"}},
	}
	for protocol, c := range cases {
		results, notes := FromGateway(protocol, c.source, nil, nil)
		if len(results) != len(c.want) {
			t.Fatalf("%s: %s %v", protocol, jsonText(t, PlanJSON(results)), notes)
		}
		for i, want := range c.want {
			if got := text(results[i].Point, "type"); got != want {
				t.Fatalf("%s point %d: type %q, want %q", protocol, i, got, want)
			}
		}
	}
}
