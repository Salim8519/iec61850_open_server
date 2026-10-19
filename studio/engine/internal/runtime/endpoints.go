package runtime

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// endpoint is an IED's listening address.
type endpoint struct {
	host string
	port any // the stored port value (int, as Python keeps it)
}

func portNumber(port any) int64 {
	value, _ := pyval.ToInt(port)
	number, _ := value.(int64)
	return number
}

// effectiveEndpoint is network_settings.effective_endpoint.
func effectiveEndpoint(ied, network *pyjson.Object) endpoint {
	port, present := ied.Get("port")
	if !present {
		port = int64(102)
	}
	if pyval.Str(network, "mode") == "lab" && pyval.Truthy(pyval.Get(ied, "lab_ip")) {
		return endpoint{pyval.Str(ied, "lab_ip"), port}
	}
	return endpoint{pyval.Str(ied, "ip"), port}
}

// realEndpoint is network_settings.real_endpoint (host "" when missing).
func realEndpoint(ied *pyjson.Object) endpoint {
	live := pyval.Map(ied, "live")
	if live == nil || live.Len() == 0 {
		live = pyval.Obj("ip", "", "port", int64(102))
	}
	host := pyval.Str(live, "ip")
	if host == "" {
		host = pyval.Str(pyval.Map(ied, "scl_address"), "ip")
	}
	return endpoint{host, pyval.Get(live, "port")}
}

// endpointConflicts is projects/ied_settings.endpoint_conflicts.
func endpointConflicts(ieds []*pyjson.Object, network *pyjson.Object) []string {
	var problems []string
	used := map[string]string{}
	for _, ied := range ieds {
		var point endpoint
		switch roleOf(ied) {
		case "simulated":
			point = effectiveEndpoint(ied, network)
		case "real":
			point = realEndpoint(ied)
			if point.host == "" {
				continue
			}
		default:
			continue
		}
		key := point.host + ":" + pyval.Repr(point.port)
		if other, ok := used[key]; ok && other != "" {
			problems = append(problems, fmt.Sprintf("%s and %s both use %s. Give each IED its own address.",
				pyval.Str(ied, "name"), other, key))
		}
		used[key] = pyval.Str(ied, "name")
	}
	return problems
}

// assignEndpoints gives every IED without an address the first free 127.50.0.x (loopback.py).
func (c *Coordinator) assignEndpoints(ieds []*pyjson.Object) error {
	used := map[string]bool{}
	for _, ied := range ieds {
		if ip := pyval.Str(ied, "ip"); ip != "" {
			used[ip] = true
		}
	}
	for _, ied := range ieds {
		if pyval.Truthy(pyval.Get(ied, "ip")) {
			continue
		}
		assigned := false
		for index := 2; index < 255; index++ {
			candidate := fmt.Sprintf("127.50.0.%d", index)
			if !used[candidate] {
				ied.Set("ip", candidate)
				c.opt.Project.AssignIP(pyval.Str(ied, "id"), candidate)
				used[candidate] = true
				assigned = true
				break
			}
		}
		if !assigned {
			return apperr.New("No free loopback addresses")
		}
	}
	return nil
}

// preflight is networking/endpoints.preflight.
func preflight(ieds []*pyjson.Object, network *pyjson.Object, allowed []string) error {
	seen := map[string]bool{}
	permitted := map[string]bool{}
	for _, address := range allowed {
		permitted[address] = true
	}
	for _, ied := range ieds {
		name := pyval.Str(ied, "name")
		point := effectiveEndpoint(ied, network)
		address := net.ParseIP(point.host)
		if address == nil {
			return apperr.ValueError(fmt.Sprintf("'%s' does not appear to be an IPv4 or IPv6 address", point.host))
		}
		if address.To4() == nil {
			return apperr.New(name + ": only IPv4 addresses are supported")
		}
		text := address.To4().String()
		if !address.IsLoopback() {
			if pyval.Str(network, "mode") != "lab" {
				return apperr.New(fmt.Sprintf("%s uses %s, but the project is in 'This PC only' mode. "+
					"Set up the lab network first (Network tab).", name, text))
			}
			if !permitted[text] {
				label := pyval.Str(network, "adapter_label")
				if label == "" {
					label = "the selected adapter"
				}
				return apperr.New(fmt.Sprintf("%s needs address %s, which is not set up on %s. "+
					"Open Network > Set up lab network and apply the address plan.", name, text, label))
			}
		}
		port := portNumber(point.port)
		if port < 1 || port > 65535 {
			return apperr.New("MMS port must be between 1 and 65535")
		}
		key := fmt.Sprintf("%s:%d", text, port)
		if seen[key] {
			return apperr.New(fmt.Sprintf("Two IEDs use %s:%d; give each IED its own address", text, port))
		}
		seen[key] = true
		if err := bindTest(text, port); err != nil {
			return apperr.New(fmt.Sprintf("%s cannot listen on %s:%d because another program is using it (%s). "+
				"Close that program or change the port.", name, text, port, err))
		}
	}
	return nil
}

// bindTest binds a TCP listener with SO_EXCLUSIVEADDRUSE like the Python preflight.
func bindTest(host string, port int64) error {
	config := net.ListenConfig{Control: exclusiveAddress}
	listener, err := config.Listen(contextBackground(), "tcp4", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return osErrorText(err)
	}
	return listener.Close()
}

// osErrorText renders a socket error as Python's OSError: "[WinError N] text".
func osErrorText(err error) error {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		message := strings.TrimRight(errno.Error(), ". \r\n")
		return fmt.Errorf("[WinError %d] %s", uint(errno), message)
	}
	return err
}
