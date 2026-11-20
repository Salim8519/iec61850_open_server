//go:build !windows

package netcfg

// RawAdapters is ip_helper.adapters off Windows: no adapters.
func RawAdapters() ([]any, error) { return []any{}, nil }

// AddressesOn is ip_helper.addresses_on off Windows: no addresses.
func AddressesOn(adapterGUID string) (map[string]bool, error) { return map[string]bool{}, nil }

// HasAddress is true off Windows: nothing can take an address away there.
func HasAddress(string) (bool, error) { return true, nil }

// DHCPEnabled is false off Windows: no adapters.
func DHCPEnabled(adapterGUID string) (bool, error) { return false, nil }
