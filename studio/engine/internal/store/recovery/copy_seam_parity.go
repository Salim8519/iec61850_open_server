//go:build parity

package recovery

// SetCopySeam replaces the copy of an unreadable autosave with wrap, which receives the
// source, the destination and the real copy. Only the parity test build has it: it carries
// tests/parity's fault "studio_app.projects.startup_recovery.shutil.copyfile". The product
// never calls it.
func SetCopySeam(wrap func(source, destination string, original func(string, string) error) error) {
	copyUnreadable = func(source, destination string) error { return wrap(source, destination, copyFile) }
}
