package hostkey

import (
	"net"
	"reflect"
	"testing"
)

func cidr(text string) net.IPNet {
	ip, network, err := net.ParseCIDR(text)
	if err != nil {
		panic(err)
	}
	return net.IPNet{IP: ip, Mask: network.Mask}
}

func adapter(name, description string, up bool, gateway string, addresses ...string) Adapter {
	a := Adapter{Name: name, Description: description, Up: up}
	for _, address := range addresses {
		a.IPv4 = append(a.IPv4, cidr(address))
	}
	if gateway != "" {
		a.Gateways = []net.IP{net.ParseIP(gateway)}
	}
	return a
}

func listenIPs(adapters []Adapter) []string {
	found := []string{}
	for _, listen := range ListenAddresses(adapters) {
		found = append(found, listen.IP.String())
	}
	return found
}

func TestOnlyTheHostSideOfVirtualSwitchesIsEverServed(t *testing.T) {
	adapters := []Adapter{
		adapter("Ethernet", "Intel(R) Ethernet Connection", true, "192.168.1.1", "192.168.1.20/24"),
		adapter("Wi-Fi", "Intel(R) Wi-Fi 6 AX201", true, "192.168.1.1", "192.168.1.21/24"),
		adapter("vEthernet (Default Switch)", "Hyper-V Virtual Ethernet Adapter", true, "", "172.20.80.1/20"),
		adapter("vEthernet (Lab)", "Hyper-V Virtual Ethernet Adapter #2", true, "", "10.0.5.1/24"),
		adapter("VMware Network Adapter VMnet1", "VMware Virtual Ethernet Adapter for VMnet1", true, "", "192.168.56.1/24"),
		adapter("VMware Network Adapter VMnet8", "VMware Virtual Ethernet Adapter for VMnet8", true, "", "192.168.74.1/24"),
		adapter("VirtualBox Host-Only Network", "VirtualBox Host-Only Ethernet Adapter", true, "", "192.168.99.1/24"),
		adapter("Loopback Pseudo-Interface 1", "Software Loopback Interface 1", true, "", "127.0.0.1/8"),
	}
	want := []string{"10.0.5.1", "172.20.80.1", "192.168.56.1", "192.168.74.1", "192.168.99.1"}
	if got := listenIPs(adapters); !reflect.DeepEqual(got, want) {
		t.Fatalf("served on %v, want %v", got, want)
	}
}

func TestNoPhysicalOrLookAlikeAdapterIsServed(t *testing.T) {
	for _, a := range []Adapter{
		adapter("Ethernet", "Realtek PCIe GbE Family Controller", true, "192.168.1.1", "192.168.1.20/24"),
		adapter("Ethernet 2", "Realtek PCIe GbE Family Controller", true, "", "192.168.1.30/24"), // no gateway, still physical
		adapter("VMware Network Adapter VMnet2", "VMware Virtual Ethernet Adapter for VMnet2", true, "", "192.168.2.1/24"),
		adapter("VMware Network Adapter VMnet10", "VMware Virtual Ethernet Adapter for VMnet10", true, "", "192.168.3.1/24"),
		adapter("vEthernet (WSL)", "Hyper-V Virtual Ethernet Adapter #3", true, "", "172.30.0.1/20"),
		adapter("Bluetooth Network Connection", "Bluetooth Device (Personal Area Network)", true, "", "169.254.5.5/16"),
		adapter("OpenVPN TAP", "TAP-Windows Adapter V9", true, "", "10.8.0.2/24"),
	} {
		if got := ListenAddresses([]Adapter{a}); len(got) != 0 {
			t.Errorf("%q was served on %v", a.Name, got)
		}
	}
}

func TestAHyperVExternalSwitchThatCarriesTheRealNetworkIsNeverServed(t *testing.T) {
	external := adapter("vEthernet (External)", "Hyper-V Virtual Ethernet Adapter #4", true, "192.168.1.1", "192.168.1.20/24")
	if got := ListenAddresses([]Adapter{external}); len(got) != 0 {
		t.Fatalf("an adapter with a default gateway (the LAN) was served on %v", got)
	}
}

func TestAnAdapterThatIsDownOrHasNoIPv4AddressIsSkipped(t *testing.T) {
	down := adapter("vEthernet (Default Switch)", "", false, "", "172.20.80.1/20")
	none := adapter("vEthernet (Other)", "", true, "")
	loopbackOnly := adapter("vEthernet (Odd)", "", true, "", "127.0.0.1/8")
	if got := ListenAddresses([]Adapter{down, none, loopbackOnly}); len(got) != 0 {
		t.Fatalf("served on %v", got)
	}
}

func TestEachServedAddressNamesTheSubnetOfItsSwitch(t *testing.T) {
	listens := ListenAddresses([]Adapter{adapter("vEthernet (Default Switch)", "", true, "", "172.20.80.1/20")})
	if len(listens) != 1 || listens[0].Subnet.String() != "172.20.80.0/20" {
		t.Fatalf("subnet of the served address: %+v", listens)
	}
	if !listens[0].Subnet.Contains(net.ParseIP("172.20.81.7")) || listens[0].Subnet.Contains(net.ParseIP("192.168.1.9")) {
		t.Fatal("the subnet does not tell the switch's peers from other computers")
	}
	if got := ListenerAddress(listens[0].IP, Port); got != "172.20.80.1:47862" {
		t.Fatalf("listener address %q", got)
	}
}

func TestAVMAsksItsGatewayFirstThenTheFirstAddressOfEverySubnetButNeverItself(t *testing.T) {
	vm := []Adapter{
		adapter("Ethernet", "Microsoft Hyper-V Network Adapter", true, "172.20.80.1", "172.20.80.5/20"),
		adapter("Ethernet 2", "second card", true, "", "10.1.2.7/24"),
		adapter("Down", "", false, "10.9.9.1", "10.9.9.9/24"),
	}
	want := []string{"172.20.80.1", "10.1.2.1"}
	if got := CandidateHosts(vm); !reflect.DeepEqual(got, want) {
		t.Fatalf("candidates %v, want %v", got, want)
	}
	// The host of a subnet (its .1) never asks itself.
	host := []Adapter{adapter("vEthernet (Default Switch)", "", true, "", "172.20.80.1/20")}
	if got := CandidateHosts(host); len(got) != 0 {
		t.Fatalf("a computer asked itself: %v", got)
	}
	if got := CandidateHosts([]Adapter{adapter("Tiny", "", true, "", "10.0.0.5/31"), adapter("Host", "", true, "", "10.0.1.5/32")}); len(got) != 0 {
		t.Fatalf("a /31 or /32 has no .1: %v", got)
	}
}
