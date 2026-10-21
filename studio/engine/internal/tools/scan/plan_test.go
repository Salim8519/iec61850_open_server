package scan

import (
	"errors"
	"path/filepath"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pyoracle"
)

func errorText(err error) string {
	var studio *apperr.Error
	if errors.As(err, &studio) && studio.Kind == apperr.Value {
		return "ValueError: " + studio.Message
	}
	return "Go: " + err.Error()
}

// TestPlanAgainstPython compares plan_range and preview with Python on random range texts.
func TestPlanAgainstPython(t *testing.T) {
	cases := pyoracle.Cases(t, filepath.Join("testdata", "plan_oracle.py"), nil, "61850", "3000")
	if len(cases) < 1000 {
		t.Fatalf("only %d cases", len(cases))
	}
	service := NewService("", nil, nil, nil)
	failures := 0
	for _, item := range cases {
		if item.Error != "" {
			failures++
		}
		var got any
		var err error
		switch item.Fn {
		case "plan":
			own, _ := item.Args[1].(string)
			var plan *Plan
			plan, err = PlanRange(item.Args[0], own)
			if err == nil {
				hosts := make([]any, len(plan.Hosts))
				for index, host := range plan.Hosts {
					hosts[index] = host
				}
				got = pyjson.NewObject().Set("range", plan.Range).Set("hosts", hosts).Set("count", int64(plan.Count)).
					Set("first", plan.First).Set("last", plan.Last)
			}
		case "preview":
			got, err = service.Preview(item.Args[0], item.Args[1])
		}
		if item.Error != "" {
			if err == nil || errorText(err) != item.Error {
				t.Errorf("%s%s: got %v want %s", item.Fn, pyoracle.Dumps(t, item.Args), err, item.Error)
			}
			continue
		}
		if err != nil || pyoracle.Dumps(t, got) != pyoracle.Dumps(t, item.Result) {
			t.Errorf("%s%s: got %v %v want %s", item.Fn, pyoracle.Dumps(t, item.Args), got, err, pyoracle.Dumps(t, item.Result))
		}
	}
	if failures < len(cases)/10 || failures > len(cases)*9/10 {
		t.Fatalf("unbalanced oracle: %d of %d cases are errors", failures, len(cases))
	}
}
