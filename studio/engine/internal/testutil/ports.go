package testutil

import (
	"math/rand"
	"net"
	"strconv"
	"testing"
)

// FreePort returns a TCP port >= 20000 that is free on host right now. Tests that start real
// workers use it instead of a fixed port, which another test or a running instance may hold.
func FreePort(t testing.TB, host string) int {
	t.Helper()
	for attempt := 0; attempt < 200; attempt++ {
		port := 20000 + rand.Intn(40000)
		listener, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err == nil {
			_ = listener.Close()
			return port
		}
	}
	t.Fatalf("no free port >= 20000 on %s", host)
	return 0
}
