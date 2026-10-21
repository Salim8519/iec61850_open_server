package netcfg

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
)

type miscVectors struct {
	Loopback []struct {
		outcome
		IPs   []string `json:"ips"`
		After []string `json:"after"`
	} `json:"loopback"`
	Preflight []struct {
		outcome
		IEDs      json.RawMessage `json:"ieds"`
		Network   NetworkModeJSON `json:"network"`
		Addresses []string        `json:"addresses"`
		Busy      json.RawMessage `json:"busy"`
		BindError string          `json:"bind_error"`
		Binds     json.RawMessage `json:"binds"`
	} `json:"preflight"`
	Npcap   []struct{ Layout, Out string } `json:"npcap"`
	Readers []struct {
		outcome
		Reader string          `json:"reader"`
		Stdout string          `json:"stdout"`
		Calls  json.RawMessage `json:"calls"`
	} `json:"readers"`
	Shell []struct {
		outcome
		Module   string `json:"module"`
		ExitCode int    `json:"exit_code"`
		Stderr   string `json:"stderr"`
		Calls    []struct {
			Argv    []string `json:"argv"`
			Timeout float64  `json:"timeout"`
			Flags   int      `json:"flags"`
		} `json:"calls"`
	} `json:"shell"`
	Fallback struct {
		outcome
		Fallback    string `json:"fallback"`
		Script      string `json:"script"`
		RulesScript string `json:"rules_script"`
	} `json:"fallback"`
	Probe []struct {
		outcome
		Addresses []string `json:"addresses"`
		Asked     []string `json:"asked"`
	} `json:"probe"`
}

// NetworkModeJSON is the network block of a preflight vector.
type NetworkModeJSON struct {
	Mode         string `json:"mode"`
	AdapterLabel string `json:"adapter_label"`
}

func TestLoopbackPreflightAndNpcapMatchPython(t *testing.T) {
	var v miscVectors
	loadVectors(t, "misc.json", &v)
	for index, c := range v.Loopback {
		ips := append([]string{}, c.IPs...)
		err := AssignLoopback(ips)
		checkOutcome(t, fmt.Sprintf("loopback %d", index), "null", err, c.outcome, true)
		if strings.Join(ips, ",") != strings.Join(c.After, ",") {
			t.Errorf("loopback %d: %v want %v", index, ips, c.After)
		}
	}
	for index, c := range v.Preflight {
		ieds := []EndpointIED{}
		for _, object := range objects(py(t, c.IEDs)) {
			port, found := object.Get("port")
			if !found {
				port = int64(102)
			}
			ieds = append(ieds, EndpointIED{Name: field(object, "name", ""), IP: field(object, "ip", ""),
				LabIP: field(object, "lab_ip", ""), Port: port})
		}
		addresses := map[string]bool{}
		for _, address := range c.Addresses {
			addresses[address] = true
		}
		busy := dumpsText(t, py(t, c.Busy))
		binds := []any{}
		binder := BinderFunc(func(address string, port int) error {
			endpoint := []any{address, port}
			binds = append(binds, endpoint)
			if strings.Contains(busy, dumpsText(t, endpoint)) {
				return errors.New(c.BindError)
			}
			return nil
		})
		err := Preflight(ieds, NetworkMode{Mode: c.Network.Mode, AdapterLabel: c.Network.AdapterLabel}, addresses, binder)
		label := fmt.Sprintf("preflight %d", index)
		checkOutcome(t, label, "null", err, c.outcome, true)
		checkRecorded(t, label+" binds", binds, c.Binds)
	}
	for _, c := range v.Npcap {
		root := t.TempDir()
		library := filepath.Join(root, "System32", "Npcap", "wpcap.dll")
		switch c.Layout {
		case "file":
			os.MkdirAll(filepath.Dir(library), 0o777)
			os.WriteFile(library, []byte("MZ"), 0o666)
		case "folder":
			os.MkdirAll(library, 0o777)
		}
		status := NpcapStatus(root)
		path, _ := status.Get("path")
		status.Set("path", strings.Replace(path.(string), root, "<ROOT>", 1))
		if got := dumpsText(t, status); got != c.Out {
			t.Errorf("npcap %s:\n got %s\nwant %s", c.Layout, got, c.Out)
		}
	}
}

// recordingShell answers every PowerShell call with stdout and records the script.
func recordingShell(stdout string, calls *[]any) Executor {
	return ExecutorFunc(func(command Command) (Output, error) {
		*calls = append(*calls, pyjson.NewObject().Set("script", command.Argv[4]).
			Set("timeout", int64(command.Timeout/time.Second)))
		return Output{Stdout: stdout}, nil
	})
}

func runReader(t *testing.T, name string, run Executor) (string, error) {
	sortedText := func(set map[string]bool) []any {
		names := []string{}
		for name := range set {
			names = append(names, name)
		}
		sort.Strings(names)
		return toAny(names)
	}
	switch name {
	case "gateway_rules", "studio_rules":
		read := ListGatewayRules
		if name == "studio_rules" {
			read = ListStudioRules
		}
		names, err := read(run)
		return dumpsText(t, toAny(names)), err
	case "blocking", "blocking_quote":
		program := `C:\Studio\native\studio_gateway.exe`
		if name == "blocking_quote" {
			program = `C:\O'Brien\g.exe`
		}
		rules, err := BlockingRules(program, run)
		items := []any{}
		for _, rule := range rules {
			items = append(items, pyjson.NewObject().Set("name", rule.Name).Set("display", rule.Display))
		}
		return dumpsText(t, items), err
	}
	names := []string{"a'b", "TCP x"}
	if name == "state_empty" {
		names = nil
	}
	gateway, disabled, err := GatewayState(names, run)
	return dumpsText(t, []any{sortedText(gateway), sortedText(disabled)}), err
}

func TestFirewallReadersMatchPython(t *testing.T) {
	var v miscVectors
	loadVectors(t, "misc.json", &v)
	for index, c := range v.Readers {
		calls := []any{}
		got, err := runReader(t, c.Reader, recordingShell(c.Stdout, &calls))
		label := fmt.Sprintf("reader %d %s %q", index, c.Reader, c.Stdout)
		checkOutcome(t, label, got, err, c.outcome, true)
		checkRecorded(t, label+" calls", calls, c.Calls)
	}
	prefixes := map[string]string{"adapters": adapterShellError, "firewall_rules": firewallReadError}
	for index, c := range v.Shell {
		var commands []Command
		run := ExecutorFunc(func(command Command) (Output, error) {
			commands = append(commands, command)
			return Output{ExitCode: c.ExitCode, Stdout: "[]", Stderr: c.Stderr}, nil
		})
		got, err := shellQuery(run, "Get-Thing", 7*time.Second, prefixes[c.Module])
		checkOutcome(t, fmt.Sprintf("shell %d", index), got, err, c.outcome, true)
		want := c.Calls[0]
		if strings.Join(commands[0].Argv, "|") != strings.Join(want.Argv, "|") || commands[0].Timeout != 7*time.Second ||
			want.Flags != 0x08000000 {
			t.Errorf("shell %d: %v want %v", index, commands[0], want)
		}
	}
	if v.Fallback.Script != AdapterScript || v.Fallback.RulesScript != RulesScript {
		t.Error("PowerShell scripts differ from Python")
	}
	source := SystemAdapters{Native: func() ([]any, error) { return nil, errors.New("[Errno 50] not supported") },
		Shell: ExecutorFunc(func(command Command) (Output, error) {
			if command.Argv[4] == AdapterScript {
				return Output{Stdout: v.Fallback.Fallback}, nil
			}
			return Output{}, nil
		})}
	adapters, err := source.Adapters()
	checkOutcome(t, "fallback", dumpsText(t, AdaptersObject(adapters)), err, v.Fallback.outcome, true)
}

func TestARPProbeMatchesPython(t *testing.T) {
	var v miscVectors
	loadVectors(t, "misc.json", &v)
	for index, c := range v.Probe {
		var lock sync.Mutex
		asked := []string{}
		prober := ARPProber{Answers: func(address string) bool {
			lock.Lock()
			asked = append(asked, address)
			lock.Unlock()
			return strings.HasSuffix(address, ".21") || address == "10.0.0.1"
		}}
		used, err := prober.InUse(c.Addresses)
		found := []string{}
		for address := range used {
			found = append(found, address)
		}
		sort.Strings(found)
		sort.Strings(asked)
		label := fmt.Sprintf("probe %d", index)
		checkOutcome(t, label, dumpsText(t, toAny(found)), err, c.outcome, false)
		if err == nil && strings.Join(asked, ",") != strings.Join(c.Asked, ",") {
			t.Errorf("%s asked %v want %v", label, asked, c.Asked)
		}
	}
}
