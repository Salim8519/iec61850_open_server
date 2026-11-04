package app

import (
	"strings"

	"studio-engine/internal/api/platformapi"
	"studio-engine/internal/labnet"
	"studio-engine/internal/netcfg"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
	snmpservice "studio-engine/internal/snmp/service"
)

// snmpNetwork is the SNMP tool as Network setup sees it (ADR-0092 amendment): its devices get
// addresses on the same card as the IEDs and follow the project's network mode.
type snmpNetwork struct {
	service *snmpservice.Service
}

func (n snmpNetwork) SNMPDevices() ([]netcfg.SNMPDevice, error) {
	devices := []netcfg.SNMPDevice{}
	for _, device := range n.service.LabDevices() {
		devices = append(devices, netcfg.SNMPDevice{ID: device.ID, Name: device.Name, LabIP: device.LabIP, Port: device.Port})
	}
	return devices, nil
}

func (n snmpNetwork) SetSNMPLabIPs(addresses map[string]string) error {
	return n.service.SetLabIPs(addresses)
}

// snmpLabCard puts a device's lab address on the project's card before the device starts: the
// ScadaBench Lab card (change 0213) or, through its approval, a real card (ADR-0136). Stop, exit
// and the next launch take it off again.
func (e *Engine) snmpLabCard() snmpservice.LabCard {
	return snmpservice.LabCard{
		Ensure: func(address, owner string) error {
			mode, guid := labProjects{service: e.Projects}.Network()
			if mode != "lab" || guid == "" {
				return nil
			}
			if card := labnet.InstalledCard(); card != "" && strings.EqualFold(card, guid) {
				return e.Lab.EnsureOwn(address, owner)
			}
			return e.Lab.EnsureRealOwn(guid, address, owner)
		},
		Release: e.Lab.Release, // keeps the running IEDs' and devices' addresses (ADR-0127)
	}
}

// wireSNMPNetwork joins the SNMP devices to Network setup and to the project's network mode.
// Moves between this PC and the lab network reach them through project_changed (wirePush).
func (e *Engine) wireSNMPNetwork(network *netcfg.NetworkService, project platformapi.ProjectNetwork) {
	if e.SNMP == nil {
		return
	}
	network.SNMP = snmpNetwork{service: e.SNMP}
	e.SNMP.SetNetwork(func() bool {
		network, _ := project.Network().(*pyjson.Object)
		return network != nil && pyval.Lookup(network, "mode", nil) == "lab"
	})
}
