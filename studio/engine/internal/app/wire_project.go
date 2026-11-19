package app

import (
	"errors"
	"fmt"
	"os"

	"studio-engine/internal/events"
	"studio-engine/internal/project"
	projects "studio-engine/internal/project/service"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime"
	"studio-engine/internal/scl"
	"studio-engine/internal/store/autosave"
	"studio-engine/internal/store/modelcache"
	"studio-engine/internal/store/projectfile"
	"studio-engine/internal/store/recovery"
	"studio-engine/internal/store/settings"
	"studio-engine/internal/store/xsd"
)

// wireProject renames old-extension project files, folds a journal left by an earlier Go run, creates the event bus, opens the
// project service (its changes are published as project_changed, like create_app's
// on_change lambda) and starts startup recovery in the background (BackgroundRecovery).
func (e *Engine) wireProject() error {
	o := e.options
	if err := os.MkdirAll(o.DataDir, 0o755); err != nil {
		return err
	}
	// The app's own .subsim files (autosave, recovery copies) become .scadabench first (ADR-0096).
	if _, err := projectfile.Migrate(o.DataDir); err != nil {
		return err
	}
	notice, err := foldJournal(o.DataDir)
	if err != nil {
		return err
	}
	e.validator = xsd.New(xsd.Config{HelperPath: o.XSDHelper, SchemasDir: o.SchemasDir, DataDir: o.DataDir})
	if err := seedVerdicts(e.validator, o.XSDSeed); err != nil {
		return err
	}
	bus := events.NewBus(events.Capacity)
	e.bus = bus
	e.startWarm = runtime.NewStartWarmer(func() *pyjson.Object { return e.Projects.Current() },
		runtime.StartWarmDelay)
	service, err := projects.New(projects.Config{
		DataDir:  o.DataDir,
		Importer: &scl.Importer{Validator: e.validator},
		// OnChange runs under the project mutation lock: it only publishes (lock order
		// lifeMu -> commandMu -> project.mutate -> table.mu -> bus.mu).
		OnChange: func(revision int64, source any) {
			publish(bus, o.Logf, "project_changed", pyjson.NewObject().Set("revision", revision).Set("source", source))
			e.startWarm.Changed() // never blocks
		},
		OnSnapshotError: func(err error) { o.Logf("Background autosave failed: %v", err) },
		ModelCache:      modelcache.New(o.DataDir),
	})
	if err != nil {
		return err
	}
	e.Projects = service
	// Auto save off is remembered (ADR-0139); it applies before the first change of this run.
	_ = service.SetAutoSave(!project.AutoSaveOff(settings.NewPreferences(o.DataDir).Load()))
	recover := service.RecoverStartup
	if o.RecoverStartup != nil {
		recover = func() (*pyjson.Object, error) { return o.RecoverStartup(service.RecoverStartup) }
	}
	if notice != nil {
		inner := recover
		recover = func() (*pyjson.Object, error) {
			result, err := inner()
			return notice.Apply(result), err
		}
	}
	e.Recovery = recovery.StartBackground(recover,
		func(err error) { o.Logf("Startup recovery stopped: %v", err) })
	return nil
}

// foldJournal folds a journal left by an earlier Go run. A journal whose content cannot be
// applied is moved to the recovery folder (the autosave is copied next to it and kept), so
// the app starts with the last autosave and shows the returned notice; disk errors, or a
// journal that cannot be moved, still stop start-up with both files kept.
func foldJournal(dataDir string) (*recovery.JournalNotice, error) {
	_, err := autosave.Fold(dataDir, nil)
	if err == nil {
		return nil, nil
	}
	if errors.Is(err, autosave.ErrJournalUnreadable) {
		if notice, moveErr := recovery.QuarantineJournal(dataDir, err); moveErr == nil {
			return notice, nil
		}
	}
	return nil, fmt.Errorf("the autosave journal of the last Go run could not be folded "+
		"(both files were kept in %s): %w", dataDir, err)
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
