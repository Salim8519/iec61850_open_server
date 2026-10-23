// Package goosemsg answers what the GOOSE messages of one stored IED hold (ADR-0112): its
// sendable control blocks, every dataset member expanded to the IED's leaf signals with the
// member's own functional constraint, and the writes one send makes. It reads only the stored
// IED record (goose.publishes, datasets, signals); it has no I/O and keeps one small cache.
package goosemsg

import (
	"sort"
	"strings"
	"sync"

	"studio-engine/internal/pyjson"
)

// Leaf is one attribute of the IED that a GOOSE member covers.
type Leaf struct {
	Ref, Kind, FC, Description string
	Writable                   bool
}

// Member is one dataset entry (FCDA) of a message, in dataset order (Position starts at 1).
type Member struct {
	Position int
	Text     string // "XCBR1.Pos [ST]"; another logical device keeps its name: "CTRL/XCBR1.Pos [ST]"
	FC       string
	Leaves   []Leaf
}

// Block is one sendable GOOSE control block (an entry of the IED's goose.publishes).
type Block struct {
	GocbRef, Name, Dataset, GoID string
	AppID, ConfRev               any
	Members                      []Member
	leaves                       map[string]Leaf
}

// Leaf returns the member leaf with this reference.
func (b *Block) Leaf(ref string) (Leaf, bool) {
	leaf, found := b.leaves[ref]
	return leaf, found
}

// Catalog is the GOOSE view of one IED record.
type Catalog struct {
	IEDID, Name string
	Blocks      []*Block
	byRef       map[string]*Block
	byLeaf      map[string][]string
}

// Block returns the sendable block with this control block reference, or nil.
func (c *Catalog) Block(gocbRef string) *Block { return c.byRef[gocbRef] }

// BlocksOf lists the blocks (gocbRefs) whose members cover this leaf.
func (c *Catalog) BlocksOf(ref string) []string { return c.byLeaf[ref] }

func object(value any) *pyjson.Object {
	result, _ := value.(*pyjson.Object)
	return result
}

func list(value any) []any {
	result, _ := value.([]any)
	return result
}

func text(o *pyjson.Object, key string) string {
	if o == nil {
		return ""
	}
	value, _ := o.Get(key)
	result, _ := value.(string)
	return result
}

func field(o *pyjson.Object, key string) any {
	if o == nil {
		return nil
	}
	value, _ := o.Get(key)
	return value
}

// Build reads the catalog of one stored IED record. A block whose dataset is missing, or whose
// stored members and member references disagree (a record imported before change 0094), is
// listed without members: the messages route then says to import the IED again.
func Build(ied *pyjson.Object) *Catalog {
	catalog := &Catalog{IEDID: text(ied, "id"), Name: text(ied, "name"), byRef: map[string]*Block{},
		byLeaf: map[string][]string{}}
	signals := indexSignals(list(field(ied, "signals")))
	datasets := map[string]*pyjson.Object{}
	for _, item := range list(field(ied, "datasets")) {
		if dataset := object(item); dataset != nil {
			datasets[text(dataset, "ref")] = dataset
		}
	}
	for _, item := range list(field(object(field(ied, "goose")), "publishes")) {
		control := object(item)
		if control == nil || text(control, "gocb_ref") == "" {
			continue
		}
		block := &Block{GocbRef: text(control, "gocb_ref"), Name: text(control, "name"),
			Dataset: text(control, "dataset"), GoID: text(control, "go_id"), AppID: field(control, "app_id"),
			ConfRev: field(control, "conf_rev"), leaves: map[string]Leaf{}}
		domain, _, _ := strings.Cut(text(control, "dataset_ref"), "/")
		block.Members = members(list(field(control, "members")), datasets[text(control, "dataset_ref")], domain, signals)
		for _, member := range block.Members {
			for _, leaf := range member.Leaves {
				if _, seen := block.leaves[leaf.Ref]; !seen {
					block.leaves[leaf.Ref] = leaf
					catalog.byLeaf[leaf.Ref] = append(catalog.byLeaf[leaf.Ref], block.GocbRef)
				}
			}
		}
		catalog.Blocks = append(catalog.Blocks, block)
		catalog.byRef[block.GocbRef] = block
	}
	return catalog
}

// indexSignals groups the IED's signals by functional constraint, sorted by reference, so a
// member finds its leaves with a binary search.
func indexSignals(items []any) map[string][]Leaf {
	byFC := map[string][]Leaf{}
	for _, item := range items {
		signal := object(item)
		if signal == nil {
			continue
		}
		writable, _ := field(signal, "writable").(bool)
		leaf := Leaf{Ref: text(signal, "ref"), Kind: text(signal, "kind"), FC: text(signal, "fc"),
			Description: text(signal, "description"), Writable: writable}
		byFC[leaf.FC] = append(byFC[leaf.FC], leaf)
	}
	for _, leaves := range byFC {
		sort.Slice(leaves, func(a, b int) bool { return leaves[a].Ref < leaves[b].Ref })
	}
	return byFC
}

type cached struct {
	ied     *pyjson.Object
	catalog *Catalog
}

var cache sync.Map // IED id -> cached; one entry per IED, rebuilt when its record is replaced

// For is Build remembered per IED record: stored records are never written once published, so
// the same record pointer always gives the same catalog (built once per IED and import).
func For(ied *pyjson.Object) *Catalog {
	id := text(ied, "id")
	if found, ok := cache.Load(id); ok && found.(cached).ied == ied {
		return found.(cached).catalog
	}
	catalog := Build(ied)
	cache.Store(id, cached{ied: ied, catalog: catalog})
	return catalog
}
