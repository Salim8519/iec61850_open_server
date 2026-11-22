package archive

import "testing"

// lowerLimits shrinks the size limits for one test, as the Python tests monkeypatch
// MAX_UNPACKED / MAX_LEGACY_UNPACKED, so a limit test does not build a 1 GiB document.
func lowerLimits(t *testing.T) {
	unpacked, legacy := MaxUnpacked, MaxLegacyUnpacked
	MaxUnpacked, MaxLegacyUnpacked = 1024, 4096
	t.Cleanup(func() { MaxUnpacked, MaxLegacyUnpacked = unpacked, legacy })
}
