package hostkey

import (
	"net"
	"regexp"
	"sort"
	"strconv"
)

// Adapter is what the rules need to know about one network adapter of this PC.
type Adapter struct {
	Name        string      // the friendly name, e.g. "vEthernet (Default Switch)"
	Description string      // e.g. "Hyper-V Virtual Ethernet Adapter #2"
	Up          bool        // operational status "up"
	IPv4        []net.IPNet // each address with its mask (IP is the address itself)
	Gateways    []net.IP    // IPv4 default gateways of this adapter
}

// The adapters of the virtual switches the contract names: Hyper-V "vEthernet", VMware VMnet1 and
// VMnet8 (host-only and NAT) and VirtualBox "Host-Only". Names are matched on the friendly name
// and the description, ignoring case.
var (
	hyperV     = regexp.MustCompile(`(?i)^vethernet|hyper-v virtual ethernet adapter`)
	vmware     = regexp.MustCompile(`(?i)vmnet(1|8)([^0-9]|$)`)
	virtualBox = regexp.MustCompile(`(?i)virtualbox host-only`)
	// wsl is the switch of the Windows Subsystem for Linux: no Windows program runs behind it.
	wsl = regexp.MustCompile(`(?i)\bwsl\b`)
)

// IsVirtualSwitch reports whether a is the host side of a virtual switch.
func IsVirtualSwitch(a Adapter) bool {
	text := a.Name + "\n" + a.Description
	if wsl.MatchString(text) {
		return false
	}
	return hyperV.MatchString(a.Name) || hyperV.MatchString(a.Description) ||
		vmware.MatchString(text) || virtualBox.MatchString(text)
}

// Listen is one address the host serves on, with the subnet its peers must belong to.
type Listen struct {
	IP     net.IP
	Subnet net.IPNet
}

// ListenAddresses are the IPv4 addresses the host helper may listen on: adapters that are up, are
// the host side of a virtual switch, and have NO default gateway. A gateway means the adapter also
// carries the real network (a Hyper-V "External" switch shares the physical card), and the proof
// service must never be reachable from there. Physical adapters never qualify.
func ListenAddresses(adapters []Adapter) []Listen {
	found := []Listen{}
	for _, adapter := range adapters {
		if !adapter.Up || len(adapter.Gateways) > 0 || !IsVirtualSwitch(adapter) {
			continue
		}
		for _, address := range adapter.IPv4 {
			ip := address.IP.To4()
			if ip == nil || ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() {
				continue
			}
			found = append(found, Listen{IP: ip, Subnet: net.IPNet{IP: ip.Mask(address.Mask), Mask: address.Mask}})
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i].IP.String() < found[j].IP.String() })
	return found
}

// CandidateHosts are the addresses a VM asks, nearest first: its default gateways, then the first
// address (".1") of every local IPv4 subnet. A VM's own addresses are never asked.
func CandidateHosts(adapters []Adapter) []string {
	own := map[string]bool{}
	for _, adapter := range adapters {
		for _, address := range adapter.IPv4 {
			own[address.IP.String()] = true
		}
	}
	seen, hosts := map[string]bool{}, []string{}
	add := func(ip net.IP) {
		ip = ip.To4()
		if ip == nil || ip.IsLoopback() || ip.IsUnspecified() || own[ip.String()] || seen[ip.String()] {
			return
		}
		seen[ip.String()] = true
		hosts = append(hosts, ip.String())
	}
	for _, adapter := range adapters {
		if !adapter.Up {
			continue
		}
		for _, gateway := range adapter.Gateways {
			add(gateway)
		}
	}
	for _, adapter := range adapters {
		if !adapter.Up {
			continue
		}
		for _, address := range adapter.IPv4 {
			if first := firstHost(address); first != nil {
				add(first)
			}
		}
	}
	return hosts
}

// firstHost is the network address plus one, e.g. 172.20.80.1 for 172.20.80.5/20.
func firstHost(address net.IPNet) net.IP {
	ip := address.IP.To4()
	ones, bits := address.Mask.Size()
	if ip == nil || bits != 32 || ones >= 31 || ones == 0 {
		return nil
	}
	first := append(net.IP(nil), ip.Mask(address.Mask)...)
	first[3]++
	return first
}

// ListenerAddress formats ip with the service port.
func ListenerAddress(ip net.IP, port int) string {
	return net.JoinHostPort(ip.String(), strconv.Itoa(port))
}
