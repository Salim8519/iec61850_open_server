package goosemsg

import (
	"studio-engine/internal/pyjson"
)

// The test IED "PUB": logical device LD0 (domain PUBLD0, no ldName) and CTRL (ldName BAYCTRL).
// GoCB1 sends DS1 (members below, in order); GoCB2's dataset has no member references (a record
// imported before change 0094).

const gocb1 = "PUBLD0/LLN0$GO$GoCB1"

func o(pairs ...any) *pyjson.Object {
	result := pyjson.NewObject()
	for index := 0; index < len(pairs); index += 2 {
		result.Set(pairs[index].(string), pairs[index+1])
	}
	return result
}

func signal(ref, kind, fc string, writable bool) any {
	return o("ref", ref, "fc", fc, "kind", kind, "value", nil, "description", "desc of "+ref, "writable", writable)
}

var ds1Members = []any{"XCBR1$ST$Pos", "QA1XSWI1$ST$Pos$stVal", "GGIO1$ST$Ind1$stVal", "GGIO1$ST$Ind2$stVal",
	"MMXU1$MX$TotW$mag$f", "CTRL/CSWI1$ST$Pos$stVal", "XCBR1$SV$Pos", "GGIO1$ST$Ind3$stVal", "GGIO1$ST$Ind1$q"}

var ds1Refs = []any{"PUBLD0/XCBR1.Pos", "PUBLD0/QA1XSWI1.Pos.stVal", "PUBLD0/GGIO1.Ind1.stVal",
	"PUBLD0/GGIO1.Ind2.stVal", "PUBLD0/MMXU1.TotW.mag.f", "BAYCTRL/CSWI1.Pos.stVal", "PUBLD0/XCBR1.Pos",
	"PUBLD0/GGIO1.Ind3.stVal", "PUBLD0/GGIO1.Ind1.q"}

func publisher() *pyjson.Object {
	signals := []any{
		signal("PUBLD0/XCBR1.Pos.stVal", "Dbpos", "ST", true), signal("PUBLD0/XCBR1.Pos.q", "Quality", "ST", true),
		signal("PUBLD0/XCBR1.Pos.t", "Timestamp", "ST", true), signal("PUBLD0/XCBR1.Pos.ctlModel", "Enum", "CF", true),
		signal("PUBLD0/XCBR1.Pos.d", "VisString255", "DC", true), signal("PUBLD0/XCBR1.Pos.subVal", "Dbpos", "SV", false),
		signal("PUBLD0/XCBR1.PosX.stVal", "Dbpos", "ST", true), signal("PUBLD0/QA1XSWI1.Pos.stVal", "Dbpos", "ST", true),
		signal("PUBLD0/GGIO1.Ind1.stVal", "BOOLEAN", "ST", true), signal("PUBLD0/GGIO1.Ind1.q", "Quality", "ST", true),
		signal("PUBLD0/GGIO1.Ind2.stVal", "BOOLEAN", "ST", true), signal("PUBLD0/GGIO1.Ind3.stVal", "BOOLEAN", "ST", false),
		signal("PUBLD0/MMXU1.TotW.mag.f", "FLOAT32", "MX", true), signal("BAYCTRL/CSWI1.Pos.stVal", "Dbpos", "ST", true),
		signal("PUBLD0/GGIO1.Cnt.stVal", "INT32", "ST", true),
	}
	ds1 := o("name", "DS1", "members", ds1Members, "ref", "PUBLD0/LLN0$DS1", "fcda_refs", ds1Refs)
	ds2 := o("name", "DS2", "members", []any{"GGIO1$ST$Cnt$stVal"}, "ref", "PUBLD0/LLN0$DS2")
	control := func(name, dataset string, members []any) any {
		return o("name", name, "ld_inst", "LD0", "gocb_ref", "PUBLD0/LLN0$GO$"+name, "go_id", "PUB/"+name,
			"dataset", dataset, "dataset_ref", "PUBLD0/LLN0$"+dataset, "conf_rev", int64(1), "app_id", int64(4097),
			"members", members)
	}
	return o("id", "p", "name", "PUB", "role", "simulated", "signals", signals, "datasets", []any{ds1, ds2},
		"goose", o("publishes", []any{control("GoCB1", "DS1", ds1Members),
			control("GoCB2", "DS2", []any{"GGIO1$ST$Cnt$stVal"})}, "subscribes", []any{}))
}
