package app

import (
	"strconv"
	"time"

	"studio-engine/internal/netcfg"
	"studio-engine/internal/project"
	projects "studio-engine/internal/project/service"
	"studio-engine/internal/pyjson"
)

// acquisitionProjects is the project service as acquisition.Projects: the three project
// mutations Python made in place under ProjectService.lock are committed through Update
// (journal line, revision, project_changed).
type acquisitionProjects struct{ *projects.Service }

// RunID is the health report's f"{session_id}-{revision}".
func (p acquisitionProjects) RunID() string {
	_, revision := p.State()
	return p.SessionID() + "-" + strconv.FormatInt(revision, 10)
}

// SetTestMode is test_mode_apply.set_test_mode's save: ied["test_mode"] = settings,
// save, changed; it returns the IED as published.
func (p acquisitionProjects) SetTestMode(iedID string, settings *pyjson.Object) (*pyjson.Object, error) {
	return p.Update(func(draft *pyjson.Object) (*pyjson.Object, error) {
		ied, err := project.FindIED(draft, iedID)
		if err != nil {
			return nil, err
		}
		ied.Set("test_mode", settings)
		return ied, nil
	})
}

// SaveReach is projects/goose_reach.save_reach.
func (p acquisitionProjects) SaveReach(reach any, uplinkGUID, shareGUID string) error {
	_, err := p.Update(func(draft *pyjson.Object) (*pyjson.Object, error) {
		return project.SaveReach(draft, reach, uplinkGUID, shareGUID)
	})
	return err
}

// systemNetwork is networking.adapters.list_adapters (IP Helper, PowerShell fallback) and
// npcap.npcap_status for the GOOSE share and the SV monitor.
type systemNetwork struct{}

func (systemNetwork) Adapters() ([]*pyjson.Object, error) {
	adapters, err := netcfg.SystemAdapters{Shell: netcfg.SystemExecutor{}}.Adapters()
	if err != nil {
		return nil, err
	}
	objects := make([]*pyjson.Object, len(adapters))
	for i, adapter := range adapters {
		objects[i] = adapter.Object()
	}
	return objects, nil
}

func (systemNetwork) Npcap() *pyjson.Object { return netcfg.NpcapStatus("") }

// systemClock is time.time() and time.monotonic().
type systemClock struct{}

var processStart = time.Now()

func (systemClock) Now() time.Time { return time.Now() }

func (systemClock) Monotonic() float64 { return time.Since(processStart).Seconds() }
