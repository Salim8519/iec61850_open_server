package history

import (
	"testing"
	"time"

	"studio-engine/internal/pyjson"
)

type plain struct{}

func (plain) Skip(key string) bool         { return key == "id" }
func (plain) Same(_ string, a, b any) bool { return Equal(a, b, 2, nil) }
func (plain) Keep(_ string, r, _ any) any  { return r }

func project(values ...any) *pyjson.Object {
	out := pyjson.NewObject().Set("id", "p")
	for index := 0; index+1 < len(values); index += 2 {
		out.Set(values[index].(string), values[index+1])
	}
	return out
}

// TestEqualByIdentityAndShape: a copied list or object with the same values is equal; a
// changed value, a missing field or a deeper change past the depth is not.
func TestEqualByIdentityAndShape(t *testing.T) {
	item := pyjson.NewObject().Set("a", int64(1))
	list := []any{item}
	copied := []any{pyjson.NewObject().Set("a", int64(1))}
	if !Equal(list, copied, 2, nil) || Equal(list, copied, 1, nil) {
		t.Fatal("copied list")
	}
	if Equal(list, []any{pyjson.NewObject().Set("a", int64(2))}, 2, nil) {
		t.Fatal("changed value")
	}
	labeled := []any{pyjson.NewObject().Set("a", int64(1)).Set("lab_ip", "1.2.3.4")}
	if Equal(list, labeled, 2, nil) || !Equal(list, labeled, 2, func(f string) bool { return f == "lab_ip" }) {
		t.Fatal("ignored field")
	}
	if !Identical(Missing, Missing) || Identical([]any{}, nil) || !Identical("x", "x") {
		t.Fatal("identity")
	}
	if Identical(map[string]int{}, map[string]int{}) {
		t.Fatal("uncomparable values must not panic or match")
	}
}

// TestRecordNextDone: a step keeps only the changed keys; a key added by the change is
// removed again on Undo (Missing); skipped keys are never steps.
func TestRecordNextDone(t *testing.T) {
	clock := time.Unix(0, 0)
	h := New(plain{}, func() time.Time { clock = clock.Add(5 * time.Second); return clock })
	first := project("a", "1")
	second := project("a", "1", "b", "2")
	h.Record(first, second, nil)
	h.Record(second, project("a", "1", "b", "2").Set("id", "q"), nil)
	if keys := h.Keys(true); len(keys) != 1 || keys[0] != "b" {
		t.Fatalf("keys %v", keys)
	}
	plan, ok := h.Next(true, second)
	if !ok || plan.Values[0] != Missing {
		t.Fatalf("plan %v %v", plan, ok)
	}
	h.Done(true)
	if h.Keys(true) != nil || h.Keys(false) == nil {
		t.Fatal("step did not move to redo")
	}
	if _, ok := h.Next(false, project("a", "1", "b", "9")); ok || h.Keys(false) != nil {
		t.Fatal("a stale redo step must be dropped")
	}
}
