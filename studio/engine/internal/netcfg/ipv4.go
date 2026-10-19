package netcfg

import (
	"math/big"
	"math/bits"
	"strconv"
	"strings"
)

// ipv4Iface is Python's ipaddress.IPv4Interface: an address with its network prefix.
type ipv4Iface struct {
	IP     uint32
	Prefix int
}

const allOnes = uint32(0xFFFFFFFF)

func isASCIIDigits(text string) bool {
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

// parseIPv4 follows ipaddress._ip_int_from_string: four decimal octets, at most three digits,
// no leading zeros, each 0-255.
func parseIPv4(text string) (uint32, bool) {
	parts := strings.Split(text, ".")
	if text == "" || len(parts) != 4 {
		return 0, false
	}
	var value uint32
	for _, part := range parts {
		if !isASCIIDigits(part) || len(part) > 3 || (len(part) > 1 && part[0] == '0') {
			return 0, false
		}
		octet, _ := strconv.Atoi(part)
		if octet > 255 {
			return 0, false
		}
		value = value<<8 | uint32(octet)
	}
	return value, true
}

// ipv4Address is ipaddress.IPv4Address(value) for a decoded JSON value.
func ipv4Address(value any) (uint32, bool) {
	if flag, ok := value.(bool); ok {
		if flag {
			return 1, true
		}
		return 0, true
	}
	if number, ok := pyInt(value); ok {
		if number.Sign() < 0 || number.Cmp(big.NewInt(int64(allOnes))) > 0 {
			return 0, false
		}
		return uint32(number.Uint64()), true
	}
	text := pyStr(value)
	if strings.Contains(text, "/") {
		return 0, false
	}
	return parseIPv4(text)
}

func formatIPv4(value uint32) string {
	return strconv.Itoa(int(value>>24)) + "." + strconv.Itoa(int(value>>16&0xFF)) + "." +
		strconv.Itoa(int(value>>8&0xFF)) + "." + strconv.Itoa(int(value&0xFF))
}

// prefixFromMaskInt is ipaddress._prefix_from_ip_int: the mask must be contiguous ones.
func prefixFromMaskInt(mask uint32) (int, bool) {
	zeros := 32
	if mask != 0 {
		zeros = bits.TrailingZeros32(mask)
	}
	prefix := 32 - zeros
	var ones uint64 = (1 << uint(prefix)) - 1
	return prefix, uint64(mask)>>uint(zeros) == ones
}

// parsePrefix is ipaddress._make_netmask for a str: a prefix length, a netmask or a hostmask.
func parsePrefix(text string) (int, bool) {
	if isASCIIDigits(text) {
		trimmed := strings.TrimLeft(text, "0")
		if len(trimmed) > 2 {
			return 0, false
		}
		prefix, _ := strconv.Atoi("0" + trimmed)
		return prefix, prefix <= 32
	}
	mask, ok := parseIPv4(text)
	if !ok {
		return 0, false
	}
	if prefix, ok := prefixFromMaskInt(mask); ok {
		return prefix, true
	}
	return prefixFromMaskInt(mask ^ allOnes)
}

// parseInterface is ipaddress.IPv4Interface(text) for "address/prefix" or "address/mask".
func parseInterface(text string) (ipv4Iface, bool) {
	parts := strings.Split(text, "/")
	if len(parts) > 2 {
		return ipv4Iface{}, false
	}
	address, ok := parseIPv4(parts[0])
	if !ok {
		return ipv4Iface{}, false
	}
	prefix := 32
	if len(parts) == 2 {
		if prefix, ok = parsePrefix(parts[1]); !ok {
			return ipv4Iface{}, false
		}
	}
	return ipv4Iface{IP: address, Prefix: prefix}, true
}

func maskOf(prefix int) uint32 {
	if prefix == 0 {
		return 0
	}
	return allOnes << uint(32-prefix)
}

// Network is the network address (strict=False masking).
func (i ipv4Iface) Network() uint32 { return i.IP & maskOf(i.Prefix) }

// Broadcast is the last address of the network.
func (i ipv4Iface) Broadcast() uint32 { return i.Network() | ^maskOf(i.Prefix) }

// Contains is "address in interface.network".
func (i ipv4Iface) Contains(address uint32) bool { return address&maskOf(i.Prefix) == i.Network() }

// eachHost walks network.hosts() in order (/31 lists both, /32 its one address) until visit
// returns false.
func (i ipv4Iface) eachHost(visit func(index int, host uint32) bool) {
	first, last := i.Network(), i.Broadcast()
	if i.Prefix <= 30 {
		first, last = first+1, last-1
	}
	for index, host := 0, uint64(first); host <= uint64(last); index, host = index+1, host+1 {
		if !visit(index, uint32(host)) {
			return
		}
	}
}

func isLoopback(address uint32) bool    { return address>>24 == 127 }
func isMulticast(address uint32) bool   { return address>>28 == 0xE }
func isLinkLocal(address uint32) bool   { return address>>16 == 0xA9FE }
func isUnspecified(address uint32) bool { return address == 0 }
