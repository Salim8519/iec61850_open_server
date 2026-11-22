package project

import "strconv"

// MaxIEDs is studio_app.contracts.limits.MAX_IEDS: the IEDs a project may hold, the most one
// import may select and the most one bulk removal may take (change 0232, owner 2026-11-22).
const MaxIEDs = 500

var maxText = strconv.Itoa(MaxIEDs)

// The limit messages, word for word as the Python backend builds them.
var (
	MsgImportSelection = "Select between 1 and " + maxText + " IEDs"
	MsgProjectFull     = "This release supports up to " + maxText + " IEDs per project"
	msgStoredList      = "Project must contain a list of at most " + maxText + " IEDs"
	msgBulkSelection   = "Select 1 to " + maxText + " unique IEDs"
)
