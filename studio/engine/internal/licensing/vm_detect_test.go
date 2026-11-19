package licensing

import (
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// isVM is the IsVM seam of the tests whose computer plays a virtual machine.
func isVM() bool { return true }

// firmware is a raw SMBIOS table with a system (type 1) and a baseboard (type 2) structure, each
// with its manufacturer as string 1 and its product as string 2.
func firmware(maker, product, boardMaker, board string) []byte {
	system := structure(1, 27, maker, product)
	system[4], system[5] = 1, 2
	baseboard := structure(2, 15, boardMaker, board)
	baseboard[4], baseboard[5] = 1, 2
	return smbiosBlob(structure(0, 24, "Vendor", "1.0"), system, baseboard)
}

func TestTheFirmwareOfHyperVVMwareAndVirtualBoxGuestsIsAVirtualMachine(t *testing.T) {
	for name, blob := range map[string][]byte{
		"Hyper-V":           firmware("Microsoft Corporation", "Virtual Machine", "Microsoft Corporation", "Virtual Machine"),
		"VMware (old)":      firmware("VMware, Inc.", "VMware Virtual Platform", "Intel Corporation", "440BX Desktop Reference Platform"),
		"VMware (UEFI)":     firmware("VMware, Inc.", "VMware7,1", "Intel Corporation", "440BX Desktop Reference Platform"),
		"VirtualBox":        firmware("innotek GmbH", "VirtualBox", "Oracle Corporation", "VirtualBox"),
		"VirtualBox board":  firmware("To be filled by O.E.M.", "To be filled by O.E.M.", "Oracle Corporation", "VirtualBox"),
		"QEMU/KVM":          firmware("QEMU", "Standard PC (Q35 + ICH9, 2009)", "", ""),
		"Xen":               firmware("Xen", "HVM domU", "", ""),
		"Parallels Desktop": firmware("Parallels Software International Inc.", "Parallels Virtual Platform", "", ""),
	} {
		if !SMBIOSIsVirtualMachine(blob) {
			t.Errorf("%s is not recognised as a virtual machine", name)
		}
	}
}

func TestThePhysicalPCsFirmwareIsNotAVirtualMachine(t *testing.T) {
	for name, blob := range map[string][]byte{
		"Dell laptop":       firmware("Dell Inc.", "Latitude 7440", "Dell Inc.", "0WX4K2"),
		"Lenovo":            firmware("LENOVO", "21HD0012GE", "LENOVO", "21HD0012GE"),
		"Surface":           firmware("Microsoft Corporation", "Surface Pro 9", "Microsoft Corporation", "Surface Pro 9"),
		"self-built":        firmware("To be filled by O.E.M.", "To be filled by O.E.M.", "ASUSTeK COMPUTER INC.", "PRIME B550-PLUS"),
		"no strings at all": smbiosBlob(structure(1, 27), structure(2, 15)),
		"no table":          nil,
		"truncated table":   append(make([]byte, 8), 1, 27, 0, 0, 1),
	} {
		if SMBIOSIsVirtualMachine(blob) {
			t.Errorf("%s counts as a virtual machine", name)
		}
	}
}

// A physical PC never uses a host's license: it never asks one, RefreshVMHost refuses in plain
// words, and a vm_proof.json in its license home (copied there, or left from an earlier answer) is
// ignored. The same home on a virtual machine is licensed by that very proof.
func TestAComputerThatIsNotAVirtualMachineNeverAsksAHostAndIgnoresAStoredProof(t *testing.T) {
	rig := newVMRig(t)
	rig.untilLicensed() // a genuine proof for this machine ID is now in rig.vmHome
	var asked atomic.Int32
	on := func(vm bool) *Service {
		return New(Options{Home: rig.vmHome, DataDir: rig.vmHome, Enforce: true, Keys: rig.host.signer.keys(),
			HWID: func() (string, error) { return rig.vmHWID, nil }, Clock: func() float64 { return *rig.now },
			Debugger: func() bool { return false }, Logf: t.Logf, Registry: newHKRegistry(),
			HostKey: HostKeyOptions{VM: &VMOptions{IsVM: func() bool { return vm },
				Hosts: func() []string { asked.Add(1); return []string{rig.address} }}}})
	}
	physical := on(false)
	status, err := physical.Current(true)
	if err != nil || status.Licensed || status.Mode == ModeVMHost || status.Reason != NoLicense {
		t.Fatalf("a physical PC used the stored proof: %+v (%v)", status, err)
	}
	if err := physical.RefreshVMHost(); err == nil || err.Error() != NotAVMText {
		t.Fatalf("a physical PC was allowed to ask a host: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if asked.Load() != 0 {
		t.Fatalf("a physical PC asked a host %d times", asked.Load())
	}
	if !strings.Contains(NotAVMText, "not a virtual machine") {
		t.Fatalf("the refusal does not say why: %q", NotAVMText)
	}
	vm := on(true)
	if status, err := vm.Current(true); err != nil || !status.Licensed || status.Mode != ModeVMHost {
		t.Fatalf("a virtual machine lost its stored proof: %+v (%v)", status, err)
	}
	if err := vm.RefreshVMHost(); err != nil || asked.Load() == 0 {
		t.Fatalf("a virtual machine could not ask its host: %v (asked %d)", err, asked.Load())
	}
}
