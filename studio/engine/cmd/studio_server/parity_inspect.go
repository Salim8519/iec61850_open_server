//go:build parity

package main

import (
	"errors"
	"fmt"
	"net/http"

	"studio-engine/internal/app"
	"studio-engine/internal/apperr"
	"studio-engine/internal/netcfg"
	"studio-engine/internal/pyjson"
)

// The inspect operations that reach into services (tests/parity/inspect_ops.py): network
// seams and journal, tool teardown stops, runtime state writes, runtime faults.

func init() {
	for key, op := range map[string]parityOp{
		"POST network/seam":              networkSeam,
		"GET network/journal/outstanding": journalOutstanding,
		"POST service/stop":              serviceStop,
		"POST fault":                     installFault,
		"POST runtime/state": func(e *app.Engine, body *pyjson.Object, _ *http.Request) (any, error) {
			state, _ := field(body, "state").(string)
			e.Runtime.ForceState(state)
			return nil, nil
		},
		// supervisor.workers ({ied id: pid}); the Go runtime does not publish worker pids.
		"GET runtime/workers": func(e *app.Engine, _ *pyjson.Object, _ *http.Request) (any, error) {
			workers := pyjson.NewObject()
			if e.Runtime.State() != "stopped" {
				for _, id := range e.Runtime.RunningIEDs() {
					workers.Set(id, nil)
				}
			}
			return workers, nil
		},
	} {
		parityOps[key] = op
	}
}

// networkSeam is `network.<name> = fake` for the four Windows calls of NetworkService.
func networkSeam(e *app.Engine, body *pyjson.Object, _ *http.Request) (any, error) {
	name, _ := field(body, "name").(string)
	fake := callback{}
	fake.URL, _ = field(field(body, "callback"), "url").(string)
	fake.Key, _ = field(field(body, "callback"), "key").(string)
	fake.ID, _ = field(field(body, "callback"), "id").(string)
	network := e.Network
	switch name {
	case "_list_adapters":
		network.Source = netcfg.AdapterFunc(func() ([]netcfg.Adapter, error) {
			value, err := fake.call()
			if err != nil {
				return nil, err
			}
			var adapters []netcfg.Adapter
			for _, item := range listOf(value) {
				adapters = append(adapters, adapterOf(item))
			}
			return adapters, nil
		})
	case "_list_rules":
		network.Rules = func() ([]string, error) {
			value, err := fake.call()
			return stringsOf(value), err
		}
	case "_probe":
		network.Probe = netcfg.ProberFunc(func(addresses []string) (map[string]bool, error) {
			value, err := fake.call(addresses)
			found := map[string]bool{}
			for _, address := range stringsOf(value) {
				found[address] = true
			}
			return found, err
		})
	case "_elevate":
		network.Elevator = netcfg.ElevatorFunc(func(_ string, arguments []string) (uint32, error) {
			if len(arguments) < 3 || arguments[0] != netcfg.HelperFlag {
				return 1, fmt.Errorf("unexpected helper command %v", arguments)
			}
			_, err := fake.call(wirePath(arguments[1]), arguments[2])
			var raised *remoteError
			if errors.As(err, &raised) && raised.Type == "ElevationCancelled" {
				return 1, cancelled{raised.Error()}
			}
			return 0, err
		})
	default:
		return nil, apperr.ValueError("not a network seam: " + name)
	}
	return nil, nil
}

// cancelled is the fake's ElevationCancelled: its own text, matched as ErrElevationCancelled.
type cancelled struct{ text string }

func (c cancelled) Error() string        { return c.text }
func (c cancelled) Is(target error) bool { return target == netcfg.ErrElevationCancelled }

func journalOutstanding(e *app.Engine, _ *pyjson.Object, _ *http.Request) (any, error) {
	items, err := e.Network.Journal.Outstanding()
	list := make([]any, len(items))
	for i, item := range items {
		list[i] = item
	}
	return list, err
}

// serviceStop is the tool teardown of STOP_CALLS (explorer/scanner/gateway/master).
func serviceStop(e *app.Engine, body *pyjson.Object, _ *http.Request) (any, error) {
	call, _ := field(body, "call").(string)
	switch call {
	case "explorer.disconnect":
		return e.Explorer.Disconnect(), nil
	case "explorer.close":
		e.Explorer.Close()
	case "explorer.control.close":
		e.Control.Close()
	case "scanner.stop":
		return e.Scanner.Stop(), nil
	case "gateway.stop":
		return e.Gateway.Stop(true)
	case "master.disconnect":
		return e.Master.Disconnect(true)
	default:
		return nil, apperr.ValueError("not a parity stop call: " + call)
	}
	return nil, nil
}

// installFault is a monkeypatch made while the engine runs. Only startup recovery has Go
// seams, and it has run by then (like patching the module after create_app in Python).
func installFault(_ *app.Engine, body *pyjson.Object, _ *http.Request) (any, error) {
	target, _ := field(body, "target").(string)
	if !faultSeams[target] {
		return nil, fmt.Errorf("the Go parity build has no fault seam for %s", target)
	}
	return nil, nil
}

func listOf(value any) []any { items, _ := value.([]any); return items }

func stringsOf(value any) []string {
	var result []string
	for _, item := range listOf(value) {
		if text, ok := item.(string); ok {
			result = append(result, text)
		}
	}
	return result
}

// adapterOf reads a normalized adapter dict (adapters.normalize) a fake returned.
func adapterOf(value any) netcfg.Adapter {
	text := func(key string) string { s, _ := field(value, key).(string); return s }
	adapter := netcfg.Adapter{GUID: text("guid"), Name: text("name"), Description: text("description"),
		Status: text("status"), MAC: text("mac"), Speed: text("speed"), Kind: text("kind")}
	if index, ok := field(value, "index").(int64); ok {
		adapter.Index = &index
	}
	adapter.DefaultRoute, _ = field(value, "default_route").(bool)
	for _, item := range listOf(field(value, "addresses")) {
		address := netcfg.Address{}
		address.IP, _ = field(item, "ip").(string)
		prefix, _ := field(item, "prefix").(int64)
		address.Prefix = int(prefix)
		address.SkipAsSource, _ = field(item, "skip_as_source").(bool)
		address.Origin, _ = field(item, "origin").(string)
		adapter.Addresses = append(adapter.Addresses, address)
	}
	return adapter
}
