package signalindex

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// kindVectors is the one IEC 61850 kind table shared with the frontend's kindOf61850
// (studio/tests/frontend/pointKindVectors.test.ts reads the same file).
func kindVectors(t *testing.T) [][4]string {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "tests", "fixtures", "point_kinds", "iec61850.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Cases [][4]string `json:"cases"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatal(err)
	}
	return file.Cases
}

// ADR-0113: every standard functional constraint × basic type, and the data classes that refine
// them, get the kind of the shared table; UN is left for vendor types only.
func TestPointKindMatchesTheSharedTable(t *testing.T) {
	cases := kindVectors(t)
	fcs, types := map[string]bool{}, map[string]bool{}
	for _, c := range cases {
		if got := pointKind(c[0], c[1], c[2]); got != c[3] {
			t.Errorf("pointKind(%q, %q, %q) = %s, the table says %s", c[0], c[1], c[2], got, c[3])
		}
		if c[3] == "UN" && (functionalConstraint(c[0]) && standardType(c[1])) {
			t.Errorf("%v: a standard FC and type must never be unknown", c)
		}
		fcs[c[0]], types[c[1]] = true, true
	}
	for _, fc := range []string{"ST", "MX", "SP", "SV", "CF", "DC", "SG", "SE", "SR", "OR", "BL", "EX", "CO", "US",
		"MS", "RP", "BR", "LG", "GO"} {
		if !fcs[fc] {
			t.Errorf("the table has no case for FC %s", fc)
		}
	}
	for _, basic := range standardTypes {
		if !types[basic] {
			t.Errorf("the table has no case for basic type %s", basic)
		}
	}
	if kindCodes["AT"] != kindAT || kindCodes["UN"] != kindUN {
		t.Fatal("kind codes moved")
	}
}

// standardTypes are the leaf basic types of IEC 61850-6 (tPredefinedBasicTypeEnum without Struct).
var standardTypes = []string{"BOOLEAN", "INT8", "INT16", "INT24", "INT32", "INT64", "INT128", "INT8U", "INT16U",
	"INT24U", "INT32U", "FLOAT32", "FLOAT64", "Enum", "Dbpos", "Tcmd", "Quality", "Timestamp", "VisString32",
	"VisString64", "VisString65", "VisString129", "VisString255", "Octet64", "Unicode255", "EntryTime", "Check",
	"ObjRef", "Currency", "PhyComAddr", "TrgOps", "OptFlds", "SvOptFlds", "LogOptFlds", "EntryID", "Octet6", "Octet16"}

func standardType(basic string) bool {
	for _, known := range standardTypes {
		if known == basic {
			return true
		}
	}
	return false
}

func functionalConstraint(fc string) bool {
	return settingFCs[fc] || describingFC[fc] || serviceFCs[fc] || fc == "ST" || fc == "MX" || fc == "OR" ||
		fc == "SV" || fc == "BL" || fc == "CO"
}
