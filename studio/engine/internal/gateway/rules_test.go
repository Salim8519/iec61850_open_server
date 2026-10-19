package gateway

import (
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pyoracle"
)

// TestRulesAgainstPython replays gateway_rules: settings, point validation (valid
// proposals mutated with bad types, overlaps and unknown references), listen choices and
// normalize_gateway with random stored sections and master blocks.
func TestRulesAgainstPython(t *testing.T) {
	replay(t, "rules_oracle.py", "2000", func(item pyoracle.Case, fixtures map[string][]*pyjson.Object) {
		args := item.Args
		switch item.Fn {
		case "validate_settings":
			got, err := ValidateSettings(args[0].(string), args[1])
			expect(t, item, got, err)
		case "normalize_gateway":
			got, err := NormalizeGateway(args[0])
			expect(t, item, got, err)
		case "listen_address":
			got, err := ListenAddress(args[0])
			expect(t, item, got, err)
		case "allowed_kinds":
			expect(t, item, strs(AllowedKinds(args[0].(string))), nil)
		case "empty_gateway":
			expect(t, item, EmptyGateway(), nil)
		case "validate_points":
			clean, problems, err := ValidatePoints(args[1].(string), args[2], fixtures[args[0].(string)])
			expect(t, item, []any{clean, strs(problems)}, err)
		default:
			t.Errorf("unknown oracle function %s", item.Fn)
		}
	})
}

// TestNormalizeGatewayKeepsInput checks the port copies stored points instead of adding the
// id in place (the project snapshot is shared).
func TestNormalizeGatewayKeepsInput(t *testing.T) {
	point := pyjson.NewObject().Set("kind", "single").Set("address", int64(3))
	raw := pyjson.NewObject().Set("points", pyjson.NewObject().Set("modbus", []any{point}))
	result, err := NormalizeGateway(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := point.Get("id"); found {
		t.Fatal("input point was changed")
	}
	stored := list(get(object(get(result, "points")), "modbus"))
	if id := get(object(stored[0]), "id"); id != "::single" {
		t.Fatalf("id %v", id)
	}
}
