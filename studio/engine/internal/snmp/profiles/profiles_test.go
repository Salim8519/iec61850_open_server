package profiles

import (
	"strings"
	"testing"

	"studio-engine/internal/snmp/mib"
	"studio-engine/internal/snmp/model"
	"studio-engine/internal/snmp/proto"
)

// TestProfilePointsAreStandardObjects: every point of a profile is a known object of the
// built-in MIBs with a matching type, so managers see the names they expect.
func TestProfilePointsAreStandardObjects(t *testing.T) {
	library, err := mib.Standard()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []string{Switch, GPS, Empty} {
		for _, point := range Points(profile, "Device 1") {
			node, instance := library.Lookup(point.OID)
			if node == nil || node.Kind != "column" && node.Kind != "object" || instance == "" {
				t.Fatalf("%s: %s is not a known object instance (%s)", profile, point.OID, library.Name(point.OID))
			}
			if node.Syntax != point.Type && !(node.Syntax == model.TypeGauge32 && point.Type == model.TypeGauge32) {
				t.Errorf("%s: %s is %s in the MIB but %s here", profile, library.Name(point.OID), node.Syntax, point.Type)
			}
			if _, err := proto.ParseValue(point.Type, point.Value, point.Hex); err != nil {
				t.Errorf("%s: %s: %v", profile, point.OID, err)
			}
		}
	}
}

func TestEventsUseTheirStandardNames(t *testing.T) {
	library, _ := mib.Standard()
	device := model.Device{Name: "S", Profile: Switch, Points: Points(Switch, "S")}
	event, err := Apply(device, "link_down:3")
	if err != nil {
		t.Fatal(err)
	}
	if name := library.Name(event.Notification); !strings.HasSuffix(name, "::linkDown") {
		t.Fatalf("link down sends %s", name)
	}
	if name := library.Name(event.VarBinds[2].OID); name != "IF-MIB::ifOperStatus.3" || event.VarBinds[2].Value != "2" {
		t.Fatalf("link down carries %s = %s", name, event.VarBinds[2].Value)
	}
	gps := model.Device{Name: "G", Profile: GPS, Points: Points(GPS, "G")}
	event, err = Apply(gps, "sync_lost")
	if err != nil {
		t.Fatal(err)
	}
	if name := library.Name(event.Notification); name != "NTPv4-MIB::ntpEntNotifModeChange" {
		t.Fatalf("sync lost sends %s", name)
	}
	if name := library.Name(event.Changes[0].OID); name != "NTPv4-MIB::ntpEntStatusCurrentMode.0" ||
		library.EnumText(event.Changes[0].OID, 2) != "notSynchronized" {
		t.Fatalf("sync lost changes %s", name)
	}
	if len(Actions(device)) != 2*SwitchPorts+2 || len(Actions(gps)) != 4 {
		t.Fatalf("actions: switch %d, gps %d", len(Actions(device)), len(Actions(gps)))
	}
	if _, err := Apply(gps, "link_down:1"); err == nil {
		t.Fatal("a GPS clock accepted a link event")
	}
}
