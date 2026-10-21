// Package netset validates the project network mode and the per-IED role, endpoint, GOOSE
// and test-mode settings exactly as domain/network_settings.py and domain/test_mode.py do.
package netset

import (
	"strconv"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
)

// parseIPv4 is ipaddress.IPv4Address(text) for a string: four dot-separated ASCII decimal
// octets of at most three digits, no leading zeros, each at most 255.
func parseIPv4(text string) ([4]int, bool) {
	var octets [4]int
	if text == "" || strings.Contains(text, "/") {
		return octets, false
	}
	parts := strings.Split(text, ".")
	if len(parts) != 4 {
		return octets, false
	}
	for index, part := range parts {
		if part == "" || len(part) > 3 || (len(part) > 1 && part[0] == '0') {
			return octets, false
		}
		for _, c := range []byte(part) {
			if c < '0' || c > '9' {
				return octets, false
			}
		}
		value, _ := strconv.Atoi(part)
		if value > 255 {
			return octets, false
		}
		octets[index] = value
	}
	return octets, true
}

// ipv4 is network_settings._ipv4: "" or None give "" when allowed; anything else must be a
// normal IPv4 address (loopback only when allowed, never multicast or 0.0.0.0).
func ipv4(value any, label string, allowEmpty, allowLoopback bool) (string, error) {
	if (value == nil || value == "") && allowEmpty {
		return "", nil
	}
	octets, ok := parseIPv4(pyval.Str(value))
	if !ok {
		return "", apperr.ValueError(label + " must be an IPv4 address such as 192.168.1.20")
	}
	text := strconv.Itoa(octets[0]) + "." + strconv.Itoa(octets[1]) + "." + strconv.Itoa(octets[2]) + "." +
		strconv.Itoa(octets[3])
	loopback := octets[0] == 127
	multicast := octets[0] >= 224 && octets[0] <= 239
	unspecified := octets == [4]int{}
	if (loopback && !allowLoopback) || multicast || unspecified {
		return "", apperr.ValueError(label + " must be a normal device address, not " + text)
	}
	return text, nil
}

// port is network_settings._port: a whole number from 1 to 65535 (not a bool).
func port(value any, label string) (any, error) {
	if !pyval.IsInt(value) || !pyval.Between(value, 1, 65535) {
		return nil, apperr.ValueError(label + " must be a whole number from 1 to 65535 (MMS normally uses 102)")
	}
	return value, nil
}
