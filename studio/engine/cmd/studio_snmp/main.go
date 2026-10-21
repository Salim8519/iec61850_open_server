// studio_snmp is ScadaBench's SNMP engine (ADR-0092). The server starts it while SNMP devices
// are simulated or read and talks to it over stdin/stdout frames; diagnostics go to stderr.
package main

import (
	"log"
	"os"

	"studio-engine/internal/snmp/worker"
)

func main() {
	log.SetOutput(os.Stderr)
	if err := worker.Serve(os.Stdin, os.Stdout); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
