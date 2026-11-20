package app

import (
	"studio-engine/internal/api"
	"studio-engine/internal/api/labapi"
	"studio-engine/internal/api/platformapi"
	gatewayservice "studio-engine/internal/gateway/service"
	"studio-engine/internal/labnet"
	"studio-engine/internal/labsetup"
	"studio-engine/internal/netcfg"
	"studio-engine/internal/project"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/project/service"
	"studio-engine/internal/pyjson"
	snmpservice "studio-engine/internal/snmp/service"
)

// labProgram is the lab network program beside the native workers (ADR-0120).
const labProgram = "scadabench_labnet.exe"

// labRoutes wires the "This PC lab network" page (ADR-0120): its service over the project, the
// coordinator, the network service's adapter list and the lab network service.
func (e *Engine) labRoutes(network *netcfg.NetworkService, projects platformapi.ProjectNetwork) []api.Route {
	o := e.options
	journal := netcfg.NewJournal(o.DataDir)
	journal.Path = journal.Path[:len(journal.Path)-len("journal.jsonl")] + "lab_journal.jsonl"
	e.Lab = &labsetup.Service{Projects: labProjects{e.Projects, e.Gateway, e.Master, e.SNMP}, Runtime: e.Runtime,
		Windows: labsetup.Windows{
			Adapters: func() ([]netcfg.Adapter, error) { return network.Adapters(true) },
			State:    labnet.ServiceState, Card: labnet.InstalledCard, Call: labnet.Call, Ping: labnet.Ping,
			Ours: func() map[string][]string {
				ours, _ := network.OurAddresses()
				return ours
			},
			FreeOurs: network.RemoveOurAddresses, Approved: labnet.ReadApproved},
		Asker: labsetup.NewAsker(labnet.AskOn), Elevator: netcfg.UACElevator{}, Program: o.native(labProgram),
		Directory: o.DataDir, Journal: journal, NewID: network.NewID}
	if e.Gateway != nil {
		e.Gateway.Lab = e.Lab // the gateway's and the master's own addresses on the card (ADR-0125)
	}
	if e.Master != nil {
		e.Master.Lab = e.Lab
	}
	if e.SNMP != nil {
		e.SNMP.SetLabCard(e.snmpLabCard())
	}
	return labapi.New(labapi.Deps{Lab: e.Lab, Leave: projects.LeaveLabMode}).Routes()
}

// labProjects adapts the project service to labsetup.Projects; the gateway, the master and the
// SNMP devices name the card addresses they are set to use (ADR-0125) and the ones they hold on
// the card now (ADR-0141).
type labProjects struct {
	service *service.Service
	gateway *gatewayservice.Manager
	master  *gatewayservice.Master
	snmp    *snmpservice.Service
}

func (p labProjects) Reserved() map[string]string {
	used := map[string]string{}
	if p.gateway != nil {
		for address, name := range p.gateway.OwnAddresses() {
			used[address] = name
		}
	}
	if p.master != nil {
		for address, name := range p.master.OwnAddresses() {
			used[address] = name
		}
	}
	if p.snmp != nil {
		for address, name := range p.snmp.CardAddresses() {
			used[address] = name
		}
	}
	return used
}

func (p labProjects) InUse() map[string]string {
	used := map[string]string{}
	if p.gateway != nil {
		for address, name := range p.gateway.CardAddresses() {
			used[address] = name
		}
	}
	if p.master != nil {
		for address, name := range p.master.CardAddresses() {
			used[address] = name
		}
	}
	if p.snmp != nil {
		for address, name := range p.snmp.CardAddresses() {
			used[address] = name
		}
	}
	return used
}

func (p labProjects) SimulatedIEDs() ([]labsetup.IED, error) {
	ieds := []labsetup.IED{}
	list, _ := pyval.Lookup(p.service.Project(), "ieds", nil).([]any)
	for _, item := range list {
		ied, ok := item.(*pyjson.Object)
		if !ok || pyval.Lookup(ied, "role", "simulated") != "simulated" {
			continue
		}
		entry := labsetup.IED{ID: pyval.Str(pyval.Lookup(ied, "id", "")), Name: pyval.Str(pyval.Lookup(ied, "name", "")),
			LabIP: pyval.Str(pyval.Lookup(ied, "lab_ip", ""))}
		if address, ok := pyval.Lookup(ied, "scl_address", nil).(*pyjson.Object); ok {
			entry.SCLIP = pyval.Str(pyval.Lookup(address, "ip", ""))
		}
		ieds = append(ieds, entry)
	}
	return ieds, nil
}

func (p labProjects) Network() (string, string) {
	network, _ := pyval.Lookup(p.service.Project(), "network", nil).(*pyjson.Object)
	if network == nil {
		return "", ""
	}
	return pyval.Str(pyval.Lookup(network, "mode", "")), pyval.Str(pyval.Lookup(network, "adapter_guid", ""))
}

func (p labProjects) UseCard(addresses map[string]string, guid, label string) error {
	_, err := p.service.Update(func(draft *pyjson.Object) (*pyjson.Object, error) {
		for id, address := range addresses {
			ied, err := project.FindIED(draft, id)
			if err != nil {
				return nil, err
			}
			ied.Set("lab_ip", address)
		}
		return project.UpdateNetwork(draft, pyjson.NewObject().Set("mode", "lab").
			Set("adapter_guid", guid).Set("adapter_label", label))
	})
	return err
}

func (p labProjects) SetLabIP(id, address string) error {
	_, err := p.service.Update(func(draft *pyjson.Object) (*pyjson.Object, error) {
		ied, err := project.FindIED(draft, id)
		if err != nil {
			return nil, err
		}
		ied.Set("lab_ip", address)
		return ied, nil
	})
	return err
}
