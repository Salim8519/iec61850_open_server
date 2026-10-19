// Package flow is domain/signal_flow.py: the declared signal-flow graph (GOOSE and
// reports) of one SCL document. Every edge comes from an explicit SCL declaration.
package flow

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
)

func id(parts ...string) string {
	digest := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(digest[:])[:16]
}

func get(object *pyjson.Object, key string) any {
	value, _ := object.Get(key)
	return value
}

func str(object *pyjson.Object, key string) string {
	value, _ := get(object, key).(string)
	return value
}

func list(object *pyjson.Object, key string) []any {
	value, _ := get(object, key).([]any)
	return value
}

func diagnostic(level, ied, message string) *pyjson.Object {
	return pyjson.NewObject().Set("level", level).Set("ied", ied).Set("message", message)
}

type graph struct {
	blocks, edges, diagnostics []*pyjson.Object
}

// goose is _goose: one edge per subscription, one block per publishable GoCB.
func (g *graph) goose(flow *scl.Flow) {
	subscribed := map[string]bool{}
	names := make([]string, 0, len(flow.Subscriptions))
	for name := range flow.Subscriptions {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, subscriber := range names {
		entry := flow.Subscriptions[subscriber]
		for _, message := range entry.Problems {
			g.diagnostics = append(g.diagnostics, diagnostic("problem", subscriber, message.(string)))
		}
		for _, raw := range entry.Subscriptions {
			item := raw.(*pyjson.Object)
			ref := str(item, "gocb_ref")
			subscribed[ref] = true
			match, signals := "data", []any{}
			named, all := false, true
			for _, input := range list(item, "inputs") {
				in := input.(*pyjson.Object)
				if str(in, "match") == "named" {
					named = true
				} else {
					all = false
				}
				signals = append(signals, pyjson.NewObject().Set("source", get(in, "source")).
					Set("int_addr", get(in, "int_addr")).Set("desc", get(in, "desc")))
			}
			if named && all {
				match = "named"
			}
			g.edges = append(g.edges, pyjson.NewObject().Set("id", id("goose", ref, subscriber)).Set("type", "goose").
				Set("publisher", get(item, "publisher")).Set("block", id("block", ref)).Set("subscriber", subscriber).
				Set("match", match).Set("signals", signals))
		}
	}
	for _, control := range flow.GoCBs {
		ref := str(control, "gocb_ref")
		g.blocks = append(g.blocks, pyjson.NewObject().Set("id", id("block", ref)).Set("type", "goose").
			Set("ied", get(control, "publisher")).Set("ref", ref).Set("name", get(control, "name")).
			Set("dataset", get(control, "dataset")).Set("app_id", get(control, "app_id")).
			Set("members", get(control, "members")))
		if !subscribed[ref] {
			g.diagnostics = append(g.diagnostics, diagnostic("info", str(control, "publisher"),
				"Nobody in this file subscribes to GOOSE "+ref))
		}
	}
}

// reports is _reports: one block per report, one edge per client inside the file.
func (g *graph) reports(flow *scl.Flow, names map[string]bool) {
	for _, report := range flow.Reports {
		ref, publisher := str(report, "ref"), str(report, "publisher")
		block := id("block", ref)
		g.blocks = append(g.blocks, pyjson.NewObject().Set("id", block).Set("type", "report").Set("ied", publisher).
			Set("ref", ref).Set("name", get(report, "name")).Set("dataset", get(report, "dataset")).
			Set("buffered", get(report, "buffered")).Set("members", get(report, "members")))
		if str(report, "dataset") != "" && get(report, "dataset_found") == false {
			g.diagnostics = append(g.diagnostics, diagnostic("problem", publisher, "Report "+ref+" uses dataset "+
				str(report, "dataset")+", which is not in this file"))
		}
		for _, raw := range list(report, "clients") {
			client := str(raw.(*pyjson.Object), "iedName")
			if !names[client] {
				shown := client
				if shown == "" {
					shown = "(unnamed)"
				}
				g.diagnostics = append(g.diagnostics, diagnostic("info", publisher, "Report "+ref+" has client "+shown+
					", which is outside this file (for example a gateway or HMI)"))
				continue
			}
			signals := []any{}
			for _, member := range list(report, "members") {
				signals = append(signals, pyjson.NewObject().Set("source", member).Set("int_addr", "").Set("desc", ""))
			}
			g.edges = append(g.edges, pyjson.NewObject().Set("id", id("report", ref, client)).Set("type", "report").
				Set("publisher", publisher).Set("block", block).Set("subscriber", client).Set("match", "named").
				Set("signals", signals))
		}
	}
}
