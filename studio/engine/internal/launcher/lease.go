// Package launcher holds what both engines share around the server process: the data
// folder lease (studio.lock), the launcher.json record, server-mode options and the
// engine choice.
package launcher

// Texts of launcher/directory_lease.py, shown to the user when a second launch fails.
const (
	LeaseLinkText = "ScadaBench lock file is a link; inspect the data folder before starting"
	LeaseBusyText = "This ScadaBench data folder is already open or cannot be locked. " +
		"Close its other launcher before opening it again."
)
