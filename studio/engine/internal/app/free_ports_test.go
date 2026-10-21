package app

import (
	"math/rand"
	"net"
	"strconv"
	"testing"
)

// freePortBase returns a port base >= 20000 such that base+1 .. base+count are free TCP
// ports on every host (the IED addresses of a test project). A fixed base collided with
// workers of an earlier test still in TIME_WAIT or another test binary (gap 10).
func freePortBase(t testing.TB, count int, hosts ...string) int {
	t.Helper()
	for attempt := 0; attempt < 200; attempt++ {
		base := 20000 + rand.Intn(40000)
		if portsFree(base, count, hosts) {
			return base
		}
	}
	t.Fatalf("no %d free ports >= 20000 on %v", count, hosts)
	return 0
}

func portsFree(base, count int, hosts []string) bool {
	var held []net.Listener
	defer func() {
		for _, listener := range held {
			_ = listener.Close()
		}
	}()
	for offset := 1; offset <= count; offset++ {
		for _, host := range hosts {
			listener, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(base+offset)))
			if err != nil {
				return false
			}
			held = append(held, listener)
		}
	}
	return true
}

// scenarioHosts are the IED addresses of the runtime parity project (testdata/project.json).
var scenarioHosts = []string{"127.63.1.11", "127.63.1.12"}
