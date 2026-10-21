// Package scan is Find IEDs: the range rules of domain/scan_range.py and the scan service
// of tools/ied_scan.py (64 parallel TCP handshakes, then studio_live identify).
package scan

import (
	"errors"
	"strconv"
	"strings"
)

// errAddress stands for Python's AddressValueError/NetmaskValueError: the caller words it.
var errAddress = errors.New("not a valid IPv4 address or network")

func asciiDigits(text string) bool {
	if text == "" {
		return false
	}
	for index := 0; index < len(text); index++ {
		if text[index] < '0' || text[index] > '9' {
			return false
		}
	}
	return true
}

// parseAddress is ipaddress.IPv4Address(text) for a str: four decimal octets, no leading
// zeros, each at most 255; nothing is stripped.
func parseAddress(text string) (uint32, error) {
	octets := strings.Split(text, ".")
	if text == "" || len(octets) != 4 {
		return 0, errAddress
	}
	var value uint32
	for _, octet := range octets {
		if !asciiDigits(octet) || len(octet) > 3 || (len(octet) > 1 && octet[0] == '0') {
			return 0, errAddress
		}
		number, _ := strconv.Atoi(octet)
		if number > 255 {
			return 0, errAddress
		}
		value = value<<8 | uint32(number)
	}
	return value, nil
}

// parseNetwork is ipaddress.IPv4Network(text, strict=False): address/prefix, where the
// prefix may also be a netmask or a hostmask. It returns the network address and length.
func parseNetwork(text string) (uint32, int, error) {
	parts := strings.Split(text, "/")
	if len(parts) > 2 {
		return 0, 0, errAddress
	}
	address, err := parseAddress(parts[0])
	if err != nil {
		return 0, 0, err
	}
	prefix := 32
	if len(parts) == 2 {
		if prefix, err = parsePrefix(parts[1]); err != nil {
			return 0, 0, err
		}
	}
	return address & mask(prefix), prefix, nil
}

func parsePrefix(text string) (int, error) {
	if asciiDigits(text) {
		number, err := strconv.Atoi(text)
		if err != nil || number > 32 {
			return 0, errAddress
		}
		return number, nil
	}
	value, err := parseAddress(text)
	if err != nil {
		return 0, errAddress
	}
	if prefix, ok := prefixOf(value); ok {
		return prefix, nil
	}
	if prefix, ok := prefixOf(^value); ok { // a hostmask such as 0.0.0.255
		return prefix, nil
	}
	return 0, errAddress
}

// prefixOf returns the length of a contiguous netmask.
func prefixOf(value uint32) (int, bool) {
	for prefix := 0; prefix <= 32; prefix++ {
		if mask(prefix) == value {
			return prefix, true
		}
	}
	return 0, false
}

func mask(prefix int) uint32 {
	if prefix == 0 {
		return 0
	}
	return ^uint32(0) << uint(32-prefix)
}

func format(address uint32) string {
	return strconv.Itoa(int(address>>24)) + "." + strconv.Itoa(int(address>>16&255)) + "." +
		strconv.Itoa(int(address>>8&255)) + "." + strconv.Itoa(int(address&255))
}
