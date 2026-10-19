// studio_goose is ScadaBench's GOOSE engine. The backend starts one per Run and talks
// to it over stdin/stdout frames; diagnostics go to stderr.
package main

import (
	"log"
	"os"

	"studio-goose-engine/internal/engine"
	"studio-goose-engine/internal/ipc"
	"studio-goose-engine/internal/uplink"
)

func main() {
	log.SetOutput(os.Stderr)
	restore := millisecondTimer()
	defer restore()
	out := ipc.NewWriter(os.Stdout)
	emit := func(event map[string]any) {
		if err := out.Write(event); err != nil {
			log.Printf("event lost: %v", err)
		}
	}
	e := engine.New(emit, func(guid string, onFrame func([]byte), onState func(string, string)) uplink.Port {
		return uplink.Start(guid, onFrame, onState)
	})
	if err := ipc.Serve(os.Stdin, out, engine.Handlers(e), e.Close); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
