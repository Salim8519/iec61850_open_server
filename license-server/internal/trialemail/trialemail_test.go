package trialemail

import "testing"

func TestOneMailboxAlwaysNormalisesToTheSameAddress(t *testing.T) {
	for _, typed := range []string{"John.Smith+trial@GMail.com", " johnsmith@googlemail.com ", "j.o.h.n.smith@gmail.com"} {
		if got, err := Normalize(typed); err != nil || got != "johnsmith@gmail.com" {
			t.Fatalf("%q -> %q %v", typed, got, err)
		}
	}
	if got, _ := Normalize("First.Last+x@Company.com"); got != "first.last@company.com" {
		t.Fatalf("dots must stay outside Gmail: %q", got)
	}
	for _, bad := range []string{"", "no-at-sign", "a@b", "+tag@gmail.com", "a b@c.com", "a@b@c.com"} {
		if _, err := Normalize(bad); err == nil {
			t.Fatalf("%q accepted", bad)
		}
	}
	if !Disposable("Mailinator.com") || Disposable("company.com") {
		t.Fatal("disposable list wrong")
	}
	if MAC("s", "h", "e", "123456") == MAC("s", "h", "e", "123457") || len(NewCode()) != 6 {
		t.Fatal("codes not distinct")
	}
}
