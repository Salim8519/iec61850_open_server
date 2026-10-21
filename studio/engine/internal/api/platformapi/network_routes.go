package platformapi

import (
	"errors"
	"net/http"

	"studio-engine/internal/apperr"
	"studio-engine/internal/httpapi"
	"studio-engine/internal/netcfg"
	"studio-engine/internal/pyjson"
)

// requireStopped is api/context.require_stopped.
func (a *Area) requireStopped() error {
	if a.deps.Runtime.State() != "stopped" {
		return apperr.New("Stop the simulation before editing its structure")
	}
	return nil
}

// locked runs fn under the coordinator's cycle_lock after require_stopped, as
// `with runtime().cycle_lock: require_stopped(); ...` does.
func (a *Area) locked(fn func() (any, error)) (any, error) {
	lock := a.deps.Runtime.CycleLock()
	lock.Lock()
	defer lock.Unlock()
	if err := a.requireStopped(); err != nil {
		return nil, err
	}
	return fn()
}

// networkStatus is GET /network: project["network"] and the Npcap state.
func (a *Area) networkStatus(r *http.Request) (any, error) {
	return pyjson.NewObject().Set("network", a.deps.Projects.Network()).Set("npcap", a.deps.Npcap()), nil
}

// networkAdapters is GET /network/adapters; ?refresh=1 (exactly) skips the 10 s cache.
func (a *Area) networkAdapters(r *http.Request) (any, error) {
	refresh, _, err := httpapi.Arg(r, "refresh")
	if err != nil {
		return nil, err
	}
	adapters, err := a.deps.Network.Adapters(refresh == "1")
	if err != nil {
		return nil, err
	}
	return pyjson.NewObject().Set("adapters", netcfg.AdaptersObject(adapters)), nil
}

// planProgress is GET /network/plan/progress: steps, answered addresses and rows of the
// running (or last) address check, polled by Network setup while its preview runs.
func (a *Area) planProgress(r *http.Request) (any, error) {
	return a.deps.Network.PreviewProgress(), nil
}

// networkMode is PUT /network: update_network(projects, body()) while stopped.
func (a *Area) networkMode(r *http.Request) (any, error) {
	return a.locked(func() (any, error) {
		body, err := bodyOrEmpty(r)
		if err != nil {
			return nil, err
		}
		network, err := a.deps.Projects.UpdateNetwork(body)
		if err != nil {
			return nil, err
		}
		return pyjson.NewObject().Set("network", network), nil
	})
}

// strategyRefused is the ValueError plan_addresses raises for an unknown strategy.
const strategyRefused = "Address strategy must be 'scd' or 'adapter'"

// planPreview is POST /network/plan/preview: {"adapter_guid", "strategy" = "scd", "manual"}.
func (a *Area) planPreview(r *http.Request) (any, error) {
	data, err := bodyOrEmpty(r)
	if err != nil {
		return nil, err
	}
	if err := a.requireStopped(); err != nil {
		return nil, err
	}
	guid, err := get(data, "adapter_guid", nil)
	if err != nil {
		return nil, err
	}
	raw, _ := get(data, "strategy", "scd")
	manual, _ := get(data, "manual", nil)
	strategy, isText := raw.(string)
	if !isText {
		// Not "scd" or "adapter": Python's `strategy not in STRATEGIES` refuses it with the
		// ValueError once the adapter and IEDs passed, or with a TypeError (500) when the
		// value cannot be hashed (a list or an object).
		strategy = "\x00not a strategy"
	}
	result, err := a.deps.Network.Preview(guid, strategy, manual)
	if !isText && isStrategyRefusal(err) && unhashable(raw) {
		return nil, errors.New("TypeError: unhashable strategy")
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

func isStrategyRefusal(err error) bool {
	var failure *apperr.Error
	return errors.As(err, &failure) && failure.Kind == apperr.Value && failure.Message == strategyRefused
}

func unhashable(value any) bool {
	switch value.(type) {
	case []any, *pyjson.Object:
		return true
	}
	return false
}

// planApply is POST /network/plan/apply: {"fingerprint", "confirm"} while stopped.
func (a *Area) planApply(r *http.Request) (any, error) {
	data, err := bodyOrEmpty(r)
	if err != nil {
		return nil, err
	}
	return a.locked(func() (any, error) {
		fingerprint, confirm, err := pair(data, "fingerprint", "confirm")
		if err != nil {
			return nil, err
		}
		return a.deps.Network.Apply(fingerprint, confirm)
	})
}

// cleanupPreview is GET /network/cleanup/preview (allowed while running).
func (a *Area) cleanupPreview(r *http.Request) (any, error) {
	return a.deps.Network.CleanupPreview()
}

// cleanupApply is POST /network/cleanup/apply: {"fingerprint", "confirm"} while stopped.
func (a *Area) cleanupApply(r *http.Request) (any, error) {
	data, err := bodyOrEmpty(r)
	if err != nil {
		return nil, err
	}
	return a.locked(func() (any, error) {
		fingerprint, confirm, err := pair(data, "fingerprint", "confirm")
		if err != nil {
			return nil, err
		}
		return a.deps.Network.Cleanup(fingerprint, confirm)
	})
}

// firewallPreview is GET /network/gateway-firewall (ADR-0062).
func (a *Area) firewallPreview(r *http.Request) (any, error) {
	return a.deps.Firewall.Preview()
}

// firewallApply is POST /network/gateway-firewall: {"fingerprint", "confirm"}.
func (a *Area) firewallApply(r *http.Request) (any, error) {
	data, err := bodyOrEmpty(r)
	if err != nil {
		return nil, err
	}
	fingerprint, confirm, err := pair(data, "fingerprint", "confirm")
	if err != nil {
		return nil, err
	}
	return a.deps.Firewall.Apply(fingerprint, confirm)
}
