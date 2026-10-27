package acquisition

import (
	"strings"
	"sync"

	"studio-engine/internal/acquisition/goosedom"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// Texts of acquisition/goose_share.py.
const (
	ShareConfirm = "Read the warning and confirm before sharing GOOSE with the network"
	ShareMoved   = "The network adapter changed. Review the warning again before sharing GOOSE."
)

// GooseShare applies "Where GOOSE goes": adapter choice, risk and confirmed sharing.
type GooseShare struct {
	deps   Deps
	engine *GooseEngine

	mu        sync.Mutex
	uplink    *pyjson.Object
	holdState string
	hold      string
	alone     aloneClock

	// Waiting reports whether a GOOSE the project expects has not arrived yet (the monitor
	// sets it); nil means nothing waits.
	Waiting func() bool
}

// NewGooseShare returns the share service.
func NewGooseShare(deps Deps, engine *GooseEngine) *GooseShare {
	return &GooseShare{deps: deps, engine: engine, uplink: pyjson.NewObject()}
}

func (g *GooseShare) resolve(network *pyjson.Object) ([]*pyjson.Object, *pyjson.Object) {
	adapters, err := g.deps.Network.Adapters()
	if err != nil {
		adapters = nil
	}
	return adapters, goosedom.PickUplink(adapters, network)
}

func (g *GooseShare) installed() bool {
	return pyv.Truthy(pyv.Get(g.deps.Network.Npcap(), "installed"))
}

func (g *GooseShare) heardRealGoose() (bool, error) {
	if !g.engine.Running() {
		return false, nil
	}
	stats, err := g.engine.Stats()
	if err != nil {
		return false, err
	}
	if goosedom.PythonTwin() {
		return pyv.Truthy(pyv.Get(stats, "delivered_wire")), nil
	}
	return pyv.Truthy(pyv.Get(stats, "real_frames")), nil
}

// Preview is everything the "Where GOOSE goes" panel and its warning need.
func (g *GooseShare) Preview(project *pyjson.Object) (*pyjson.Object, error) {
	network := pyv.GetObj(project, "network")
	adapters, uplink := g.resolve(network)
	installed := g.installed()
	engine := g.engine.Snapshot()
	ieds := pyv.Objects(pyv.GetList(project, "ieds"))
	holdState, hold := goosedom.Hold(network, uplink, installed, ieds)
	state, message := goosedom.Status(network, uplink, holdState, hold, engine)
	heard, err := g.heardRealGoose()
	if err != nil {
		return nil, err
	}
	preview := pyv.O("reach", goosedom.Reach(network), "uplink", uplink,
		"uplink_guid", pyv.GetOr(network, "goose_uplink_guid", ""), "choices", goosedom.AdvancedChoices(adapters),
		"risk", goosedom.Risk(network, uplink, ieds, heard),
		"npcap", installed, "running", pyv.Get(engine, "running"), "state", state, "message", message,
		"conflicts", pyv.Get(engine, "conflicts"))
	if goosedom.PythonTwin() {
		return preview, nil
	}
	alone, err := g.nobodyThere(state)
	if err != nil {
		return nil, err
	}
	return preview.Set("goes_out", goosedom.Shares(network, uplink)).Set("alone", alone), nil
}

// Apply saves the user's choice; sharing needs confirm:true and the adapter they saw.
func (g *GooseShare) Apply(body *pyjson.Object) (*pyjson.Object, error) {
	reach := pyv.GetOr(body, "reach", "inside")
	network := pyv.Shallow(pyv.GetObj(g.deps.Projects.Current(), "network"))
	network.Set("goose_reach", reach)
	if uplinkGUID := pyv.Get(body, "uplink_guid"); uplinkGUID != nil {
		network.Set("goose_uplink_guid", strings.ToUpper(pyv.PyStr(uplinkGUID)))
	}
	shareGUID := ""
	if pyv.Equal(reach, "outside") {
		if confirm, ok := pyv.Get(body, "confirm").(bool); !ok || !confirm {
			return nil, apperr.New(ShareConfirm)
		}
		_, uplink := g.resolve(network)
		if npcap := g.deps.Network.Npcap(); !pyv.Truthy(pyv.Get(npcap, "installed")) {
			return nil, apperr.New(pyv.GetStr(npcap, "message"))
		}
		if !pyv.Truthy(pyv.Get(uplink, "guid")) {
			return nil, apperr.New(pyv.PyStr(pyv.Get(uplink, "reason")))
		}
		adapter := pyv.Get(body, "adapter_guid")
		if !pyv.Truthy(adapter) {
			adapter = ""
		}
		if strings.ToUpper(pyv.PyStr(adapter)) != pyv.GetStr(uplink, "guid") {
			return nil, apperr.New(ShareMoved)
		}
		shareGUID = pyv.GetStr(uplink, "guid")
	}
	return g.save(reach, pyv.PyStr(pyv.GetOr(network, "goose_uplink_guid", "")), shareGUID)
}

// save stores where GOOSE goes and applies it to a running engine at once.
func (g *GooseShare) save(reach any, uplinkGUID, shareGUID string) (*pyjson.Object, error) {
	if err := g.deps.Projects.SaveReach(reach, uplinkGUID, shareGUID); err != nil {
		return nil, err
	}
	project := g.deps.Projects.Current()
	if g.engine.Running() {
		if err := g.Start(project); err != nil {
			return nil, err
		}
	}
	return g.Preview(project)
}

// Start listens on the adapter at Run (or after a change while running) and shares when allowed.
func (g *GooseShare) Start(project *pyjson.Object) error {
	network := pyv.GetObj(project, "network")
	_, uplink := g.resolve(network)
	installed := g.installed()
	holdState, hold := goosedom.Hold(network, uplink, installed, pyv.Objects(pyv.GetList(project, "ieds")))
	g.mu.Lock()
	g.uplink, g.holdState, g.hold = uplink, holdState, hold
	g.mu.Unlock()
	listen := ""
	if installed {
		listen = pyv.GetStr(uplink, "guid")
	}
	share := goosedom.Shares(network, uplink) && hold == ""
	_, err := g.engine.Uplink(listen, share, goosedom.Confirmed(network, uplink))
	return err
}

// Status is the (state, message) of the GOOSE traffic view while running.
func (g *GooseShare) Status(project *pyjson.Object) (string, string) {
	g.mu.Lock()
	uplink, holdState, hold := g.uplink, g.holdState, g.hold
	g.mu.Unlock()
	return goosedom.Status(pyv.GetObj(project, "network"), uplink, holdState, hold, g.engine.Snapshot())
}

// Choose answers a conflict: stop this IED's GOOSE outside, or continue (confirmed).
func (g *GooseShare) Choose(body *pyjson.Object) (any, error) {
	choice, _ := pyv.Get(body, "choice").(string)
	if choice != "stop" && choice != "continue" {
		return nil, apperr.New("Choose 'stop' or 'continue'")
	}
	if confirm, ok := pyv.Get(body, "confirm").(bool); choice == "continue" && (!ok || !confirm) {
		return nil, apperr.New("Confirm that ScadaBench may keep sending this IED's GOOSE next to the real device")
	}
	id := pyv.Get(body, "ied_id")
	if !pyv.Truthy(id) {
		id = ""
	}
	result, err := g.engine.Gate(pyv.PyStr(id), choice)
	if err != nil {
		if IsTimeout(err) { // Python catches RuntimeError/ValueError only; a timeout stays a 503
			return nil, apperr.RuntimeError(err.Error())
		}
		return nil, apperr.New(err.Error())
	}
	return result, nil
}
