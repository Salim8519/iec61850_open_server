package licensing

import (
	"errors"
	"os"
	"regexp"
	"runtime"
	"testing"
)

type fakeGate struct {
	enforced, licensed bool
	err                error
	asked              int
}

func (g *fakeGate) Enforced() bool { return g.enforced }
func (g *fakeGate) Licensed() (bool, error) {
	g.asked++
	return g.licensed, g.err
}

func TestRequiredBodyIsFlaskBytes(t *testing.T) {
	want := `{"code":"license_required","error":"A license is required. Start the free trial or activate your key."}` + "\n"
	if got := string(RequiredBody()); got != want || RequiredStatus != 402 {
		t.Fatalf("402 body %q", got)
	}
}

func TestRequireFollowsSecurityRules(t *testing.T) {
	for _, path := range []string{"/api/v1/license/status", "/api/v1/license", "/api/v1/licenses"} {
		if !Exempt(path) {
			t.Errorf("%s must be exempt", path)
		}
	}
	if Exempt("/api/v1/project") || Exempt("/api/v1/lic") {
		t.Error("only the license prefix is exempt")
	}
	if blocked, err := Require(nil, "/api/v1/project"); blocked || err != nil {
		t.Error("no license service never gates")
	}
	off := &fakeGate{enforced: false}
	if blocked, _ := Require(off, "/api/v1/project"); blocked || off.asked != 0 {
		t.Error("development never gates or evaluates")
	}
	unlicensed := &fakeGate{enforced: true}
	if blocked, _ := Require(unlicensed, "/api/v1/license/trial"); blocked || unlicensed.asked != 0 {
		t.Error("license paths are never gated")
	}
	if blocked, _ := Require(unlicensed, "/api/v1/project"); !blocked {
		t.Error("an unlicensed packaged build answers 402")
	}
	if blocked, _ := Require(&fakeGate{enforced: true, licensed: true}, "/api/v1/project"); blocked {
		t.Error("a licensed build passes")
	}
	failing := &fakeGate{enforced: true, err: errors.New("machine id")}
	if _, err := Require(failing, "/api/v1/project"); err == nil {
		t.Error("evaluation errors reach the server (HTTP 500)")
	}
	var _ Gate = (*Service)(nil)
}

// TestMachineIDOnThisPC reads the real machine ID (read-only Windows calls). Set
// STUDIO_EXPECTED_HWID to the native guard's `studio_license_check hwid` to compare both on
// one PC; the value is never stored in the repository.
func TestMachineIDOnThisPC(t *testing.T) {
	if runtime.GOOS != "windows" {
		if _, err := MachineID(); err == nil {
			t.Fatal("machine licensing is Windows only")
		}
		return
	}
	current, err := MachineID()
	if err != nil {
		t.Skipf("machine ID unreadable here: %v", err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(current) {
		t.Fatalf("machine ID %q", current)
	}
	again, err := MachineID()
	if err != nil || again != current {
		t.Fatalf("the machine ID is not stable: %q then %q (%v)", current, again, err)
	}
	if want := os.Getenv("STUDIO_EXPECTED_HWID"); want != "" && want != current {
		t.Fatal("Go machine ID differs from the native guard's")
	}
}
