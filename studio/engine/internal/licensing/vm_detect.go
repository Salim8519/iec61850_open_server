package licensing

import (
	"regexp"
	"strings"
)

// A computer may use a host's license only when it is itself a virtual machine (owner's rule: a
// licensed PC shares its license with the virtual machines running on it, never with other PCs).
// It is one when its firmware says so: the SMBIOS system (type 1) or baseboard (type 2) maker or
// product names a hypervisor. Each hypervisor writes these strings into the firmware it emulates
// and Windows reads them from the raw table (the same one the machine ID comes from), so neither a
// rename nor a registry edit changes them. Real values:
//   - Hyper-V (also Windows Sandbox): "Microsoft Corporation" / "Virtual Machine"
//   - VMware Workstation/Player/ESXi: "VMware, Inc." / "VMware Virtual Platform" or "VMware7,1"
//   - VirtualBox: "innotek GmbH" / "VirtualBox", baseboard "Oracle Corporation" / "VirtualBox"
//   - QEMU/KVM, Xen, Parallels: "QEMU", "Xen" / "HVM domU", "Parallels Software International Inc."
var vmFirmware = regexp.MustCompile(`(?i)vmware|virtualbox|innotek|\bqemu\b|\bkvm\b|\bxen\b|parallels`)

// SMBIOSIsVirtualMachine reports whether a RawSMBIOSData blob is the firmware of a virtual machine.
func SMBIOSIsVirtualMachine(raw []byte) bool {
	for _, kind := range []byte{1, 2} {
		maker, product := smbiosMakerProduct(raw, kind)
		if vmFirmware.MatchString(maker + "\n" + product) {
			return true
		}
		// Microsoft also makes physical PCs (Surface), so only its product "Virtual Machine" counts.
		if strings.Contains(strings.ToLower(maker), "microsoft") &&
			strings.EqualFold(strings.Trim(product, asciiSpace), "Virtual Machine") {
			return true
		}
	}
	return false
}

// smbiosMakerProduct is the manufacturer (string number at offset 4) and product name (offset 5)
// of the first SMBIOS structure of kind (1 system, 2 baseboard), or "" each.
func smbiosMakerProduct(raw []byte, kind byte) (string, string) {
	data := smbiosData(raw)
	for pos := 0; pos+4 <= len(data); {
		current, length := data[pos], int(data[pos+1])
		if length < 4 || pos+length > len(data) {
			return "", ""
		}
		start, end := pos+length, pos+length
		for end+1 < len(data) && !(data[end] == 0 && data[end+1] == 0) {
			end++
		}
		if current == kind && length >= 6 {
			set := data[start:min(end, len(data))]
			return smbiosString(set, int(data[pos+4])), smbiosString(set, int(data[pos+5]))
		}
		if current == 127 {
			return "", ""
		}
		pos = end + 2
	}
	return "", ""
}
