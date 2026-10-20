package profiles

import "studio-engine/internal/snmp/model"

// NTPv4-MIB (RFC 5907) objects of the GPS time server.
const (
	ntpEntInfo         = "1.3.6.1.2.1.197.1.1."
	ntpEntStatus       = "1.3.6.1.2.1.197.1.2."
	ntpCurrentMode     = ntpEntStatus + "1.0"
	ntpStratum         = ntpEntStatus + "2.0"
	ntpRefSourceName   = ntpEntStatus + "4.0"
	ntpNotifModeChange = "1.3.6.1.2.1.197.0.1"
)

// ntpEntStatusCurrentMode values.
const (
	modeNotSynchronized = "2"
	modeSyncToRefclock  = "5"
)

func gpsPoints(name string) []model.Point {
	points := system(name, "ScadaBench simulated GPS time server (NTP and PTP grandmaster)", "0.0")
	return append(points,
		text(ntpEntInfo+"1.0", "ScadaBench GPS clock", false),
		text(ntpEntInfo+"2.0", "1.0", false),
		text(ntpEntInfo+"3.0", "ScadaBench", false),
		text(ntpEntInfo+"4.0", "GPS time server", false),
		number(ntpEntInfo+"5.0", model.TypeGauge32, 1000000000),
		number(ntpEntInfo+"6.0", model.TypeInteger, -20),
		text(ntpEntInfo+"7.0", "0.000 ms", false),
		number(ntpCurrentMode, model.TypeInteger, 5),
		number(ntpStratum, model.TypeGauge32, 1),
		number(ntpEntStatus+"3.0", model.TypeGauge32, 1),
		text(ntpRefSourceName, "GPS", false),
		text(ntpEntStatus+"5.0", "0.000012 ms", false),
		number(ntpEntStatus+"6.0", model.TypeGauge32, 1),
		text(ntpEntStatus+"7.0", "0.000100 ms", false),
		model.Point{OID: ntpEntStatus + "8.0", Type: model.TypeTimeTicks, Value: "0", Behaviour: model.BehaviourUptime},
		model.Point{OID: ntpEntStatus + "10.0", Type: model.TypeOctets, Value: "", Hex: true}, // no leap second announced
		number(ntpEntStatus+"11.0", model.TypeInteger, 0),
		counter(ntpEntStatus+"12.0", model.TypeCounter32, 40),
		counter(ntpEntStatus+"13.0", model.TypeCounter32, 40),
		number(ntpEntStatus+"14.0", model.TypeCounter32, 0),
		number(ntpEntStatus+"15.0", model.TypeCounter32, 0),
		number(ntpEntStatus+"16.0", model.TypeCounter32, 0),
	)
}

func gpsActions() []Action {
	return []Action{{ID: "sync_lost", Label: "GPS signal lost (not synchronized)"},
		{ID: "sync_back", Label: "GPS signal back (synchronized)"}}
}

// gpsEvent changes the mode, stratum and reference name and sends ntpEntNotifModeChange
// with the new mode.
func gpsEvent(action string) (Event, bool) {
	mode := model.Point{OID: ntpCurrentMode, Type: model.TypeInteger}
	stratum := model.Point{OID: ntpStratum, Type: model.TypeGauge32}
	source := text(ntpRefSourceName, "", false)
	switch action {
	case "sync_lost":
		mode.Value, stratum.Value, source.Value = modeNotSynchronized, "16", "none"
	case "sync_back":
		mode.Value, stratum.Value, source.Value = modeSyncToRefclock, "1", "GPS"
	default:
		return Event{}, false
	}
	return Event{Changes: []model.Point{mode, stratum, source}, Notification: ntpNotifModeChange,
		VarBinds: []model.Point{mode}}, true
}
