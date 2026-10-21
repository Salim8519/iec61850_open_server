package app

import (
	"fmt"
	"os"

	"studio-engine/internal/events"
	projects "studio-engine/internal/project/service"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
	"studio-engine/internal/store/autosave"
	"studio-engine/internal/store/modelcache"
	"studio-engine/internal/store/recovery"
	"studio-engine/internal/store/xsd"
)

// wireProject folds a journal left by an earlier Go run, creates the event bus, opens the
// project service (its changes are published as project_changed, like create_app's
// on_change lambda) and starts startup recovery in the background (BackgroundRecovery).
func (e *Engine) wireProject() error {
	o := e.options
	if err := os.MkdirAll(o.DataDir, 0o755); err != nil {
		return err
	}
	if _, err := autosave.Fold(o.DataDir, nil); err != nil {
		return fmt.Errorf("the autosave journal of the last Go run could not be folded "+
			"(both files were kept in %s): %w", o.DataDir, err)
	}
	e.validator = xsd.New(xsd.Config{HelperPath: o.XSDHelper, SchemasDir: o.SchemasDir, DataDir: o.DataDir})
	if err := seedVerdicts(e.validator, o.XSDSeed); err != nil {
		return err
	}
	bus := events.NewBus(events.Capacity)
	e.bus = bus
	service, err := projects.New(projects.Config{
		DataDir:  o.DataDir,
		Importer: &scl.Importer{Validator: e.validator},
		// OnChange runs under the project mutation lock: it only publishes (lock order
		// lifeMu -> commandMu -> project.mutate -> table.mu -> bus.mu).
		OnChange: func(revision int64, source any) {
			publish(bus, o.Logf, "project_changed", pyjson.NewObject().Set("revision", revision).Set("source", source))
		},
		OnSnapshotError: func(err error) { o.Logf("Background autosave failed: %v", err) },
		ModelCache:      modelcache.New(o.DataDir),
	})
	if err != nil {
		return err
	}
	e.Projects = service
	recover := service.RecoverStartup
	if o.RecoverStartup != nil {
		recover = func() (*pyjson.Object, error) { return o.RecoverStartup(service.RecoverStartup) }
	}
	e.Recovery = recovery.StartBackground(recover,
		func(err error) { o.Logf("Startup recovery stopped: %v", err) })
	return nil
}

// seedVerdicts imports verdict files written by Python (lists of {sha256, schema, status,
// errors}) into the validator's cache.
func seedVerdicts(validator *xsd.Validator, files []string) error {
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("XSD verdict seed: %w", err)
		}
		entries, err := xsd.DecodeSeed(data)
		if err != nil {
			return fmt.Errorf("XSD verdict seed %s: %w", file, err)
		}
		if _, err := validator.ImportSeed(entries); err != nil {
			return fmt.Errorf("XSD verdict seed %s: %w", file, err)
		}
	}
	return nil
}

// publish puts one event on the bus; an event that cannot be encoded is logged (Python's
// json.dumps would have raised inside the publishing thread).
func publish(bus *events.Bus, logf func(string, ...any), kind string, payload *pyjson.Object) *events.Event {
	event, err := bus.Publish(kind, payload)
	if err != nil {
		logf("Event %s could not be published: %v", kind, err)
	}
	return event
}
