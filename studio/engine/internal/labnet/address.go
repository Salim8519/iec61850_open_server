package labnet

import (
	"errors"
	"net/netip"
	"strings"
)

// CardName is the Windows name of the lab card; CardDescription is what its driver calls it.
const (
	CardName        = "ScadaBench Lab"
	CardDescription = "KM-TEST"
)

// ErrAddress is the plain reason an address cannot be put on the lab card.
var ErrAddress = errors.New("use an IPv4 address such as 192.168.216.1; loopback (127.x), " +
	"169.254.x.x, multicast, 0.x and broadcast addresses cannot be used")

// ParseAddress returns the canonical text of an address the lab card may hold: one IPv4 unicast
// host address. Spaces around it are dropped.
func ParseAddress(text string) (string, error) {
	address, err := netip.ParseAddr(strings.TrimSpace(text))
	if err != nil || !address.Is4() {
		return "", ErrAddress
	}
	octets := address.As4()
	switch {
	case address.IsLoopback(), address.IsLinkLocalUnicast(), address.IsMulticast(),
		address.IsUnspecified(), octets[0] == 0, octets[0] >= 240:
		return "", ErrAddress
	}
	return address.String(), nil
}

// IsCard reports whether a Windows adapter description is the Windows loopback (KM-TEST) card.
func IsCard(description string) bool {
	text := strings.ToLower(description)
	return strings.Contains(text, "km-test") && strings.Contains(text, "loopback")
}

// SameSlash24 lists the host addresses .130 to .254, then .20 to .129, of the address's /24: the
// order free addresses are offered in when a real device already uses the IED's own address.
func SameSlash24(address string) []string {
	parsed, err := netip.ParseAddr(address)
	if err != nil || !parsed.Is4() {
		return nil
	}
	octets := parsed.As4()
	hosts := []string{}
	for _, last := range append(span(130, 254), span(20, 129)...) {
		octets[3] = byte(last)
		hosts = append(hosts, netip.AddrFrom4(octets).String())
	}
	return hosts
}

func span(from, to int) []int {
	values := make([]int, 0, to-from+1)
	for value := from; value <= to; value++ {
		values = append(values, value)
	}
	return values
}

// InNetwork reports whether address lies in ip/prefix.
func InNetwork(address, ip string, prefix int) bool {
	parsed, err := netip.ParseAddr(address)
	if err != nil {
		return false
	}
	network, err := netip.ParseAddr(ip)
	if err != nil || prefix < 1 || prefix > 32 {
		return false
	}
	return netip.PrefixFrom(network, prefix).Masked().Contains(parsed)
}
