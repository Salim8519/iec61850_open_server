package ports

import (
	"studio-engine/internal/gateway/service"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime"
)

// GatewayTyped gives the gateway the coordinator's typed IEC 61850 commands (ADR-0104):
// the routes of ADR-0107 send SCADA selects, executes and cancels through them.
func GatewayTyped(c *runtime.Coordinator) service.TypedCommands { return gatewayTyped{c} }

type gatewayTyped struct{ c *runtime.Coordinator }

func command(request service.TypedRequest) runtime.TypedCommand {
	return runtime.TypedCommand{IEDID: request.IEDID, Ref: request.Ref, Value: request.Value,
		Options: runtime.CommandOptions{Test: request.Test, OrCat: request.OrCat, OrIdent: request.OrIdent}}
}

func (g gatewayTyped) Select(request service.TypedRequest) (int, error) {
	result, err := g.c.SelectTyped(command(request))
	return result.ExpiresMs, err
}

func (g gatewayTyped) Operate(request service.TypedRequest) (*pyjson.Object, error) {
	return g.c.OperateTyped(command(request))
}

func (g gatewayTyped) Cancel(iedID, ref string) error { return g.c.CancelTyped(iedID, ref) }
