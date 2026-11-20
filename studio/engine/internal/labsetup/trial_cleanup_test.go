//go:build trial

package labsetup

import (
	"os"
	"strings"
	"testing"

	"studio-engine/internal/labnet"
)

// TestTrialRemoveLeftovers takes group C's trial addresses (198.18.24.x only, listed in
// TRIAL_REMOVE, comma-separated) off the Lab card through the installed service. A trial engine
// that was killed and whose data folder was deleted leaves its session's addresses on the card,
// and since ADR-0141 no other copy removes them (studio/tools/trials/labgroup.py clean_card).
func TestTrialRemoveLeftovers(t *testing.T) {
	var addresses []string
	for _, address := range strings.Split(os.Getenv("TRIAL_REMOVE"), ",") {
		if address = strings.TrimSpace(address); strings.HasPrefix(address, "198.18.24.") {
			addresses = append(addresses, address)
		}
	}
	if len(addresses) == 0 {
		t.Skip("TRIAL_REMOVE lists no 198.18.24.x address")
	}
	reply, err := labnet.Call(labnet.Request{Op: labnet.OpRemove, IPs: addresses})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("removed %v: ok %v, failed %v, card now %v", addresses, reply.OK, reply.Failed, reply.Addresses)
}
