package app

import (
	"studio-engine/internal/api"
	"studio-engine/internal/api/snmpapi"
	"studio-engine/internal/api/toolsapi"
	"studio-engine/internal/gateway"
	"studio-engine/internal/gateway/service"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/ports"
	"studio-engine/internal/scl"
	snmpservice "studio-engine/internal/snmp/service"
	"studio-engine/internal/tools/control"
	"studio-engine/internal/tools/explorer"
	"studio-engine/internal/tools/memory"
	"studio-engine/internal/tools/scan"
)

// wireTools builds create_app's tools: one studio_control client shared by the Explorer
// and the gateway, tool memory, the Explorer (bus-only events), Find IEDs, the station
// gateway on the value hub, the SCADA master (coordinator.emit) and the protocol converter.
func (e *Engine) wireTools() []api.Route {
	o := e.options
	bus := e.bus
	store := memory.New(o.DataDir)
	e.Control = control.NewClient(o.native("studio_control.exe"), nil)
	e.Explorer = explorer.NewService(o.native("studio_live.exe"), nil, func(kind string, payload *pyjson.Object) {
		publish(bus, o.Logf, kind, payload)
	}, store, e.Control)
	e.Scanner = scan.NewService(o.native("studio_live.exe"), store, nil, nil)
	// The gateway reads ied["state"] (point values, command check): IEDs with the runtime overlay.
	e.Gateway = service.NewManager(gatewayProjects{e.Projects, e.Runtime}, ports.Gateway(e.Runtime), e.Control,
		o.native("studio_gateway.exe"), nil)
	e.Master = service.NewMaster(e.Projects, e.Runtime.Emit, o.native("studio_master.exe"), nil)
	e.Gateway.Converter = &service.Converter{Master: e.Master}
	e.Gateway.Typed = ports.GatewayTyped(e.Runtime) // SCADA select / execute / cancel (ADR-0107)
	gateway.ExtendedControls = typedControlsOf      // BSC/APC/INC links and Add points (ADR-0104)
	// Python's gateway diffed every point each 100 ms and so saw converter values; the Go
	// gateway is change-driven, so new device values mark the converter's points.
	e.Master.OnValues = e.Gateway.MarkRemote
	// The Go-only SNMP devices tool (ADR-0092): studio_snmp runs only while SNMP is in use.
	e.SNMP = snmpservice.New(e.Projects, o.native("studio_snmp.exe"), o.DataDir, nil, o.Logf)
	routes := toolsapi.New(toolsapi.Deps{Explorer: e.Explorer, Scanner: e.Scanner, Memory: store,
		Gateway: e.Gateway, Master: e.Master, Projects: e.Projects, Runtime: e.Runtime}).Routes()
	return append(routes, snmpapi.New(e.SNMP).Routes()...)
}

// typedControlsOf is an IED's typed control list from its stored SCL (ADR-0104); none when the
// SCL cannot be read, so a link to a typed control is refused in words instead.
func typedControlsOf(ied *pyjson.Object) []any {
	text := func(key string) string { value, _ := ied.Get(key); result, _ := value.(string); return result }
	list, err := scl.CachedExtendedControls(text("id"), text("name"), text("source"))
	if err != nil {
		return nil
	}
	return list
}
