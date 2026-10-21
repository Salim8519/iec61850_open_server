package main

import (
	"errors"
	"fmt"
	"os"

	"studio-engine/internal/launcher"
	"studio-engine/internal/netcfg"
	"studio-engine/internal/store/autosave"
	"studio-engine/internal/store/projectfile"
)

// networkHelper is entrypoint.py's "--network-helper" mode: the elevated helper that
// applies an exact, confirmed lab-network request (networking/helper/runner.main).
func networkHelper(args []string) int {
	return netcfg.HelperMain(args, netcfg.SystemExecutor{})
}

// foldJournal is "--fold-journal DATA_DIR": with the data-folder lease held (so no engine
// runs on it), old .subsim files of the app are renamed to .scadabench (ADR-0096) and the
// autosave journal a Go run left behind is folded into autosave.scadabench, so the Python
// engine opens exactly what the Go engine had saved. The launcher runs it before starting Python. Exit 0 when nothing was left or the fold succeeded, 1 otherwise (both
// files are then kept).
func foldJournal(args []string) int {
	if len(args) != 1 || args[0] == "" {
		fmt.Fprintln(os.Stderr, "usage: studio_server --fold-journal DATA_DIR")
		return 2
	}
	lease, err := launcher.AcquireLease(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer lease.Release()
	if _, err := projectfile.Migrate(args[0]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	applied, err := autosave.Fold(args[0], nil)
	if errors.Is(err, autosave.ErrJournalMismatch) {
		fmt.Fprintf(os.Stderr, "The Go engine's autosave journal does not belong to autosave.scadabench in %s; "+
			"both files were kept. Start the Go engine to inspect the project, or move the journal aside.\n", args[0])
		return 1
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "The Go engine's autosave journal could not be folded (both files were kept): %v\n", err)
		return 1
	}
	if applied > 0 {
		fmt.Printf("Folded %d saved change(s) of the Go engine into autosave.scadabench.\n", applied)
	}
	return 0
}
