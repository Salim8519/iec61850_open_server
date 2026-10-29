//go:build windows

package netcfg

import (
	"reflect"
	"testing"

	"studio-engine/internal/pyjson"
)

// Skipping the DNS information (ADR-0128) must not change any adapter record. The link speed
// is left out: a Wi-Fi card changes it by itself between two listings.
func TestListingWithoutDNSInfoKeepsEveryRecord(t *testing.T) {
	lean, err := RawAdapters()
	if err != nil {
		t.Fatal(err)
	}
	listingFlags = gaaFlags &^ 0x800
	full, err := RawAdapters()
	listingFlags = gaaFlags
	if err != nil {
		t.Fatal(err)
	}
	if len(lean) == 0 || !reflect.DeepEqual(withoutSpeed(t, full), withoutSpeed(t, lean)) {
		t.Fatalf("listings differ:\nwith DNS info    %v\nwithout DNS info %v", withoutSpeed(t, full), withoutSpeed(t, lean))
	}
}

func withoutSpeed(t *testing.T, records []any) []string {
	texts := []string{}
	for _, record := range records {
		object := record.(*pyjson.Object)
		object.Delete("speed")
		text, err := pyjson.Flask(object)
		if err != nil {
			t.Fatal(err)
		}
		texts = append(texts, string(text))
	}
	return texts
}
