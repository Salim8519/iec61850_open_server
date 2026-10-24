package scl

import (
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/pyrules"
	"studio-engine/internal/scl/xmldoc"
)

// blockInventory is goose_inventory.block_inventory: every GSEControl of the document,
// sendable or not, keyed (ied, ldInst, cbName) like publisherCatalogue. Each entry's
// control is the block with "fcdas" (attribute dicts); its fcdas are the FCDA elements.
func blockInventory(root *xmldoc.Element) *catalogue {
	sendable := publisherCatalogue(root)
	blocks := &catalogue{index: map[[3]string]int{}}
	for _, ied := range root.FindAll("IED") {
		name := ied.GetOr("name", "")
		comm, _ := readCommunication(root, name)
		addresses := map[[2]string]*pyjson.Object{}
		for _, item := range comm.gse {
			addresses[[2]string{get(item, "ld_inst").(string), get(item, "cb_name").(string)}] = item
		}
		for _, ld := range stationDevices(root, ied) {
			ln0 := ld.Find("LN0")
			if ln0 == nil {
				continue
			}
			inst := ld.GetOr("inst", "")
			domain := ld.GetOr("ldName", "")
			if domain == "" {
				domain = name + inst
			}
			datasets := map[string]*xmldoc.Element{} // a dict: the last dataset of a name wins
			for _, item := range ln0.FindAll("DataSet") {
				datasets[item.GetOr("name", "")] = item
			}
			for _, element := range ln0.FindAll("GSEControl") {
				key := [3]string{name, inst, element.GetOr("name", "")}
				_, known := sendable.index[key]
				blocks.put(inventoryBlock(element, key, domain, datasets, addresses[[2]string{inst, key[2]}], known))
			}
		}
	}
	return blocks
}

func inventoryBlock(element *xmldoc.Element, key [3]string, domain string, datasets map[string]*xmldoc.Element,
	address *pyjson.Object, sendable bool) *publisher {
	dataset := element.GetOr("datSet", "")
	var fcdas []*xmldoc.Element
	attributes := []any{}
	if declared := datasets[dataset]; declared != nil {
		fcdas = declared.FindAll("FCDA")
		for _, fcda := range fcdas {
			attributes = append(attributes, attrib(fcda))
		}
	}
	reference := domain + "/LLN0$GO$" + key[2]
	goID := element.GetOr("appID", "")
	if goID == "" {
		goID = reference
	}
	datasetRef := ""
	if dataset != "" {
		datasetRef = domain + "/LLN0$" + dataset
	}
	var confRev, appID, vlanID any
	if value, err := pyrules.IntBig(element.GetOr("confRev", "0"), 10); err == nil {
		confRev = intValue(value)
	}
	mac := ""
	if address != nil {
		appID, mac, vlanID = get(address, "app_id"), get(address, "mac").(string), get(address, "vlan_id")
	}
	block := pyjson.NewObject().Set("publisher", key[0]).Set("ld_inst", key[1]).Set("name", key[2]).
		Set("gocb_ref", reference).Set("go_id", goID).Set("dataset", dataset).Set("dataset_ref", datasetRef).
		Set("conf_rev", confRev).Set("app_id", appID).Set("mac", mac).Set("vlan_id", vlanID).
		Set("status", blockStatus(element, datasets, address, sendable)).Set("label", geLabel(element)).
		Set("vendor", vendorAttributes(element)).Set("fcdas", attributes)
	return &publisher{key: key, control: block, fcdas: fcdas}
}

// blockStatus is goose_inventory._status.
func blockStatus(element *xmldoc.Element, datasets map[string]*xmldoc.Element, address *pyjson.Object,
	sendable bool) string {
	switch kind, ok := element.Get("type"); {
	case sendable:
		return "sendable"
	case ok && kind != "GOOSE":
		return "not_goose"
	case geFunction(element) == "disabled":
		return "disabled_in_file"
	case datasets[element.GetOr("datSet", "")] == nil:
		return "no_dataset"
	case address == nil:
		return "no_address"
	}
	return "not_sendable"
}
