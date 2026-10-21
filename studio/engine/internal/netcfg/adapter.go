package netcfg

import "studio-engine/internal/pyjson"

// Address is one IPv4 address on an adapter, as adapters.normalize writes it.
type Address struct {
	IP           string
	Prefix       int
	SkipAsSource bool
	Origin       string
}

// Adapter is one normalized network adapter (adapters.normalize). Index is nil when the
// source gave none (Python None).
type Adapter struct {
	GUID         string
	Name         string
	Description  string
	Status       string
	Index        *int64
	MAC          string
	Speed        string
	Kind         string
	DefaultRoute bool
	Addresses    []Address
}

// AdapterSource lists the adapters (the real one reads Windows; tests use a fake).
type AdapterSource interface {
	Adapters() ([]Adapter, error)
}

// AdapterFunc adapts a plain function to AdapterSource.
type AdapterFunc func() ([]Adapter, error)

// Adapters calls the function.
func (f AdapterFunc) Adapters() ([]Adapter, error) { return f() }

// Object returns the adapter in the key order Python's dictionary has.
func (a Adapter) Object() *pyjson.Object {
	addresses := make([]any, len(a.Addresses))
	for position, item := range a.Addresses {
		addresses[position] = pyjson.NewObject().Set("ip", item.IP).Set("prefix", item.Prefix).
			Set("skip_as_source", item.SkipAsSource).Set("origin", item.Origin)
	}
	var index any
	if a.Index != nil {
		index = *a.Index
	}
	return pyjson.NewObject().Set("guid", a.GUID).Set("name", a.Name).Set("description", a.Description).
		Set("status", a.Status).Set("index", index).Set("mac", a.MAC).Set("speed", a.Speed).
		Set("kind", a.Kind).Set("default_route", a.DefaultRoute).Set("addresses", addresses)
}

// AdaptersObject returns the list as JSON values.
func AdaptersObject(adapters []Adapter) []any {
	items := make([]any, len(adapters))
	for position, adapter := range adapters {
		items[position] = adapter.Object()
	}
	return items
}

// ips returns the set of addresses on the adapter.
func (a Adapter) ips() map[string]bool {
	found := map[string]bool{}
	for _, item := range a.Addresses {
		found[item.IP] = true
	}
	return found
}

// indexValue is adapter["index"] as a JSON value (None when unknown).
func (a Adapter) indexValue() any {
	if a.Index == nil {
		return nil
	}
	return *a.Index
}
