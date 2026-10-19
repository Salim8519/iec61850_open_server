package npcap

import "testing"

// A failed filter used to read the error text after pcap_close, through a null
// handle, and crashed the engine (seen on the Npcap loopback adapter, which is not
// Ethernet). The text must come from an open handle or a plain message.
func TestErrorTextOfAClosedHandleDoesNotTouchNpcap(t *testing.T) {
	h := &Handle{}
	if got := h.lastError(); got != "the adapter is closed" {
		t.Fatalf("lastError on a closed handle = %q", got)
	}
	h.Close() // closing twice is harmless
	if h.Event() != 0 {
		t.Fatal("a closed handle still reports a receive event")
	}
}
