//go:build !windows

package desktop

// RequestShow always reports false outside Windows, as single_instance.request_show did.
func RequestShow(dir string) bool { return false }

// StartShowListener listens to nothing outside Windows; close does nothing.
func StartShowListener(dir string, onShow func(), log *Logger) (close func()) {
	return func() {}
}
