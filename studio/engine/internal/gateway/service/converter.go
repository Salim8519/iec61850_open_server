package service

import (
	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// Converter is ConverterSource: in the protocol converter role the gateway serves what the
// master reads from the device, and relays upstream commands to it.
type Converter struct {
	Master *Master
}

func (c *Converter) routes() []any {
	routes, _ := get(c.Master.Config(), "routes").([]any)
	return routes
}

// Relay sends an upstream command on to the device: (ok, plain reason).
func (c *Converter) Relay(point *pyjson.Object, value, allowed bool) (bool, string) {
	route, _ := get(point, "route").(*pyjson.Object)
	if route == nil {
		return false, "this point is not a command"
	}
	if !allowed {
		return false, "commands to real devices are switched off in the gateway settings"
	}
	selected, found := route.Get("select")
	if !found {
		selected = false
	}
	command := pyjson.NewObject().Set("kind", get(route, "kind")).Set("address", get(route, "address")).
		Set("value", gateway.DeviceValue(route, value)).Set("select", selected)
	if get(route, "kind") == "binary_output" {
		command.Set("code", "latch")
	}
	answer, err := c.Master.Command(command, "SCADA via the converter")
	if err != nil {
		panic(err.Error()) // the executor turns it into (False, str(error)) like Python
	}
	said, found := answer.Get("text")
	if !found {
		said = ""
	}
	return pytext.Truthy(get(answer, "ok")), pytext.Str(said)
}
