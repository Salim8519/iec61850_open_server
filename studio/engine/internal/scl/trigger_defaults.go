package scl

import "studio-engine/internal/scl/xmldoc"

const (
	trgDchg = 1
	trgQchg = 2
	trgDupd = 4
)

// standardTriggers are the trigger options (TrgOp column) IEC 61850-7-3 Ed2 clause 7 gives
// the status and measured attributes of its common data classes (ADR-0114), keyed by
// functional constraint and attribute name. They are the same in every class that has the
// attribute: stVal/q of SPS, DPS, INS, ENS and the controls; general, dirGeneral, phsX, neut
// and dirX of ACT/ACD; cnt of SEC; actVal, frVal of BCR; hstVal of HST; mag, range of MV;
// cVal, rangeAng of CMV; valWTr of BSC/ISC; mxVal of APC/BAC; stSeld of the controls.
// Attributes the standard gives no trigger option (t, instMag, instCVal, subVal …) and other
// functional constraints (CF, DC, SP, SG, SE, OR, BL, EX) are not listed: nothing is added.
var standardTriggers = map[string]map[string]int{
	"ST": {
		"stVal": trgDchg, "q": trgQchg, "general": trgDchg, "dirGeneral": trgDchg,
		"phsA": trgDchg, "phsB": trgDchg, "phsC": trgDchg, "neut": trgDchg,
		"dirPhsA": trgDchg, "dirPhsB": trgDchg, "dirPhsC": trgDchg, "dirNeut": trgDchg,
		"cnt": trgDchg, "actVal": trgDchg | trgDupd, "frVal": trgDupd, "hstVal": trgDchg | trgDupd,
		"valWTr": trgDchg, "stSeld": trgDchg,
	},
	"MX": {
		"q": trgQchg, "mag": trgDchg | trgDupd, "cVal": trgDchg | trgDupd, "range": trgDchg,
		"rangeAng": trgDchg, "mxVal": trgDchg,
	},
}

// hasTriggerOption reports whether the SCL states any trigger option on the element.
func hasTriggerOption(e *xmldoc.Element) bool {
	for _, key := range []string{"dchg", "qchg", "dupd"} {
		if _, ok := e.Get(key); ok {
			return true
		}
	}
	return false
}

// defaultTriggers returns the 7-3 trigger options of a DA whose template states none, or 0.
func defaultTriggers(e *xmldoc.Element, name, fc string) int {
	if !adr0114() || e.Tag != "DA" || hasTriggerOption(e) {
		return 0
	}
	return standardTriggers[fc][name]
}
