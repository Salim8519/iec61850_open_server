package scl

import "strconv"

const sclNamespace = "http://www.iec.ch/61850/2003/SCL"

// MakeTemplate is templates.make_template(name): the documented laboratory IED as the
// exact bytes ElementTree writes.
func MakeTemplate(name string) ([]byte, error) {
	if _, err := Identifier(name, true, "IED name"); err != nil {
		return nil, err
	}
	root := &etNode{tag: "SCL", attrs: [][2]string{{"xmlns", sclNamespace}, {"version", "2007"},
		{"revision", "B"}, {"release", "4"}}}
	root.add("Header", "id", name, "version", "1", "revision", "0")
	ied := root.add("IED", "name", name, "manufacturer", "ScadaBench", "type", "Lab bay controller",
		"configVersion", "1", "desc", "Simulation template; not a manufacturer relay model")
	server := ied.add("AccessPoint", "name", "AP1").add("Server")
	server.add("Authentication", "none", "true")
	ld := server.add("LDevice", "inst", "CTRL")
	ln0 := ld.add("LN0", "lnClass", "LLN0", "inst", "", "lnType", "StudioLLN0")
	ds := ln0.add("DataSet", "name", "Events")
	for _, m := range [][5]string{{"XCBR", "1", "Pos", "stVal", "ST"}, {"XSWI", "1", "Pos", "stVal", "ST"},
		{"XSWI", "2", "Pos", "stVal", "ST"}, {"GGIO", "1", "AnIn1", "mag.f", "MX"}, {"GGIO", "1", "SPCSO1", "stVal", "ST"}} {
		ds.add("FCDA", "ldInst", "CTRL", "lnClass", m[0], "lnInst", m[1], "doName", m[2], "daName", m[3], "fc", m[4])
	}
	for _, buffered := range []bool{false, true} {
		rcName, rptID, flag := "EventsRCB", name+"_RP", "false"
		if buffered {
			rcName, rptID, flag = "Buffered", name+"_BR", "true"
		}
		rc := ln0.add("ReportControl", "name", rcName, "rptID", rptID, "datSet", "Events", "confRev", "1",
			"buffered", flag, "indexed", "false", "bufTime", "0", "intgPd", "1000")
		rc.add("TrgOps", "dchg", "true", "qchg", "true", "dupd", "false", "period", "true", "gi", "true")
		rc.add("OptFields", "seqNum", "true", "timeStamp", "true", "reasonCode", "true", "dataSet", "true",
			"dataRef", "false", "bufOvfl", "true", "entryID", "true", "configRef", "true")
	}
	ld.add("LN", "lnClass", "LPHD", "inst", "1", "lnType", "StudioLPHD")
	for _, l := range [][3]string{{"XCBR", "1", "StudioCB"}, {"XSWI", "1", "StudioSwitch"},
		{"XSWI", "2", "StudioSwitch"}, {"GGIO", "1", "StudioAnalog"}} {
		ln := ld.add("LN", "lnClass", l[0], "inst", l[1], "lnType", l[2])
		if l[0] == "XCBR" || l[0] == "XSWI" {
			ln.add("DOI", "name", "Pos").add("DAI", "name", "stVal").add("Val").text = "1"
		}
	}
	templateTypes(root.add("DataTypeTemplates"))
	return root.tostring(), nil
}

// da is templates.da: DA when fc is given, else BDA, with an optional Val.
func da(parent *etNode, name, kind, fc, value string, extra ...string) {
	tag, attrs := "BDA", []string{"name", name, "bType", kind}
	if fc != "" {
		tag, attrs = "DA", append(attrs, "fc", fc)
	}
	item := parent.add(tag, append(attrs, extra...)...)
	if value != "" {
		item.add("Val").text = value
	}
}

func templateTypes(types *etNode) {
	for _, ln := range []struct {
		id, class string
		objects   [][2]string
	}{
		{"StudioLLN0", "LLN0", [][2]string{{"Beh", "StudioStatus"}, {"Health", "StudioStatus"}, {"NamPlt", "StudioName"}}},
		{"StudioLPHD", "LPHD", [][2]string{{"PhyHealth", "StudioStatus"}}},
		{"StudioCB", "XCBR", [][2]string{{"Pos", "StudioDPC"}, {"Loc", "StudioBoolean"}}},
		{"StudioSwitch", "XSWI", [][2]string{{"Pos", "StudioDPC"}, {"Loc", "StudioBoolean"}}},
		{"StudioAnalog", "GGIO", [][2]string{{"AnIn1", "StudioMV"}, {"AnIn2", "StudioMV"}, {"SPCSO1", "StudioSPC"}}},
	} {
		node := types.add("LNodeType", "id", ln.id, "lnClass", ln.class)
		for _, object := range ln.objects {
			node.add("DO", "name", object[0], "type", object[1])
		}
	}
	status := types.add("DOType", "id", "StudioStatus", "cdc", "INS")
	da(status, "stVal", "INT32", "ST", "1", "dchg", "true")
	statusQT(status, "ST")
	boolean := types.add("DOType", "id", "StudioBoolean", "cdc", "SPS")
	da(boolean, "stVal", "BOOLEAN", "ST", "false", "dchg", "true")
	statusQT(boolean, "ST")
	plate := types.add("DOType", "id", "StudioName", "cdc", "LPL")
	for _, field := range []string{"vendor", "swRev", "d"} {
		da(plate, field, "VisString255", "DC", "ScadaBench")
	}
	controlType(types.add("DOType", "id", "StudioDPC", "cdc", "DPC"), "Dbpos", "")
	controlType(types.add("DOType", "id", "StudioSPC", "cdc", "SPC"), "BOOLEAN", "false")
	analog := types.add("DOType", "id", "StudioMV", "cdc", "MV")
	da(analog, "mag", "Struct", "MX", "", "type", "StudioMagnitude", "dchg", "true")
	statusQT(analog, "MX")
	da(types.add("DAType", "id", "StudioMagnitude"), "f", "FLOAT32", "", "0")
	operate := types.add("DAType", "id", "StudioOperate")
	da(operate, "ctlVal", "BOOLEAN", "", "")
	da(operate, "origin", "Struct", "", "", "type", "StudioOrigin")
	da(operate, "ctlNum", "INT8U", "", "")
	da(operate, "T", "Timestamp", "", "")
	da(operate, "Test", "BOOLEAN", "", "")
	da(operate, "Check", "Check", "", "")
	cancel := types.add("DAType", "id", "StudioCancel")
	da(cancel, "ctlVal", "BOOLEAN", "", "")
	da(cancel, "origin", "Struct", "", "", "type", "StudioOrigin")
	da(cancel, "ctlNum", "INT8U", "", "")
	da(cancel, "T", "Timestamp", "", "")
	da(cancel, "Test", "BOOLEAN", "", "")
	origin := types.add("DAType", "id", "StudioOrigin")
	da(origin, "orCat", "Enum", "", "", "type", "StudioOriginCategory")
	da(origin, "orIdent", "Octet64", "", "")
	for _, enum := range []struct {
		id     string
		values []string
	}{
		{"StudioControlModel", []string{"status-only", "direct-normal", "sbo-normal", "direct-enhanced", "sbo-enhanced"}},
		{"StudioSboClass", []string{"operate-once", "operate-many"}},
		{"StudioOriginCategory", []string{"not-supported", "bay-control", "station-control", "remote-control"}},
	} {
		node := types.add("EnumType", "id", enum.id)
		for index, label := range enum.values {
			node.add("EnumVal", "ord", strconv.Itoa(index)).text = label
		}
	}
}

func statusQT(parent *etNode, fc string) {
	da(parent, "q", "Quality", fc, "", "qchg", "true")
	da(parent, "t", "Timestamp", fc, "")
}

func controlType(node *etNode, status, initial string) {
	da(node, "stVal", status, "ST", initial, "dchg", "true")
	statusQT(node, "ST")
	da(node, "ctlModel", "Enum", "CF", "3", "type", "StudioControlModel")
	da(node, "sboTimeout", "INT32U", "CF", "10000")
	da(node, "sboClass", "Enum", "CF", "0", "type", "StudioSboClass")
	da(node, "SBO", "VisString129", "CO", "")
	da(node, "Oper", "Struct", "CO", "", "type", "StudioOperate")
	da(node, "SBOw", "Struct", "CO", "", "type", "StudioOperate")
	da(node, "Cancel", "Struct", "CO", "", "type", "StudioCancel")
}
