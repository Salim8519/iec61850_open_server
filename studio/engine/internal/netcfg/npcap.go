package netcfg

import (
	"os"
	"path/filepath"

	"studio-engine/internal/pyjson"
)

// NpcapDownload is where the user gets the free Npcap driver.
const NpcapDownload = "https://npcap.com/#download"

// NpcapStatus is npcap.npcap_status: whether wpcap.dll is installed. GOOSE between simulated
// IEDs needs no Npcap (ADR-0073); sharing GOOSE with the network does. systemRoot "" means
// %SystemRoot% (or C:\Windows).
func NpcapStatus(systemRoot string) *pyjson.Object {
	if systemRoot == "" {
		systemRoot = os.Getenv("SystemRoot")
	}
	if systemRoot == "" {
		systemRoot = `C:\Windows`
	}
	library := filepath.Join(systemRoot, "System32", "Npcap", "wpcap.dll")
	info, err := os.Stat(library)
	installed := err == nil && info.Mode().IsRegular()
	path, message := "", "Sharing GOOSE with the network needs the free Npcap driver. Install it from npcap.com, "+
		"then restart ScadaBench. GOOSE between your simulated IEDs works without it."
	if installed {
		path = library
		message = "Npcap is installed, so ScadaBench can share GOOSE with the network and hear real devices."
	}
	return pyjson.NewObject().Set("installed", installed).Set("path", path).Set("download", NpcapDownload).
		Set("message", message)
}
