package pyval

import "testing"

func TestPythonEqualityAndConversions(t *testing.T) {
	if !Equal(true, int64(1)) || !Equal(2.0, int64(2)) || Equal("1", int64(1)) || Equal(nil, false) {
		t.Fatal("Python == semantics")
	}
	if !Equal(Obj("a", int64(1), "b", []any{1.0}), Obj("b", []any{int64(1)}, "a", true)) {
		t.Fatal("dict equality ignores order and compares numbers by value")
	}
	if _, err := ToInt("1.5"); err == nil || err.Error() != "invalid literal for int() with base 10: '1.5'" {
		t.Fatalf("int('1.5'): %v", err)
	}
	if value, err := ToInt(" +1_000 "); err != nil || value != int64(1000) {
		t.Fatalf("int(' +1_000 '): %v %v", value, err)
	}
	if value, _ := ToInt(-2.7); value != int64(-2) {
		t.Fatalf("int(-2.7) = %v", value)
	}
	if _, err := ToFloat("abc"); err == nil || err.Error() != "could not convert string to float: 'abc'" {
		t.Fatalf("float('abc'): %v", err)
	}
	if Repr(1.0) != "1.0" || Repr(nil) != "None" || Repr(true) != "True" || Repr([]any{"it's", 2.5}) != `["it's", 2.5]` {
		t.Fatalf("str(): %s", Repr([]any{"it's", 2.5}))
	}
}
