package profiles

import (
	"fmt"
	"strconv"
	"strings"

	"studio-engine/internal/snmp/model"
)

// IF-MIB columns of the switch's ports (ifTable and ifXTable).
const (
	ifNumber      = "1.3.6.1.2.1.2.1.0"
	ifEntry       = "1.3.6.1.2.1.2.2.1."
	ifXEntry      = "1.3.6.1.2.1.31.1.1.1."
	ifIndex       = ifEntry + "1."
	ifAdminStatus = ifEntry + "7."
	ifOperStatus  = ifEntry + "8."
	linkDown      = "1.3.6.1.6.3.1.1.5.3"
	linkUp        = "1.3.6.1.6.3.1.1.5.4"
)

// connectedPorts are up with traffic; the others are unplugged (down), like a real switch.
const connectedPorts = 6

func switchPoints(name string) []model.Point {
	points := system(name, "ScadaBench simulated managed Ethernet switch, 8 ports", "0.0")
	points = append(points, number(ifNumber, model.TypeInteger, SwitchPorts))
	for port := 1; port <= SwitchPorts; port++ {
		n := strconv.Itoa(port)
		status := int64(1)
		rate := 125000.0 * float64(port) // bytes per second: port 1 about 1 Mbit/s, port 6 about 6
		if port > connectedPorts {
			status, rate = 2, 0
		}
		points = append(points,
			number(ifEntry+"1."+n, model.TypeInteger, int64(port)),
			text(ifEntry+"2."+n, "Port "+n, false),
			number(ifEntry+"3."+n, model.TypeInteger, 6), // ethernetCsmacd
			number(ifEntry+"4."+n, model.TypeInteger, 1500),
			number(ifEntry+"5."+n, model.TypeGauge32, 1000000000),
			model.Point{OID: ifEntry + "6." + n, Type: model.TypeOctets, Value: fmt.Sprintf("02 5c 42 00 00 %02x", port), Hex: true},
			model.Point{OID: ifEntry + "7." + n, Type: model.TypeInteger, Value: "1", Writable: true},
			number(ifEntry+"8."+n, model.TypeInteger, status),
			number(ifEntry+"9."+n, model.TypeTimeTicks, 0),
			counter(ifEntry+"10."+n, model.TypeCounter32, rate),
			counter(ifEntry+"11."+n, model.TypeCounter32, rate/500),
			number(ifEntry+"14."+n, model.TypeCounter32, 0),
			counter(ifEntry+"16."+n, model.TypeCounter32, rate/2),
			counter(ifEntry+"17."+n, model.TypeCounter32, rate/1000),
			number(ifEntry+"20."+n, model.TypeCounter32, 0),
			text(ifXEntry+"1."+n, "port"+n, false),
			counter(ifXEntry+"6."+n, model.TypeCounter64, rate),
			counter(ifXEntry+"10."+n, model.TypeCounter64, rate/2),
			number(ifXEntry+"15."+n, model.TypeGauge32, 1000),
			text(ifXEntry+"18."+n, "", true),
		)
	}
	return points
}

func switchActions(device model.Device) []Action {
	actions := []Action{}
	for port := 1; port <= SwitchPorts; port++ {
		n := strconv.Itoa(port)
		if _, found := find(device, ifOperStatus+n); !found {
			continue
		}
		actions = append(actions, Action{ID: "link_down:" + n, Label: "Port " + n + " link down"},
			Action{ID: "link_up:" + n, Label: "Port " + n + " link up"})
	}
	return actions
}

// switchEvent sets a port's ifOperStatus and sends linkDown or linkUp with the ifIndex,
// ifAdminStatus and ifOperStatus of that port (IF-MIB).
func switchEvent(device model.Device, action string) (Event, bool) {
	kind, port, found := strings.Cut(action, ":")
	if !found || (kind != "link_down" && kind != "link_up") {
		return Event{}, false
	}
	operational, found := find(device, ifOperStatus+port)
	if !found {
		return Event{}, false
	}
	operational.Value = "1"
	notification := linkUp
	if kind == "link_down" {
		operational.Value, notification = "2", linkDown
	}
	admin, found := find(device, ifAdminStatus+port)
	if !found {
		admin = model.Point{OID: ifAdminStatus + port, Type: model.TypeInteger, Value: "1"}
	}
	index := model.Point{OID: ifIndex + port, Type: model.TypeInteger, Value: port}
	return Event{Changes: []model.Point{operational}, Notification: notification,
		VarBinds: []model.Point{index, admin, operational}}, true
}
