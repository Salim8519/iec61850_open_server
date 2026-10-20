package app

import (
	"studio-engine/internal/api/platformapi"
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
