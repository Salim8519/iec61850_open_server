//go:build !windows

package netcfg

// RawAdapters is ip_helper.adapters off Windows: no adapters.
func RawAdapters() ([]any, error) { return []any{}, nil }

// AddressesOn is ip_helper.addresses_on off Windows: no addresses.
func AddressesOn(adapterGUID string) (map[string]bool, error) { return map[string]bool{}, nil }
