package livedom

import (
	"testing"

	"studio-engine/internal/acquisition/goldentest"
	"studio-engine/internal/acquisition/pyv"
)

func TestCompareMatchesPython(t *testing.T) {
	for _, c := range goldentest.Load(t, "livedom_compare.json") {
		goldentest.Text(t, c, dumps(t, Compare(c.Obj(0), c.Obj(1))))
	}
}

func TestChooseAccessMatchesPython(t *testing.T) {
	for _, c := range goldentest.Load(t, "livedom_access.json") {
		switch pyv.Str(c.Args[0]) {
		case "short_name":
			goldentest.Text(t, c, ShortName(pyv.Str(c.Args[1])))
		case "choose":
			goldentest.Text(t, c, dumps(t, ChooseAccess(c.Obj(1), c.Obj(2), pyv.Str(c.Args[3]))))
		default:
			t.Fatalf("%s: unknown case", c.Name)
		}
	}
}
