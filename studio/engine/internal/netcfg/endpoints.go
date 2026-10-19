package netcfg

import (
	"math"
	"net/netip"
	"strconv"
	"strings"

	"studio-engine/internal/apperr"
)

// EndpointIED is what the Run preflight reads of one simulated IED.
type EndpointIED struct {
	Name  string
	IP    string
	LabIP string
	Port  any // ied.get("port", 102): normally an int
}

// NetworkMode is the project's network block as the preflight reads it.
type NetworkMode struct {
	Mode         string // "loopback" or "lab"
	AdapterLabel string
}

// Binder test-binds a listener endpoint (SO_EXCLUSIVEADDRUSE on Windows) and releases it.
// Its error text is what Python's str(OSError) shows, e.g. "[WinError 10048] Only one ...".
type Binder interface {
	Bind(address string, port int) error
}

// BinderFunc adapts a function to Binder.
type BinderFunc func(address string, port int) error

// Bind calls the function.
func (f BinderFunc) Bind(address string, port int) error { return f(address, port) }

// portNumber is int(value) for the values a project can hold.
func portNumber(value any) (int64, error) {
	switch v := value.(type) {
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	case float64:
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			return int64(math.Trunc(v)), nil
		}
	case string:
		if number, err := strconv.ParseInt(strings.ReplaceAll(pyStrip(v), "_", ""), 10, 64); err == nil {
			return number, nil
		}
		return 0, apperr.ValueError("invalid literal for int() with base 10: " + stringRepr(v))
	default:
		if number, ok := pyInt(v); ok && number.IsInt64() {
			return number.Int64(), nil
		}
	}
	kind := "NoneType"
	switch value.(type) {
	case []any:
		kind = "list"
	case nil:
	default:
		kind = "dict"
	}
	return 0, apperr.ValueError("int() argument must be a string, a bytes-like object or a real number, not '" +
		kind + "'")
}

// Preflight is endpoints.preflight: a plain, fixable error unless every endpoint can be
// bound safely in the current network mode.
func Preflight(ieds []EndpointIED, network NetworkMode, adapterAddresses map[string]bool, binder Binder) error {
	type endpoint struct {
		address string
		port    int64
	}
	seen := map[endpoint]bool{}
	for _, ied := range ieds {
		host := ied.IP
		if network.Mode == "lab" && ied.LabIP != "" {
			host = ied.LabIP
		}
		address, isV4 := parseIPv4(host)
		if !isV4 {
			if parsed, err := netip.ParseAddr(host); err == nil && parsed.Is6() {
				return apperr.New(ied.Name + ": only IPv4 addresses are supported")
			}
			return apperr.ValueError(stringRepr(host) + " does not appear to be an IPv4 or IPv6 address")
		}
		text := formatIPv4(address)
		if !isLoopback(address) {
			if network.Mode != "lab" {
				return apperr.New(ied.Name + " uses " + text + ", but the project is in 'This PC only' " +
					"mode. Set up the lab network first (Network tab).")
			}
			if !adapterAddresses[text] {
				label := network.AdapterLabel
				if label == "" {
					label = "the selected adapter"
				}
				return apperr.New(ied.Name + " needs address " + text + ", which is not set up on " + label + ". " +
					"Open Network > Set up lab network and apply the address plan.")
			}
		}
		port, err := portNumber(ied.Port)
		if err != nil {
			return err
		}
		if port < 1 || port > 65535 {
			return apperr.New("MMS port must be between 1 and 65535")
		}
		key := endpoint{text, port}
		if seen[key] {
			return apperr.New("Two IEDs use " + text + ":" + strconv.FormatInt(port, 10) + "; give each IED its own address")
		}
		seen[key] = true
		if err := binder.Bind(text, int(port)); err != nil {
			return apperr.New(ied.Name + " cannot listen on " + text + ":" + strconv.FormatInt(port, 10) +
				" because another program is using it (" + err.Error() + "). Close that program or change the port.")
		}
	}
	return nil
}
