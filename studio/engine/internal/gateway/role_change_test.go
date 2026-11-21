package gateway

import "testing"

func TestRoleChangeRefusal(t *testing.T) {
	cases := []struct {
		current, next   string
		gateway, master bool
		want            string
	}{
		{"gateway", "master", true, false, ""}, // gateway keeps serving while the master view opens
		{"master", "gateway", false, true, ""}, // master stays connected while the gateway view opens
		{"master", "gateway", true, true, ""},  // both running
		{"gateway", "converter", false, false, ""},
		{"converter", "master", false, false, ""},
		{"gateway", "converter", true, false, RefuseEnterConverter},
		{"master", "converter", false, true, RefuseEnterConverter},
		{"converter", "gateway", true, false, RefuseLeaveConverter},
		{"converter", "master", false, true, RefuseLeaveConverter},
		{"converter", "converter", true, true, ""},
	}
	for _, c := range cases {
		if got := RoleChangeRefusal(c.current, c.next, c.gateway, c.master); got != c.want {
			t.Errorf("%s->%s gateway=%v master=%v: %q, want %q", c.current, c.next, c.gateway, c.master, got, c.want)
		}
	}
}

func TestRoleLocks(t *testing.T) {
	if locks := RoleLocks("gateway", false, false); len(locks) != 0 {
		t.Fatal(locks)
	}
	if locks := RoleLocks("master", true, true); len(locks) != 1 || locks["converter"] != RefuseEnterConverter {
		t.Fatal(locks)
	}
	locks := RoleLocks("converter", false, true)
	if len(locks) != 2 || locks["gateway"] != RefuseLeaveConverter || locks["master"] != RefuseLeaveConverter {
		t.Fatal(locks)
	}
}
