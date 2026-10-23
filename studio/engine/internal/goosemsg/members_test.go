package goosemsg

import (
	"reflect"
	"testing"
)

func refsOf(leaves []Leaf) []string {
	refs := []string{}
	for _, leaf := range leaves {
		refs = append(refs, leaf.Ref)
	}
	return refs
}

// A whole data object member (FCD) covers only the attributes with its functional constraint;
// an attribute member (FCDA) only itself. Prefixed nodes and other logical devices (ldName) work.
func TestMembersExpandWithTheirFunctionalConstraint(t *testing.T) {
	block := Build(publisher()).Block(gocb1)
	if block == nil || len(block.Members) != len(ds1Members) {
		t.Fatalf("block %+v", block)
	}
	want := [][]string{
		{"PUBLD0/XCBR1.Pos.q", "PUBLD0/XCBR1.Pos.stVal", "PUBLD0/XCBR1.Pos.t"}, // never ctlModel CF, d DC, subVal SV, PosX
		{"PUBLD0/QA1XSWI1.Pos.stVal"},
		{"PUBLD0/GGIO1.Ind1.stVal"},
		{"PUBLD0/GGIO1.Ind2.stVal"},
		{"PUBLD0/MMXU1.TotW.mag.f"},
		{"BAYCTRL/CSWI1.Pos.stVal"},
		{"PUBLD0/XCBR1.Pos.subVal"}, // the same object with SV covers only its SV attribute
		{"PUBLD0/GGIO1.Ind3.stVal"},
		{"PUBLD0/GGIO1.Ind1.q"},
	}
	for index, member := range block.Members {
		if got := refsOf(member.Leaves); !reflect.DeepEqual(got, want[index]) {
			t.Errorf("member %d %s: %v, want %v", member.Position, member.Text, got, want[index])
		}
	}
	texts := []string{block.Members[0].Text, block.Members[1].Text, block.Members[5].Text, block.Members[6].Text}
	if want := []string{"XCBR1.Pos [ST]", "QA1XSWI1.Pos.stVal [ST]", "BAYCTRL/CSWI1.Pos.stVal [ST]", "XCBR1.Pos [SV]"}; !reflect.DeepEqual(texts, want) {
		t.Fatalf("member texts %v, want %v", texts, want)
	}
	if _, found := block.Leaf("PUBLD0/XCBR1.Pos.ctlModel"); found {
		t.Fatal("Pos.ctlModel [CF] is not a member of an ST data object")
	}
	if got := Build(publisher()).BlocksOf("PUBLD0/GGIO1.Ind1.stVal"); !reflect.DeepEqual(got, []string{gocb1}) {
		t.Fatalf("leaf index %v", got)
	}
}

func TestMemberFunctionalConstraintAndAlignment(t *testing.T) {
	for entry, want := range map[string]string{"XCBR1$ST$Pos": "ST", "CTRL/CSWI1$CO$Pos$Oper": "CO",
		"MMXU1$MX$TotW$mag$f": "MX", "GGIO1$ST$Arr 2 stVal": "ST", "broken": ""} {
		if got := memberFC(entry); got != want {
			t.Errorf("memberFC(%q) = %q, want %q", entry, got, want)
		}
	}
	other := Build(publisher()).Block("PUBLD0/LLN0$GO$GoCB2")
	if other == nil || len(other.Members) != 0 {
		t.Fatalf("a dataset without member references gives a block without members, got %+v", other)
	}
	if got := Expand("PUBLD0/XCBR1.Pos", "", indexSignals(nil)); got != nil {
		t.Fatalf("no FC, no leaves: %v", got)
	}
}

// For builds once per stored record; a replaced record is read again.
func TestCatalogIsBuiltOncePerRecord(t *testing.T) {
	ied := publisher()
	first := For(ied)
	if For(ied) != first {
		t.Fatal("the same record must give the same catalog")
	}
	if For(publisher()) == first {
		t.Fatal("a replaced record must be read again")
	}
}

func BenchmarkCatalogBuild(b *testing.B) {
	ied := publisher()
	for n := 0; n < b.N; n++ {
		Build(ied)
	}
}

func BenchmarkCatalogFor(b *testing.B) {
	ied := publisher()
	For(ied)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		For(ied)
	}
}
