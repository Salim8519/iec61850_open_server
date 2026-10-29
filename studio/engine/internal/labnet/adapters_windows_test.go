//go:build windows

package labnet

import (
	"reflect"
	"testing"

	"golang.org/x/sys/windows"
)

// Skipping the DNS information (ADR-0128) must not change anything the lab network reads.
func TestListingWithoutDNSInfoKeepsEverythingRead(t *testing.T) {
	full, err := listAdapters(windows.GAA_FLAG_SKIP_ANYCAST | windows.GAA_FLAG_SKIP_MULTICAST |
		windows.GAA_FLAG_SKIP_DNS_SERVER)
	if err != nil {
		t.Fatal(err)
	}
	lean, err := ListAdapters()
	if err != nil {
		t.Fatal(err)
	}
	if len(lean) == 0 || !reflect.DeepEqual(full, lean) {
		t.Fatalf("listings differ:\nwith DNS info    %+v\nwithout DNS info %+v", full, lean)
	}
}
