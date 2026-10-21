package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/project"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/store/archive"
	"studio-engine/internal/store/autosave"
)

// Synthetic large project for the click-path benchmarks: 6 IEDs x 15 000 signals (90k
// points) and 6 distinct SCL texts of about 3.7 MB each (22 MB), plus a diagram of 300
// bound switches. Only the parts the save path touches are realistic; nothing is parsed.
const (
	largeIEDs     = 6
	largeSignals  = 15000
	largeControls = 60
	largeNodes    = 300
	largeSCLBytes = 22 << 20
)

func largeSource(index int) string {
	var builder strings.Builder
	builder.Grow(largeSCLBytes/largeIEDs + 4096)
	fmt.Fprintf(&builder, "<?xml version=\"1.0\"?>\n<SCL><!-- large %d -->\n", index)
	for line := 0; builder.Len() < largeSCLBytes/largeIEDs; line++ {
		fmt.Fprintf(&builder, "<DAI name=\"stVal\" sAddr=\"%d.%d\"><Val>%d</Val></DAI>\n", index, line, line%7)
	}
	builder.WriteString("</SCL>\n")
	return builder.String()
}

func largeIED(index int) *pyjson.Object {
	signals := make([]any, 0, largeSignals)
	for n := 0; n < largeSignals; n++ {
		kind, fc := "FLOAT32", "MX"
		if n%5 == 0 {
			kind, fc = "Dbpos", "ST"
		}
		signals = append(signals, pyjson.NewObject().Set("ref", fmt.Sprintf("LD%d/XCBR%d.Pos.stVal", n%4, n)).
			Set("kind", kind).Set("fc", fc).Set("writable", true).Set("value", "intermediate"))
	}
	controls := make([]any, 0, largeControls)
	for n := 0; n < largeControls; n++ {
		controls = append(controls, pyjson.NewObject().Set("ref", fmt.Sprintf("LD0/CSWI%d.Pos", n)).Set("cdc", "DPC"))
	}
	return pyjson.NewObject().Set("id", fmt.Sprintf("%032x", index+1)).Set("name", fmt.Sprintf("IED_%d", index)).
		Set("filename", fmt.Sprintf("large_%d.scd", index)).Set("source", largeSource(index)).
		Set("cfg", strings.Repeat("LD0/LLN0.Mod.stVal\n", 2000)).Set("signals", signals).Set("controls", controls).
		Set("ip", "").Set("port", int64(102)).Set("role", "simulated").Set("live", pyjson.NewObject()).
		Set("lab_ip", "").Set("goose_publish", "auto").Set("test_mode", "off")
}

// largeDiagram binds every switch to a signal near the end of its IED's list (worst case
// for a linear lookup) and chains them with wires.
func largeDiagram(ieds []*pyjson.Object) any {
	nodes, edges := make([]any, 0, largeNodes), make([]any, 0, largeNodes)
	for n := 0; n < largeNodes; n++ {
		ied := ieds[n%len(ieds)]
		id, _ := ied.Get("id")
		signal := largeSignals - 5*(n/len(ieds)+1)
		state := pyjson.NewObject().Set("mode", "direct").Set("source", pyjson.NewObject().Set("ied_id", id).
			Set("ref", fmt.Sprintf("LD%d/XCBR%d.Pos.stVal", signal%4, signal)).Set("report_ref", nil))
		control := pyjson.NewObject().Set("ied_id", id).Set("ref", fmt.Sprintf("LD0/CSWI%d.Pos", n%largeControls))
		data := pyjson.NewObject().Set("label", fmt.Sprintf("CB %d", n)).Set("kind", "CB").Set("rotation", int64(0)).
			Set("state_mapping", state).Set("control_binding", control)
		nodes = append(nodes, pyjson.NewObject().Set("id", fmt.Sprintf("node-%d", n)).Set("type", "equipment").
			Set("position", pyjson.NewObject().Set("x", int64(n%20*160)).Set("y", int64(n/20*200))).Set("data", data))
		if n > 0 {
			edges = append(edges, pyjson.NewObject().Set("id", fmt.Sprintf("wire-%d", n)).
				Set("source", fmt.Sprintf("node-%d", n-1)).Set("target", fmt.Sprintf("node-%d", n)).
				Set("sourceHandle", "south").Set("targetHandle", "north"))
		}
	}
	return pyjson.NewObject().Set("nodes", nodes).Set("edges", edges)
}

// newLargeService returns a service holding the synthetic project, committed once (the
// journal exists and the snapshot of the full project has been written).
func newLargeService(t testing.TB) *Service {
	s := newService(t)
	ieds := make([]*pyjson.Object, largeIEDs)
	for index := range ieds {
		ieds[index] = largeIED(index)
	}
	diagram, err := project.ValidateDiagram(largeDiagram(ieds), ieds)
	if err != nil {
		t.Fatal(err)
	}
	next := shallow(s.Project())
	next.Set("ieds", objects(ieds)).Set("diagram", diagram)
	s.mutate.Lock()
	err = s.commit(next, nil, true, nil)
	s.mutate.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.snapshots.Flush(); err != nil {
		t.Fatal(err)
	}
	return s
}

// checkStoredMatches decodes autosave.scadabench and compares it with the in-memory project.
func checkStoredMatches(t testing.TB, s *Service) {
	payload, err := os.ReadFile(filepath.Join(s.config.DataDir, autosave.AutosaveName))
	if err != nil {
		t.Fatal(err)
	}
	stored, err := archive.Decode(payload)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := pyjson.Dumps(stored)
	want, _ := pyjson.Dumps(s.current.Load().compact)
	if string(got) != string(want) {
		t.Fatalf("autosave differs from the in-memory project (%d vs %d bytes)", len(got), len(want))
	}
}
