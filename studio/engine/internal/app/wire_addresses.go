package app

import (
	"studio-engine/internal/api"
	"studio-engine/internal/api/addressapi"
	"studio-engine/internal/labnet"
	"studio-engine/internal/netaddr"
	"studio-engine/internal/netcfg"
)

// addressRoutes let the Devices page and the SNMP device editor set one address on the project's
// network (change 0215): through the lab page's checks on the ScadaBench Lab card, through the
// address plan (one Windows prompt) on a real card.
func (e *Engine) addressRoutes(network *netcfg.NetworkService) []api.Route {
	projects := labProjects{service: e.Projects}
	service := netaddr.New(netaddr.Deps{
		Network: projects.Network, LabCard: labnet.InstalledCard, Running: e.Runtime.Running,
		IEDs: func() ([]netaddr.Device, error) {
			ieds, err := projects.SimulatedIEDs()
			devices := make([]netaddr.Device, len(ieds))
			for index, ied := range ieds {
				devices[index] = netaddr.Device{ID: ied.ID, Name: ied.Name, LabIP: ied.LabIP}
			}
			return devices, err
		},
		SNMP: func() []netaddr.Device {
			devices := []netaddr.Device{}
			if e.SNMP != nil {
				for _, device := range e.SNMP.LabDevices() {
					devices = append(devices, netaddr.Device{ID: device.ID, Name: device.Name, LabIP: device.LabIP})
				}
			}
			return devices
		},
		LabSetIED: func(id, ip string) error {
			_, err := e.Lab.SetAddress(id, ip)
			return err
		},
		LabCheckOwn: e.Lab.CheckOwn,
		SetSNMPLab: func(id, ip string) error {
			return e.SNMP.SetLabIPs(map[string]string{id: ip})
		},
		Preview: network.Preview, Apply: network.Apply,
		Adapters: func() ([]netcfg.Adapter, error) { return network.Adapters(true) },
	})
	return addressapi.Routes(service)
}
