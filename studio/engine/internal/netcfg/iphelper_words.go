package netcfg

import (
	"fmt"
	"strconv"
	"strings"
)

// IP Helper constants (ip_helper.py).
const (
	ifTypeSoftwareLoopback = 24
	ifTypeTunnel           = 131
	// Skip anycast, multicast, DNS servers and DNS information (0x800, GAA_FLAG_SKIP_DNS_INFO: without it
	// each listing asks the Windows DNS Client about every card, ADR-0128); include gateways.
	gaaFlags = 0x2 | 0x4 | 0x8 | 0x800 | 0x80
	afInet                 = 2
)

var hiddenWords = []string{"wi-fi direct virtual adapter", "wan miniport", "kernel debug"}

var prefixOrigins = map[int32]string{0: "Other", 1: "Manual", 2: "WellKnown", 3: "Dhcp", 4: "RouterAdvertisement"}

// prefixOrigin is PREFIX_ORIGINS.get(value, "Other").
func prefixOrigin(value int32) string {
	if word, found := prefixOrigins[value]; found {
		return word
	}
	return "Other"
}

// StatusWord is ip_helper.status_word: the words Get-NetAdapter uses.
func StatusWord(operStatus int32) string {
	switch operStatus {
	case 1:
		return "Up"
	case 2, 7:
		return "Disconnected"
	}
	return "Down"
}

// SpeedWords is ip_helper.speed_words, e.g. "1 Gbps", "866.7 Mbps", "0 bps".
func SpeedWords(bitsPerSecond uint64) string {
	if bitsPerSecond == 0 || bitsPerSecond >= 1<<63 {
		return "0 bps"
	}
	for _, unit := range []struct {
		name string
		size uint64
	}{{"Gbps", 1e9}, {"Mbps", 1e6}, {"Kbps", 1e3}} {
		if bitsPerSecond >= unit.size {
			value := float64(bitsPerSecond) / float64(unit.size)
			return strconv.FormatFloat(value, 'g', 6, 64) + " " + unit.name
		}
	}
	return strconv.FormatUint(bitsPerSecond, 10) + " bps"
}

// MACWords is ip_helper.mac_words: the first length bytes (at most 8) as XX-XX-...
func MACWords(raw [8]byte, length uint32) string {
	if length > 8 {
		length = 8
	}
	parts := make([]string, length)
	for index := range parts {
		parts[index] = fmt.Sprintf("%02X", raw[index])
	}
	return strings.Join(parts, "-")
}

// IsListed is ip_helper.is_listed: adapters Get-NetAdapter shows (no loopback, tunnels or
// hidden miniports).
func IsListed(ifType uint32, description string) bool {
	if ifType == ifTypeSoftwareLoopback || ifType == ifTypeTunnel {
		return false
	}
	return !containsAny(strings.ToLower(description), hiddenWords...)
}
