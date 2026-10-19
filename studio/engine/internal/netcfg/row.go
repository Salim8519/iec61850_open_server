package netcfg

import "studio-engine/internal/pyjson"

// PlanIED is what the address planner reads of one simulated IED.
type PlanIED struct {
	ID        string
	Name      string
	SCLIP     string // scl_address.ip ("" when the SCD has none)
	SCLSubnet string // scl_address.subnet ("" means 255.255.255.0)
	Port      int    // ied.get("port", 102)
}

// Alternative is a free address on the adapter's own network offered for an unusable row.
type Alternative struct {
	IP     string
	Prefix *int
}

// Row is one proposed lab address with its status and a plain hint (address_plan rows).
type Row struct {
	IEDID       string
	Name        string
	SCDIP       string
	Proposed    string
	Prefix      *int
	Status      string
	Hint        string
	Alternative *Alternative
}

func intPointer(value int) *int { return &value }

func optionalInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func (r Row) prefixValue() any { return optionalInt(r.Prefix) }

// usable is status in {"ok", "already_on_adapter"}.
func (r Row) usable() bool { return r.Status == "ok" || r.Status == "already_on_adapter" }

// Object returns the row in Python's key order.
func (r Row) Object() *pyjson.Object {
	object := pyjson.NewObject().Set("ied_id", r.IEDID).Set("name", r.Name).Set("scd_ip", r.SCDIP).
		Set("proposed", r.Proposed).Set("prefix", r.prefixValue()).Set("status", r.Status).Set("hint", r.Hint)
	if r.Alternative != nil {
		object.Set("alternative", pyjson.NewObject().Set("ip", r.Alternative.IP).
			Set("prefix", optionalInt(r.Alternative.Prefix)))
	}
	return object
}

// RowsObject returns the rows as JSON values.
func RowsObject(rows []Row) []any {
	items := make([]any, len(rows))
	for index, row := range rows {
		items[index] = row.Object()
	}
	return items
}
