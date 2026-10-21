package platformapi

import (
	"studio-engine/internal/netcfg"
	"studio-engine/internal/project"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/project/service"
	"studio-engine/internal/pyjson"
)

// ProjectNetwork adapts the project service to the network routes (Projects) and to the
// lab-network workflow (netcfg.Projects), as Python's NetworkService used ProjectService.
type ProjectNetwork struct {
	Service *service.Service
}

// Network is project["network"] of the published project.
func (p ProjectNetwork) Network() any {
	return pyval.Lookup(p.Service.Project(), "network", nil)
}

// UpdateNetwork is ied_settings.update_network: a checked change, committed and announced.
func (p ProjectNetwork) UpdateNetwork(value any) (*pyjson.Object, error) {
	return p.Service.Update(func(draft *pyjson.Object) (*pyjson.Object, error) {
		return project.UpdateNetwork(draft, value)
	})
}

// SimulatedIEDs lists the IEDs whose role is "simulated" (the default), in project order.
func (p ProjectNetwork) SimulatedIEDs() ([]netcfg.PlanIED, error) {
	ieds := []netcfg.PlanIED{}
	list, _ := pyval.Lookup(p.Service.Project(), "ieds", nil).([]any)
	for _, item := range list {
		ied, ok := item.(*pyjson.Object)
		if !ok || pyval.Lookup(ied, "role", "simulated") != "simulated" {
			continue
		}
		plan := netcfg.PlanIED{ID: pyval.Str(pyval.Lookup(ied, "id", "")), Name: pyval.Str(pyval.Lookup(ied, "name", "")),
			Port: 102}
		if port, ok := pyval.Lookup(ied, "port", nil).(int64); ok {
			plan.Port = int(port)
		}
		if address, ok := pyval.Lookup(ied, "scl_address", nil).(*pyjson.Object); ok {
			if ip := pyval.Lookup(address, "ip", nil); truthy(ip) {
				plan.SCLIP = pyval.Str(ip)
			}
			if subnet := pyval.Lookup(address, "subnet", nil); truthy(subnet) {
				plan.SCLSubnet = pyval.Str(subnet)
			}
		}
		ieds = append(ieds, plan)
	}
	return ieds, nil
}

// EnterLabMode is NetworkService._enter_lab_mode: each row's address becomes the IED's
// lab_ip and the network switches to lab mode on the adapter, in one committed change.
func (p ProjectNetwork) EnterLabMode(rows []netcfg.Row, adapter netcfg.Adapter) error {
	_, err := p.Service.Update(func(draft *pyjson.Object) (*pyjson.Object, error) {
		for _, row := range rows {
			ied, err := project.FindIED(draft, row.IEDID)
			if err != nil {
				return nil, err
			}
			ied.Set("lab_ip", row.Proposed)
		}
		return project.UpdateNetwork(draft, pyjson.NewObject().Set("mode", "lab").
			Set("adapter_guid", adapter.GUID).Set("adapter_label", adapter.Name))
	})
	return err
}

// LeaveLabMode switches back to "This PC only" when the network is in lab mode.
func (p ProjectNetwork) LeaveLabMode() error {
	network, _ := p.Network().(*pyjson.Object)
	if network == nil || pyval.Lookup(network, "mode", nil) != "lab" {
		return nil
	}
	_, err := p.UpdateNetwork(pyjson.NewObject().Set("mode", "loopback"))
	return err
}
